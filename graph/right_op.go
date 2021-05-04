package graph

import (
    "fmt"
    "context"

    . "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/codec"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
)


// Authorization Hook enum
type AuthHookValue int
const (
    OwnerHook AuthHookValue = 1 // @DEBUG: Not used for now as the owner is implemented in CheckUserRights
    AuthorHook AuthHookValue = 1 << 1
    AssigneeHook AuthHookValue = 1 << 2
)

type EventMap struct {
    Auth model.ContractType
    AuthHook AuthHookValue
    Action func(*model.UserCtx, *model.Tension, *model.EventRef) (bool, error)
}

type EventsMap = map[model.TensionEvent]EventMap
var EMAP EventsMap

func (em EventMap) Check(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    var err error
    var hookEnabled bool = !(em.Auth == model.ContractTypeAnyCoordoDual && GetBlob(tension) != nil )
    // Check Hook authorization
    // --
    if em.AuthHook & AuthorHook == 1 && hookEnabled {
        // isAuthorCheck: Check if the user is the creator of the ressource
        if uctx.Username == tension.CreatedBy.Username {
            return true, nil, err
        }
    }

    if em.AuthHook & AssigneeHook == 1 && hookEnabled {
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
    switch em.Auth {
    case model.ContractTypeAnyParticipants:
        f = AnyParticipants
    case model.ContractTypeAnyCoordoDual:
        f = AnyCoordoDual
    case model.ContractTypeAnyCoordoSource:
        f = AnyCoordoSource
    case model.ContractTypeAnyCoordoTarget:
        f = AnyCoordoTarget
    case "": // Empty, passing
        return true, nil, err
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
        return false, nil, nil
    } else {
        return false, nil, nil
    }
}

func AnyCoordoDual(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    ok1, _, err := AnyCoordoSource(uctx, tension, event)
    if err != nil { return false, nil, err }

    ok2, _, err := AnyCoordoTarget(uctx, tension, event)
    if err != nil { return false, nil, err }

    if ok1 && ok2 {
        return true, nil, err
    } else if ok1 || ok2 {
        var ev model.Event
        StructMap(*event, &ev)
        var node model.Node
        if ok1 {
            node.Nameid = tension.Emitter.Nameid
        } else if ok2 {
            node.Nameid = tension.Receiver.Nameid
        }
        contract := &model.Contract{
            Event: &ev,
            Tension: tension,
            Status: model.ContractStatusOpen,
            ContractType: model.ContractTypeAnyCoordoDual,
            Participants: []*model.Vote{&model.Vote{
                Node: &node,
                Data: []int{1},
            }, },
        }
        return false, contract, err
    } else {
        return false, nil, err
    }
}

func AnyCoordoSource(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    return AnyCoordo(uctx, tension.Emitter.Nameid, tension.Emitter.Charac)
}

func AnyCoordoTarget(uctx *model.UserCtx, tension *model.Tension, event *model.EventRef) (bool, *model.Contract, error) {
    return AnyCoordo(uctx, tension.Receiver.Nameid, tension.Receiver.Charac)
}

//
// Base authaurisation methods
//

func AnyCoordo(uctx *model.UserCtx, nameid string, charac *model.NodeCharac) (bool, *model.Contract, error) {
    // Check user rights
    ok, err := CheckUserRights(uctx, nameid, charac)
    if err != nil { return ok, nil, LogErr("Internal error", err) }

    // Check if user has rights in any parents if the node has no Coordo role.
    if !ok && !db.GetDB().HasCoordos(nameid) {
        ok, err = CheckUpperRights(uctx, nameid, charac)
    }
    return ok, nil, err
}

// chechUserRight return true if the user has access right (e.g. Coordo) on the given node
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
        ok = auth.UserIsMember(uctx, nameid) >= 0
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

//
// With Ctx method (used in graph/resolver.go)
//

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
