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

// Handle user email responses. Receiving email response from email notifications.
func Notifications(w http.ResponseWriter, r *http.Request) {
	// Validate WebHook identity
	if err := ValidatePostalSignature(r, postalWebhookPK); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get request form
	var form EmailForm
	if !decodeBody(w, r, &form) {
		return
	}

	// Prefer html_body to avoid email line-wrapping artifacts in plain_body
	msg := form.Msg
	if form.HtmlMsg != "" {
		if converted, err := HTMLToMarkdown(form.HtmlMsg); err == nil && converted != "" {
			msg = converted
		}
	}
	// Strip quoted original message from the reply
	msg = StripEmailQuote(msg)

	// Determine where from and to where it goes
	var isTid string
	var isCid string
	for _, r := range strings.Split(form.References, " ") {
		l := strings.TrimPrefix(r, "<")
		if strings.HasPrefix(l, "tension/") {
			isTid = l[8:strings.Index(l, "@")]
			break
		}
		if strings.HasPrefix(l, "contract/") {
			isCid = l[9:strings.Index(l, "@")]
			break
		}

	}

	// Get author
	uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	createdAt := Now()
	createdBy := model.UserRef{Username: &uctx.Username}

	if isTid != "" { // Is a tension reply/comment
		// Build Event
		e := model.TensionEventCommentPushed
		history := []*model.EventRef{{
			CreatedAt: &createdAt,
			CreatedBy: &createdBy,
			EventType: &e,
		}}
		// Check event
		ok, _, err := graph.TensionEventHook(uctx, isTid, history, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if !ok {
			http.Error(w, "access denied", 400)
			return
		}
		// Publish  event
		db.GetDB().Update(db.GetDB().GetRootUctx(), "tension", model.UpdateTensionInput{
			Filter: &model.TensionFilter{ID: []string{isTid}},
			Set: &model.TensionPatch{
				Comments: []*model.CommentRef{{
					CreatedAt: &createdAt,
					CreatedBy: &createdBy,
					Message:   &msg,
				}},
			},
		})

		// Attachments — best-effort; never abort the comment on failure.
		// Reads the freshly-inserted comment's uid + tension rootnameid in
		// the same hop so processInboundAttachments has what it needs to
		// route into the comment-anchor storage layout.
		if len(form.Attachments) > 0 {
			if m, err := db.GetDB().Meta("getLastComment", map[string]string{
				"tid": isTid, "username": uctx.Username,
			}); err == nil && len(m) > 0 {
				cid, _ := m[0]["id"].(string)
				rootnameid, _ := m[0]["rootnameid"].(string)
				if cid != "" && rootnameid != "" {
					processInboundAttachments(
						r.Context(), uctx, isTid, cid, rootnameid, msg, form.Attachments,
					)
				}
			} else if err != nil {
				log.Printf("Warning: inbound attachments getLastComment: %v", err)
			}
		}

		// Publish Notification
		// --
		notif := model.EventNotif{
			Uctx:    uctx,
			Tid:     isTid,
			History: history,
		}
		// Push notification
		if err := graph.PublishTensionEvent(notif); err != nil {
			http.Error(w, "PublishTensionEvent error: "+err.Error(), 500)
			return
		}
	} else if isCid != "" { // If contract reply/comment
		// Build Event
		contract, err := db.GetDB().GetContractHook(isCid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Check  event
		ok, err := graph.HasContractRight(uctx, contract)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if !ok {
			// Check if user is candidate
			for _, c := range contract.Candidates {
				if c.Username == uctx.Username {
					ok = true
					break
				}
			}
			if !ok {
				http.Error(w, "access denied", 400)
				return
			}
		}
		// Publish  event
		db.GetDB().Update(db.GetDB().GetRootUctx(), "contract", model.UpdateContractInput{
			Filter: &model.ContractFilter{ID: []string{isCid}},
			Set: &model.ContractPatch{
				Comments: []*model.CommentRef{{
					CreatedAt: &createdAt,
					CreatedBy: &createdBy,
					Message:   &msg,
				}},
			},
		})

		// Publish Notification
		// --
		notif := model.ContractNotif{
			Uctx:          uctx,
			Tid:           contract.Tension.ID,
			Contract:      contract,
			ContractEvent: model.NewComment,
		}
		// Push notification
		if err := graph.PublishContractEvent(notif); err != nil {
			http.Error(w, "PublishContractEvent error: "+err.Error(), 500)
			return
		}
	} else {
		// In every other case, it returns an error.
		http.Error(w, "Unknown references", 400)
		return
	}
}

// Handle email sent to orga. Convert email to tension.
func Mailing(w http.ResponseWriter, r *http.Request) {
	// Validate WebHook identity
	if err := ValidatePostalSignature(r, postalWebhookPK); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get request form
	var form EmailForm
	if !decodeBody(w, r, &form) {
		return
	}

	// Prefer html_body to avoid email line-wrapping artifacts in plain_body
	msg := form.Msg
	if form.HtmlMsg != "" {
		if converted, err := HTMLToMarkdown(form.HtmlMsg); err == nil && converted != "" {
			msg = converted
		}
	}
	// Strip quoted original message from the reply
	msg = StripEmailQuote(msg)

	// Get author
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

	// Build the tension
	e := model.TensionEventCreated
	rootnameid, _ := codec.Nid2rootid(receiverid)
	emitterid := codec.MemberIdCodec(rootnameid, uctx.Username)
	event := model.Event{
		CreatedAt: createdAt,
		CreatedBy: &createdBy,
		EventType: e,
	}
	tension := model.Tension{
		CreatedAt:  createdAt,
		CreatedBy:  &model.User{Username: uctx.Username},
		Emitterid:  emitterid,
		Emitter:    &model.Node{Nameid: emitterid},
		Receiverid: receiverid,
		Receiver:   &model.Node{Nameid: receiverid},
		Type:       model.TensionTypeOperational,
		Status:     model.TensionStatusOpen,
		Title:      form.Title,
		Comments: []*model.Comment{
			{
				CreatedAt: createdAt,
				CreatedBy: &createdBy,
				Message:   msg,
			},
		},
		Subscribers: []*model.User{{Username: uctx.Username}},
	}

	// Verify author can create tension
	eventRef := StructMap[model.EventRef](event)
	ok, _, err := graph.ProcessEvent(uctx, &tension, &eventRef, nil, nil, true, false)
	if !ok || err != nil {
		http.Error(w, "NOT AUTHORIZED TO CREATE TENSION HERE", 400)
		return
	}

	// Create tension
	tensionInput := StructMap[model.AddTensionInput](tension)
	tid, err := db.GetDB().Add(*uctx, "tension", tensionInput)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Publish Notification
	// --
	notif := model.EventNotif{
		Uctx:    uctx,
		Tid:     tid,
		History: []*model.EventRef{&eventRef},
	}
	// Push notification
	if err := graph.PublishTensionEvent(notif); err != nil {
		http.Error(w, "PublishTensionEvent: "+err.Error(), 500)
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
// upsert. Sequential within this handler, so no upload-gate plumbing.
func processInboundAttachments(
	ctx context.Context, uctx *model.UserCtx,
	tid, cid, rootnameid, msg string, atts []InboundAttachment,
) {
	cli := storage.Global()
	if cli == nil {
		return
	}

	maxCount := ViperPositiveInt("notify.inbound_attachment_max_count", 20)
	perFileBytes := int64(ViperPositiveInt(
		"notify.inbound_attachment_per_file_bytes",
		ViperPositiveInt("storage.max_upload_bytes", 10*1024*1024),
	))
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
