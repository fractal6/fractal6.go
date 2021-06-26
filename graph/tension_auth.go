package graph

import (
    "fmt"
    "context"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
    . "zerogov/fractal6.go/tools"
)

// Node Action Rights Enum.
// Each node has a rights value (literal) which is set of activated rights.
// Those rights are encoded as a XOR between the different possible actions.
// Note that the `authEventsLut` map which rights are needed for each event to
// be triggered.
type AuthValue int
const (
    Creating       = 1
    Reopening      = 1 << 1
    Closing        = 1 << 2
    TitleUpdating  = 1 << 3
    CommentPushing = 1 << 4
)
var authEventsLut map[model.TensionEvent]AuthValue

// Authorization Hook enum.
// Each event have a set of hook activated to allow user
// to trigger an event.
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
)

type EventMap struct {
    Validation model.ContractType
    Auth AuthHookValue
    Action func(*model.UserCtx, *model.Tension, *model.EventRef) (bool, error)
}
type EventsMap = map[model.TensionEvent]EventMap

func init() {
    authEventsLut = map[model.TensionEvent]AuthValue{
        model.TensionEventCreated       : Creating,
        model.TensionEventReopened      : Reopening,
        model.TensionEventClosed        : Closing,
        model.TensionEventTitleUpdated  : TitleUpdating,
        model.TensionEventCommentPushed : CommentPushing,
    }
}

func (em EventMap) Check(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    var err error
    var hookEnabled bool =(
        em.Validation != model.ContractTypeAnyCoordoDual ||
        GetBlob(tension) == nil) // Moving node, doc etc

    // Exception Hook
    // --

    if em.Auth == PassingHook {
        return true, nil, err
    }

    // <!> Bot Hook <!>
    // If emitter is a Bot, check its rights
    if tension.Emitter.RoleType != nil && model.RoleTypeBot == *tension.Emitter.RoleType &&
    (tension.Emitter.Rights & int(authEventsLut[*event.EventType])) > 0 {
        // Can only create tension in the parent circle og the bot.
        if pid, _ := codec.Nid2pid(tension.Emitter.Nameid); pid == tension.Receiver.Nameid {
            return true, nil, err
        } else {
            return false, nil, fmt.Errorf("The tension receiver only support the following node: %s", pid)
        }
    }

    // Check Hook authorization
    // --

    if AuthorHook & em.Auth > 0 && hookEnabled {
        // isAuthorCheck: Check if the user is the creator of the ressource
        if uctx.Username == tension.CreatedBy.Username {
            return true, nil, err
        }
    }

    if MemberHook & em.Auth > 0 && hookEnabled {
        rootid, err := codec.Nid2rootid(tension.Receiver.Nameid)
        if auth.UserIsMember(uctx, rootid) >= 0 { return true, nil, err }
    }

    if TargetCoordoHook & em.Auth > 0 && hookEnabled {
        ok, err := auth.HasCoordoRole(uctx, tension.Receiver.Nameid, tension.Receiver.Charac)
        if ok { return ok, nil, err }
    }

    if SourceCoordoHook & em.Auth > 0 && hookEnabled {
        ok, err := auth.HasCoordoRole(uctx, tension.Emitter.Nameid, tension.Emitter.Charac)
        if ok { return ok, nil, err }
    }

    if AssigneeHook & em.Auth > 0 && hookEnabled {
        // isAssigneeCheck: Check if the user is an assignee of the curent tension
        // @debug: use checkAssignee function, but how to pass the context ?
        var assignees []interface{}
        res, err := db.GetDB().GetSubFieldById(tension.ID, "Tension.assignees", "User.username")
        if err != nil { return false, nil, err }
        if res != nil { assignees = res.([]interface{}) }
        for _, a := range(assignees) {
            if a.(string) == uctx.Username {
                return true, nil, err
            }
        }
    }

    // Check the contract authorization
    // --
    var f func(*model.UserCtx, *model.Tension, *model.EventRef) (bool, *model.Contract, error)
    switch em.Validation {
    case model.ContractTypeAnyParticipants:
        f = AnyParticipants
    case model.ContractTypeAnyCoordoDual:
        f = AnyCoordoDual
    case model.ContractTypeAnyCoordoSource:
        f = AnyCoordoSource
    case model.ContractTypeAnyCoordoTarget:
        f = AnyCoordoTarget
    case "": // Empty, blocking
        return false, nil, err
    default:
        return false, nil, fmt.Errorf("not implemented contract type.")
    }
    return f(uctx, tension, event)
}

func AnyParticipants(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    ok, _, err := AnyCoordoTarget(uctx, tension, event)
    if err != nil { return false, nil, err }

    if ok {
        //@TODO
        //return contract
        panic("not implemented.")
        //return false, nil, nil
    } else {
        return false, nil, nil
    }
}

func AnyCoordoDual(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    if event.Old == nil || event.New == nil { return false, nil, fmt.Errorf("old and new event data must be defined.") }
    // @debug manege event.old values in general ?
    event.Old = &tension.Receiver.Nameid
    nameidNew := *event.New
    // Source (old destination)
    ok1, err := auth.HasCoordoRole(uctx, tension.Receiver.Nameid, tension.Receiver.Charac)
    if err != nil { return false, nil, err }

    // Target (New destination)
    ok2, err := auth.HasCoordoRole(uctx, nameidNew, nil)
    if err != nil { return false, nil, err }

    if ok1 && ok2 {
        return true, nil, err
    } else if ok1 || ok2 {
        var ev model.EventFragment
        StructMap(*event, &ev)
        var rid string
        if ok1 {
            rid, _ = codec.Nid2rootid(tension.Receiver.Nameid)
        } else if ok2 {
            rid, _ = codec.Nid2rootid(nameidNew)
        }
        contract := &model.Contract{
            CreatedAt: Now(),
            CreatedBy: &model.User{Username: uctx.Username},
            Event: &ev,
            Tension: tension,
            Status: model.ContractStatusOpen,
            ContractType: model.ContractTypeAnyCoordoDual,
            Participants: []*model.Vote{&model.Vote{
                Node: &model.Node{Nameid: codec.MemberIdCodec(rid, uctx.Username)},
                Data: []int{1},
            }, },
        }
        return false, contract, err
    } else {
        return false, nil, err
    }
}

func AnyCoordoSource(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    panic("not implemented.")
}

func AnyCoordoTarget(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    panic("not implemented.")
}


////////////////////////////////////////////////
// With Ctx method (used in graph/resolver.go)
////////////////////////////////////////////////

// Check if an user owns the given object
func CheckUserOwnership(ctx context.Context, uctx *model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    var username string
    var err error
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        // Tension here
        id := ctx.Value("id")
        if id == nil || id .(string) == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
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

func GetNodeCharacStrict() model.NodeCharac {
    return model.NodeCharac{UserCanJoin: false, Mode: model.NodeModeCoordinated}
}

