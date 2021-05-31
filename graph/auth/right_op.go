package auth

import (
    //"fmt"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/db"
    . "zerogov/fractal6.go/tools"
)

////////////////////////////////////////////////
// Base authorization methods
////////////////////////////////////////////////

func HasCoordoRole(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    // Check user rights
    ok, err := CheckUserRights(uctx, nameid, charac)
    if err != nil { return ok, LogErr("Internal error", err) }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(nameid) {
        ok, err = CheckUpperRights(uctx, nameid, charac)
    }
    return ok, err
}

// ChechUserRight return true if the user has access right (e.g. Coordo) on the given node
func CheckUserRights(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    // Get the nearest circle
    if codec.IsRole(nameid) {
        nameid, _ = codec.Nid2pid(nameid)
    }

    // Escape if the user is an owner
    rootnameid, _ := codec.Nid2rootid(nameid)
    if UserIsOwner(uctx, rootnameid) >= 0 { return true, err }

    // Get the mode of the node
    if charac == nil {
        charac, err = db.GetDB().GetNodeCharac("nameid", nameid)
        if err != nil { return ok, LogErr("Internal error", err) }
    }

    if charac.Mode == model.NodeModeAgile {
        ok = UserHasRole(uctx, nameid) >= 0
    } else if charac.Mode == model.NodeModeCoordinated {
        ok = UserIsCoordo(uctx, nameid) >= 0
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
