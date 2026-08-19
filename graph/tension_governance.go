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

	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
)

type governanceOperation uint8

const (
	governancePublish governanceOperation = iota
	governanceArchive
	governanceUnarchive
	// governanceUpdate covers authority, visibility, membership and move: they only need a linked node.
	governanceUpdate
)

type governanceSubject struct {
	blob   *model.Blob
	node   *model.Node
	nameid string
	create bool
}

// isNodeArchived reads the archive lifecycle flag: a root org uses the lightweight isRootArchived flag (no recursion).
func isNodeArchived(node *model.Node) bool {
	if codec.IsRoot(node.Nameid) {
		return node.IsRootArchived != nil && *node.IsRootArchived
	}
	return node.IsArchived
}

// resolveGovernanceSubject is the single shape and lifecycle gate for Node governance events.
func resolveGovernanceSubject(tension *model.Tension, operation governanceOperation) (*governanceSubject, error) {
	if tension == nil {
		return nil, fmt.Errorf("governance tension not found")
	}
	if tension.Receiver == nil {
		return nil, fmt.Errorf("governance tension receiver is required")
	}
	blob := GetBlob(tension)
	if blob == nil {
		return nil, fmt.Errorf("governance event requires a blob")
	}
	if blob.Node == nil {
		return nil, fmt.Errorf("governance blob must contain a node fragment")
	}
	fragment := blob.Node
	if fragment.Type == nil || !fragment.Type.IsValid() {
		return nil, fmt.Errorf("governance node fragment requires a valid type")
	}
	if fragment.Name == nil || *fragment.Name == "" {
		return nil, fmt.Errorf("governance node fragment requires a name")
	}
	if err := auth.ValidateName(*fragment.Name); err != nil {
		return nil, err
	}
	if fragment.RoleType != nil && codec.IsMembershipRoleType(*fragment.RoleType) {
		return nil, fmt.Errorf("Membership roles are protected and cannot be created like this.")
	}

	subject := &governanceSubject{blob: blob, node: tension.GovernedNode}
	if operation == governancePublish && subject.node == nil {
		if fragment.Nameid == nil || *fragment.Nameid == "" {
			return nil, fmt.Errorf("new governance node fragment requires a nameid")
		}
		if *fragment.Type == model.NodeTypeRole && fragment.RoleType == nil {
			return nil, fmt.Errorf("new role fragment requires a role_type")
		}
		_, nameid, err := codec.NodeIdCodec(tension.Receiver.Nameid, *fragment.Nameid, *fragment.Type)
		if err != nil {
			return nil, err
		}
		subject.nameid = nameid
		subject.create = true
	} else {
		if subject.node == nil {
			return nil, fmt.Errorf("governance event requires a governed node")
		}
		if subject.node.ID == "" || subject.node.Nameid == "" || !subject.node.Type.IsValid() {
			return nil, fmt.Errorf("governed node identity and type are required")
		}
		if subject.node.Type != *fragment.Type {
			return nil, fmt.Errorf("node fragment type %q does not match governed node type %q", *fragment.Type, subject.node.Type)
		}
		subject.nameid = subject.node.Nameid

		switch operation {
		case governancePublish:
			if subject.node.IsArchived {
				return nil, fmt.Errorf("cannot publish an archived node")
			}
		case governanceArchive:
			if isNodeArchived(subject.node) {
				return nil, fmt.Errorf("governed node is already archived")
			}
		case governanceUnarchive:
			if !isNodeArchived(subject.node) {
				return nil, fmt.Errorf("governed node is not archived")
			}
		}
	}

	rootnameid, err := codec.Nid2rootid(subject.nameid)
	if err != nil {
		return nil, err
	}
	if err := auth.ValidateNameid(subject.nameid, rootnameid); err != nil {
		return nil, err
	}

	return subject, nil
}
