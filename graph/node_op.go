package graph

import (
    "fmt"
    "strconv"
    "strings"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    webauth "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/text/en"
    . "zerogov/fractal6.go/tools"
)

// tryAddNode add a new node is user has the correct right
// * it inherits node charac
func TryAddNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, bid *string) (bool, error) {
    //tid := tension.ID
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac
    isPrivate := tension.Receiver.IsPrivate

    // Inherits node properties
    if node.Charac == nil {
        node.Charac = charac
    }
    if node.IsPrivate == nil {
        node.IsPrivate = &isPrivate
    }

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, true)
    if err != nil || !ok { return ok, err }

    err = PushNode(uctx, bid, node, emitterid, nameid, parentid)
    return ok, err
}

func TryUpdateNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, bid *string) (bool, error) {
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac

    // Get References
    _, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
    if err != nil || !ok { return ok, err }

    err = UpdateNode(uctx, bid, node, emitterid, nameid, parentid)
    return ok, err
}

func TryArchiveNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
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
    charac := tension.Receiver.Charac

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
    if err != nil || !ok { return ok, err }

    // Check that node has no parent archived
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

// canAddNode check that a user can add a given role or circle in an organisation.
func CanAddNode(uctx *model.UserCtx, node *model.NodeFragment, nameid, parentid string, charac *model.NodeCharac, isNew bool) (bool, error) {
    var ok bool = false
    var err error

    name := *node.Name
    nodeType := *node.Type
    roleType := node.RoleType

    rootnameid, _ := codec.Nid2rootid(nameid)
    err = webauth.ValidateNameid(nameid, rootnameid)
    if err != nil { return ok, err }
    err = webauth.ValidateName(name)
    if err != nil { return ok, err }

    // RoleType Hook
    if nodeType == model.NodeTypeRole {
        // Validate input
        if roleType == nil {
            err = fmt.Errorf("role should have a RoleType.")
        }
    } else if nodeType == model.NodeTypeCircle {
        //pass
    }

    // Check user rights
    ok, err = CheckUserRights(uctx, parentid, charac)
    if err != nil { return ok, LogErr("Internal error", err) }

    // Check if user has rights of any parents if the node has no Coordo role.
    if !ok {
        parents, err := db.GetDB().GetParents(parentid)
        // Check of pid has coordos
        if len(parents) > 0 && !db.GetDB().HasCoordos(parentid) {
            // @debug: move to CheckCoordoPath function
            if err != nil { return ok, LogErr("Internal Error", err) }
            for _, p := range(parents) {
                ok, err = CheckUserRights(uctx, p, charac)
                if err != nil { return ok, LogErr("Internal error", err) }
                if ok { break }
            }
        }
    }

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
    nodeInput.IsRoot = false
    nodeInput.IsArchived = false
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.CreatedAt = Now()
    nodeInput.CreatedBy = &model.UserRef{Username: &uctx.Username}
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
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
                child := makeNewChild(i, *c.FirstLink, nameid, *c.RoleType, node.Charac, node.IsPrivate)
                children = append(children, child)
            }
        }
    }

    // Push the nodes into the database
    _, err := db.GetDB().Add("node", nodeInput)
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
            tid_c, err := db.GetDB().Add("tension", tensionInput)
            if err != nil { return err }
            // Push child
            bid_c := db.GetDB().GetLastBlobId(tid_c)
            err = PushNode(uctx, bid_c, &child, emitterid, *child.Nameid, nameid)
        }
    }

    return err
}


// updateNode update a node from the given fragment
// @DEBUG: only set the field that have been modified in NodePatch
func UpdateNode(uctx *model.UserCtx, bid *string, node *model.NodeFragment, emitterid, nameid, parentid string) (error) {
    // Map NodeFragment to Node Patch Input
    var nodePatch model.NodePatch
    delMap := make(map[string]interface{}, 2)
    StructMap(node, &nodePatch)

    // Blob reference update
    if bid != nil {
        nodePatch.Source = &model.BlobRef{ ID: bid }
    }

    // Fix automatic fields
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil {
            nodePatch.FirstLink = &model.UserRef{ Username: node.FirstLink }
        } else {
            // if first_link is empty, remove it.
            delMap["Node.first_link"] = nil
        }
    case model.NodeTypeCircle:
        nodePatch.Children = nil
    }

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
    err = db.GetDB().Update("node", nodeInput)
    if err != nil { return err }

    rootnameid, _ := codec.Nid2rootid(nameid)
    if len(delMap) > 0 { // delete the node reference
        //err = db.GetDB().DeleteEdges("Node.nameid", nameid, delMap) // that do not delete the reverse edge (User.roles)
        if firstLink_ != nil {
            err = webauth.RemoveUserRole(firstLink_.(string), nameid)
            if err != nil { return err }
            err = maybeUpdateMembership(rootnameid, firstLink_.(string), model.RoleTypeGuest)
        }

    } else if node.FirstLink != nil  {
        // @debug: if the firstlink user has already this role,
        //         the update is useless
        err = webauth.AddUserRole(*node.FirstLink, nameid)
        if err != nil { return err }
        err = maybeUpdateMembership(rootnameid, *node.FirstLink, model.RoleTypeMember)
        if err != nil { return err }
        if firstLink_ != nil && firstLink_.(string) != *node.FirstLink {
            err = maybeUpdateMembership(rootnameid, firstLink_.(string), model.RoleTypeGuest)
        }
    }

    return err
}

//
// Internals
//

func makeNewChild(i int, username string, parentid string, roleType model.RoleType, charac *model.NodeCharac, isPrivate *bool) model.NodeFragment {
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
        Charac: charac,
        IsPrivate: isPrivate,
    }
    if roleType == model.RoleTypeCoordinator {
        mandate := model.Mandate{Purpose: en.CoordoPurpose}
        child.Mandate = &mandate
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
