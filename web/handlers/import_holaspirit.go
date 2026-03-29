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
	"sort"
	"strings"
	"time"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/tools"
)

// mustHTMLToMarkdown converts HTML to markdown, returning the input as-is on error.
func mustHTMLToMarkdown(s string) string {
	result, err := tools.HTMLToMarkdown(s)
	if err != nil {
		return s
	}
	return result
}

// holaCircleRow holds the parsed fields from a "Circles & Roles" row.
type holaCircleRow struct {
	circleID   string
	circleName string
	roleID     string
	roleName   string
	isCircle   bool
	purpose    string
	domains    string
	accountab  string // accountabilities -> maps to Mandate.responsabilities
	strategy   string
	template   bool
	created    time.Time
}

// holaPolicyRow holds the parsed fields from a "Policies" row.
type holaPolicyRow struct {
	circleID   string
	circleName string
	roleID     string
	roleName   string
	policyName string
	policyDesc string
}

// circleInfo pairs an ImportNode with its parent circle ID for tree assembly.
type circleInfo struct {
	node     *ImportNode
	parentID string
}

// parseHolaSpirit parses HolaSpirit export sheets into an ImportNode tree.
func parseHolaSpirit(sheets map[string][][]string) (*ImportNode, error) {
	crRows, err := parseHolaCirclesSheet(sheets)
	if err != nil {
		return nil, err
	}
	policies, err := parseHolaPoliciesSheet(sheets)
	if err != nil {
		return nil, err
	}
	return buildHolaTree(crRows, policies)
}

// parseHolaCirclesSheet parses the "Circles & Roles" sheet.
func parseHolaCirclesSheet(sheets map[string][][]string) ([]holaCircleRow, error) {
	rows, ok := sheets["Circles & Roles"]
	if !ok {
		return nil, fmt.Errorf("missing sheet 'Circles & Roles'")
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet 'Circles & Roles' has no data rows")
	}

	header := rows[0]
	colIdx := mapColumns(header)
	for _, r := range []string{"Circle ID", "Circle", "Role ID", "Role", "IsCircle", "Purpose"} {
		if _, ok := colIdx[r]; !ok {
			return nil, fmt.Errorf("missing column %q in 'Circles & Roles'", r)
		}
	}

	var result []holaCircleRow
	for _, row := range rows[1:] {
		r := holaCircleRow{
			circleID:   getCol(row, colIdx, "Circle ID"),
			circleName: getCol(row, colIdx, "Circle"),
			roleID:     getCol(row, colIdx, "Role ID"),
			roleName:   getCol(row, colIdx, "Role"),
			isCircle:   strings.EqualFold(getCol(row, colIdx, "IsCircle"), "TRUE"),
			purpose:    mustHTMLToMarkdown(getCol(row, colIdx, "Purpose")),
			domains:    mustHTMLToMarkdown(getCol(row, colIdx, "Domains")),
			accountab:  mustHTMLToMarkdown(getCol(row, colIdx, "Accountabilities")),
			template:   strings.EqualFold(getCol(row, colIdx, "Template"), "TRUE"),
		}
		// Strategy column (may be named "Stratégie" or "Strategy")
		strat := getCol(row, colIdx, "Stratégie")
		if strat == "" {
			strat = getCol(row, colIdx, "Strategy")
		}
		r.strategy = mustHTMLToMarkdown(strat)
		// Parse Created timestamp
		if cs := getCol(row, colIdx, "Created"); cs != "" {
			if t, err := time.Parse("2006-01-02 15:04:05.999999", cs); err == nil {
				r.created = t
			}
		}
		result = append(result, r)
	}

	// Sort by creation time (stable to preserve original order for ties)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].created.IsZero() && result[j].created.IsZero() {
			return false
		}
		if result[i].created.IsZero() {
			return false
		}
		if result[j].created.IsZero() {
			return true
		}
		return result[i].created.Before(result[j].created)
	})

	return result, nil
}

// parseHolaPoliciesSheet parses the "Policies" sheet.
func parseHolaPoliciesSheet(sheets map[string][][]string) ([]holaPolicyRow, error) {
	rows, ok := sheets["Policies"]
	if !ok {
		return nil, nil
	}
	if len(rows) < 2 {
		return nil, nil
	}

	header := rows[0]
	colIdx := mapColumns(header)

	var result []holaPolicyRow
	for _, row := range rows[1:] {
		r := holaPolicyRow{
			circleID:   getCol(row, colIdx, "Circle ID"),
			circleName: getCol(row, colIdx, "Circle"),
			roleID:     getCol(row, colIdx, "Role ID"),
			roleName:   getCol(row, colIdx, "Role"),
			policyName: getCol(row, colIdx, "Policy"),
			policyDesc: mustHTMLToMarkdown(getCol(row, colIdx, "Description")),
		}
		result = append(result, r)
	}
	return result, nil
}

