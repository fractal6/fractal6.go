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

// Pure-logic tests for the markdown rewrite path. No DB / S3 — the integration
// tests in integration_files_test.go cover the wire-up.

package handlers

import "testing"

func TestRewriteMessageForFile_HappyPath(t *testing.T) {
	got, ok := rewriteMessageForFile("here it is: ![alt](paste-1.png) end", "paste-1.png", "0xfid")
	if !ok {
		t.Fatal("expected match")
	}
	want := "here it is: ![alt](/file/0xfid) end"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewriteMessageForFile_NoMatchOnPathToken(t *testing.T) {
	// URL is a path → not a bare-token match.
	_, ok := rewriteMessageForFile("![](some/paste-1.png)", "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match: URL contains a path separator")
	}
}

func TestRewriteMessageForFile_NoMatchOnAbsoluteUrl(t *testing.T) {
	_, ok := rewriteMessageForFile("![](https://example.com/paste-1.png)", "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match: URL has scheme")
	}
}

func TestRewriteMessageForFile_DifferentFilename(t *testing.T) {
	_, ok := rewriteMessageForFile("![](other.png)", "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match for different filename")
	}
}

func TestRewriteMessageForFile_FirstMatchWins(t *testing.T) {
	// First match wins; second instance is left as-is. Frontend is responsible
	// for unique filenames per paste.
	got, ok := rewriteMessageForFile("a ![](dup.png) b ![](dup.png) c", "dup.png", "0xfid")
	if !ok {
		t.Fatal("expected first match to win")
	}
	want := "a ![](/file/0xfid) b ![](dup.png) c"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewriteMessageForFile_FencedBlockSkipped(t *testing.T) {
	// Filename only mentioned inside a fenced ``` block must not match.
	msg := "before\n```\n![](paste-1.png)\n```\nafter"
	_, ok := rewriteMessageForFile(msg, "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match inside fenced code block")
	}
}

func TestRewriteMessageForFile_TildeFenceSkipped(t *testing.T) {
	msg := "before\n~~~\n![](paste-1.png)\n~~~\nafter"
	_, ok := rewriteMessageForFile(msg, "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match inside ~~~ fenced block")
	}
}

func TestRewriteMessageForFile_InlineCodeSkipped(t *testing.T) {
	msg := "see `![](paste-1.png)` for details"
	_, ok := rewriteMessageForFile(msg, "paste-1.png", "0xfid")
	if ok {
		t.Error("expected no match inside inline code span")
	}
}

func TestRewriteMessageForFile_OutsideFenceMatches(t *testing.T) {
	// Filename appears once inside a fence (skipped) and once outside (must match).
	msg := "```\n![](paste-1.png)\n```\nbody: ![](paste-1.png)"
	got, ok := rewriteMessageForFile(msg, "paste-1.png", "0xfid")
	if !ok {
		t.Fatal("expected match outside fence")
	}
	want := "```\n![](paste-1.png)\n```\nbody: ![](/file/0xfid)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewriteMessageForFile_EmptyInputs(t *testing.T) {
	if _, ok := rewriteMessageForFile("", "x", "y"); ok {
		t.Error("empty msg should not match")
	}
	if _, ok := rewriteMessageForFile("x", "", "y"); ok {
		t.Error("empty filename should not match")
	}
	if _, ok := rewriteMessageForFile("x", "y", ""); ok {
		t.Error("empty fid should not match")
	}
}

func TestMaskCodeRegions_UnclosedFenceMasksThroughEOF(t *testing.T) {
	in := "before\n```\n![](paste.png)"
	out := maskCodeRegions(in)
	// Anything after the opening fence should be spaces (newlines preserved).
	if out[:len("before\n")] != "before\n" {
		t.Errorf("prefix wrongly masked: %q", out)
	}
	for i := len("before\n"); i < len(out); i++ {
		if out[i] != ' ' && out[i] != '\n' {
			t.Errorf("unmasked byte at %d: %q (out: %q)", i, out[i], out)
			break
		}
	}
}
