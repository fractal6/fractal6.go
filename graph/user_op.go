package graph

import (
    "fmt"

    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/graph/auth"
    "fractale/fractal6.go/db"
    //. "fractale/fractal6.go/tools"
)

func LinkUser(rootnameid, nameid, username string) error {
    // Anchor role should already exists
    if codec.MemberIdCodec(rootnameid, username) != nameid {
        err := db.GetDB().AddUserRole(username, nameid)
        if err != nil { return err }
    }

    err := maybeUpdateMembership(rootnameid, username, model.RoleTypeMember)
    return err
}

func UnlinkUser(rootnameid, nameid, username string) error {
    // Keep Retired user for references (tension)
    if codec.MemberIdCodec(rootnameid, username) != nameid {
        err := db.GetDB().RemoveUserRole(username, nameid)
        if err != nil { return err }
    }

    err := maybeUpdateMembership(rootnameid, username, model.RoleTypeGuest)
    return err
}

func LeaveRole(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
    parentid := tension.Receiver.Nameid

    // Type check
    if node.RoleType == nil { return false, fmt.Errorf("Node need a role type for this action.") }

    // Get References
    rootnameid, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)

    switch *node.RoleType {
    case model.RoleTypeOwner:
        return false, fmt.Errorf("Doh, organisation destruction is not yet implemented.")
    case model.RoleTypeMember:
        return false, fmt.Errorf("Doh, you ave active role in this organisation. Please leave your roles first.")
    case model.RoleTypePending:
        return false, fmt.Errorf("Doh, you cannot leave a pending role. Please reject the invitation.")
    case model.RoleTypeRetired:
        return false, fmt.Errorf("You are already retired from this role.")
    default: // Guest Peer, Coordinator + user defined roles
        err = UnlinkUser(rootnameid, nameid, uctx.Username)
        if err != nil {return false, err}
        // @obsolete: Remove user from last blob if present
        //err = db.GetDB().MaybeDeleteFirstLink(tension.ID, uctx.Username)
    }

    return true, err
}

// maybeUpdateMembership check try to toggle user membership to Guest or Member
func maybeUpdateMembership(rootnameid string, username string, rt model.RoleType) error {
    var uctxFs *model.UserCtx
    var err error
    DB := db.GetDB()
    uctxFs, err = DB.GetUctxFull("username", username)
    if err != nil { return err }

    // Don't touch owner state
    if auth.UserIsOwner(uctxFs, rootnameid) >= 0 {
        return nil
    }

    nid := codec.MemberIdCodec(rootnameid, username)
    roles := auth.GetRoles(uctxFs, rootnameid)

    if len(roles) > 2 { return nil }

    // User Downgrade
    if rt == model.RoleTypeGuest {
        if len(roles) == 1 && *roles[0].RoleType == model.RoleTypeMember {
            err = db.GetDB().UpgradeMember(nid, model.RoleTypeGuest)
        } else if len(roles) == 1 && *roles[0].RoleType == model.RoleTypeGuest {
            err = DB.UpgradeMember(nid, model.RoleTypeRetired)
        }
        return err
    }

    // User Upgrade
    if rt == model.RoleTypeMember {
        if len(roles) == 1 {
            err = DB.UpgradeMember(nid, model.RoleTypeGuest)
        } else if len(roles) == 2 {
            err = DB.UpgradeMember(nid, model.RoleTypeMember)
        }
        return err
    }

    return fmt.Errorf("role upgrade not implemented: %s", rt)
}

