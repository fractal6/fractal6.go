package graph

import (
    "fmt"
    "time"
    "strings"
    "strconv"
    //"github.com/mitchellh/mapstructure"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/text/en"
    "zerogov/fractal6.go/tools"
)

func canAddOrgaNode(uctx model.UserCtx, node *model.NodeFragment, parentid string, charac *model.NodeCharac) (bool, error) {
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

// pushOrgaNode add a new role or circle in an graph.
// * It adds automatic fields such as createdBy, createdAt, etc
// * It automatically add tension associated to potential children.
func pushOrgaNode(uctx model.UserCtx, tid string, node *model.NodeFragment, emitterid string, parentid string, charac *model.NodeCharac, isPrivate bool) (error) {
    // Get References
    now := time.Now().Format(time.RFC3339)
    rootnameid, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)
    if err != nil {
        return err
    }

    var nodeInput model.AddNodeInput
    tools.StructMap(node, &nodeInput)
    var characRef model.NodeCharacRef
    tools.StructMap(charac, &characRef)

    // Set Automatic fields
    nodeInput.IsRoot = false
    nodeInput.IsPrivate = isPrivate
    nodeInput.Charac = &characRef
    nodeInput.Nameid = nameid
    nodeInput.Rootnameid = rootnameid
    nodeInput.CreatedAt = now
    nodeInput.CreatedBy = &model.UserRef{Username: &uctx.Username}
    nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
    nodeInput.Source = &model.TensionRef{ID: &tid}
    nodeInput.Children = nil

    var children []model.NodeFragment
    switch *node.Type {
    case model.NodeTypeRole:
        nodeInput.FirstLink = &model.UserRef{Username: &uctx.Username}
    case model.NodeTypeCircle:
        for i, c := range(node.Children) {
            child := makeNewCoordo(i, *c.FirstLink, charac)
            children = append(children, child)
        }
    }


    // Push the nodes to the Database
    err = db.GetDB().AddNode(nodeInput)
    if err != nil {
        return err
    }

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
            if err != nil {
                return err
            }
            // Push child
            err = pushOrgaNode(uctx, tid_c, &child, emitterid, nameid, charac, isPrivate)
        }
    }

    return err
}

func makeNewCoordo(i int, username string, charac *model.NodeCharac) model.NodeFragment {
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
    blob_type := model.BlobTypeInitBlob
    var childref model.NodeFragmentRef
    tools.StructMap(child, &childref)
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
        if err != nil {
            return err
        }
    } else {
        uctxFs = &uctx
    }

    i := userIsGuest(*uctxFs, rootnameid)
    if i >= 0 {
        // Update RoleType to Member
        err := DB.UpgradeGuest(uctxFs.Roles[i].Nameid, model.RoleTypeMember)
        if err != nil {
            return err
        }
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

//
// User Codecs
//

func nodeIdCodec(parentid string, targetid string,  nodeType model.NodeType) (string, string, error) {
    var nameid string
    rootnameid, err := nid2rootid(parentid)
    if nodeType == model.NodeTypeRole {
        if rootnameid == parentid {
            nameid = strings.Join([]string{rootnameid, "", targetid}, "#")
        } else {
            nameid = strings.Join([]string{parentid, targetid}, "#")
        }
    } else if nodeType == model.NodeTypeCircle {
        nameid = strings.Join([]string{rootnameid, targetid}, "#")
    }
    return rootnameid, nameid, err
}

// Get the parent nameid from the given nameid
func nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else {
        pid = strings.Join(parts[:len(parts)-1],  "#")
    }
    return pid, nil
}

// Get the rootnameid from the given nameid
func nid2rootid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    return parts[0], nil
}

func isCircle(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 1 || len(parts) == 2
}
func isRole(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 3
}
