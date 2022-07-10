package handlers

import (
    "net"
    "net/http"
    "encoding/json"
    "strings"

    "fractale/fractal6.go/db"
    "fractale/fractal6.go/graph"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/tools"
)

// Handle email responses.
func Notifications(w http.ResponseWriter, r *http.Request) {
    // Temporary solution while Postal can't be identify
    if ip, _, err := net.SplitHostPort(r.RemoteAddr);
    err != nil || (ip != "5.196.4.6" && ip != "2001:41d0:401:3200::3be6") {
        http.Error(w, "IP NOT AUTHORIZED", 400)
    }

    // Get request form
    var form EmailForm
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 500); return }

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

    uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil { http.Error(w, err.Error(), 500); return }
    if uctx == nil || uctx.Username == "" {
        http.Error(w, "Unknown user (spam ?)", 400)
        return
    }

    if isTid != "" { // Is a tension reply/comment
        // Build Event
        history := []*model.EventRef{&model.EventRef{}}
        notif := model.EventNotif{
            Uctx: uctx,
            Tid: isTid,
            History: history,
        }
        // Check and publish event
        ok, _, err := graph.TensionEventHook(uctx, isTid, history, nil)
        if err != nil { http.Error(w, err.Error(), 500); return }
        if !ok { http.Error(w, "access denied", 400); return }
        // Publish Notification
        // --
        // Push event in tension event history
        if err := graph.PushHistory(&notif); err != nil {
            http.Error(w, "PushHistory error: " + err.Error(), 500); return
        }
        // Push notification
        if err := graph.PushEventNotifications(notif); err != nil {
            http.Error(w, "PushEventNotifications error: " + err.Error(), 500); return
        }
    } else if isCid != "" { // If contract reply/comment
        // Build Event
        contract, err := db.GetDB().GetContractHook(isCid)
        if err != nil { http.Error(w, err.Error(), 500); return }
        notif := model.ContractNotif{
            Uctx: uctx,
            Tid: contract.Tension.ID,
            Contract: contract,
            ContractEvent: model.NewComment,
        }
        // Check and publish event
        ok, err := graph.HasContractRight(uctx, contract)
        if err != nil { http.Error(w, err.Error(), 500); return }
        if !ok {
            // Check if user is candidate
            for _, c := range contract.Candidates {
                if c.Username == uctx.Username  {
                    ok = true
                    break
                }
            }
            if !ok { http.Error(w, "access denied", 400); return }
        }
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

        // Publish Notification
        // --
        // Push notification
        if err := graph.PushContractNotifications(notif); err != nil {
            http.Error(w, "PushContractNotification error: " + err.Error(), 500); return
        }
    } else {
        // In every other case, it returns an error.
        http.Error(w, "Unknown references", 400)
        return
    }

}
