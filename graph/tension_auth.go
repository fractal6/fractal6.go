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
    "fractale/fractal6.go/web/auth"
    "fractale/fractal6.go/db"
    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    . "fractale/fractal6.go/tools"
)


// EventMap structure contains all the information needed to process a given event.
//
// If an event has a Validation value, the tension need to satisfy the authorization in a bidirectional way.
// The rules in both directions must be implemented in the coresponding functions.
//     I.e. To invite an user, a AnyCandidates validation method should be set,
//     and the action will be the processed once any the candidate (or any candidates)
//     has been validated by any participants (the one that satisfy the given authorization).
//
type EventMap struct {
    // Validation defined how the tension should be validated according to corresponding event.
    // It is implemented by the kind of contract used to validated the tension.
    // - If Validation is nil, no contract are create and the tension  and we just check if the Authorization hook.
    // - Else the validations are defined in the function mapped to it. see the validationMap map.
    Validation model.ContractType
    // Auth defined rules that restrict the users that can create the corresponding event.
    Auth AuthHookValue
    // Restrict defined rules to be respected acording the event values
    Restrict []RestrictValue
    // Defined a propertie/variable the should be updated by the event (taking value from the event old/new attributes)
    Propagate string
    // Action defined the fonction that should be executed if the user has been authorized.
    Action func(*model.UserCtx, *model.Tension, *model.EventRef, *model.BlobRef) (bool, error)
}
type EventsMap = map[model.TensionEvent]EventMap

// Validation ~ Contract
// Validation function return a triplet:
// ok bool -> ok means the contract has been validated and the event can be processed.
// contract -> returns the updated contract if is has been altered else nil
// err -> is something got wrong
var validationMap map[model.ContractType]func(EventMap, *model.UserCtx, *model.Tension, *model.EventRef, *model.Contract) (bool, *model.Contract, error)


/*
*
* @FUTURE: Those the following structure should be defined automatically or in the schema ?
*
*/


// Authorization **Hook** Enum.
// Each event have a set of hook activated to allow users to trigger an event.
type AuthHookValue int
const (
    PassingHook AuthHookValue      = 1 // for public event
    // Graph Role based
    OwnerHook AuthHookValue        = 1 << 1 // @DEBUG: Not used for now as the owner is implemented in CheckUserAuth
    MemberHook AuthHookValue       = 1 << 2
    MemberStrictHook AuthHookValue = 1 << 3
    MemberActiveHook AuthHookValue = 1 << 4
    SourceCoordoHook AuthHookValue = 1 << 5
    TargetCoordoHook AuthHookValue = 1 << 6
    // Granted based
    AuthorHook AuthHookValue       = 1 << 7
    AssigneeHook AuthHookValue     = 1 << 8
    // Contract based
    CandidateHook AuthHookValue    = 1 << 9
)


// RestrictValue defined condition to be validated based on event values.
type RestrictValue int
const (
    NoRestriction RestrictValue = 1 // default
    UserIsMemberRestrict RestrictValue = 1 << 1 // the user (asking) should be a member of the receiver circle
    UserNewIsMemberRestrict RestrictValue = 1 << 2 // the new user (event.new) should be a member of the receiver circle
)

// Node Action **Rights** Enum.
// Each node has a rights value (literal) which represents a set of activated rights.
// Those rights are encoded as a XOR between the different possible actions.
// Note that the `authEventsLut` map which rights are needed for each event to
// be triggered.
type AuthValue int
const (
    Creating       = 1
    Reopening      = 1 << 1
    Closing        = 1 << 2
    TitleUpdating  = 1 << 3
    TypeUpdating   = 1 << 4
    CommentPushing = 1 << 5
    // To be completed
)
var authEventsLut map[model.TensionEvent]AuthValue


func init() {

    validationMap = map[model.ContractType]func(EventMap, *model.UserCtx, *model.Tension, *model.EventRef, *model.Contract) (bool, *model.Contract, error){
        model.ContractTypeAnyCandidates   : AnyCandidates,
        model.ContractTypeAnyCoordoDual   : AnyCoordoDual,
        model.ContractTypeAnyCoordoSource : AnyCoordoSource,
        model.ContractTypeAnyCoordoTarget : AnyCoordoTarget,
    }

    authEventsLut = map[model.TensionEvent]AuthValue{
        model.TensionEventCreated       : Creating,
        model.TensionEventReopened      : Reopening,
        model.TensionEventClosed        : Closing,
        model.TensionEventTitleUpdated  : TitleUpdating,
        model.TensionEventTypeUpdated   : TypeUpdating,
        model.TensionEventCommentPushed : CommentPushing,
    }
}

