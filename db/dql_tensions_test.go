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

package db

import (
	"testing"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/tools"
)

func TestTensionListResponseDecodesDerivedGovernanceState(t *testing.T) {
	raw := []map[string]any{
		{
			"uid":           "0x1",
			"Tension.title": "Draft role",
			"Tension.blobs": []any{map[string]any{
				"Blob.node": map[string]any{"NodeFragment.type_": "Role"},
			}},
		},
		{
			"uid":           "0x2",
			"Tension.title": "Active circle",
			"Tension.governed_node": map[string]any{
				"uid": "0x20", "Node.nameid": "org#circle", "Node.type_": "Circle", "Node.isArchived": false,
			},
		},
		{
			"uid":           "0x3",
			"Tension.title": "Archived role",
			"Tension.governed_node": map[string]any{
				"uid": "0x30", "Node.nameid": "org##role", "Node.type_": "Role", "Node.isArchived": true,
			},
		},
	}

	tensions, err := tools.DecodeDql[[]model.TensionRef](raw)
	if err != nil {
		t.Fatalf("decoding tension list response: %v", err)
	}
	if len(tensions) != 3 {
		t.Fatalf("decoded %d tensions, want 3", len(tensions))
	}

	draft := tensions[0]
	if draft.GovernedNode != nil || len(draft.Blobs) != 1 || draft.Blobs[0].Node == nil ||
		draft.Blobs[0].Node.Type == nil || *draft.Blobs[0].Node.Type != model.NodeTypeRole {
		t.Errorf("draft derivation inputs missing: %+v", draft)
	}

	active := tensions[1].GovernedNode
	if active == nil || active.ID == nil || active.Nameid == nil || active.Type == nil || active.IsArchived == nil ||
		*active.Nameid != "org#circle" || *active.Type != model.NodeTypeCircle || *active.IsArchived {
		t.Errorf("active derivation inputs missing: %+v", active)
	}

	archived := tensions[2].GovernedNode
	if archived == nil || archived.ID == nil || archived.Nameid == nil || archived.Type == nil || archived.IsArchived == nil ||
		*archived.Nameid != "org##role" || *archived.Type != model.NodeTypeRole || !*archived.IsArchived {
		t.Errorf("archived derivation inputs missing: %+v", archived)
	}
}
