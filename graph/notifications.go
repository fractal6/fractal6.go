package graph

import (
	//"fmt"
	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/model"
	. "zerogov/fractal6.go/tools"
)

func PushHistory(uctx *model.UserCtx, tid string, evts []*model.EventRef) error {
    var inputs []model.AddEventInput
    for _, e := range evts {
        // Build AddtensionInput
        var temp model.AddEventInput
        StructMap(e, &temp)
        temp.Tension = &model.TensionRef{ID: &tid}

        // Push AddtensionInput
        inputs = append(inputs, temp)
    }
    // Push events
    ids, err := db.GetDB().AddMany(*uctx, "event", inputs)
    if err != nil { return err }
    // Set event ids for further notifications
    for i, id := range ids {
        evts[i].ID = &id
    }
    return err
}

// Notify users for Event events, where events can be batch of event.
// @performance: @defer this with Redis
func PushEventNotifications(tid string, evts []*model.EventRef) error {
    // Only the event with an ID will be notified.
    var eventBatch []*model.EventKindRef
    var createdAt string
    for i, e := range evts {
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
    users, err := GetUsersToNotify(tid, true, true)
    if err != nil { return err }

    // Push user event notification
    for _, u := range users {
        // suscription list
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
func PushContractNotifications(tid string, contract *model.Contract) error {
    // Only the event with an ID will be notified.
    var eventBatch []*model.EventKindRef
    var createdAt string
    if contract == nil {
        return nil
    }
    createdAt = contract.CreatedAt
    eventBatch = append(eventBatch, &model.EventKindRef{ContractRef: &model.ContractRef{ID: &contract.ID}})

    // Get people to notify
    users, err := GetUsersToNotify(tid, true, false)
    if err != nil { return err }
    // Add Candidates
    for _, c := range contract.Candidates {
        for _, dup := range users {
            if dup == c.Username { break }
        }
        users = append(users, c.Username)
    }

    // Push user event notification
    for _, u := range users {
        //@TODO
        // don't self notify.
        //if u == uctx.Username { continue }
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