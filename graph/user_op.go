/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package graph

import (
	"fmt"
    "strings"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/auth"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/sessions"
	. "fractale/fractal6.go/tools"
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

    // If user doesn't play role, return error
    if i := auth.UserPlaysRole(uctx, nameid); i < 0 {
        return false, fmt.Errorf("Role already leaved or not played.")
    }

    switch *node.RoleType {
    case model.RoleTypeOwner:
        return false, fmt.Errorf("Doh, organisation destruction is not yet implemented.")
    case model.RoleTypeMember:
        return false, fmt.Errorf("Doh, you have active role in this organisation. Please leave your roles first.")
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

    // Update NodeFragment
    // @debug: should delete instead...
    if node.ID != "" {
        err = db.GetDB().SetFieldById(node.ID, "NodeFragment.first_link", "")
    }

    return true, err
}

// maybeUpdateMembership check try to toggle user membership to Guest or Member
func maybeUpdateMembership(rootnameid string, username string, rt model.RoleType) error {
    var uctxFs *model.UserCtx
    var err error
    DB := db.GetDB()
    uctxFs, err = DB.GetUctx("username", username)
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
        } else if len(roles) == 1 && (*roles[0].RoleType == model.RoleTypeGuest || *roles[0].RoleType == model.RoleTypePending) {
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

    // @TODO: The uctx cache (.Roles) maybe out of date here for few seconds...

    return fmt.Errorf("role upgrade not implemented: %s", rt)
}

// Pending user operations

func MaybeSetPendingUserToken(email string) error {
    // Add a verification token if not exists
    // (assumes PendingUser has already been created)
    token := sessions.GenerateToken()
    _, err := db.GetDB().Meta("setPendingUserToken", map[string]string{"email":email, "token":token})
    return err
}

func SyncPendingUser(username, email string) error {
    // Get the linked contract
    contracts, err := db.GetDB().GetSubFieldByEq("PendingUser.email", email, "PendingUser.contracts", "uid Post.createdAt")
    if err != nil { return err }

    // Build inputs
    var inputs []model.AddUserEventInput
    if contracts != nil {
        for _, c := range contracts.([]interface{}) {
            // Aggregate event inputs
            con := c.(model.JsonAtom)
            cid := con["id"].(string)
            createdAt, ok := con["createdAt"].(string)
            if !ok { continue } // If a contract gets deletes, the uid only will subsits in the list.
            inputs = append(inputs, model.AddUserEventInput{
                User: &model.UserRef{Email: &email},
                IsRead: false,
                CreatedAt: createdAt,
                Event: []*model.EventKindRef{&model.EventKindRef{ContractRef: &model.ContractRef{ID: &cid}}},
            })

            // Fetch contract
            contract, err := db.GetDB().GetContractHook(cid)
            if err != nil { return err }

            // Update contract
            // --
            var contractPatch model.ContractPatch
            // Set event type
            StructMap(contract.Event, &contractPatch.Event)
            // Set candidate
            contractPatch.Candidates = []*model.UserRef{&model.UserRef{Email: &email}}
            emailPart := strings.Split(email, "@")[0]
            if contract.Event.Old != nil && strings.HasPrefix(*contract.Event.Old, emailPart) {
                contractPatch.Event.Old = &username
            }
            if contract.Event.New != nil && strings.HasPrefix(*contract.Event.New, emailPart) {
                contractPatch.Event.New = &username
            }
            contractid := codec.ContractIdCodec(
                contract.Tension.ID,
                *contractPatch.Event.EventType,
                *contractPatch.Event.Old,
                *contractPatch.Event.New,
            )
            contractPatch.Contractid = &contractid
            err = db.DB.Update(db.DB.GetRootUctx(), "contract", model.UpdateContractInput{
                Filter: &model.ContractFilter{ID: []string{cid}},
                Set: &contractPatch,
            })
            if err != nil { return err }

            // Do MaybeAddPendingNode for each invitation.
            if contract.Event.EventType == model.TensionEventMemberLinked || contract.Event.EventType == model.TensionEventUserJoined {
                // Add pending Nodes
                for _, pc := range contract.PendingCandidates {
                    if *pc.Email == email {
                        _, err = MaybeAddPendingNode(username, &model.Tension{ID: contract.Tension.ID})
                        if err != nil { return err }
                    }
                }
            }
        }
    }

    // Push user events
    _, err = db.GetDB().AddMany(db.GetDB().GetRootUctx(), "userEvent", inputs)
    if err != nil { return err }

    // Remove pending user
    err = db.DB.Delete(db.DB.GetRootUctx(), "pendingUser", model.PendingUserFilter{
        Email: &model.StringHashFilter{Eq: &email},
    })

    return err
}
