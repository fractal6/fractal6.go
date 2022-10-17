package handlers

import (
    "fmt"
    "net/http"
    "encoding/json"
    "strings"

    "fractale/fractal6.go/db"
    "fractale/fractal6.go/graph"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/tools"
)


/*
 *
 * This code manage receiving email as HTTP requests from the MTA
 *
 */

type EmailForm struct {
    From string            `json:"mail_from"`
    To string              `json:"rcpt_to"`
    Title string           `json:"subject"`
    Msg string             `json:"plain_body"`
    References string      `json:"references"`
    //AttachmentQuantity int  `json:"attachment_quantity"`
    //Attachments []string    `json:"attachments"`
}

// Handle email responses. Receiving email response from email notifications.
func Notifications(w http.ResponseWriter, r *http.Request) {
    // Temporary solution while Postal can't be identify
    //if ip, _, err := net.SplitHostPort(r.RemoteAddr);
    //err != nil || (ip != "5.196.4.6" && ip != "2001:41d0:401:3200::3be6") {
    //    http.Error(w, "IP NOT AUTHORIZED", 400); return
    //}

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

    // Get author
    uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil { http.Error(w, err.Error(), 500); return }
    createdAt := tools.Now()
    createdBy := model.UserRef{Username: &uctx.Username}

    if isTid != "" { // Is a tension reply/comment
        // Build Event
        e := model.TensionEventCommentPushed
        history := []*model.EventRef{&model.EventRef{
            CreatedAt: &createdAt,
            CreatedBy: &createdBy,
            EventType: &e,
        }}
        notif := model.EventNotif{
            Uctx: uctx,
            Tid: isTid,
            History: history,
        }
        // Check event
        ok, _, err := graph.TensionEventHook(uctx, isTid, history, nil)
        if err != nil { http.Error(w, err.Error(), 500); return }
        if !ok { http.Error(w, "access denied", 400); return }
        // Publish  event
        db.GetDB().Update(db.DB.GetRootUctx(), "tension", model.UpdateTensionInput{
            Filter: &model.TensionFilter{ID:[]string{isTid}},
            Set: &model.TensionPatch{
                Comments: []*model.CommentRef{&model.CommentRef{
                    CreatedAt: &createdAt,
                    CreatedBy: &createdBy,
                    Message: &form.Msg,
                }},
            },
        })

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
        // Check  event
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
        // Publish  event
        db.GetDB().Update(db.DB.GetRootUctx(), "contract", model.UpdateContractInput{
            Filter: &model.ContractFilter{ID:[]string{isCid}},
            Set: &model.ContractPatch{
                Comments: []*model.CommentRef{&model.CommentRef{
                    CreatedAt: &createdAt,
                    CreatedBy: &createdBy,
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
        http.Error(w, "Unknown references", 400); return
    }

}

// Handle eemail sent to orga. Convert email to tension.
func Mailing(w http.ResponseWriter, r *http.Request) {
    // Validate WebHook identity
    xp := r.Header.Get("X-Postal-Signature")
    fmt.Println("X-Postal-Sign", xp)
    if err := tools.ValidatePostalSignature(r); err != nil {
        http.Error(w, err.Error(), 400); return
    }

    // Get request form
    var form EmailForm
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 500); return }

    fmt.Println(
        fmt.Sprintf("Got mailing mail from: %s to: %s ", form.From, form.To),
    )

    // Get the nameid of the targeted circle
    receiverid := form.To
    filter := `eq(Node.isArchived, false)`
    if ex, _ := db.GetDB().Exists("Node.nameid", receiverid, &filter); !ex {
        http.Error(w, "NAMEID NOT FOUND", 500); return
    }

    // Get author
    uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil { http.Error(w, err.Error(), 500); return }
    createdAt := tools.Now()
    createdBy := model.UserRef{Username: &uctx.Username}

    // Build the tension
    et := model.TensionEventCreated
    rootnameid, _ := codec.Nid2rootid(receiverid)
    emitterid := codec.MemberIdCodec(rootnameid, uctx.Username)
    event := model.EventRef{
        CreatedAt: &createdAt,
        CreatedBy: &createdBy,
        EventType: &et,
    }
    var ev model.Event
    tools.StructMap(event, &ev)
    tension := model.Tension{
        CreatedAt: createdAt,
        CreatedBy: &model.User{Username: uctx.Username},
        Emitterid: emitterid,
        Emitter: &model.Node{Nameid: emitterid},
        Receiverid: receiverid,
        Receiver: &model.Node{Nameid: receiverid},
        Type: model.TensionTypeOperational,
        Status: model.TensionStatusOpen,
        Title: "",
        Comments:[]*model.Comment{
            &model.Comment{
                CreatedAt: createdAt,
                CreatedBy: &model.User{Username: uctx.Username},
                Message: "",
            },
        },
        History:[]*model.Event{&ev},
        Subscribers:[]*model.User{&model.User{Username: uctx.Username}},
    }

    // Verify author can create tension
    ok, _, err := graph.ProcessEvent(uctx, &tension, &event, nil, nil, true, false)
    if !ok || err != nil {
        http.Error(w, "NOT AUTHORIZED TO CREATE TENSION HERE", 500); return
    }

    // Create tension
    var tensionInput  model.AddTensionInput
    tools.StructMap(tension, &tensionInput)
    if _, err = db.GetDB().Add(*uctx, "tension", tensionInput); err != nil {
        http.Error(w, err.Error(), 400); return
    }

}
