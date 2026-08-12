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

// Email attachment plumbing. Files attached to the comment ride out as
// Postal `attachments` entries so the recipient sees them locally in the
// email instead of behind the auth-gated /file/<id> proxy. Two paths:
//
//   Bucket A — inline images: render in-body via `<img src="cid:...">`,
//              base64-encoded with a Content-ID header on the attachment.
//   Bucket B — plain attachments: paperclip in the mail UI; the HTML body
//              also gains a footer list with click-through links so the
//              recipient sees them even if their client hides the paperclip.
//
// Per-file and total-size caps protect both Postal's transport budget and
// the recipient mailbox; oversized files are dropped from the payload but
// still appear in the footer link list.

package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/internal/tools"
)

// inlineImageTypes are the MIME types we'll embed as CID inline attachments.
// Everything else (PDFs, video, SVG, …) goes to Bucket B regardless of
// File.embedded. SVG is excluded on purpose — it can carry <script> and
// some mail clients still execute it inline.
var inlineImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// attachmentLimits is the per-email caps, resolved from viper at call time
// (so config reloads pick up without process restart, and tests can override
// via viper.Set).
type attachmentLimits struct {
	PerFileBytes int64
	TotalBytes   int64
	MaxCount     int
}

func loadAttachmentLimits() attachmentLimits {
	return attachmentLimits{
		PerFileBytes: int64(tools.ViperPositiveInt("notify.attachment_per_file_bytes", 10*1024*1024)),
		TotalBytes:   int64(tools.ViperPositiveInt("notify.attachment_total_bytes", 100*1024*1024)),
		MaxCount:     tools.ViperPositiveInt("notify.attachment_max_count", 20),
	}
}

// postalAttachment is the Postal API attachment shape. content_id is omitted
// (rather than empty-stringed) for Bucket-B entries; otherwise some clients
// hide the paperclip thinking it's an inline-only image.
type postalAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Data        string `json:"data"`
	ContentID   string `json:"content_id,omitempty"`
}

// emailFile decouples the email package from db.CommentFile so unit tests
// don't need a Dgraph fixture.
type emailFile struct {
	ID          string
	StorageKey  string
	Filename    string
	ContentType string
	Size        int64
	Embedded    bool
}

func fromDBFiles(in []db.CommentFile) []emailFile {
	out := make([]emailFile, len(in))
	for i, f := range in {
		out[i] = emailFile{
			ID:          f.ID,
			StorageKey:  f.StorageKey,
			Filename:    f.Filename,
			ContentType: f.ContentType,
			Size:        int64(f.Size),
			Embedded:    f.Embedded,
		}
	}
	return out
}

// inlineCandidate reports whether the file should be tried for in-body CID
// embedding: it was referenced as `![](...)` in the comment AND is one of
// the safe browser-renderable image types.
func (f emailFile) inlineCandidate() bool {
	return f.Embedded && inlineImageTypes[strings.ToLower(strings.TrimSpace(strings.SplitN(f.ContentType, ";", 2)[0]))]
}

// partition splits files into (inlineCandidates, plain). Order within each
// bucket follows the input order; the caller then walks the lists in order
// and applies the per-email caps.
func partition(files []emailFile) (inline, plain []emailFile) {
	for _, f := range files {
		if f.inlineCandidate() {
			inline = append(inline, f)
		} else {
			plain = append(plain, f)
		}
	}
	return
}

// fetchedBytes is an emailFile + its body, base64-encoded ready to drop
// into the Postal payload.
type fetchedBytes struct {
	File    emailFile
	Base64  string
	UsedCID bool
}

// buildAttachments fetches bytes for each file (Bucket A first, then B),
// stops once a cap is hit, and returns:
//
//   - attachments: Postal payload entries (Bucket A first then B).
//   - inlineByID:  the subset of file uids that DID get an inline CID slot —
//                  used to rewrite `<img src="/file/<id>">` to `cid:<id>@DOMAIN`.
//   - footerFiles: every plain (Bucket B) file the caller should render in
//                  the footer link list; includes both attached and
//                  cap-overflow entries so nothing is silently hidden.
//
// Failing to fetch a single file's bytes is logged but never fatal — that
// file degrades to footer-link-only (Bucket B) or absolute-URL fallback
// (Bucket A, handled by the caller via the inlineByID set).
func buildAttachments(ctx context.Context, cli *storage.Client, files []emailFile) (attachments []postalAttachment, inlineByID map[string]bool, footerFiles []emailFile) {
	inlineByID = make(map[string]bool)
	if len(files) == 0 || cli == nil {
		// Storage unset or no files: still expose every plain file in the
		// footer so the recipient can click through.
		_, plain := partition(files)
		footerFiles = plain
		return
	}
	limits := loadAttachmentLimits()

	inline, plain := partition(files)
	footerFiles = plain

	var total int64
	// Bucket A — inline images first. They're the highest signal payload
	// (visually rendered in-body); the recipient's mail client may not
	// even surface paperclip attachments, but it WILL render <img cid:>.
	for _, f := range inline {
		if len(attachments) >= limits.MaxCount {
			break
		}
		if f.Size > 0 && f.Size > limits.PerFileBytes {
			continue
		}
		if total+f.Size > limits.TotalBytes {
			continue
		}
		fetched, err := fetchBase64(ctx, cli, f)
		if err != nil {
			// Drop silently from CID attachments; caller's <img src="/file/...">
			// fallback (absolute URL) will at least surface the link.
			continue
		}
		attachments = append(attachments, postalAttachment{
			Name:        f.Filename,
			ContentType: f.ContentType,
			Data:        fetched.Base64,
			ContentID:   cidForFile(f.ID),
		})
		inlineByID[f.ID] = true
		total += int64(len(fetched.Base64) * 3 / 4) // approximate decoded size
	}

	// Bucket B — plain attachments. Dropped first if we approach the total
	// budget; spillover lives on as footer links.
	for _, f := range plain {
		if len(attachments) >= limits.MaxCount {
			break
		}
		if f.Size > 0 && f.Size > limits.PerFileBytes {
			continue
		}
		if total+f.Size > limits.TotalBytes {
			continue
		}
		fetched, err := fetchBase64(ctx, cli, f)
		if err != nil {
			continue
		}
		attachments = append(attachments, postalAttachment{
			Name:        f.Filename,
			ContentType: f.ContentType,
			Data:        fetched.Base64,
		})
		total += int64(len(fetched.Base64) * 3 / 4)
	}
	return
}

