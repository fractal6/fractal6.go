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
    emailUrl = os.Getenv("EMAIL_API_URL")
    emailSecret = os.Getenv("EMAIL_API_KEY")
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
    if !notif.IsEmailable() {
        return nil
    }

    // Get inputs
    var err error
    var url_redirect string
    var subject string
    var body string
    var email string = ui.User.Email
    var author string
    var payload string
    var message string
    var type_hint string
    if email == "" {
        if x, err := db.GetDB().GetFieldByEq("User.username", ui.User.Username, "User.email"); err != nil {
            return err
        } else {
            email = x.(string)
        }
    }
    if ui.Reason == model.ReasonIsAlert {
        type_hint = "[Alert]"
    }

    // Redirect Url
    url_redirect = fmt.Sprintf("https://fractale.co/tension//%s", notif.Tid)
    if ui.Eid != "" {
        // Eid var is used to mark the event as read from the client.
        url_redirect += fmt.Sprintf("?eid=%s", ui.Eid)
    }
    if createdAt := notif.GetCreatedAt(); createdAt != "" {
        url_redirect += fmt.Sprintf("&goto=%s", createdAt)
    }


    // Build body
    if notif.HasEvent(model.TensionEventCreated) { // Tension added
        // Fetch tension data
        if m, err := db.GetDB().Meta("getFirstComment", map[string]string{"tid": notif.Tid}); err != nil {
            return err
        } else if len(m) > 0 {
            if a, ok := m[0]["author_name"].(string); a != "" && ok {
                author = a
            } else {
                author = "@" + notif.Uctx.Username
            }
            title := m[0]["title"].(string)
            recv := strings.Replace(m[0]["receiverid"].(string), "#", "/", -1)
            message, _ = m[0]["message"].(string)
            subject = fmt.Sprintf("[%s]%s %s", recv, type_hint, title)
        } else {
            return fmt.Errorf("tension %s not found.", notif.Tid)
        }

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
        // Fetch tension data
        if m, err := db.GetDB().Meta("getLastComment", map[string]string{"tid":notif.Tid, "username": notif.Uctx.Username}); err != nil {
            return err
        } else if len(m) > 0 {
            if a, ok := m[0]["author_name"].(string); a != "" && ok {
                author = a
            } else {
                author = "@" + notif.Uctx.Username
            }
            title := m[0]["title"].(string)
            recv := strings.Replace(m[0]["receiverid"].(string), "#", "/", -1)
            message, _ = m[0]["message"].(string)
            subject = fmt.Sprintf("Re: [%s]%s %s", recv, type_hint, title)
        } else {
            return fmt.Errorf("tension %s not found.", notif.Tid)
        }

        // Add automatic message
        if notif.HasEvent(model.TensionEventClosed) {
            payload = fmt.Sprintf(`Closed <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
        } else if notif.HasEvent(model.TensionEventReopened) {
            payload = fmt.Sprintf(`Reopened <a href="%s">%s</a>.<br>`, url_redirect, notif.Tid)
        } else if notif.HasEvent(model.TensionEventUserLeft) || notif.HasEvent(model.TensionEventMemberUnlinked) {
            u := notif.GetExUser()
            payload = fmt.Sprintf(`User %s left or was unlinked in <a href="%s">%s</a>.<br>`, u, url_redirect, notif.Tid)
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
    <div style="color:#666;font-size:small">You are receiving this because %s. <a href="%s">View it on Fractale</a>.</div>
    `, ui.Reason.ToText(), url_redirect)

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
            "References": "<tension/%s@fractale.co>"
        }
    }`, author, email, subject, tools.CleanString(content, true), notif.Tid, notif.Tid)
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
    var email string = ui.User.Email
    var rcpt_name string
    var author string
    var payload string
    var recv string
    var message string
    if email == "" {
        if x, err := db.GetDB().GetFieldByEq("User.username", ui.User.Username, "User.email"); err != nil {
            return err
        } else {
            email = x.(string)
        }
    }
    if ui.User.Name != nil {
        rcpt_name = " " + *ui.User.Name
    } else if ui.User.Username != "" {
		rcpt_name = " @" + ui.User.Username
	} else {
        rcpt_name = ""
    }

    url_redirect = fmt.Sprintf("https://fractale.co/tension//%s/contract/%s", notif.Tid, notif.Contract.ID)

    if ui.Reason == model.ReasonIsPendingCandidate {
        // Puid var is used to identify the pending users from client.
        token, err := db.GetDB().GetFieldByEq("PendingUser.email", email, "PendingUser.token")
        if err != nil { return err }
        url_redirect += fmt.Sprintf("?puid=%s", token)
    }

    // Fetch tension data
    if m, err := db.GetDB().Meta("getLastContractComment", map[string]string{"cid":notif.Contract.ID, "username": notif.Uctx.Username}); err != nil {
        return err
    } else if len(m) > 0 {
        if a, ok := m[0]["author_name"].(string); a != "" && ok {
            author = a
        } else {
            author = "@" + notif.Uctx.Username
        }
        recv = strings.Replace(m[0]["receiverid"].(string), "#", "/", -1)
        message, _ = m[0]["message"].(string)
    } else {
        return fmt.Errorf("contract %s not found.", notif.Tid)
    }

    // Candidate text for open contract
    candidate_subject := func(e model.TensionEvent) (t string) {
        switch  e {
        case model.TensionEventUserJoined:
            t = fmt.Sprintf("[%s] You have a new membership invitation", recv)
        case model.TensionEventMemberLinked:
            t = fmt.Sprintf("[%s] You have a new role invitation", recv)
        default:
            t = fmt.Sprintf("[%s] You have a new invitation", recv)
        }
        return
    }
    candidate_payload := func(e model.TensionEvent	) (t string) {
        switch  e {
        case model.TensionEventUserJoined:
            t = fmt.Sprintf(`Hi%s,<br><br> Your are kindly invited as a new member by %s.<br><br>
            You can see this invitation and accept or reject it by clicking on the following link:<br><a href="%s">%s</a>`, rcpt_name, author, url_redirect, url_redirect)
        case model.TensionEventMemberLinked:
            t = fmt.Sprintf(`Hi%s,<br><br> Your are kindly invited to take a new role by %s.<br><br>
            You can see this invitation and accept or reject it by clicking on the following link:<br><a href="%s">%s</a>`, rcpt_name, author, url_redirect, url_redirect)
        default:
            t = fmt.Sprintf(`Hi%s,<br><br> Your have a new invitation from %s.<br><br>
            You can see this invitation and accept or reject it by clicking on the following link:<br><a href="%s">%s</a>`, rcpt_name, author, url_redirect, url_redirect)
        }
        return
    }

    // Other than candidate text for open contract
    default_subject := func(e model.TensionEvent) string {
        return fmt.Sprintf("[%s][%s] A pending contract needs your attention", recv, e.ToContractText())
    }
    default_payload := func(e model.TensionEvent) string {
        return fmt.Sprintf(`Hi%s,<br><br>
        A vote is requested to process the following contract:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
    }

    comment_subject := func(e model.TensionEvent) string {
        return fmt.Sprintf("[%s][%s] You have a new comment", recv, e.ToContractText())
    }

    closed_subject := func(e model.TensionEvent) string {
        return fmt.Sprintf("[%s][%s] contract accepted", recv, e.ToContractText())
    }
    closed_payload := func(e model.TensionEvent) string {
        return fmt.Sprintf(`Hi%s,<br><br>
        The following contract has been accepted:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
    }

    canceled_subject := func(e model.TensionEvent) string {
        return fmt.Sprintf("[%s][%s] contract canceled", recv, e.ToContractText())
    }
    canceled_payload := func(e model.TensionEvent) string {
        return fmt.Sprintf(`Hi%s,<br><br>
        The following contract has been canceled:<br><a href="%s">%s</a>`, rcpt_name, url_redirect, url_redirect)
    }

    // Build body
    switch notif.ContractEvent {
    case model.NewContract:
        e := notif.Contract.Event.EventType
        switch notif.Contract.Status {
        case model.ContractStatusOpen:
            if ui.Reason == model.ReasonIsCandidate || ui.Reason == model.ReasonIsPendingCandidate {
                subject = candidate_subject(e)
                payload = candidate_payload(e)
            } else {
                subject = default_subject(e)
                payload = default_payload(e)
            }
        case model.ContractStatusClosed:
            // notify only if event has no email notification
            if !notif.IsEmailable() {
                subject = closed_subject(e)
                payload = closed_payload(e)
            } else {
                return nil
            }
        case model.ContractStatusCanceled:
            // notify only participant
            if ui.Reason == model.ReasonIsParticipant {
                subject = canceled_subject(e)
                payload = canceled_payload(e)
            } else {
                return nil
            }
        default:
            // no notification
            return nil
        }
        if message != "" {
            payload += "<br><br>—<br>"
        }
    case model.NewComment:
        subject = comment_subject(notif.Contract.Event.EventType)
    }

    // Add eventual comment
    if message != "" {
        // Convert markdown to Html
        var buf bytes.Buffer
        if err = goldmark.Convert([]byte(message), &buf); err != nil {
            return err
        }
        payload += bluemonday.UGCPolicy().Sanitize(buf.String())
    }

    if notif.ContractEvent == model.NewComment {
        payload += "<br>" + fmt.Sprintf(`—
        <div style="color:#666;font-size:small">You are receiving this because %s. <a href="%s">View it on Fractale</a>.</div>
        `, ui.Reason.ToText(), url_redirect)
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

