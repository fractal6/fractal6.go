/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package graph

import (
    "fmt"
    "context"
    "strings"
    "github.com/99designs/gqlgen/graphql"

    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/db"
    "fractale/fractal6.go/graph/auth"
    webauth"fractale/fractal6.go/web/auth"
    . "fractale/fractal6.go/tools"
)


////////////////////////////////////////////////
// Contract Resolver
////////////////////////////////////////////////


func addContractInputHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    data, err := next(ctx)
    if err != nil {
        return data, err
    }

    // Move pendingCandidate to candidate if email exists in User
    newData := data.([]*model.AddContractInput)
    for i, input := range newData {
        var pendings []*model.PendingUserRef
        var candidates []*model.UserRef
        for _, c := range input.PendingCandidates {
            if c.Email == nil { continue }
            if v, _ := db.GetDB().GetFieldByEq("User.email", *c.Email, "User.username"); v != nil {
                username := v.(string)
                candidates = append(candidates, &model.UserRef{Username: &username})

                // Update Contract
                emailPart := strings.Split(*c.Email, "@")[0]
                if input.Event.Old != nil && strings.HasPrefix(*input.Event.Old, emailPart) {
                    newData[i].Event.Old = &username
                }
                if input.Event.New != nil && strings.HasPrefix(*input.Event.New, emailPart) {
                    newData[i].Event.New = &username
                }
                newData[i].Contractid = codec.ContractIdCodec(
                    *newData[i].Tension.ID,
                    *newData[i].Event.EventType,
                    *newData[i].Event.Old,
                    *newData[i].Event.New,
                )
            } else if err := webauth.ValidateEmail(*c.Email); err != nil {
                return nil, err
            } else {
               pendings = append(pendings, c)
            }
        }
        newData[i].PendingCandidates = pendings
        newData[i].Candidates = append(newData[i].Candidates, candidates...)
    }

    return newData, err
}

// Add Contract hook
func addContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate Input
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddContractInput)
    if len(inputs) != 1 {
        return nil, LogErr("add contract", fmt.Errorf("One and only one contract allowed."))
    }
    if !PayloadContains(ctx, "id") {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in contract payload"))
    }
    input := inputs[0]

    // Eventually add a pending node
    pendingNodeCreated := false
    if *input.Event.EventType == model.TensionEventMemberLinked || *input.Event.EventType == model.TensionEventUserJoined {
        // Add pending Nodes
        for _, c := range input.Candidates {
            pendingNodeCreated, err = MaybeAddPendingNode(*c.Username, &model.Tension{ID: *input.Tension.ID})
            if err != nil { return nil, err }
        }
    }

    // Execute query
    data, err := next(ctx)
    if err != nil { return nil, err }
    if data.(*model.AddContractPayload) == nil {
        return nil, LogErr("add contract", fmt.Errorf("no contract added."))
    }
    id := data.(*model.AddContractPayload).Contract[0].ID
    tid := *input.Tension.ID
    cid := *&input.Contractid

    // Validate and process Blob Event
    var event model.EventRef
    StructMap(*input.Event, &event)
    ok, contract, err := contractEventHook(uctx, cid, tid, &event, nil)
    if !ok || err != nil {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("contract", id)
        if e != nil { panic(e) }

        if pendingNodeCreated {
            // Delete pending Nodes
            for _, c := range contract.Candidates {
                err = MaybeDeletePendingNode(c.Username, contract.Tension)
                if err != nil { return nil, err }
            }
        }

        if err != nil { return nil, err }
    } else if ok {
        // Push Notifications
        if contract != nil {
            PublishContractEvent(model.ContractNotif{Uctx: uctx, Tid: tid, Contract: contract, ContractEvent: model.NewContract})
        }

        return data, err
    }

    return data, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Contract hook
func updateContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate Input
    input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateContractInput)
    ids := input.Filter.ID
    if len(ids) != 1 {
        return nil, LogErr("update tension", fmt.Errorf("One and only one tension allowed."))
    }

    // Validate and process Blob Event
    var ok bool
    if input.Set != nil {
        // getContractHook
        contract, err := db.GetDB().GetContractHook(ids[0])
        if err != nil  { return nil, err }
        // Check if user has admin right
        ok, err = HasContractRight(uctx, contract)
        if err != nil  { return nil, err }
        if !ok {
            // Check if user is candidate
            for _, c := range contract.Candidates {
                if c.Username == uctx.Username  {
                    ok = true
                    break
                }
            }
        }
        if ok {
            // Notify users by email
            if input.Set.Comments != nil && len(input.Set.Comments) > 0 {
                PublishContractEvent(model.ContractNotif{Uctx: uctx, Tid: contract.Tension.ID, Contract: contract, ContractEvent: model.NewComment})
            }
            // Execute query
            return next(ctx)
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("You are not authorized to access this ressource."))
        }
    }

    return nil, LogErr("Access denied", fmt.Errorf("Input remove not implemented."))
}

