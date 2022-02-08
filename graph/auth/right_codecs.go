package auth

import (
    //"fmt"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    webauth "fractale/fractal6.go/web/auth"
)


/* Authorization function based on the UserCtx struct
 * got from the user token.
 */


// GetRoles returns the list of the users roles below the given node
func GetRoles(uctx *model.UserCtx, rootnameid string) []*model.Node {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    var roles []*model.Node
    for _, r := range uctx.Roles {
        rid, err := codec.Nid2rootid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if rid == rootnameid  {
            roles = append(roles, r)
        }
    }

    return roles
}

// userPlayRole return true if the user play the given role (Nameid)
func UserPlayRole(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if r.Nameid == nameid  {
            return i
        }
    }
    return -1
}

// UserIsMember return true if the user belongs to the given root
func UserIsMember(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        rid, err := codec.Nid2rootid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if rid == rootnameid {
            return i
        }
    }
    return -1
}

// UserIsGuest return true if the user is a guest (has only one role) in the given organisation
func UserIsGuest(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        rid, err := codec.Nid2rootid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if rid == rootnameid && *r.RoleType == model.RoleTypeGuest {
            return i
        }
    }

    return -1
}

// UserHasRole return true if the user has at least one role in the given node
// other than a Guest role.
func UserHasRole(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        pid, err := codec.Nid2pid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if pid == nameid && *r.RoleType != model.RoleTypeGuest {
            return i
        }
    }
    return -1
}

// useIsCoordo return true if the user has at least one role of Coordinator in the given node
func UserIsCoordo(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        pid, err := codec.Nid2pid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if pid == nameid && *r.RoleType == model.RoleTypeCoordinator {
            return i
        }
    }

    return -1
}

func UserIsOwner(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        rid, err := codec.Nid2rootid(r.Nameid)
        if err != nil { panic(err.Error()) }

        if rid == rootnameid && *r.RoleType == model.RoleTypeOwner {
            return i
        }
    }

    return -1
}

