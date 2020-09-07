package graph

import (
    "fmt"
    "time"
    "strconv"
    //"github.com/mitchellh/mapstructure"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/text/en"
    . "zerogov/fractal6.go/tools"
)

// tryAddNode add a new node is user has the correct right
// * it inherits node charac
func tryAddNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
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

    ok, err := canAddNode(uctx, node, parentid, charac)
    if err != nil || !ok {
        return ok, err
    }

    err = pushNode(uctx, tid, node, emitterid, parentid)
    return ok, err
}

func tryUpdateNode(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    tid := tension.ID
    emitterid := tension.Emitter.Nameid
    parentid := tension.Receiver.Nameid

    charac := tension.Receiver.Charac

    ok, err := canAddNode(uctx, node, parentid, charac)
    if err != nil || !ok {
        return ok, err
    }

    err = updateNode(uctx, tid, node, emitterid, parentid)
    return ok, err
}

// canAddNode check that a user can add a given role or circle in an organisation.
func canAddNode(uctx model.UserCtx, node *model.NodeFragment, parentid string, charac *model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    nameid := *node.Nameid // @TODO (nameid @codec): verify that nameid match parentid
    name := *node.Name
    nodeType := *node.Type
    roleType := node.RoleType

    rootnameid, _ := nid2rootid(nameid)
    err = auth.ValidateNameid(nameid, rootnameid, name)
    if err != nil {
        return false, err
    }

    //
    // New Role hook
    //
    if nodeType == model.NodeTypeRole {
        if roleType == nil {
            err = fmt.Errorf("role should have a RoleType")
        }

        // Add node Policies
        if charac.Mode == model.NodeModeChaos {
            ok = userIsMember(uctx, parentid)
        } else if charac.Mode == model.NodeModeCoordinated {
            ok = userIsCoordo(uctx, parentid)
        }
        return ok, err
    }

    //
    // New sub-circle hook
    //
    if nodeType == model.NodeTypeCircle {
        // Add node Policies
        if charac.Mode == model.NodeModeChaos {
            ok = userIsMember(uctx, parentid)
        } else if charac.Mode == model.NodeModeCoordinated {
            ok = userIsCoordo(uctx, parentid)
        }
        return ok, err
    }

    return false, fmt.Errorf("unknown node type")
}

// pushNode add a new role or circle in an graph.
// * It adds automatic fields such as createdBy, createdAt, etc
// * It automatically add tension associated to potential children.
func pushNode(uctx model.UserCtx, tid string, node *model.NodeFragment, emitterid string, parentid string) (error) {
    // Get References
    rootnameid, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil {
        return err
    }

    // Map NodeFragment to Node Input
    var nodeInput model.AddNodeInput
    StructMap(node, &nodeInput)

    // Fix Automatic fields
    nodeInput.IsRoot = false
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.CreatedAt = time.Now().Format(time.RFC3339)
    nodeInput.CreatedBy = &model.UserRef{Username: &uctx.Username}
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
    nodeInput.Source = &model.TensionRef{ID: &tid}
    var children []model.NodeFragment
    switch *node.Type {
    case model.NodeTypeRole:
        nodeInput.FirstLink = &model.UserRef{Username: &uctx.Username}
    case model.NodeTypeCircle:
        nodeInput.Children = nil
        for i, c := range(node.Children) {
            child := makeNewCoordo(i, *c.FirstLink, node.Charac, node.IsPrivate)
            children = append(children, child)
        }
    }

    // Push the nodes into the database
    err = db.GetDB().AddNode(nodeInput)
    if err != nil { return err }

    // Change Guest to member if user got its first role.
    // add tension and child for existing children.
    switch *node.Type {
    case model.NodeTypeRole:
        err = maybeUpdateGuest2Peer(uctx, rootnameid, *node.FirstLink)
    case model.NodeTypeCircle:
        for _, child := range(children) {
            // Add the child tension
            tensionInput := makeNewCoordoTension(uctx, emitterid, nameid, child)
            tid_c, err := db.GetDB().AddTension(tensionInput)
            if err != nil { return err }
            // Push child
            err = pushNode(uctx, tid_c, &child, emitterid, nameid)
        }
    }

    return err
}


// updateNode update a node from the given fragment
// @DEBUG: only set the field that have been modified in NodePatch
func updateNode(uctx model.UserCtx, tid string, node *model.NodeFragment, emitterid string, parentid string) (error) {
    // Get References
    _, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil { return err }

    // Map NodeFragment to Node Patch Input
    var nodePatch model.NodePatch
    StructMap(node, &nodePatch)

    // Fix automatic fields
    switch *node.Type {
    case model.NodeTypeRole:
        nodePatch.FirstLink = &model.UserRef{Username: &uctx.Username}
    case model.NodeTypeCircle:
        nodePatch.Children = nil
    }

    // Build input
    nodeInput := model.UpdateNodeInput{
        Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
        Set: &nodePatch,
    }

    // Update the node in database
    err = db.GetDB().UpdateNode(nodeInput)
    return err
}

func makeNewCoordo(i int, username string, charac *model.NodeCharac, isPrivate *bool) model.NodeFragment {
    name := "Coordinator"
    nameid := "coordo" + strconv.Itoa(i)
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
    action := model.TensionActionNewRole
    evt1 := model.TensionEventCreated
    evt2 := model.TensionEventBlobCreated
    evt3 := model.TensionEventBlobPushed
    blob_type := model.BlobTypeOnNode
    var childref model.NodeFragmentRef
    StructMap(child, &childref)
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
