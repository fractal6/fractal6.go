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

package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/spf13/viper"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/internal/tools"
)

// Inline parsers without RawHTMLParser: text like "<tension>" stays literal
// (escaped to &lt;tension&gt;) instead of being parsed as raw HTML and dropped.
// Bracketed autolinks (<https://...>, <a@b>) keep working via AutoLinkParser.
var inlineParsersNoRawHTML = []util.PrioritizedValue{
	util.Prioritized(parser.NewCodeSpanParser(), 100),
	util.Prioritized(parser.NewLinkParser(), 200),
	util.Prioritized(parser.NewAutoLinkParser(), 300),
	util.Prioritized(parser.NewEmphasisParser(), 500),
}

var md goldmark.Markdown = goldmark.New(
	goldmark.WithParser(parser.NewParser(
		parser.WithBlockParsers(parser.DefaultBlockParsers()...),
		parser.WithInlineParsers(inlineParsersNoRawHTML...),
		parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
	)),
	goldmark.WithExtensions(
		extension.GFM,
		&detailsExtension{},
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

// sanitizer extends bluemonday's UGCPolicy with <details>/<summary> support
// and allows the inline-styled wrapper div used by the details extension.
// `cid:` is added to the URL scheme allowlist so the inline-image rewriter
// (rewriteFileImgs) can swap `<img src="/file/<id>">` to `<img src="cid:...">`
// without bluemonday stripping the src attribute.
var sanitizer = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowElements("details", "summary")
	p.AllowAttrs("open").OnElements("details")
	p.AllowAttrs("style").OnElements("div", "span", "details")
	p.AllowURLSchemes("cid", "http", "https", "mailto")
	return p
}()

var (
	emailSecret     string
	emailUrl        string
	maintainerEmail string
	DOMAIN          string
)

var mailerHTTPClient = func() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &http.Client{Transport: transport, Timeout: 60 * time.Second}
}()

func init() {
	emailUrl = viper.GetString("mailer.email_api_url")
	emailSecret = viper.GetString("mailer.email_api_key")
	if emailUrl == "" {
		emailUrl = os.Getenv("EMAIL_API_URL")
	}
	if emailSecret == "" {
		emailSecret = os.Getenv("EMAIL_API_KEY")
	}
	// Missing url/key is reported by the startup healthcheck (cmd/health.go).

	DOMAIN = viper.GetString("server.domain")
	maintainerEmail = viper.GetString("mailer.admin_email")
}

// SetTestConfig overrides email configuration for integration tests.
func SetTestConfig(url, secret string) {
	emailUrl = url
	emailSecret = secret
}

// IsConfigured reports whether the mailer API url and key are both set.
func IsConfigured() bool { return emailUrl != "" && emailSecret != "" }

