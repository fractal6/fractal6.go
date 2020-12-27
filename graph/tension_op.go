package graph

import (
    "fmt"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/db"
    . "zerogov/fractal6.go/tools"
)

// Take action based on the given Event
// * get tension node target NodeCharac and either
//      * last blob if bid is null
//      * given blob otherwiser
// * if event == blobPushed
//      * check user hasa the right authorization based on NodeCharac
//      * update the tension action value AND the blob pushedFlag
//      * copy the Blob data in the target Node.source (Uses GQL requests)
// * elif event == TensionEventBlobArchived/Unarchived
//     * link or unlink role
//     * set archive evnet and flag
// * elif event == TensionEventUserLeft
//    * remove User role
//    * unlink Orga role (Guest/Member) if role_type is Guest|Member
//    * upgrade user membership
// Note: @Debug: Only one BlobPushed will be processed
// Note: @Debug: remove added tension on error ?
func tensionEventHook(uctx model.UserCtx, tid string, events []*model.EventRef, bid *string) (bool, error) {
    var ok bool = true
    var err error
    var nameid string
    for _, event := range(events) {
        if *event.EventType == model.TensionEventBlobPushed ||
           *event.EventType == model.TensionEventBlobArchived ||
           *event.EventType == model.TensionEventBlobUnarchived ||
           *event.EventType == model.TensionEventUserLeft ||
           *event.EventType == model.TensionEventUserJoin {
               // Process the special event
               ok, err, nameid = processTensionEventHook(uctx, event, tid, bid)
               if ok && err == nil {
                   // Set the Update time into the target node
                   err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.updatedAt", Now())
                   pid, _ := nid2pid(nameid) // @debug: real parent needed here (ie event for circle)
                   if pid != nameid && err == nil {
                       err = db.GetDB().SetFieldByEq("Node.nameid", pid, "Node.updatedAt", Now())
                   }
               }
               // Break after the first hooked event
               break
           }
    }

    return ok, err
}

// Add, Update or Archived a Node
func processTensionEventHook(uctx model.UserCtx, event *model.EventRef, tid string, bid *string) (bool, error, string) {
    var nameid string
    // Get Tension, target Node and blob charac (last if bid undefined)
    tension, err := db.GetDB().GetTensionHook(tid, bid)
    if err != nil { return false, LogErr("Access denied", err), nameid}
    if tension == nil { return false, LogErr("Access denied", fmt.Errorf("tension not found.")), nameid }

    // Check that Blob exists
    blob := tension.Blobs[0]
    if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
    bid = &blob.ID

    // Extract Tension characteristic
    tensionCharac, err:= TensionCharac{}.New(*tension.Action)
    if err != nil { return false, LogErr("internal error", err), nameid }

    var ok bool
    var node *model.NodeFragment = blob.Node

    // Nameid Codec
    if node != nil && node.Nameid != nil {
        _, nameid, err = nodeIdCodec(tension.Receiver.Nameid, *node.Nameid, *node.Type)
    }

    if *event.EventType == model.TensionEventBlobPushed {
        // Add or Update Node
        // --
        // 1. switch on TensionCharac.DocType (not blob type) -> rule differ from doc type!
        // 2. swith on TensionCharac.ActionType to add update etc...
        switch tensionCharac.ActionType {
        case NewAction:
            // First time a blob is pushed.
            switch tensionCharac.DocType {
            case NodeDoc:
                ok, err = TryAddNode(uctx, tension, node, bid)
            case MdDoc:
                md := blob.Md
                ok, err = TryAddDoc(uctx, tension, md)
            }
        case EditAction:
            switch tensionCharac.DocType {
            case NodeDoc:
                ok, err = TryUpdateNode(uctx, tension, node, bid)
            case MdDoc:
                md := blob.Md
                ok, err = TryUpdateDoc(uctx, tension, md)
            }
        case ArchiveAction:
            err = LogErr("Access denied", fmt.Errorf("Cannot publish archived document."))
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob pushed flag
            err = db.GetDB().SetPushedFlagBlob(*bid, Now(), tid, tensionCharac.EditAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventBlobArchived {
        // Archived Node
        // --
        switch tensionCharac.DocType {
        case NodeDoc:
            ok, err = TryArchiveNode(uctx, tension, node)
        case MdDoc:
            md := blob.Md
            ok, err = TryArchiveDoc(uctx, tension, md)
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob archived flag
            err = db.GetDB().SetArchivedFlagBlob(*bid, Now(), tid, tensionCharac.ArchiveAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventBlobUnarchived {
        // Unarchived Node
        // --
        switch tensionCharac.DocType {
        case NodeDoc:
            ok, err = TryUnarchiveNode(uctx, tension, node)
        case MdDoc:
            md := blob.Md
            ok, err = TryUnarchiveDoc(uctx, tension, md)
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob pushed flag
            err = db.GetDB().SetPushedFlagBlob(*bid, Now(), tid, tensionCharac.EditAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventUserLeft {
        // Remove user reference
        // --
        if model.RoleType(*event.Old) == model.RoleTypeGuest {
            rootid, e := nid2rootid(*event.New)
            if e != nil { return ok, e, nameid }
            i := userIsGuest(uctx, rootid)
            if i<0 {return ok, LogErr("Value error", fmt.Errorf("You are not a guest in this organisation.")), nameid}
            var nf model.NodeFragment
            var t model.NodeType = model.NodeTypeRole
            StructMap(uctx.Roles[i], &nf)
            nf.FirstLink = &uctx.Username
            nf.Type = &t
            node = &nf
        }

        ok, err = LeaveRole(uctx, tension, node)
    } else if *event.EventType == model.TensionEventUserJoin {
        // Only root node can be join
        rootid, e := nid2rootid(*event.New)
        if e != nil { return ok, e, nameid }
        if rootid != *event.New {return ok, LogErr("Value error", fmt.Errorf("guest user can only join the root circle.")), nameid}
        i := userIsMember(uctx, rootid)
        if i>=0 {return ok, LogErr("Value error", fmt.Errorf("You are already a member of this organisation.")), nameid}

        // Validate
        // --
        // check the invitation if a hash is given
        // * orga invtation ? <> user invitation hash ?
        // * else check if User Can Join Organisation
        if tension.Receiver.Charac.UserCanJoin {
            guestid := guestIdCodec(rootid, uctx.Username)
            ex, e :=  db.GetDB().Exists("Node", "nameid", guestid, nil, nil)
            if e != nil { return ok, e, nameid }
            if ex {
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
                    IsPrivate: &tension.Receiver.IsPrivate,
                    Charac: tension.Receiver.Charac,
                }
                err = PushNode(uctx, nil, n, "", guestid, rootid)
            }
            ok = true
        }
    }

    return ok, err, nameid
}
