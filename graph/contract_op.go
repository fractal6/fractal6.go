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

package graph

import (
	"fmt"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/graph/codec"
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
            rootid, err := codec.Nid2rootid(contract.Tension.Receiverid)
            if err != nil { return false, contract,  err }
            if rootid != contract.Tension.Receiverid {
                return false, contract, fmt.Errorf("Only the root circle can be joined.")
            }

            if i := auth.IsMember("username", c.Username, contract.Tension.Receiverid); i >= 0 {
                return false, contract, fmt.Errorf("Candidate '%s' is already member.", c.Username)
            }
        }
        for _, c := range contract.PendingCandidates {
            if c.Email == nil { continue }
            if i := auth.IsMember("email", *c.Email, contract.Tension.Receiverid); i >= 0 {
                return false, contract, fmt.Errorf("Candidate '%s' is already member.", *c.Email)
            }
        }
    case model.TensionEventMemberLinked:
        // pass, this shouldn't be a security flaw.
        // @todo: check if role has already a first-link.
    default:
        if contract.Candidates != nil {
            return false, contract, fmt.Errorf("Contract candidates not implemented for this event (contract).")
        }
    }

    // Process event
    ok, contract, err = ProcessEvent(uctx, tension, event, nil, contract, true, true)
    return (ok || contract != nil), contract, err
}

func voteEventHook(uctx *model.UserCtx, cid string) (bool, *model.Contract, error) {
    var ok bool = false

    // Fetch the contract
    contract, err := db.GetDB().GetContractHook(cid)
    if err != nil { return ok, contract, err }
    if contract == nil { return ok, contract, fmt.Errorf("contract not found.") }

    // Fetch linked tension
    // @DEBUG: blob is not always needed (Moving non node tension, Invite, etc)
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, true, nil)
    if err != nil { return false,  nil, err }

    // Process event
    var event model.EventRef
    StructMap(contract.Event, &event)
    ok, contract, err = ProcessEvent(uctx, tension, &event, nil, contract, true, true)
    if contract == nil || err != nil { return false, contract, err }

    // Mark contract as read
    _, err = db.GetDB().Meta("markContractAsRead", map[string]string{
        "username": uctx.Username,
        "id": contract.ID,
    })
    if err != nil { return false, contract, err }

    return ok, contract, err
}

// HasContractRight check if user has validation rights (Coordo right like).
func HasContractRight(uctx *model.UserCtx, contract *model.Contract) (bool, error) {
    var event model.EventRef
    StructMap(contract.Event, &event)

    if contract == nil { return false, fmt.Errorf("Contract not found") }
    if contract.Tension == nil { return false, fmt.Errorf("Tension not found in contract") }

    // Get linked tension
    tension, err := db.GetDB().GetTensionHook(contract.Tension.ID, false, nil)
    if err != nil { return false, err }

    ok, c, err := ProcessEvent(uctx, tension, &event, nil, nil, true, false)
    return ok || c != nil, err
}


