package graph

import (
    "fmt"
    "strconv"
    "strings"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    webauth "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/text/en"
    . "zerogov/fractal6.go/tools"
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

    err = PushNode(uctx, bid, node, emitterid, nameid, parentid)
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
    return ok, err
}

func TryArchiveNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    // Check that circle has no children
    if *node.Type == model.NodeTypeCircle {
        children, err := db.GetDB().GetChildren(nameid)
        if err != nil { return ok, err }
        if len(children) > 0 {
            return ok, fmt.Errorf("Cannot archive circle with active children. Please archive children first.")
        }
    }

    // Archive Node
    // --
    if node.FirstLink != nil {
        err := UnlinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil { return false, err }
    }
    // Toggle the node flag
    err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.isArchived", strconv.FormatBool(true))
    return ok, err
}

func TryUnarchiveNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    // Check that parent node is not archived
    parentIsArchived, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid, "Node.parent", "Node.isArchived")
    if err != nil { return ok, err }
    if parentIsArchived != nil && parentIsArchived.(bool) == true{
        return ok, fmt.Errorf("Cannot unarchive node with archived parent. Please unarchive parent first.")
    }

    // Unarchive Node
    // --
    if node.FirstLink != nil {
        err := LinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil { return false, err }
    }
    // Toggle the node flag
    err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.isArchived", strconv.FormatBool(false))
    return ok, err
}

func TryChangeAuthority(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, value string) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := NodeCheck(uctx, node, nameid, tension.Action)
    if err != nil || !ok { return ok, err }

    DB := db.GetDB()

    switch *node.Type {
    case model.NodeTypeRole:
        if !model.RoleType(value).IsValid() { return false, fmt.Errorf("Bad value for role_type.") }
        err = DB.SetFieldByEq("Node.nameid", nameid, "Node.role_type", value)
        if err != nil { return false, err }
        err = DB.SetSubFieldByEq("Node.nameid", nameid, "Node.role_ext", "RoleExt.role_type", value)
        if err != nil { return false, err }
        err = DB.SetFieldById(node.ID, "NodeFragment.role_type", value)
    case model.NodeTypeCircle:
        if !model.NodeMode(value).IsValid() { return false, fmt.Errorf("Bad value for mode.") }
        err = DB.SetFieldByEq("Node.nameid", nameid, "Node.mode", value)
        if err != nil { return false, err }
        err = DB.SetFieldById(node.ID, "NodeFragment.mode", value)
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

    DB := db.GetDB()

    if !model.NodeVisibility(value).IsValid() { return false, fmt.Errorf("Bad value for visibility.") }
    err = DB.SetFieldByEq("Node.nameid", nameid, "Node.visibility", value)
    if err != nil { return false, err }
    err = DB.SetFieldById(node.ID, "NodeFragment.visibility", value)

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
                err = fmt.Errorf("role should have a RoleType.")
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
func PushNode(uctx *model.UserCtx, bid *string, node *model.NodeFragment, emitterid, nameid, parentid string) (error) {
    rootnameid, _ := codec.Nid2rootid(nameid)

    // Map NodeFragment to Node Input
    var nodeInput model.AddNodeInput
    StructMap(node, &nodeInput)

    // Fix Automatic fields
    nodeInput.CreatedAt = Now()
    nodeInput.CreatedBy = &model.UserRef{Username: &uctx.Username}
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
    nodeInput.IsRoot = false
    nodeInput.IsArchived = false
    nodeInput.Rights = 0
    if bid != nil {
        nodeInput.Source = &model.BlobRef{ ID: bid }
    }
    var children []model.NodeFragment
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil {
            nodeInput.FirstLink = &model.UserRef{Username: node.FirstLink}
        }
    case model.NodeTypeCircle:
        nodeInput.Children = nil
        for i, c := range(node.Children) {
            if c.FirstLink != nil {
                child := makeNewChild(i, *c.FirstLink, nameid, *c.RoleType, node)
                children = append(children, child)
            }
        }
    }

    // Push the nodes into the database
    _, err := db.GetDB().Add(db.DB.GetRootUctx(), "node", nodeInput)
    if err != nil { return err }

    // Change Guest to member if user got its first role.
    // add tension and child for existing children.
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil && *node.RoleType != model.RoleTypeGuest {
            err = maybeUpdateMembership(rootnameid, *node.FirstLink, model.RoleTypeMember)
        }
    case model.NodeTypeCircle:
        for _, child := range(children) {
            // Add the child tension
            tensionInput := makeNewChildTension(uctx, emitterid, nameid, child)
            tid_c, err := db.GetDB().Add(db.DB.GetRootUctx(), "tension", tensionInput)
            if err != nil { return err }
            // Push child
            bid_c := db.GetDB().GetLastBlobId(tid_c)
            err = PushNode(uctx, bid_c, &child, emitterid, *child.Nameid, nameid)
        }
    }

    return err
}

