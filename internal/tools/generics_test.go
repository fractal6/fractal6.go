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

package tools_test

import (
	"testing"

	. "fractale/fractal6.go/internal/tools"
)

func TestDerefSlice(t *testing.T) {
	a, b := "x", "y"
	input := []*string{&a, &b}
	got := DerefSlice(input)
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("DerefSlice: got %v", got)
	}
}

func TestInterfaceToSlice_Strings(t *testing.T) {
	input := any([]any{"a", "b"})
	got := InterfaceToSlice[string](input)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("InterfaceToSlice strings: got %v", got)
	}
}

func TestInterfaceToSlice_Nil(t *testing.T) {
	got := InterfaceToSlice[string](nil)
	if got != nil {
		t.Errorf("InterfaceToSlice nil: expected nil, got %v", got)
	}
}
