//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
    "reflect"
    "strings"
    "github.com/99designs/gqlgen/graphql"

    . "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/db"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
)

//
// Resolver initialisation
//

// Mutation type Enum
type mutationType string
const (
    AddMut mutationType = "add"
    UpdateMut mutationType = "update"
    DelMut mutationType = "delete"
)
type MutationContext struct  {
    type_ mutationType
    argName string
}

type Resolver struct{
    // Pointer on Dgraph client
    db *db.Dgraph
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    r := Resolver{
        db:db.GetDB(),
    }

    // Dgraph directives
    c := gen.Config{Resolvers: &r}
    c.Directives.Id = nothing
    c.Directives.HasInverse = nothing2
    c.Directives.Search = nothing3

    //
    // Query
    //

    // Fields directives
    c.Directives.Hidden = hidden
    c.Directives.Count = count
    c.Directives.Meta_getNodeStats = getNodeStats

    // Objects directives
    c.Directives.HidePrivate = hidePrivate

    //
    // Mutation
    //

    // Add inputs directives
    c.Directives.Add_isOwner = isOwner

    // Update or Remove inputs directives
    c.Directives.Patch_isOwner = isOwner
    c.Directives.Patch_RO = readOnly
    c.Directives.Patch_hasRole = hasRole

    // Add, Update and Remove inputs directives
    c.Directives.Alter_toLower = toLower
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_assertType = assertType
    c.Directives.Alter_hasRole = hasRole
    c.Directives.Alter_hasRoot = hasRoot

    // Mutation Hook directives
    c.Directives.Hook_addNode = addNodeHook
    c.Directives.Hook_updateNode = updateNodeHook
    c.Directives.Hook_addTension = addTensionHook
    c.Directives.Hook_updateTension = updateTensionHook
    c.Directives.Hook_updateComment = updateCommentHook

    c.Directives.Hook_addNodePost = nothing
    c.Directives.Hook_updateNodePost = nothing
    c.Directives.Hook_addTensionPost = addTensionPostHook
    c.Directives.Hook_updateTensionPost = updateTensionPostHook
    c.Directives.Hook_updateCommentPost = nothing

    return c
}


/*
*
* Business logic layer methods
*
*/

func nothing(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    return next(ctx)
}

func nothing2(ctx context.Context, obj interface{}, next graphql.Resolver, key string) (interface{}, error) {
    return next(ctx)
}

func nothing3(ctx context.Context, obj interface{}, next graphql.Resolver, idx []model.DgraphIndex) (interface{}, error) {
    return next(ctx)
}

//
// Query
//

