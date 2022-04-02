package graph

import (
	"fmt"
	"context"
    "strings"
    re "regexp"
	"encoding/json"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/auth"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/graph/codec"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/email"
)

var ctx context.Context = context.Background()
var user_selection string = "User.username User.email User.name User.notifyByEmail"

//
// Publisher functions (Redis)
//

// Will trigger processTensionNotification in cmd/notifier.go
// and PushEventNotifications
func PublishTensionEvent(notif model.EventNotif) error {
    payload, _ := json.Marshal(notif)
    if err := cache.Publish(ctx, "api-tension-notification", payload).Err(); err != nil {
        fmt.Printf("Redis publish error: %v", err)
        panic(err)
    }

    return nil
}

// Will trigger processContractNotification in cmd/notifier.go
// and PushContractNotifications
func PublishContractEvent(notif model.ContractNotif) error {
    payload, _ := json.Marshal(notif)
    if err := cache.Publish(ctx, "api-contract-notification", payload).Err(); err != nil {
        fmt.Printf("Redis publish error: %v", err)
        panic(err)
    }

    return nil
}

// Will trigger processNotifNotification in cmd/notifier.go
// and PushNotifNotifications
func PublishNotifEvent(notif model.NotifNotif) error {
    payload, _ := json.Marshal(notif)
    if err := cache.Publish(ctx, "api-notif-notification", payload).Err(); err != nil {
        fmt.Printf("Redis publish error: %v", err)
        panic(err)
    }

    return nil
}

//
// User helpers
//

// GetUserToNotify returns a list of user should receive notifications uponf tension updates.
func GetUsersToNotify(tid string, withAssignees, withSubscribers bool) (map[string]model.UserNotifInfo, error) {
    users := make(map[string]model.UserNotifInfo)

    {
        // Get Coordos
        coordos, err := auth.GetCoordosFromTid(tid)
        if err != nil { return users, err }
        for _, user := range coordos {
            if _, ex := users[user.Username]; ex { continue }
            users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsCoordo}
        }
    }

    {
        // Get First-link
        res, err := db.GetDB().GetSubSubFieldById(tid, "Tension.receiver", "Node.first_link", user_selection)
        if err != nil { return users, err }
        if res != nil {
            var user model.User
            if err := Map2Struct(res.(model.JsonAtom), &user); err == nil {
                if _, ex := users[user.Username]; !ex {
                    users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsFirstLink}
                }
            }
        }
    }

    if withAssignees {
        // Get Assignees
        res, err := db.GetDB().GetSubFieldById(tid, "Tension.assignees", user_selection)
        if err != nil { return users, err }
        if assignees, ok := InterfaceSlice(res); ok {
            for _, u := range assignees {
                var user model.User
                if err := Map2Struct(u.(model.JsonAtom), &user); err == nil {
                    if _, ex := users[user.Username]; ex { continue }
                    users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsAssignee}
                }
            }
        }
    }

    if withSubscribers {
        // Get Subscribers
        res, err := db.GetDB().GetSubFieldById(tid, "Tension.subscribers", user_selection)
        if err != nil { return users, err }
        if subscribers, ok := InterfaceSlice(res); ok {
            for _, u := range subscribers {
                var user model.User
                if err := Map2Struct(u.(model.JsonAtom), &user); err == nil {
                    if _, ex := users[user.Username]; ex { continue }
                    users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsSubscriber}
                }
            }
        }
    }


    return users, nil
}

