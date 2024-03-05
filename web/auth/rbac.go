/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
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

package auth

import (
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
)

/*
 *
 * Authorization function based on the UserCtx struct
 * got from the user token.
 *
 */

// GetRoles returns a list of the users roles inside an organisation
func GetRoles(uctx *model.UserCtx, nameid string) []*model.Node {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}
	rootnameid, e := codec.Nid2rootid(nameid)
	if e != nil {
		panic(e)
	}

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

// UserPlaysRole return true if the user play the given role (Nameid)
func UserPlaysRole(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}

	if !codec.IsRole(nameid) {
		panic("nameid is ambigous. Only role's nameid are supported.")
	}

	for i, r := range uctx.Roles {
		if r.Nameid == nameid {
			return i
		}
	}

	return -1
}

// UserIsOwner returns true if user is Owner in the given organisation
func UserIsOwner(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}
	rootnameid, e := codec.Nid2rootid(nameid)
	if e != nil {
		panic(e)
	}

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

// UserIsMember return true if the user belongs to an organisation.
func UserIsMember(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}
	rootnameid, e := codec.Nid2rootid(nameid)
	if e != nil {
		panic(e)
	}

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

// UserIsGuest return true if the user is a guest in an organisation.
func UserIsGuest(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}
	rootnameid, e := codec.Nid2rootid(nameid)
	if e != nil {
		panic(e)
	}

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

// UserHasRole return true if the user has at least one role other than guest
// in the given circle.
func UserHasRole(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}

	if codec.IsRole(nameid) {
		panic("nameid is ambigous. Only circle's nameid are supported.")
	}

	for i, r := range uctx.Roles {
		if *r.RoleType == model.RoleTypeMember ||
			*r.RoleType == model.RoleTypeGuest ||
			*r.RoleType == model.RoleTypePending ||
			*r.RoleType == model.RoleTypeRetired {
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

// UserHasCoordoRole return true if the user has at least one Coordinator role
// in the given circle.
func UserHasCoordoRole(uctx *model.UserCtx, nameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}

	if codec.IsRole(nameid) {
		panic("nameid is ambigous. Only circle's nameid are supported.")
	}

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

// HasCoordo returns true if he user has at least one coordinator role
// in the given organisation.
func HasCoordo(uctx *model.UserCtx, rootnameid string) int {
	uctx, e := MaybeRefresh(uctx)
	if e != nil {
		panic(e)
	}
	rootnameid, e = codec.Nid2rootid(rootnameid)
	if e != nil {
		panic(e)
	}

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

//
// Wrapper when uctx is unknown
//

// IsMember check is user is a member of an organisation from
// its email or username {fieldname}.
func IsMember(fieldname, username, nameid string) int {
	uctx, e := db.GetDB().GetUctx(fieldname, username)
	// We do not panic here, because user, maybe just does no exits.
	// Which is the case when a new user is invented !
	if e != nil {
		return -1
	}
	return UserIsMember(uctx, nameid)
}
