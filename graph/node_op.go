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
	"strconv"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

// tryAddNode add a new node if user has the correct right
func TryAddNode(uctx *model.UserCtx, tension *model.Tension, node *model.NodeFragment, bid *string) (bool, string, error) {
	parentid := tension.Receiver.Nameid

	auth.InheritNodeCharacDefault(node, tension.Receiver)

	// Get References
	_, nameid, err := codec.NodeIdCodec(parentid, *node.Nameid, *node.Type)
	if err != nil {
		return false, "", err
	}

	ok, err := NodeCheck(node, nameid)
	if err != nil || !ok {
		return ok, "", err
	}

	nid, err := PushNode(uctx.Username, tension, bid, node, nameid, parentid)
	return ok, nid, err
}

func TryUpdateNode(tension *model.Tension, node *model.NodeFragment, governed *model.Node, bid *string) (bool, error) {
	ok, err := NodeCheck(node, governed.Nameid)
	if err != nil || !ok {
		return ok, err
	}

	return ok, UpdateNode(tension, bid, node, governed)
}

func TryChangeArchiveNode(node *model.NodeFragment, governed *model.Node, bid string, archived bool) (bool, error) {
	nameid := governed.Nameid
	ok, err := NodeCheck(node, nameid)
	if err != nil || !ok {
		return ok, err
	}

	if archived {
		// Archive
		// --
		// Check that circle has no children
		if *node.Type == model.NodeTypeCircle {
			children, err := db.GetDB().GetChildren(nameid)
			if err != nil {
				return ok, err
			}
			if len(children) > 0 {
				return ok, fmt.Errorf("Cannot archive circle with active children. Please archive children first.")
			}
		}
	} else {
		// Unarchive
		// --
		// Check that parent node is not archived
		parentIsArchived, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.parent", "Node.isArchived")
		if err != nil {
			return ok, err
		}
		if parentIsArchived != nil && parentIsArchived.(bool) {
			return ok, fmt.Errorf("Cannot unarchive node with archived parent. Please unarchive parent first.")
		}
	}

	if err = db.GetDB().SetGovernedNodeArchived(governed.ID, bid, Now(), archived); err != nil {
		return false, err
	}

	// Eventually unlink first-link, once the archive is persisted. Unlink errors do not block it.
	if archived && governed.FirstLink != nil {
		rootnameid, _ := codec.Nid2rootid(nameid) // NodeCheck already validated the nameid.
		UnlinkUser(rootnameid, nameid, governed.FirstLink.Username)
	}

	return ok, err
}

func TryChangeAuthority(node *model.NodeFragment, governed *model.Node, value string) (bool, error) {
	nameid := governed.Nameid
	ok, err := NodeCheck(node, nameid)
	if err != nil || !ok {
		return ok, err
	}

	switch *node.Type {
	case model.NodeTypeRole:
		if !model.RoleType(value).IsValid() {
			return false, fmt.Errorf("Bad value for role_type.")
		}
		if codec.IsMembershipRoleType(model.RoleType(value)) {
			return false, fmt.Errorf("Membership roles are protected and cannot be created like this.")
		}
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.role_type", value)
		if err != nil {
			return false, err
		}
		_, err = db.GetDB().Meta("setSubFieldByEq", map[string]string{
			"fieldid": "Node.nameid", "objid": nameid,
			"predicate1": "Node.role_ext", "predicate2": "RoleExt.role_type", "value": value,
		})
		if err != nil {
			return false, err
		}
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.role_type", value)
	case model.NodeTypeCircle:
		if !model.NodeMode(value).IsValid() {
			return false, fmt.Errorf("Bad value for mode.")
		}
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.mode", value)
		if err != nil {
			return false, err
		}
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.mode", value)
	}

	return ok, err
}

func TryChangeVisibility(node *model.NodeFragment, governed *model.Node, value string) (bool, error) {
	nameid := governed.Nameid
	ok, err := NodeCheck(node, nameid)
	if err != nil || !ok {
		return ok, err
	}

	visibility := model.NodeVisibility(value)
	if !visibility.IsValid() {
		return false, fmt.Errorf("Bad value for visibility.")
	}
	// Update Node
	_, err = db.GetDB().Meta("setNodeVisibility", map[string]string{"nameid": nameid, "value": value})
	if err != nil {
		return false, err
	}

	// If nameid is the root, fix the organisation config.
	rootid, _ := codec.Nid2rootid(nameid)
	if visibility != model.NodeVisibilityPublic && nameid == rootid {
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.userCanJoin", strconv.FormatBool(false))
		if err != nil {
			return false, err
		}
	}

	// Change all role direct children
	err = db.GetDB().SetChildrenRoleVisibility(nameid, value)
	return ok, err
}

