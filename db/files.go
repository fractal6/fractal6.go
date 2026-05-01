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

package db

import (
	"context"
	"fmt"
	"time"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/storage"
)

// FileAuth aggregates everything web/handlers/files.go needs to authorise
// a /file/<id> request and to mint a presigned URL: the storage key + content
// metadata for the redirect, plus the comment author username and the parent
// tension's receiver visibility (consumed by IsNodeVisible).
type FileAuth struct {
	StorageKey      string               `mapstructure:"File.storageKey"`
	Filename        string               `mapstructure:"File.filename"`
	ContentType     string               `mapstructure:"File.contentType"`
	Size            int                  `mapstructure:"File.size"`
	AuthorUsername  string               // resolved from File.comment > Post.createdBy
	ReceiverNameid  string               // resolved from File.comment > Tension.receiver
	ReceiverVisible model.NodeVisibility // resolved from File.comment > Tension.receiver
}

// CommentAuth is FileAuth's smaller sibling for upload-time checks: we don't
// have a File yet, so we resolve auth straight from the comment id.
type CommentAuth struct {
	AuthorUsername  string
	ReceiverNameid  string
	ReceiverVisible model.NodeVisibility
}

// GetFileAuth runs the getFileAuth template and flattens the nested DQL
// response into a FileAuth. Returns (nil, nil) when no file matches; the
// caller surfaces 404 in that case.
func (dg Dgraph) GetFileAuth(fileid string) (*FileAuth, error) {
	res, err := dg.Meta("getFileAuth", map[string]string{"id": fileid})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	r := res[0]
	out := &FileAuth{
		StorageKey:  asString(r["File.storageKey"]),
		Filename:    asString(r["File.filename"]),
		ContentType: asString(r["File.contentType"]),
		Size:        asInt(r["File.size"]),
	}
	c, ok := firstChild(r, "File.comment")
	if !ok {
		return nil, fmt.Errorf("file %s: missing parent comment", fileid)
	}
	out.AuthorUsername = nestedUsername(c, "Post.createdBy")
	t, ok := firstChild(c, "Comment.tension")
	if !ok {
		return nil, fmt.Errorf("file %s: missing parent tension", fileid)
	}
	rcv, ok := firstChild(t, "Tension.receiver")
	if !ok {
		return nil, fmt.Errorf("file %s: missing tension receiver", fileid)
	}
	out.ReceiverNameid = asString(rcv["Node.nameid"])
	out.ReceiverVisible = model.NodeVisibility(asString(rcv["Node.visibility"]))
	return out, nil
}

// GetCommentAuth resolves the upload-time auth for a comment id.
func (dg Dgraph) GetCommentAuth(cid string) (*CommentAuth, error) {
	res, err := dg.Meta("getCommentAuth", map[string]string{"id": cid})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	r := res[0]
	out := &CommentAuth{
		AuthorUsername: nestedUsername(r, "Post.createdBy"),
	}
	t, ok := firstChild(r, "Comment.tension")
	if !ok {
		return nil, fmt.Errorf("comment %s: missing parent tension", cid)
	}
	rcv, ok := firstChild(t, "Tension.receiver")
	if !ok {
		return nil, fmt.Errorf("comment %s: missing tension receiver", cid)
	}
	out.ReceiverNameid = asString(rcv["Node.nameid"])
	out.ReceiverVisible = model.NodeVisibility(asString(rcv["Node.visibility"]))
	return out, nil
}

// AddFileToComment inserts a File node and returns its newly assigned uid.
// The caller MUST have already verified that uctx.Username is the comment
// author (see web/handlers/files.go).
func (dg Dgraph) AddFileToComment(cid, username, filename, contentType, storageKey, nowRFC3339 string, size int64) (string, error) {
	q := dqlMutations["addFileToComment"]
	res, err := dg.MutateWithQueryDql3(q, map[string]string{
		"cid":         cid,
		"username":    username,
		"filename":    escapeNQuad(filename),
		"contentType": escapeNQuad(contentType),
		"storageKey":  escapeNQuad(storageKey),
		"sizeStr":     fmt.Sprintf("%d", size),
		"now":         nowRFC3339,
	})
	if err != nil {
		return "", err
	}
	uid, ok := res.Uids["f"]
	if !ok || uid == "" {
		return "", fmt.Errorf("addFileToComment: no uid returned")
	}
	return uid, nil
}

// DeleteFile removes a File node by uid. Idempotent.
func (dg Dgraph) DeleteFile(fileid string) error {
	_, err := dg.Meta("deleteFile", map[string]string{"id": fileid})
	return err
}

// GetCommentFileKeys returns (uid, storageKey) pairs for every file attached
// to a comment. Used by the comment-delete cleanup path.
func (dg Dgraph) GetCommentFileKeys(cid string) ([]struct{ UID, StorageKey string }, error) {
	res, err := dg.Meta("getCommentFiles", map[string]string{"id": cid})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	files, _ := res[0]["Comment.files"].([]any)
	out := make([]struct{ UID, StorageKey string }, 0, len(files))
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, struct{ UID, StorageKey string }{
			UID:        asString(m["uid"]),
			StorageKey: asString(m["File.storageKey"]),
		})
	}
	return out, nil
}

// CleanupCommentFiles removes every S3 object attached to the given comment.
// Called by the comment-delete event path before the deleteComment template
// drops the File nodes from Dgraph. Per-file failures are logged but do NOT
// abort: leaving an orphan object is preferable to blocking the delete (the
// bucket can be swept out-of-band).
//
// Lives in the db package (not web/handlers) so graph/ can call it without
// creating an import cycle (handlers already imports graph).
func (dg Dgraph) CleanupCommentFiles(cid string) error {
	files, err := dg.GetCommentFileKeys(cid)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	cli, err := storage.GetDefault()
	if err != nil {
		// Storage not configured — comment-deletion still proceeds; the DQL
		// template will drop the File nodes. Operators can sweep the bucket
		// out-of-band if storage is reattached later.
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, f := range files {
		if f.StorageKey == "" {
			continue
		}
		if delErr := cli.Delete(ctx, f.StorageKey); delErr != nil {
			fmt.Printf("CleanupCommentFiles: failed to delete %s: %v\n", f.StorageKey, delErr)
		}
	}
	return nil
}

// --- small helpers (kept private to this file) ---

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// firstChild unwraps Dgraph's [{...}] nesting for single-cardinality edges.
func firstChild(parent map[string]any, key string) (map[string]any, bool) {
	switch v := parent[key].(type) {
	case map[string]any:
		return v, true
	case []any:
		if len(v) == 0 {
			return nil, false
		}
		m, ok := v[0].(map[string]any)
		return m, ok
	}
	return nil, false
}

// nestedUsername extracts Post.createdBy > User.username through the parent map.
func nestedUsername(parent map[string]any, edge string) string {
	c, ok := firstChild(parent, edge)
	if !ok {
		return ""
	}
	return asString(c["User.username"])
}

// escapeNQuad escapes characters that would break an N-Quad literal.
// Dgraph's RDF parser requires \" and \\ inside quoted literals.
func escapeNQuad(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			out = append(out, '\\', '\\')
		case '"':
			out = append(out, '\\', '"')
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
