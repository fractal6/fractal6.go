package auth

import (
    //"fmt"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/db"
    . "fractale/fractal6.go/tools"
)

// Inherits node properties
func InheritNodeCharacDefault(node *model.NodeFragment, parent *model.Node) {
    if node.Mode == nil {
        node.Mode = &parent.Mode
    }
    if node.Visibility == nil {
        node.Visibility = &parent.Visibility
    }
}

////////////////////////////////////////////////
// Base authorization methods
// @future: GBAC authorization with @auth directive (DGraph)
////////////////////////////////////////////////

//
// Getters
//

func HasCoordoRole(uctx *model.UserCtx, nameid string, mode *model.NodeMode) (bool, error) {
    // Get the node mode eventually
    if mode == nil {
        mode_, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.mode")
        if err != nil { return false, LogErr("Internal error", err) }
        mode = mode_.(*model.NodeMode)
    }

    // Check user rights
    ok, err := CheckUserRights(uctx, nameid, *mode)
    if err != nil { return ok, LogErr("Internal error", err) }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(nameid) {
        ok, err = CheckUpperRights(uctx, nameid, *mode)
    }
    return ok, err
}

func GetCoordosFromTid(tid string) ([]model.User, error) {
    var coordos []model.User

    // Check user rights
    nodes, err := db.GetDB().Meta("getCoordosFromTid", map[string]string{"tid":tid})
    if err != nil { return coordos, LogErr("Internal error", err) }

    // Return direct coordos
    if len(nodes) > 0 {
        for _, c := range nodes {
            var coordo model.User
            if err := Map2Struct(c, &coordo); err != nil {
                return coordos, err
            }
            coordos = append(coordos, coordo)
        }
        return coordos, err
    }

    // Return first met parent coordos
    var parents []string
    node, err := db.GetDB().Meta("getParentFromTid", map[string]string{"tid":tid})
    if err != nil { return coordos, LogErr("Internal Error", err) }
    if len(node) == 0 || node[0]["parent"] == nil {
        return coordos, err
    }
    // @debug: dql decoding !
    if nodes := node[0]["parent"].([]interface{}); len(nodes) > 0 {
        if nids, ok :=  nodes[0].(model.JsonAtom)["nameid"]; ok && nids != nil {
            for _, v := range nids.([]interface{}) {
                parents = append(parents, v.(string))
            }
        }
    }

    for _, nameid := range parents {
        res, err := db.GetDB().Meta("getCoordos2", map[string]string{"nameid": nameid})
        if err != nil { return coordos, LogErr("Internal error", err) }

        // stop at the first circle with coordos
        if len(res) > 0 {
            for _, c := range res {
                var coordo model.User
                if err := Map2Struct(c, &coordo); err != nil {
                    return coordos, err
                }
                coordos = append(coordos, coordo)
            }
            return coordos, err
        }
    }

    return coordos, err
}

//
// Checkers
//

// ChechUserRight return true if the user has access right (e.g. Coordo) on the given node
func CheckUserRights(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
    var ok bool = false
    var err error

    // Get the nearest circle
    if codec.IsRole(nameid) {
        nameid, _ = codec.Nid2pid(nameid)
    }

    // Escape if the user is an owner
    rootnameid, _ := codec.Nid2rootid(nameid)
    if UserIsOwner(uctx, rootnameid) >= 0 { return true, err }

    if mode == model.NodeModeAgile {
        ok = UserHasRole(uctx, nameid) >= 0
    } else if mode == model.NodeModeCoordinated {
        ok = UserIsCoordo(uctx, nameid) >= 0
    }

    return ok, err
}

// chechUpperRight return true if the user has access right (e.g. Coordo) on any on its parents.
func CheckUpperRights(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
    var ok bool = false
    parents, err := db.GetDB().GetParents(nameid)
    if err != nil { return ok, LogErr("Internal Error", err) }

    for _, p := range(parents) {
        ok, err = CheckUserRights(uctx, p, mode)
        if err != nil { return ok, LogErr("Internal error", err) }
        if ok { break }
    }

    return ok, err
}
