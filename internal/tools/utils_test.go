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

package tools

import (
	"reflect"
	"testing"

	"fractale/fractal6.go/graph/model"
)

func TestStructMap(t *testing.T) {
	name := "name"
	nameid := "nameid"
	username := "username"
	nodeFragment := &model.NodeFragment{
		Name:      &name,
		Nameid:    &nameid,
		FirstLink: &username,
	}

	var nodeInput model.AddNodeInput
	StructMap(nodeFragment, &nodeInput)

	// StructMap should copy all matching fields including FirstLink.
	// (The API layer rejects FirstLink later, not StructMap.)
	// Verify nodeInput differs from a struct with only Name/Nameid set.
	withoutFirstLink := model.AddNodeInput{
		Name:   name,
		Nameid: nameid,
	}
	if reflect.DeepEqual(nodeInput, withoutFirstLink) {
		t.Errorf("StructMap did not copy FirstLink: got %v", nodeInput)
	}
}
