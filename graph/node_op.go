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

// node_op implements the DB-effect side of Node governance operations.
// All Try* functions share the uniform (uctx, tension, subject, extras...) error
// signature for consistency with the EMAP action layer, even when uctx/tension
// are unused. Shape and lifecycle validation happens earlier in
// resolveGovernanceSubject (tension_governance.go), so Try* trusts the subject.

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

// TryAddNode creates the governed Node from the blob fragment, links it to the
// tension and flags the blob as pushed. Returns the new node uid.
func TryAddNode(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject) (string, error) {
	fragment := subject.blob.Node
	auth.InheritNodeCharacDefault(fragment, tension.Receiver)

	nid, err := PushNode(uctx.Username, tension, &subject.blob.ID, fragment, subject.nameid, tension.Receiver.Nameid)
	if err != nil {
		return "", err
	}
	if err := db.GetDB().LinkGovernedNode(tension.ID, nid, subject.blob.ID); err != nil {
		return "", err
	}
	return nid, db.GetDB().SetPushedFlagBlob(subject.blob.ID, Now())
}

// TryUpdateNode updates the governed Node from the blob fragment and flags the blob as pushed.
func TryUpdateNode(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject) error {
	nameid := subject.nameid
	fragment := subject.blob.Node
	// Map NodeFragment to Node Patch Input.
	// The NodeFragment copy is only necessary for the @search feature.
	// see https://discuss.dgraph.io/t/fulltext-search-across-multiple-fields/14354
	nodePatchFilter := StructMap[model.NodePatchFromFragment](fragment)
	nodePatch := StructMap[model.NodePatch](nodePatchFilter)
	nodePatch.Source = &model.BlobRef{ID: &subject.blob.ID}
	nodeInput := model.UpdateNodeInput{
		Filter: &model.NodeFilter{Nameid: &model.StringHashFilterStringRegExpFilter{Eq: &nameid}},
		Set:    &nodePatch,
		// Remove: &delNodePatch, // @debug: omitempty issues
	}
	if err := db.GetDB().Update(db.GetDB().GetRootUctx(), "node", nodeInput); err != nil {
		return err
	}

	// Update tension title
	if err := db.GetDB().SetFieldById(tension.ID, "Tension.title", codec.UpdateTensionTitle(subject.node.Type, codec.IsRoot(nameid), *fragment.Name)); err != nil {
		return err
	}
	return db.GetDB().SetPushedFlagBlob(subject.blob.ID, Now())
}

func TryChangeArchiveNode(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject, archived bool) error {
	nameid := subject.nameid

	if archived {
		// Archive
		// --
		// Check that circle has no children
		if *subject.blob.Node.Type == model.NodeTypeCircle {
			children, err := db.GetDB().GetChildren(nameid)
			if err != nil {
				return err
			}
			if len(children) > 0 {
				return fmt.Errorf("Cannot archive circle with active children. Please archive children first.")
			}
		}
	} else {
		// Unarchive
		// --
		// Check that parent node is not archived
		parentIsArchived, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.parent", "Node.isArchived")
		if err != nil {
			return err
		}
		if parentIsArchived != nil && parentIsArchived.(bool) {
			return fmt.Errorf("Cannot unarchive node with archived parent. Please unarchive parent first.")
		}
	}

	if err := db.GetDB().SetFieldById(subject.node.ID, "Node.isArchived", strconv.FormatBool(archived)); err != nil {
		return err
	}

	// Eventually unlink first-link, once the archive is persisted. Unlink errors do not block it.
	if archived && subject.node.FirstLink != nil {
		rootnameid, _ := codec.Nid2rootid(nameid) // subject.nameid was validated by the resolver.
		UnlinkUser(rootnameid, nameid, subject.node.FirstLink.Username)
	}

	return nil
}

func TryChangeAuthority(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject, value string) error {
	nameid := subject.nameid
	node := subject.blob.Node
	var err error

	switch *node.Type {
	case model.NodeTypeRole:
		if !model.RoleType(value).IsValid() {
			return fmt.Errorf("Bad value for role_type.")
		}
		if codec.IsMembershipRoleType(model.RoleType(value)) {
			return fmt.Errorf("Membership roles are protected and cannot be created like this.")
		}
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.role_type", value)
		if err != nil {
			return err
		}
		_, err = db.GetDB().Meta("setSubFieldByEq", map[string]string{
			"fieldid": "Node.nameid", "objid": nameid,
			"predicate1": "Node.role_ext", "predicate2": "RoleExt.role_type", "value": value,
		})
		if err != nil {
			return err
		}
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.role_type", value)
	case model.NodeTypeCircle:
		if !model.NodeMode(value).IsValid() {
			return fmt.Errorf("Bad value for mode.")
		}
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.mode", value)
		if err != nil {
			return err
		}
		err = db.GetDB().SetFieldById(node.ID, "NodeFragment.mode", value)
	}

	return err
}

func TryChangeVisibility(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject, value string) error {
	nameid := subject.nameid

	visibility := model.NodeVisibility(value)
	if !visibility.IsValid() {
		return fmt.Errorf("Bad value for visibility.")
	}
	// Update Node
	_, err := db.GetDB().Meta("setNodeVisibility", map[string]string{"nameid": nameid, "value": value})
	if err != nil {
		return err
	}

	// If nameid is the root, fix the organisation config.
	rootid, _ := codec.Nid2rootid(nameid)
	if visibility != model.NodeVisibilityPublic && nameid == rootid {
		err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.userCanJoin", strconv.FormatBool(false))
		if err != nil {
			return err
		}
	}

	// Change all role direct children
	return db.GetDB().SetChildrenRoleVisibility(nameid, value)
}

func TryUpdateLink(uctx *model.UserCtx, tension *model.Tension, subject *governanceSubject, event *model.EventRef) error {
	nameid := subject.nameid
	rootnameid, err := codec.Nid2rootid(nameid)
	if err != nil {
		return err
	}

	// Get the current first link
	firstLink, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.first_link", "User.username")
	if err != nil {
		return err
	}

	// MemberUnlinked carries the role type in event.New, not a username: the
	// fragment first_link is cleared on unlink (consistent with LeaveRole).
	fragmentLink := ""
	switch *event.EventType {
	case model.TensionEventMemberLinked:
		if firstLink != nil {
			return fmt.Errorf("Role is already linked.")
		}
		if err := LinkUser(rootnameid, nameid, *event.New); err != nil {
			return err
		}
		fragmentLink = *event.New
	case model.TensionEventMemberUnlinked:
		if err := UnlinkUser(rootnameid, nameid, *event.Old); err != nil {
			return err
		}
	}

	// Update NodeFragment
	fragment := subject.blob.Node
	if fragment.ID != "" {
		return db.GetDB().SetFieldById(fragment.ID, "NodeFragment.first_link", fragmentLink)
	}
	return nil
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
	noderef := StructMap[model.NodeFragmentRef](node)
	emptyString := "" // root's tension feature
	noderef.Nameid = &emptyString
	noderef.About = about
	noderef.Mandate = mandate
	blob := model.BlobRef{
		CreatedAt:  &now,
		CreatedBy:  &createdBy,
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
