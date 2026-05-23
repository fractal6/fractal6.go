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
	"log"
	"time"

	"github.com/dgraph-io/dgo/v200/protos/api"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/storage"
)

// FileKind discriminates the populated anchor branch in FileAuth.
type FileKind string

const (
	KindComment FileKind = "comment"
	KindUser    FileKind = "user"
	KindNode    FileKind = "node"
)

// FileAuth aggregates everything web/handlers/files.go needs to authorise
// a /file/<id> request and to mint a presigned URL. Files are anchor-
// polymorphic: exactly one of (comment+tension), user, node is set per file.
// `Kind` records which branch is populated; the relevant subset of fields is
// then valid:
//
//   - KindComment:  ReceiverNameid + ReceiverVisible (parent tension's receiver)
//   - KindUser:     no extra fields — user avatars are public
//   - KindNode:     NodeNameid + NodeVisibility
//
// UploaderUsername is always set (File.createdBy.username); the DELETE handler
// uses it for the uploader-only rule.
type FileAuth struct {
	Kind             FileKind
	StorageKey       string
	Filename         string
	ContentType      string
	Size             int
	UploaderUsername string

	// KindComment branch.
	ReceiverNameid  string
	ReceiverVisible model.NodeVisibility

	// KindNode branch.
	NodeNameid     string
	NodeVisibility model.NodeVisibility
}

// GetFileAuth runs getFileAuth_v2 and flattens the response. Returns
// (nil, nil) when no file matches; the caller surfaces 404 in that case.
//
// db.Meta runs the response through tools.CleanDqlMap which strips "Type."
// prefixes from keys, so we look up "comment" not "File.comment", "username"
// not "User.username", etc.
func (dg Dgraph) GetFileAuth(fileid string) (*FileAuth, error) {
	res, err := dg.Meta("getFileAuth_v2", map[string]string{"id": fileid})
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	f := res[0]

	fa := &FileAuth{
		StorageKey:       asString(f["storageKey"]),
		Filename:         asString(f["filename"]),
		ContentType:      asString(f["contentType"]),
		Size:             asInt(f["size"]),
		UploaderUsername: nestedUsername(f, "createdBy"),
	}

	switch {
	case hasChild(f, "comment") && hasChild(f, "tension"):
		fa.Kind = KindComment
		t, _ := firstChild(f, "tension")
		rcv, ok := firstChild(t, "receiver")
		if !ok {
			return nil, fmt.Errorf("file %s: missing tension receiver", fileid)
		}
		fa.ReceiverNameid = asString(rcv["nameid"])
		fa.ReceiverVisible = model.NodeVisibility(asString(rcv["visibility"]))
	case hasChild(f, "user"):
		fa.Kind = KindUser
	case hasChild(f, "node"):
		fa.Kind = KindNode
		n, _ := firstChild(f, "node")
		fa.NodeNameid = asString(n["nameid"])
		fa.NodeVisibility = model.NodeVisibility(asString(n["visibility"]))
	default:
		return nil, fmt.Errorf("file %s: no anchor (orphan)", fileid)
	}

	return fa, nil
}

// CommentForUpload aggregates the upload-time facts about the parent comment:
// its message (used for the inline-screenshot rewrite) and its author (used
// for the author-only attach rule). The comment is required to belong to
// the named tension; Found=false signals the relation doesn't hold.
type CommentForUpload struct {
	Message        string
	AuthorUsername string
	Found          bool
}

// GetCommentForUpload reads message + author, validating cid belongs to tid.
func (dg Dgraph) GetCommentForUpload(tid, cid string) (CommentForUpload, error) {
	res, err := dg.Meta("getCommentMessage", map[string]string{"tid": tid, "cid": cid})
	if err != nil {
		return CommentForUpload{}, err
	}
	if len(res) == 0 {
		return CommentForUpload{}, nil
	}
	return CommentForUpload{
		Message:        asString(res[0]["message"]),
		AuthorUsername: nestedUsername(res[0], "createdBy"),
		Found:          true,
	}, nil
}

