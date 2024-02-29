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

package codec

import (
	"fmt"

	"fractale/fractal6.go/graph/model"
)

// Action Type enum
type actionType string

const (
	NewAction     actionType = "new"
	EditAction    actionType = "edit"
	ArchiveAction actionType = "archive"
)

// Doc Type enum
type docType string

const (
	NodeDoc docType = "node"
	MdDoc   docType = "md"
)

// Tension Action information
type TensionCharac struct {
	ActionType actionType
	DocType    docType
}

// Create new TensionCharac from a TensionAction type.
func (TensionCharac) New(action model.TensionAction) (*TensionCharac, error) {
	var l []string
	var err error

	switch action {
	case model.TensionActionNewRole:
		l = append(l, "new", "node")
	case model.TensionActionNewCircle:
		l = append(l, "new", "node")
	case model.TensionActionNewMd:
		l = append(l, "new", "md")
	case model.TensionActionEditRole:
		l = append(l, "edit", "node")
	case model.TensionActionEditCircle:
		l = append(l, "edit", "node")
	case model.TensionActionEditMd:
		l = append(l, "edit", "md")
	case model.TensionActionArchivedRole:
		l = append(l, "archive", "node")
	case model.TensionActionArchivedCircle:
		l = append(l, "archive", "node")
	case model.TensionActionArchivedMd:
		l = append(l, "archive", "md")
	default:
		err = fmt.Errorf("Tension Action type unknown: " + string(action))
	}

	tc := &TensionCharac{
		ActionType: actionType(l[0]),
		DocType:    docType(l[1]),
	}
	return tc, err
}

func (tc TensionCharac) EditAction(t *model.NodeType) model.TensionAction {
	var a model.TensionAction
	switch tc.DocType {
	case MdDoc:
		a = model.TensionActionEditMd
	case NodeDoc:
		switch *t {
		case model.NodeTypeRole:
			a = model.TensionActionEditRole
		case model.NodeTypeCircle:
			a = model.TensionActionEditCircle
		}
	}
	return a
}

func (tc TensionCharac) ArchiveAction(t *model.NodeType) model.TensionAction {
	var a model.TensionAction
	switch tc.DocType {
	case MdDoc:
		a = model.TensionActionArchivedMd
	case NodeDoc:
		switch *t {
		case model.NodeTypeRole:
			a = model.TensionActionArchivedRole
		case model.NodeTypeCircle:
			a = model.TensionActionArchivedCircle
		}
	}
	return a
}
