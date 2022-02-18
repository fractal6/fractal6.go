package auth

import (
    //"fmt"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/db"
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
        if rid, err := codec.Nid2rootid(r.Nameid); err == nil && rid == rootnameid {
            roles = append(roles, r)
        } else if err != nil {
            panic(err.Error())
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
        if *r.RoleType == model.RoleTypeRetired || *r.RoleType == model.RoleTypePending {
            continue
        }

        if rid, err := codec.Nid2rootid(r.Nameid); err == nil && rid == rootnameid {
            return i
        } else if err != nil {
            panic(err.Error())
        }
    }
    return -1
}

// UserIsGuest return true if the user is a guest (has only one role) in the given organisation
func UserIsGuest(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if *r.RoleType != model.RoleTypeGuest {
            continue
        }

        if rid, err := codec.Nid2rootid(r.Nameid); err == nil && rid == rootnameid {
            return i
        } else if err != nil {
            panic(err.Error())
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
        if *r.RoleType == model.RoleTypeGuest || *r.RoleType == model.RoleTypePending || *r.RoleType == model.RoleTypeRetired {
            continue
        }

        if pid, err := codec.Nid2pid(r.Nameid); err == nil && pid == nameid {
            return i
        } else if err != nil {
            panic(err.Error())
        }
    }
    return -1
}

// useIsCoordo return true if the user has at least one role of Coordinator in the given node
func UserIsCoordo(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if *r.RoleType != model.RoleTypeCoordinator && *r.RoleType != model.RoleTypeOwner {
            continue
        }

        if pid, err := codec.Nid2pid(r.Nameid); err == nil && pid == nameid {
            return i
        } else if err != nil {
            panic(err.Error())
        }
    }

    return -1
}

// IsCoordo returns true if a user has at least one role with right coordinator in a organisation
func IsCoordo(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    rootnameid, e := codec.Nid2rootid(nameid)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if *r.RoleType != model.RoleTypeCoordinator && *r.RoleType != model.RoleTypeOwner {
            continue
        }

        if rid, err := codec.Nid2rootid(r.Nameid); err == nil && rid == rootnameid {
            return i
        } else if err != nil {
            panic(err.Error())
        }
    }

    return -1
}

func UserIsOwner(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.MaybeRefresh(uctx)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if *r.RoleType != model.RoleTypeOwner {
            continue
        }

        if rid, err := codec.Nid2rootid(r.Nameid); err == nil && rid == rootnameid {
            return i
        } else if err != nil {
            panic(err.Error())
        }
    }

    return -1
}

//
// Wrapper when uctx is unknown
//

func IsMember(username, rootnameid string) int {
    uctx, e := db.GetDB().GetUctxFull("username", username)
    if e != nil { panic(e) }
    return UserIsMember(uctx, rootnameid)

}
