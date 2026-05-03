//go:build integration

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

// End-to-end coverage for /file/upload, GET /file/<id>, DELETE /file/<id>, and
// the comment-delete cleanup hook. Storage is the MinIO container from
// docker-compose.test.yml; the bucket is bootstrapped by cmd/testsetup.
//
// Comments are resolved by Post.message marker (see seed.nq) so tests don't
// depend on uids that change every DropAll.

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
)

// --- Helpers ---

// resolveCommentByMessage looks up a seeded comment uid by its unique
// Post.message marker. Fails the test if the comment is not present —
// without it the rest of the file-attachment suite is meaningless.
func resolveCommentByMessage(t *testing.T, marker string) string {
	t.Helper()
	ids, err := db.GetDB().GetIDs("Post.message", marker, nil, nil)
	if err != nil {
		t.Fatalf("resolveCommentByMessage(%q): %v", marker, err)
	}
	if len(ids) == 0 {
		t.Fatalf("resolveCommentByMessage(%q): no comment found — re-run cmd/testsetup", marker)
	}
	return ids[0]
}

// uploadFile builds a multipart POST /file/upload request and runs it through
// the test router. The body field name and form field names match the handler
// in web/handlers/files.go.
func uploadFile(t *testing.T, commentID, filename, declaredContentType string, body []byte, jwt *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("comment_id", commentID); err != nil {
		t.Fatalf("multipart write field: %v", err)
	}
	// Build the file part with an explicit Content-Type so we can verify the
	// server's MIME-sniff overrides the client claim.
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{
		fmt.Sprintf(`form-data; name="file"; filename=%q`, filename),
	}
	if declaredContentType != "" {
		hdr["Content-Type"] = []string{declaredContentType}
	}
	part, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatalf("multipart create part: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("multipart write body: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}

	req := httptest.NewRequest("POST", "/file/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if jwt != nil {
		req.AddCookie(jwt)
	}
	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	return rr
}

// uploadResp is the JSON shape returned by POST /file/upload.
type uploadResp struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

func decodeUpload(t *testing.T, rr *httptest.ResponseRecorder) uploadResp {
	t.Helper()
	var out uploadResp
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode upload response: %v (body: %s)", err, rr.Body.String())
	}
	if out.ID == "" {
		t.Fatalf("upload response missing id: %s", rr.Body.String())
	}
	return out
}

// storageKeyOf reads File.storageKey straight from Dgraph. Tests use this to
// verify object-level state (rollback, cleanup) on MinIO.
func storageKeyOf(t *testing.T, fileID string) string {
	t.Helper()
	v, err := db.GetDB().GetFieldById(fileID, "File.storageKey")
	if err != nil {
		t.Fatalf("GetFieldById(File.storageKey): %v", err)
	}
	s, _ := v.(string)
	if s == "" {
		t.Fatalf("File.storageKey is empty for %s", fileID)
	}
	return s
}

// objectExists asks MinIO directly. Used to assert post-conditions.
func objectExists(t *testing.T, key string) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, err := testStorageCli.Exists(ctx, key)
	if err != nil {
		t.Fatalf("storage Exists(%q): %v", key, err)
	}
	return ok
}

// purgeFile cleans a File node + its S3 object so tests stay independent.
// Best-effort: errors are reported but don't fail the test.
func purgeFile(t *testing.T, fileID, storageKey string) {
	t.Helper()
	if storageKey != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = testStorageCli.Delete(ctx, storageKey)
		cancel()
	}
	if fileID != "" {
		if err := db.GetDB().DeleteFile(fileID); err != nil {
			t.Logf("purgeFile: db.DeleteFile(%s) failed: %v", fileID, err)
		}
	}
}

// --- Upload tests ---

