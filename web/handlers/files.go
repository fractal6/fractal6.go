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

// File-attachment HTTP handlers.
//
// Files are anchor-polymorphic: a comment attachment carries (tid, cid),
// a user avatar carries username, an org avatar carries rootnameid. Exactly
// one anchor triple is set per upload; the handler picks the auth path from
// the populated triple.
//
// Bytes never travel through Fractale: every read goes through GET /file/<id>,
// which re-authorises against the populated anchor and 302-redirects to a
// short-lived presigned URL. See docs/file-storage.md.
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/notify"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

// inlineSafeContentTypes lists the MIME types we serve with
// `Content-Disposition: inline`. Anything else is forced to `attachment` to
// neutralise the most common XSS vectors (SVG with embedded <script>,
// HTML masquerading as an image, …).
//
// Inclusion criteria: types that browsers render passively in <img>/<video>/
// <audio>/<embed> contexts without script execution. Notably NOT included:
//
//	image/svg+xml — SVG can carry <script>; rendered inline executes JS.
//	text/html     — obviously executable.
//	application/* (other than pdf) — varies wildly; force download.
var inlineSafeContentTypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"image/avif":      true,
	"image/bmp":       true,
	"image/x-icon":    true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"audio/mpeg":      true,
	"audio/ogg":       true,
	"audio/wav":       true,
	"audio/x-wav":     true,
	"application/pdf": true,
	"text/plain":      true,
}

// --- anchor parsing ---

// uploadAnchor is the resolved (validated) form input. Exactly one of the
// kind-specific fields is populated; Kind records which.
type uploadAnchor struct {
	Kind db.FileKind
	// KindComment.
	Tid string
	Cid string
	// KindUser.
	Username string
	// KindNode.
	Rootnameid string
}

// resolveAnchor inspects the form and returns the single populated anchor.
// Returns an HTTP-status-coded error if zero or multiple anchors are present,
// or if the parts within a triple are missing.
func resolveAnchor(r *http.Request) (uploadAnchor, int, error) {
	tid := strings.TrimSpace(r.FormValue("tid"))
	cid := strings.TrimSpace(r.FormValue("cid"))
	userid := strings.TrimSpace(r.FormValue("userid"))
	orgaid := strings.TrimSpace(r.FormValue("orgaid"))

	// `tid` and `cid` must travel together.
	commentSet := tid != "" || cid != ""
	if commentSet && (tid == "" || cid == "") {
		return uploadAnchor{}, http.StatusBadRequest, fmt.Errorf("tid and cid must be provided together")
	}

	count := 0
	if commentSet {
		count++
	}
	if userid != "" {
		count++
	}
	if orgaid != "" {
		count++
	}
	switch count {
	case 0:
		return uploadAnchor{}, http.StatusBadRequest, fmt.Errorf("an anchor is required: (tid+cid) | userid | orgaid")
	case 1:
		// fallthrough
	default:
		return uploadAnchor{}, http.StatusBadRequest, fmt.Errorf("exactly one anchor allowed; got multiple")
	}

	switch {
	case commentSet:
		return uploadAnchor{Kind: db.KindComment, Tid: tid, Cid: cid}, 0, nil
	case userid != "":
		return uploadAnchor{Kind: db.KindUser, Username: userid}, 0, nil
	default:
		return uploadAnchor{Kind: db.KindNode, Rootnameid: orgaid}, 0, nil
	}
}

// --- Handlers ---

