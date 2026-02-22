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

package auth

import (
	"fmt"
	"slices"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
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

// Authorize converts a (bool, error) auth result into a single error suitable
// for callers that just need a pass/fail gate.  It returns nil on success,
// passes through system errors, and produces a LogErr on denial.
func Authorize(ok bool, err error) error {
	if err != nil {
		return err
	}
	if !ok {
		return LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this resource."))
	}
	return nil
}

// CheckNodesAuth checks that user satisfies strict condition (coordo roles on the given nodes).
// Mandatory field in each NodeRef: nameid.
func CheckNodesAuth(uctx *model.UserCtx, nodes []model.NodeRef, passAll bool) (bool, error) {
	var ok bool
	var err error

	// Check @auth
	// @optimize
	mode := model.NodeModeCoordinated
	for _, n := range nodes {
		if n.Nameid == nil {
			return false, LogErr("Access denied", fmt.Errorf("nameid in required in artefact nodes fields."))
		}

		ok, err = HasCoordoAuth(uctx, *n.Nameid, &mode)
		if err != nil {
			return false, err
		}

		if passAll && !ok {
			break
		} else if ok {
			break
		}
	}

	if len(nodes) == 0 {
		ok = true
	}
	return ok, nil
}

// CheckProjectAuth verifies the user has write access to a project.
// Access is granted if the user is either:
//   - a project collaborator, or
//   - a coordinator of any node linked to the project
func CheckProjectAuth(uctx *model.UserCtx, projectid string) (bool, error) {
	// Check collaborator access first (cheap username match)
	if ok, err := isProjectCollaborator(uctx, projectid); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}

	// Fall back to node-based coordinator authorization
	return checkProjectNodeAuth(uctx, projectid)
}

// isProjectCollaborator checks if the user is listed as a project collaborator.
func isProjectCollaborator(uctx *model.UserCtx, projectid string) (bool, error) {
	x, err := db.GetDB().GetSubFieldById(projectid, "Project.collaborators", "User.username")
	if err != nil {
		return false, LogErr("Internal error", err)
	}
	return slices.Contains(InterfaceToSlice[string](x), uctx.Username), nil
}

// checkProjectNodeAuth verifies the user has coordinator authority on at least
// one node linked to the project.
func checkProjectNodeAuth(uctx *model.UserCtx, projectid string) (bool, error) {
	x, err := db.GetDB().GetSubFieldById(projectid, "Project.nodes", "Node.nameid")
	if err != nil {
		return false, LogErr("Internal error", err)
	}
	nameids := InterfaceToSlice[string](x)
	if len(nameids) == 0 {
		// Allow access when the project has no linked nodes
		return true, nil
	}
	nodes := make([]model.NodeRef, len(nameids))
	for i, n := range nameids {
		nodes[i] = model.NodeRef{Nameid: &n}
	}
	return CheckNodesAuth(uctx, nodes, false)
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
		return ok, LogErr("Internal error", err)
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
	// Fetch Coordo users in receiver circle.
	coordos, err := db.Meta[model.User]("getCoordosFromTid", map[string]string{"tid": tid, "user_payload": UserSelection})
	if err != nil {
		return nil, LogErr("Internal error", err)
	}

	// Return direct coordos if present
	if len(coordos) > 0 {
		return coordos, nil
	}

	// Return first met parent coordos
	var parents []string
	node, err := db.GetDB().Meta("getParentFromTid", map[string]string{"tid": tid})
	if err != nil {
		return coordos, LogErr("Internal error", err)
	}
	if len(node) == 0 || node[0]["parent"] == nil {
		return coordos, err
	}
	// @debug: dql decoding !
	if nodes := node[0]["parent"].([]any); len(nodes) > 0 {
		if nids, ok := nodes[0].(model.JsonAtom)["nameid"]; ok && nids != nil {
			switch x := nids.(type) {
			case []any:
				for _, v := range x {
					parents = append(parents, v.(string))
				}
			case string:
				parents = append(parents, x)
			}
		}
	}
	for _, nameid := range parents {
		res, err := db.Meta[model.User]("getCoordos2", map[string]string{"nameid": nameid, "user_payload": UserSelection})
		if err != nil {
			return coordos, LogErr("Internal error", err)
		}

		// stop at the first circle with coordos
		if len(res) > 0 {
			return res, nil
		}
	}

	return coordos, nil
}

func GetPeersFromTid(tid string) ([]model.User, error) {
	// Fetch Peer users in receiver circle.
	peers, err := db.Meta[model.User]("getPeersFromTid", map[string]string{"tid": tid, "user_payload": UserSelection})
	if err != nil {
		return nil, LogErr("Internal error", err)
	}

	return peers, nil
}

//
// Sanitize TensionQuery
//

// NodeVisibilityFilter checks a set of node nameids and returns only those
// the user is authorized to see based on visibility rules:
//   - Public nodes are always visible
//   - Private nodes require org membership
//   - Secret nodes require a role in the circle
func NodeVisibilityFilter(uctx *model.UserCtx, nameids []string) (map[string]bool, error) {
	visible := make(map[string]bool)
	if len(nameids) == 0 {
		return visible, nil
	}

	res, err := db.GetDB().Query(*uctx, "node", "nameid", nameids, "nameid visibility")
	if err != nil {
		return nil, err
	}

	for _, r := range res {
		nameid := r["nameid"]
		visibility := r["visibility"]
		nid, err := codec.Nid2pid(nameid)
		if err != nil {
			return nil, err
		}

		switch visibility {
		case string(model.NodeVisibilityPrivate):
			if UserIsMember(uctx, nid) >= 0 {
				visible[nameid] = true
			}
		case string(model.NodeVisibilitySecret):
			if UserHasRole(uctx, nid) >= 0 {
				visible[nameid] = true
			}
		default: // Public
			visible[nameid] = true
		}
	}

	return visible, nil
}

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