func TryUpdateLink(node *model.NodeFragment, governed *model.Node, event *model.EventRef, unsafe bool) (bool, error) {
	var err error
	var rootnameid string
	var nameid string

	// unsafe allows Guest user to be unlinked, as the nameid includes a "@" char.
	if unsafe {
		nameid = *node.Nameid
		rootnameid, err = codec.Nid2rootid(nameid)
		if err != nil {
			return false, err
		}
	} else {
		nameid = governed.Nameid
		rootnameid, err = codec.Nid2rootid(nameid)
		if err != nil {
			return false, err
		}
		ok, err := NodeCheck(node, nameid)
		if err != nil || !ok {
			return false, err
		}
	}

	// Get the current first link
	firstLink, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.first_link", "User.username")
	if err != nil {
		return false, err
	}

	switch *event.EventType {
	case model.TensionEventMemberLinked:
		// Link user
		// --
		if firstLink != nil {
			return false, fmt.Errorf("Role is already linked.")
		}
		err = LinkUser(rootnameid, nameid, *event.New)
		if err != nil {
			return false, err
		}
	case model.TensionEventMemberUnlinked:
		// UnLink user
		// --
		err = UnlinkUser(rootnameid, nameid, *event.Old)
		if err != nil {
			return false, err
		}
	}

	// Update NodeFragment
	if node.ID != "" {
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.first_link", *event.New)
	}

	return true, err
}

// NodeCheck validates the fragment fields written to a Node.
func NodeCheck(node *model.NodeFragment, nameid string) (bool, error) {
	if node == nil || node.Name == nil {
		return false, fmt.Errorf("node fragment name is required")
	}
	var ok bool
	var err error

	// Validate nameid
	// @obsolete with NodeIdCodec ?
	rootnameid, err := codec.Nid2rootid(nameid)
	if err != nil {
		return ok, err
	}
	err = auth.ValidateNameid(nameid, rootnameid)
	if err != nil {
		return ok, err
	}

	// Validate Name
	name := *node.Name
	err = auth.ValidateName(name)
	if err != nil {
		return ok, err
	}

	// Validate special role-type from being created
	if node.RoleType != nil && codec.IsMembershipRoleType(*node.RoleType) {
		return false, fmt.Errorf("Membership roles are protected and cannot be created like this.")
	}

	ok = true
	return ok, err
}

// PushNode add a new role or circle in an graph.
// * It adds automatic fields such as createdBy, createdAt, etc
// * It automatically add tension associated to potential children.
// The tension is the governance tension owning the node, or nil when no tension title follows the node.
func PushNode(username string, tension *model.Tension, bid *string, node *model.NodeFragment, nameid, parentid string) (string, error) {
	rootnameid, _ := codec.Nid2rootid(nameid)

	// Map NodeFragment to Node Input
	nodeInput := StructMap[model.AddNodeInput](node)

	// Fix Automatic fields
	nodeInput.CreatedAt = Now()
	nodeInput.CreatedBy = &model.UserRef{Username: &username}
	nodeInput.Nameid = nameid
	nodeInput.Rootnameid = rootnameid
	nodeInput.Parent = &model.NodeRef{Nameid: &parentid}
	nodeInput.IsRoot = false
	nodeInput.IsArchived = false
	nodeInput.Rights = 0
	if node.RoleExt != nil {
		nodeInput.RoleExt = &model.RoleExtRef{ID: node.RoleExt}
	}
	if bid != nil {
		nodeInput.Source = &model.BlobRef{ID: bid}
	}

	// Push the nodes into the database
	nid, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
	if err != nil || tension == nil {
		return nid, err
	}

	// Update tension title
	err = db.GetDB().SetFieldById(tension.ID, "Tension.title", codec.UpdateTensionTitle(*node.Type, *node.Nameid == "", *node.Name))
	return nid, err
}

// UpdateNode update a node from the given fragment
func UpdateNode(tension *model.Tension, bid *string, node *model.NodeFragment, governed *model.Node) error {
	nameid := governed.Nameid
	// Map NodeFragment to Node Patch Input
	// The NodeFraglent copy is only necesary for the @search feature.
	// see https://discuss.dgraph.io/t/fulltext-search-across-multiple-fields/14354
	nodePatchFilter := StructMap[model.NodePatchFromFragment](node)
	nodePatch := StructMap[model.NodePatch](nodePatchFilter)
	// Blob reference update
	if bid != nil {
		nodePatch.Source = &model.BlobRef{ID: bid}
	}
	// Build input
	nodeInput := model.UpdateNodeInput{
		Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
		Set:    &nodePatch,
		// Remove: &delNodePatch, // @debug: omitempty issues
	}
	// Update the node in database
	err := db.GetDB().Update(db.GetDB().GetRootUctx(), "node", nodeInput)
	if err != nil {
		return err
	}

	// Update tension title
	return db.GetDB().SetFieldById(tension.ID, "Tension.title", codec.UpdateTensionTitle(governed.Type, codec.IsRoot(nameid), *node.Name))
}