// buildHolaTree assembles the ImportNode tree from parsed rows and policies.
func buildHolaTree(crRows []holaCircleRow, policies []holaPolicyRow) (*ImportNode, error) {
	if len(crRows) == 0 {
		return nil, fmt.Errorf("no data rows found")
	}

	// In HolaSpirit exports, each circle has two different IDs:
	//   - roleID: used in its own isCircle=TRUE row declaration
	//   - circleID: used when referenced as parent in child rows
	// We build a mapping between them via circle name matching.

	// Collect circleID -> circleName from all rows (parent references)
	circleIDToName := make(map[string]string)
	for _, r := range crRows {
		if r.circleID != "" {
			circleIDToName[r.circleID] = r.circleName
		}
	}

	// First pass: create circle nodes, keyed by roleID
	circlesByRoleID := make(map[string]*circleInfo) // roleID -> info
	// NOTE: circleNameToRoleID assumes unique circle names within the org.
	// If two circles share the same name, the second overwrites the first and
	// the first circle's children may be misattached. To fix, use a multimap
	// (map[string][]string) and disambiguate via parentID chain or row order.
	circleNameToRoleID := make(map[string]string) // roleName -> roleID (for name-based matching)
	var rootRoleID string

	for _, r := range crRows {
		if !r.isCircle {
			continue
		}
		cid := r.roleID
		if _, exists := circlesByRoleID[cid]; exists {
			continue
		}

		purpose := r.purpose
		if r.strategy != "" {
			purpose += "\n\n### Strategy\n\n" + r.strategy
		}

		node := &ImportNode{
			Name:             r.roleName,
			Purpose:          purpose,
			Domains:          r.domains,
			Responsabilities: r.accountab,
			Type:             model.NodeTypeCircle,
		}
		circlesByRoleID[cid] = &circleInfo{node: node, parentID: r.circleID}
		circleNameToRoleID[r.roleName] = cid

		// Root circle: the one with no parent (empty circleID)
		if r.circleID == "" {
			rootRoleID = cid
		}
	}

	// Build circleID -> roleID mapping via name matching
	// This maps the parent reference ID to the circle's own roleID
	circleIDToRoleID := make(map[string]string)
	for circleID, name := range circleIDToName {
		if roleID, ok := circleNameToRoleID[name]; ok {
			circleIDToRoleID[circleID] = roleID
		}
	}

	// Resolve root
	if rootRoleID == "" {
		// Fallback: find circles whose parent circleID doesn't map to any known circle
		for roleID, ci := range circlesByRoleID {
			if ci.parentID == "" || circleIDToRoleID[ci.parentID] == "" {
				rootRoleID = roleID
				break
			}
		}
	}
	rootInfo := circlesByRoleID[rootRoleID]
	if rootInfo == nil {
		return nil, fmt.Errorf("could not determine root circle")
	}

	// Build a unified lookup: accepts both circleID (parent ref) and roleID
	// Returns the circleInfo for the referenced circle
	lookupCircle := func(id string) (*circleInfo, bool) {
		// Try as roleID first
		if ci, ok := circlesByRoleID[id]; ok {
			return ci, true
		}
		// Try as circleID (parent reference)
		if roleID, ok := circleIDToRoleID[id]; ok {
			if ci, ok := circlesByRoleID[roleID]; ok {
				return ci, true
			}
		}
		return nil, false
	}

	// Collect template-marked roles and role occurrences for deduplication
	type roleSignature struct {
		purpose   string
		domains   string
		accountab string
	}
	templateRoles := make(map[string]*ImportRoleExt) // roleName -> template (from Template=TRUE column)
	roleOccurrences := make(map[string][]roleSignature)

	// Second pass: create role nodes and attach to parent circles
	for _, r := range crRows {
		if r.isCircle {
			continue
		}
		parentCircle, ok := lookupCircle(r.circleID)
		if !ok {
			continue
		}

		purpose := r.purpose
		if r.strategy != "" {
			purpose += "\n\n### Strategy\n\n" + r.strategy
		}

		roleOccurrences[r.roleName] = append(roleOccurrences[r.roleName], roleSignature{
			purpose:   purpose,
			domains:   r.domains,
			accountab: r.accountab,
		})

		rt := mapHolaRoleType(r.roleName)
		roleNode := &ImportNode{
			Name:             r.roleName,
			Purpose:          purpose,
			Domains:          r.domains,
			Responsabilities: r.accountab,
			Type:             model.NodeTypeRole,
			RoleType:         &rt,
		}
		parentCircle.node.Children = append(parentCircle.node.Children, roleNode)

		// Track roles marked as templates in HolaSpirit
		if r.template {
			if _, exists := templateRoles[r.roleName]; !exists {
				templateRoles[r.roleName] = &ImportRoleExt{
					Name:     r.roleName,
					RoleType: rt,
				}
				// Set mandate only if there's actual content
				if purpose != "" || r.domains != "" || r.accountab != "" {
					templateRoles[r.roleName].Mandate = &ImportMandate{
						Purpose:          purpose,
						Domains:          r.domains,
						Responsabilities: r.accountab,
					}
				}
			}
		}
	}

	// Also build RoleExt templates for roles that appear multiple times
	// with identical content (deduplication), if not already a template
	for roleName, sigs := range roleOccurrences {
		if _, exists := templateRoles[roleName]; exists {
			continue
		}
		if len(sigs) < 2 {
			continue
		}
		first := sigs[0]
		allSame := true
		for _, s := range sigs[1:] {
			if s.purpose != first.purpose || s.domains != first.domains || s.accountab != first.accountab {
				allSame = false
				break
			}
		}
		if !allSame {
			continue
		}
		rt := mapHolaRoleType(roleName)
		templateRoles[roleName] = &ImportRoleExt{
			Name:     roleName,
			RoleType: rt,
		}
		if first.purpose != "" || first.domains != "" || first.accountab != "" {
			templateRoles[roleName].Mandate = &ImportMandate{
				Purpose:          first.purpose,
				Domains:          first.domains,
				Responsabilities: first.accountab,
			}
		}
	}

	// Mark role nodes that have a RoleExt template
	for _, ci := range circlesByRoleID {
		for _, child := range ci.node.Children {
			if child.Type == model.NodeTypeRole {
				if _, ok := templateRoles[child.Name]; ok {
					child.RoleExtRef = child.Name
				}
			}
		}
	}

	// Build circle hierarchy using the circleID mapping
	for roleID, ci := range circlesByRoleID {
		if roleID == rootRoleID {
			continue
		}
		parent, ok := lookupCircle(ci.parentID)
		if !ok {
			// Orphan circle: attach to root
			parent = rootInfo
		}
		parent.node.Children = append(parent.node.Children, ci.node)
	}

	// Merge policies (use lookupCircle for circleID resolution)
	mergePoliciesWithLookup(circlesByRoleID, circleIDToRoleID, policies)

	// Attach RoleExt templates to root
	for _, re := range templateRoles {
		rootInfo.node.RoleExtTemplates = append(rootInfo.node.RoleExtTemplates, re)
	}

	return rootInfo.node, nil
}

