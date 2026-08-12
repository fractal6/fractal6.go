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

import "testing"

func TestCountInlineImageCandidates(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want int
	}{
		{"empty", "", 0},
		{"single paste", "![](paste-1.png)", 1},
		{"two pastes", "intro ![](a.png) middle ![](b.jpg) end", 2},
		{"with alt text", "![screenshot](paste-1.png)", 1},
		{"absolute URL skipped", "![](https://example.com/a.png)", 0},
		{"path skipped", "![](/file/0xabc)", 0},
		{"nested path skipped", "![](dir/x.png)", 0},
		{"data URI skipped", "![](data:image/png;base64,AAA)", 0},
		{"inside code fence skipped", "```\n![](paste.png)\n```", 0},
		{"inside backticks skipped", "see `![](paste.png)`", 0},
		{"mixed paste + code-mention", "![](paste.png) and `![](other.png)`", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CountInlineImageCandidates(c.msg); got != c.want {
				t.Errorf("CountInlineImageCandidates(%q) = %d, want %d", c.msg, got, c.want)
			}
		})
	}
}