// Check if a tension event can be processed.
// Returns a triple following the ValidationMap function semantics.
func (em EventMap) Check(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    var ok bool
    var err error
    var hookEnabled bool =(
        em.Validation == "" ||
        (contract == nil && GetBlob(tension) == nil)) // Moving node, doc etc

    if tension == nil || event == nil {
        return false, nil, fmt.Errorf("non existing tension or event not allowed")
    }

    // Restriction check for authorizsation
    // --
    ok, err = em.checkTensionRestriction(uctx, tension, event, contract)
    if !ok || err != nil { return ok, contract, err }

    // Exception Hook Authorization (EventMap:Auth)
    // --
    if hookEnabled {
        ok, err = em.checkTensionAuth(uctx, tension, event, contract)
        if ok || err != nil { return ok, contract, err }
    }

    if contract != nil {
        // Exit if contract is not open
        if contract.Status != "" && contract.Status != model.ContractStatusOpen {
            return false, nil, fmt.Errorf("Contract status is closed or missing.")
        }
    }

    // Contract Authorization (EventMap:Validation)
    // --
    f := validationMap[em.Validation]
    if f == nil { return false, nil, LogErr("Contract not implemened", fmt.Errorf("Contact a coordinator to access this ressource.")) }
    return f(em, uctx, tension, event, contract)

}


// checkTensionRestriction checks the tension can be processed based specific restriction
func (em EventMap) checkTensionRestriction(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, error) {
    var ok bool = true
    var err error
    if len(em.Restrict) == 0 { return ok, err }

    // Each member of the list is a OR RestrictValue,
    // the list is an AND of the RestrictValue.
    for _, restrict := range em.Restrict {
        if restrict == NoRestriction {
            continue
        }

        if restrict & UserIsMemberRestrict > 0 {
            if i := auth.UserIsMember(uctx, tension.Receiver.Nameid); i >= 0 {
                continue
            }
        }

        if restrict & UserNewIsMemberRestrict > 0 {
            if event.New != nil {
                if i := auth.IsMember("username", *event.New, tension.Receiver.Nameid); i >= 0 {
                    continue
                }
            }
        }

        ok = false
        break
    }

    return ok, err
}


// checkTensionAuth checks the tension can be processed based on graph based properties of the user asking.
func (em EventMap) checkTensionAuth(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, error) {
    var err error

    if em.Auth == PassingHook {
        return true, err
    }

    // <!> Bot Hook <!>
    // If emitter is a Bot, check its rights
    if tension.Emitter.RoleType != nil && *tension.Emitter.RoleType == model.RoleTypeBot &&
    (tension.Emitter.Rights & int(authEventsLut[*event.EventType])) > 0 {
        // Can only create tension in the parent circle of the bot.
        // @DEBUG: run the BOT logics here...
        if pid, _ := codec.Nid2pid(tension.Emitter.Nameid); pid == tension.Receiver.Nameid {
            return true, err
        } else {
            return false, fmt.Errorf("The tension receiver only support the following node: %s", pid)
        }
    }

    // Check Hook authorization
    // --

    if AuthorHook & em.Auth > 0 {
        // isAuthorCheck: Check if the user is the creator of the ressource
        if uctx.Username == tension.CreatedBy.Username {
            return true, err
        }
    }

    if MemberHook & em.Auth > 0 {
        if auth.UserIsMember(uctx, tension.Receiver.Nameid) >= 0 {
            return true, err
        }
    }

    if MemberStrictHook & em.Auth > 0 {
        // Check guest right or membership
        if auth.UserIsGuest(uctx, tension.Receiver.Nameid) >= 0 {
            rid, _ := codec.Nid2rootid(tension.Receiver.Nameid)
            r, err := db.GetDB().GetFieldByEq("Node.nameid", rid, "Node.guestCanCreateTension")
            if err != nil { return false, err }
            if r != nil && r.(bool) {
                return true, err
            } else {
                return false, fmt.Errorf("Sorry, Guest cannot create tension in this organisation at the moment.")
            }
        } else if auth.UserIsMember(uctx, tension.Receiver.Nameid) >= 0 {
            return true, err
        }
    }

    if TargetCoordoHook & em.Auth > 0 {
        ok, err := auth.HasCoordoAuth(uctx, tension.Receiver.Nameid, &tension.Receiver.Mode)
        if ok { return ok, err }
    }

    if SourceCoordoHook & em.Auth > 0 {
        ok, err := auth.HasCoordoAuth(uctx, tension.Emitter.Nameid, &tension.Emitter.Mode)
        if ok { return ok, err }
    }

    if AssigneeHook & em.Auth > 0 {
        // isAssigneeCheck: Check if the user is an assignee of the curent tension
        // @debug: use checkAssignee function, but how to pass the context ?
        var assignees []interface{}
        res, err := db.GetDB().GetSubFieldById(tension.ID, "Tension.assignees", "User.username")
        if err != nil { return false, err }
        if res != nil { assignees = res.([]interface{}) }
        for _, a := range(assignees) {
            if a.(string) == uctx.Username {
                return true, err
            }
        }
    }

    if CandidateHook & em.Auth > 0 && contract != nil {
        // Check if uctx is a contract candidate
        for _, c := range contract.Candidates {
            if c.Username == uctx.Username && len(contract.Candidates) == 1 {
                return true, err
            }
        }
    }

    return false, err
}


/*
 *
 * Contract validation implementations
 *
 */