// FileGetHandler returns the GET /file/<id> handler. cli may be nil — in that
// case the handler returns 503 (storage not configured).
func FileGetHandler(cli *storage.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileid := chi.URLParam(r, "id")
		if fileid == "" {
			http.Error(w, "missing file id", http.StatusBadRequest)
			return
		}

		uctx := auth.GetUserContextOrEmpty(r.Context())

		fa, err := db.GetDB().GetFileAuth(fileid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fa == nil {
			http.NotFound(w, r)
			return
		}

		visible, err := isFileVisible(&uctx, fa)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !visible {
			// 404 covers "no such file" and "you can't see this file" alike,
			// to avoid leaking existence to anyone probing IDs.
			http.NotFound(w, r)
			return
		}

		if cli == nil {
			http.Error(w, "storage not configured", http.StatusServiceUnavailable)
			return
		}

		ttl := time.Duration(tools.ViperPositiveInt("storage.presign_ttl_sec", 600)) * time.Second
		dispo := contentDispositionFor(fa.ContentType, fa.Filename)

		presigned, err := cli.PresignGet(r.Context(), fa.StorageKey, ttl, dispo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// private        — browser may cache, intermediaries must not.
		// no-store       — defence in depth on top of the short presign TTL.
		// nosniff        — browsers should not second-guess Content-Type
		//                  (relevant when storage returns the wrong header).
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.Redirect(w, r, presigned, http.StatusFound)
	}
}

// isFileVisible runs the GET-time visibility check appropriate to the file's
// anchor kind. User avatars are public; comment files inherit their parent
// tension's receiver visibility; node avatars follow the node's visibility.
func isFileVisible(uctx *model.UserCtx, fa *db.FileAuth) (bool, error) {
	switch fa.Kind {
	case db.KindUser:
		return true, nil
	case db.KindComment:
		return auth.IsNodeVisible(uctx, fa.ReceiverNameid, fa.ReceiverVisible)
	case db.KindNode:
		return auth.IsNodeVisible(uctx, fa.NodeNameid, fa.NodeVisibility)
	default:
		return false, fmt.Errorf("unknown file kind: %q", fa.Kind)
	}
}

// FileUploadHandler returns the POST /file/upload handler. Multipart fields:
//
//	one of:
//	  tid + cid           — comment attachment
//	  userid              — user avatar
//	  orgaid              — org (root Node) avatar
//	file                  — the file part (required)
//
// On success, returns JSON: {"id": "...", "url": "/file/<id>", "embedded": …}.
func FileUploadHandler(cli *storage.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, uctx, err := auth.GetUserContext(r.Context())
		if err != nil || uctx.Username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if cli == nil {
			http.Error(w, "storage not configured", http.StatusServiceUnavailable)
			return
		}

		maxBytes := tools.ViperPositiveInt("storage.max_upload_bytes", 10*1024*1024)
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

		if err := r.ParseMultipartForm(int64(maxBytes)); err != nil {
			http.Error(w, "upload too large or malformed: "+err.Error(), http.StatusBadRequest)
			return
		}

		anchor, status, err := resolveAnchor(r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file form field is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// MIME sniff: never trust the client-provided Content-Type header,
		// otherwise an attacker can upload an HTML file labelled image/png and
		// have it served back inline. http.DetectContentType returns
		// "application/octet-stream" only when truly unknown; in every other
		// case the sniffed value is used as the persisted contentType.
		sniffBuf := make([]byte, 512)
		n, _ := io.ReadFull(file, sniffBuf)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "cannot rewind upload: "+err.Error(), http.StatusInternalServerError)
			return
		}
		contentType := http.DetectContentType(sniffBuf[:n])
		safeName := safeFilename(header.Filename)

		switch anchor.Kind {
		case db.KindComment:
			handleCommentUpload(w, r, cli, uctx, anchor, file, header, safeName, contentType)
		case db.KindUser:
			handleUserAvatarUpload(w, r, cli, uctx, anchor, file, header, safeName, contentType)
		case db.KindNode:
			handleNodeAvatarUpload(w, r, cli, uctx, anchor, file, header, safeName, contentType)
		default:
			http.Error(w, "unknown anchor kind", http.StatusBadRequest)
		}
	}
}

// --- per-anchor upload paths ---

