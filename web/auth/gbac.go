/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
	"fmt"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

/*
 *
 * Base authorization methods
 * @future: GBAC authorization with @auth directive (Dgraph)
 *
 */

var UserSelection string = "User.username User.email User.name User.notifyByEmail"

// Inherits node properties
func InheritNodeCharacDefault(node *model.NodeFragment, parent *model.Node) {
	if node.Mode == nil {
		node.Mode = &parent.Mode
	}
	if node.Visibility == nil {
		node.Visibility = &parent.Visibility
	}
}

// Check that user satisfies strict condition (coordo roles on the given nodes)
// @DEBUG: add a schema like validation for mandatory field when passing a list of NodeRef?
// Mandatory field
// - nameid
func CheckNodesAuth(uctx *model.UserCtx, d interface{}, passAll bool) error {
	var ok bool
	var err error

	nodes := []model.NodeRef{}
	// Extract NodeRef from input data.
	switch v := d.(type) {
	case []model.NodeRef:
		nodes = append(nodes, v...)
	case []*model.NodeRef:
		for _, n := range v {
			nodes = append(nodes, *n)
		}
	case []string:
		for _, n := range v {
			nodes = append(nodes, model.NodeRef{Nameid: &n})
		}
	}

	// Check @auth
	// @optimize
	mode := model.NodeModeCoordinated
	for _, n := range nodes {
		if n.Nameid == nil {
			return LogErr("Access denied", fmt.Errorf("nameid in required in artefact nodes fields."))
		}

		ok, err = HasCoordoAuth(uctx, *n.Nameid, &mode)
		if err != nil {
			return LogErr("Internal error", err)
		}

		if passAll && !ok {
			break
		} else if ok {
			break
		}
	}

	if len(nodes) == 0 {
		ok = true
	} else if !ok {
		err = LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
	}
	return err
}

// HasCoordoAuth tells if the user has authority in the given node.
func HasCoordoAuth(uctx *model.UserCtx, nameid string, mode *model.NodeMode) (bool, error) {
	// Get the node mode eventually
	if mode == nil {
		mode_, err := db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.mode")
		if err != nil {
			return false, LogErr("Internal error", err)
		}
		m := model.NodeMode(mode_.(string))
		mode = &m
	}

	// Check user rights
	ok, err := CheckUserAuth(uctx, nameid, *mode)
	if err != nil {
		return ok, LogErr("Internal error", err)
	}

	// If the node has no Coordo roles,
	// check auhority in parent circles.
	if !ok && !db.GetDB().HasCoordos(nameid) {
		ok, err = CheckUpperAuth(uctx, nameid, *mode)
	}
	return ok, err
}

//
// Checkers
//

// CheckUserAuth return true if the user has authority in the given circle.
func CheckUserAuth(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
	var ok bool = false
	var err error

	// Get the nearest circle
	nid, err := codec.Nid2pid(nameid)
	if err != nil {
		return ok, err
	}

	// Escape if the user is an owner
	if UserIsOwner(uctx, nid) >= 0 {
		return true, err
	}

	if mode == model.NodeModeAgile {
		ok = UserHasRole(uctx, nid) >= 0
	} else if mode == model.NodeModeCoordinated {
		ok = UserHasCoordoRole(uctx, nid) >= 0
	}

	return ok, err
}

// CheckUpperRights return true if the user has authority on any on the parents of the given circle.
func CheckUpperAuth(uctx *model.UserCtx, nameid string, mode model.NodeMode) (bool, error) {
	var ok bool = false
	parents, err := db.GetDB().GetParents(nameid)
	if err != nil {
		return ok, LogErr("Internal Error", err)
	}

	for _, p := range parents {
		ok, err = CheckUserAuth(uctx, p, mode)
		if err != nil {
			return ok, LogErr("Internal error", err)
		}
		if ok {
			break
		} else if db.GetDB().HasCoordos(p) {
			// Intermediate coordo prevent upper coordo
			// to take authority.
			return false, err
		}
	}

	return ok, err
}

//
// Getters
//

