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
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/web/auth"
)

// FileGet serves /file/<id>. Auth-and-redirect: validates the user can see the
// parent comment's tension, then 302s to a presigned URL. Cache-Control is
// private + no-store so intermediaries don't leak the redirect across users.
func FileGet(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Read auth: same rule as for the parent comment — visibility on the
	// receiver node of the tension this comment belongs to.
	visible, err := auth.IsNodeVisible(&uctx, fa.ReceiverNameid, fa.ReceiverVisible)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !visible {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	cli, err := storage.GetDefault()
	if err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

	ttl := time.Duration(viperPositiveInt("storage.presign_ttl_sec", 600)) * time.Second
	url, err := cli.PresignGet(r.Context(), fa.StorageKey, ttl, fa.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// private = browser may cache, intermediaries must not.
	// no-store on the redirect response itself; the presigned URL has its own
	// short TTL that bounds bytes-cache lifetime regardless of headers.
	w.Header().Set("Cache-Control", "private, no-store")
	http.Redirect(w, r, url, http.StatusFound)
}

// FileUpload serves POST /file/upload (multipart/form-data).
// Form fields:
//
//	comment_id : Dgraph uid of the parent comment (required)
//	file       : the file part (required)
//
// On success, returns JSON: {"id": "<file uid>", "url": "/file/<id>", ...meta}.
func FileUpload(w http.ResponseWriter, r *http.Request) {
	_, uctx, err := auth.GetUserContext(r.Context())
	if err != nil || uctx.Username == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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

	// Sanitise filename: strip path components and limit length. The original
	// name is preserved as metadata for Content-Disposition; the storage key
	// uses a uuid prefix to prevent collisions and key-guessing.
	safeName := safeFilename(header.Filename)
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	keyPrefix := commentKeyPrefix(cid)
	storageKey := keyPrefix + randomID() + "-" + safeName

	cli, err := storage.GetDefault()
	if err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

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

// FileDelete serves DELETE /file/<id>. Author-only.
func FileDelete(w http.ResponseWriter, r *http.Request) {
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

	fa, err := db.GetDB().GetFileAuth(fileid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fa == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if fa.AuthorUsername != uctx.Username {
		http.Error(w, "only the comment author can delete this file", http.StatusForbidden)
		return
	}

	cli, err := storage.GetDefault()
	if err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
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

// --- helpers ---

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

