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

package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
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
	Responsabilities string
}

//
// HTTP Handler
//

// ImportOrga handles spreadsheet upload and creates an organisation from it.
func ImportOrga(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	uctx, err := auth.GetUserContextLight(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "failed to parse form: "+err.Error(), 400)
		return
	}

	// Extract form fields
	name := strings.TrimSpace(r.FormValue("name"))
	nameid := strings.TrimSpace(r.FormValue("nameid"))
	format := strings.TrimSpace(r.FormValue("format"))
	about := r.FormValue("about")

	var visibility model.NodeVisibility
	visStr := r.FormValue("visibility")
	if visStr != "" && model.NodeVisibility(visStr).IsValid() {
		visibility = model.NodeVisibility(visStr)
	} else {
		visibility = model.NodeVisibilityPublic
	}

	// Validate required fields
	if err := auth.ValidateName(name); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := auth.ValidateNameid(nameid, nameid); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if strings.Contains(nameid, "#") {
		http.Error(w, "Illegal character '#' in nameid", 400)
		return
	}

	// Check plan permissions
	form := model.OrgaForm{
		Name:       name,
		Nameid:     nameid,
		About:      &about,
		Visibility: &visibility,
	}
	ok, err := auth.CanNewOrga(*uctx, form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if !ok {
		http.Error(w, "permission denied: cannot create organisation", 403)
		return
	}

	// Extract uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing or invalid file: "+err.Error(), 400)
		return
	}
	defer file.Close()

	// Read spreadsheet
	sheets, err := readSpreadsheet(file, handler.Filename)
	if err != nil {
		http.Error(w, "failed to read spreadsheet: "+err.Error(), 400)
		return
	}

	// Detect source format and parse into tree
	if format == "" {
		format = detectSourceFormat(sheets)
	}
	tree, err := parseByFormat(format, sheets)
	if err != nil {
		http.Error(w, "failed to parse spreadsheet: "+err.Error(), 400)
		return
	}

	// Override root node name/purpose with form values
	tree.Name = name
	if about != "" {
		tree.Purpose = about
	}

	// Build the org
	err = buildOrgFromTree(uctx, form, tree, visibility)
	if err != nil {
		http.Error(w, "failed to create organisation: "+err.Error(), 500)
		return
	}

	writeJSON(w, model.Node{Nameid: nameid})
}

//
// Format detection and dispatch
//

// detectSourceFormat guesses the source platform from sheet names.
func detectSourceFormat(sheets map[string][][]string) string {
	if _, ok := sheets["Circles & Roles"]; ok {
		return "holaspirit"
	}
	return ""
}

// parseByFormat dispatches to the appropriate adapter.
func parseByFormat(format string, sheets map[string][][]string) (*ImportNode, error) {
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

// buildOrgFromTree creates a Fractale organisation from an ImportNode tree.
func buildOrgFromTree(uctx *model.UserCtx, form model.OrgaForm, tree *ImportNode, visibility model.NodeVisibility) error {
	nameid := form.Nameid
	nidOwner := nameid + "##" + "@" + uctx.Username
	isPersonal := true
	var userCanJoin bool
	guestCanCreateTension := true
	mode := model.NodeModeCoordinated

	if visibility == model.NodeVisibilityPublic {
		userCanJoin = true
	}

	// Create root node with owner child
	nodeInput := model.AddNodeInput{
		Name:                  form.Name,
		Nameid:                nameid,
		Rootnameid:            nameid,
		About:                 form.About,
		Type:                  model.NodeTypeCircle,
		IsRoot:                true,
		IsPersonal:            &isPersonal,
		Watchers:              []*model.UserRef{{Username: &uctx.Username}},
		Visibility:            visibility,
		Mode:                  mode,
		Rights:                0,
		IsArchived:            false,
		UserCanJoin:           &userCanJoin,
		GuestCanCreateTension: &guestCanCreateTension,
		CreatedAt:             Now(),
		CreatedBy:             &model.UserRef{Username: &uctx.Username},
	}

	// Set Owner role
	owner := StructMap[model.NodeRef](nodeInput)
	t := model.NodeTypeRole
	rt := model.RoleTypeOwner
	n := string(rt)
	_root := false
	owner.Type = &t
	owner.Nameid = &nidOwner
	owner.Name = &n
	owner.RoleType = &rt
	owner.IsRoot = &_root
	owner.About = nil
	owner.Watchers = nil
	nodeInput.Children = []*model.NodeRef{&owner}

	// Create root node
	rootNodeID, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "node", nodeInput)
	if err != nil {
		return fmt.Errorf("creating root node: %w", err)
	}

	// Create root control tension
	mandate := buildMandateRef(tree)
	tensionInput := graph.MakeNewRootTension(nameid, nodeInput, form.About, mandate)
	tid, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "tension", tensionInput)
	if err != nil {
		return fmt.Errorf("creating root tension: %w", err)
	}

	// Link source tension
	bid := db.GetDB().GetLastBlobId(tid)
	if bid == nil {
		return fmt.Errorf("linking source: no blob found for tension %s", tid)
	}
	err = db.GetDB().LinkGovernedNode(tid, rootNodeID, *bid)
	if err != nil {
		return fmt.Errorf("linking root governance: %w", err)
	}

	// Add owner role to user
	err = db.GetDB().AddUserRole(uctx.Username, nidOwner)
	if err != nil {
		return fmt.Errorf("adding owner role: %w", err)
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
	nodeNameid := NameidEncoder(node.Name)
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
		CreatedAt:  Now(),
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
	now := Now()
	createdBy := model.UserRef{Username: &username}
	emitter := model.NodeRef{Nameid: &parentid}
	receiver := model.NodeRef{Nameid: &parentid}

	evt1 := model.TensionEventCreated
	evt2 := model.TensionEventBlobCreated
	evt3 := model.TensionEventBlobPushed
	blobType := model.BlobTypeOnNode
	isAnchor := nameid == rootnameid
	emptyMsg := ""

	// Build NodeFragmentRef for the blob
	nodeName := node.Name
	localNameid := NameidEncoder(node.Name)
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
		BlobType:   &blobType,
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
		// Only set mandate if at least one field is non-empty
		if m.Purpose != nil || m.Responsabilities != nil || m.Domains != nil || m.Policies != nil {
			input.Mandate = m
		}
	}
	return db.GetDB().Add(db.GetDB().GetRootUctx(), "roleExt", input)
}

// buildMandateRef creates a MandateRef from an ImportNode if it has mandate data.
// Empty fields are left nil rather than set to empty strings.
func buildMandateRef(node *ImportNode) *model.MandateRef {
	if node.Purpose == "" && node.Domains == "" && node.Policies == "" && node.Responsabilities == "" {
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
	return m
}
