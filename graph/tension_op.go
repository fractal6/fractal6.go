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
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)


var EMAP EventsMap
var SubscribingEvents map[model.TensionEvent]bool

func init() {
    EMAP = EventsMap{
        model.TensionEventCreated: EventMap{
            Auth: MemberStrictHook,
        },
        model.TensionEventCommentPushed: EventMap{
            Auth: MemberHook | AuthorHook,
        },
        model.TensionEventBlobCreated: EventMap{
            Auth: MemberStrictHook,
        },
        model.TensionEventBlobCommitted: EventMap{
            Auth: MemberStrictHook,
        },
        model.TensionEventTitleUpdated: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
            Propagate: "title",
        },
        model.TensionEventTypeUpdated: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
            Propagate: "type_",
        },
        model.TensionEventReopened: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
            Propagate: "status",
        },
        model.TensionEventClosed: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
            Propagate: "status",
        },
        model.TensionEventLabelAdded: EventMap{
            Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventLabelRemoved: EventMap{
            Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventAssigneeAdded: EventMap{
            Auth: TargetCoordoHook,
            Restrict: []RestrictValue{
                UserNewIsMemberRestrict,
            },
        },
        model.TensionEventAssigneeRemoved: EventMap{
            Auth: TargetCoordoHook,
        },
        model.TensionEventPinned: EventMap{
            Auth: TargetCoordoHook,
            Action: PinTension,
        },
        model.TensionEventUnpinned: EventMap{
            Auth: TargetCoordoHook,
            Action: UnpinTension,
        },
        // --- Trigger Action ---
        model.TensionEventBlobPushed: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: PushBlob,
        },
        model.TensionEventBlobArchived: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: ChangeArchiveBlob,
        },
        model.TensionEventBlobUnarchived: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: ChangeArchiveBlob,
        },
        model.TensionEventAuthority: EventMap{
            Auth: TargetCoordoHook,
            Action: ChangeAuhtority,
        },
        model.TensionEventVisibility: EventMap{
            Auth: TargetCoordoHook,
            Action: ChangeVisibility,
        },
        model.TensionEventMoved: EventMap{
            Validation: model.ContractTypeAnyCoordoDual,
            Auth: AuthorHook | SourceCoordoHook | TargetCoordoHook | AssigneeHook,
            Action: MoveTension,
            Restrict: []RestrictValue{
                UserIsMemberRestrict,
            },
        },
        model.TensionEventMemberLinked: EventMap{
            Validation: model.ContractTypeAnyCandidates,
            // @DEBUG: auth, can a user open a contract in a private Circle ???
            // if yes, constraint the candidateHook to Public circle only.
            Auth: TargetCoordoHook | AssigneeHook | CandidateHook,
            Action: ChangeFirstLink,
        },
        model.TensionEventMemberUnlinked: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: ChangeFirstLink,
        },
        model.TensionEventUserJoined: EventMap{
            // @FIXFEAT: Either Check Receiver NodeCharac or contract value to check that user has been invited !
            Validation: model.ContractTypeAnyCandidates,
            Auth: TargetCoordoHook | AssigneeHook | CandidateHook,
            Action: UserJoin,
        },
        model.TensionEventUserLeft: EventMap{
            // Authorisation is done in the method for now ("FirstLinkHook").
            Auth: PassingHook,
            Action: UserLeave,
        },
    }

    SubscribingEvents = map[model.TensionEvent]bool{
        model.TensionEventCreated: true,
        model.TensionEventCommentPushed: true,
        model.TensionEventReopened: true,
        model.TensionEventClosed: true,
    }
}

// tensionEventHook is applied for addTension and updateTension query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload) with
// All events in History must pass.
func TensionEventHook(uctx *model.UserCtx, tid string, events []*model.EventRef, blob *model.BlobRef) (bool, *model.Contract, error) {
    var ok bool = true
    var addSubscriber bool
    var err error
    var tension *model.Tension
    var contract *model.Contract
    if events == nil {
        return false, nil, LogErr("Access denied", fmt.Errorf("No event given."))
    }

    for _, event := range(events) {
        if tension == nil { // don't fetch if there is no events (Comment updated...)
            // Fetch Tension, target Node and blob charac (last if blob if nil)
            var bid *string
            if blob != nil { bid = blob.ID }
            tension, err = db.GetDB().GetTensionHook(tid, true, bid)
            if err != nil { return false, nil, LogErr("Access denied", err) }
        }

        // Process event
        ok, contract, err = ProcessEvent(uctx, tension, event, blob, nil, true, true)
        if !ok || err != nil { break }

        // Check if event make a new subscriber
        addSubscriber = addSubscriber || SubscribingEvents[*event.EventType]
    }

    // Add subscriber
    // @performance: @defer this with Redis
    if addSubscriber && ok && err == nil {
		err = db.GetDB().Update(*uctx, "tension", &model.UpdateTensionInput{
			Filter: &model.TensionFilter{ID: []string{tension.ID}},
			Set: &model.TensionPatch{Subscribers: []*model.UserRef{&model.UserRef{Username: &uctx.Username}}},
		})
    }

    return ok, contract, err
}

