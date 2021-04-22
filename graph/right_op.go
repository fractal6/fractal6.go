package graph

import (
    "fmt"
    "context"

    . "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
)

func GetNodeCharacStrict() model.NodeCharac {
    return model.NodeCharac{UserCanJoin: false, Mode: model.NodeModeCoordinated}
}

// chechUserRight return true if the user has access right (e.g. Coordo) on the given node
func CheckUserRights(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    // Get the nearest circle
    if codec.IsRole(nameid) {
        nameid, _ = codec.Nid2pid(nameid)
    }

    // Escape if the user is an owner
    rootnameid, _ := codec.Nid2rootid(nameid)
    if auth.UserIsOwner(uctx, rootnameid) >= 0 { return true, err }

    // Get the mode of the node
    if charac == nil {
        charac, err = db.GetDB().GetNodeCharac("nameid", nameid)
        if err != nil { return ok, LogErr("Internal error", err) }
    }

    if charac.Mode == model.NodeModeAgile {
        ok = auth.UserIsMember(uctx, nameid) >= 0
    } else if charac.Mode == model.NodeModeCoordinated {
        ok = auth.UserIsCoordo(uctx, nameid) >= 0
    }

    return ok, err
}

// chechUpperRight return true if the user has access right (e.g. Coordo) on any on its parents.
func CheckUpperRights(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    var ok bool
    parents, err := db.GetDB().GetParents(nameid)
    if err != nil { return ok, LogErr("Internal Error", err) }
    if len(parents) == 0 { return ok, err }

    for _, p := range(parents) {
        ok, err = CheckUserRights(uctx, p, charac)
        if err != nil { return ok, LogErr("Internal error", err) }
        if ok { break }
    }

    return ok, err
}

// Check if an user owns the given object
func CheckUserOwnership(ctx context.Context, uctx *model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    var username string
    var err error
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        // Tension here
        id := ctx.Value("id").(string)
        if id == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        // Request the database to get the field
        // @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface: ToTypeName(reflect.TypeOf(nodeObj).String())
        username_, err := db.GetDB().GetSubFieldById(id, "Post."+userField, "User.username")
        if err != nil { return false, err }
        username = username_.(string)
    } else {
        // User here
        username = user.(model.JsonAtom)["username"].(string)
    }

    // Check user ID match
    return uctx.Username == username, err
}

// check if the an user has the given role of the given (nested) node
func CheckAssignees(ctx context.Context, uctx *model.UserCtx, nodeObj interface{}) (bool, error) {
    // Check that nodes are present
    var assignees []interface{}
    var err error
    id_ := ctx.Value("id")
    nameid_ := ctx.Value("nameid")
    if id_ != nil {
        // Tension here
        res, err := db.GetDB().GetSubFieldById(id_.(string), "Tension.assignees", "User.username")
        if err != nil { return false, err }
        if res != nil { assignees = res.([]interface{}) }
    } else if (nameid_ != nil) {
        // Node Here
        res, err := db.GetDB().GetSubSubFieldByEq("Node.nameid", nameid_.(string), "Node.source", "Tension.assignees", "User.username")
        if err != nil { return false, err }
        if res != nil { assignees = res.([]interface{}) }
    } else {
        return false, fmt.Errorf("node target unknown, need a database request here...")
    }

    // Search for assignees
    for _, a := range(assignees) {
        if a.(string) == uctx.Username {
            return true, err
        }
    }
    return false, err
}

