/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
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
    "fmt"
	"os"
	"bytes"
	"strings"
	"net/http"
    "crypto/tls"
	"github.com/spf13/viper"
    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/extension"
    "github.com/yuin/goldmark/renderer/html"
    "github.com/microcosm-cc/bluemonday"
    "fractale/fractal6.go/tools"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/db"
)

var md goldmark.Markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

var emailSecret string
var emailUrl string
var maintainerEmail string
var DOMAIN string

func init() {
    emailUrl = viper.GetString("mailer.email_api_url")
    emailSecret = viper.GetString("mailer.email_api_key")
    if emailUrl == "" {
        emailUrl = os.Getenv("EMAIL_API_URL")
    }
    if emailSecret == "" {
        emailSecret = os.Getenv("EMAIL_API_KEY")
    }
    if emailUrl == "" || emailSecret == "" {
        fmt.Println("EMAIL_API_URL/KEY not found. email notifications disabled.")
    }

    DOMAIN = viper.GetString("server.domain")
    maintainerEmail = viper.GetString("mailer.admin_email")
}

// Send an email with a http request to the email server API to the admin email.
func SendMaintainerEmail(subject, body string) error {
    if maintainerEmail == "" { return nil }

    body = fmt.Sprintf(`{
        "from": "%s <alert@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "%s",
        "plain_body": "%s"
    }`, "Fractal6 Alert", maintainerEmail, subject, tools.QuoteString(body))
    // Other fields: http://apiv1.postalserver.io/controllers/send/message

    req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer([]byte(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Server-API-Key", emailSecret)

    customTransport := http.DefaultTransport.(*http.Transport).Clone()
    customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    client := &http.Client{Transport: customTransport}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("http postal error, see body. (code %s)", resp.Status)
    }

    return nil
}

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

    body := fmt.Sprintf(`{
        "from": "Fractale <noreply@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "Activate your account at `+DOMAIN+`",
        "html_body": "%s"
    }`, email, tools.CleanString(content, true))

    req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer([]byte(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Server-API-Key", emailSecret)

    customTransport := http.DefaultTransport.(*http.Transport).Clone()
    customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    client := &http.Client{Transport: customTransport}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()

    return nil
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

    body := fmt.Sprintf(`{
        "from": "Fractale <noreply@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "Reset your password at `+DOMAIN+`",
        "html_body": "%s"
    }`, email, tools.CleanString(content, true))

    req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer([]byte(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Server-API-Key", emailSecret)

    customTransport := http.DefaultTransport.(*http.Transport).Clone()
    customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    client := &http.Client{Transport: customTransport}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()

    return nil
}

func SendEventNotificationEmail(ui model.UserNotifInfo, notif model.EventNotif) error {
    // Get inputs
    var err error
    var url_redirect string
    var subject string
    var body string
    var author string
    var payload string
    var recv string = strings.Replace(notif.Receiverid, "#", "/", -1)
    var title string = notif.Title
    var message string = notif.Msg
    // Recipient email
    var email string = ui.User.Email
    if email == "" {
        if x, err := db.GetDB().GetFieldByEq("User.username", ui.User.Username, "User.email"); err != nil {
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
        type_hint = "[Alert]"
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
            // Convert markdown to Html
            var buf bytes.Buffer
            if err = md.Convert([]byte(message), &buf); err != nil {
                return err
            }
            payload = bluemonday.UGCPolicy().Sanitize(buf.String())
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
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            if itsYou {
                // Notification happens in contract_op.VoteEventHook function since we never go here
                // (except if the user has subscrided to the anchor tensionn which is unlikelly).
                return nil
            }  else {
                auto_msg = fmt.Sprintf(`%s joined this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            }

        } else if notif.HasEvent(model.TensionEventUserLeft) {
            u := notif.GetExUser()
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            anchorTid, _ := db.GetDB().GetSubSubFieldByEq("Node.nameid", notif.Receiverid, "Node.source", "Blob.tension", "uid" )
            if anchorTid != nil && anchorTid.(string) == notif.Tid  {
                auto_msg = fmt.Sprintf(`%s left this organization in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            } else {
                auto_msg = fmt.Sprintf(`%s left his role in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            }
        } else if notif.HasEvent(model.TensionEventMemberLinked) {
            u := notif.GetNewUser()
            itsYou := u == ui.User.Username
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
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
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            anchorTid, _ := db.GetDB().GetSubSubFieldByEq("Node.nameid", notif.Receiverid, "Node.source", "Blob.tension", "uid" )
            if anchorTid != nil && anchorTid.(string) == notif.Tid  {
                if itsYou {
                    auto_msg = fmt.Sprintf(`You have been removed from this organization in <a href="%s">%s</a>.<br>`,  url_redirect, notif.Tid)
                } else {
                    auto_msg = fmt.Sprintf(`%s has been removed from this organization in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
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
            // Convert markdown to Html
            var buf bytes.Buffer
            if err = md.Convert([]byte(message), &buf); err != nil {
                return err
            }
            comment = bluemonday.UGCPolicy().Sanitize(buf.String())
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

    // Add footer
    var url_unsubscribe string
    payload += fmt.Sprintf(`—
    <div style="color:#666;font-size:small">You are receiving this because %s.<br>
    <a href="%s">View it on Fractale</a>`, ui.Reason.ToText(), url_redirect)
    if ui.Reason == model.ReasonIsSubscriber {
        url_unsubscribe = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s?unsubscribe=email", notif.Rootnameid, notif.Tid)
        payload += fmt.Sprintf(`, reply to this email directly, or <a href="%s">unsubscribe</a>.</div>`, url_unsubscribe)
    } else if ui.Reason == model.ReasonIsAnnouncement {
        url_unsubscribe = fmt.Sprintf("https://"+DOMAIN+"/tension/%s/%s?unwatch=email", notif.Rootnameid, notif.Tid)
        payload += fmt.Sprintf(`, or <a href="%s">unsubscribe</a> from all announcements for this organization.</div>`, url_unsubscribe)
    } else if ui.Reason == model.ReasonIsAlert {
        payload += ", reply to this email directly or leave this organization to stop receiving these alerts.</div>"
    } else {
        payload += " or reply to this email directly.</div>"
    }

    // Buid email
    content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body> %s </body>
    </html>`, payload)

    body = fmt.Sprintf(`{
        "from": "%s <notifications@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "%s",
        "html_body": "%s",
        "headers": {
            "In-Reply-To": "<tension/%s@`+DOMAIN+`>",
            "References": "<tension/%s@`+DOMAIN+`>"
        }
    }`, author, email, subject, tools.CleanString(content, true), notif.Tid, notif.Tid)
    // @TODO; "List-Unsubscribe": "<%s>"
    // Other fields: http://apiv1.postalserver.io/controllers/send/message

    req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer([]byte(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Server-API-Key", emailSecret)

    customTransport := http.DefaultTransport.(*http.Transport).Clone()
    customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    client := &http.Client{Transport: customTransport}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("http postal error, see body. (code %s)", resp.Status)
    }

    return nil
}

func SendContractNotificationEmail(ui model.UserNotifInfo, notif model.ContractNotif) error {
    // Get inputs
    var err error
    var url_redirect string
    var subject string
    var body string
    var rcpt_name string
    var author string
    var payload string
    var recv string = strings.Replace(notif.Receiverid, "#", "/", -1)
    // Recipient email
    var email string = ui.User.Email
    if email == "" {
        if x, err := db.GetDB().GetFieldByEq("User.username", ui.User.Username, "User.email"); err != nil {
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
        token, err := db.GetDB().GetFieldByEq("PendingUser.email", email, "PendingUser.token")
        if err != nil { return err }
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
            if ui.Reason == model.ReasonIsInvited {
                subject = fmt.Sprintf("[%s] You are invited to this organisation", recv)
                payload = fmt.Sprintf(`Hi%s,<br><br> Your are kindly invited in the organisation <a style="color:#002e62;font-weight: 600;" href="https://`+DOMAIN+`/o/%s">%s</a> by %s.<br><br>
                You can see this invitation and accept or reject it by clicking on the following link:<br><a href="%s">%s</a>`, rcpt_name, recv, recv, author, url_redirect, url_redirect)
            } else if ui.Reason == model.ReasonIsLinkCandidate {
                subject = fmt.Sprintf("[%s] You have a new role invitation", recv)
                payload = fmt.Sprintf(`Hi%s,<br><br> Your are kindly invited to take a new role by %s.<br><br>
                You can see this invitation and accept or reject it by clicking on the following link:<br><a href="%s">%s</a>`, rcpt_name, author, url_redirect, url_redirect)
            } else {
                subject = fmt.Sprintf("[%s][%s] A pending contract needs your attention", recv, e.ToContractText())
                payload = fmt.Sprintf(`Hi%s,<br><br>
                A vote is needed to process the following contract:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
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
        // Convert markdown to Html
        var buf bytes.Buffer
        if err = md.Convert([]byte(notif.Msg), &buf); err != nil {
            return err
        }
        payload += bluemonday.UGCPolicy().Sanitize(buf.String())
    } else {
            payload += "<br><br>"
    }

    payload += fmt.Sprintf(`—
    <div style="color:#666;font-size:small">You are receiving this because %s.`, ui.Reason.ToText())
    if !ui.IsPending  {
        payload += fmt.Sprintf(`<br>
        <a href="%s">View it on Fractale</a>, reply to this email directly or <a href="%s">disable</a> email notifications.
        </div>`, url_redirect, url_unsubscribe)
    }

    // Buid email
    content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body> %s </body>
    </html>`, payload)

    body = fmt.Sprintf(`{
        "from": "%s <notifications@`+DOMAIN+`>",
        "to": ["%s"],
        "subject": "%s",
        "html_body": "%s",
        "headers": {
            "In-Reply-To": "<contract/%s@`+DOMAIN+`>",
            "References": "<contract/%s@`+DOMAIN+`>"
        }
    }`, author, email, subject, tools.CleanString(content, true), notif.Contract.ID, notif.Contract.ID)

    req, err := http.NewRequest("POST", emailUrl, bytes.NewBuffer([]byte(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Server-API-Key", emailSecret)

    customTransport := http.DefaultTransport.(*http.Transport).Clone()
    customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    client := &http.Client{Transport: customTransport}
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("http postal error, see body. (code %s)", resp.Status)
    }

    return nil
}

