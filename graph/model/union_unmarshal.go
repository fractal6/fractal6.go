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

package model

import (
	"encoding/json"
	"fmt"
)

// decodeEventKind reads __typename from raw JSON and returns the concrete EventKind.
func decodeEventKind(raw json.RawMessage) (EventKind, error) {
	var probe struct {
		Typename string `json:"__typename"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	switch probe.Typename {
	case "Event":
		var v Event
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "Contract":
		var v Contract
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "Notif":
		var v Notif
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return &v, nil
	default:
		return nil, fmt.Errorf("unknown EventKind __typename: %q", probe.Typename)
	}
}

// decodeCardKind reads __typename from raw JSON and returns the concrete CardKind.
func decodeCardKind(raw json.RawMessage) (CardKind, error) {
	var probe struct {
		Typename string `json:"__typename"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	switch probe.Typename {
	case "Tension":
		var v Tension
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return &v, nil
	case "ProjectDraft":
		var v ProjectDraft
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return &v, nil
	default:
		return nil, fmt.Errorf("unknown CardKind __typename: %q", probe.Typename)
	}
}

// UnmarshalJSON implements custom JSON unmarshaling for UserEvent to handle
// the EventKind union type in the event field.
func (ue *UserEvent) UnmarshalJSON(data []byte) error {
	// Alias to avoid infinite recursion
	type Alias UserEvent
	var raw struct {
		Alias
		Event []json.RawMessage `json:"event,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*ue = UserEvent(raw.Alias)
	ue.Event = nil
	for _, rawItem := range raw.Event {
		kind, err := decodeEventKind(rawItem)
		if err != nil {
			return err
		}
		if kind != nil {
			ue.Event = append(ue.Event, kind)
		}
	}
	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for ProjectCard to handle
// the CardKind union type in the card field.
func (pc *ProjectCard) UnmarshalJSON(data []byte) error {
	// Alias to avoid infinite recursion
	type Alias ProjectCard
	var raw struct {
		Alias
		Card json.RawMessage `json:"card,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*pc = ProjectCard(raw.Alias)
	pc.Card = nil
	if len(raw.Card) > 0 && string(raw.Card) != "null" {
		kind, err := decodeCardKind(raw.Card)
		if err != nil {
			return err
		}
		pc.Card = kind
	}
	return nil
}
