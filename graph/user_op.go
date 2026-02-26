/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
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
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/sessions"
)

func LinkUser(rootnameid, nameid, username string) error {
	// Do not remove membership nodes for history
	if codec.MemberIdCodec(rootnameid, username) != nameid {
		err := db.GetDB().AddUserRole(username, nameid)
		if err != nil {
			return err
		}
	}

	err := maybeUpdateMembership(rootnameid, username, model.RoleTypeMember)
	return err
}

func UnlinkUser(rootnameid, nameid, username string) error {
	// Do not remove membership nodes for history
	if codec.MemberIdCodec(rootnameid, username) != nameid {
		err := db.GetDB().RemoveUserRole(username, nameid)
		if err != nil {
			return err
		}
	}

	err := maybeUpdateMembership(rootnameid, username, model.RoleTypeGuest)
	return err
}

func LeaveRole(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment) (bool, error) {
	var err error
	var rootnameid string
	var nameid string
	parentid := tension.Receiver.Nameid

	// Type check
	if node.RoleType == nil {
		return false, LogErr("access denied", fmt.Errorf("Node needs a role type for this action."))
	}

	// Special case for membership role
	IsMembershipRole := codec.IsMembershipRoleType(*node.RoleType)
	if IsMembershipRole {
		nameid = *node.Nameid
		rootnameid, err = codec.Nid2rootid(nameid)
		if err != nil {
			return false, err
		}
		if len(auth.GetRoles(uctx, nameid)) > 1 && *node.RoleType != model.RoleTypeOwner {
			return false, LogErr("access denied", fmt.Errorf("Doh, you have active roles in this organisation. Please leave your roles first."))
		} else if *node.RoleType == model.RoleTypePending {
			return false, LogErr("access denied", fmt.Errorf("Doh, you cannot leave a pending role. Please reject the invitation."))
		} else if *node.RoleType == model.RoleTypeRetired {
			return false, LogErr("access denied", fmt.Errorf("You are already retired from this role."))
		} else if *node.RoleType == model.RoleTypeOwner {
			// Owner can leave if not alone
			// --
			// Get all owners of the organization
			owners := []string{}
			if users, err := db.GetDB().Meta("getOwners", map[string]string{"nameid": rootnameid}); err != nil {
				return false, err
			} else {
				for _, u := range users {
					owners = append(owners, u["username"].(string))
				}
			}
			// If owner is alone, prevent orphan organization
			if len(owners) < 2 {
				return false, LogErr("access denied", fmt.Errorf("An organization needs at least one Owner. Please contact us if you need to transfer ownership."))
			}

			// Downgrade Owner to Member
			if err = db.GetDB().UpgradeMember(nameid, model.RoleTypeMember); err != nil {
				return false, err
			}
		}
	} else {
		// Get References
		rootnameid, nameid, err = codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
		if err != nil {
			return false, err
		}
	}

	// If user doesn't play role, return error
	if i := auth.UserPlaysRole(uctx, nameid); i < 0 {
		return false, LogErr("access denied", fmt.Errorf("Role already left or not played."))
	}

	err = UnlinkUser(rootnameid, nameid, uctx.Username)
	if err != nil {
		return false, err
	}

	// Update NodeFragment
	if node.ID != "" {
		// @debug: should delete instead...DelFieldById => `<x> <x> * .`
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.first_link", "")
	}

	return true, err
}