func TestFileUpload_AsAuthor_Succeeds(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)

	rr := uploadFile(t, cid, "hello.png", "image/png", pngBytes(), jwt)
	requireStatus(t, rr, http.StatusOK)

	resp := decodeUpload(t, rr)
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	if resp.URL != "/file/"+resp.ID {
		t.Errorf("upload url = %q, want /file/%s", resp.URL, resp.ID)
	}
	if !strings.HasPrefix(resp.ContentType, "image/png") {
		t.Errorf("contentType = %q, want image/png", resp.ContentType)
	}
	if resp.Filename != "hello.png" {
		t.Errorf("filename = %q, want hello.png", resp.Filename)
	}

	key := storageKeyOf(t, resp.ID)
	if !strings.HasPrefix(key, "comments/"+cid+"/") {
		t.Errorf("storageKey = %q, want prefix comments/%s/", key, cid)
	}
	if !objectExists(t, key) {
		t.Errorf("expected object %q to exist in MinIO", key)
	}
}

func TestFileUpload_AsNonAuthor_Forbidden(t *testing.T) {
	// testuser2's comment, attempting upload as testuser → 403.
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser2)

	rr := uploadFile(t, cid, "x.txt", "text/plain", []byte("nope"), jwt)
	requireStatus(t, rr, http.StatusForbidden)
}

func TestFileUpload_Unauthenticated_401(t *testing.T) {
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)

	rr := uploadFile(t, cid, "x.txt", "text/plain", []byte("hi"), nil)
	requireStatus(t, rr, http.StatusUnauthorized)
}

func TestFileUpload_MissingCommentId_400(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)

	rr := uploadFile(t, "", "x.txt", "text/plain", []byte("hi"), jwt)
	requireStatus(t, rr, http.StatusBadRequest)
}

func TestFileUpload_NonExistentComment_404(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)

	// 0xdead is a valid uid format but should not match any comment.
	rr := uploadFile(t, "0xdead", "x.txt", "text/plain", []byte("hi"), jwt)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestFileUpload_TooLarge_400(t *testing.T) {
	// Tighten the upload limit just for this test.
	prev := viper.GetInt("storage.max_upload_bytes")
	viper.Set("storage.max_upload_bytes", 16)
	defer viper.Set("storage.max_upload_bytes", prev)

	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)

	rr := uploadFile(t, cid, "big.bin", "application/octet-stream", bytes.Repeat([]byte("A"), 1024), jwt)
	requireStatus(t, rr, http.StatusBadRequest)
}

func TestFileUpload_MIMESniffOverridesClientType(t *testing.T) {
	// Client claims image/png but body is HTML — server must persist the
	// sniffed type (text/html), and the GET must serve `attachment` so the
	// browser doesn't render the script.
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)

	htmlBody := []byte(`<html><body><script>alert(1)</script></body></html>`)
	rr := uploadFile(t, cid, "evil.png", "image/png", htmlBody, jwt)
	requireStatus(t, rr, http.StatusOK)

	resp := decodeUpload(t, rr)
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	if !strings.HasPrefix(resp.ContentType, "text/html") {
		t.Errorf("contentType = %q, want text/html (sniff must override client claim)", resp.ContentType)
	}

	getRR := getFile(t, resp.ID, jwt)
	requireStatus(t, getRR, http.StatusFound)
	dispo := dispositionFromLocation(t, getRR)
	if !strings.HasPrefix(dispo, "attachment") {
		t.Errorf("Content-Disposition = %q, want attachment for text/html", dispo)
	}
}

// --- GET tests ---

func TestFileGet_Public_Anonymous_302WithHeaders(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	resp := decodeUpload(t, uploadFile(t, cid, "ok.png", "image/png", pngBytes(), jwt))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	// Anonymous fetch (test-org tension is Public).
	rr := getFile(t, resp.ID, nil)
	requireStatus(t, rr, http.StatusFound)

	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "private, no-store")
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("missing Location header")
	}
	// The presigned URL must reference the configured MinIO endpoint and the
	// stored key (Dgraph uid is in the key prefix).
	if !strings.Contains(loc, testutil.MinioAddr) {
		t.Errorf("Location %q does not reference MinIO endpoint %q", loc, testutil.MinioAddr)
	}
	if !strings.Contains(loc, "X-Amz-Signature=") {
		t.Errorf("Location %q is not a presigned URL", loc)
	}
}