//
// Internals
//

// MakeNewRootTension build the tension that manage a root node. Authors will be suscribed.
func MakeNewRootTension(rootnameid string, node model.AddNodeInput, about *string, mandate *model.MandateRef) model.AddTensionInput {
	now := Now()
	createdBy := *node.CreatedBy
	emitter := model.NodeRef{Nameid: &rootnameid}
	receiver := model.NodeRef{Nameid: &rootnameid}
	evt1 := model.TensionEventCreated
	evt2 := model.TensionEventBlobCreated
	evt3 := model.TensionEventBlobPushed
	blob_type := model.BlobTypeOnNode
	noderef := StructMap[model.NodeFragmentRef](node)
	emptyString := "" // root's tension feature
	noderef.Nameid = &emptyString
	noderef.About = about
	noderef.Mandate = mandate
	blob := model.BlobRef{
		CreatedAt:  &now,
		CreatedBy:  &createdBy,
		BlobType:   &blob_type,
		Node:       &noderef,
		PushedFlag: &now,
	}
	tension := model.AddTensionInput{
		CreatedAt:  now,
		CreatedBy:  &createdBy,
		Title:      codec.UpdateTensionTitle(model.NodeTypeCircle, true, node.Name),
		Type:       model.TensionTypeGovernance,
		Status:     model.TensionStatusClosed,
		Emitter:    &emitter,
		Receiver:   &receiver,
		Emitterid:  rootnameid,
		Receiverid: rootnameid,
		History: []*model.EventRef{
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt1},
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt2},
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt3},
		},
		Blobs:       []*model.BlobRef{&blob},
		Comments:    []*model.CommentRef{{CreatedAt: &now, CreatedBy: &createdBy, Message: nil}},
		Subscribers: []*model.UserRef{&createdBy},
	}
	return tension
}

func MaybeAddPendingNode(username string, tension *model.Tension) (bool, error) {
	ok := false
	if tension.Receiver == nil {
		t, err := First(db.Meta[model.Tension]("getTensionSimple", map[string]string{"id": tension.ID}))
		if err != nil {
			return ok, err
		} else if t.Receiver == nil {
			return ok, fmt.Errorf("no tension found for tid: %s", tension.ID)
		}
		*tension = t
	}

	rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
	if err != nil {
		return ok, err
	}
	nid := codec.MemberIdCodec(rootid, username)
	n, err := db.GetDB().GetByEq("Node.nameid", nid, "Node.role_type Node.first_link{User.username}")
	if err != nil {
		return ok, err
	}
	node, _ := n.(model.JsonAtom)
	if node["role_type"] == nil {
		rt := model.RoleTypePending
		t := model.NodeTypeRole
		name := "Pending"
		n := &model.NodeFragment{
			Name:     &name,
			RoleType: &rt,
			Type:     &t,
		}
		auth.InheritNodeCharacDefault(n, tension.Receiver)
		_, err = PushNode(username, nil, nil, n, nid, rootid)
		if err != nil {
			return ok, err
		}
		err = db.GetDB().AddUserRole(username, nid)
		ok = true
	} else if node["first_link"] == nil {
		if err = db.GetDB().AddUserRole(username, nid); err != nil {
			return ok, err
		}
		err = db.GetDB().UpgradeMember(nid, model.RoleTypePending)
	} else if node["role_type"].(string) == string(model.RoleTypeRetired) {
		err = db.GetDB().UpgradeMember(nid, model.RoleTypePending)
	}

	return ok, err
}

func MaybeDeletePendingNode(username string, tension *model.Tension) error {
	rootid, err := codec.Nid2rootid(tension.Receiverid)
	if err != nil {
		return err
	}
	nid := codec.MemberIdCodec(rootid, username)
	filter := fmt.Sprintf(`eq(%s, "%s")`, "Node.role_type", "Pending")
	ex, err := db.GetDB().Exists("Node.nameid", nid, &filter)
	if err != nil {
		return err
	}
	if ex {
		// REMOVING node have unattended effect (emitter missing)
		//err := db.GetDB().RemoveUserRole(username, nid)
		//if err != nil { return err }
		//err = db.GetDB().Delete(db.GetDB().GetRootUctx(), "node", model.NodeFilter{
		//    Nameid: &model.StringHashFilterStringRegExpFilter{Eq:&nid},
		//})
		err = UnlinkUser(rootid, nid, username)
	}

	return err
}
