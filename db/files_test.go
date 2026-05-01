/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 */

package db

import "testing"

func TestEscapeNQuad(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{`with "quotes"`, `with \"quotes\"`},
		{`back\slash`, `back\\slash`},
		{"tab\there", `tab\there`},
		{"line\nbreak", `line\nbreak`},
		{"cr\rlf", `cr\rlf`},
		{`mix \" \\ both`, `mix \\\" \\\\ both`},
	}
	for _, c := range cases {
		got := escapeNQuad(c.in)
		if got != c.want {
			t.Errorf("escapeNQuad(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFirstChild(t *testing.T) {
	// map[string]any branch
	parent := map[string]any{"k": map[string]any{"x": "y"}}
	if c, ok := firstChild(parent, "k"); !ok || c["x"] != "y" {
		t.Errorf("firstChild map: got %v ok=%v", c, ok)
	}
	// []any branch
	parent = map[string]any{"k": []any{map[string]any{"x": "y"}}}
	if c, ok := firstChild(parent, "k"); !ok || c["x"] != "y" {
		t.Errorf("firstChild slice: got %v ok=%v", c, ok)
	}
	// missing key
	if _, ok := firstChild(parent, "missing"); ok {
		t.Error("firstChild missing should return ok=false")
	}
	// empty slice
	parent = map[string]any{"k": []any{}}
	if _, ok := firstChild(parent, "k"); ok {
		t.Error("firstChild empty slice should return ok=false")
	}
}