func ProcessEvent(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, blob *model.BlobRef, contract *model.Contract,
                    doCheck, doProcess bool) (bool, *model.Contract, error) {
    var ok bool
    var err error

    if tension == nil {
        return ok, contract, LogErr("Access denied", fmt.Errorf("tension not found."))
    }

    em, hasEvent := EMAP[*event.EventType]
    if !hasEvent { // Minimum level of authorization
        return false, nil, LogErr("Access denied", fmt.Errorf("Event not implemented."))
    }

    // Check Authorization (optionally generate a contract)
    if doCheck {
        ok, contract, err = em.Check(uctx, tension, event, contract)
        if !ok || err != nil { return ok, contract, err }
    }

    // act is false if contract is cancelled for example !
    act := contract == nil || contract.Status == model.ContractStatusClosed

    // Trigger Action
    if act && doProcess {
        if em.Propagate != "" {
            v, err := CheckEvent(tension, event)
            if err != nil { return ok, contract, err }
            err = db.GetDB().UpdateValue(*uctx, "tension", tension.ID, em.Propagate, v)
            if err != nil { return ok, contract, err }
        }
        if em.Action != nil {
            ok, err = em.Action(uctx, tension, event, blob)
            if !ok || err != nil { return ok, contract, err }
        }

        // leave trace
        leaveTrace(tension)
    }

    // Set contract status if any
    if contract != nil && doProcess {
        err = db.GetDB().SetFieldById(contract.ID, "Contract.status", string(contract.Status))
        if err != nil { return false, contract, err }

        // Assumes contract is either closed or cancelled.
        // @DEBUG: wouldn't it be a bettter way to work with voteid & contractid ?
        err = db.GetDB().RewriteContractId(contract.ID)
        if err != nil { return false, contract, err }
    }

    return ok, contract, err
}

// GetBlob returns the first blob found in the given tension.
func GetBlob(tension *model.Tension) *model.Blob {
    if tension.Blobs != nil { return tension.Blobs[0] }
    return nil
}

func leaveTrace(tension *model.Tension) {
    var err error
    var nameid string

    blob := GetBlob(tension)
    if blob != nil {
        // Get Node and Nameid (from Codec)
        node := blob.Node
        if node != nil && node.Nameid != nil {
            _, nameid, err = codec.NodeIdCodec(tension.Receiver.Nameid, *node.Nameid, *node.Type)
        }

        // Set the Update time into the affected node.
        err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.updatedAt", Now())
        if err != nil { panic(err) }
        // Set the Update of its parent node (tension.receiver)
        err = db.GetDB().SetFieldByEq("Node.nameid", tension.Receiver.Nameid, "Node.updatedAt", Now())
        if err != nil { panic(err) }
    }
}

//
// Event Actions
//

func PushBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Add or Update Node
    // --
    // 1. switch on TensionCharac.DocType (not blob type) -> rule differ from doc type!
    // 2. swith on TensionCharac.ActionType to add update etc...
    // * update the tension action value AND the blob pushedFlag
    // * copy the Blob data in the target Node.source (Uses GQL requests)
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }

    // Extract tension blob characteristic
    tensionCharac, err := codec.TensionCharac{}.New(*tension.Action)
    if err != nil { return false, fmt.Errorf("tensionCharac unknown.") }

    switch tensionCharac.ActionType {
    case codec.NewAction:
        // First time a blob is pushed.
        switch tensionCharac.DocType {
        case codec.NodeDoc:
            ok, err = TryAddNode(uctx, tension, blob.Node, &blob.ID)
        case codec.MdDoc:
            ok, err = TryAddDoc(uctx, tension, blob.Md)
        }
    case codec.EditAction:
        switch tensionCharac.DocType {
        case codec.NodeDoc:
            ok, err = TryUpdateNode(uctx, tension, blob.Node, &blob.ID)
        case codec.MdDoc:
            ok, err = TryUpdateDoc(uctx, tension, blob.Md)
        }
    case codec.ArchiveAction:
        err = LogErr("Access denied", fmt.Errorf("Cannot publish archived document."))
    }

    if err != nil { return ok, err }
    if ok { // Update blob pushed flag
        err = db.GetDB().SetPushedFlagBlob(blob.ID, Now(), tension.ID, tensionCharac.EditAction(blob.Node.Type))
    }

    return ok, err
}

func ChangeArchiveBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Archived/Unarchive Node
    // * link or unlink role
    // * set archive event and flag
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }

    // Extract tension blob characteristic
    tensionCharac, err := codec.TensionCharac{}.New(*tension.Action)
    if err != nil { return false, fmt.Errorf("tensionCharac unknown.") }

    switch tensionCharac.DocType {
    case codec.NodeDoc:
        ok, err = TryChangeArchiveNode(uctx, tension, blob.Node, *event.EventType)
    case codec.MdDoc:
        md := blob.Md
        ok, err = TryChangeArchiveDoc(uctx, tension, md, *event.EventType)
    }

    if err != nil { return ok, err }
    if ok { // Update blob archived flag
        if *event.EventType == model.TensionEventBlobArchived {
            err = db.GetDB().SetArchivedFlagBlob(blob.ID, Now(), tension.ID, tensionCharac.ArchiveAction(blob.Node.Type))
        } else if *event.EventType == model.TensionEventBlobUnarchived {
            err = db.GetDB().SetPushedFlagBlob(blob.ID, Now(), tension.ID, tensionCharac.EditAction(blob.Node.Type))
        } else {
            err = fmt.Errorf("bad tension event '%s'.", string(*event.EventType))
        }
    }

    return ok, err
}

func ChangeAuhtority(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // ChangeAuthory
    // * If Circle : change mode on pointed node
    // * If Role : change role_type on the pointed node (on Node + Node.RoleExt)
    // * Don't touch the current blob as we do not use "authority" properties at the moment (just when adding node)
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }

    ok, err := TryChangeAuthority(uctx, tension, blob.Node, *event.New)

    return ok, err
}

func ChangeVisibility(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // ChangeVisibility
    // * Change the visiblity of the node
    // * Don't touch the current blob as we do not use "authority" properties at the moment (just when adding node)
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }

    ok, err := TryChangeVisibility(uctx, tension, blob.Node, *event.New)

    return ok, err
}

func ChangeFirstLink(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // ChangeFirstLink
    // * ensure first_link is free on link
    // * Link/unlink user
    var ok bool
    var unsafe bool = false

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }
    node := blob.Node

    if node != nil && node.Type != nil && *node.Type == model.NodeTypeCircle {
        // Get membership node node
        rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
        if err != nil { return ok, err }
        nid := codec.MemberIdCodec(rootid, *event.Old)
        n, err := db.GetDB().GetFieldByEq("Node.nameid", nid, "Node.name Node.nameid Node.type_ Node.role_type")
        if err != nil { return ok, err }
        var nf model.NodeFragment
        StructMap(n, &nf)
        if *nf.RoleType != model.RoleTypeGuest {
            return false, LogErr("Value error", fmt.Errorf("You cannot detach this role (%s) like this.", string(*nf.RoleType)))
        }
        nf.FirstLink = event.Old
        node = &nf
        unsafe = true
    }

    ok, err := TryUpdateLink(uctx, tension, node, event, unsafe)

    return ok, err
}


func MoveTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    if event.Old == nil || event.New == nil { return false, fmt.Errorf("old and new event data must be defined.") }
    if *event.Old != tension.Receiver.Nameid {
        return false, fmt.Errorf("Contract outdated: event source (%s) and actual source (%s) differ. Please, refresh or remove this contract.", *event.Old, tension.Receiver.Nameid)
    }

    var err error
    receiverid_old := *event.Old // == tension.Receiverid
    receiverid_new := *event.New

    // Update node and blob
    if tension.Blobs != nil && tension.Blobs[0].Node != nil {
        node := tension.Blobs[0].Node
        _, nameid_old, err := codec.NodeIdCodec(receiverid_old, *node.Nameid, *node.Type)
        if err != nil { return false, err }
        _, nameid_new, err := codec.NodeIdCodec(receiverid_new, *node.Nameid, *node.Type)
        if err != nil { return false, err }

        // test root node
        if codec.IsRoot(tension.Emitter.Nameid) {
            return false, fmt.Errorf("You can't move the root node.")
        }
        // test self-loop
        if receiverid_new == nameid_new {
            return false, fmt.Errorf("A node cannot be its own parent.")
        }
        // test recursion
        isChild, err := db.GetDB().IsChild(nameid_old, receiverid_new)
        if err != nil { return false, err }
        if isChild {
            return false, fmt.Errorf("You can't move a node in their children.")
        }

        // node input
        nodeInput := model.UpdateNodeInput{
            Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid_old}},
            Set: &model.NodePatch{
                Parent: &model.NodeRef{Nameid: &receiverid_new},
            },
        }

        // update node
        err = db.GetDB().Update(db.DB.GetRootUctx(), "node", nodeInput)
        if err != nil { return false, err }

        // DQL mutation (extra node update)
        if nameid_old != nameid_new { // node is a role
            err = db.GetDB().PatchNameid(nameid_old, nameid_new)
            if err != nil { return false, err }
        }
    }

    // tension input
    tensionInput := model.UpdateTensionInput{
        Filter: &model.TensionFilter{ID: []string{tension.ID}},
        Set: &model.TensionPatch{
            Receiver: &model.NodeRef{Nameid: &receiverid_new},
            Receiverid: &receiverid_new,
        },
    }

    // update tension
    err = db.GetDB().Update(db.GetDB().GetRootUctx(), "tension", tensionInput)
    if err != nil { return false, err }

    // Update tension pin
    _, err = db.GetDB().Meta("movePinnedTension", map[string]string{"nameid_old":receiverid_old, "nameid_new":receiverid_new, "tid": tension.ID})

    return true, err
}

