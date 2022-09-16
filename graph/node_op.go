package graph

import (
    "fmt"
    "strconv"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/graph/auth"
    webauth "fractale/fractal6.go/web/auth"
    "fractale/fractal6.go/db"
    "fractale/fractal6.go/text/en"
    . "fractale/fractal6.go/tools"
)

// tryAddNode add a new node if user has the correct right
func TryAddNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, bid *string) (bool, error) {
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid

    auth.InheritNodeCharacDefault(node, tension.Receiver)

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    err = PushNode(uctx.Username, bid, node, emitterid, nameid, parentid)
    if err == nil {
        // Update tension title
        err = db.GetDB().SetFieldById(tension.ID, "Tension.title", codec.UpdateTensionTitle(*node.Type, *node.Nameid == "", *node.Name))
    }
    return ok, err
}

func TryUpdateNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, bid *string) (bool, error) {
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    err = UpdateNode(uctx, bid, node, emitterid, nameid)
    if err == nil {
        // Update tension title
        err = db.GetDB().SetFieldById(tension.ID, "Tension.title", codec.UpdateTensionTitle(*node.Type, *node.Nameid == "", *node.Name))
    }
    return ok, err
}

func TryChangeArchiveNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, eventType model.TensionEvent) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    var archiveFlag string

    if eventType == model.TensionEventBlobArchived {
        // Archive
        // --
        // Check that circle has no children
        if *node.Type == model.NodeTypeCircle {
            children, err := db.GetDB().GetChildren(nameid)
            if err != nil { return ok, err }
            if len(children) > 0 {
                return ok, fmt.Errorf("Cannot archive circle with active children. Please archive children first.")
            }
        }
        archiveFlag = strconv.FormatBool(true)

        // Eventually Unlink first-link
        if node.FirstLink != nil {
            err = UnlinkUser(rootnameid, nameid, *node.FirstLink)
            if err != nil { return ok, err }
        }
    } else if eventType == model.TensionEventBlobUnarchived {
        // Unarchive
        // --
        // Check that parent node is not archived
        parentIsArchived, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid, "Node.parent", "Node.isArchived")
        if err != nil { return ok, err }
        if parentIsArchived != nil && parentIsArchived.(bool) == true{
            return ok, fmt.Errorf("Cannot unarchive node with archived parent. Please unarchive parent first.")
        }
        archiveFlag = strconv.FormatBool(false)
    } else {
        return false, fmt.Errorf("bad tension event '%s'.", string(eventType))
    }

    // Set the archive flag
    err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.isArchived", archiveFlag)
    return ok, err
}

func TryChangeAuthority(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, value string) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    switch *node.Type {
    case model.NodeTypeRole:
        if !model.RoleType(value).IsValid() { return false, fmt.Errorf("Bad value for role_type.") }
        err = db.DB.SetFieldByEq("Node.nameid", nameid, "Node.role_type", value)
        if err != nil { return false, err }
        err = db.DB.SetSubFieldByEq("Node.nameid", nameid, "Node.role_ext", "RoleExt.role_type", value)
        if err != nil { return false, err }
        err = db.DB.SetFieldById(node.ID, "NodeFragment.role_type", value)
    case model.NodeTypeCircle:
        if !model.NodeMode(value).IsValid() { return false, fmt.Errorf("Bad value for mode.") }
        err = db.DB.SetFieldByEq("Node.nameid", nameid, "Node.mode", value)
        if err != nil { return false, err }
        err = db.DB.SetFieldById(node.ID, "NodeFragment.mode", value)
    }

    return ok, err
}

func TryChangeVisibility(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, value string) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    visibility := model.NodeVisibility(value)
    if !visibility.IsValid() { return false, fmt.Errorf("Bad value for visibility.") }
    // Update Node
    _, err = db.GetDB().Meta("setNodeVisibility", map[string]string{"nameid":nameid, "value":value})
    if err != nil { return false, err }

    // If nameid is the root, fix the organisation config.
    rootid, _ := codec.Nid2rootid(nameid)
    if visibility != model.NodeVisibilityPublic && nameid == rootid {
        err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.userCanJoin", strconv.FormatBool(false))
        if err != nil { return false, err }
    }

    // Change all role direct children
    err = db.DB.SetChildrenRoleVisibility(nameid, value)
    return ok, err
}

func TryUpdateLink(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, event *model.EventRef) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    // Get the current first link
    firstLink, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid, "Node.first_link", "User.username")
    if err != nil { return false, err }

    if *event.EventType == model.TensionEventMemberLinked {
        // Link user
        // --
        if firstLink != nil {return false, fmt.Errorf("Role is already linked.")}
        err = LinkUser(rootnameid, nameid, *event.New)
        if err != nil { return false, err }
    } else if *event.EventType == model.TensionEventMemberUnlinked {
        // UnLink user
        // --
        err = UnlinkUser(rootnameid, nameid, *event.Old)
        if err != nil { return false, err }
    }

    // Update NodeFragment
    // @debug: should delete instead...
    if node.ID != "" {
        err = db.GetDB().SetFieldById(node.ID, "NodeFragment.first_link", *event.New)
    }
    return ok, err
}

