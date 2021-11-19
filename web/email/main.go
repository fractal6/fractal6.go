package email

import (
    "fmt"
	"os"
	"bytes"
	"net/http"
    "crypto/tls"
    "zerogov/fractal6.go/tools"
)

var emailSecret string
var emailUrl string

func init() {
    emailSecret = os.Getenv("EMAIL_API_KEY")
    emailUrl = os.Getenv("EMAIL_API_URL")
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