// updateNode update a node from the given fragment
func UpdateNode(uctx *model.UserCtx, bid *string, node *model.NodeFragment, emitterid, nameid string) (error) {
    // Map NodeFragment to Node Patch Input
    var nodePatchFilter model.NodePatchFromFragment
    var nodePatch model.NodePatch
    delMap := make(map[string]interface{}, 2)
    StructMap(node, &nodePatchFilter)
    StructMap(nodePatchFilter, &nodePatch)

    // Blob reference update
    if bid != nil {
        nodePatch.Source = &model.BlobRef{ ID: bid }
    }

    //// Fix automatic fields
    //switch *node.Type {
    //case model.NodeTypeRole:
    //    if node.FirstLink != nil {
    //        nodePatch.FirstLink = &model.UserRef{ Username: node.FirstLink }
    //    } else {
    //        // if first_link is empty, remove it.
    //        delMap["Node.first_link"] = nil
    //    }
    //case model.NodeTypeCircle:
    //    nodePatch.Children = nil
    //}

    // Get the first link prior updating the node
    firstLink_, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid, "Node.first_link", "User.username")
    if err != nil { return err }

    // Build input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Set: &nodePatch,
        //Remove: &delNodePatch, // @debug: omitempty issues
    }
    // Update the node in database
    err = db.GetDB().Update(db.DB.GetRootUctx(), "node", nodeInput)
    if err != nil { return err }

    rootnameid, _ := codec.Nid2rootid(nameid)
    if len(delMap) > 0 { // delete the node reference
        if firstLink_ != nil {
            err := UnlinkUser(rootnameid, nameid, firstLink_.(string))
            if err != nil { return err }
        }
    } else if node.FirstLink != nil  {
        // @debug: if the firstlink user has already this role,
        //         the update is useless
        err := LinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil { return err }
        if firstLink_ != nil && firstLink_.(string) != *node.FirstLink {
            // Someone loose his role here...contract ?
            // @DEBUG: does the role of the previsou firstLinnk get removed ???!
            err = maybeUpdateMembership(rootnameid, firstLink_.(string), model.RoleTypeGuest)
        }
    }

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

func makeNewChildTension(uctx *model.UserCtx, emitterid string, receiverid string, child model.NodeFragment) model.AddTensionInput {
    now := Now()
    createdBy := model.UserRef{Username: &uctx.Username}
    emitter := model.NodeRef{Nameid: &emitterid}
    receiver := model.NodeRef{Nameid: &receiverid}
    action := model.TensionActionEditRole
    evt1 := model.TensionEventCreated
    evt2 := model.TensionEventBlobCreated
    evt3 := model.TensionEventBlobPushed
    blob_type := model.BlobTypeOnNode
    var childref model.NodeFragmentRef
    StructMap(child, &childref)
    parts := strings.Split(*child.Nameid, "#")
    childref.Nameid = &parts[len(parts)-1]
    blob := model.BlobRef{
        CreatedAt: &now,
        CreatedBy : &createdBy,
        BlobType: &blob_type,
        Node: &childref,
        PushedFlag: &now,
    }
    tension := model.AddTensionInput{
        CreatedAt: now,
        CreatedBy : &createdBy,
        Title: "[Role] "+ *child.Name,
        Type: model.TensionTypeGovernance,
        Status: model.TensionStatusClosed,
        Emitter: &emitter,
        Receiver: &receiver,
        Emitterid: emitterid,
        Receiverid: receiverid,
        Action: &action,
        History : []*model.EventRef{
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt1},
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt2},
            &model.EventRef{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt3},
        },
        Blobs: []*model.BlobRef{&blob},
        Comments:  []*model.CommentRef{&model.CommentRef{CreatedAt: &now, CreatedBy: &createdBy, Message: nil }},
    }
    return tension
}

func MakeNewRootTension(uctx *model.UserCtx, rootnameid string, node model.AddNodeInput) model.AddTensionInput {
    now := Now()
    createdBy := *node.CreatedBy
    emitter := model.NodeRef{Nameid: &rootnameid}
    receiver := model.NodeRef{Nameid: &rootnameid}
    action := model.TensionActionEditRole
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
        Title: "[Circle] Anchor node",
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
        Comments:  []*model.CommentRef{&model.CommentRef{CreatedAt: &now, CreatedBy: &createdBy, Message: nil }},
    }
    return tension
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