func hidePrivate(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Check that isPrivate is present in payload
    var fok, yes bool
    var err error
    var nameid string

    data, err := next(ctx)
    if obj == nil {
        switch v := data.(type) {
        // Get Node
        case *model.Node:
            if v == nil {
                break
            }
            // Check fields
            for _, f := range GetPreloads(ctx) { if f == "isPrivate" {fok = true; break } }
            if !fok {
                // @debug: Query the database if field is not present and print a warning.
                return nil, LogErr("Access denied", fmt.Errorf("`isPrivate' field is required for this request."))
            }
            // Validate
            yes, err = isHidePrivate(ctx, v.Nameid, v.IsPrivate)
            if err != nil { return nil, err }
            if yes { return nil, LogErr("Access denied", fmt.Errorf("private node")) }
        // Query Nodes
        case []*model.Node:
            // Check fields
            for _, f := range GetPreloads(ctx) { if f == "isPrivate" {fok = true; break } }
            if !fok {
                // @debug: Query the database if field is not present and print a warning.
                return nil, LogErr("Access denied", fmt.Errorf("`isPrivate' field is required for this request."))
            }
            // Validate
            for _, node := range(v) {
                yes, err = isHidePrivate(ctx, node.Nameid, node.IsPrivate)
                if err != nil { return nil, err }
                if yes { return nil,  LogErr("Access denied", fmt.Errorf("private node")) }
            }
         // Get Tension
        case *model.Tension:
            if v == nil {
                break
            }
            // Check fields
            for _, f := range GetPreloads(ctx) { if f == "id" {fok = true; break } }
            if !fok {
                // @debug: Query the database if field is not present and print a warning.
                return nil, LogErr("Access denied", fmt.Errorf("`id' field is required for this request."))
            }
            if v.Receiver == nil {
                nameid_, err := db.GetDB().GetSubFieldById(v.ID, "Tension.receiver", "Node.nameid")
                if err != nil { return nil, err }
                nameid = nameid_.(string)
            } else {
                nameid = v.Receiver.Nameid
            }
            // validate
            isPrivate, err := db.GetDB().GetSubFieldById(v.ID, "Tension.receiver", "Node.isPrivate")
            if err != nil { return nil, err }
            yes, err = isHidePrivate(ctx, nameid, isPrivate.(bool))
            if err != nil { return nil, err }
            if yes { return nil, LogErr("Access denied", fmt.Errorf("private node")) }
        // Query Tensions
        case []*model.Tension:
            // Check fields
            for _, f := range GetPreloads(ctx) { if f == "id" {fok = true; break } }
            if !fok {
                // @debug: Query the database if field is not present and print a warning.
                return nil, LogErr("Access denied", fmt.Errorf("`id' field is required for this request."))
            }
            for _, tension := range(v) {
                if tension.Receiver == nil {
                    nameid_, err := db.GetDB().GetSubFieldById(tension.ID, "Tension.receiver", "Node.nameid")
                    if err != nil { return nil, err }
                    nameid = nameid_.(string)
                } else {
                    nameid = tension.Receiver.Nameid
                }
                // validate
                isPrivate, err := db.GetDB().GetSubFieldById(tension.ID, "Tension.receiver", "Node.isPrivate")
                if err != nil { return nil, err }
                yes, err = isHidePrivate(ctx, nameid, isPrivate.(bool))
                if err != nil { return nil, err }
                if yes { return nil,  LogErr("Access denied", fmt.Errorf("private node")) }
            }
        default:
            panic("@isPrivate: node type unknonwn: " + fmt.Sprintf("%T", v))
        }
    }
    return data, err
}

func isHidePrivate(ctx context.Context, nameid string, isPrivate bool) (bool, error) {
    var yes bool = true
    var err error

    if nameid == "" {
        err = LogErr("Access denied", fmt.Errorf("`nameid' field is required for this request."))
    } else {
        // Get the public status of the node
        //isPrivate, err :=  db.GetDB().GetFieldByEq("Node.nameid", nameid, "Node.isPrivate")
        //if err != nil {
        //    return yes, LogErr("Access denied", err)
        //}
        if isPrivate {
            // check user role.
            uctx, err := auth.UserCtxFromContext(ctx)
            //if err == jwtauth.ErrExpired {
            //    // Uctx claims is not parsed for unverified token
            //    u, err := db.GetDB().GetUser("username", uctx.Username)
            //    if err != nil { return yes, LogErr("internal error", err) }
            //    err = nil
            //    uctx = *u
            if err != nil { return yes, LogErr("Access denied", err) }

            rootnameid, err := nid2rootid(nameid)
            if userHasRoot(uctx, rootnameid) {
                return false, err
            }
        } else {
            yes = false
        }
    }
    return yes, err
}

func hidden(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    return nil, fmt.Errorf("`%s' field is hidden", fieldName)
}

func count(ctx context.Context, obj interface{}, next graphql.Resolver, field string) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    goFieldfDef := ToGoNameFormat(fieldName)

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    id := reflect.ValueOf(obj).Elem().FieldByName("ID").String()
    if id == "" {
        err := fmt.Errorf("`id' field is needed to query `%s'", fieldName)
        return nil, err
    }
    typeName := ToTypeName(reflect.TypeOf(obj).String())
    v := db.GetDB().Count(id, typeName, field)
    if v >= 0 {
        reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&v))
    }
    return next(ctx)
}

func getNodeStats(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    nameid := reflect.ValueOf(obj).Elem().FieldByName("Nameid").String()
    if nameid == "" {
        err := fmt.Errorf("`nameid' field is needed to query `%s'", fieldName)
        return nil, err
    }
    stats := db.GetDB().GetNodeStats(nameid)
    n_guest := stats["n_guest"]
    n_member := stats["n_member"]
    stats_ := model.NodeStats{
        NGuest: &n_guest,
        NMember: &n_member,
    }
    reflect.ValueOf(obj).Elem().FieldByName("Stats").Set(reflect.ValueOf(&stats_))
    //for k, v := range stats {
    //    goFieldfDef := ToGoNameFormat(k)
    //    reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&stats))
    //}
    return next(ctx)
}