// Delete Contract hook
func deleteContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    filter := rc.Args["filter"].(model.ContractFilter)
    ids := filter.ID
    if len(ids) != 1 {
        return nil, LogErr("delete contract", fmt.Errorf("One and only one contract allowed."))
    }

    // AUTHORIZATION
    // --
    var ok bool = false
    // isAuthor
    author, err := db.GetDB().GetSubFieldById(ids[0], "Post.createdBy", "User.username")
    if err != nil { return nil, err }
    if author == nil { panic("empty createdBy field") }
    ok = author.(string) == uctx.Username
    // OR has rights (coordo or assigned).
    if !ok {
        nameid, err := db.GetDB().GetSubFieldById(ids[0], "Contract.tension", "Tension.receiverid")
        if err != nil { return nil, err }
        if nameid == nil { panic("empty receiverid field") }
        mode := model.NodeModeCoordinated
        ok, err = auth.HasCoordoRole(uctx, nameid.(string), &mode)
        if err != nil { return nil, err }
    }
    if !ok {
        return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
    }

    // Eventually reset the pending node state
    contract, err := db.GetDB().GetContractHook(ids[0])
    if err != nil { return nil, err }

    // Clear eventual pending roles.
    if contract.Event.EventType == model.TensionEventMemberLinked || contract.Event.EventType == model.TensionEventUserJoined {
        for _, c := range contract.Candidates {
            err = MaybeDeletePendingNode(c.Username, contract.Tension)
            if err != nil { return nil, err }
        }
    }

    // Notify user of the cancel
    msg := fmt.Sprintf("Contract %s has been cancelled.", contract.ID)
    var to []string
    for _, p := range contract.Participants {
        to = append(to, p.Node.FirstLink.Username)
    }
    PublishNotifEvent(model.NotifNotif{Uctx: uctx, Tid: &contract.Tension.ID, Cid: &contract.ID, Msg: msg, To: to})

    // Deep delete
    err = db.GetDB().DeepDelete("contract", ids[0])
    if err != nil { return nil, LogErr("Delete contract error", err) }

    var d model.DeleteContractPayload
    d.Contract = []*model.Contract{&model.Contract{ID: ids[0]}}
    return &d, err
}

// ------------------------------------------------------------------- Contracts

func isContractValidator(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    if !PayloadContainsGo(ctx, "ID") {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in vote payload"))
    }

    _, err = next(ctx)
    if err != nil { return nil, err }

    data := false
    d := obj.(*model.Contract)

    // If user has already voted
    // NOT CHECKING, user can change its vote.

    // Check rights
    data, err = HasContractRight(uctx, d)

    return &data, err
}

////////////////////////////////////////////////
// Vote Resolver
////////////////////////////////////////////////


func addVoteHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate Input
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddVoteInput)
    if len(inputs) != 1 {
        return nil, LogErr("add vote", fmt.Errorf("One and only one vote allowed."))
    }
    if !PayloadContains(ctx, "id") {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in vote payload"))
    }
    input := inputs[0]
    cid := *input.Contract.Contractid
    nameid := *input.Node.Nameid
    //vote := input.Data[0]

    // Ensure the vote ID
    if input.Voteid != cid + "#" + nameid {
        return nil, LogErr("add vote", fmt.Errorf("bad format for voteID."))
    }
    // Ensure that user own the vote
    rid, _ := codec.Nid2rootid(nameid)
    if nameid != codec.MemberIdCodec(rid, uctx.Username) {
        return nil, LogErr("add vote", fmt.Errorf("You must own your vote."))
    }

    // Try to add vote
    d, err := next(ctx)
    if err != nil { return nil, err }
    data := d.(*model.AddVotePayload)
    if data == nil {
        return nil, LogErr("add vote", fmt.Errorf("no vote added."))
    }

    // Post process vote
    ok, contract, err := voteEventHook(uctx, cid)
    if err != nil {
        id := data.Vote[0].ID
        e := db.GetDB().Delete(*uctx, "vote", model.VoteFilter{ID:[]string{id}})
        if e != nil { panic(e) }
        return nil, err
    } else if !ok {
        return d, err
    }

    if contract.Status == model.ContractStatusCanceled {
        // Eventually reset the pending node state
        if contract.Event.EventType == model.TensionEventMemberLinked || contract.Event.EventType == model.TensionEventUserJoined {
            for _, c := range contract.Candidates {
                err = MaybeDeletePendingNode(c.Username, contract.Tension)
                if err != nil { return nil, err }
            }
        }

        // Notify user of the cancel
        msg := fmt.Sprintf("Contract %s has been cancelled.", contract.ID)
        var to []string
        for _, p := range contract.Participants {
            to = append(to, p.Node.FirstLink.Username)
        }
        PublishNotifEvent(model.NotifNotif{Uctx: uctx, Tid: &contract.Tension.ID, Cid: &contract.ID, Msg: msg, To: to})
    } else if contract.Status == model.ContractStatusClosed {
        PublishContractEvent(model.ContractNotif{Uctx: uctx, Tid: contract.Tension.ID, Contract: contract, ContractEvent: model.CloseContract})
    }

    data.Vote[0].Contract = contract
    return data, err
}


