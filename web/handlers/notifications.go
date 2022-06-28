package handlers

import (
    "net/http"
    "encoding/json"
    "strings"

    "fractale/fractal6.go/db"
    "fractale/fractal6.go/graph"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/tools"
)

type EmailForm struct {
    Email string            `json:"mail_from"`
    Title string            `json:"subject"`
    Msg string              `json:"plain_body"`
    References string       `json:"references"`
    //AttachmentQuantity int  `json:"attachment_quantity"`
    //Attachments []string    `json:"attachments"`
}

// Handle email responses.
func Notifications(w http.ResponseWriter, r *http.Request) {

    // Get request form
    var form EmailForm
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 400); return }

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

    uctx, err := db.GetDB().GetUctx("email", form.Email)
	if err != nil { http.Error(w, err.Error(), 500); return }

    if isTid != "" {
        // If reply to a tension
        if err != nil { http.Error(w, err.Error(), 500); return }
        notif := model.EventNotif{
            Uctx: uctx,
            Tid: isTid,
            History: []*model.EventRef{&model.EventRef{}},
        }
        // Push event in tension event history
        if err := graph.PushHistory(&notif); err != nil {
            http.Error(w, "PushHistory error: " + err.Error(), 500); return
        }
        // Push notification
        if err := graph.PushEventNotifications(notif); err != nil {
            http.Error(w, "PushEventNotifications error: " + err.Error(), 500); return
        }
    } else if isCid != "" {
        // If reply to a contract
        contract, err := db.GetDB().GetContractHook(isCid)
        if err != nil { http.Error(w, err.Error(), 500); return }
        notif := model.ContractNotif{
            Uctx: uctx,
            Tid: contract.Tension.ID,
            Contract: contract,
            ContractEvent: model.NewComment,
        }
        // Publish event !
        createdAt := tools.Now()
        db.GetDB().Update(db.DB.GetRootUctx(), "contract", model.UpdateContractInput{
            Filter: &model.ContractFilter{ID:[]string{isCid}},
            Set: &model.ContractPatch{
                Comments: []*model.CommentRef{&model.CommentRef{
                    CreatedAt: &createdAt,
                    CreatedBy: &model.UserRef{Username: &uctx.Username},
                    Message: &form.Msg,
                }},
            },
        })

        // Push notification
        if err := graph.PushContractNotifications(notif); err != nil {
            http.Error(w, "PushContractNotification error: " + err.Error(), 500); return
        }
    }

    // In every other case, it returns an error.
    http.Error(w, "Unknown references", 500)
    return

}