// TestFileGet_BytesRoundTrip follows the 302 to MinIO and verifies the bytes,
// MIME type, and Content-Disposition all survive the presign hand-off. The
// other GET tests stop at the redirect, which doesn't catch presign-signature
// errors, response-content-disposition rejection, or bucket-policy issues —
// real concerns when swapping MinIO for Garage.
func TestFileGet_BytesRoundTrip(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	body := pngBytes()
	resp := decodeUpload(t, uploadFile(t, cid, "rt.png", "image/png", body, jwt))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	rr := getFile(t, resp.ID, jwt)
	requireStatus(t, rr, http.StatusFound)
	loc := rr.Header().Get("Location")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", loc, nil)
	if err != nil {
		t.Fatalf("build presigned request: %v", err)
	}
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch presigned URL: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		preview, _ := io.ReadAll(httpResp.Body)
		t.Fatalf("MinIO returned %d for presigned URL %q\nbody: %s",
			httpResp.StatusCode, loc, string(preview))
	}
	got, err := io.ReadAll(httpResp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body length mismatch: got %d bytes, want %d", len(got), len(body))
	}
	if ct := httpResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	// The handler set response-content-disposition=inline; MinIO must echo it
	// back as the Content-Disposition response header.
	if dispo := httpResp.Header.Get("Content-Disposition"); !strings.HasPrefix(dispo, "inline") {
		t.Errorf("Content-Disposition = %q, want inline (set by response-content-disposition param)", dispo)
	}
}

func TestFileGet_Private_MemberSees(t *testing.T) {
	// Author = testuser2, on sec-org#private-circle. Upload as the author.
	authorJWT := loginAs(testutil.TestUser2, testutil.TestPassword2)
	cid := resolveCommentByMessage(t, testutil.FileTestPrivateCommentByUser2)
	resp := decodeUpload(t, uploadFile(t, cid, "p.png", "image/png", pngBytes(), authorJWT))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	// testuser is a Member of sec-org → may see Private circles.
	memberJWT := loginAs(testutil.TestUser, testutil.TestPassword)
	rr := getFile(t, resp.ID, memberJWT)
	requireStatus(t, rr, http.StatusFound)
}