// mergePoliciesWithLookup merges policy rows into the corresponding circle or role nodes.
// It uses the circleIDToRoleID mapping to resolve HolaSpirit circleID references.
func mergePoliciesWithLookup(circlesByRoleID map[string]*circleInfo, circleIDToRoleID map[string]string, policies []holaPolicyRow) {
	for _, p := range policies {
		// Resolve circleID to the circle's roleID
		roleID := circleIDToRoleID[p.circleID]
		ci, ok := circlesByRoleID[roleID]
		if !ok {
			// Try direct lookup as fallback
			ci, ok = circlesByRoleID[p.circleID]
			if !ok {
				continue
			}
		}

		entry := fmt.Sprintf("- **%s**", p.policyName)
		if p.policyDesc != "" {
			indented := indentText(p.policyDesc, "  ")
			entry += "\n" + indented
		}

		if p.roleID == "" {
			// Circle-level policy
			if ci.node.Policies != "" {
				ci.node.Policies += "\n"
			}
			ci.node.Policies += entry
		} else {
			// Role-level policy: find the role in the circle's children
			for _, child := range ci.node.Children {
				if child.Type == model.NodeTypeRole && child.Name == p.roleName {
					if child.Policies != "" {
						child.Policies += "\n"
					}
					child.Policies += entry
					break
				}
			}
		}
	}
}

// indentText adds a prefix to each line of text.
func indentText(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// mapHolaRoleType maps HolaSpirit role names to Fractale RoleType.
func mapHolaRoleType(roleName string) model.RoleType {
	lower := strings.ToLower(roleName)
	switch {
	case strings.Contains(lower, "leader") || strings.Contains(lower, "lead") ||
		lower == "coordinateur" || lower == "coordinatrice" ||
		strings.Contains(lower, "1er lien") || strings.Contains(lower, "premier lien") ||
		strings.Contains(lower, "facilitateur") || strings.Contains(lower, "facilitatrice"):
		return model.RoleTypeCoordinator
	default:
		return model.RoleTypePeer
	}
}

// mapColumns builds a column name -> index map from a header row.
func mapColumns(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

// getCol safely extracts a column value from a row.
func getCol(row []string, colIdx map[string]int, colName string) string {
	idx, ok := colIdx[colName]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
