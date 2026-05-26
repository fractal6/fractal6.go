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

package handlers

import (
	"strings"
	"testing"
)

func TestExtractCIDRefs_basic(t *testing.T) {
	msg := "hello ![img](cid:foo@bar.local) world"
	refs := extractCIDRefs(msg)
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1", len(refs))
	}
	r := refs[0]
	if r.token != "foo@bar.local" {
		t.Errorf("token = %q, want %q", r.token, "foo@bar.local")
	}
	if msg[r.tagStart:r.tagEnd] != "![img](cid:foo@bar.local)" {
		t.Errorf("tag slice = %q", msg[r.tagStart:r.tagEnd])
	}
	if msg[r.urlStart:r.urlEnd] != "cid:foo@bar.local" {
		t.Errorf("url slice = %q", msg[r.urlStart:r.urlEnd])
	}
}

func TestExtractCIDRefs_multiple(t *testing.T) {
	msg := "![a](cid:one) and ![b](cid:two)"
	refs := extractCIDRefs(msg)
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}
	if refs[0].token != "one" || refs[1].token != "two" {
		t.Errorf("tokens: %q %q", refs[0].token, refs[1].token)
	}
}

func TestExtractCIDRefs_codeFence(t *testing.T) {
	msg := "before\n```\n![ignored](cid:nope)\n```\nafter ![keep](cid:yes)"
	refs := extractCIDRefs(msg)
	if len(refs) != 1 {
		t.Fatalf("got %d refs, want 1 (fenced cid skipped)", len(refs))
	}
	if refs[0].token != "yes" {
		t.Errorf("token = %q, want %q", refs[0].token, "yes")
	}
}

func TestExtractCIDRefs_inlineBacktick(t *testing.T) {
	msg := "literal `![x](cid:skip)` vs real ![y](cid:keep)"
	refs := extractCIDRefs(msg)
	if len(refs) != 1 || refs[0].token != "keep" {
		t.Fatalf("got %+v", refs)
	}
}

func TestExtractCIDRefs_notImageTag(t *testing.T) {
	// `cid:` outside of an `![...](cid:...)` is ignored.
	msg := "see cid:bare and [link](cid:also)"
	refs := extractCIDRefs(msg)
	if len(refs) != 0 {
		t.Fatalf("got %d refs, want 0", len(refs))
	}
}

func TestResolveCIDs_quotedBack(t *testing.T) {
	refs := []cidRef{{token: "0xabc@fractale.co"}}
	pair, prefilled, orphans := resolveCIDs(refs, nil, "fractale.co")
	if len(pair) != 0 {
		t.Errorf("pair: %+v, want empty", pair)
	}
	if len(orphans) != 0 {
		t.Errorf("orphans: %+v, want empty", orphans)
	}
	if len(prefilled) != 1 || prefilled[0].kind != cidRewrite || prefilled[0].fid != "0xabc" {
		t.Errorf("prefilled: %+v", prefilled)
	}
}

func TestResolveCIDs_quotedBackWrongDomain(t *testing.T) {
	refs := []cidRef{{token: "0xabc@evil.example"}}
	pair, prefilled, orphans := resolveCIDs(refs, nil, "fractale.co")
	if len(pair) != 0 || len(prefilled) != 0 {
		t.Errorf("expected only orphan; pair=%v prefilled=%v", pair, prefilled)
	}
	if len(orphans) != 1 {
		t.Errorf("orphans = %v", orphans)
	}
}

func TestResolveCIDs_quotedBackNotFidShape(t *testing.T) {
	// Local-part isn't a 0x-hex uid → not a quoted-back; falls through to
	// filename matching, then orphan.
	refs := []cidRef{{token: "screenshot@mua.local"}}
	pair, prefilled, orphans := resolveCIDs(refs, nil, "mua.local")
	if len(prefilled) != 0 {
		t.Errorf("prefilled should be empty: %+v", prefilled)
	}
	if len(pair) != 0 || len(orphans) != 1 {
		t.Errorf("expected orphan; pair=%v orphans=%v", pair, orphans)
	}
}

func TestResolveCIDs_filenameEqualsToken(t *testing.T) {
	refs := []cidRef{{token: "logo.png"}}
	atts := []InboundAttachment{{Filename: "logo.png"}}
	pair, _, orphans := resolveCIDs(refs, atts, "")
	if len(orphans) != 0 {
		t.Errorf("orphans: %v", orphans)
	}
	if pair[0] != 0 {
		t.Errorf("pair[0] = %d, want 0", pair[0])
	}
}

func TestResolveCIDs_filenameEqualsLocalPart(t *testing.T) {
	refs := []cidRef{{token: "image001@mua.local"}}
	atts := []InboundAttachment{{Filename: "image001"}}
	pair, _, orphans := resolveCIDs(refs, atts, "")
	if len(orphans) != 0 || pair[0] != 0 {
		t.Errorf("pair=%v orphans=%v", pair, orphans)
	}
}