func handleCommentUpload(w http.ResponseWriter, r *http.Request, cli *storage.Client, uctx *model.UserCtx, anchor uploadAnchor, file io.Reader, header *multipart.FileHeader, safeName, contentType string) {
	// Auth: caller must be allowed to push CommentPushed on the tension. We
	// reuse the EMAP entry so any future tightening (e.g. quota, throttle)
	// applies uniformly. ProcessEvent with doProcess=false runs Check only —
	// no side effects.
	tension, err := db.GetDB().GetTensionHook(anchor.Tid, false, nil)
	if err != nil || tension == nil {
		http.Error(w, "tension not found", http.StatusNotFound)
		return
	}
	e := model.TensionEventCommentPushed
	event := &model.EventRef{EventType: &e}
	ok, _, err := graph.ProcessEvent(uctx, tension, event, nil, nil, true, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if !ok {
		http.Error(w, "not authorised to attach files in this tension", http.StatusForbidden)
		return
	}

	// Verify cid belongs to tid + read message + author in the same hop.
	c, err := db.GetDB().GetCommentForUpload(anchor.Tid, anchor.Cid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !c.Found {
		http.Error(w, "comment does not belong to tension", http.StatusBadRequest)
		return
	}
	if c.AuthorUsername != uctx.Username {
		http.Error(w, "only the comment author can attach files", http.StatusForbidden)
		return
	}
	oldMessage := c.Message

	rootnameid, err := codec.Nid2rootid(tension.Receiver.Nameid)
	if err != nil {
		http.Error(w, "bad receiver nameid: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fid, err := writeCommentAttachment(r.Context(), cli,
		rootnameid, anchor.Tid, anchor.Cid, uctx.Username,
		safeName, contentType, file, header.Size)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errStoragePut) {
			status = http.StatusBadGateway
		}
		http.Error(w, err.Error(), status)
		return
	}

	// Inline-screenshot rewrite: if `safeName` appears as a bare ![](filename)
	// token in the comment, swap it for /file/<fid> and flip File.embedded=true.
	// Successful rewrites decrement the per-tension upload gate so the notifier
	// daemon stops waiting on this paste.
	embedded := embedIfReferenced(r.Context(), anchor.Tid, anchor.Cid, fid, safeName, oldMessage)

	writeJSON(w, map[string]any{
		"id":          fid,
		"url":         "/file/" + fid,
		"filename":    safeName,
		"contentType": contentType,
		"size":        header.Size,
		"embedded":    embedded,
	})
}

// errStoragePut / errPersistFile are sentinels wrapped around the underlying
// driver error so callers can pick an HTTP status (502 vs 500) via errors.Is
// without string-sniffing err.Error().
var (
	errStoragePut  = errors.New("upload failed")
	errPersistFile = errors.New("persist failed")
)

// writeCommentAttachment puts bytes to S3 and inserts the File row anchored
// on (tid, cid). Auth and the optional message-rewrite are the caller's
// responsibility — this is the post-auth core shared by /file/upload and the
// inbound-email path. Returns the new fid.
//
// Rollback policy: an S3 success followed by a DB failure removes the
// orphaned object before returning.
func writeCommentAttachment(
	ctx context.Context, cli *storage.Client,
	rootnameid, tid, cid, username, safeName, contentType string,
	body io.Reader, size int64,
) (string, error) {
	storageKey := commentKeyPrefix(rootnameid, tid, cid) + randomID() + "-" + safeName
	if err := cli.Put(ctx, storageKey, body, size, contentType); err != nil {
		return "", fmt.Errorf("%w: %v", errStoragePut, err)
	}
	fid, err := db.GetDB().AddCommentFile(
		tid, cid, username, safeName, contentType, storageKey,
		size, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		_ = cli.Delete(ctx, storageKey)
		return "", fmt.Errorf("%w: %v", errPersistFile, err)
	}
	return fid, nil
}

// embedIfReferenced rewrites the comment message in place when the uploaded
// filename is referenced as an inline `![alt](filename)` token. Last-writer-
// wins on Comment.message: if two uploads race, the second overwrite can
// drop the first's URL substitution. The File rows themselves are unaffected
// (they're independent), and the UI is lenient about embedded=true files
// whose URL is no longer in the message (renders them as plain attachments).
//
// On a successful inline rewrite we Signal the per-tension upload gate so the
// notifier daemon can release its Wait once every expected paste has landed.
// Plain (non-inline) attachments don't signal — they're handled best-effort
// via the gate's baseline wait + Comment.files re-fetch on the notifier side.
func embedIfReferenced(ctx context.Context, tid, cid, fid, filename, message string) bool {
	newMsg, matched := rewriteMessageForFile(message, filename, fid)
	if !matched {
		return false
	}
	if err := db.GetDB().EmbedCommentMessage(cid, newMsg, []string{fid}); err != nil {
		log.Printf("Warning: embedCommentMessage: %v", err)
		return false
	}
	if err := notify.Global().Signal(ctx, tid); err != nil {
		log.Printf("Warning: upload-gate signal (tid=%s): %v", tid, err)
	}
	return true
}

func handleUserAvatarUpload(w http.ResponseWriter, r *http.Request, cli *storage.Client, uctx *model.UserCtx, anchor uploadAnchor, file io.Reader, header *multipart.FileHeader, safeName, contentType string) {
	if !strings.EqualFold(anchor.Username, uctx.Username) {
		http.Error(w, "you may only upload your own avatar", http.StatusForbidden)
		return
	}
	storageKey := userKeyPrefix(uctx.Username) + randomID() + "-" + safeName
	if err := cli.Put(r.Context(), storageKey, file, header.Size, contentType); err != nil {
		http.Error(w, "upload failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	fid, err := db.GetDB().ReplaceUserAvatar(
		uctx.Username, safeName, contentType, storageKey,
		header.Size, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		_ = cli.Delete(r.Context(), storageKey)
		http.Error(w, "persist failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"id":          fid,
		"url":         "/file/" + fid,
		"filename":    safeName,
		"contentType": contentType,
		"size":        header.Size,
	})
}

func handleNodeAvatarUpload(w http.ResponseWriter, r *http.Request, cli *storage.Client, uctx *model.UserCtx, anchor uploadAnchor, file io.Reader, header *multipart.FileHeader, safeName, contentType string) {
	if auth.UserHasCoordoRole(uctx, anchor.Rootnameid) < 0 {
		http.Error(w, "coordinator role required", http.StatusForbidden)
		return
	}
	storageKey := orgaKeyPrefix(anchor.Rootnameid) + randomID() + "-" + safeName
	if err := cli.Put(r.Context(), storageKey, file, header.Size, contentType); err != nil {
		http.Error(w, "upload failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	fid, err := db.GetDB().ReplaceNodeAvatar(
		anchor.Rootnameid, uctx.Username, safeName, contentType, storageKey,
		header.Size, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		_ = cli.Delete(r.Context(), storageKey)
		http.Error(w, "persist failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"id":          fid,
		"url":         "/file/" + fid,
		"filename":    safeName,
		"contentType": contentType,
		"size":        header.Size,
	})
}

// --- delete ---

// FileDeleteHandler returns the DELETE /file/<id> handler. Uploader-only
// across all anchor kinds; same 404-instead-of-403 policy as FileGet.
// DB-first: db.DeleteFile drops the row then GCs the object async.
func FileDeleteHandler(cli *storage.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, uctx, err := auth.GetUserContext(r.Context())
		if err != nil || uctx.Username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fileid := chi.URLParam(r, "id")
		if fileid == "" {
			http.Error(w, "missing file id", http.StatusBadRequest)
			return
		}
		if cli == nil {
			http.Error(w, "storage not configured", http.StatusServiceUnavailable)
			return
		}

		fa, err := db.GetDB().GetFileAuth(fileid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fa == nil || fa.UploaderUsername != uctx.Username {
			http.NotFound(w, r)
			return
		}

		if err := db.GetDB().DeleteFile(fileid); err != nil {
			http.Error(w, "db delete failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- markdown rewrite ---

// rewriteMessageForFile substitutes the first ![alt](filename) whose URL is
// the bare token `filename` (no slashes, no scheme) with `![alt](/file/<fid>)`.
// Code regions (fenced ``` and ~~~ blocks, inline backticks) are masked
// before matching so filenames mentioned in code don't trigger a rewrite.
//
// Returns (newMessage, true) when a substitution was made; (msg, false) when
// the filename is not referenced inline.
func rewriteMessageForFile(msg, filename, fid string) (string, bool) {
	if msg == "" || filename == "" || fid == "" {
		return msg, false
	}
	masked := tools.MaskCodeRegions(msg)

	matches := tools.InlineImageRe.FindAllStringSubmatchIndex(masked, -1)
	for _, m := range matches {
		urlStart, urlEnd := m[2], m[3]
		urlInOriginal := msg[urlStart:urlEnd]
		if urlInOriginal != filename {
			continue
		}
		// Bare filename only: reject anything carrying a path separator ("/")
		// or a scheme delimiter (":"). The ":" guard is what does the work
		// here — safeFilename already strips "/", but it does NOT strip ":",
		// so a filename like "weird:name.png" would otherwise sneak through.
		if strings.ContainsAny(urlInOriginal, "/:") {
			continue
		}
		newMsg := msg[:urlStart] + "/file/" + fid + msg[urlEnd:]
		return newMsg, true
	}
	return msg, false
}

// --- helpers ---

// contentDispositionFor returns the Content-Disposition header value for a
// file served by /file/<id>. Inline-safe MIME types render in the page;
// everything else is forced to attachment.
func contentDispositionFor(contentType, filename string) string {
	verb := "attachment"
	if inlineSafeContentTypes[strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))] {
		verb = "inline"
	}
	if filename == "" {
		return verb
	}
	return fmt.Sprintf(`%s; filename*=UTF-8''%s`, verb, url.QueryEscape(filename))
}

// safeFilename keeps the basename, replaces path separators / NUL, and caps
// length so the storage key stays bounded.
func safeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20, r == 0x7f:
			continue
		case r == '/', r == '\\':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}

// commentKeyPrefix / userKeyPrefix / orgaKeyPrefix are the per-anchor
// namespacing conventions. See docs/file-storage.md.
func commentKeyPrefix(rootnameid, tid, cid string) string {
	return "orgas/" + rootnameid + "/tensions/" + tid + "/" + cid + "/"
}
func userKeyPrefix(username string) string  { return "users/" + username + "/" }
func orgaKeyPrefix(rootnameid string) string { return "orgas/" + rootnameid + "/" }

// randomID returns a 16-hex-char random string (~64 bits of entropy) used as
// a storage-key collision guard. Not security-critical (auth is server-side).
func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

