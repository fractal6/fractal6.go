package graph

import (
    "fmt"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    //"zerogov/fractal6.go/text/en"
    //. "zerogov/fractal6.go/tools"
)

func LinkUser(rootnameid, nameid, firstLink string) error {
    err := auth.AddUserRole(firstLink, nameid)
    if err != nil { return err }

    err = maybeUpdateMembership(rootnameid, firstLink, model.RoleTypeMember)
    if err != nil { return  err }

    return err
}

func UnlinkUser(rootnameid, nameid, firstLink string) error {
    err := auth.RemoveUserRole(firstLink, nameid)
    if err != nil { return err }

    err = maybeUpdateMembership(rootnameid, firstLink, model.RoleTypeGuest)
    if err != nil { return err }

    return err
}

func LeaveRole(uctx model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    tid := tension.ID
    parentid := tension.Receiver.Nameid

    // CanLeaveRole
    // --
    if node.RoleType == nil { return false, fmt.Errorf("Node need a role type for this action.") }
    if node.FirstLink == nil { return false, fmt.Errorf("Node need a linked user for this action.") }

    // Check the identity of the user asking
    if *node.FirstLink != uctx.Username {
        return false, fmt.Errorf("Access denied")
    }
    // --

    // Get References
    rootnameid, nameid, err := nodeIdCodec(parentid, *node.Nameid, *node.Type)

    switch *node.RoleType {
    case model.RoleTypeOwner:
        return false, fmt.Errorf("Doh, organisation destruction is not yet implemented, WIP.")
    case model.RoleTypeMember:
        return false, fmt.Errorf("Doh, you avec active role in this organisation.")
    case model.RoleTypeGuest:
        err = UnlinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil {return false, err}
        err = db.GetDB().Delete("node", model.NodeFilter{ Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}})
        if err != nil {return false, err}
    default: // Peer, Coordinator
        err = UnlinkUser(rootnameid, nameid, *node.FirstLink)
        if err != nil {return false, err}
        // @Debug: Remove user from last blob if present
        err = db.GetDB().MaybeDeleteFirstLink(tid, uctx.Username)
    }

    return true, err
}