// NodeCheck validate and type checks.
func NodeCheck(uctx *model.UserCtx, node *model.NodeFragment, nameid string, action *model.TensionAction) (bool, error) {
    var ok bool = false
    var err error

    name := *node.Name
    rootnameid, _ := codec.Nid2rootid(nameid)

    err = webauth.ValidateNameid(nameid, rootnameid)
    if err != nil { return ok, err }
    err = webauth.ValidateName(name)
    if err != nil { return ok, err }

    if *action == model.TensionActionNewRole {
        // RoleType Hook
        nodeType := *node.Type
        roleType := node.RoleType
        if nodeType == model.NodeTypeRole {
            // Validate input
            if roleType == nil {
                err = fmt.Errorf("role must have a RoleType.")
            }
        } else if nodeType == model.NodeTypeCircle {
            //pass
        }
    }

    ok = true
    return ok, err
}

// pushNode add a new role or circle in an graph.
// * It adds automatic fields such as createdBy, createdAt, etc
// * It automatically add tension associated to potential children.
func PushNode(username string, bid *string, node *model.NodeFragment, emitterid, nameid, parentid string) (error) {
    rootnameid, _ := codec.Nid2rootid(nameid)

    // Map NodeFragment to Node Input
    var nodeInput model.AddNodeInput
    StructMap(node, &nodeInput)

    // Fix Automatic fields
    nodeInput.CreatedAt = Now()
    nodeInput.CreatedBy = &model.UserRef{Username: &username}
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
    nodeInput.IsRoot = false
    nodeInput.IsArchived = false
    nodeInput.Rights = 0
    if node.RoleExt != nil {
        nodeInput.RoleExt = &model.RoleExtRef{ID: node.RoleExt}
    }
    if bid != nil {
        nodeInput.Source = &model.BlobRef{ ID: bid }
    }
    // @future: children not implemented
    //var children []model.NodeFragment

    // @future: for the moment, first_link and children are linked separetly, see contracts....
    //switch *node.Type {
    //case model.NodeTypeRole:
    //    if node.FirstLink != nil {
    //        nodeInput.FirstLink = &model.UserRef{Username: node.FirstLink}
    //    }
    //case model.NodeTypeCircle:
    //    nodeInput.Children = nil
    //    for i, c := range(node.Children) {
    //        if c.FirstLink != nil {
    //            child := makeNewChild(i, *c.FirstLink, nameid, *c.RoleType, node)
    //            children = append(children, child)
    //        }
    //    }
    //}

    // Push the nodes into the database
    _, err := db.GetDB().Add(db.DB.GetRootUctx(), "node", nodeInput)
    if err != nil { return err }

    return err
}

// updateNode update a node from the given fragment
func UpdateNode(uctx *model.UserCtx, bid *string, node *model.NodeFragment, emitterid, nameid string) (error) {
    // Map NodeFragment to Node Patch Input
    var nodePatchFilter model.NodePatchFromFragment
    var nodePatch model.NodePatch
    StructMap(node, &nodePatchFilter)
    StructMap(nodePatchFilter, &nodePatch)
    // Blob reference update
    if bid != nil {
        nodePatch.Source = &model.BlobRef{ ID: bid }
    }
    // Build input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Set: &nodePatch,
        //Remove: &delNodePatch, // @debug: omitempty issues
    }
    // Update the node in database
    err := db.GetDB().Update(db.DB.GetRootUctx(), "node", nodeInput)
    return err
}

//
// Internals
//

func makeNewChild(i int, username string, parentid string, roleType model.RoleType, node *model.NodeFragment) model.NodeFragment {
    //name := "Coordinator"
    name := string(roleType)
    nameid := parentid +"#"+ name + strconv.Itoa(i)
    type_ := model.NodeTypeRole
    fs := username
    child := model.NodeFragment{
        Name: &name,
        Nameid: &nameid,
        Type: &type_,
        RoleType: &roleType,
        FirstLink: &fs,
    }
    var nodeProxy *model.Node
    StructMap(node, &nodeProxy)
    auth.InheritNodeCharacDefault(&child, nodeProxy)
    if roleType == model.RoleTypeCoordinator {
        child.Mandate = &model.Mandate{Purpose: en.CoordoPurpose}
    }
    return child
}