// Ping checks the mailer API is reachable and the key accepted, by POSTing an
// empty payload: no recipient means nothing is ever sent, but a bad key still
// comes back as InvalidServerAPIKey. Postal answers 200 with a JSON status
// field, hence the body check. Startup healthcheck, see cmd/health.go.
func Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, emailUrl, bytes.NewBufferString("{}"))
	if err != nil {
		return fmt.Errorf("email: bad api url %q: %w", emailUrl, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-API-Key", emailSecret)

	resp, err := mailerHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("email: %s unreachable: %w", emailUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("email: %s: %s", emailUrl, resp.Status)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if strings.Contains(string(body), "InvalidServerAPIKey") {
		return fmt.Errorf("email: %s: api key rejected", emailUrl)
	}
	return nil
}

// sendPostal POSTs a JSON payload to the mailer API. Postal replies HTTP 200
// even for refused messages, with {"status": "error", ...} in the body, so
// the body-level status is checked too (absent status — e.g. test mocks — passes).
func sendPostal(body []byte) error {
	req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-API-Key", emailSecret)

	resp, err := mailerHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return fmt.Errorf("postal: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var r struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(raw, &r) == nil && r.Status != "" && r.Status != "success" {
		return fmt.Errorf("postal: status %q: %s", r.Status, strings.TrimSpace(string(raw)))
	}
	return nil
}

//
// System Email
//

// Send an email with a http request to the email server API to the admin email.
func SendMaintainerEmail(subject, body string) error {
	if maintainerEmail == "" {
		return nil
	}

	body = fmt.Sprintf(`{
        "from": "%s <alert@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "%s",
        "plain_body": "%s"
    }`, "Fractal6 Alert", maintainerEmail, subject, tools.QuoteString(body))
	// Other fields: http://apiv1.postalserver.io/controllers/send/message

	return sendPostal([]byte(body))
}

//
// Login Email
//

// Send an verification email for signup
func SendVerificationEmail(email, token string) error {
	url_redirect := fmt.Sprintf("https://"+DOMAIN+"/verification?email_token=%s", token)

	content := fmt.Sprintf(`<html>
	<head>
	<title>Activate your Fractale account</title>
	<meta charset="utf-8">
	</head>
	<body>
	<p>To activate your account at <b>`+DOMAIN+`</b>, click the link below (valid one hour):</p>
	<a href="%s">%s</a>
	<br><br>—<br>
	<small>If you are not at the origin of this request, please ignore this mail.</small>
	</body>
    </html>`, url_redirect, url_redirect)

	plainContent, _ := tools.HTMLToMarkdown(content)
	body := fmt.Sprintf(`{
        "from": "Fractale <noreply@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "Activate your account at `+DOMAIN+`",
        "html_body": "%s",
        "plain_body": "%s"
    }`, email, tools.CleanString(content, true), tools.QuoteString(plainContent))

	return sendPostal([]byte(body))
}

// Send an email to reset a user password
func SendResetEmail(email, token string) error {
	url_redirect := fmt.Sprintf("https://"+DOMAIN+"/password-reset?x=%s", token)

	content := fmt.Sprintf(`<html>
	<head>
	<title>Reset your Fractale Password</title>
	<meta charset="utf-8">
	</head>
	<body>
	<h2>Forgot your password?</h2>
	<p>To reset your password at <b>`+DOMAIN+`</b>, click the link below (valid one hour):</p>
	<a href="%s">%s</a>
	<br><br>—<br>
	<small>If you are not at the origin of this request, please ignore this mail.</small>
	</body>
    </html>`, url_redirect, url_redirect)

	plainContent, _ := tools.HTMLToMarkdown(content)
	body := fmt.Sprintf(`{
        "from": "Fractale <noreply@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "Reset your password at `+DOMAIN+`",
        "html_body": "%s",
        "plain_body": "%s"
    }`, email, tools.CleanString(content, true), tools.QuoteString(plainContent))

	return sendPostal([]byte(body))
}

//
// REST API Email
//

func SendOwnerGrantedEmail(username, nameid, orgName string) error {
	var email string
	if x, err := db.GetDB().GetByEq("User.username", username, "User.email"); err != nil {
		return err
	} else {
		email = x.(string)
	}
	orgUrl := fmt.Sprintf("https://"+DOMAIN+"/o/%s", nameid)
	memberUrl := fmt.Sprintf("https://"+DOMAIN+"/m/%s", nameid)

	content := fmt.Sprintf(`<html>
	<head>
	<meta charset="utf-8">
	</head>
	<body>
    <br>
    <p>You have been granted owner of the <a href="%s">%s (%s)</a> organisation.</p><br>

    If this was a mistake you can <a href="%s">leave this role</a>.<br><br>

    <i>The Fractale Team</i>
	</body>
    </html>`, orgUrl, orgName, nameid, memberUrl)

	plainContent, _ := tools.HTMLToMarkdown(content)
	body := fmt.Sprintf(`{
        "from": "Fractale <noreply@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "Ownership of %s was granted",
        "html_body": "%s",
        "plain_body": "%s"
    }`, email, orgName, tools.CleanString(content, true), tools.QuoteString(plainContent))

	return sendPostal([]byte(body))
}

//
// Graph/Structured email
//

// FetchEventAttachments loads the files of the comment rendered by the email
// (Created / CommentPushed only: state-change events like Closed / UserJoined
// never render a comment body, so attaching files to those would be a spurious
// side-channel in the recipient's mail UI).
func FetchEventAttachments(notif model.EventNotif) *Attachments {
	if !notif.HasEvent(model.TensionEventCreated) && !notif.HasEvent(model.TensionEventCommentPushed) {
		return &Attachments{}
	}
	commentFiles, _ := db.GetDB().GetLastCommentFiles(notif.Tid, notif.Uctx.Username)
	// Bound the S3 fetches so a hung storage backend can't stall the daemon.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	att := buildAttachments(ctx, storage.Global(), fromDBFiles(commentFiles))
	return &att
}

// FetchContractAttachments loads the files on the contract's latest comment
// authored by the actor.
func FetchContractAttachments(notif model.ContractNotif) *Attachments {
	if notif.Contract == nil {
		return &Attachments{}
	}
	cfiles, _ := db.GetDB().GetLastContractCommentFiles(notif.Contract.ID, notif.Uctx.Username)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	att := buildAttachments(ctx, storage.Global(), fromDBFiles(cfiles))
	return &att
}

func SendEventNotificationEmail(ui model.UserNotifInfo, notif model.EventNotif, att *Attachments) error {
	// Get inputs
	var err error
	var url_redirect string
	var subject string
	var author string
	var payload string
	var recv string = strings.ReplaceAll(notif.Receiverid, "#", "/")
	var title string = notif.Title
	var message string = notif.Msg

	// Attachments are prefetched once per notification (FetchEventAttachments).
	if att == nil {
		att = &Attachments{}
	}
	// Recipient email
	var email string = ui.User.Email
	if email == "" {
		if x, err := db.GetDB().GetByEq("User.username", ui.User.Username, "User.email"); err != nil {
			return err
		} else {
			email = x.(string)
		}
	}
	// Author
	if notif.Uctx.Name != nil {
		author = fmt.Sprintf("%s (@%s)", *notif.Uctx.Name, notif.Uctx.Username)
	} else {
		author = "@" + notif.Uctx.Username
	}

	var type_hint string
	if ui.Reason == model.ReasonIsAlert {
		type_hint = ""
	}

	// Redirect Url
	url_redirect = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s", notif.Rootnameid, notif.Tid)
	vars := []string{}
	if ui.Eid != "" {
		// Eid var is used to mark the event as read from the client.
		vars = append(vars, fmt.Sprintf("eid=%s", ui.Eid))
	}
	if createdAt := notif.GetCreatedAt(); createdAt != "" {
		vars = append(vars, fmt.Sprintf("goto=%s", createdAt))
	}
	if len(vars) > 0 {
		url_redirect += "?" + strings.Join(vars, "&")
	}

	// Build body
	if notif.HasEvent(model.TensionEventCreated) { // Tension added
		subject = fmt.Sprintf("[%s]%s %s", recv, type_hint, title)

		// Add eventual comment
		if message == "" {
			payload = "<i>No message provided.</i><br><br>"
		} else {
			// Convert markdown to Html, then rewrite /file/<id> img tags to
			// cid:<id>@DOMAIN for inline attachments and absolute URLs for
			// the rest. Sanitisation runs AFTER the rewrite so bluemonday
			// validates the final shape (cid scheme is allowlisted above).
			var buf bytes.Buffer
			if err = md.Convert([]byte(message), &buf); err != nil {
				return err
			}
			rendered := rewriteFileImgs(buf.String(), att.inlineByID)
			payload = sanitizer.Sanitize(rendered)
		}

	} else { // Tension updated
		subject = fmt.Sprintf("Re: [%s]%s %s", recv, type_hint, title)
		auto_msg := ""
		comment := ""

		// Add automatic message
		if notif.HasEvent(model.TensionEventClosed) {
			auto_msg = fmt.Sprintf(`Closed <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
		} else if notif.HasEvent(model.TensionEventReopened) {
			auto_msg = fmt.Sprintf(`Reopened <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
		} else if notif.HasEvent(model.TensionEventBlobPushed) {
			auto_msg = fmt.Sprintf(`Mandate updated <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
		} else if notif.HasEvent(model.TensionEventUserJoined) {
			u := notif.GetNewUser()
			itsYou := u == ui.User.Username
			if x, _ := db.GetDB().GetByEq("User.username", u, "User.name"); x != nil {
				u = fmt.Sprintf("%s (@%s)", x.(string), u)
			}
			if itsYou {
				// Notification happens in contract_op.VoteEventHook function since we never go here
				// (except if the user has subscrided to the anchor tensionn which is unlikelly).
				return nil
			} else {
				auto_msg = fmt.Sprintf(`%s joined this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
			}

		} else if notif.HasEvent(model.TensionEventUserLeft) {
			u := notif.GetExUser()
			if x, _ := db.GetDB().GetByEq("User.username", u, "User.name"); x != nil {
				u = fmt.Sprintf("%s (@%s)", x.(string), u)
			}
			anchorTid, _ := db.GetDB().GetByEq("Node.nameid", notif.Receiverid, "Node.source", "Blob.tension", "uid")
			if anchorTid != nil && anchorTid.(string) == notif.Tid {
				switch model.RoleType(notif.GetExRoleType()) {
				case model.RoleTypeGuest:
					auto_msg = fmt.Sprintf(`%s left this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
				case model.RoleTypeOwner:
					auto_msg = fmt.Sprintf(`%s left his owner role in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
				default:
					return fmt.Errorf("unhandled ex-role type %q on UserLeft event email (tension %s)", notif.GetExRoleType(), notif.Tid)
				}
			} else {
				auto_msg = fmt.Sprintf(`%s left his role in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
			}
		} else if notif.HasEvent(model.TensionEventMemberLinked) {
			u := notif.GetNewUser()
			itsYou := u == ui.User.Username
			if x, _ := db.GetDB().GetByEq("User.username", u, "User.name"); x != nil {
				u = fmt.Sprintf("%s (@%s)", x.(string), u)
			}
			if itsYou {
				auto_msg = fmt.Sprintf(`Hi %s,<br><br>Congratulation, your application has been accepted in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
			} else {
				auto_msg = fmt.Sprintf(`%s is lead link in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
			}
		} else if notif.HasEvent(model.TensionEventMemberUnlinked) {
			u := notif.GetExUser()
			itsYou := u == ui.User.Username
			if x, _ := db.GetDB().GetByEq("User.username", u, "User.name"); x != nil {
				u = fmt.Sprintf("%s (@%s)", x.(string), u)
			}
			anchorTid, _ := db.GetDB().GetByEq("Node.nameid", notif.Receiverid, "Node.source", "Blob.tension", "uid")
			if anchorTid != nil && anchorTid.(string) == notif.Tid {
				if itsYou {
					auto_msg = fmt.Sprintf(`You have been removed from this organisation in <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
				} else {
					auto_msg = fmt.Sprintf(`%s has been removed from this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
				}
			} else {
				if itsYou {
					auto_msg = fmt.Sprintf(`You have has been unlinked from this role in <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
				} else {
					auto_msg = fmt.Sprintf(`%s has been unlinked from this role in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
				}
			}
		}

		// Add eventual comment
		if notif.HasEvent(model.TensionEventCommentPushed) && message != "" {
			// Convert markdown to Html, then rewrite /file/<id> img tags as for
			// the "Created" path above.
			var buf bytes.Buffer
			if err = md.Convert([]byte(message), &buf); err != nil {
				return err
			}
			rendered := rewriteFileImgs(buf.String(), att.inlineByID)
			comment = sanitizer.Sanitize(rendered)
		}

		if comment != "" {
			payload += comment
		}

		if auto_msg != "" {
			if payload != "" {
				payload += "—<br>"
			}
			payload += auto_msg + "<br>"
		}
	}

	// Append the plain-attachment footer (Bucket B) so the recipient sees a
	// click-through list even when their mail client hides Postal
	// attachments behind a paperclip. Bucket A files (inline CID) are
	// already visible in-body and intentionally NOT listed here.
	if footer := renderAttachmentFooter(att.footer); footer != "" {
		payload += footer
	}

	// Add footer
	var url_unsubscribe string
	var url_leave string
	payload += fmt.Sprintf(`—
    <div style="color:#666;font-size:small">You are receiving this because %s.<br>
    <a href="%s">View it on Fractale</a>`, ui.Reason.ToText(), url_redirect)
	switch ui.Reason {
	case model.ReasonIsSubscriber:
		url_unsubscribe = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s?unsubscribe=email", notif.Rootnameid, notif.Tid)
		payload += fmt.Sprintf(`, reply to this email directly, or <a href="%s">unsubscribe</a>.</div>`, url_unsubscribe)
	case model.ReasonIsAnnouncement:
		url_unsubscribe = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s?unwatch=email", notif.Rootnameid, notif.Tid)
		payload += fmt.Sprintf(`, or <a href="%s">unsubscribe</a> from all announcements for this organisation.</div>`, url_unsubscribe)
	case model.ReasonIsAlert:
		url_leave = fmt.Sprintf("https://"+DOMAIN+"/m/%s", notif.Rootnameid)
		payload += fmt.Sprintf(`, reply to this email directly or <a href="%s">leave this organisation</a> to stop receiving these alerts.</div>`, url_leave)
	default:
		payload += " or reply to this email directly.</div>"
	}

	// Buid email
	content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body> %s </body>
    </html>`, payload)
	plainContent, _ := tools.HTMLToMarkdown(content)

	// Postal request body — built via encoding/json now that `attachments`
	// carries variable-length base64 payloads. Hand-rolled %q-escaping
	// across megabyte-sized attachment blobs is too fragile.
	bodyPayload := map[string]any{
		"from":       fmt.Sprintf("%s <notifications@"+DOMAIN+">", author),
		"to":         []string{email},
		"subject":    subject,
		"html_body":  content,
		"plain_body": plainContent,
		"headers": map[string]string{
			"In-Reply-To": fmt.Sprintf("<tension/%s@"+DOMAIN+">", notif.Tid),
			"References":  fmt.Sprintf("<tension/%s@"+DOMAIN+">", notif.Tid),
		},
	}
	if len(att.payload) > 0 {
		bodyPayload["attachments"] = att.payload
	}
	// @TODO; "List-Unsubscribe": "<%s>"
	// see https://github.com/postalserver/postal/issues/2788
	// Other fields: http://apiv1.postalserver.io/controllers/send/message
	body, err := json.Marshal(bodyPayload)
	if err != nil {
		return err
	}

	return sendPostal(body)
}

func SendContractNotificationEmail(ui model.UserNotifInfo, notif model.ContractNotif, att *Attachments) error {
	// Get inputs
	var err error
	var url_redirect string
	var subject string
	var rcpt_name string
	var author string
	var payload string
	var recv string = strings.ReplaceAll(notif.Receiverid, "#", "/")

	// Contract emails carry the latest comment authored by the actor (if any);
	// its files are prefetched once per notification (FetchContractAttachments).
	if att == nil {
		att = &Attachments{}
	}
	// Recipient email
	var email string = ui.User.Email
	if email == "" {
		if x, err := db.GetDB().GetByEq("User.username", ui.User.Username, "User.email"); err != nil {
			return err
		} else {
			email = x.(string)
		}
	}
	// Recipient name
	if ui.User.Name != nil {
		rcpt_name = fmt.Sprintf(" %s (@%s)", *ui.User.Name, ui.User.Username)
	} else if ui.User.Username != "" {
		rcpt_name = " @" + ui.User.Username
	} else {
		rcpt_name = ""
	}
	// Author
	if notif.Uctx.Name != nil {
		author = fmt.Sprintf("%s (@%s)", *notif.Uctx.Name, notif.Uctx.Username)
	} else {
		author = "@" + notif.Uctx.Username
	}

	url_unsubscribe := fmt.Sprintf("https://"+DOMAIN+"/user/%s/settings?m=email", ui.User.Username)
	url_redirect = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s/contract/%s", notif.Rootnameid, notif.Tid, notif.Contract.ID)
	vars := []string{}
	if ui.IsPending {
		// Puid var is used to identify the pending users from client.
		token, err := db.GetDB().GetByEq("PendingUser.email", email, "PendingUser.token")
		if err != nil {
			return err
		}
		vars = append(vars, fmt.Sprintf("puid=%s", token))
	}
	if len(vars) > 0 {
		url_redirect += "?" + strings.Join(vars, "&")
	}

	// Build body
	e := notif.Contract.Event.EventType
	switch notif.ContractEvent {
	case model.NewContract:
		switch notif.Contract.Status {
		case model.ContractStatusOpen:
			switch ui.Reason {
			case model.ReasonIsInvited:
				x, err := db.GetDB().GetByEq("Node.nameid", notif.Receiverid, "Node.name")
				if err != nil {
					return err
				}
				orga_name := x.(string)
				subject = fmt.Sprintf("[%s] You are invited to this organisation", recv)
				payload = fmt.Sprintf(`Hi%s,<br><br> You have been invited by %s to join the organisation <a style="color:#002e62;font-weight: 600;" href="https://`+DOMAIN+`/o/%s">%s</a>.<br><br>
                Please click the link below to accept or reject the invitation:<br><a href="%s">%s</a>`, rcpt_name, author, recv, orga_name, url_redirect, url_redirect)
			case model.ReasonIsLinkCandidate:
				subject = fmt.Sprintf("[%s] You have a new role invitation", recv)
				payload = fmt.Sprintf(`Hi%s,<br><br> You have been invited by %s to take a new role.<br><br>
                Please click the link below to accept or reject the invitation:<br><a href="%s">%s</a>`, rcpt_name, author, url_redirect, url_redirect)
			default:
				subject = fmt.Sprintf("[%s][%s] A pending contract needs your attention", recv, e.ToContractText())
				payload = fmt.Sprintf(`Hi%s,<br><br>
                A vote is needed to process a pending contract.<br><br>
                Please click the link below to accept or reject the proposition:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
			}
		case model.ContractStatusCanceled:
			// notify only participant
			if ui.Reason == model.ReasonIsParticipant {
				subject = fmt.Sprintf("[%s][%s] Contract canceled", recv, e.ToContractText())
				payload = fmt.Sprintf(`Hi%s,<br><br>
                The following contract has been canceled:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
			} else {
				return nil
			}
		default:
			// no notification
			return nil
		}
		if notif.Msg != "" {
			payload += "<br><br>—<br>"
		}
	case model.CloseContract:
		// -- notify only the if event has no email notification
		// -- Or invited user exception (because can only be notified if subscribed to the anchor tension...)
		if ui.Reason == model.ReasonIsInvited {
			subject = fmt.Sprintf("[%s] Invitation accepted", recv)
			payload = fmt.Sprintf(`Hi%s,<br><br>Congratulation, your invitation has been accepted in <a href="%s">%s</a>.<br>`, rcpt_name, url_redirect, notif.Tid)
		} else if !notif.IsEventEmailable(ui) {
			subject = fmt.Sprintf("[%s][%s] Contract accepted", recv, e.ToContractText())
			payload = fmt.Sprintf(`Hi%s,<br><br>
            The following contract has been accepted:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
		} else {
			return nil
		}
		// dont repeat a already read message
		notif.Msg = ""
	case model.NewComment:
		subject = fmt.Sprintf("[%s][%s] You have a new comment", recv, e.ToContractText())
	}

	// Add eventual comment
	if notif.Msg != "" {
		// Convert markdown to Html; rewrite /file/<id> img tags for inline
		// CID attachments / absolute URLs the same way tension emails do.
		var buf bytes.Buffer
		if err = md.Convert([]byte(notif.Msg), &buf); err != nil {
			return err
		}
		rendered := rewriteFileImgs(buf.String(), att.inlineByID)
		payload += sanitizer.Sanitize(rendered)
	} else {
		payload += "<br><br>"
	}

	if footer := renderAttachmentFooter(att.footer); footer != "" {
		payload += footer
	}

	payload += fmt.Sprintf(`—
    <div style="color:#666;font-size:small">You are receiving this because %s.`, ui.Reason.ToText())
	if !ui.IsPending {
		payload += fmt.Sprintf(`<br>
        <a href="%s">View it on Fractale</a>, reply to this email directly or <a href="%s">disable</a> email notifications.
        </div>`, url_redirect, url_unsubscribe)
	}

	// Buid email
	content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body> %s </body>
    </html>`, payload)
	plainContent, _ := tools.HTMLToMarkdown(content)

	bodyPayload := map[string]any{
		"from":       fmt.Sprintf("%s <notifications@"+DOMAIN+">", author),
		"to":         []string{email},
		"subject":    subject,
		"html_body":  content,
		"plain_body": plainContent,
		"headers": map[string]string{
			"In-Reply-To": fmt.Sprintf("<contract/%s@"+DOMAIN+">", notif.Contract.ID),
			"References":  fmt.Sprintf("<contract/%s@"+DOMAIN+">", notif.Contract.ID),
		},
	}
	if len(att.payload) > 0 {
		bodyPayload["attachments"] = att.payload
	}
	body, err := json.Marshal(bodyPayload)
	if err != nil {
		return err
	}

	return sendPostal(body)
}
