package graph

import (
	"fmt"
    "context"
    "encoding/json"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

var ctx context.Context = context.Background()

func PublishTensionEvent(notif model.EventNotif) error {
    payload, _ := json.Marshal(notif)
    if err := cache.Publish(ctx, "api-tension-notification", payload).Err(); err != nil {
        fmt.Printf("Redis publish error: %v", err)
        panic(err)
    }

    return nil
}

func PublishContractEvent(notif model.ContractNotif) error {
    payload, _ := json.Marshal(notif)
    if err := cache.Publish(ctx, "api-contract-notification", payload).Err(); err != nil {
        fmt.Printf("Redis publish error: %v", err)
        panic(err)
    }

    return nil
}

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

// GetUserToNotify returns a list of user should receive notifications uponf tension updates.
func GetUsersToNotify(tid string, withAssigness, withSuscribers bool) ([]string, error) {
    us := []string{}
    users := make(map[string]bool)

    // Get Coordos @TODO
    // * Get direct coordos
    // * Get node first_link yf any (watchout when linking/unlink)
    if withAssigness {
        // Get Assignees
        res, err := db.GetDB().GetSubFieldById(tid, "Tension.assignees", "User.username")
        if err != nil { return us, err }
        assignees := InterfaceToStringSlice(res)
        // Append without duplicate
        for _, u := range assignees {
            if users[u] { continue }
            users[u] = true
        }
    }
    if withSuscribers {
        // Get Subscribers
        res, err := db.GetDB().GetSubFieldById(tid, "Tension.subscribers", "User.username")
        if err != nil { return us, err }
        subscribers := InterfaceToStringSlice(res)
        // Append without duplicate
        for _, u := range subscribers {
            if users[u] { continue }
            users[u] = true
        }
    }

    // @todo: Check go 1.19 for generic and maps.Keys !
    for u, _ := range users {
        us = append(us, u)
    }

    return us, nil
}

// Notify users for Event events, where events can be batch of event.
// @performance: @defer this with Redis
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
    users, err := GetUsersToNotify(notif.Tid, true, true)
    if err != nil { return err }

    // Push user event notification
    for _, u := range users {
        // Don't self notify.
        if u == notif.Uctx.Username { continue }

        // User Event
        _, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
            User: &model.UserRef{Username: &u},
            IsRead: false,
            CreatedAt: createdAt,
            Event: eventBatch,
        })
        if err != nil { return err }

        // Email
        // if user.notiftByEmail {
        //   sendEventNotificationEmail(u, eventBatch)
        // }
    }

    return err
}

// Notify users for Contract event.
// @performance: @defer this with Redis
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
    // Add Candidates
    for _, c := range notif.Contract.Candidates {
        for _, dup := range users {
            if dup == c.Username { break }
        }
        users = append(users, c.Username)
    }

    // Push user event notification
    for _, u := range users {
        // Don't self notify.
        if u == notif.Uctx.Username { continue }

        // User Event
        _, err = db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
            User: &model.UserRef{Username: &u},
            IsRead: false,
            CreatedAt: createdAt,
            Event: eventBatch,
        })
        if err != nil { return err }

        // Email
        // if user.notiftByEmail {
        //   sendContractNotificationEmail(u, eventBatch)
        // }
    }

    return err
}

