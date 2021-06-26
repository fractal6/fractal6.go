package graph

import (
	"fmt"

	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/model"
	. "zerogov/fractal6.go/tools"
)

var CEMAP EventsMap

func init() {
    CEMAP = EventsMap{
        model.TensionEventCreated: EventMap{ },
        model.TensionEventCommentPushed: EventMap{ },
        model.TensionEventBlobCreated: EventMap{ },
        model.TensionEventBlobCommitted: EventMap{ },
        model.TensionEventTitleUpdated: EventMap{ },
        model.TensionEventReopened: EventMap{ },
        model.TensionEventClosed: EventMap{ },
        model.TensionEventLabelAdded: EventMap{ },
        model.TensionEventLabelRemoved: EventMap{ },
        model.TensionEventAssigneeAdded: EventMap{ },
        model.TensionEventAssigneeRemoved: EventMap{ },
        // --- Trigger Action ---
        model.TensionEventBlobPushed: EventMap{ },
        model.TensionEventBlobArchived: EventMap{ },
        model.TensionEventBlobUnarchived: EventMap{ },
        model.TensionEventUserLeft: EventMap{ },
        model.TensionEventUserJoined: EventMap{
            Auth: SourceCoordoHook,
        },
        model.TensionEventMoved: EventMap{
            Auth: AuthorHook | SourceCoordoHook | TargetCoordoHook | AssigneeHook,
        },
    }
}


// contractEventHook is applied for addContract query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload).
// All events in History must pass.
func contractEventHook(uctx *model.UserCtx, tid string, events []*model.EventRef, bid *string) (bool, *model.Contract, error) {
    var ok bool = true
    var err error
    var tension *model.Tension
    var contract *model.Contract
    if events == nil {
        return false, nil, LogErr("Access denied", fmt.Errorf("No event given."))
    }

    for _, event := range(events) {
        em, hasEvent := CEMAP[*event.EventType]
        if hasEvent { // Process the special event
               if tension == nil {
                   // Get Tension, target Node and blob charac (last if bid undefined)
                   tension, err = db.GetDB().GetTensionHook(tid, false, nil)
                   if err != nil { return false, nil, LogErr("Access denied", err) }
                   if tension == nil { return false, nil, LogErr("Access denied", fmt.Errorf("tension not found.")) }
               }

               // Check Authorization (optionally generate a contract)
               ok, contract, err = em.Check(uctx, tension, event)
               if err != nil { return ok, nil, err }
               if !ok { break }

               if em.Action != nil { // Trigger Action
                   ok, err = em.Action(uctx, tension, event)
               }

               // Notify users
               // push notification (somewhere ?!)

           } else {
               // Minimum level of authorization
               return false, nil, LogErr("Access denied", fmt.Errorf("Event not implemented."))
           }
    }

    return ok, contract, err
}

