package graph

import (
	"fmt"

	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/auth"
	"zerogov/fractal6.go/graph/codec"
	"zerogov/fractal6.go/graph/model"
	. "zerogov/fractal6.go/tools"
)


var EMAP EventsMap

func init() {
    EMAP = EventsMap{
        model.TensionEventCreated: EventMap{
            Auth: MemberHook,
        },
        model.TensionEventCommentPushed: EventMap{
            Auth: MemberHook,
        },
        model.TensionEventBlobCreated: EventMap{
            Auth: MemberHook,
        },
        model.TensionEventBlobCommitted: EventMap{
            Auth: MemberHook,
        },
        model.TensionEventTitleUpdated: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventTypeUpdated: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventReopened: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventClosed: EventMap{
            Auth: SourceCoordoHook | TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventLabelAdded: EventMap{
            Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventLabelRemoved: EventMap{
            Auth: TargetCoordoHook | AuthorHook | AssigneeHook,
        },
        model.TensionEventAssigneeAdded: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
        },
        model.TensionEventAssigneeRemoved: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
        },
        // --- Trigger Action ---
        model.TensionEventBlobPushed: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: PushBlob,
        },
        model.TensionEventBlobArchived: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: ArchiveBlob,
        },
        model.TensionEventBlobUnarchived: EventMap{
            Auth: TargetCoordoHook | AssigneeHook,
            Action: UnarchiveBlob,
        },
        model.TensionEventUserLeft: EventMap{
            // Authorisation is done in the method for now (to avoid dealing with Guest node two times).
            Auth: PassingHook,
            Action: UserLeave,
        },
        model.TensionEventUserJoined: EventMap{
            // @FIXFEAT: Either Check Receiver NodeCharac or contract value to check that user has been invited !
            Validation: model.ContractTypeAnyCandidates,
            Auth: TargetCoordoHook | CandidateHook,
            Action: UserJoin,
        },
        model.TensionEventMoved: EventMap{
            Validation: model.ContractTypeAnyCoordoDual,
            Auth: AuthorHook | SourceCoordoHook | TargetCoordoHook | AssigneeHook,
            Action: MoveTension,
        },
    }
}

// tensionEventHook is applied for addTension and updateTension query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload) with
// All events in History must pass.
func tensionEventHook(uctx *model.UserCtx, tid string, events []*model.EventRef, blob *model.BlobRef) (bool, *model.Contract, error) {
    var ok bool = true
    var err error
    var tension *model.Tension
    var contract *model.Contract
    if events == nil {
        return false, nil, LogErr("Access denied", fmt.Errorf("No event given."))
    }

    for _, event := range(events) {
        if tension == nil { // don't fetch if there is no events (Comment updated...)
            // Fetch Tension, target Node and blob charac (last if bid undefined)
            var bid *string
            if blob != nil { bid = blob.ID }
            tension, err = db.GetDB().GetTensionHook(tid, true, bid)
            if err != nil { return false, nil, LogErr("Access denied", err) }
        }

        // Process event
        ok, contract, err = processEvent(uctx, tension, event, blob, nil, true, true, true)
        if !ok || err != nil { break }

    }

    return ok, contract, err
}

