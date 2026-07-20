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

package email

import (
	"strings"
	"testing"
)

func init() {
	// Tests run before init() in main.go reads viper; force a deterministic
	// DOMAIN so cid:/absolute-URL rewrites produce stable output.
	DOMAIN = "test.example"
}

func TestPartition_InlineVsPlain(t *testing.T) {
	files := []emailFile{
		{ID: "a", ContentType: "image/png", Embedded: true},
		{ID: "b", ContentType: "image/png", Embedded: false},
		{ID: "c", ContentType: "application/pdf", Embedded: true},
		{ID: "d", ContentType: "image/svg+xml", Embedded: true},
	}
	inline, plain := partition(files)
	if len(inline) != 1 || inline[0].ID != "a" {
		t.Errorf("inline = %v, want [a]", ids(inline))
	}
	if len(plain) != 3 {
		t.Errorf("plain = %v, want 3 entries (b, c, d)", ids(plain))
	}
}

func TestRewriteFileImgs_BasicSubstitution(t *testing.T) {
	in := `<p><img src="/file/0xabc" alt="paste"></p>`
	out := rewriteFileImgs(in, map[string]bool{"0xabc": true})
	want := "cid:0xabc@test.example"
	if !strings.Contains(out, want) {
		t.Errorf("output missing %q\ngot: %s", want, out)
	}
	if strings.Contains(out, `src="/file/0xabc"`) {
		t.Errorf("original /file/<id> URL still present\ngot: %s", out)
	}
}

func TestRewriteFileImgs_AbsolutisesUnknownIDs(t *testing.T) {
	in := `<img src="/file/known"><img src="/file/unknown">`
	out := rewriteFileImgs(in, map[string]bool{"known": true})
	if !strings.Contains(out, "cid:known@test.example") {
		t.Errorf("missing CID rewrite for known: %s", out)
	}
	if !strings.Contains(out, `https://test.example/file/unknown`) {
		t.Errorf("unknown id should get the absolute fallback: %s", out)
	}
}

func TestRewriteFileImgs_LeftoverGetsAbsolute(t *testing.T) {
	in := `<img src="/file/0xabc">`
	out := rewriteFileImgs(in, nil)
	want := `https://test.example/file/0xabc`
	if !strings.Contains(out, want) {
		t.Errorf("absolutise didn't apply\ngot: %s", out)
	}
}

func TestRenderAttachmentFooter_IncludesPlainFilesOnly(t *testing.T) {
	files := []emailFile{
		{ID: "f1", Filename: "doc.pdf", Size: 12_500},
		{ID: "f2", Filename: "spreadsheet.xlsx", Size: 2_000_000},
	}
	footer := renderAttachmentFooter(files)
	if footer == "" {
		t.Fatal("footer should not be empty")
	}
	for _, want := range []string{
		`href="https://test.example/file/f1"`,
		`href="https://test.example/file/f2"`,
		`doc.pdf`,
		`spreadsheet.xlsx`,
	} {
		if !strings.Contains(footer, want) {
			t.Errorf("missing %q in footer\ngot: %s", want, footer)
		}
	}
}

func TestRenderAttachmentFooter_EmptyOnNoFiles(t *testing.T) {
	if got := renderAttachmentFooter(nil); got != "" {
		t.Errorf("expected empty footer, got %q", got)
	}
}

func TestRenderAttachmentFooter_EscapesFilename(t *testing.T) {
	files := []emailFile{
		{ID: "f1", Filename: `<img src=x onerror=alert(1)>`, Size: 0},
	}
	footer := renderAttachmentFooter(files)
	if strings.Contains(footer, "<img src=x onerror=alert(1)>") {
		t.Errorf("filename not escaped:\n%s", footer)
	}
	if !strings.Contains(footer, "&lt;img") {
		t.Errorf("expected escaped angle bracket:\n%s", footer)
	}
}

func TestSanitiser_AllowsCIDImgSrc(t *testing.T) {
	in := `<img src="cid:fid@test.example" alt="paste">`
	out := sanitizer.Sanitize(in)
	if !strings.Contains(out, "cid:fid@test.example") {
		t.Errorf("bluemonday stripped cid: scheme\ngot: %s", out)
	}
}

func TestSanitiser_StripsJavascriptImgSrc(t *testing.T) {
	in := `<img src="javascript:alert(1)" alt="x">`
	out := sanitizer.Sanitize(in)
	if strings.Contains(out, "javascript:") {
		t.Errorf("javascript: scheme leaked through sanitiser\ngot: %s", out)
	}
}

func TestCIDForFile(t *testing.T) {
	if got := cidForFile("0xabc"); got != "0xabc@test.example" {
		t.Errorf("cidForFile = %q, want 0xabc@test.example", got)
	}
}

func TestInlineCandidate_RespectsTypeAndEmbedded(t *testing.T) {
	cases := []struct {
		name string
		f    emailFile
		want bool
	}{
		{"png embedded", emailFile{ContentType: "image/png", Embedded: true}, true},
		{"png not embedded", emailFile{ContentType: "image/png", Embedded: false}, false},
		{"jpeg with charset suffix", emailFile{ContentType: "image/jpeg; charset=utf-8", Embedded: true}, true},
		{"svg embedded — never inline", emailFile{ContentType: "image/svg+xml", Embedded: true}, false},
		{"pdf embedded — never inline", emailFile{ContentType: "application/pdf", Embedded: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.f.inlineCandidate(); got != c.want {
				t.Errorf("inlineCandidate() = %v, want %v", got, c.want)
			}
		})
	}
}

func ids(files []emailFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.ID
	}
	return out
}