//
// Mutation
//

func toLower(ctx context.Context, obj interface{}, next graphql.Resolver, field string) (interface{}, error) {
    data, err := next(ctx)
    return strings.ToLower(data.(string)), err
}

func inputMaxLength(ctx context.Context, obj interface{}, next graphql.Resolver, field string, max int) (interface{}, error) {
    v := obj.(model.JsonAtom)[field].(string)
    if len(v) > max {
        return nil, fmt.Errorf("`%s' to long. Maximum length is %d", field, max)
    }
    return next(ctx)
}

func assertType(ctx context.Context, obj interface{}, next graphql.Resolver, field string, type_ model.NodeType) (interface{}, error) {
    v := obj.(model.JsonAtom)[field].(model.JsonAtom)
    return nil, fmt.Errorf("only Node type is supported, this is not: %T", v )
}

// Add Node Hook:
// * addNode: New Orga (Root creation) -> check UserRight.CanCreateRoot
// * addNode: Join orga (push Node) -> check if NodeCharac.UserCanJoin is True and if user is not already a member
// * addNode: (push Node) Add role and subcircle
// Check user right for special query
func addNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    input := data.([]*model.AddNodeInput)
    if len(input) != 1 {
        return nil, LogErr("Add node error", fmt.Errorf("Only one node supported in input."))
    }
    node := *input[0]

    // Get the Node Characteristics of the **Parent Node**
    if node.Parent == nil && (*node.Parent).Nameid == nil {
        return nil, LogErr("Access denied", fmt.Errorf("Parent node not found."))
    }
    parentid := *(node.Parent).Nameid
    charac_, err := db.GetDB().GetNodeCharac("nameid", parentid)
    if err != nil { return nil, LogErr("Access denied", err) }
    if charac_ == nil { return nil, LogErr("Access denied", fmt.Errorf("Node characteristic not found.")) }
    fmt.Println(parentid, *charac_)

    ok, err := doAddNodeHook(uctx, node, parentid, *charac_)
    if err != nil { return nil, LogErr("Access denied", err) }
    if ok { return data, nil }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Node hook:
// * add the nameid field in the context for further inspection in new resolver
func updateNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    nameid_ := filter["nameid"].(model.JsonAtom)["eq"]
    if nameid_ != nil {
        ctx = context.WithValue(ctx, "nameid", nameid_.(string))
    }

    return next(ctx)
}

// Add Tension Hook
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    input := data.([]*model.AddTensionInput)
    if len(input) != 1 {
        return nil, LogErr("Add tension error", fmt.Errorf("Only one tension supported in input."))
    }

    return data, err
}

// Update Tension hook:
// * add the id field in the context for further inspection in new resolver
func updateTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) > 1 {
        return nil, LogErr("not implemented", fmt.Errorf("multiple post not supported"))
    }

    ctx = context.WithValue(ctx, "id", ids[0].(string))
    return next(ctx)
}

// Add Tension Post Hook
func addTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get Input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    input := rc.Args["input"].([]*model.AddTensionInput)[0]
    tid := data.(*model.AddTensionPayload).Tension[0].ID
    if tid == "" {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension payload"))
    }

    // Validate and process Blob Event
    ok, err := tensionBlobHook(uctx, tid, input.History, nil)
    if err != nil || !ok {
        // Delete the tension just added
        e := db.GetDB().DeleteNodes(tid)
        if e != nil { panic(e) }
    }

    if err != nil  { return nil, err }
    if ok { return data, err }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Tension Post Hook
func updateTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get Input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    input := rc.Args["input"].(model.UpdateTensionInput)
    tids := input.Filter.ID
    if len(tids) == 0 {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension filter."))
    }

    // Validate Blob Event
    if input.Set != nil && len(input.Set.Blobs) > 0 {
        bid := input.Set.Blobs[0].ID
        ok, err := tensionBlobHook(uctx, tids[0], input.Set.History, bid)
        if err != nil  { return nil, err }
        if ok {
            return data, err
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }

    return data, err
}

