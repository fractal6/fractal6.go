package graph

import (
    "fmt"
    "context"

    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/graph/codec"
    "fractale/fractal6.go/graph/auth"
    "fractale/fractal6.go/db"
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
    // Validation defined how the tension should be validated. Typically, what kind of contract
    // is used to validated the tension. If Validation is nil, no contract are create and the tension
    // and we just check if the Authorization hook. Else the validations are defined in the function mapped
    // to it. see the validationMap map.
    Validation model.ContractType
    // Auth defined a set of authorization hooks that can be satisfied by a user to process the event.
    Auth AuthHookValue
    // Defined a propertie/variable the should be updated by the event (taking value from the event old/new attributes)
    Propagate string
    // Action definfed the fonction that should be executed if the user has been authorized.
    Action func(*model.UserCtx, *model.Tension, *model.EventRef, *model.BlobRef) (bool, error)
}
type EventsMap = map[model.TensionEvent]EventMap

// Validation ~ Contract
// Validation function return a triplet:
// ok bool -> ok means the contract has been validated and can be closed.
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
    OwnerHook AuthHookValue        = 1 << 1 // @DEBUG: Not used for now as the owner is implemented in CheckUserRights
    MemberHook AuthHookValue       = 1 << 2
    MemberActiveHook AuthHookValue = 1 << 3
    SourceCoordoHook AuthHookValue = 1 << 4
    TargetCoordoHook AuthHookValue = 1 << 5
    // Granted based
    AuthorHook AuthHookValue       = 1 << 6
    AssigneeHook AuthHookValue     = 1 << 7
    // Contract based
    CandidateHook AuthHookValue    = 1 << 8
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

func (em EventMap) Check(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    var ok bool
    var err error
    var hookEnabled bool =(
        em.Validation == "" ||
        (contract == nil && GetBlob(tension) == nil)) // Moving node, doc etc

    if tension == nil || event == nil {
        return false, nil, fmt.Errorf("non existing tension or event not allowed")
    }

    // Exception Hook Authorization
    // --
    if hookEnabled {
        ok, err = em.checkTensionAuth(uctx, tension, event, contract)
        if ok || err != nil { return ok, contract, err }
    }

    if contract != nil {
        // Exit if contract is not open
        if contract.Status != "" && contract.Status != model.ContractStatusOpen {
            return ok, contract, fmt.Errorf("Contract status is closed or missing.")
        }
    }

    // Contract Authorization
    // --
    f := validationMap[em.Validation]
    if f == nil { return false, nil, fmt.Errorf("not implemented contract type.") }
    return f(em, uctx, tension, event, contract)
}


/*
 *
 * Tension Event Authorization implementation
 *
 */


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
        rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
        if auth.UserIsMember(uctx, rootid) >= 0 { return true, err }
    }

    if TargetCoordoHook & em.Auth > 0 {
        ok, err := auth.HasCoordoRole(uctx, tension.Receiver.Nameid, &tension.Receiver.Mode)
        if ok { return ok, err }
    }

    if SourceCoordoHook & em.Auth > 0 {
        ok, err := auth.HasCoordoRole(uctx, tension.Emitter.Nameid, &tension.Emitter.Mode)
        if ok { return ok,  err }
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
    if err != nil { return false, nil, err }
    if !ok {
        return false, contract, err
    }

    if contract == nil {
        // If user is participant return true whitout a contract
        if uctx.Username == *event.New {
            return true, nil, nil
        } else {
            // Futur: return the contract instead of using addContract ?
            return false, nil, fmt.Errorf("User addContract query instead.")
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
            return false, contract, err
        }
    } else {
        return false, contract, err
    }
}

func AnyCoordoSource(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    panic("not implemented.")
}

func AnyCoordoTarget(em EventMap, uctx *model.UserCtx, tension *model.Tension, event *model.EventRef, contract *model.Contract) (bool, *model.Contract, error) {
    panic("not implemented.")
}


////////////////////////////////////////////////
// With Ctx method (used in graph/resolver.go) -- (could be done in Dgraph Lamba)
////////////////////////////////////////////////

// Check if an user owns the given object
func CheckUserOwnership(ctx context.Context, uctx *model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    var username string
    var err error
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        // Non user type here (userField/createdBy must be present)
        id := ctx.Value("id")
        if id == nil || id .(string) == "" {
            return false, fmt.Errorf("object target unknown(id), see setContextWithID...")
        }
        // Request the database to get the field
        // @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface: ToTypeName(reflect.TypeOf(nodeObj).String())
        username_, err := db.GetDB().GetSubFieldById(id.(string), "Post."+userField, "User.username")
        if err != nil { return false, err }
        username = username_.(string)
    } else {
        // User here
        username = user.(model.JsonAtom)["username"].(string)
    }

    // Check user ID match
    return uctx.Username == username, err
}

