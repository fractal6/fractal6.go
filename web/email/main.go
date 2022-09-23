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
    "github.com/microcosm-cc/bluemonday"
    "fractale/fractal6.go/tools"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/db"
)

var emailSecret string
var emailUrl string
var maintainerEmail string

func init() {
    emailUrl = viper.GetString("server.email_api_url")
    emailSecret = viper.GetString("server.email_api_key")
    if emailUrl == "" {
        emailUrl = os.Getenv("EMAIL_API_URL")
    }
    if emailSecret == "" {
        emailSecret = os.Getenv("EMAIL_API_KEY")
    }
    if emailUrl == "" || emailSecret == "" {
        fmt.Println("EMAIL_API_URL/KEY not found. email notifications disabled.")
    }

    maintainerEmail = viper.GetString("server.maintainer_email")
}

// Send an email with a http request to the email server API to the admin email.
func SendMaintainerEmail(subject, body string) error {
    if maintainerEmail == "" { return nil }

    body = fmt.Sprintf(`{
        "from": "%s <alert@fractale.co>",
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
        return fmt.Errorf("http error, see body. (code %s)", resp.Status)
    }

    return nil
}

// Send an verification email for signup
func SendVerificationEmail(email, token string) error {
    url_redirect := fmt.Sprintf("https://fractale.co/verification?email_token=%s", token)

    content := fmt.Sprintf(`<html>
	<head>
	<title>Activate your Fractale account</title>
	<meta charset="utf-8">
	</head>
	<body>
	<p>To activate your account at <b>fractale.co</b>, click the link below (valid one hour):</p>
	<a href="%s">%s</a>
	<br><br>—<br>
	<small>If you are not at the origin of this request, please ignore this mail.</small>
	</body>
    </html>`, url_redirect, url_redirect)

    body := fmt.Sprintf(`{
        "from": "Fractale <noreply@fractale.co>",
        "to": ["%s"],
        "subject": "Activate your account at fractale.co",
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
    url_redirect := fmt.Sprintf("https://fractale.co/password-reset?x=%s", token)

    content := fmt.Sprintf(`<html>
	<head>
	<title>Reset your Fractale Password</title>
	<meta charset="utf-8">
	</head>
	<body>
	<h2>Forgot your password?</h2>
	<p>To reset your password at <b>fractale.co</b>, click the link below (valid one hour):</p>
	<a href="%s">%s</a>
	<br><br>—<br>
	<small>If you are not at the origin of this request, please ignore this mail.</small>
	</body>
    </html>`, url_redirect, url_redirect)

    body := fmt.Sprintf(`{
        "from": "Fractale <noreply@fractale.co>",
        "to": ["%s"],
        "subject": "Reset your password at fractale.co",
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
    url_unsubscribe := fmt.Sprintf("https://fractale.co/tension//%s?unsubscribe=email", notif.Tid)
    url_redirect = fmt.Sprintf("https://fractale.co/tension//%s", notif.Tid)
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
            if err = goldmark.Convert([]byte(message), &buf); err != nil {
                return err
            }
            payload = bluemonday.UGCPolicy().Sanitize(buf.String())
        }

    } else { // Tension updated
        subject = fmt.Sprintf("Re: [%s]%s %s", recv, type_hint, title)

        // Add automatic message
        if notif.HasEvent(model.TensionEventClosed) {
            payload = fmt.Sprintf(`Closed <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
        } else if notif.HasEvent(model.TensionEventReopened) {
            payload = fmt.Sprintf(`Reopened <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
        } else if notif.HasEvent(model.TensionEventBlobPushed) {
            payload = fmt.Sprintf(`Mandate updated <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
        } else if notif.HasEvent(model.TensionEventUserJoined) {
            u := notif.GetNewUser()
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            if u == ui.User.Username {
                // Notification happens in contract_op.VoteEventHook function since we never go here
                // (except if the user has subscrided to the anchor tensionn which is unlikelly).
                return nil
            }  else {
                payload = fmt.Sprintf(`%s joined this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            }

        } else if notif.HasEvent(model.TensionEventUserLeft) {
            u := notif.GetExUser()
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            anchorTid, _ := db.GetDB().GetSubSubFieldByEq("Node.nameid", notif.Receiverid, "Node.source", "Blob.tension", "uid" )
            if anchorTid != nil && anchorTid.(string) == notif.Tid  {
                payload = fmt.Sprintf(`%s left this organisation in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            } else {
                payload = fmt.Sprintf(`%s left this role in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            }
        } else if notif.HasEvent(model.TensionEventMemberLinked) {
            u := notif.GetNewUser()
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            if u == ui.User.Username {
                payload = fmt.Sprintf(`Hi %s,<br><br>Congratulation, your application has been accepted in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            } else {
                payload = fmt.Sprintf(`%s is lead link in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
            }
        } else if notif.HasEvent(model.TensionEventMemberUnlinked) {
            u := notif.GetExUser()
            if x, _ := db.GetDB().GetFieldByEq("User.username", u, "User.name"); x != nil {
                u = fmt.Sprintf("%s (@%s)", x.(string), u)
            }
            payload = fmt.Sprintf(`%s was unlinked in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
        }

        // Add eventual comment
        if notif.HasEvent(model.TensionEventCommentPushed) && message != "" {
            // Convert markdown to Html
            var buf bytes.Buffer
            if err = goldmark.Convert([]byte(message), &buf); err != nil {
                return err
            }
            if payload != "" {
                payload += "<br>—<br>"
            }
            payload += bluemonday.UGCPolicy().Sanitize(buf.String())
        } else {
            payload += "<br>"
        }
    }

    // Add footer
    payload += fmt.Sprintf(`—
    <div style="color:#666;font-size:small">You are receiving this because %s.<br>
    <a href="%s">View it on Fractale</a>`, ui.Reason.ToText(), url_redirect)
    if ui.Reason == model.ReasonIsSubscriber {
        payload += fmt.Sprintf(`, reply to this email directly, or <a href="%s">unsubscribe</a>.</div>`, url_unsubscribe)
    } else {
        payload += " or reply to this email directly.</div>"
    }

    // Buid email
    content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body> %s </body>
    </html>`, payload)

    body = fmt.Sprintf(`{
        "from": "%s <notifications@fractale.co>",
        "to": ["%s"],
        "subject": "%s",
        "html_body": "%s",
        "headers": {
            "In-Reply-To": "<tension/%s@fractale.co>",
            "References": "<tension/%s@fractale.co>",
            "List-Unsubscribe": "<%s>"
        }
    }`, author, email, subject, tools.CleanString(content, true), notif.Tid, notif.Tid, url_unsubscribe)
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
        return fmt.Errorf("http error, see body. (code %s)", resp.Status)
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

    url_unsubscribe := fmt.Sprintf("https://fractale.co/user/%s/settings?m=email", ui.User.Username)
    url_redirect = fmt.Sprintf("https://fractale.co/tension//%s/contract/%s", notif.Tid, notif.Contract.ID)
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
                payload = fmt.Sprintf(`Hi%s,<br><br> Your are kindly invited in the organisation <a style="color:#002e62;" href="https://fractale.co/o/%s">%s</a> by %s.<br><br>
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
        if err = goldmark.Convert([]byte(notif.Msg), &buf); err != nil {
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
        "from": "%s <notifications@fractale.co>",
        "to": ["%s"],
        "subject": "%s",
        "html_body": "%s",
        "headers": {
            "In-Reply-To": "<contract/%s@fractale.co>",
            "References": "<contract/%s@fractale.co>"
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
        return fmt.Errorf("http error, see body. (code %s)", resp.Status)
    }

    return nil
}