// Update Comment hook
// * add the id field in the context for further inspection in new resolver
func updateCommentHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) > 1 {
        return nil, LogErr("not implemented", fmt.Errorf("multiple post not supported"))
    }

    ctx = context.WithValue(ctx, "id", ids[0].(string))
    return next(ctx)
}

// HasRole check the user has the authorisation to update a ressource by checking if it satisfies at least one of
// 1. user rights
// 2. user ownership (u field)
// 3. check user role, (n r field)
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nFields []string, r model.RoleType, uField *string, assignee *int) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // If uField is given check if the current user is the creator ressource
    var ok bool
    if uField != nil {
        ok, err = checkUserOwnership(ctx, uctx, *uField, obj)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    // Check that user has the given role on the asked node
    for _, nField := range nFields {
        ok, err = checkUserRole(ctx, uctx, nField, obj, r)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    // Check is the user is an assigne of the curent tension
    if assignee != nil {
        ok, err = checkAssignees(ctx, uctx, obj)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    // Check if user is Coordinator of any parents if the PID has no coordinator
    if !ok && ctx.Value("nameid") != nil { // is a Node
        parents, err := db.GetDB().GetParents(ctx.Value("nameid").(string))
        // Check of pid has coordos
        if len(parents) > 0 && !db.GetDB().HasCoordos(ctx.Value("nameid").(string)) {
            // @debug: move to CheckCoordoPath function
            if err != nil { return nil, LogErr("Internal Error", err) }
            for _, p := range(parents) {
                // @debug: chec charac mode !
                if userIsCoordo(uctx, p) >= 0 {
                    ok = true
                    break
                }
            }
        }
    }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// HasRoot check the list of node to check if the user has root node in common.
func hasRoot(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Check that user has the given role on the asked node
    var ok bool
    for _, nodeField := range nodeFields {
        ok, err = checkUserRoot(ctx, uctx, nodeField, obj)
        if err != nil { return nil, LogErr("Access denied", err) }
        if ok { return next(ctx) }
    }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Only the onwer of the object can edit it.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get attributes and check everything is ok
    var userObj model.JsonAtom
    var f string
    if userField == nil {
        f = "user"
        userObj[f] = obj
    } else {
        f = *userField
        userObj = obj.(model.JsonAtom)
    }

    ok, err := checkUserOwnership(ctx, uctx, f, userObj)
    if err != nil { return nil, LogErr("Access denied", err) }
    if ok { return next(ctx) }

    return nil, LogErr("Access Denied", fmt.Errorf("bad ownership."))
}

func readOnly(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    return nil, LogErr("Forbiden", fmt.Errorf("Read only field on `%s'", fieldName))
}



//
// Private auth methods
//



// Check if the an user owns the given object
func checkUserOwnership(ctx context.Context, uctx model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    var username string
    var err error
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        // Database request
        id := ctx.Value("id").(string)
        if id == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        // Request the database to get the field
        username_, err := db.GetDB().GetSubFieldById(id, "Post."+userField, "User.username") // @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface: ToTypeName(reflect.TypeOf(nodeObj).String())
        if err != nil { return false, err }
        username = username_.(string)
    } else {
        username = user.(model.JsonAtom)["username"].(string)
    }

    // Check user ID match
    return uctx.Username == username, err
}

// check if the an user has the given role of the given (nested) node
func checkUserRole(ctx context.Context, uctx model.UserCtx, nodeField string, nodeObj interface{}, role model.RoleType) (bool, error) {
    // Check that nodes are present
    var nameid string
    var err error
    node := nodeObj.(model.JsonAtom)[nodeField]
    id_ := ctx.Value("id")
    nameid_ := ctx.Value("nameid")
    if id_ != nil {
        // Tension here
        // Request the database to get the field
        nameid_, err = db.GetDB().GetSubFieldById(id_.(string), "Tension."+nodeField, "Node.nameid")
        if err != nil { return false, err }
        nameid = nameid_.(string)
        if isRole(nameid) {
            nameid, _ = nid2pid(nameid)
        }
    } else if (nameid_ != nil) {
        // Node Here
        nameid_, err := db.GetDB().GetSubFieldByEq("Node.nameid", nameid_.(string), "Node."+nodeField, "Node.nameid")
        if err != nil { return false, err }
        if nameid_ == nil {
            // Assume root node
            return false, fmt.Errorf("Root node updates are not implemented yet...")
        }
        nameid = nameid_.(string)
    } else if (node != nil && node.(model.JsonAtom)["nameid"] != nil) {
        nameid = node.(model.JsonAtom)["nameid"].(string)
    } else {
        return false, fmt.Errorf("node target unknown, need a database request here...")
    }

    // Check if user is an owner
    rootnameid, _ := nid2rootid(nameid)
    if userIsOwner(uctx, rootnameid) >= 0 { return true, err }

    // Search for rights
    ok := userHasRole(uctx, role, nameid)
    return ok, err
}

// check if an user as at least one of his role whithin the given root.
func checkUserRoot(ctx context.Context, uctx model.UserCtx, nodeField string, nodeObj interface{}) (bool, error) {
    // Check that nodes are present
    var rootnameid string
    var err error
    nodeTarget_ := getNestedObj(nodeObj, nodeField)
    if nodeTarget_ == nil {
        // Database request
        id := ctx.Value("id").(string)
        if id == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        rootnameid_, err := db.GetDB().GetSubFieldById(id, "Tension."+nodeField, "Node.rootnameid")
        if err != nil { return false, err }
        if rootnameid_ != nil {
            rootnameid = rootnameid_.(string)
        }
    } else {
        // Extract node identifiers
        nameid_ := nodeTarget_.(model.JsonAtom)["nameid"]
        if nameid_ == nil {
            return false, fmt.Errorf("node target unknown (nameid), need a database request here...")
        }
        rootnameid, err = nid2rootid(nameid_.(string))
        if err != nil {
            panic(err.Error())
        }
    }

    // Search for rights
    ok := userHasRoot(uctx, rootnameid)
    return ok, err
}

// check if the an user has the given role of the given (nested) node
func checkAssignees(ctx context.Context, uctx model.UserCtx, nodeObj interface{}) (bool, error) {
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

func doAddNodeHook(uctx model.UserCtx, node model.AddNodeInput, parentid string, charac model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    isRoot := node.IsRoot
    nameid := node.Nameid // @TODO (nameid @codec): verify that nameid match parentid
    rootnameid := node.Rootnameid
    name := node.Name
    //parent_ := node.Parent

    err = auth.ValidateNameid(nameid, rootnameid)
    if err != nil { return ok, err }
    err = auth.ValidateName(name)
    if err != nil { return ok, err }

    //
    // Create new organisation Hook
    //
    if isRoot{
        //if uctx.Rights.CanCreateRoot {
        //    if parent_ != nil {
        //        err = fmt.Errorf("root node can't have a parent")
        //    } else if nameid != rootnameid {
        //        err = fmt.Errorf("root node nameid and rootnameid are different")
        //    } else {
        //        ok = true
        //    }
        //} else {
        //    err = fmt.Errorf("you are not authorized to create new organisation")
        //}

        err = fmt.Errorf("You are not authorized to create new organisation.")
        return ok, err
    }

    //
    // New member hook (Guest)
    //
    nodeType := node.Type
    roleType := node.RoleType
    if roleType != nil && *roleType == model.RoleTypeGuest {
        fmt.Println(node, charac)
        if !charac.UserCanJoin {
            err = fmt.Errorf("this organisation does not accept new members.")
        } else if rootnameid != parentid {
            err = fmt.Errorf("guest user can only join the root circle.")
        } else if nodeType != model.NodeTypeRole {
            // @DEBUG; this will be obsolete with union schema
            err = fmt.Errorf("circle with role_type defined should be of type RoleType.")
        } else {
            ok = true
        }

        return ok, err
    }

    return false, fmt.Errorf("not implemented addNode request")
}

// Take action base on the Event value:
// * get tension node target NodeCharac and either
//      * last blob if bid is null
//      * given blob otherwiser
// * if event == blobPushed
//      * check user hasa the right authorization based on NodeCharac
//      * update the tension action value AND the blob pushedFlag
//      * copy the Blob data in the target Node.source (Uses GQL requests)
// * elif event == TensionEventBlobArchived/Unarchived
//     * link or unlink role
//     * set archive evnet and flag
// * elif event == TensionEventUserLeft
//    * remove User role
//    * remove Orga role (Guest/Member) if role_type is Guest|Member
// Note: @Debug: Only one BlobPushed will be processed
// Note: @Debug: remove added tension on error ?
func tensionBlobHook(uctx model.UserCtx, tid string, events []*model.EventRef, bid *string) (bool, error) {
    var ok bool = true
    var err error
    var nameid string
    for _, event := range(events) {
        if *event.EventType == model.TensionEventBlobPushed ||
           *event.EventType == model.TensionEventBlobArchived ||
           *event.EventType == model.TensionEventBlobUnarchived ||
           *event.EventType == model.TensionEventUserLeft {
               // Process the special event
               ok, err, nameid = processTensionEventHook(uctx, event, tid, bid)
               if ok && err == nil {
                   // Set the Update time into the target node
                   err = db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.updatedAt", Now())
                   pid, _ := nid2pid(nameid) // @debug: real parent needed here (ie event for circle)
                   if pid != nameid && err == nil {
                       err = db.GetDB().SetFieldByEq("Node.nameid", pid, "Node.updatedAt", Now())
                   }
               }
               // Break after the first hooked event
               break
           }
    }

    return ok, err
}

// Add, Update or Archived a Node
func processTensionEventHook(uctx model.UserCtx, event *model.EventRef, tid string, bid *string) (bool, error, string) {
    var nameid string
    // Get Tension, target Node and blob charac (last if bid undefined)
    tension, err := db.GetDB().GetTensionHook(tid, bid)
    if err != nil { return false, LogErr("Access denied", err), nameid}
    if tension == nil { return false, LogErr("Access denied", fmt.Errorf("tension not found.")), nameid }

    // Check that Blob exists
    blob := tension.Blobs[0]
    if blob == nil { return false, LogErr("internal error", fmt.Errorf("blob not found.")), nameid }
    bid = &blob.ID

    // Extract Tension characteristic
    tensionCharac, err:= TensionCharac{}.New(*tension.Action)
    if err != nil { return false, LogErr("internal error", err), nameid }

    var ok bool
    var node *model.NodeFragment = blob.Node

    // Nameid Codec
    if node != nil && node.Nameid != nil {
        _, nameid, err = nodeIdCodec(tension.Receiver.Nameid, *node.Nameid, *node.Type)
    }

    if *event.EventType == model.TensionEventBlobPushed {
        // Add or Update Node
        // --
        // 1. switch on TensionCharac.DocType (not blob type) -> rule differ from doc type!
        // 2. swith on TensionCharac.ActionType to add update etc...
        switch tensionCharac.ActionType {
        case NewAction:
            // First time a blob is pushed.
            switch tensionCharac.DocType {
            case NodeDoc:
                ok, err = TryAddNode(uctx, tension, node, bid)
            case MdDoc:
                md := blob.Md
                ok, err = TryAddDoc(uctx, tension, md)
            }
        case EditAction:
            switch tensionCharac.DocType {
            case NodeDoc:
                ok, err = TryUpdateNode(uctx, tension, node, bid)
            case MdDoc:
                md := blob.Md
                ok, err = TryUpdateDoc(uctx, tension, md)
            }
        case ArchiveAction:
            err = LogErr("Access denied", fmt.Errorf("Cannot publish archived document."))
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob pushed flag
            err = db.GetDB().SetPushedFlagBlob(*bid, Now(), tid, tensionCharac.EditAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventBlobArchived {
        // Archived Node
        // --
        switch tensionCharac.DocType {
        case NodeDoc:
            ok, err = TryArchiveNode(uctx, tension, node)
        case MdDoc:
            md := blob.Md
            ok, err = TryArchiveDoc(uctx, tension, md)
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob archived flag
            err = db.GetDB().SetArchivedFlagBlob(*bid, Now(), tid, tensionCharac.ArchiveAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventBlobUnarchived {
        // Unarchived Node
        // --
        switch tensionCharac.DocType {
        case NodeDoc:
            ok, err = TryUnarchiveNode(uctx, tension, node)
        case MdDoc:
            md := blob.Md
            ok, err = TryUnarchiveDoc(uctx, tension, md)
        }

        if err != nil { return ok, err, nameid }
        if ok { // Update blob pushed flag
            err = db.GetDB().SetPushedFlagBlob(*bid, Now(), tid, tensionCharac.EditAction(node.Type))
        }
    } else if *event.EventType == model.TensionEventUserLeft {
        // Remove user reference
        // --
        if model.RoleType(*event.Old) == model.RoleTypeGuest {
            rootid, err := nid2rootid(*event.New)
            if err != nil { return ok, err, nameid }
            i := userIsGuest(uctx, rootid)
            if i<0 {return ok, LogErr("Value error", fmt.Errorf("You are not a guest in this organisation.")), nameid}
            var nf model.NodeFragment
            var t model.NodeType = model.NodeTypeRole
            StructMap(uctx.Roles[i], &nf)
            nf.FirstLink = &uctx.Username
            nf.Type = &t
            node = &nf
        }

        ok, err = LeaveRole(uctx, tension, node)
    }

    return ok, err, nameid
}

//
// User Rights Seeker
//

// useHasRoot return true if the user has at least one role in above given node
func userHasRoot(uctx model.UserCtx, rootnameid string) bool {
    uctx, e := auth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil {
        panic(e)
    }

    for _, ur := range uctx.Roles {
        if ur.Rootnameid == rootnameid  {
            return true
        }
    }
    return false
}

// userHasRole return true if the user has the given role on the given node
func userHasRole(uctx model.UserCtx, role model.RoleType, nameid string) bool {
    uctx, e := auth.CheckUserCtxIat(uctx, nameid)
    if e != nil {
        panic(e)
    }

    for _, ur := range uctx.Roles {
        pid, err := nid2pid(ur.Nameid)
        if err != nil {
            panic(err.Error())
        }
        if pid == nameid && ur.RoleType == role {
            return true
        }
    }
    return false
}

// usePlayRole return true if the user play the given role (Nameid)
func userPlayRole(uctx model.UserCtx, nameid string) bool {
    uctx, e := auth.CheckUserCtxIat(uctx, nameid)
    if e != nil {
        panic(e)
    }

    for _, ur := range uctx.Roles {
        if ur.Nameid == nameid  {
            return true
        }
    }
    return false
}

// useIsCoordo return true if the user has at least one role in the given node
func userIsMember(uctx model.UserCtx, nameid string) int {
    uctx, e := auth.CheckUserCtxIat(uctx, nameid)
    if e != nil {
        panic(e)
    }

    for i, ur := range uctx.Roles {
        pid, err := nid2pid(ur.Nameid)
        if err != nil {
            panic(err.Error())
        }
        if pid == nameid {
            return i
        }
    }
    return -1
}

// useIsCoordo return true if the user has at least one role of Coordinator in the given node
func userIsCoordo(uctx model.UserCtx, nameid string) int {
    uctx, e := auth.CheckUserCtxIat(uctx, nameid)
    if e != nil {
        panic(e)
    }

    for i, ur := range uctx.Roles {
        pid, err := nid2pid(ur.Nameid)
        if err != nil {
            panic("bad nameid format for coordo test: "+ ur.Nameid)
        }
        if pid == nameid && ur.RoleType == model.RoleTypeCoordinator {
            return i
        }
    }

    return -1
}

// userIsGuest return true if the user is a guest (has only one role) in the given organisation
func userIsGuest(uctx model.UserCtx, rootnameid string) int {
    uctx, e := auth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil {
        panic(e)
    }

    for i, r := range uctx.Roles {
        if r.Rootnameid == rootnameid && r.RoleType == model.RoleTypeGuest {
            return i
        }
    }

    return -1
}

func userIsOwner(uctx model.UserCtx, rootnameid string) int {
    uctx, e := auth.CheckUserCtxIat(uctx, rootnameid)
    if e != nil {
        panic(e)
    }

    for i, r := range uctx.Roles {
        if r.Rootnameid == rootnameid && r.RoleType == model.RoleTypeOwner {
            return i
        }
    }

    return -1
}

//
// Go Utils
//
func getNestedObj(obj interface {}, field string) interface{} {
    var source model.JsonAtom
    var target interface{}

    source =  obj.(model.JsonAtom)
    fields := strings.Split(field, ".")

    for _, f := range fields {
        target = source[f]
        if target == nil { return nil }
        source = target.(model.JsonAtom)
    }

    return target
}

func get(obj model.JsonAtom, field string, deflt interface{}) interface{} {
    v := obj[field]
    if v == nil {
        return deflt
    }

    return v
}
