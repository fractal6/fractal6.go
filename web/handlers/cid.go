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

// Pure helpers for `cid:` rewriting in inbound email replies. Postal delivers
// attachments as a flat array without Content-ID/Content-Disposition, so the
// correlation between `<img src="cid:...">` references in the reply body and
// the attachment bytes is best-effort: filename heuristic + document-order
// fallback. A mis-pairing degrades to a plain paperclip — never a security
// issue, because /file/<id> re-authorises per request.
//
// See docs/file-storage.md "Inbound email replies" for the policy.
package handlers

import (
	"regexp"
	"strings"

	"fractale/fractal6.go/internal/tools"
)

// cidRef records a `![alt](cid:<token>)` occurrence inside a markdown message.
// All indices are byte offsets into the *original* message (not the masked
// copy used for matching), so applyCIDResolutions can splice without
// translating back.
type cidRef struct {
	token    string // bare token, e.g. "image001@mua.local"
	tagStart int    // index of '!' in '![alt](cid:...)'
	tagEnd   int    // index just after ')'
	urlStart int    // index of 'c' in "cid:..."
	urlEnd   int    // index just after the token (before ')')
}

type cidResolutionKind int

const (
	cidRewrite cidResolutionKind = iota
	cidDrop
)

// cidResolution carries a per-ref decision out of resolveCIDs / into
// applyCIDResolutions.
type cidResolution struct {
	refIdx int
	kind   cidResolutionKind
	fid    string // populated when kind==cidRewrite
}

// cidImgRe matches `![alt](cid:<token>)`. The token group captures everything
// up to whitespace or a closing paren. Composed against the *masked* copy of
// the message so code regions don't match.
var cidImgRe = regexp.MustCompile(`!\[[^\]]*\]\(cid:([^)\s]+)\)`)

// fidLikeRe matches a Dgraph-uid-shaped local-part. Used to recognise a
// "quoted-back" cid token of form `<fid>@<ourDomain>` — i.e. the original
// outbound notification's Content-ID surviving the reply quote.
var fidLikeRe = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)

// extractCIDRefs walks the message and returns every `![alt](cid:<token>)`
// occurrence outside code regions, in document order. Indices are anchored
// against the original message.
func extractCIDRefs(msg string) []cidRef {
	if msg == "" {
		return nil
	}
	masked := tools.MaskCodeRegions(msg)
	matches := cidImgRe.FindAllStringSubmatchIndex(masked, -1)
	if len(matches) == 0 {
		return nil
	}
	refs := make([]cidRef, 0, len(matches))
	for _, m := range matches {
		// m: [tagStart, tagEnd, tokenStart, tokenEnd].
		tagStart, tagEnd := m[0], m[1]
		tokenStart, tokenEnd := m[2], m[3]
		// "cid:" sits between the opening '(' and tokenStart — length 4.
		urlStart := tokenStart - len("cid:")
		refs = append(refs, cidRef{
			token:    msg[tokenStart:tokenEnd],
			tagStart: tagStart,
			tagEnd:   tagEnd,
			urlStart: urlStart,
			urlEnd:   tokenEnd,
		})
	}
	return refs
}

// resolveCIDs decides per-ref outcomes against the inbound attachment list:
//
//  1. Token "<fid>@<ourDomain>" — quoted-back from the original notification.
//     Returned as a prefilled rewrite to /file/<fid>; does not consume an
//     attachment slot. /file/<id> re-authorises, so trusting the user-supplied
//     fid is safe.
//  2. Attachment.Filename equals the whole token, the token's local-part
//     (before '@'), or "<local-part>.<ext>". First wins; an attachment is
//     consumed at most once.
//  3. Document-order fallback for refs that didn't filename-match: pair them
//     with remaining attachments in the order they appear. Postal doesn't
//     label inline vs plain, so this is the only mechanism that recovers
//     pasted-image references when the MUA renamed the part.
//  4. Refs that exhaust both passes are reported as orphans — the caller
//     drops them.
//
// resolveCIDs does NOT touch S3 or Dgraph. The caller walks `pair`, writes
// each attachment, and appends a cidRewrite resolution carrying the new fid.
func resolveCIDs(refs []cidRef, atts []InboundAttachment, ourDomain string) (pair map[int]int, prefilled []cidResolution, orphans []int) {
	pair = make(map[int]int, len(refs))
	if len(refs) == 0 {
		return pair, nil, nil
	}
	usedAtt := make(map[int]bool, len(atts))
	var unmatched []int

	// Pass 1: quoted-back rewrites + filename-equals matches.
	for i, ref := range refs {
		// Case 1: quoted-back.
		if at := strings.IndexByte(ref.token, '@'); at >= 0 {
			local := ref.token[:at]
			domain := ref.token[at+1:]
			if ourDomain != "" && strings.EqualFold(domain, ourDomain) && fidLikeRe.MatchString(local) {
				prefilled = append(prefilled, cidResolution{
					refIdx: i, kind: cidRewrite, fid: local,
				})
				continue
			}
		}
		// Case 2: filename-equals.
		local := ref.token
		if at := strings.IndexByte(local, '@'); at >= 0 {
			local = local[:at]
		}
		matched := -1
		for j, att := range atts {
			if usedAtt[j] {
				continue
			}
			name := att.Filename
			if name == ref.token || name == local {
				matched = j
				break
			}
			if local != "" && strings.HasPrefix(name, local+".") {
				ext := name[len(local)+1:]
				if ext != "" && !strings.ContainsAny(ext, "/\\.") {
					matched = j
					break
				}
			}
		}
		if matched >= 0 {
			pair[i] = matched
			usedAtt[matched] = true
		} else {
			unmatched = append(unmatched, i)
		}
	}

	// Pass 2: document-order fallback.
	for _, i := range unmatched {
		j := nextUnused(usedAtt, len(atts))
		if j < 0 {
			orphans = append(orphans, i)
			continue
		}
		pair[i] = j
		usedAtt[j] = true
	}
	return pair, prefilled, orphans
}

func nextUnused(used map[int]bool, n int) int {
	for j := 0; j < n; j++ {
		if !used[j] {
			return j
		}
	}
	return -1
}

// applyCIDResolutions rewrites the message in one forward pass using the
// resolved set. Refs without a matching resolution are left in place — the
// caller is expected to cover every ref, but we degrade safely otherwise.
func applyCIDResolutions(msg string, refs []cidRef, res []cidResolution) string {
	if len(refs) == 0 || len(res) == 0 {
		return msg
	}
	byRef := make(map[int]cidResolution, len(res))
	for _, r := range res {
		byRef[r.refIdx] = r
	}
	var b strings.Builder
	b.Grow(len(msg))
	pos := 0
	for i, ref := range refs {
		r, ok := byRef[i]
		if !ok {
			continue
		}
		switch r.kind {
		case cidRewrite:
			b.WriteString(msg[pos:ref.urlStart])
			b.WriteString("/file/")
			b.WriteString(r.fid)
			pos = ref.urlEnd
		case cidDrop:
			b.WriteString(msg[pos:ref.tagStart])
			pos = ref.tagEnd
		}
	}
	b.WriteString(msg[pos:])
	return b.String()
}