// maybeUpdateMembership check aitomatically to toggle user membership to Guest or Member if needed
func maybeUpdateMembership(rootnameid string, username string, rt model.RoleType) error {
	var uctxFs *model.UserCtx
	var err error
	uctxFs, err = db.GetDB().GetUctx("username", username)
	if err != nil {
		return err
	}

	// Don't touch owner state here
	if auth.UserIsOwner(uctxFs, rootnameid) >= 0 {
		return nil
	}

	nid := codec.MemberIdCodec(rootnameid, username)
	roles := auth.GetRoles(uctxFs, rootnameid)
	if len(roles) > 2 {
		return nil
	}

	// User Downgrade
	if rt == model.RoleTypeGuest {
		if len(roles) == 1 && *roles[0].RoleType == model.RoleTypeMember {
			// Member is downgraded to Guest
			err = db.GetDB().UpgradeMember(nid, model.RoleTypeGuest)
		} else if len(roles) == 1 && (*roles[0].RoleType == model.RoleTypeGuest || *roles[0].RoleType == model.RoleTypePending) {
			// Member is retiring
			err = db.GetDB().UpgradeMember(nid, model.RoleTypeRetired)
			if err != nil {
				return err
			}

			// User is leaving an organisation: Remove user assignement from tensions in organisation
			_, err = db.GetDB().Meta("removeAssignedTension", map[string]string{"username": username, "rootnameid": rootnameid})
		}
		return err
	}

	// User Upgrade
	if rt == model.RoleTypeMember {
		if len(roles) == 1 {
			// Upgrade to Guest
			err = db.GetDB().UpgradeMember(nid, model.RoleTypeGuest)
		} else if len(roles) == 2 {
			// Upgrade to Member
			err = db.GetDB().UpgradeMember(nid, model.RoleTypeMember)
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
	_, err := db.GetDB().Meta("setPendingUserToken", map[string]string{"email": email, "token": token})
	return err
}

func SyncPendingUser(username, email string) error {
	// Get the linked contract
	contracts, err := db.GetDB().GetSubFieldByEq("PendingUser.email", email, "PendingUser.contracts", "uid Post.createdAt")
	if err != nil {
		return err
	}

	// Build inputs
	var inputs []model.AddUserEventInput
	if contracts != nil {
		for _, c := range contracts.([]any) {
			// Aggregate event inputs
			con := c.(model.JsonAtom)
			cid := con["id"].(string)
			createdAt, ok := con["createdAt"].(string)
			if !ok {
				continue
			} // If a contract gets deletes, the uid only will subsits in the list.
			inputs = append(inputs, model.AddUserEventInput{
				User:      &model.UserRef{Email: &email},
				IsRead:    false,
				CreatedAt: createdAt,
				Event:     []*model.EventKindRef{{ContractRef: &model.ContractRef{ID: &cid}}},
			})

			// Fetch contract
			contract, err := db.GetDB().GetContractHook(cid)
			if err != nil {
				return err
			}

			// Update contract
			// --
			var contractPatch model.ContractPatch
			// Set event type
			contractPatch.Event = StructMap[*model.EventFragmentRef](contract.Event)
			// Set candidate
			contractPatch.Candidates = []*model.UserRef{{Email: &email}}
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
			err = db.GetDB().Update(db.GetDB().GetRootUctx(), "contract", model.UpdateContractInput{
				Filter: &model.ContractFilter{ID: []string{cid}},
				Set:    &contractPatch,
			})
			if err != nil {
				return err
			}
			// @id field cant't be update with graphql (@debug dgraph)
			err = db.GetDB().SetFieldById(cid, "Contract.contractid", contractid)
			if err != nil {
				return err
			}

			// Do MaybeAddPendingNode for each invitation.
			if contract.Event.EventType == model.TensionEventMemberLinked || contract.Event.EventType == model.TensionEventUserJoined {
				// Add pending Nodes
				for _, pc := range contract.PendingCandidates {
					if pc.Email == email {
						_, err = MaybeAddPendingNode(username, &model.Tension{ID: contract.Tension.ID})
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// Push user events
	_, err = db.GetDB().AddMany(db.GetDB().GetRootUctx(), "userEvent", inputs)
	if err != nil {
		return err
	}

	// Remove pending user
	err = db.GetDB().Delete(db.GetDB().GetRootUctx(), "pendingUser", model.PendingUserFilter{
		Email: &model.StringHashFilter{Eq: &email},
	})

	return err
}
