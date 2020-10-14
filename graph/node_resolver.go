package graph

import (
    "fmt"
    "time"
    "strconv"
    "strings"
    //"github.com/mitchellh/mapstructure"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/text/en"
    . "zerogov/fractal6.go/tools"
)

// tryAddNode add a new node is user has the correct right
// * it inherits node charac
func TryAddNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    tid := tension.ID
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
    _, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, true)
    if err != nil || !ok {
        return ok, err
    }

    err = PushNode(uctx, tid, node, emitterid, nameid, parentid)
    return ok, err
}

func TryUpdateNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac

    // Get References
    _, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
    if err != nil || !ok {
        return ok, err
    }

    err = UpdateNode(uctx, node, emitterid, nameid, parentid)
    return ok, err
}

func TryArchiveNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac

    // Get References
    _, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
    if err != nil || !ok {
        return ok, err
    }

    // Archive Node
    err = db.GetDB().SetNodeLiteral(nameid, "isArchived", strconv.FormatBool(true))
    // remove user role
    return ok, err
}
func TryUnarchiveNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid
    charac := tension.Receiver.Charac

    // Get References
    _, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return false, err }

    ok, err := CanAddNode(uctx, node, nameid, parentid, charac, false)
    if err != nil || !ok {
        return ok, err
    }

    // Unarchive Node
    err = db.GetDB().SetNodeLiteral(nameid, "isArchived", strconv.FormatBool(false))
    // add user role
    return ok, err
}

// canAddNode check that a user can add a given role or circle in an organisation.
func CanAddNode(uctx model.UserCtx, node *model.NodeFragment, nameid, parentid string, charac *model.NodeCharac, isNew bool) (bool, error) {
    var ok bool = false
    var err error

    name := *node.Name
    nodeType := *node.Type
    roleType := node.RoleType

    rootnameid, _ := nid2rootid(nameid)
    err = auth.ValidateNameid(nameid, rootnameid)
    if err != nil { return ok, err }
    err = auth.ValidateName(name)
    if err != nil { return ok, err }

    // RoleType Hook
    if nodeType == model.NodeTypeRole {
        // Validate input
        if roleType == nil {
            err = fmt.Errorf("role should have a RoleType")
        }
    } else if nodeType == model.NodeTypeCircle {
        //pass
    }

    // Add node Policies
    if charac.Mode == model.NodeModeChaos {
        ok = userIsMember(uctx, parentid)
    } else if charac.Mode == model.NodeModeCoordinated {
        ok = userIsCoordo(uctx, parentid)
    }

    // Check if user is Coordinator of any parents if the PID has no coordinator
    if !ok {
        parents, err := db.GetDB().GetParents(parentid)
        // Check of pid has coordos
        if len(parents) > 0 && !db.GetDB().HasCoordos(parents[0]) {
            // @debug: move to checkCoordoPath
            if err != nil { return ok, LogErr("Internal Error", err) }
            for _, p := range(parents) {
                if userIsCoordo(uctx, p) {
                    ok = true
                    break
                }
            }
        }
    }

    return ok, err
}

