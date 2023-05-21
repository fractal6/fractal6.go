/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
)

func TestFindUsername(t *testing.T) {
	testcases := []struct {
		input string
		want  []string
	}{
		{"me", []string{}},
		{"@me", []string{"me"}},
		{"@me.", []string{"me"}},
		{"me @me me", []string{"me"}},
		{"@me @me_me", []string{"me", "me_me"}},
		{"(@me)", []string{"me"}},
		{"[@me]", []string{}},
	}

	for _, test := range testcases {
		got := FindUsernames(test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %s. Got %s (len %d).",
				test.input, test.want, got, len(got))
		}
	}
}

func TestFindTension(t *testing.T) {
	testcases := []struct {
		input string
		want  []string
	}{
		{"123", []string{}},
		{"0x0123f", []string{"0x0123f"}},
		{"0x0123f.", []string{"0x0123f"}},
		{"0x0123fg", []string{}},
		{"me 0x123 me", []string{"0x123"}},
		{"0x123 0xabc", []string{"0x123", "0xabc"}},
		{"(0x123)", []string{"0x123"}},
		{"[0x123]", []string{}},
	}

	for _, test := range testcases {
		got := FindTensions(test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("For p = %s, want %s. Got %s (len %d).",
				test.input, test.want, got, len(got))
		}
	}
}