// @REFACTOR: this is an DQL impementation of HasCoordoAuth
func GetCoordosFromTid(tid string) ([]model.User, error) {
	var coordos []model.User

	// Fetch Coordo users in receiver circle.
	nodes, err := db.GetDB().Meta("getCoordosFromTid", map[string]string{"tid": tid, "user_payload": UserSelection})
	if err != nil {
		return coordos, LogErr("Internal error", err)
	}

	// Return direct coordos if present
	if len(nodes) > 0 {
		for _, c := range nodes {
			var coordo model.User
			if err := Map2Struct(c, &coordo); err != nil {
				return coordos, err
			}
			coordos = append(coordos, coordo)
		}
		return coordos, err
	}

	// Return first met parent coordos
	var parents []string
	node, err := db.GetDB().Meta("getParentFromTid", map[string]string{"tid": tid})
	if err != nil {
		return coordos, LogErr("Internal Error", err)
	}
	if len(node) == 0 || node[0]["parent"] == nil {
		return coordos, err
	}
	// @debug: dql decoding !
	if nodes := node[0]["parent"].([]interface{}); len(nodes) > 0 {
		if nids, ok := nodes[0].(model.JsonAtom)["nameid"]; ok && nids != nil {
			switch x := nids.(type) {
			case []interface{}:
				for _, v := range x {
					parents = append(parents, v.(string))
				}
			case string:
				parents = append(parents, x)
			}
		}
	}
	for _, nameid := range parents {
		res, err := db.GetDB().Meta("getCoordos2", map[string]string{"nameid": nameid, "user_payload": UserSelection})
		if err != nil {
			return coordos, LogErr("Internal error", err)
		}

		// stop at the first circle with coordos
		if len(res) > 0 {
			for _, c := range res {
				var coordo model.User
				if err := Map2Struct(c, &coordo); err != nil {
					return coordos, err
				}
				coordos = append(coordos, coordo)
			}
			return coordos, err
		}
	}

	return coordos, err
}

func GetPeersFromTid(tid string) ([]model.User, error) {
	var peers []model.User

	// Fetch Peer users in receiver circle.
	nodes, err := db.GetDB().Meta("getPeersFromTid", map[string]string{"tid": tid, "user_payload": UserSelection})
	if err != nil {
		return peers, LogErr("Internal error", err)
	}

	// Return direct peers
	for _, c := range nodes {
		var peer model.User
		if err := Map2Struct(c, &peer); err != nil {
			return peers, err
		}
		peers = append(peers, peer)
	}

	return peers, err
}

//
// Sanitize TensionQuery
//

// NameidsProtected and Username information into the query.
func QueryAuthFilter(uctx model.UserCtx, q *db.TensionQuery) error {
	if q == nil {
		return fmt.Errorf("Empty query")
	}

	res, err := db.GetDB().Query(uctx, "node", "nameid", q.Nameids, "nameid visibility")
	if err != nil {
		return err
	}

	// For circle with visibility right
	var nameids []string
	// For circle with restricted visibility right
	var nameidsProtected []string

	for _, r := range res {
		nameid := r["nameid"]
		visibility := r["visibility"]

		// Get the nearest circle
		nid, err := codec.Nid2pid(r["nameid"])
		if err != nil {
			return err
		}

		if visibility == string(model.NodeVisibilityPrivate) && UserIsMember(&uctx, nid) < 0 {
			// If Private & non Member
			nameidsProtected = append(nameidsProtected, nameid)
		} else if visibility == string(model.NodeVisibilitySecret) && UserHasRole(&uctx, nid) < 0 {
			// If Secret & non Peer
			nameidsProtected = append(nameidsProtected, nameid)
		} else {
			// else (Public or with rights)
			nameids = append(nameids, nameid)
		}
	}

	q.Nameids = nameids
	q.NameidsProtected = nameidsProtected
	q.Username = uctx.Username
	// add NameidsProtected attribute in TensionQuery
	if len(nameids)+len(nameidsProtected) == 0 {
		return fmt.Errorf("error: no node name given (nameid empty)")
	}

	return nil
}
