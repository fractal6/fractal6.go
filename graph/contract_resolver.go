package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/graph/auth"
    webauth"zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)


////////////////////////////////////////////////
// Contract Resolver
////////////////////////////////////////////////


// Add Contract hook
func addContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
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

    // Execute query
    data, err := next(ctx)
    if err != nil { return nil, err }
    if data.(*model.AddContractPayload) == nil {
        return nil, LogErr("add contract", fmt.Errorf("no contract added."))
    }
    id := data.(*model.AddContractPayload).Contract[0].ID
    tid := *input.Tension.ID

    // Validate and process Blob Event
    var event model.EventRef
    StructMap(*input.Event, &event)
    ok, err := contractEventHook(uctx, tid, &event, nil)
    if !ok || err != nil {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("contract", id)
        if e != nil { panic(e) }
    }
    if ok || err != nil {
        return data, err
    }
    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Contract hook
func updateContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    //uctx, err := webauth.GetUserContext(ctx)
    //if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateContractInput)
    ids := input.Filter.ID
    if len(ids) != 1 {
        return nil, LogErr("update contract", fmt.Errorf("One and only one contract allowed."))
    }

    // Validate Event prior the mutation
    // <!> Only used to post comment <!>
    if input.Remove != nil || input.Set == nil || len(input.Set.Comments) != 1 {
        return nil, LogErr("update contract", fmt.Errorf("comment missing"))
    }

    //c := input.Set.Comments[0]
    // @obsolete ??
    // use only add/upsert insead.

    data, err := next(ctx)
    if err != nil { return nil, err }
    if data.(*model.UpdateContractPayload) == nil {
        return nil, LogErr("update contract", fmt.Errorf("no contract updated."))
    }

    // notify participants and candidates
    return data, err
}

// Delete Contract hook
func deleteContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
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

    // Deep delete
    err = db.GetDB().DeepDelete("contract", ids[0])
    if err != nil { return nil, LogErr("Delete contract error", err) }

    var d model.DeleteContractPayload
    d.Contract = []*model.Contract{&model.Contract{ID: ids[0]}}
    return &d, err

    //data, err := next(ctx)
    //if err != nil { return nil, err }
    //return data, err
}

// ------------------------------------------------------------------- Contracts

func isContractValidator(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
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
    data, err = hasContractRight(uctx, d)

    return &data, err
}

////////////////////////////////////////////////
// Vote Resolver
////////////////////////////////////////////////


func addVoteHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
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
        return nil, LogErr("add vote", fmt.Errorf("You should own yout vote."))
    }

    // Try to add vote
    d, err := next(ctx)
    if err != nil { return nil, err }
    data := d.(*model.AddVotePayload)
    if data == nil {
        return nil, LogErr("add vote", fmt.Errorf("no vote added."))
    }

    // Post process vote
    ok, contract, err := processVote(uctx, cid)
    if !ok || err != nil {
        id := data.Vote[0].ID
        vi := model.VoteFilter{ID:[]string{id}}
        e := db.GetDB().Delete(*uctx, "vote", vi)
        if e != nil { panic(e) }
        return nil, err
    }

    data.Vote[0].Contract = contract

    return data, err
}