// Update the users map with notified users.
// Note: only add user that are member of the given rootnameid
// @debug: user inside block code will be notified here...
func UpdateWithMentionnedUser(msg string, nid string, users map[string]model.UserNotifInfo) error {
    rootnameid, err := codec.Nid2rootid(nid)
    if err != nil { return err}
    r := re.MustCompile(`(^|\s|[^\w\[\` + "`" + `])@([\w\-\.]+)\b`)
    for i, U := range r.FindStringSubmatch(msg) {
        u := strings.ToLower(U)
        if _, ex := users[u]; ex { continue }
        // Check that user is a member
        filter := `has(Node.first_link) AND NOT eq(Node.role_type, "Pending") AND NOT eq(Node.role_type, "Retired")`
        if ex, _ := db.GetDB().Exists("Node.nameid", codec.MemberIdCodec(rootnameid, u), &filter); !ex { continue }
        res, err := db.GetDB().GetFieldByEq("User.username", u, user_selection)
        if err != nil { return err }
        if res != nil {
            var user model.User
            if err := Map2Struct(res.(model.JsonAtom), &user); err == nil {
                fmt.Println("=== 3")
                users[u] = model.UserNotifInfo{User: user, Reason: model.ReasonIsMentionned}
            }
        }
        if i > 100 {
            return fmt.Errorf("Too many user memtioned. Please consider using an Alert tension to notify group of users.")
        }
    }
    return nil
}


//
// Notifiers functions
//

/* INTERNAL (websocket, platform notification etc) */

// PushHistory publish event to a tension history.
func PushHistory(notif *model.EventNotif) error {
    var inputs []model.AddEventInput
    for _, e := range notif.History {
        // Build AddtensionInput
        var temp model.AddEventInput
        StructMap(e, &temp)
        temp.Tension = &model.TensionRef{ID: &notif.Tid}

        // Push AddtensionInput
        inputs = append(inputs, temp)
    }
    // Push events
    ids, err := db.GetDB().AddMany(*notif.Uctx, "event", inputs)
    if err != nil { return err }
    // Set event ids for further notifications
    for i, id := range ids {
        notif.History[i].ID = &id
    }
    return err
}

/* EXTERNAL (email, chat, etc) */

// Notify users for Event events, where events can be batch of event.
func PushEventNotifications(notif model.EventNotif) error {
    // Only the event with an ID will be notified.
    var eventBatch []*model.EventKindRef
    var createdAt string
    for i, e := range notif.History {
        if i == 0 {
            createdAt = *e.CreatedAt
        }
        if *e.ID != "" {
            eventBatch = append(eventBatch, &model.EventKindRef{EventRef: &model.EventRef{ID: e.ID}})
        }
    }
    if len(eventBatch) == 0 {
        return nil
    }

    // Get people to notify
    users := make(map[string]model.UserNotifInfo)
	var err error
    var isAlert bool
    var receiverid string
    if notif.HasEvent(model.TensionEventCreated) {
        if t, err := db.GetDB().GetFieldById(notif.Tid, "Tension.type_ Tension.receiverid"); err != nil {
            return err
        } else if t != nil {
            tension := t.(model.JsonAtom)
            isAlert = tension["type_"].(string) == string(model.TensionTypeAlert)
            receiverid = tension["receiverid"].(string)
        }
    }
    // Handle Alert tension
    if isAlert {
        // Alert tension created: Notify everyone
        if data, err := db.GetDB().GetSubMembers("nameid", receiverid, user_selection); err != nil {
            return err
        } else {
            for _, n := range data {
                user := *n.FirstLink
                if _, ex := users[user.Username]; ex { continue }
                users[user.Username] = model.UserNotifInfo{User: user, Reason: model.ReasonIsAlert}
            }
        }
    } else {
        // Notify only suscribers and relative.
        users, err = GetUsersToNotify(notif.Tid, true, true)
        if err != nil { return err }
    }

    // Add mentionned and **set tension data**
    if m, err := db.GetDB().Meta("getLastComment", map[string]string{"tid":notif.Tid}); err != nil {
        return err
    } else if len(m) > 0 {
        notif.Receiverid = m[0]["receiverid"].(string)
        notif.Msg, _ = m[0]["message"].(string)
        notif.Title = m[0]["title"].(string)
        if notif.Msg != "" {
            UpdateWithMentionnedUser(notif.Msg, notif.Receiverid, users)
        }
    } else {
        return fmt.Errorf("tension %s not found.", notif.Tid)
    }

    // Push user event notification
    for u, ui := range users {
        // Don't self notify.
        if u == notif.Uctx.Username { continue }
        // Pending user has no history yet
        if ui.Reason == model.ReasonIsPendingCandidate { continue }

        // User Event
        eid, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
            User: &model.UserRef{Username: &u},
            IsRead: false,
            CreatedAt: createdAt,
            Event: eventBatch,
        })
        if err != nil { return err }

        // Email
         if notif.Uctx.Rights.HasEmailNotifications && ui.User.NotifyByEmail {
             fmt.Println("send event notif: ",  u)
             ui.Eid = eid
             err = email.SendEventNotificationEmail(ui, notif)
             if err != nil { return err }
        }
    }

    return err
}

