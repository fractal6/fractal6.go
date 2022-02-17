package graph

import (
	"fmt"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/graph/auth"
	. "fractale/fractal6.go/tools"
)

// contractEventHook is applied for addContract query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload).
// All events in History must pass.
func contractEventHook(uctx *model.UserCtx, cid, tid string, event *model.EventRef, bid *string) (bool, *model.Contract, error) {
    var ok bool = true
    var err error
    var tension *model.Tension
    if event == nil {
        return false, nil, fmt.Errorf("No event given.")
    }
    if tension == nil {
        // Fetch Tension, target Node and blob charac (last if bid undefined)
        // @DEBUG: blob is not always needed (Moving non node tension, Invite, etc)
        tension, err = db.GetDB().GetTensionHook(tid, true, nil)
        if err != nil { return false, nil, err }
    }

    // Fetch the contract
    contract, err := db.GetDB().GetContractHook(cid)
    if err != nil { return false, contract, err }
    if contract == nil { return false, contract, fmt.Errorf("contract not found.") }

    // Validate Candidates
    // for now...
    if len(contract.Candidates) > 1 {
        return false, contract, fmt.Errorf("Candidate need to be singleton for security reason.")
    }
    switch contract.Event.EventType {
    case model.TensionEventUserJoined:
        for _, c := range contract.Candidates {
            if i := auth.IsMember(c.Username, contract.Tension.Receiverid); i >= 0 {
                return false, contract, fmt.Errorf("A candidate is already member.")
            }
        }
    case model.TensionEventMemberLinked:
        // pass, this shouldn't be a security flaw.
    default:
        if contract.Candidates != nil {
            return false, contract, fmt.Errorf("Contract candidates not implemented for this event (contract).")
        }
    }

    // Process event
    ok, contract, err = processEvent(uctx, tension, event, nil, contract, true, true)
    if err != nil { return false, contract, err }
    ok = ok || contract != nil
    if contract == nil { return ok, contract, err }

    // Push Notifications
    PublishContractEvent(model.ContractNotif{Uctx: uctx, Tid: tid, Contract: contract})
    //for _, c := range contract.PendingCandidates {
    //    // @todo: send signup+contract invitation
    //    fmt.Println(c.Email)
    //}

    return ok, contract, err
}

func processVote(uctx *model.UserCtx, cid string) (bool, *model.Contract, error) {
    var ok bool = false

    // Fetch the contract
    contract, err := db.GetDB().GetContractHook(cid)
    if err != nil { return ok, contract, err }
    if contract == nil { return  ok, contract, fmt.Errorf("contract not found.") }

    // Fetch linked tension
    // @DEBUG: blob is not always needed (Moving non node tension, Invite, etc)
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, true, nil)
    if err != nil { return false,  nil, err }

    // Process event
    var event model.EventRef
    StructMap(contract.Event, &event)
    ok, contract, err = processEvent(uctx, tension, &event, nil, contract, true, true)
    if err != nil { return false, contract, err }
    ok = ok || contract != nil
    if contract == nil { return ok, contract, err }

    // Mark contract as read
    _, err = db.GetDB().Meta("markContractAsRead", map[string]string{
        "username": uctx.Username,
        "id": contract.ID,
    })
    if err != nil { return false, contract, err }

    if contract.Status == model.ContractStatusClosed {
        now := Now()
        event.CreatedAt = &now
        event.CreatedBy = &model.UserRef{Username: &uctx.Username}

        // Push Event History and Notifications
        PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: tension.ID, History: []*model.EventRef{&event}})
    }

    return ok, contract, err
}

//hasContractRight check if user has validation rights (Coordo right like).
func hasContractRight(uctx *model.UserCtx, contract *model.Contract) (bool, error) {
    var event model.EventRef
    StructMap(contract.Event, &event)

    // Get linked tension
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, false, nil)
    if err != nil { return false, err }

    ok, c, err := processEvent(uctx, tension, &event, nil, nil, true, false)
    ok = ok || c != nil
    return ok, err
}