// pushNode add a new role or circle in an graph.
// * It adds automatic fields such as createdBy, createdAt, etc
// * It automatically add tension associated to potential children.
func PushNode(uctx model.UserCtx, tid string, node *model.NodeFragment, emitterid, nameid, parentid string) (error) {
    rootnameid, _ := nid2rootid(nameid)

    // Map NodeFragment to Node Input
    var nodeInput model.AddNodeInput
    StructMap(node, &nodeInput)

    // Fix Automatic fields
    nodeInput.IsRoot = false
    nodeInput.IsArchived = false
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.CreatedAt = time.Now().Format(time.RFC3339)
    nodeInput.CreatedBy = &model.UserRef{Username: &uctx.Username}
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
    nodeInput.Source = &model.TensionRef{ID: &tid}
    var children []model.NodeFragment
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil {
            nodeInput.FirstLink = &model.UserRef{Username: node.FirstLink}
        }
    case model.NodeTypeCircle:
        nodeInput.Children = nil
        for i, c := range(node.Children) {
            child := makeNewChild(i, *c.FirstLink, nameid, *c.RoleType, node.Charac, node.IsPrivate)
            children = append(children, child)
        }
    }

    // Push the nodes into the database
    err := db.GetDB().AddNode(nodeInput)
    if err != nil { return err }

    // Change Guest to member if user got its first role.
    // add tension and child for existing children.
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil {
            err = maybeUpdateGuest2Peer(uctx, rootnameid, *node.FirstLink)
        }
    case model.NodeTypeCircle:
        for _, child := range(children) {
            // Add the child tension
            tensionInput := makeNewCoordoTension(uctx, emitterid, nameid, child)
            tid_c, err := db.GetDB().AddTension(tensionInput)
            if err != nil { return err }
            // Push child
            err = PushNode(uctx, tid_c, &child, emitterid, *child.Nameid, nameid)
        }
    }

    return err
}


// updateNode update a node from the given fragment
// @DEBUG: only set the field that have been modified in NodePatch
func UpdateNode(uctx model.UserCtx, node *model.NodeFragment, emitterid, nameid, parentid string) (error) {
    // Map NodeFragment to Node Patch Input
    var nodePatch model.NodePatch
    delMap := make(map[string]interface{}, 2)
    StructMap(node, &nodePatch)

    // Fix automatic fields
    switch *node.Type {
    case model.NodeTypeRole:
        if node.FirstLink != nil {
            nodePatch.FirstLink = &model.UserRef{Username: node.FirstLink}
        } else {
            delMap["Node.first_link"] = nil
        }
    case model.NodeTypeCircle:
        nodePatch.Children = nil
    }

    // Build input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Set: &nodePatch,
        //Remove: &delNodePatch, // @debug: omitempty issues
    }
    // Update the node in database
    err := db.GetDB().UpdateNode(nodeInput)
    if err != nil { return err }

    if len(delMap) > 0 {
        err = db.GetDB().DeleteEdges("Node.nameid", nameid, delMap)
    }

    return err
}

// internals

func makeNewChild(i int, username string, parentid string, role_type model.RoleType, charac *model.NodeCharac, isPrivate *bool) model.NodeFragment {
    //name := "Coordinator"
    //nameid := "coordo" + strconv.Itoa(i)
    name := string(role_type)
    nameid := parentid +"#"+ name + strconv.Itoa(i)
    type_ := model.NodeTypeRole
    roleType := model.RoleTypeCoordinator
    fs := username
    mandate := model.Mandate{Purpose: en.CoordoPurpose}
    child := model.NodeFragment{
        Name: &name,
        Nameid: &nameid,
        Type: &type_,
        RoleType: &roleType,
        FirstLink: &fs,
        Mandate: &mandate,
        Charac: charac,
        IsPrivate: isPrivate,
    }
    return child
}

func makeNewCoordoTension(uctx model.UserCtx, emitterid string, receiverid string, child model.NodeFragment) model.AddTensionInput {
    now := time.Now().Format(time.RFC3339)
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
        Title: "[Role] Coordinator",
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

// maybeUpdateGuest2Peer check if Guest should be upgrade to Member role type
func maybeUpdateGuest2Peer(uctx model.UserCtx, rootnameid string, username string) error {
    var uctxFs *model.UserCtx
    var err error
    DB := db.GetDB()
    if username != uctx.Username {
        uctxFs, err = DB.GetUser("username", username)
        if err != nil { return err }
    } else {
        uctxFs = &uctx
    }

    i := userIsGuest(*uctxFs, rootnameid)
    if i >= 0 {
        // Update RoleType to Member
        err := DB.UpgradeGuest(uctxFs.Roles[i].Nameid, model.RoleTypeMember)
        if err != nil { return err }
    }

    return nil
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