func processEvent(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, blob *model.BlobRef, contract *model.Contract,
                    doCheck, doProcess, doNotify bool) (bool, *model.Contract, error) {
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

    act := contract == nil || contract.Status == model.ContractStatusClosed

    // Trigger Action
    if act && doProcess && em.Action != nil {
        ok, err = em.Action(uctx, tension, event, blob)
        if !ok || err != nil { return ok, contract, err }
        // leave trace
        leaveTrace(tension)
    }

    // Set contract status if any
    if contract != nil {
        err = db.GetDB().SetFieldById(contract.ID, "Contract.status", string(contract.Status))
        if err != nil { return false, contract, err }

        // Assumes contract is either closed or cancelled.
        err = db.GetDB().SetFieldById(contract.ID, "Contract.contractid", contract.ID)
        if err != nil { return false, contract, err }
    }

    // Notify users
    if doNotify {
        // push notification (somewhere ?!)
        //if act {
            // * participants
            // * candidates
            // * assigness
            // * coordo
            // * suscriber
        //} else {
            //notify only the participants and candidates
            // inform what has beend voted...
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

        // Set the Update time into the target node
        err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.updatedAt", Now())
        pid, _ := codec.Nid2pid(nameid) // @debug: real parent needed here (ie event for circle)
        if pid != nameid && err == nil {
            err = db.GetDB().SetFieldByEq("Node.nameid", pid, "Node.updatedAt", Now())
        }
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
    if blob == nil { return false, fmt.Errorf("blob not found.")}

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

func ArchiveBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Archived Node
    // * link or unlink role
    // * set archive event and flag
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.")}

    // Extract tension blob characteristic
    tensionCharac, err := codec.TensionCharac{}.New(*tension.Action)
    if err != nil { return false, fmt.Errorf("tensionCharac unknown.") }

    switch tensionCharac.DocType {
    case codec.NodeDoc:
        ok, err = TryArchiveNode(uctx, tension, blob.Node)
    case codec.MdDoc:
        md := blob.Md
        ok, err = TryArchiveDoc(uctx, tension, md)
    }

    if err != nil { return ok, err }
    if ok { // Update blob archived flag
        err = db.GetDB().SetArchivedFlagBlob(blob.ID, Now(), tension.ID, tensionCharac.ArchiveAction(blob.Node.Type))
    }

    return ok, err
}

func UnarchiveBlob(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Unarchived Node
    // * link or unlink role
    // * set archive event and flag
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.")}

    // Extract tension blob characteristic
    tensionCharac, err := codec.TensionCharac{}.New(*tension.Action)
    if err != nil { return false, fmt.Errorf("tensionCharac unknown.") }

    switch tensionCharac.DocType {
    case codec.NodeDoc:
        ok, err = TryUnarchiveNode(uctx, tension, blob.Node)
    case codec.MdDoc:
        md := blob.Md
        ok, err = TryUnarchiveDoc(uctx, tension, md)
    }

    if err != nil { return ok, err }
    if ok { // Update blob pushed flag
        err = db.GetDB().SetPushedFlagBlob(blob.ID, Now(), tension.ID, tensionCharac.EditAction(blob.Node.Type))
    }

    return ok, err
}

func UserLeave(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    // Remove user reference
    // * remove User role
    // * unlink Orga role (Guest/Member) if role_type is Guest|Member
    // * upgrade user membership
    // --
    var ok bool

    blob := GetBlob(tension)
    if blob == nil { return false, fmt.Errorf("blob not found.")}
    node := blob.Node

    if model.RoleType(*event.Old) == model.RoleTypeGuest {
        rootid, e := codec.Nid2rootid(*event.New)
        if e != nil { return ok, e }
        i := auth.UserIsGuest(uctx, rootid)
        if i<0 {return ok, LogErr("Value error", fmt.Errorf("You are not a guest in this organisation.")) }
        var nf model.NodeFragment
        t := model.NodeTypeRole
        StructMap(uctx.Roles[i], &nf)
        nf.FirstLink = &uctx.Username
        nf.Type = &t
        node = &nf
    }

    ok, err := LeaveRole(uctx, tension, node)
    return ok, err
}


func UserJoin(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, b *model.BlobRef) (bool, error) {
    var ok bool
    // Only root node can be join
    // --

    rootid, err := codec.Nid2rootid(*event.New)
    if err != nil { return ok, err }
    if rootid != *event.New {return ok, LogErr("Value error", fmt.Errorf("guest user can only join the root circle.")) }
    i := auth.UserIsMember(uctx, rootid)
    if i>=0 {return ok, LogErr("Value error", fmt.Errorf("You are already a member of this organisation.")) }

    // Validate
    // --
    // check the invitation if a hash is given
    // * orga invtation ? <> user invitation hash ?
    // * else check if User Can Join Organisation
    if *tension.Receiver.UserCanJoin  {
        guestid := codec.MemberIdCodec(rootid, uctx.Username)
        ex, err :=  db.GetDB().Exists("Node.nameid", guestid, nil, nil)
        if err != nil { return ok, err }
        if ex {
            // Ensure a correct state for this Guest node.
            err = db.GetDB().UpgradeMember(guestid, model.RoleTypeGuest)
        } else {
            rt := model.RoleTypeGuest
            t := model.NodeTypeRole
            name := "Guest"
            n := &model.NodeFragment{
                Name: &name,
                RoleType: &rt,
                Type: &t,
                FirstLink: &uctx.Username,
            }
            auth.InheritNodeCharacDefault(n, tension.Receiver)
            err = PushNode(uctx, nil, n, "", guestid, rootid)
        }
        ok = true
    }

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
    return true, err
}

