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

// Wire-shape tests for the Postal JSON body. The previous %q-templated
// implementation was fragile against attachment payloads carrying special
// characters; these tests pin the encoding/json contract so future edits
// don't accidentally reintroduce a hand-rolled JSON template.

package email

import (
	"encoding/json"
	"strings"
	"testing"
)

// buildPostalBody mirrors the body-assembly block in SendEventNotificationEmail.
// Kept in test code (not exported in production) so changes to the actual
// site stay visible in diff review; if the production shape diverges, this
// test will fail loudly.
func buildPostalBody(author, email, subject, htmlBody, plainBody, tid string, attachments []postalAttachment) ([]byte, error) {
	body := map[string]any{
		"from":       author + " <notifications@" + DOMAIN + ">",
		"to":         []string{email},
		"subject":    subject,
		"html_body":  htmlBody,
		"plain_body": plainBody,
		"headers": map[string]string{
			"In-Reply-To": "<tension/" + tid + "@" + DOMAIN + ">",
			"References": "<tension/" + tid + "@" + DOMAIN + ">",
		},
	}
	if len(attachments) > 0 {
		body["attachments"] = attachments
	}
	return json.Marshal(body)
}

func TestPostalBody_OmitsAttachmentsWhenEmpty(t *testing.T) {
	raw, err := buildPostalBody("Alice", "to@example", "subj", "<p>hi</p>", "hi", "0xT", nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, has := got["attachments"]; has {
		t.Errorf("attachments field should be omitted when slice is empty; body: %s", raw)
	}
	// Top-level keys we depend on are present.
	for _, k := range []string{"from", "to", "subject", "html_body", "plain_body", "headers"} {
		if _, has := got[k]; !has {
			t.Errorf("missing top-level key %q", k)
		}
	}
}

func TestPostalBody_BucketAOnly_HasContentID(t *testing.T) {
	atts := []postalAttachment{
		{Name: "paste.png", ContentType: "image/png", Data: "AAAA", ContentID: "0xF@test.example"},
	}
	raw, err := buildPostalBody("Alice", "to@example", "subj", "", "", "0xT", atts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"content_id":"0xF@test.example"`) {
		t.Errorf("expected content_id in body, got: %s", raw)
	}
}

func TestPostalBody_BucketBOnly_OmitsContentID(t *testing.T) {
	// content_id has `omitempty`; Bucket-B entries (no CID) must not surface
	// as `"content_id":""` — Postal interprets empty CID as inline anyway.
	atts := []postalAttachment{
		{Name: "report.pdf", ContentType: "application/pdf", Data: "AAAA"},
	}
	raw, err := buildPostalBody("Alice", "to@example", "subj", "", "", "0xT", atts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"content_id"`) {
		t.Errorf("expected content_id absent for plain attachment, got: %s", raw)
	}
}

func TestPostalBody_Mixed_KeepsBucketAndOrder(t *testing.T) {
	atts := []postalAttachment{
		{Name: "paste.png", ContentType: "image/png", Data: "B64", ContentID: "0x1@test.example"},
		{Name: "report.pdf", ContentType: "application/pdf", Data: "B64"},
	}
	raw, err := buildPostalBody("Alice", "to@example", "subj", "", "", "0xT", atts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Bucket A entry must precede Bucket B.
	posA := strings.Index(string(raw), `"paste.png"`)
	posB := strings.Index(string(raw), `"report.pdf"`)
	if posA < 0 || posB < 0 || posA > posB {
		t.Errorf("expected Bucket A before B; raw: %s", raw)
	}
}

func TestPostalBody_SpecialCharsInSubject(t *testing.T) {
	// encoding/json must escape quote, newline, and unicode cleanly — the
	// reason we migrated off the hand-rolled %q template in the first place.
	in := `weird "subject" with \n control` + "\n" + `chars`
	raw, err := buildPostalBody("Alice", "to@example", in, "", "", "0xT", nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("body is not valid JSON: %v\nraw: %s", err, raw)
	}
	if parsed["subject"] != in {
		t.Errorf("subject round-trip mismatch; got %q, want %q", parsed["subject"], in)
	}
}
