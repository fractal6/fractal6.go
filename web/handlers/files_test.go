/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 */

package handlers

import (
	"strings"
	"testing"
)

func TestSafeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hello.png", "hello.png"},
		{"../../etc/passwd", "passwd"},                      // path traversal stripped
		{`C:\Windows\evil.exe`, "evil.exe"},                 // backslash separators normalised
		{"name with space & punct.txt", "name with space & punct.txt"},
		{"with\x00nul\x07bell.bin", "withnulbell.bin"},      // control chars stripped
		{"", "file"},                                        // empty falls back to "file"
		{".", "file"},
		{"/", "file"},
		{strings.Repeat("a", 200) + ".jpg", strings.Repeat("a", 120)}, // length cap
	}
	for _, c := range cases {
		got := safeFilename(c.in)
		if got != c.want {
			t.Errorf("safeFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRandomIDFormat(t *testing.T) {
	// Two calls should yield different ids of the documented length.
	a, b := randomID(), randomID()
	if a == b {
		t.Fatalf("randomID() returned the same value twice: %q", a)
	}
	if len(a) != 16 {
		t.Errorf("randomID() length = %d, want 16", len(a))
	}
	for _, r := range a {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("randomID() = %q contains non-hex char %q", a, r)
			break
		}
	}
}

func TestCommentKeyPrefix(t *testing.T) {
	got := commentKeyPrefix("0xabc123")
	if got != "comments/0xabc123/" {
		t.Errorf("commentKeyPrefix = %q, want %q", got, "comments/0xabc123/")
	}
}