func TestFileGet_Private_Anonymous_404(t *testing.T) {
	// Same setup as above but anonymous client must get 404 (existence hidden).
	authorJWT := loginAs(testutil.TestUser2, testutil.TestPassword2)
	cid := resolveCommentByMessage(t, testutil.FileTestPrivateCommentByUser2)
	resp := decodeUpload(t, uploadFile(t, cid, "p.png", "image/png", pngBytes(), authorJWT))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	rr := getFile(t, resp.ID, nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestFileGet_NonExistentId_404(t *testing.T) {
	rr := getFile(t, "0xdead", nil)
	requireStatus(t, rr, http.StatusNotFound)
}

func TestFileGet_PNG_AllowsInline(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	resp := decodeUpload(t, uploadFile(t, cid, "img.png", "image/png", pngBytes(), jwt))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	rr := getFile(t, resp.ID, jwt)
	requireStatus(t, rr, http.StatusFound)
	if dispo := dispositionFromLocation(t, rr); !strings.HasPrefix(dispo, "inline") {
		t.Errorf("Content-Disposition = %q, want inline for image/png", dispo)
	}
}

func TestFileGet_SVG_ForcesAttachment(t *testing.T) {
	// SVG isn't on the inline-safe allowlist (XSS via <script>) — must download.
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`)
	resp := decodeUpload(t, uploadFile(t, cid, "x.svg", "image/svg+xml", svg, jwt))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	rr := getFile(t, resp.ID, jwt)
	requireStatus(t, rr, http.StatusFound)
	if dispo := dispositionFromLocation(t, rr); !strings.HasPrefix(dispo, "attachment") {
		t.Errorf("Content-Disposition = %q, want attachment for SVG", dispo)
	}
}

// --- DELETE tests ---

func TestFileDelete_AsAuthor_204_AndObjectGone(t *testing.T) {
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	resp := decodeUpload(t, uploadFile(t, cid, "del.png", "image/png", pngBytes(), jwt))
	key := storageKeyOf(t, resp.ID)

	rr := doRequest("DELETE", "/file/"+resp.ID, nil, jwt)
	requireStatus(t, rr, http.StatusNoContent)

	// Subsequent GET → 404 (DB row dropped).
	if g := getFile(t, resp.ID, jwt); g.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE: status %d, want 404", g.Code)
	}
	// Object gone from MinIO.
	if objectExists(t, key) {
		t.Errorf("expected object %q to be removed from MinIO", key)
	}
}

func TestFileDelete_AsNonAuthor_404(t *testing.T) {
	// Upload as testuser, then attempt DELETE as testuser2 → 404 (not 403,
	// to avoid leaking existence).
	authorJWT := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	resp := decodeUpload(t, uploadFile(t, cid, "x.png", "image/png", pngBytes(), authorJWT))
	defer purgeFile(t, resp.ID, storageKeyOf(t, resp.ID))

	intruderJWT := loginAs(testutil.TestUser2, testutil.TestPassword2)
	rr := doRequest("DELETE", "/file/"+resp.ID, nil, intruderJWT)
	requireStatus(t, rr, http.StatusNotFound)
}

// --- Comment-delete cleanup hook ---

func TestCommentDelete_GCsAttachments(t *testing.T) {
	// Create a fresh comment authored by testuser (so RemoveComment will accept
	// the call) and attach a file. Then remove the comment via a direct DQL
	// mutation simulation through graph.RemoveComment is in another package;
	// instead exercise db.CleanupCommentFiles directly with the same wiring.
	jwt := loginAs(testutil.TestUser, testutil.TestPassword)
	cid := resolveCommentByMessage(t, testutil.FileTestPublicCommentByUser1)
	resp := decodeUpload(t, uploadFile(t, cid, "gc.png", "image/png", pngBytes(), jwt))
	key := storageKeyOf(t, resp.ID)

	// Sanity.
	if !objectExists(t, key) {
		t.Fatalf("precondition: object %q should exist before cleanup", key)
	}

	// Run the same cleanup the comment-delete event path runs. We do NOT
	// actually drop the seeded comment (other tests rely on it) — purge the
	// File ourselves afterwards. CleanupCommentFiles is a no-op on the DB
	// side; it only removes S3 objects.
	if err := db.GetDB().CleanupCommentFiles(cid, testStorageCli); err != nil {
		t.Fatalf("CleanupCommentFiles: %v", err)
	}
	if objectExists(t, key) {
		t.Errorf("expected object %q to be removed by CleanupCommentFiles", key)
	}

	// File node still exists in Dgraph (CleanupCommentFiles is S3-only).
	// Delete it so the seeded comment is back to a clean state.
	_ = db.GetDB().DeleteFile(resp.ID)
}

// --- internal helpers (need access to handler internals only via routing) ---

func getFile(t *testing.T, id string, jwt *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/file/"+id, nil)
	if jwt != nil {
		req.AddCookie(jwt)
	}
	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	return rr
}

// dispositionFromLocation parses the response-content-disposition query
// parameter from the presigned redirect URL. The handler attaches it so the
// storage backend stamps it on the served response.
func dispositionFromLocation(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	loc := rr.Header().Get("Location")
	if loc == "" {
		t.Fatal("missing Location header in redirect")
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location %q: %v", loc, err)
	}
	dispo := u.Query().Get("response-content-disposition")
	if dispo == "" {
		t.Fatalf("Location %q has no response-content-disposition param", loc)
	}
	return dispo
}

// pngBytes returns a minimal valid 1x1 PNG so http.DetectContentType yields
// image/png and presign serves it cleanly.
func pngBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}

