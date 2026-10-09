/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 */

package handlers

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
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

func TestValidateAvatar(t *testing.T) {
	pngOf := func(w, h int) []byte {
		var b bytes.Buffer
		if err := png.Encode(&b, image.NewGray(image.Rect(0, 0, w, h))); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	// Lossless WebP header only (VP8L): a few bytes declaring w x h.
	webpOf := func(w, h int) []byte {
		var b bytes.Buffer
		b.WriteString("RIFF")
		_ = binary.Write(&b, binary.LittleEndian, uint32(4+8+6))
		b.WriteString("WEBPVP8L")
		_ = binary.Write(&b, binary.LittleEndian, uint32(6))
		b.WriteByte(0x2f)
		_ = binary.Write(&b, binary.LittleEndian, uint32(w-1)|uint32(h-1)<<14)
		b.WriteByte(0)
		return b.Bytes()
	}
	cases := []struct {
		name   string
		body   []byte
		size   int64 // 0 = len(body)
		status int
		ct     string
	}{
		{"png ok", pngOf(256, 256), 0, 0, "image/png"},
		{"webp ok", webpOf(256, 256), 0, 0, "image/webp"},
		{"max side ok", pngOf(1024, 1), 0, 0, "image/png"},
		{"too wide", pngOf(1025, 1), 0, http.StatusBadRequest, ""},
		{"webp pixel bomb", webpOf(16383, 16383), 0, http.StatusBadRequest, ""},
		{"html", []byte("<html><script>alert(1)</script></html>"), 0, http.StatusBadRequest, ""},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0, http.StatusBadRequest, ""},
		{"too large", pngOf(1, 1), 512*1024 + 1, http.StatusRequestEntityTooLarge, ""},
	}
	for _, c := range cases {
		size := c.size
		if size == 0 {
			size = int64(len(c.body))
		}
		r := bytes.NewReader(c.body)
		ct, status, err := validateAvatar(r, size)
		if status != c.status || ct != c.ct || (err == nil) != (c.status == 0) {
			t.Errorf("%s: got (%q, %d, %v), want (%q, %d)", c.name, ct, status, err, c.ct, c.status)
		}
		if pos, _ := r.Seek(0, 1); c.status != http.StatusRequestEntityTooLarge && pos != 0 {
			t.Errorf("%s: reader not rewound (pos %d)", c.name, pos)
		}
	}
}
