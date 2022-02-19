package email

import (
    "fmt"
	"os"
	"bytes"
	"strings"
	"net/http"
    "crypto/tls"
    "github.com/yuin/goldmark"
    "github.com/microcosm-cc/bluemonday"
    "fractale/fractal6.go/tools"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/db"
)

var emailSecret string
var emailUrl string

func init() {
    emailUrl = os.Getenv("EMAIL_API_URL")
    emailSecret = os.Getenv("EMAIL_API_KEY")
    if emailUrl == "" || emailSecret == "" {
        fmt.Println("EMAIL_API_URL/KEY not found. email notifications disabled.")
    }
}

// Post send an email with a http request to the email server API
func SendResetEmail(email, token string) error {
    url_redirect := fmt.Sprintf("https://fractale.co/password-reset?x=%s", token)

    content := fmt.Sprintf(`<html>
	<head>
	<title>Reset your Fractale Password</title>
	<meta charset="utf-8">
	</head>
	<body>
	<h2> Forgot your password?</h2>
	<p>To reset your password at <b>fractale.co</b>, click the link below (valid one hour):</p>
	<a href="%s">%s</a>
	<br><br>
	<p>If you are not at the origin of this request, please ignore this mail.</p>
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
    var email string = ui.User.Email
    var author string
    var payload string
    var footer string
    if email == "" {
        if x, err := db.GetDB().GetFieldByEq("User.username", ui.User.Username, "User.email"); err != nil {
            return err
        } else {
            email = x.(string)
        }
    }

    // Build body
    {
        url_redirect = fmt.Sprintf("https://fractale.co/tension//%s?eid=%s", notif.Tid, ui.Eid)
        reason := fmt.Sprintf("%s", ui.Reason.ToText())
        if m, err := db.GetDB().Meta("getFirstComment", map[string]string{"tid": notif.Tid}); err != nil {
            return err
        } else if len(m) > 0 {
            if a, ok := m[0]["author_name"].(string); a != "" && ok {
                author = a
            } else {
                author = "@"+m[0]["author_username"].(string)
            }
            title := m[0]["title"].(string)
            recv := strings.Replace(m[0]["receiverid"].(string), "#", "/", -1)
            subject = fmt.Sprintf("[%s] %s", recv, title)
            message := m[0]["message"].(string)
            // Convert markdown to Html
            var buf bytes.Buffer
            if err = goldmark.Convert([]byte(message), &buf); err != nil {
                return err
            }
            payload = bluemonday.UGCPolicy().Sanitize(buf.String())
        }

        footer = fmt.Sprintf(`<br><br>
        —
        <p style="color:#666;font-size:small">You are receiving this because %s. <a href="%s">View it on Fractale</a>.</p>
        `, reason, url_redirect)

    }

    // Buid email
    content := fmt.Sprintf(`<html>
    <head> <meta charset="utf-8"> </head>
    <body>
    %s
    %s
    </body>
    </html>`, payload, footer)

    body = fmt.Sprintf(`{
        "from": "%s <notifications@fractale.co>",
        "to": ["%s"],
        "subject": "%s",
        "html_body": "%s"
    }`, author, email, subject, tools.CleanString(content, true))

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

func SendContractNotificationEmail(ui model.UserNotifInfo, notif model.ContractNotif) error {
    return nil
}

