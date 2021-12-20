package graph

import (
	"fmt"

	"zerogov/fractal6.go/db"
	"zerogov/fractal6.go/graph/model"
	. "zerogov/fractal6.go/tools"
)

// contractEventHook is applied for addContract query directives.
// Take action based on the given Event. The targeted tension is fetch (see TensionHookPayload).
// All events in History must pass.
func contractEventHook(uctx *model.UserCtx, cid, tid string, event *model.EventRef, bid *string) (bool, error) {
    var ok bool = true
    var err error
    var tension *model.Tension
    if event == nil {
        return false, fmt.Errorf("No event given.")
    }
    if tension == nil {
        // Fetch Tension, target Node and blob charac (last if bid undefined)
        // @DEBUG: blob is not always needed (Moving non node tension, Invite, etc)
        tension, err = db.GetDB().GetTensionHook(tid, true, nil)
        if err != nil { return false, err }
    }

    // Fetch the contract
    contract, err := db.GetDB().GetContractHook(cid)
    if err != nil { return false, err }
    if contract == nil { return false, fmt.Errorf("contract not found.") }

    // Process event
    ok, contract, err = processEvent(uctx, tension, event, nil, contract, true, true)
    ok = ok || contract != nil

    if contract != nil && err != nil {
        // Post-contract action
        if contract.Event.EventType == model.TensionEventMemberLinked || contract.Event.EventType == model.TensionEventUserJoined {
            // Add pending Nodes
            for _, c := range contract.Candidates {
                err = AddPendingNode(c.Username, tension)
                if err != nil { return false, err }
            }
        }

        // @TODO here:
        // PushContractNotification and dend email (watch special case for pendingCandidates).
        // * notify only the participants and authorized participants (coordo)
        //for _, c := range contract.PendingCandidates {
        //    // @todo
        //    fmt.Println(c.Email)
        //}
    }

    return ok, err
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

    if ok {
        // @TODO here:
        // 1. push the event
        //err = PushHistory(uctx, tension.ID, [contract.event])
        // 2. do pushEventNotification, with that event !
        //PushEventNotifications(tension.ID, [contract.event])
    }

    return ok, contract, err
}

//hasContractRight check if user has validation rights (Coordo right like).
func hasContractRight(uctx *model.UserCtx, contract *model.Contract) (bool, error) {
    var event model.EventRef
    StructMap(contract.Event, &event)

    // Exit if contract is not open
    if contract.Status != "" && contract.Status != model.ContractStatusOpen {
        return false, fmt.Errorf("Contract status is closed or missing.")
    }

    // Get linked tension
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, false, nil)
    if err != nil { return false, err }

    ok, c, err := processEvent(uctx, tension, &event, nil, nil, true, false)
    return ok||c!=nil, err
}


