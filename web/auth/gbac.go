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
			return false, LogErr("Access denied", fmt.Errorf("nameid in required in nodes fields."))
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

// projectAuthData holds the fields fetched by a single DQL call in CheckProjectAuth.
type projectAuthData struct {
	Collaborators       []struct{ Username string } `json:"collaborators"`
	PeerCanEditProject  bool                        `json:"peerCanEditProject"`
	GuestCanEditProject bool                        `json:"guestCanEditProject"`
	Rootnameid          string                      `json:"rootnameid"`
	Nodes               []struct{ Nameid string }   `json:"nodes"`
}

// projectAuthFields is the DQL predicate list fetched once for project authorization.
const projectAuthFields = `Project.collaborators { User.username }
            Project.peerCanEditProject
            Project.guestCanEditProject
            Project.rootnameid
            Project.nodes { Node.nameid }`

// CheckProjectAuth verifies the user has write access to a project.
// A single DB call fetches collaborators, permission flags, and linked nodes,
// then checks access in order:
//  1. Collaborator (username match)
//  2. Permission flags (peerCanEditProject / guestCanEditProject)
//  3. Coordinator on any linked node
func CheckProjectAuth(uctx *model.UserCtx, projectid string) (bool, error) {
	if db.ValidateUids(projectid) != nil {
		// Zero-value loc (resource deleted/not found) or raw client id;
		// reject before it reaches uid(...) as a Dgraph parse error.
		return false, fmt.Errorf("project not found")
	}
	r, err := db.GetDB().GetByUid(projectid, projectAuthFields)
	if err != nil {
		return false, LogErr("Internal error", err)
	}

	p := StructMap[projectAuthData](r)

	// 1. Collaborator (fast path)
	for _, c := range p.Collaborators {
		if c.Username == uctx.Username {
			return true, nil
		}
	}

	// 2. Permission flags
	if p.PeerCanEditProject && p.Rootnameid != "" {
		if UserIsGuest(uctx, p.Rootnameid) >= 0 {
			if p.GuestCanEditProject {
				return true, nil // guests need both flags
			}
		} else if UserIsMember(uctx, p.Rootnameid) >= 0 {
			return true, nil // non-guest members pass with peerFlag alone
		}
	}

	// 3. Coordinator on linked nodes
	if len(p.Nodes) == 0 {
		return true, nil // no linked nodes — allow access
	}
	nodes := make([]model.NodeRef, len(p.Nodes))
	for i, n := range p.Nodes {
		nameid := n.Nameid
		nodes[i] = model.NodeRef{Nameid: &nameid}
	}
	return CheckNodesAuth(uctx, nodes, false)
}

// HasCoordoAuth tells if the user has authority in the given node.
func HasCoordoAuth(uctx *model.UserCtx, nameid string, mode *model.NodeMode) (bool, error) {
	// Get the node mode eventually
	if mode == nil {
		mode_, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.mode")
		if err != nil {
			return false, LogErr("Internal error", err)
		}
		if mode_ == nil {
			return false, fmt.Errorf("node not found: %s", nameid)
		}
		m := model.NodeMode(mode_.(string))
		mode = &m
	}

	// Check user rights
	ok, err := CheckUserAuth(uctx, nameid, *mode)
	if err != nil {
		return ok, LogErr("Internal error", err)
	}

	// If the node has no Coordo roles, check authority in parent circles.
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

	switch mode {
	case model.NodeModeAgile:
		ok = UserHasRole(uctx, nid) >= 0
	case model.NodeModeCoordinated:
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

// IsNodeVisible returns true when uctx may see a node with the given visibility:
//   - Public nodes are always visible
//   - Private nodes require org membership
//   - Secret nodes require a role in the circle
func IsNodeVisible(uctx *model.UserCtx, nameid string, vis model.NodeVisibility) (bool, error) {
	nid, err := codec.Nid2pid(nameid)
	if err != nil {
		return false, err
	}
	switch vis {
	case model.NodeVisibilityPrivate:
		return UserIsMember(uctx, nid) >= 0, nil
	case model.NodeVisibilitySecret:
		return UserHasRole(uctx, nid) >= 0, nil
	default: // Public
		return true, nil
	}
}

// ClassifyVisibleNameids returns the subset of nameids in visMap that uctx
// is authorized to see (per IsNodeVisible).
func ClassifyVisibleNameids(uctx *model.UserCtx, visMap map[string]model.NodeVisibility) ([]string, error) {
	out := make([]string, 0, len(visMap))
	for nameid, vis := range visMap {
		ok, err := IsNodeVisible(uctx, nameid, vis)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, nameid)
		}
	}
	return out, nil
}

// NameidsProtected and Username information into the query.
func QueryAuthFilter(uctx model.UserCtx, q *db.TensionQuery) error {
	if q == nil {
		return fmt.Errorf("Empty query")
	}

	visMap, err := db.GetDB().GetNodeVisibilities(q.Nameids)
	if err != nil {
		return err
	}

	// For circle with visibility right
	var nameids []string
	// For circle with restricted visibility right
	var nameidsProtected []string

	for _, nameid := range q.Nameids {
		vis, ok := visMap[nameid]
		if !ok {
			continue
		}
		visible, err := IsNodeVisible(&uctx, nameid, vis)
		if err != nil {
			return err
		}
		if visible {
			nameids = append(nameids, nameid)
		} else {
			nameidsProtected = append(nameidsProtected, nameid)
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
