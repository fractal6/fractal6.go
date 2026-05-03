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
// Design (see docs/file-attachments.md for the full picture):
//
//   - Bytes never travel through Fractale. Every read goes through /file/<id>,
//     which re-runs the parent comment's tension visibility check, then issues
//     a 302 to a short-lived presigned URL minted by the storage backend.
//
//   - The same /file/<id> URL is used for both first-class attachments and
//     markdown-embedded images. There is one auth path, one stable URL shape.
//
//   - Uploads (POST /file/upload) require the caller to be the comment author —
//     attachments are bound to a comment at creation time and inherit its auth.
//
//   - Deletes (DELETE /file/<id>) require the comment author too. The S3 object
//     is removed first, then the Dgraph node; if S3 deletion fails, the DB
//     record is kept so we can retry rather than leak orphan bytes.
//
// Handlers are constructed with an injected *storage.Client (or nil when the
// [storage] section of config.toml is unset). This keeps the storage backend
// swappable in tests and lets `cmd/server.go` fail closed (503) when storage
// is not configured rather than panicking at request time.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/storage"
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
		// 404 covers both "no such file" and "you can't see this file".
		// Returning 403 in the second case would leak existence to anyone
		// probing IDs (Dgraph uids aren't sequential, but it's a free fix).
		if fa == nil {
			http.NotFound(w, r)
			return
		}

		visible, err := auth.IsNodeVisible(&uctx, fa.ReceiverNameid, fa.ReceiverVisible)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !visible {
			http.NotFound(w, r)
			return
		}

		if cli == nil {
			http.Error(w, "storage not configured", http.StatusServiceUnavailable)
			return
		}

		ttl := time.Duration(viperPositiveInt("storage.presign_ttl_sec", 600)) * time.Second
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

// FileUploadHandler returns the POST /file/upload handler. multipart fields:
//
//	comment_id : Dgraph uid of the parent comment (required)
//	file       : the file part (required)
//
// On success, returns JSON: {"id": "<file uid>", "url": "/file/<id>", ...meta}.
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

		maxBytes := viperPositiveInt("storage.max_upload_bytes", 10*1024*1024)
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

		if err := r.ParseMultipartForm(int64(maxBytes)); err != nil {
			http.Error(w, "upload too large or malformed: "+err.Error(), http.StatusBadRequest)
			return
		}

		cid := strings.TrimSpace(r.FormValue("comment_id"))
		if cid == "" {
			http.Error(w, "comment_id is required", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file form field is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Auth: only the comment author may attach files. This mirrors the existing
		// rule for editing/deleting one's own comments (see graph/tension_op.go).
		ca, err := db.GetDB().GetCommentAuth(cid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ca == nil {
			http.Error(w, "comment not found", http.StatusNotFound)
			return
		}
		if ca.AuthorUsername != uctx.Username {
			http.Error(w, "only the comment author can attach files", http.StatusForbidden)
			return
		}

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

		// Sanitise filename: strip path components and limit length. The original
		// name is preserved as metadata for Content-Disposition; the storage key
		// uses a uuid prefix to prevent collisions and key-guessing.
		safeName := safeFilename(header.Filename)

		keyPrefix := commentKeyPrefix(cid)
		storageKey := keyPrefix + randomID() + "-" + safeName

		if err := cli.Put(r.Context(), storageKey, file, header.Size, contentType); err != nil {
			http.Error(w, "upload failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		uid, err := db.GetDB().AddFileToComment(
			cid, uctx.Username, safeName, contentType, storageKey,
			time.Now().UTC().Format(time.RFC3339), header.Size,
		)
		if err != nil {
			// DB persistence failed after the upload succeeded — roll back the
			// object so we don't accumulate orphans. Log but don't fail the rollback
			// to the client; the original error is what matters.
			_ = cli.Delete(r.Context(), storageKey)
			http.Error(w, "persist failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"id":          uid,
			"url":         "/file/" + uid,
			"filename":    safeName,
			"contentType": contentType,
			"size":        header.Size,
		})
	}
}

// FileDeleteHandler returns the DELETE /file/<id> handler. Author-only; same
// 404-instead-of-403 policy as FileGet.
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
		if fa == nil || fa.AuthorUsername != uctx.Username {
			// Same 404 policy as FileGet — don't differentiate "missing" vs
			// "not yours".
			http.NotFound(w, r)
			return
		}

		// Delete the object first. If this fails we keep the DB record so we can
		// retry cleanup; the alternative would leak bytes that are no longer
		// referenced from Dgraph.
		if err := cli.Delete(r.Context(), fa.StorageKey); err != nil {
			http.Error(w, "storage delete failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		if err := db.GetDB().DeleteFile(fileid); err != nil {
			http.Error(w, "db delete failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- helpers ---

// contentDispositionFor returns the Content-Disposition header value for a file
// served by /file/<id>. Inline-safe MIME types render in the page; everything
// else is forced to attachment (browser saves to disk instead of executing).
func contentDispositionFor(contentType, filename string) string {
	verb := "attachment"
	if inlineSafeContentTypes[strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))] {
		verb = "inline"
	}
	if filename == "" {
		return verb
	}
	// RFC 5987 encoding for non-ASCII filenames; storage backends forward this
	// verbatim via the response-content-disposition presign parameter.
	return fmt.Sprintf(`%s; filename*=UTF-8''%s`, verb, url.QueryEscape(filename))
}

// safeFilename keeps the basename, replaces path separators / NUL, and caps
// length so the storage key stays bounded.
func safeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == "/" {
		name = "file"
	}
	// Strip control characters and anything that would break an N-Quad literal
	// without escaping; storage keys live in URLs and Dgraph predicates.
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

// commentKeyPrefix is the namespacing convention; see config.toml [storage]
// for the per-kind prefixes (comments/, avatars/, orgas/...).
func commentKeyPrefix(cid string) string {
	return "comments/" + cid + "/"
}

// randomID returns a 16-hex-char random string (~64 bits of entropy) used as
// a storage-key collision guard. Not security-critical (auth is server-side).
func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func viperPositiveInt(key string, fallback int) int {
	v := viper.GetInt(key)
	if v <= 0 {
		return fallback
	}
	return v
}