func TestResolveCIDs_filenameEqualsLocalPartDotExt(t *testing.T) {
	refs := []cidRef{{token: "image001@mua.local"}}
	atts := []InboundAttachment{{Filename: "image001.png"}}
	pair, _, orphans := resolveCIDs(refs, atts, "")
	if len(orphans) != 0 || pair[0] != 0 {
		t.Errorf("pair=%v orphans=%v", pair, orphans)
	}
}

func TestResolveCIDs_documentOrderFallback(t *testing.T) {
	// Two refs, two atts, no filename overlap → paired in order.
	refs := []cidRef{{token: "a@x"}, {token: "b@x"}}
	atts := []InboundAttachment{{Filename: "first.png"}, {Filename: "second.png"}}
	pair, _, orphans := resolveCIDs(refs, atts, "")
	if len(orphans) != 0 {
		t.Errorf("orphans: %v", orphans)
	}
	if pair[0] != 0 || pair[1] != 1 {
		t.Errorf("pair = %+v", pair)
	}
}

func TestResolveCIDs_orphanRef(t *testing.T) {
	refs := []cidRef{{token: "a"}, {token: "b"}}
	atts := []InboundAttachment{{Filename: "only.png"}}
	pair, _, orphans := resolveCIDs(refs, atts, "")
	if len(pair) != 1 {
		t.Errorf("pair = %+v, want 1 entry", pair)
	}
	if len(orphans) != 1 {
		t.Errorf("orphans = %+v, want 1", orphans)
	}
}

func TestResolveCIDs_mixed(t *testing.T) {
	// 1 quoted-back, 1 filename-match, 1 fallback, 1 orphan.
	refs := []cidRef{
		{token: "0x10@fractale.co"},
		{token: "logo.png"},
		{token: "weirdthing@mua"},
		{token: "nomatch"},
	}
	atts := []InboundAttachment{
		{Filename: "logo.png"},
		{Filename: "random.bin"},
	}
	pair, prefilled, orphans := resolveCIDs(refs, atts, "fractale.co")
	if len(prefilled) != 1 || prefilled[0].refIdx != 0 || prefilled[0].fid != "0x10" {
		t.Errorf("prefilled = %+v", prefilled)
	}
	if pair[1] != 0 {
		t.Errorf("filename match: pair[1] = %d, want 0", pair[1])
	}
	if pair[2] != 1 {
		t.Errorf("fallback match: pair[2] = %d, want 1", pair[2])
	}
	if len(orphans) != 1 || orphans[0] != 3 {
		t.Errorf("orphans = %v", orphans)
	}
}

func TestApplyCIDResolutions_rewrite(t *testing.T) {
	msg := "see ![x](cid:foo@bar) here"
	refs := extractCIDRefs(msg)
	out := applyCIDResolutions(msg, refs, []cidResolution{
		{refIdx: 0, kind: cidRewrite, fid: "0xdead"},
	})
	want := "see ![x](/file/0xdead) here"
	if out != want {
		t.Errorf("got  %q\nwant %q", out, want)
	}
}

func TestApplyCIDResolutions_drop(t *testing.T) {
	msg := "before ![x](cid:foo) after"
	refs := extractCIDRefs(msg)
	out := applyCIDResolutions(msg, refs, []cidResolution{
		{refIdx: 0, kind: cidDrop},
	})
	want := "before  after"
	if out != want {
		t.Errorf("got  %q\nwant %q", out, want)
	}
}

func TestApplyCIDResolutions_multiple(t *testing.T) {
	msg := "A ![a](cid:one) B ![b](cid:two) C ![c](cid:three) D"
	refs := extractCIDRefs(msg)
	if len(refs) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(refs))
	}
	out := applyCIDResolutions(msg, refs, []cidResolution{
		{refIdx: 0, kind: cidRewrite, fid: "0x1"},
		{refIdx: 1, kind: cidDrop},
		{refIdx: 2, kind: cidRewrite, fid: "0x3"},
	})
	if !strings.Contains(out, "/file/0x1") || !strings.Contains(out, "/file/0x3") {
		t.Errorf("missing rewrites: %q", out)
	}
	if strings.Contains(out, "cid:") {
		t.Errorf("cid token still present: %q", out)
	}
}

func TestApplyCIDResolutions_codeRegionsUntouched(t *testing.T) {
	msg := "real ![a](cid:foo) and code `![b](cid:bar)` end"
	refs := extractCIDRefs(msg)
	// Only the real ref is in refs; the code-region one is ignored.
	if len(refs) != 1 {
		t.Fatalf("refs = %+v", refs)
	}
	out := applyCIDResolutions(msg, refs, []cidResolution{
		{refIdx: 0, kind: cidRewrite, fid: "0xabc"},
	})
	if !strings.Contains(out, "`![b](cid:bar)`") {
		t.Errorf("code region modified: %q", out)
	}
	if !strings.Contains(out, "/file/0xabc") {
		t.Errorf("rewrite missing: %q", out)
	}
}
