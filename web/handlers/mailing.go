package handlers

import (
    "fmt"
    "net"
    "net/http"
    "encoding/json"
    //"strings"

    //"fractale/fractal6.go/db"
    //"fractale/fractal6.go/graph"
    //"fractale/fractal6.go/graph/model"
    //"fractale/fractal6.go/tools"
)

type EmailForm struct {
    From string            `json:"mail_from"`
    To string              `json:"rcpt_to"`
    Title string           `json:"subject"`
    Msg string             `json:"plain_body"`
    References string      `json:"references"`
    //AttachmentQuantity int  `json:"attachment_quantity"`
    //Attachments []string    `json:"attachments"`
}

// Handle email responses.
func Mailing(w http.ResponseWriter, r *http.Request) {
    // Temporary solution while Postal can't be identify
    if ip, _, err := net.SplitHostPort(r.RemoteAddr);
    err != nil || (ip != "5.196.4.6" && ip != "2001:41d0:401:3200::3be6") {
        http.Error(w, "IP NOT AUTHORIZED", 400)
    }

    // Get request form
    var form EmailForm
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 500); return }

    fmt.Println(
        fmt.Sprintf("Got mailing mail from: %s to: %s ", form.From, form.To),
    )

    http.Error(w, "not implemented", 400)
    return

}
