package graph

import (
    "fmt"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
    webauth "zerogov/fractal6.go/web/auth"
    //"zerogov/fractal6.go/text/en"
    //. "zerogov/fractal6.go/tools"
)

func LinkUser(rootnameid, nameid, firstLink string) error {
    err := webauth.AddUserRole(firstLink, nameid)
    if err != nil { return err }

    err = maybeUpdateMembership(rootnameid, firstLink, model.RoleTypeMember)
    if err != nil { return  err }

    return err
}

func UnlinkUser(rootnameid, nameid, firstLink string) error {
    err := webauth.RemoveUserRole(firstLink, nameid)
    if err != nil { return err }

    err = maybeUpdateMembership(rootnameid, firstLink, model.RoleTypeGuest)
    if err != nil { return err }

    return err
}

func LeaveRole(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    tid := tension.ID
    parentid := tension.Receiver.Nameid

    // Type check
    if node.RoleType == nil { return false, fmt.Errorf("Node need a role type for this action.") }
    if node.FirstLink == nil { return false, fmt.Errorf("Node need a linked user for this action.") }

    // CanLeaveRole
    if *node.FirstLink != uctx.Username {
        return false, fmt.Errorf("Access denied")
    }

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)

    switch *node.RoleType {
    case model.RoleTypeOwner:
        return false, fmt.Errorf("Doh, organisation destruction is not yet implemented, WIP.")
    case model.RoleTypeMember:
        return false, fmt.Errorf("Doh, you ave active role in this organisation. Please leave your roles first.")
    case model.RoleTypePending:
        return false, fmt.Errorf("Doh, you cannot leave peding role. Please reject the invitation.")
    case model.RoleTypeGuest:
        err := db.GetDB().UpgradeMember(nameid, model.RoleTypeRetired)
        if err != nil {return false, err}
    default: // Peer, Coordinator + user defined roles
        err = UnlinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil {return false, err}
        // @Debug: Remove user from last blob if present
        err = db.GetDB().MaybeDeleteFirstLink(tid, uctx.Username)
    }

    return true, err
}

// maybeUpdateMembership check try to toggle user membership to Guest or Member
func maybeUpdateMembership(rootnameid string, username string, rt model.RoleType) error {
    var uctxFs *model.UserCtx
    var err error
    var i int
    DB := db.GetDB()
    uctxFs, err = DB.GetUser("username", username)
    if err != nil { return err }

    // Don't touch owner state
    if auth.UserIsOwner(uctxFs, rootnameid) >= 0 { return nil }

    // Update RoleType to Guest
    roles := auth.GetRoles(uctxFs, rootnameid)
    if rt == model.RoleTypeGuest && len(roles) == 1  {
        err := DB.UpgradeMember(roles[0].Nameid, model.RoleTypeGuest)
        if err != nil { return err }
        return nil
    }

    // Update RoleType to Member
    i = auth.UserIsGuest(uctxFs, rootnameid)
    if rt == model.RoleTypeMember && i >= 0 {
        // Update RoleType to Member
        err := DB.UpgradeMember(uctxFs.Roles[i].Nameid, model.RoleTypeMember)
        if err != nil { return err }
        return nil
    }

    return nil
}

