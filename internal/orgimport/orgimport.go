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

// Package orgimport parses organisation spreadsheet exports (HolaSpirit, ...)
// into an intermediate tree and materialises it as a Fractale organisation.
package orgimport

import (
	"fmt"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/tools"
)

//
// Intermediate representation for imported org trees
//

// ImportNode represents a node in the imported organisation tree.
type ImportNode struct {
	Name             string
	Purpose          string           // maps to Mandate.purpose
	Domains          string           // maps to Mandate.domains
	Policies         string           // maps to Mandate.policies
	Rules            string           // maps to Mandate.rules
	Responsabilities string           // maps to Mandate.responsabilities
	Type             model.NodeType   // Circle or Role
	RoleType         *model.RoleType  // Coordinator, Peer, etc. (roles only)
	Children         []*ImportNode    // sub-circles and roles
	RoleExtRef       string           // name of RoleExt template (if deduplicated)
	RoleExtTemplates []*ImportRoleExt // role templates (root node only)
}

// ImportRoleExt represents a role template to be created as RoleExt.
type ImportRoleExt struct {
	Name     string
	About    string
	RoleType model.RoleType
	Mandate  *ImportMandate
}

// ImportMandate holds mandate data for a role template.
type ImportMandate struct {
	Purpose          string
	Domains          string
	Policies         string
	Rules            string
	Responsabilities string
}

//
// Format detection and dispatch
//

// DetectSourceFormat guesses the source platform from sheet names.
func DetectSourceFormat(sheets map[string][][]string) string {
	if _, ok := sheets["Circles & Roles"]; ok {
		return "holaspirit"
	}
	return ""
}

// ParseByFormat dispatches to the appropriate adapter.
func ParseByFormat(format string, sheets map[string][][]string) (*ImportNode, error) {
	switch strings.ToLower(format) {
	case "holaspirit":
		return parseHolaSpirit(sheets)
	default:
		return nil, fmt.Errorf("unsupported or undetected source format %q", format)
	}
}

//
// Organisation builder
//

