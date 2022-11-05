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

package handlers

import (
    //"fmt"
    "strings"
	"io/ioutil"
    "net/http"
    "encoding/json"
	"github.com/spf13/viper"

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

var postalWebhookPK string
var matrixPostalRoom string
var matrixToken string

func init() {
    postalWebhookPK = viper.GetString("mailer.dkim_key")
    matrixPostalRoom = viper.GetString("mailer.matrix_postal_room")
    matrixToken = viper.GetString("mailer.matrix_token")
}

type EmailForm struct {
    From string            `json:"mail_from"`
    To string              `json:"rcpt_to"`
    Title string           `json:"subject"`
    Msg string             `json:"plain_body"`
    References string      `json:"references"`
    //AttachmentQuantity int  `json:"attachment_quantity"`
    //Attachments []string    `json:"attachments"`
}

// Handle user email responses. Receiving email response from email notifications.
func Notifications(w http.ResponseWriter, r *http.Request) {
    // Validate WebHook identity
    if err := tools.ValidatePostalSignature(r, postalWebhookPK); err != nil {
        http.Error(w, err.Error(), 400); return
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

    // Get author
    uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil { http.Error(w, err.Error(), 400); return }
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
        notif := model.EventNotif{
            Uctx: uctx,
            Tid: isTid,
            History: history,
        }
        // Push notification
        if err := graph.PushEventNotifications(notif); err != nil {
            http.Error(w, "PushEventNotifications error: " + err.Error(), 500); return
        }
    } else if isCid != "" { // If contract reply/comment
        // Build Event
        contract, err := db.GetDB().GetContractHook(isCid)
        if err != nil { http.Error(w, err.Error(), 500); return }
        // Check  event
        ok, err := graph.HasContractRight(uctx, contract)
        if err != nil { http.Error(w, err.Error(), 400); return }
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
        notif := model.ContractNotif{
            Uctx: uctx,
            Tid: contract.Tension.ID,
            Contract: contract,
            ContractEvent: model.NewComment,
        }
        // Push notification
        if err := graph.PushContractNotifications(notif); err != nil {
            http.Error(w, "PushContractNotification error: " + err.Error(), 500); return
        }
    } else {
        // In every other case, it returns an error.
        http.Error(w, "Unknown references", 400); return
    }

}

// Handle email sent to orga. Convert email to tension.
func Mailing(w http.ResponseWriter, r *http.Request) {
    // Validate WebHook identity
    if err := tools.ValidatePostalSignature(r, postalWebhookPK); err != nil {
        http.Error(w, err.Error(), 400); return
    }

    // Get request form
    var form EmailForm
    err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil { http.Error(w, err.Error(), 500); return }

    // Get author
    uctx, err := db.GetDB().GetUctx("email", form.From)
	if err != nil {
        http.Error(w, "You need an account on Fractale to send email to organisation, please visit https://fractale.co \n\n" + err.Error(), 400)
        return
    }
    createdAt := tools.Now()
    createdBy := model.User{Username: uctx.Username}

    // Get the nameid of the targeted circle
    receiverid := strings.Replace(strings.Split(form.To, "@")[0], "/", "#", -1)
    filter := `eq(Node.isArchived, false)`
    if ex, _ := db.GetDB().Exists("Node.nameid", receiverid, &filter); !ex {
        http.Error(w, "NAMEID NOT FOUND", 400); return
    }

    // Build the tension
    e := model.TensionEventCreated
    rootnameid, _ := codec.Nid2rootid(receiverid)
    emitterid := codec.MemberIdCodec(rootnameid, uctx.Username)
    event := model.Event{
        CreatedAt: createdAt,
        CreatedBy: &createdBy,
        EventType: e,
    }
    tension := model.Tension{
        CreatedAt: createdAt,
        CreatedBy: &model.User{Username: uctx.Username},
        Emitterid: emitterid,
        Emitter: &model.Node{Nameid: emitterid},
        Receiverid: receiverid,
        Receiver: &model.Node{Nameid: receiverid},
        Type: model.TensionTypeOperational,
        Status: model.TensionStatusOpen,
        Title: form.Title,
        Comments: []*model.Comment{
            &model.Comment{
                CreatedAt: createdAt,
                CreatedBy: &createdBy,
                Message: form.Msg,
            },
        },
        Subscribers: []*model.User{&model.User{Username: uctx.Username}},
    }

    // Verify author can create tension
    var eventRef model.EventRef
    tools.StructMap(event, &eventRef)
    ok, _, err := graph.ProcessEvent(uctx, &tension, &eventRef, nil, nil, true, false)
    if !ok || err != nil {
        http.Error(w, "NOT AUTHORIZED TO CREATE TENSION HERE", 400); return
    }

    // Create tension
    var tensionInput  model.AddTensionInput
    tools.StructMap(tension, &tensionInput)
    tid, err := db.GetDB().Add(*uctx, "tension", tensionInput)
    if err != nil {
        http.Error(w, err.Error(), 400); return
    }

    // Publish Notification
    // --
    notif := model.EventNotif{
        Uctx: uctx,
        Tid: tid,
        History: []*model.EventRef{&eventRef},
    }
    // Push notification
    if err := graph.PushEventNotifications(notif); err != nil {
        http.Error(w, "PushEventNotifications error: " + err.Error(), 500); return
    }

}

// Handle Postal WebHook - redirect it to a matrix channel
func PostalWebhook(w http.ResponseWriter, r *http.Request) {
    // Validate WebHook identity
    if err := tools.ValidatePostalSignature(r, postalWebhookPK); err != nil {
        http.Error(w, err.Error(), 400); return
    }

    // Get request string
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), 400); return
    }

    err = tools.MatrixJsonSend(string(body), matrixPostalRoom, matrixToken)
    if err != nil {
        http.Error(w, err.Error(), 400); return
    }
}
