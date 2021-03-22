package auth

import (
    //"fmt"
    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    webauth "zerogov/fractal6.go/web/auth"
)

// GetRoles returns the list of the users roles below the given node
func GetRoles(uctx *model.UserCtx, rootnameid string) []*model.Node {
    uctx, e := webauth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil { panic(e) }

    var roles []*model.Node
    for _, r := range uctx.Roles {
        if r.Rootnameid == rootnameid  {
            roles = append(roles, r)
        }
    }

    return roles
}

// usePlayRole return true if the user play the given role (Nameid)
func UserPlayRole(uctx *model.UserCtx, nameid string) bool {
    uctx, e := webauth.CheckUserCtxIat(uctx, nameid)
    if e != nil { panic(e) }

    for _, ur := range uctx.Roles {
        if ur.Nameid == nameid  {
            return true
        }
    }
    return false
}

// useHasRoot return true if the user belongs to the given root
func UserHasRoot(uctx *model.UserCtx, rootnameid string) bool {
    uctx, e := webauth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil { panic(e) }

    for _, ur := range uctx.Roles {
        if ur.Rootnameid == rootnameid {
            return true
        }
    }
    return false
}

// UserIsGuest return true if the user is a guest (has only one role) in the given organisation
func UserIsGuest(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if r.Rootnameid == rootnameid && *r.RoleType == model.RoleTypeGuest {
            return i
        }
    }

    return -1
}

// useIsMember return true if the user has at least one role in the given node
func UserIsMember(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.CheckUserCtxIat(uctx, nameid)
    if e != nil { panic(e) }

    for i, ur := range uctx.Roles {
        pid, err := codec.Nid2pid(ur.Nameid)
        if err != nil {
            panic(err.Error())
        }
        if pid == nameid {
            return i
        }
    }
    return -1
}

// useIsCoordo return true if the user has at least one role of Coordinator in the given node
func UserIsCoordo(uctx *model.UserCtx, nameid string) int {
    uctx, e := webauth.CheckUserCtxIat(uctx, nameid)
    if e != nil { panic(e) }

    for i, ur := range uctx.Roles {
        pid, err := codec.Nid2pid(ur.Nameid)
        if err != nil {
            panic("bad nameid format for coordo test: "+ ur.Nameid)
        }
        if pid == nameid && *ur.RoleType == model.RoleTypeCoordinator {
            return i
        }
    }

    return -1
}

func UserIsOwner(uctx *model.UserCtx, rootnameid string) int {
    uctx, e := webauth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil { panic(e) }

    for i, r := range uctx.Roles {
        if r.Rootnameid == rootnameid && *r.RoleType == model.RoleTypeOwner {
            return i
        }
    }

    return -1
}