func AnyCandidates(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    ok, err := em.checkTensionAuth(uctx, tension, event, contract)
    if !ok || err != nil { return false, nil, err }

    if contract == nil {
        if uctx.Username == *event.New {
            // If user is participant and have rights,
            // return true whitout a contract => self invitation
            ok, err := auth.HasCoordoAuth(uctx, tension.Receiver.Nameid, &tension.Receiver.Mode)
            return ok, nil, err
        } else {
            // Futur: return the contract instead of using addContract ?
            return false, nil, fmt.Errorf("Use addContract query instead.")
        }
    }

    if len(contract.Candidates) !=1 {
        // Only one candidate supported for now.
        if len(contract.PendingCandidates) !=1 {
            return false, nil, fmt.Errorf("Contract with no candidate.")
        }
    }

    // Check Vote
    // @Debug don't allow more than two vote....
    upVote := 0
    downVote := 0
    candidateVote := 0
    for _, p := range contract.Participants {
        if (p.Node.FirstLink == nil) { continue }

        if len(contract.Candidates) == 1 && p.Node.FirstLink.Username == contract.Candidates[0].Username {
            // Candidate
            if p.Data[0] == 1 {
                candidateVote = 1
            } else {
                candidateVote = -1
            }
        } else {
            // Coordinator
            if p.Data[0] == 1 {
                upVote += 1
            } else {
                downVote += 1
            }
        }
    }

    // if two vote (coordo + candidate) -> ok
    if candidateVote > 0 && upVote > 0 {
        contract.Status = model.ContractStatusClosed
        return true, contract, err
    } else if candidateVote < 0 || downVote > 0 {
        contract.Status = model.ContractStatusCanceled
        return true, contract, err
    } else {
        return false, contract, err
    }
}

func AnyCoordoDual(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    if event.Old == nil || event.New == nil { return false, nil, fmt.Errorf("old and new event data must be defined.") }
    // @debug manage event.old values in general ?
    if *event.Old != tension.Receiver.Nameid {
        return false, nil, fmt.Errorf("Contract outdated: event source (%s) and actual source (%s) differ. Please, refresh or remove this contract.", *event.Old, tension.Receiver.Nameid)
    }

    nameidNew := *event.New

    // Source (old destination)
    ok1, err := em.checkTensionAuth(uctx, tension, event, contract)
    if err != nil { return false, nil, err }

    // Fetch tension target/Dual
    tid2, _ := db.GetDB().GetSubSubFieldByEq("Node.nameid", nameidNew, "Node.source", "Blob.tension", "uid")
    if tid2 == nil { return false, nil, fmt.Errorf("tension source not found.") }
    tension2, err := db.GetDB().GetTensionHook(tid2.(string), false, nil)
    if err != nil { return false, nil, err }
    if tension2 == nil { return false, nil, fmt.Errorf("target tension fetch failed.") }

    // Target (new destination)
    ok2, err := em.checkTensionAuth(uctx, tension2, event, contract)
    if err != nil { return false, nil, err }

    // The (contract == nil) check means that the contract is not created yet.
    if (ok1 && ok2) && contract == nil {
        return true, contract, err
    } else if (ok1 || ok2) && contract == nil {
        var ev model.EventFragment
        StructMap(*event, &ev)
        var rid string
        if ok1 {
            rid, _ = codec.Nid2rootid(tension.Receiver.Nameid)
        } else if ok2 {
            rid, _ = codec.Nid2rootid(nameidNew)
        }
        contractid := codec.ContractIdCodec(tension.ID, *event.EventType, *event.Old, *event.New)
        contract := &model.Contract{
            //Contractid: contractid, // Build in the frontend.
            CreatedAt: Now(),
            CreatedBy: &model.User{Username: uctx.Username},
            Event: &ev,
            Tension: tension,
            Status: model.ContractStatusOpen,
            ContractType: model.ContractTypeAnyCoordoDual,
            Participants: []*model.Vote{&model.Vote{
                Voteid: codec.VoteIdCodec(contractid, rid, uctx.Username),
                Node: &model.Node{Nameid: codec.MemberIdCodec(rid, uctx.Username)},
                Data: []int{1},
            }, },
        }
        return false, contract, err
    } else if ok1 || ok2 {
        // Check Votes
        // @Debug don't allow more than two vote....
        upVote := 0
        downVote := 0
        for _, p := range contract.Participants {
            if p.Data[0] == 1 {
                upVote += 1
            } else {
                downVote += 1
            }
        }

        // if two votes (source-coordo + target-coordo) -> ok
        if upVote >= 2 {
            contract.Status = model.ContractStatusClosed
            return true, contract, err
        } else if downVote >= upVote {
            contract.Status = model.ContractStatusCanceled
            return true, contract, err
        } else {
            // @future: Agile mode: any user can create a contract ?
            return false, nil, err
        }
    } else {
        return false, nil, err
    }
}

func AnyCoordoSource(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    panic("not implemented.")
}

func AnyCoordoTarget(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    panic("not implemented.")
}


