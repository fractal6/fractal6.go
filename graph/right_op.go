package graph

import (
    "fmt"
    "context"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
    webauth "zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)

// User Action Rights Enum
type AuthValue int
const (
    Creating       = 1
    Reopening      = 1 << 1
    Closing        = 1 << 2
    TitleUpdating  = 1 << 3
    CommentPushing = 1 << 4
)
var authEventsLut map[model.TensionEvent]AuthValue

// Authorization Hook enum
type AuthHookValue int
const (
    PassingHook AuthHookValue      = 1 // for public event
    // Graph Role based
    OwnerHook AuthHookValue        = 1 << 1 // @DEBUG: Not used for now as the owner is implemented in CheckUserRights
    MemberHook AuthHookValue       = 1 << 2
    SourceCoordoHook AuthHookValue = 1 << 3
    TargetCoordoHook AuthHookValue = 1 << 4
    // Granted based
    AuthorHook AuthHookValue       = 1 << 5
    AssigneeHook AuthHookValue     = 1 << 6
)

type EventMap struct {
    Validation model.ContractType
    Auth AuthHookValue
    Action func(*model.UserCtx, *model.Tension, *model.EventRef) (bool, error)
}

type EventsMap = map[model.TensionEvent]EventMap
var EMAP EventsMap

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
    var hookEnabled bool = !(em.Validation == model.ContractTypeAnyCoordoDual && GetBlob(tension) != nil )
    // Check Hook authorization
    // --

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

    if AuthorHook & em.Auth == 1 && hookEnabled {
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
        ok, err := HasCoordoRole(uctx, tension.Receiver.Nameid, tension.Receiver.Charac)
        if ok { return ok, nil, err }
    }

    if SourceCoordoHook & em.Auth > 0 && hookEnabled {
        ok, err := HasCoordoRole(uctx, tension.Emitter.Nameid, tension.Emitter.Charac)
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
    // Source
    ok1, err := HasCoordoRole(uctx, tension.Emitter.Nameid, tension.Emitter.Charac)
    if err != nil { return false, nil, err }

    // Target
    ok2, err := HasCoordoRole(uctx, tension.Receiver.Nameid, tension.Receiver.Charac)
    if err != nil { return false, nil, err }

    if ok1 && ok2 {
        return true, nil, err
    } else if ok1 || ok2 {
        var ev model.EventFragment
        StructMap(*event, &ev)
        var rid string
        if ok1 {
            rid, _ = codec.Nid2rootid(tension.Emitter.Nameid)
        } else if ok2 {
            rid, _ = codec.Nid2rootid(tension.Receiver.Nameid)
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
// Base authorization methods
////////////////////////////////////////////////

func HasCoordoRole(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    // Check user rights
    ok, err := CheckUserRights(uctx, nameid, charac)
    if err != nil { return ok, LogErr("Internal error", err) }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(nameid) {
        ok, err = CheckUpperRights(uctx, nameid, charac)
    }
    return ok, err
}

// ChechUserRight return true if the user has access right (e.g. Coordo) on the given node
func CheckUserRights(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    // Get the nearest circle
    if codec.IsRole(nameid) {
        nameid, _ = codec.Nid2pid(nameid)
    }

    // Escape if the user is an owner
    rootnameid, _ := codec.Nid2rootid(nameid)
    if auth.UserIsOwner(uctx, rootnameid) >= 0 { return true, err }

    // Get the mode of the node
    if charac == nil {
        charac, err = db.GetDB().GetNodeCharac("nameid", nameid)
        if err != nil { return ok, LogErr("Internal error", err) }
    }

    if charac.Mode == model.NodeModeAgile {
        ok = auth.UserHasRole(uctx, nameid) >= 0
    } else if charac.Mode == model.NodeModeCoordinated {
        ok = auth.UserIsCoordo(uctx, nameid) >= 0
    }

    return ok, err
}

// chechUpperRight return true if the user has access right (e.g. Coordo) on any on its parents.
func CheckUpperRights(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, error) {
    var ok bool
    parents, err := db.GetDB().GetParents(nameid)
    if err != nil { return ok, LogErr("Internal Error", err) }
    if len(parents) == 0 { return ok, err }

    for _, p := range(parents) {
        ok, err = CheckUserRights(uctx, p, charac)
        if err != nil { return ok, LogErr("Internal error", err) }
        if ok { break }
    }

    return ok, err
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
        id := ctx.Value("id").(string)
        if id == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        // Request the database to get the field
        // @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface: ToTypeName(reflect.TypeOf(nodeObj).String())
        username_, err := db.GetDB().GetSubFieldById(id, "Post."+userField, "User.username")
        if err != nil { return false, err }
        username = username_.(string)
    } else {
        // User here
        username = user.(model.JsonAtom)["username"].(string)
    }

    // Check user ID match
    return uctx.Username == username, err
}

// check if the an user has the given role of the given (nested) node
func CheckAssignees(ctx context.Context, uctx *model.UserCtx, nodeObj interface{}) (bool, error) {
    // Check that nodes are present
    var assignees []interface{}
    var err error
    id_ := ctx.Value("id")
    nameid_ := ctx.Value("nameid")
    if id_ != nil {
        // Tension here
        res, err := db.GetDB().GetSubFieldById(id_.(string), "Tension.assignees", "User.username")
        if err != nil { return false, err }
        if res != nil { assignees = res.([]interface{}) }
    } else if (nameid_ != nil) {
        // Node Here
        res, err := db.GetDB().GetSubSubFieldByEq("Node.nameid", nameid_.(string), "Node.source", "Tension.assignees", "User.username")
        if err != nil { return false, err }
        if res != nil { assignees = res.([]interface{}) }
    } else {
        return false, fmt.Errorf("node target unknown, need a database request here...")
    }

    // Search for assignees
    for _, a := range(assignees) {
        if a.(string) == uctx.Username {
            return true, err
        }
    }
    return false, err
}

func GetNodeCharacStrict() model.NodeCharac {
    return model.NodeCharac{UserCanJoin: false, Mode: model.NodeModeCoordinated}
}

func isHidePrivate(ctx context.Context, nameid string, isPrivate bool) (bool, error) {
    var yes bool = true
    var err error

    if nameid == "" {
        err = LogErr("Access denied", fmt.Errorf("`nameid' field is required."))
    } else {
        // Get the public status of the node
        //isPrivate, err :=  db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.isPrivate")
        //if err != nil {
        //    return yes, LogErr("Access denied", err)
        //}
        if isPrivate {
            // check user role.
            uctx, err := webauth.UserCtxFromContext(ctx)
            //if err == jwtwebauth.ErrExpired {
            //    // Uctx claims is not parsed for unverified token
            //    u, err := db.GetDB().GetUser("username", uctx.Username)
            //    if err != nil { return yes, LogErr("internal error", err) }
            //    err = nil
            //    uctx = *u
            if err != nil { return yes, LogErr("Access denied", err) }

            rootnameid, err := codec.Nid2rootid(nameid)
            if auth.UserIsMember(uctx, rootnameid) >= 0 {
                return false, err
            }
        } else {
            yes = false
        }
    }
    return yes, err
}