// AddCommentFile inserts a new File anchored to (cid, tid) with embedded=false
// and returns the new fid. The optional inline-screenshot rewrite is applied
// separately via EmbedCommentMessage so the caller can know `fid` before
// composing the new message.
func (dg Dgraph) AddCommentFile(tid, cid, username, filename, contentType, storageKey string, size int64, nowRFC3339 string) (string, error) {
	q := dqlMutations["addCommentFile"]
	res, err := dg.UpsertDql(q, map[string]string{
		"tid":         tid,
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
		return "", fmt.Errorf("addCommentFile: no uid returned")
	}
	return uid, nil
}

// EmbedCommentMessage rewrites Comment.message and flips File.embedded=true.
// Concurrent uploads to the same comment race on Comment.message (last-write-
// wins); File rows are independent and both persist regardless.
func (dg Dgraph) EmbedCommentMessage(cid, fid, newMessage string) error {
	q := dqlMutations["embedCommentMessage"]
	_, err := dg.UpsertDql(q, map[string]string{
		"cid":        cid,
		"fid":        fid,
		"newMessage": escapeNQuad(newMessage),
	})
	return err
}

// ReplaceUserAvatar inserts a new avatar File for username, drops the previous
// one if any, and fires async S3 GC of the previous object (no-op when storage
// is unset). Returns the new fid.
func (dg Dgraph) ReplaceUserAvatar(username, filename, contentType, storageKey string, size int64, nowRFC3339 string) (string, error) {
	q := dqlMutations["replaceUserAvatar"]
	res, err := dg.UpsertDql(q, map[string]string{
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
	fid, ok := res.Uids["f"]
	if !ok || fid == "" {
		return "", fmt.Errorf("replaceUserAvatar: no uid returned")
	}
	if oldKey := oldKeyFromResp(res); oldKey != "" {
		deleteStorageKeysAsync([]string{oldKey})
	}
	return fid, nil
}

// ReplaceNodeAvatar mirrors ReplaceUserAvatar for Node (org) avatars.
func (dg Dgraph) ReplaceNodeAvatar(nameid, username, filename, contentType, storageKey string, size int64, nowRFC3339 string) (string, error) {
	q := dqlMutations["replaceNodeAvatar"]
	res, err := dg.UpsertDql(q, map[string]string{
		"nameid":      nameid,
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
	fid, ok := res.Uids["f"]
	if !ok || fid == "" {
		return "", fmt.Errorf("replaceNodeAvatar: no uid returned")
	}
	if oldKey := oldKeyFromResp(res); oldKey != "" {
		deleteStorageKeysAsync([]string{oldKey})
	}
	return fid, nil
}

// DeleteFile removes a File node by uid and drops the anchor's reverse edge.
// Idempotent.
func (dg Dgraph) DeleteFile(fileid string) error {
	_, err := dg.Meta("deleteFile", map[string]string{"id": fileid})
	return err
}

// deleteStorageKeysAsync fires a goroutine to drop the named S3 objects via
// storage.Global(). No-op when storage isn't configured. Callers use this
// after a cascade delete (comment, tension, user) committed the Dgraph rows
// but still owns the underlying bytes. The 2-min budget covers the worst-case
// tension cascade where a single delete can fan out to dozens of files.
func deleteStorageKeysAsync(keys []string) {
	if len(keys) == 0 {
		return
	}
	cli := storage.Global()
	if cli == nil {
		return
	}
	go func(c *storage.Client, ks []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		for _, k := range ks {
			if k == "" {
				continue
			}
			if err := c.Delete(ctx, k); err != nil {
				log.Printf("Warning: deleteStorageKeysAsync: %s: %v", k, err)
			}
		}
	}(cli, keys)
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

// hasChild reports whether `key` is present and non-empty.
func hasChild(parent map[string]any, key string) bool {
	_, ok := firstChild(parent, key)
	return ok
}

// nestedUsername extracts createdBy > username through the parent map. Keys
// are post-CleanDqlMap.
func nestedUsername(parent map[string]any, edge string) string {
	c, ok := firstChild(parent, edge)
	if !ok {
		return ""
	}
	return asString(c["username"])
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

// oldKeyFromResp pulls the previous avatar's storageKey from the upsert query
// block (returned as an `all` array). Empty when no prior avatar.
func oldKeyFromResp(res *api.Response) string {
	type row struct {
		StorageKey string `json:"File.storageKey"`
	}
	rows, err := DecodeDqlBlock[row](res, "all")
	if err != nil || len(rows) == 0 {
		return ""
	}
	return rows[0].StorageKey
}
