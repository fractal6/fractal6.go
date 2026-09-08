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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/storage"
	. "fractale/fractal6.go/internal/tools"
)

/*
 *
 * This code manage receiving email as HTTP requests from the MTA
 *
 */

var (
	postalWebhookPK  string
	matrixPostalRoom string
	matrixToken      string
	serverDomain     string
)

func init() {
	postalWebhookPK = viper.GetString("mailer.dkim_key")
	matrixPostalRoom = viper.GetString("mailer.matrix_postal_room")
	matrixToken = viper.GetString("mailer.matrix_token")
	serverDomain = viper.GetString("server.domain")
}

type EmailForm struct {
	From               string              `json:"mail_from"`
	To                 string              `json:"rcpt_to"`
	Title              string              `json:"subject"`
	Msg                string              `json:"plain_body"`
	HtmlMsg            string              `json:"html_body"`
	References         string              `json:"references"`
	AttachmentQuantity int                 `json:"attachment_quantity"`
	Attachments        []InboundAttachment `json:"attachments"`
}

// InboundAttachment is the shape Postal ships in the webhook payload. See
// postalserver/postal app/senders/http_sender.rb — there's no Content-ID or
// Content-Disposition, so cid-to-attachment matching is best-effort
// (filename heuristic + document-order). See web/handlers/cid.go.
type InboundAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Data        string `json:"data"` // base64
}

// parseEmailReferences extracts the tension or contract uid an inbound reply
// points at, from the References header set by our outbound mails
// (`<tension/{uid}@domain>` / `<contract/{uid}@domain>`, see web/email/main.go).
// Exactly one of (tid, contractid) is returned; both empty means no reference
// matched.
//
// The header is sender-controlled, so ids are validated as well-formed uids
// here — they end up in DQL uid() roots downstream (getTensionHook,
// getContractHook, addTensionComment, addContractComment).
func parseEmailReferences(references string) (tid string, contractid string, err error) {
	for _, r := range strings.Split(references, " ") {
		l := strings.TrimPrefix(r, "<")
		at := strings.Index(l, "@")
		if at < 0 {
			continue
		}
		if strings.HasPrefix(l, "tension/") {
			tid = l[len("tension/"):at]
			break
		}
		if strings.HasPrefix(l, "contract/") {
			contractid = l[len("contract/"):at]
			break
		}
	}
	if (tid != "" && db.ValidateUids(tid) != nil) ||
		(contractid != "" && db.ValidateUids(contractid) != nil) {
		return "", "", fmt.Errorf("Unknown references")
	}
	return tid, contractid, nil
}

// decodeInboundEmail validates the Postal signature, decodes the webhook body
// and returns the message as markdown with the quoted original stripped. On
// failure the HTTP error has been written.
func decodeInboundEmail(w http.ResponseWriter, r *http.Request) (EmailForm, string, bool) {
	if err := ValidatePostalSignature(r, postalWebhookPK); err != nil {
		http.Error(w, err.Error(), 400)
		return EmailForm{}, "", false
	}
	var form EmailForm
	if !decodeBody(w, r, &form) {
		return EmailForm{}, "", false
	}
	// Prefer html_body to avoid email line-wrapping artifacts in plain_body
	msg := form.Msg
	if form.HtmlMsg != "" {
		if converted, err := HTMLToMarkdown(form.HtmlMsg); err == nil && converted != "" {
			msg = converted
		}
	}
	return form, StripEmailQuote(msg), true
}