// fetchBase64 streams a file from storage and returns its body base64-encoded.
// Caller has already filtered by per-file cap so we don't re-check here.
func fetchBase64(ctx context.Context, cli *storage.Client, f emailFile) (fetchedBytes, error) {
	obj, _, err := cli.GetObject(ctx, f.StorageKey)
	if err != nil {
		return fetchedBytes{}, err
	}
	defer obj.Close()
	raw, err := io.ReadAll(obj)
	if err != nil {
		return fetchedBytes{}, err
	}
	return fetchedBytes{
		File:   f,
		Base64: base64.StdEncoding.EncodeToString(raw),
	}, nil
}

// cidForFile builds the RFC 2392 Content-ID for a given file uid. We
// scope by DOMAIN so a cid collision across orgs / instances is impossible.
func cidForFile(fid string) string {
	return fmt.Sprintf("%s@%s", fid, DOMAIN)
}

// fileImgRe matches the markdown-rendered `<img src="/file/<id>">` tag (and
// quote-variant `'`). The capture group is the file uid; we accept anything
// that doesn't contain a quote or '/' so future fid formats keep working.
var fileImgRe = regexp.MustCompile(`<img\s+([^>]*?)src=(?:"|&#34;|')/file/([^"'/&]+)("|&#34;|')([^>]*)>`)

// rewriteFileImgs rewrites every markdown-rendered `<img src="/file/<id>">`
// in a single pass: uids in inlineByID get `<img src="cid:<id>@DOMAIN">`
// (they ride out as inline attachments); the rest fall back to the absolute
// `https://<DOMAIN>/file/<id>` — the documented degradation path when an
// inline image couldn't be attached (Public-org images may load for the
// recipient even if Private-org ones 404). Other tag attributes are
// preserved verbatim.
func rewriteFileImgs(html string, inlineByID map[string]bool) string {
	return fileImgRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := fileImgRe.FindStringSubmatch(match)
		if len(sub) < 5 {
			return match
		}
		fid := sub[2]
		if inlineByID[fid] {
			return fmt.Sprintf(`<img %ssrc="cid:%s"%s>`, sub[1], cidForFile(fid), sub[4])
		}
		return fmt.Sprintf(`<img %ssrc="https://%s/file/%s"%s>`, sub[1], DOMAIN, fid, sub[4])
	})
}

// renderAttachmentFooter builds the `Attachments:` block listed at the end
// of the email body. One <a> per file, click-through to /file/<id> on the
// web UI so the recipient can fetch the asset even if Postal stripped or
// the mail client hid the paperclip. Empty when there are no plain files.
//
// Files that ended up as inline CID attachments are intentionally NOT
// listed here — they're already visible in-body, and a duplicate link
// would look like clutter.
func renderAttachmentFooter(files []emailFile) string {
	if len(files) == 0 {
		return ""
	}
	var b bytes.Buffer
	b.WriteString(`<div style="margin-top:1em;padding-top:0.5em;border-top:1px solid #eee;color:#666;font-size:small">`)
	b.WriteString(`Attachments:<br>`)
	for _, f := range files {
		fmt.Fprintf(&b,
			`<a href="https://%s/file/%s">%s</a> (%s)<br>`,
			DOMAIN, f.ID, htmlEscape(f.Filename), humanSize(f.Size),
		)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// htmlEscape is a tiny escape for filenames in the footer link text. The
// filenames pass through safeFilename() at upload time so they're already
// fairly tame, but a paranoid escape avoids opening an XSS hole if the
// filter set ever weakens.
func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

// humanSize renders a byte count as a friendly string (e.g. "1.2 MB").
func humanSize(n int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/gb)
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