// MakeNewRootTension build the tension that manage a root node.
// Authors will be suscribed.
func MakeNewRootTension(rootnameid string, node model.AddNodeInput) model.AddTensionInput {
    now := Now()
    createdBy := *node.CreatedBy
    emitter := model.NodeRef{Nameid: &rootnameid}
    receiver := model.NodeRef{Nameid: &rootnameid}
    action := model.TensionActionEditCircle
    evt1 := model.TensionEventCreated
    evt2 := model.TensionEventBlobCreated
    evt3 := model.TensionEventBlobPushed
    blob_type := model.BlobTypeOnNode
    var noderef model.NodeFragmentRef
    StructMap(node, &noderef)
    emptyString := ""
    noderef.Nameid = &emptyString
    blob := model.BlobRef{
        CreatedAt: &now,
        CreatedBy : &createdBy,
        BlobType: &blob_type,
        Node: &noderef,
        PushedFlag: &now,
    }
    tension := model.AddTensionInput{
        CreatedAt: now,
        CreatedBy : &createdBy,
        Title: codec.UpdateTensionTitle(model.NodeTypeCircle, true, node.Name),
        Type: model.TensionTypeGovernance,
        Status: model.TensionStatusClosed,
        Emitter: &emitter,
        Receiver: &receiver,
        Emitterid: rootnameid,
        Receiverid: rootnameid,
        Action: &action,
        History : []*model.EventRef{
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt1},
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt2},
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt3},
        },
        Blobs: []*model.BlobRef{&blob},
        Comments:  []*model.CommentRef{&model.CommentRef{CreatedAt: &now, CreatedBy: &createdBy, Message: nil}},
        Subscribers: []*model.UserRef{&createdBy},
    }
    return tension
}

func MaybeAddPendingNode(username string, tension *model.Tension) (bool, error) {
    ok := false
    if tension.Receiver == nil {
        var tension_m []map[string]interface{}
        var err error
        if tension_m, err = db.GetDB().Meta("getTensionSimple", map[string]string{"id": tension.ID}); err != nil {
            return ok, err
        } else if len(tension_m) == 0 {
            return ok, fmt.Errorf("no tension found for tid: %s", tension.ID)
        }
        if err = Map2Struct(tension_m[0], tension); err != nil {
            return ok, err
        }
    }

    rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
    if err != nil { return ok, err }
    nid := codec.MemberIdCodec(rootid, username)
    n, err := db.GetDB().GetFieldByEq("Node.nameid", nid, "Node.role_type Node.first_link{User.username}")
    if err != nil { return ok, err }
    node, _ := n.(model.JsonAtom)
    if node["role_type"] == nil {
        rt := model.RoleTypePending
        t := model.NodeTypeRole
        name := "Pending"
        n := &model.NodeFragment{
            Name: &name,
            RoleType: &rt,
            Type: &t,
        }
        auth.InheritNodeCharacDefault(n, tension.Receiver)
        err = PushNode(username, nil, n, "", nid, rootid)
        if err != nil { return ok, err }
        err = db.GetDB().AddUserRole(username, nid)
        ok = true
    } else if node["first_link"] == nil {
        err = db.GetDB().AddUserRole(username, nid)
        err = db.GetDB().UpgradeMember(nid, model.RoleTypePending)
    } else if node["role_type"].(string) == string(model.RoleTypeRetired) {
        err = db.GetDB().UpgradeMember(nid, model.RoleTypePending)
    }

    return ok, err
}

func MaybeDeletePendingNode(username string, tension *model.Tension) error {
    rootid, err := codec.Nid2rootid(tension.Receiverid)
    if err != nil { return err }
    nid := codec.MemberIdCodec(rootid, username)
    filter := fmt.Sprintf(`eq(%s, "%s")`, "Node.role_type", "Pending")
    ex, err :=  db.GetDB().Exists("Node.nameid", nid, &filter)
    if err != nil { return err }
    if ex {
        // REMOVING node have unattended effect (emitter missing)
        //err := db.GetDB().RemoveUserRole(username, nid)
        //if err != nil { return err }
        //err = db.DB.Delete(db.DB.GetRootUctx(), "node", model.NodeFilter{
        //    Nameid: &model.StringHashFilterStringRegExpFilter{Eq:&nid},
        //})
        err = UnlinkUser(rootid, nid, username)
    }

    return err
}

//
// Utils
//

// Set the map keys to be compliant with Dgraph.
// @obsolete
func encodeNodeMap(m map[string]interface{}, prefix string) map[string]interface{} {
    out := make(map[string]interface{}, len(m))
    for k, v := range m {
        nk := prefix + "." + k

        var nv interface{}
        switch t := v.(type) {
        case map[string]interface{}:
            var p string
            if k == "parent" || k == "receiver" || k == "emitter" {
                p = "Node"
            } else if k == "charac" {
                p = "NodeCharac"
            } else if k == "mandate" {
                p = "Mandate"
            } else if k == "createdBy" || k == "first_link" || k == "second_link" {
                p = "User"
            }
            nv = encodeNodeMap(t, p)
        case []map[string]interface{}:
            var nv_ []map[string]interface{}
            var p string
            for _, s := range(v.([]map[string]interface{})) {
                if k == "docs" {
                    p = "Tension"
                } else if k == "labels" {
                    p = "Label"
                } else if k == "comments" {
                    p = "Comment"
                } else if k == "blobs" {
                    p = "Blob"
                } else if k == "history" {
                    p = "Event"
                }
                ns := encodeNodeMap(s, p)
                nv_ = append(nv_, ns)
            }
            nv = nv_
        default:
            nv = t
        }
        out[nk] =  nv
    }

    return out
}
