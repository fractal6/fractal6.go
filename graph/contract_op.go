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
func contractEventHook(uctx *model.UserCtx, tid string, event *model.EventRef, bid *string) (bool, *model.Contract, error) {
    var ok bool = true
    var err error
    var tension *model.Tension
    var contract *model.Contract
    if event == nil {
        return false, nil, LogErr("Access denied", fmt.Errorf("No event given."))
    }

    if tension == nil {
        // Fetch Tension, target Node and blob charac (last if bid undefined)
        tension, err = db.GetDB().GetTensionHook(tid, false, nil)
        if err != nil { return false, nil, LogErr("Access denied", err) }
    }

    // Process event
    ok, contract, err = processEvent(uctx, &CEMAP, tension, event, true, true, true)

    return ok, contract, err
}

func processVote(uctx *model.UserCtx, cid string, vid string, vote int) (bool, model.ContractStatus, error) {
    var ok bool = false
    var status model.ContractStatus = model.ContractStatusOpen
    contract, err := db.GetDB().GetContractHook(cid)
    if err != nil { return ok, status, LogErr("Access denied", err) }
    if contract == nil { return  ok, status, LogErr("Access denied", fmt.Errorf("contract not found.")) }

    // Exit if contract is not open
    if contract.Status != model.ContractStatusOpen {
        return ok, status, LogErr("Access denied", fmt.Errorf("Contract is closed."))
    }

    isValidator, tension, c, err := hasContractRight(uctx, contract)
    if err != nil { return ok, status, err }
    isCandidate := IsCandidate(uctx, contract)
    //updated := HasParticipated(uctx, contract, vid) # always true

    switch contract.ContractType {
    case model.ContractTypeAnyParticipants:
        if isCandidate {
            ok = true
        }
        if vote == 1 {
            status = model.ContractStatusClosed
        } else if vote == 0 {
            status = model.ContractStatusCanceled
        }
    case model.ContractTypeAnyCoordoDual:
        if isValidator {
            ok = true
        }
        if vote == 1 && c == nil {
            status = model.ContractStatusClosed
        } else if vote == 0 {
            status = model.ContractStatusCanceled
        }
    case model.ContractTypeAnyCoordoSource:
        err = fmt.Errorf("not implemented contract type.")
    case model.ContractTypeAnyCoordoTarget:
        err = fmt.Errorf("not implemented contract type.")
    default:
        err = fmt.Errorf("not implemented contract type.")
    }

    if status == model.ContractStatusClosed {
        // Process Event
        var event model.EventRef
        StructMap(contract.Event, &event)
        ok, _, err = processEvent(uctx, nil, tension, &event, false, true, true)
        if err != nil { return false, status, err }
        if ok {
            // leave trace
            leaveTrace(tension)
            // Set status
            db.GetDB().SetFieldById(cid, "Contract.status", string(status))
        }
    }

    return ok, status, err
}

//hasContractRight check if user has validation rights.
func hasContractRight(uctx *model.UserCtx, contract *model.Contract) (bool, *model.Tension, *model.Contract, error) {
    var event model.EventRef
    StructMap(contract.Event, &event)

    // Get Tension, target Node and blob charac (last if bid undefined)
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, false, nil)
    if err != nil { return false, nil, nil, LogErr("Access denied", err) }

    ok, c, err := processEvent(uctx, &CEMAP, tension, &event, true, false, false)
    return ok, tension, c, err
}


//IsCandidate check if user is a contract candidate candidate.
func IsCandidate(uctx *model.UserCtx, contract *model.Contract) (bool) {
    for _, c := range contract.Candidates {
        if c.Username == uctx.Username {
            return true
        }
    }
    return false
}

//HasParticipated check if user is a contract candidate candidate.
func HasParticipated(uctx *model.UserCtx, contract *model.Contract, vid string) (bool) {
    for _, p := range contract.Participants {
        if p.ID == vid {
            return true
        }
    }
    return false
}