// Handle user email responses. Receiving email response from email notifications.
func Notifications(w http.ResponseWriter, r *http.Request) {
	form, msg, ok := decodeInboundEmail(w, r)
	if !ok {
		return
	}
	tid, contractid, err := parseEmailReferences(form.References)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	createdAt := Now()

	switch {
	case tid != "": // tension reply
		e := model.TensionEventCommentPushed
		history := []*model.EventRef{{
			CreatedAt: &createdAt,
			CreatedBy: &model.UserRef{Username: &uctx.Username},
			EventType: &e,
		}}
		ok, _, err := graph.TensionEventHook(uctx, tid, history, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ok {
			http.Error(w, "access denied", 400)
			return
		}
		cid, err := db.GetDB().AddTensionComment(tid, uctx.Username, msg, createdAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rid, err := db.GetDB().GetByUid(tid, "Tension.receiverid")
		if err != nil || rid == nil {
			http.Error(w, fmt.Sprintf("tension receiver: %v", err), 500)
			return
		}
		rootnameid, _ := codec.Nid2rootid(rid.(string))
		processInboundAttachments(r.Context(), uctx, tid, cid, rootnameid, msg, form.Attachments)
		graph.PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: tid, History: history})
	case contractid != "": // contract reply
		contract, err := db.GetDB().GetContractHook(contractid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		ok, err := graph.CanCommentContract(uctx, contract)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if !ok {
			http.Error(w, "access denied", 400)
			return
		}
		cid, err := db.GetDB().AddContractComment(contractid, uctx.Username, msg, createdAt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Files anchor on the contract's tension: that is what /file/<id> authorises against.
		rootnameid, _ := codec.Nid2rootid(contract.Tension.Receiverid)
		processInboundAttachments(r.Context(), uctx, contract.Tension.ID, cid, rootnameid, msg, form.Attachments)
		graph.PublishContractEvent(model.ContractNotif{Uctx: uctx, Tid: contract.Tension.ID, Contract: contract, ContractEvent: model.NewComment})
	default:
		http.Error(w, "Unknown references", 400)
	}
}

// Handle email sent to orga. Convert email to tension.
func Mailing(w http.ResponseWriter, r *http.Request) {
	form, msg, ok := decodeInboundEmail(w, r)
	if !ok {
		return
	}
	uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil {
		http.Error(w, "You need an account on Fractale to send email to organisation, please visit https://fractale.co \n\n"+err.Error(), 400)
		return
	}
	createdAt := Now()
	createdBy := model.User{Username: uctx.Username}

	// Get the nameid of the targeted circle
	toEmail, err := mail.ParseAddress(form.To)
	if err != nil {
		http.Error(w, "RECIPIENT EMAIL NOT FOUND", 400)
		return
	}
	receiverid := strings.ReplaceAll(strings.Split(toEmail.Address, "@")[0], "/", "#")
	filter := `eq(Node.isArchived, false)`
	if ex, _ := db.GetDB().Exists("Node.nameid", receiverid, &filter); !ex {
		http.Error(w, "NAMEID NOT FOUND", 400)
		return
	}

	// Build the tension. Auth, subscription and trace come from CreateTensionHook.
	e := model.TensionEventCreated
	rootnameid, _ := codec.Nid2rootid(receiverid)
	emitterid := codec.MemberIdCodec(rootnameid, uctx.Username)
	history := []*model.EventRef{{
		CreatedAt: &createdAt,
		CreatedBy: &model.UserRef{Username: &uctx.Username},
		EventType: &e,
	}}
	tension := model.Tension{
		CreatedAt:  createdAt,
		CreatedBy:  &createdBy,
		Emitterid:  emitterid,
		Emitter:    &model.Node{Nameid: emitterid},
		Receiverid: receiverid,
		Receiver:   &model.Node{Nameid: receiverid},
		Type:       model.TensionTypeOperational,
		Status:     model.TensionStatusOpen,
		Title:      form.Title,
		Comments: []*model.Comment{{
			CreatedAt: createdAt,
			CreatedBy: &createdBy,
			Message:   msg,
		}},
	}
	tid, err := db.GetDB().Add(*uctx, "tension", StructMap[model.AddTensionInput](tension))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Attachments anchor on the body comment, created inside AddTensionInput (no
	// uid of its own came back). It is the author's only comment at this point.
	attach := func() {
		last, err := db.Meta[struct{ ID string }]("getLastComment", map[string]string{"tid": tid, "username": uctx.Username})
		if err != nil || len(last) == 0 || last[0].ID == "" {
			log.Printf("Warning: inbound attachments getLastComment: %v", err)
			return
		}
		processInboundAttachments(r.Context(), uctx, tid, last[0].ID, rootnameid, msg, form.Attachments)
	}
	if err := graph.CreateTensionHook(uctx, tid, history, attach); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}

// processInboundAttachments writes the Postal-shipped attachment list under
// the freshly-inserted comment, rewrites cid: references in the message in a
// single follow-up upsert, and flips File.embedded on the inline-matched
// fids. Best-effort: any failure (storage unset, bad base64, oversize, S3
// error, DB error) is logged and skipped for that attachment; the comment
// itself stays committed.
//
// Order: parse cid refs → run resolveCIDs over (refs, atts) → write each
// attachment to S3+DB → for inline-matched ones, append a cidRewrite
// resolution carrying the new fid → finally EmbedCommentMessage rewrites
// Comment.message + flips embedded=true on every inline-matched fid in one
// upsert. Sequential within this handler, so no settle coordination needed.
func processInboundAttachments(
	ctx context.Context, uctx *model.UserCtx,
	tid, cid, rootnameid, msg string, atts []InboundAttachment,
) {
	cli := storage.Global()
	if cli == nil || len(atts) == 0 {
		return
	}

	maxCount := ViperPositiveInt("notify.inbound_attachment_max_count", 20)
	perFileBytes := int64(ViperPositiveInt(
		"notify.inbound_attachment_per_file_bytes",
		ViperPositiveInt("storage.max_upload_bytes", 10*1024*1024),
	))
	// Drop re-attached quoted images first, so they can't crowd out real
	// pastes through the count cap nor win the document-order fallback.
	if known, err := db.GetDB().GetTensionFileFingerprints(tid); err != nil {
		log.Printf("Warning: inbound attachments fingerprints: %v", err)
	} else {
		atts = dropKnownAttachments(known, atts)
	}
	if maxCount > 0 && len(atts) > maxCount {
		atts = atts[:maxCount]
	}

	refs := extractCIDRefs(msg)
	pair, resolutions, orphans := resolveCIDs(refs, atts, serverDomain)
	for _, refIdx := range orphans {
		resolutions = append(resolutions, cidResolution{refIdx: refIdx, kind: cidDrop})
	}

	// Invert pair (refIdx->attIdx) into (attIdx->refIdx) for the attachment
	// walk below — we still iterate atts in document order to give the
	// caller predictable storage-key ordering.
	attRefIdx := make(map[int]int, len(pair))
	for refIdx, attIdx := range pair {
		attRefIdx[attIdx] = refIdx
	}

	embedFids := make([]string, 0, len(pair))
	for i, att := range atts {
		raw, err := base64.StdEncoding.DecodeString(att.Data)
		// Release the (potentially large) base64 string immediately —
		// the loop holds `atts` across iterations, so the next 10 MiB
		// payload can be decoded without doubling peak heap.
		atts[i].Data = ""
		if err != nil {
			log.Printf("Warning: inbound attachment %q: bad base64: %v", att.Filename, err)
			continue
		}
		if perFileBytes > 0 && int64(len(raw)) > perFileBytes {
			log.Printf("Warning: inbound attachment %q dropped: %d bytes > %d cap", att.Filename, len(raw), perFileBytes)
			continue
		}
		// Sniff content-type from the bytes — never trust client labels.
		sniffN := 512
		if len(raw) < sniffN {
			sniffN = len(raw)
		}
		ctype := http.DetectContentType(raw[:sniffN])
		safeName := safeFilename(att.Filename)

		fid, err := writeCommentAttachment(ctx, cli,
			rootnameid, tid, cid, uctx.Username,
			safeName, ctype, bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			log.Printf("Warning: inbound attachment %q: %v", att.Filename, err)
			continue
		}

		if refIdx, inline := attRefIdx[i]; inline {
			resolutions = append(resolutions, cidResolution{
				refIdx: refIdx, kind: cidRewrite, fid: fid,
			})
			embedFids = append(embedFids, fid)
		}
	}

	newMsg := applyCIDResolutions(msg, refs, resolutions)
	if newMsg != msg || len(embedFids) > 0 {
		if err := db.GetDB().EmbedCommentMessage(cid, newMsg, embedFids); err != nil {
			log.Printf("Warning: inbound EmbedCommentMessage: %v", err)
		}
	}
}

// Handle Postal WebHook - redirect it to a matrix channel
func PostalWebhook(w http.ResponseWriter, r *http.Request) {
	// Validate WebHook identity
	if err := ValidatePostalSignature(r, postalWebhookPK); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get request string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err = MatrixJsonSend(string(body), matrixPostalRoom, matrixToken)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}
