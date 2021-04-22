package graph

import (
    "fmt"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
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
func tensionEventHook(uctx *model.UserCtx, tid string, events []*model.EventRef, bid *string) (bool, error) {
    var ok bool = true
    var err error
    var nameid string
    for _, event := range(events) {
        if *event.EventType == model.TensionEventBlobPushed ||
           *event.EventType == model.TensionEventBlobArchived ||
           *event.EventType == model.TensionEventBlobUnarchived ||
           *event.EventType == model.TensionEventUserLeft ||
           *event.EventType == model.TensionEventUserJoin ||
           *event.EventType == model.TensionEventMoved {
               // Process the special event
               ok, err, nameid = processTensionEventHook(uctx, event, tid, bid)
               if ok && err == nil {
                   // Set the Update time into the target node
                   err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.updatedAt", Now())
                   pid, _ := codec.Nid2pid(nameid) // @debug: real parent needed here (ie event for circle)
                   if pid != nameid && err == nil {
                       err = db.GetDB().SetFieldByEq("Node.nameid", pid, "Node.updatedAt", Now())
                   }
               }
               // Break after the first hooked event
               break
           }
        // else ... notify center
    }

    return ok, err
}

// Add, Update or Archived a Node
func processTensionEventHook(uctx *model.UserCtx, event *model.EventRef, tid string, bid *string) (bool, error, string) {
    var ok bool

    // Get Tension, target Node and blob charac (last if bid undefined)
    tension, err := db.GetDB().GetTensionHook(tid, bid)
    if err != nil { return false, LogErr("Access denied", err), tid}
    if tension == nil { return false, LogErr("Access denied", fmt.Errorf("tension not found.")), tid }


    // Check that Blob exists
    var blob *model.Blob
    var nameid string
    var node *model.NodeFragment
    var tensionCharac *codec.TensionCharac
    if tension.Blobs != nil {
        blob = tension.Blobs[0]
        bid = &blob.ID
        // Get Node and Nameid (from Codec)
        node = blob.Node
        if node != nil && node.Nameid != nil {
            _, nameid, err = codec.NodeIdCodec(tension.Receiver.Nameid, *node.Nameid, *node.Type)
        }

        // Extract tension blob characteristic
        tensionCharac, err = codec.TensionCharac{}.New(*tension.Action)
        if err != nil { return false, LogErr("internal error", err), nameid }
    }

    if *event.EventType == model.TensionEventBlobPushed {
        // Add or Update Node
        // --
        // 1. switch on TensionCharac.DocType (not blob type) -> rule differ from doc type!
        // 2. swith on TensionCharac.ActionType to add update etc...
        if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
        switch tensionCharac.ActionType {
        case codec.NewAction:
            // First time a blob is pushed.
            switch tensionCharac.DocType {
            case codec.NodeDoc:
                ok, err = TryAddNode(uctx, tension, node, bid)
            case codec.MdDoc:
                md := blob.Md
                ok, err = TryAddDoc(uctx, tension, md)
            }
        case codec.EditAction:
            switch tensionCharac.DocType {
            case codec.NodeDoc:
                ok, err = TryUpdateNode(uctx, tension, node, bid)
            case codec.MdDoc:
                md := blob.Md
                ok, err = TryUpdateDoc(uctx, tension, md)
            }
        case codec.ArchiveAction:
            err = LogErr("Access denied", fmt.Errorf("Cannot publish archived document."))
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob pushed flag
            err = db.GetDB().SetPushedFlagBlob(*bid, Now(), tid, tensionCharac.EditAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventBlobArchived {
        // Archived Node
        // --
        if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
        switch tensionCharac.DocType {
        case codec.NodeDoc:
            ok, err = TryArchiveNode(uctx, tension, node)
        case codec.MdDoc:
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
        if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
        switch tensionCharac.DocType {
        case codec.NodeDoc:
            ok, err = TryUnarchiveNode(uctx, tension, node)
        case codec.MdDoc:
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
        if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
        if model.RoleType(*event.Old) == model.RoleTypeGuest {
            rootid, e := codec.Nid2rootid(*event.New)
            if e != nil { return ok, e, nameid }
            i := auth.UserIsGuest(uctx, rootid)
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
        // --
        if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
        rootid, e := codec.Nid2rootid(*event.New)
        if e != nil { return ok, e, nameid }
        if rootid != *event.New {return ok, LogErr("Value error", fmt.Errorf("guest user can only join the root circle.")), nameid}
        i := auth.UserIsMember(uctx, rootid)
        if i>=0 {return ok, LogErr("Value error", fmt.Errorf("You are already a member of this organisation.")), nameid}

        // Validate
        // --
        // check the invitation if a hash is given
        // * orga invtation ? <> user invitation hash ?
        // * else check if User Can Join Organisation
        if tension.Receiver.Charac.UserCanJoin {
            guestid := codec.GuestIdCodec(rootid, uctx.Username)
            ex, e :=  db.GetDB().Exists("Node.nameid", guestid, nil, nil)
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
    } else if *event.EventType == model.TensionEventMoved {
        ok, err = TryMoveTension(uctx, tension, *event)
    }

    return ok, err, nameid
}

func TryMoveTension(uctx *model.UserCtx, t *model.Tension, event model.EventRef) (bool, error) {
    receiverid_old := *event.Old // == t.Receiverid
    receiverid_new := *event.New

    ok, err := CanChangeTension(uctx, t)
    if err != nil || !ok { return ok, err }

    // Update node and blob
    if t.Blobs != nil && t.Blobs[0].Node != nil {
        node := t.Blobs[0].Node
        _, nameid_old, err := codec.NodeIdCodec(receiverid_old, *node.Nameid, *node.Type)
        if err != nil { return false, err }
        _, nameid_new, err := codec.NodeIdCodec(receiverid_new, *node.Nameid, *node.Type)
        if err != nil { return false, err }

        // node input
        if receiverid_new == nameid_new {
            return false, fmt.Errorf("A node cannot be its own parent.")
        }
        nodeInput := model.UpdateNodeInput{
            Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid_old}},
            Set: &model.NodePatch{
                Parent: &model.NodeRef{Nameid: &receiverid_new},
            },
        }

        // update node
        err = db.GetDB().Update("node", nodeInput)
        if err != nil { return false, err }

        // DQL mutation (extra node update)
        if nameid_old != nameid_new { // node is a role
            err = db.GetDB().PatchNameid(nameid_old, nameid_new)
            if err != nil { return false, err }
        }

    }

    // tension input
    tensionInput := model.UpdateTensionInput{
        Filter: &model.TensionFilter{ID: []string{t.ID}},
        Set: &model.TensionPatch{
            Receiver: &model.NodeRef{Nameid: &receiverid_new},
            Receiverid: &receiverid_new,
        },
    }

    // update tension
    err = db.GetDB().Update("tension", tensionInput)
    return ok, err
}

func CanChangeTension(uctx *model.UserCtx, t *model.Tension) (bool, error) {
    var ok bool
    var err error

    // Check if the user is the creator of the ressource
    if uctx.Username == t.CreatedBy.Username {
        return true, err
    }

    // Check if the user is an assignee of the curent tension
    // @debug: use checkAssignee function, but how to pass the context ?
    var assignees []interface{}
    res, err := db.GetDB().GetSubFieldById(t.ID, "Tension.assignees", "User.username")
    if err != nil { return false, err }
    if res != nil { assignees = res.([]interface{}) }
    for _, a := range(assignees) {
        if a.(string) == uctx.Username {
            return true, err
        }
    }

    // Check if the user has the given (nested) role on the target node
    ok, err = CheckUserRights(uctx, t.Emitter.Nameid, nil)
    if ok || err != nil { return ok, err}
    ok, err = CheckUserRights(uctx, t.Receiver.Nameid, t.Receiver.Charac)
    if ok || err != nil { return ok, err}

    // Check if user has rights of any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(t.Receiver.Nameid) {
        ok, err = CheckUpperRights(uctx, t.Receiver.Nameid, t.Receiver.Charac)
    }

    return ok, err
}
