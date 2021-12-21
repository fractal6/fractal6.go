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

// Notify users
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

    //
    // Get people to notify
    //
    users := make(map[string]bool)
    // Get Assignees
    res, err := db.GetDB().GetSubFieldById(tid, "Tension.assignees", "User.username")
    if err != nil { return err }
    assignees := InterfaceToStringSlice(res)
    // Get Coordos
    // * (will see later)
    // Get Subscriber
    res, err = db.GetDB().GetSubFieldById(tid, "Tension.subscribers", "User.username")
    if err != nil { return err }
    subscribers := InterfaceToStringSlice(res)
    // Append without duplicate
    for _, u := range assignees {
        if users[u] { continue }
        users[u] = true
    }
    for _, u := range subscribers {
        //@TODO or u == uctx.Username
        if users[u] { continue }
        users[u] = true
    }

    //
    // Push user event notification
    //
    for u, _ := range users {
        _, err := db.GetDB().Add(db.GetDB().GetRootUctx(), "userEvent", &model.AddUserEventInput{
            User: &model.UserRef{Username: &u},
            IsRead: false,
            CreatedAt: createdAt,
            Event: eventBatch,
        })
        if err != nil { return err }
    }

    //
    // Push user email notification
    //
    // if user.notiftByEmail {
    //  ...send email...
    // }

    return err
}