// Notify users for Contract event.
func PushContractNotifications(notif model.ContractNotif) error {
    // Only the event with an ID will be notified.
    var eventBatch []*model.EventKindRef
    var createdAt string
    if notif.Contract == nil {
        return nil
    }
    createdAt = notif.Contract.CreatedAt
    eventBatch = append(eventBatch, &model.EventKindRef{ContractRef: &model.ContractRef{ID: &notif.Contract.ID}})

    // Get people to notify
    users, err := GetUsersToNotify(notif.Tid, true, false)
    if err != nil { return err }
    // +
    // Add Candidates
    for _, c := range notif.Contract.Candidates {
        users[c.Username] = model.UserNotifInfo{User: *c, Reason: model.ReasonIsCandidate}
    }
    // +
    // Add Pending Candidates
    for _, c := range notif.Contract.PendingCandidates {
        if c.Email == nil { continue }
        users[*c.Email] = model.UserNotifInfo{User: model.User{Email: *c.Email}, Reason: model.ReasonIsPendingCandidate}
    }
    // +
    // Add Participants
    for _, p := range notif.Contract.Participants {
        if _, ex := users[p.Node.FirstLink.Username]; ex { continue }
        users[p.Node.FirstLink.Username] = model.UserNotifInfo{User: *p.Node.FirstLink, Reason: model.ReasonIsParticipant}
    }
    // +
    // Add mentionned and **set tension data**
    if m, err := db.GetDB().Meta("getLastContractComment", map[string]string{"cid":notif.Contract.ID}); err != nil {
        return err
    } else if len(m) > 0 {
        notif.Receiverid = m[0]["receiverid"].(string)
        notif.Msg, _ = m[0]["message"].(string)
        if notif.Msg != "" {
            UpdateWithMentionnedUser(notif.Msg, notif.Receiverid, users)
        }
    } else {
        return fmt.Errorf("contract %s not found.", notif.Tid)
    }

    // Push user event notification
    for u, ui := range users {
        // Don't self notify.
        if u == notif.Uctx.Username { continue }

        // User Event
        var eid string
        if ui.Reason == model.ReasonIsPendingCandidate {
            // Update pending users
            err = MaybeSetPendingUserToken(u)
            if err != nil { return err }
            // Link contract
            err = db.GetDB().Update(db.GetDB().GetRootUctx(), "pendingUser", &model.UpdatePendingUserInput{
                Filter: &model.PendingUserFilter{Email: &model.StringHashFilter{Eq: &u}},
                Set: &model.PendingUserPatch{Contracts: []*model.ContractRef{&model.ContractRef{ID: &notif.Contract.ID}}},
            })
            if err != nil { return err }
        } else {
            switch notif.ContractEvent {
            case model.NewContract:
                // Push user event
                eid, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
                    User: &model.UserRef{Username: &u},
                    IsRead: false,
                    CreatedAt: createdAt,
                    Event: eventBatch,
                })
                if err != nil { return err }
            case model.NewComment:
                // Push user notif
                PushNotifNotifications(model.NotifNotif{
                    Uctx: notif.Uctx,
                    Tid: &notif.Tid,
                    Cid: &notif.Contract.ID,
                    Msg: "You have a new comment",
                    To: []string{u},
                })
            }
        }

        // Email
        if notif.Uctx.Rights.HasEmailNotifications && (ui.User.NotifyByEmail || ui.Reason == model.ReasonIsPendingCandidate) &&
        (ui.Reason == model.ReasonIsCandidate ||
        ui.Reason == model.ReasonIsPendingCandidate ||
        ui.Reason == model.ReasonIsParticipant ||
        ui.Reason == model.ReasonIsCoordo ||
        ui.Reason == model.ReasonIsAssignee ||
        ui.Reason == model.ReasonIsMentionned) {
             ui.Eid = eid
             err = email.SendContractNotificationEmail(ui, notif)
             if err != nil { return err }
        }
    }

    return err
}

// Notify users for Notif events.
func PushNotifNotifications(notif model.NotifNotif) error {
    // Only the event with an ID will be notified.
    var eventBatch []*model.EventKindRef
    var createdAt string = Now()
    var tensionRef model.TensionRef
    var contractRef model.ContractRef
    if notif.Tid != nil {
        tensionRef = model.TensionRef{ID: notif.Tid}
    }
    if notif.Cid != nil {
        contractRef = model.ContractRef{ID: notif.Cid}
    }

    eventBatch = append(eventBatch, &model.EventKindRef{NotifRef: &model.NotifRef{
        CreatedAt: &createdAt,
        CreatedBy: &model.UserRef{Username: &notif.Uctx.Username},
        Message: &notif.Msg,
        Tension: &tensionRef,
        Contract: &contractRef,
    }})


    // Push user event notification
    for _, u := range notif.To {
        // Don't self notify.
        if u == notif.Uctx.Username { continue }

        // User Event
        _, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
            User: &model.UserRef{Username: &u},
            IsRead: false,
            CreatedAt: createdAt,
            Event: eventBatch,
        })
        if err != nil { return err }

        // Email
        // No email for this one
    }

    return nil
}