// BuildOrgFromTree creates a Fractale organisation from an ImportNode tree.
func BuildOrgFromTree(uctx *model.UserCtx, form model.OrgaForm, tree *ImportNode, visibility model.NodeVisibility) error {
	nameid := form.Nameid
	mode := model.NodeModeCoordinated

	// Create root node, its control tension and the owner role
	if _, err := graph.CreateRootOrga(uctx, form, visibility, buildMandateRef(tree)); err != nil {
		return err
	}

	// Create RoleExt templates at root level, linked to the root node
	roleExtIDs := make(map[string]string) // roleName -> roleExt ID
	for _, re := range tree.RoleExtTemplates {
		reID, err := createRoleExt(nameid, re)
		if err != nil {
			return fmt.Errorf("creating role template %q: %w", re.Name, err)
		}
		roleExtIDs[re.Name] = reID
	}

	// Recursively create children
	for _, child := range tree.Children {
		err := createChildNode(uctx.Username, nameid, nameid, child, visibility, mode, roleExtIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

// createChildNode recursively creates a child node (circle or role) in the graph.
func createChildNode(username, rootnameid, parentid string, node *ImportNode, visibility model.NodeVisibility, mode model.NodeMode, roleExtIDs map[string]string) error {
	// Generate nameid
	nodeNameid := tools.NameidEncoder(node.Name)
	_, nameid, err := codec.NodeIdCodec(parentid, nodeNameid, node.Type)
	if err != nil {
		return fmt.Errorf("generating nameid for %q: %w", node.Name, err)
	}

	// Build mandate
	mandate := buildMandateRef(node)

	// Build node input
	rt := model.RoleTypePeer
	if node.RoleType != nil {
		rt = *node.RoleType
	}

	nodeInput := model.AddNodeInput{
		CreatedAt:  tools.Now(),
		CreatedBy:  &model.UserRef{Username: &username},
		Nameid:     nameid,
		Rootnameid: rootnameid,
		Parent:     &model.NodeRef{Nameid: &parentid},
		Name:       node.Name,
		Type:       node.Type,
		IsRoot:     false,
		IsArchived: false,
		Rights:     0,
		Visibility: visibility,
		Mode:       mode,
	}

	if node.Type == model.NodeTypeRole {
		nodeInput.RoleType = &rt
		// Link to RoleExt template if available
		if node.RoleExtRef != "" {
			if reID, ok := roleExtIDs[node.RoleExtRef]; ok {
				nodeInput.RoleExt = &model.RoleExtRef{ID: &reID}
			}
		}
	}

	// Create the node
	nodeID, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
	if err != nil {
		return fmt.Errorf("creating node %q: %w", node.Name, err)
	}

	// Create a governance tension for nodes with mandate data
	if mandate != nil {
		if err := createNodeTension(username, rootnameid, parentid, nameid, nodeID, node, mandate); err != nil {
			return fmt.Errorf("creating governance tension for %q: %w", node.Name, err)
		}
	}

	// Recursively create children (only circles have children)
	if node.Type == model.NodeTypeCircle {
		for _, child := range node.Children {
			err := createChildNode(username, rootnameid, nameid, child, visibility, mode, roleExtIDs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// createNodeTension creates a governance tension for an imported node.
func createNodeTension(username, rootnameid, parentid, nameid, nodeID string, node *ImportNode, mandate *model.MandateRef) error {
	now := tools.Now()
	createdBy := model.UserRef{Username: &username}
	emitter := model.NodeRef{Nameid: &parentid}
	receiver := model.NodeRef{Nameid: &parentid}

	evt1 := model.TensionEventCreated
	evt2 := model.TensionEventBlobCreated
	evt3 := model.TensionEventBlobPushed
	isAnchor := nameid == rootnameid
	emptyMsg := ""

	// Build NodeFragmentRef for the blob
	nodeName := node.Name
	localNameid := tools.NameidEncoder(node.Name)
	nodeType := node.Type
	nodeFragRef := model.NodeFragmentRef{
		Nameid:  &localNameid,
		Name:    &nodeName,
		Type:    &nodeType,
		Mandate: mandate,
	}
	if node.Type == model.NodeTypeRole {
		roleType := model.RoleTypePeer
		if node.RoleType != nil {
			roleType = *node.RoleType
		}
		nodeFragRef.RoleType = &roleType
	}

	blob := model.BlobRef{
		CreatedAt:  &now,
		CreatedBy:  &createdBy,
		Node:       &nodeFragRef,
		PushedFlag: &now,
	}

	tension := model.AddTensionInput{
		CreatedAt:  now,
		CreatedBy:  &createdBy,
		Title:      codec.UpdateTensionTitle(node.Type, isAnchor, nodeName),
		Type:       model.TensionTypeGovernance,
		Status:     model.TensionStatusClosed,
		Emitter:    &emitter,
		Receiver:   &receiver,
		Emitterid:  parentid,
		Receiverid: parentid,
		History: []*model.EventRef{
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt1},
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt2},
			{CreatedAt: &now, CreatedBy: &createdBy, EventType: &evt3},
		},
		Blobs:       []*model.BlobRef{&blob},
		Comments:    []*model.CommentRef{{CreatedAt: &now, CreatedBy: &createdBy, Message: &emptyMsg}},
		Subscribers: []*model.UserRef{&createdBy},
	}

	tid, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "tension", tension)
	if err != nil {
		return err
	}

	// Link source blob to node
	bid := db.GetDB().GetLastBlobId(tid)
	if bid == nil {
		return fmt.Errorf("no blob found for imported governance tension %s", tid)
	}
	return db.GetDB().LinkGovernedNode(tid, nodeID, *bid)
}

// createRoleExt creates a RoleExt template in the database and returns its ID.
// It links the template to the root node via the Nodes field (hasInverse with Node.roles).
func createRoleExt(rootnameid string, re *ImportRoleExt) (string, error) {
	input := model.AddRoleExtInput{
		Rootnameid: rootnameid,
		Name:       re.Name,
		RoleType:   re.RoleType,
		Nodes:      []*model.NodeRef{{Nameid: &rootnameid}},
	}
	if re.About != "" {
		input.About = &re.About
	}
	if re.Mandate != nil {
		m := &model.MandateRef{}
		if re.Mandate.Purpose != "" {
			m.Purpose = &re.Mandate.Purpose
		}
		if re.Mandate.Responsabilities != "" {
			m.Responsabilities = &re.Mandate.Responsabilities
		}
		if re.Mandate.Domains != "" {
			m.Domains = &re.Mandate.Domains
		}
		if re.Mandate.Policies != "" {
			m.Policies = &re.Mandate.Policies
		}
		if re.Mandate.Rules != "" {
			m.Rules = &re.Mandate.Rules
		}
		// Only set mandate if at least one field is non-empty
		if m.Purpose != nil || m.Responsabilities != nil || m.Domains != nil || m.Policies != nil || m.Rules != nil {
			input.Mandate = m
		}
	}
	return db.GetDB().Add(db.GetDB().GetRootUctx(), "roleExt", input)
}

// buildMandateRef creates a MandateRef from an ImportNode if it has mandate data.
// Empty fields are left nil rather than set to empty strings.
func buildMandateRef(node *ImportNode) *model.MandateRef {
	if node.Purpose == "" && node.Domains == "" && node.Policies == "" && node.Rules == "" && node.Responsabilities == "" {
		return nil
	}
	m := &model.MandateRef{}
	if node.Purpose != "" {
		m.Purpose = &node.Purpose
	}
	if node.Responsabilities != "" {
		m.Responsabilities = &node.Responsabilities
	}
	if node.Domains != "" {
		m.Domains = &node.Domains
	}
	if node.Policies != "" {
		m.Policies = &node.Policies
	}
	if node.Rules != "" {
		m.Rules = &node.Rules
	}
	return m
}