func UserJoin(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    var ok bool

    // Only root node can be joined
    // --
    username := *event.New
    rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
    if err != nil { return ok, err }

    // Validate
    // --
    // check the invitation if a hash is given
    // * orga invitation ? <> user invitation hash ?
    // * else check if User Can Join Organisation
    // @debug: this should be done before the contract creation.
    guestid := codec.MemberIdCodec(rootid, username)
    // Pending node as been created at invitation
    //ex, err :=  db.GetDB().Exists("Node.nameid", guestid, nil, nil)
    //if err != nil { return ok, err }
    err = LinkUser(rootid, guestid, username)
    if err != nil { return ok, err }

    // Make user watch that organisation.
    err = db.GetDB().Update(*uctx, "user", &model.UpdateUserInput{
        Filter: &model.UserFilter{Username: &model.StringHashFilterStringRegExpFilter{Eq: &username}},
        Set: &model.UserPatch{Watching: []*model.NodeRef{&model.NodeRef{Nameid: &rootid}}},
    })

    return true, err
}

func UserLeave(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Remove user reference
    // * remove User role
    // * update user membership
    // --
    var ok bool
    var unsafe bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.") }
    node := blob.Node
    role_type := model.RoleType(*event.New)

    if role_type == model.RoleTypeGuest {
        uctx.NoCache = true
        i := auth.UserIsGuest(uctx, tension.Emitter.Nameid)
        if i < 0 {
            return ok, LogErr("Value error", fmt.Errorf("You are not a guest in this organisation."))
        }
        var nf model.NodeFragment
        t := model.NodeTypeRole
        StructMap(uctx.Roles[i], &nf)
        nf.FirstLink = &uctx.Username
        nf.Type = &t
        node = &nf
        unsafe = true
    } else if role_type == model.RoleTypeRetired ||
    role_type == model.RoleTypeMember ||
    role_type == model.RoleTypePending {
        return false, fmt.Errorf("You cannot leave this role like this.")
    } else if role_type == model.RoleTypeOwner {
        return false, fmt.Errorf("Owner cannot leave organisation. Please contact us if you need to transfer ownership.")
    }

    ok, err := LeaveRole(uctx, tension, node, unsafe)
    return ok, err
}

func PinTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    tid := tension.ID
    nameid := tension.Receiver.Nameid
    // node input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Set: &model.NodePatch{
            Pinned: []*model.TensionRef{&model.TensionRef{ID: &tid}},
        },
    }
    // update node
    err := db.GetDB().Update(db.DB.GetRootUctx(), "node", nodeInput)
    return true, err
}

func UnpinTension(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    tid := tension.ID
    nameid := tension.Receiver.Nameid
    // node input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Remove: &model.NodePatch{
            Pinned: []*model.TensionRef{&model.TensionRef{ID: &tid}},
        },
    }
    // update node
    err := db.GetDB().Update(db.DB.GetRootUctx(), "node", nodeInput)
    return true, err
}



//
// Utilities
//

// Check event before propagation. Should be defined in directives,
// those are transactionned from event.
func CheckEvent(t *model.Tension, e *model.EventRef) (string, error) {
    if e.New == nil || *e.New == "" {
        return "", fmt.Errorf("Event new field must be given.")
    }

    b := GetBlob(t)
    var v = *e.New
    var err error

    switch *e.EventType {
    case model.TensionEventTypeUpdated:
        if b != nil && b.Node != nil && *b.Node.Type == model.NodeTypeCircle {
            err = fmt.Errorf("The type of tensions with circle attached cannot be changed.")
        } else if b != nil && b.Node != nil && *b.Node.Type == model.NodeTypeRole {
            err = fmt.Errorf("The type of tensions with role attached cannot be changed.")
        }
    default:
        // pass
    }

    return v, err
}

