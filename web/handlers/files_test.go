/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 */

package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"fractale/fractal6.go/db"
)

func TestSafeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hello.png", "hello.png"},
		{"../../etc/passwd", "passwd"},      // path traversal stripped
		{`C:\Windows\evil.exe`, "evil.exe"}, // backslash separators normalised
		{"name with space & punct.txt", "name with space & punct.txt"},
		{"with\x00nul\x07bell.bin", "withnulbell.bin"}, // control chars stripped
		{"", "file"}, // empty falls back to "file"
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

func TestResolveAnchor_UidValidation(t *testing.T) {
	// Malformed tid/cid must be rejected with 400 before reaching a DQL uid()
	// root. Comma lists are the dangerous case: uid() accepts them, so
	// "0x1,0x2" would otherwise widen the query to every listed node.
	cases := []struct {
		name, tid, cid string
		wantStatus     int
		wantKind       db.FileKind
	}{
		{"valid", "0x1", "0x2", 0, db.KindComment},
		{"comma-list tid", "0x1,0x2", "0x3", http.StatusBadRequest, ""},
		{"comma-list cid", "0x1", "0x2,0x3", http.StatusBadRequest, ""},
		{"non-hex tid", "abc", "0x2", http.StatusBadRequest, ""},
		{"quote breakout", `0x1"){q(func:uid(0x2`, "0x3", http.StatusBadRequest, ""},
		{"tid without cid", "0x1", "", http.StatusBadRequest, ""},
		{"cid without tid", "", "0x1", http.StatusBadRequest, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/", strings.NewReader(
				"tid="+url.QueryEscape(c.tid)+"&cid="+url.QueryEscape(c.cid)))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			anchor, status, err := resolveAnchor(req)
			if status != c.wantStatus {
				t.Errorf("status = %d, want %d (err=%v)", status, c.wantStatus, err)
			}
			if anchor.Kind != c.wantKind {
				t.Errorf("kind = %q, want %q", anchor.Kind, c.wantKind)
			}
		})
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
	got := commentKeyPrefix("test-org", "0xtid", "0xabc123")
	want := "orgas/test-org/tensions/0xtid/0xabc123/"
	if got != want {
		t.Errorf("commentKeyPrefix = %q, want %q", got, want)
	}
}

func TestContentDispositionFor(t *testing.T) {
	cases := []struct {
		ct, name, want string
	}{
		// inline-safe types
		{"image/png", "p.png", "inline; filename*=UTF-8''p.png"},
		{"image/jpeg", "x.jpg", "inline; filename*=UTF-8''x.jpg"},
		{"application/pdf", "doc.pdf", "inline; filename*=UTF-8''doc.pdf"},
		// SVG must NOT be inline (XSS via embedded <script>)
		{"image/svg+xml", "evil.svg", "attachment; filename*=UTF-8''evil.svg"},
		// HTML must NOT be inline
		{"text/html", "page.html", "attachment; filename*=UTF-8''page.html"},
		// unknown defaults to attachment
		{"application/octet-stream", "blob.bin", "attachment; filename*=UTF-8''blob.bin"},
		// charset suffix should not break the lookup
		{"text/plain; charset=utf-8", "n.txt", "inline; filename*=UTF-8''n.txt"},
		// case-insensitive content type
		{"IMAGE/PNG", "x.png", "inline; filename*=UTF-8''x.png"},
		// non-ASCII filename gets percent-encoded
		{"image/png", "café.png", "inline; filename*=UTF-8''caf%C3%A9.png"},
		// missing filename emits just the verb
		{"image/png", "", "inline"},
		{"text/html", "", "attachment"},
	}
	for _, c := range cases {
		got := contentDispositionFor(c.ct, c.name)
		if got != c.want {
			t.Errorf("contentDispositionFor(%q, %q) = %q, want %q", c.ct, c.name, got, c.want)
		}
	}
}
