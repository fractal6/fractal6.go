//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    //"time"
    "context"
    "reflect"
    "strings"
    "github.com/99designs/gqlgen/graphql"

    gen "zerogov/fractal6.go/graph/generated"
    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/tools"
    "zerogov/fractal6.go/db"
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

    /* Query */
    // Objects directives
    c.Directives.HidePrivate = hidePrivate

    // Fields directives
    c.Directives.Hidden = hidden
    c.Directives.Count = count
    c.Directives.Meta_getNodeStats = getNodeStats

    /* Mutation */

    // Add inputs directives
    c.Directives.Add_isOwner = isOwner
    // Update or Remove inputs directives
    c.Directives.Patch_isOwner = isOwner
    c.Directives.Patch_RO = readOnly
    c.Directives.Patch_hasRole = hasRole 
    // Add, Update and Remove inputs directives
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_assertType = assertType
    c.Directives.Alter_hasRole = hasRole 
    c.Directives.Alter_hasRoot = hasRoot
    // Mutation Hook directives
    c.Directives.Hook_addNode = addNodeHook
    c.Directives.Hook_updateNode = updateNodeHook
    c.Directives.Hook_updateTension = updatePostHook
    c.Directives.Hook_updateComment = updatePostHook

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
    //if obj != nil {
    //    switch v := obj.(type) {
    //    case *model.Node:
    //        nameid := v.Nameid
    //        if nameid == "" {
    //            return nil, tools.LogErr("@hidePrivate/node", "Access denied", fmt.Errorf("nameid not provided"))
    //        } else {
    //            fmt.Println("Get da node a private")
    //        }
    //    case *model.Tension:
    //            // pass for now
    //            // isPrivate should be inherited by the tension, and user right check if orivate
    //    default:
    //        fmt.Printf("%T\n", v)
    //        panic("type unknonw for @hidePrivate directive.")
    //    }
    //    return next(ctx)
    //} 
    node_, err := next(ctx)
    if obj == nil {
        switch v := node_.(type) {
        case *model.Node:
            if v == nil {
                break
            }
            yes, err := isHidePrivate(ctx, v.Nameid)
            if err != nil { return nil, err }
            if yes { return nil, tools.LogErr("@hidePrivate/getNode", "Access denied", fmt.Errorf("private node")) }
        case []*model.Node:
            for _, node := range(v) {
                yes, err := isHidePrivate(ctx, node.Nameid)
                if err != nil { return nil, err }
                if yes { return nil,  tools.LogErr("@hidePrivate/queryNode", "Access denied", fmt.Errorf("private node")) }
            }
        default:
            panic("@isPrivate: node type unknonwn.")
        }
    }
    return node_, err
}

func isHidePrivate(ctx context.Context, nameid string) (bool, error) {
    var yes bool = true
    var err error
    if nameid == "" {
        err = tools.LogErr("@hidePrivate", "Access denied", fmt.Errorf("nameid not provided"))
    } else {
        // Get the public status of the node
        isPrivate, err :=  db.GetDB().GetFieldByEq("Node", "nameid", nameid, "isPrivate")
        if err != nil {
            return yes, tools.LogErr("@hidePrivate", "Access denied", err)
        }
        if isPrivate.(bool) {
            // check user role.
            uctx, err := auth.UserCtxFromContext(ctx)
            if err != nil {
                return yes, tools.LogErr("@hidePrivate/userCtx", "Access denied", err)  // Login or signup
            }
            rootnameid, err := nid2pid(nameid)
            for _, ur := range uctx.Roles {
                if ur.Rootnameid == rootnameid  {
                    return false, err
                }
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
    goFieldfDef := tools.ToGoNameFormat(fieldName)

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    id := reflect.ValueOf(obj).Elem().FieldByName("ID").String()
	if id == "" {
        err := fmt.Errorf("`id' field is needed to query `%s'", fieldName)
        return nil, err
    }
    typeName := tools.ToTypeName(reflect.TypeOf(obj).String())
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
    //    goFieldfDef := tools.ToGoNameFormat(k)
    //    reflect.ValueOf(obj).Elem().FieldByName(goFieldfDef).Set(reflect.ValueOf(&stats))
    //}
    return next(ctx)
}

//
// Mutation
//

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
// * addNode/updateNode?: Join orga (push Node) -> check if NodeCharac.UserCanJoin is True and if user is not already a member
// * addNode/updateNode: Add role and subcircle
// Check user right for special query
func addNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@addNodeHook/userCtx", "Access denied", err)  // Login or signup
    }

    input_, err := next(ctx)
    if err != nil {
        return nil, tools.LogErr("@addNodeHook", "internal error", err)
    }
    input := input_.([]*model.AddNodeInput)
    if len(input) != 1 {
        return nil, tools.LogErr("@addNodeHook", "Add node error", fmt.Errorf("Only one node supported in input"))
    }
    node := *input[0]

    rc := graphql.GetResolverContext(ctx)
    queryName := rc.Field.Name
    if queryName == "addNode" {  // @obsolete with ARGUMENT_DEFINITION directive ?

        // Get the Node Characteristics of the **Parent Node**
        if node.Parent == nil && (*node.Parent).Nameid == nil {
            return nil, tools.LogErr("@addNodeHook", "Access denied", fmt.Errorf("Parent node not found"))
        }
        parentid := *(node.Parent).Nameid
        charac_, err := db.GetDB().GetNodeCharac("nameid", parentid)
        //nodeCHarac := getLut(nameid, "getNodeCharac")
        if err != nil {
            return nil, tools.LogErr("@addNodeHook/NodeCharac", "Access denied", err)
        } else if charac_ == nil {
            return nil, tools.LogErr("@addNodeHook/NodeCharac", "Access denied", fmt.Errorf("Node characteristic not found"))
        }
        charac := *charac_

        ok, err := doAddNodeHook(uctx, node, parentid, charac)
        if err != nil {
            return nil, tools.LogErr("@addNodeHook/doAddNodeHook", "Access denied", err)
        }
        if ok {
            //// Add the default node
            //userCanJoin := false
            //typeCo := model.NodeTypeRole
            //roleTypeCo := model.RoleTypeCoordinator
            //nameCo := "Coordinator"
            //nameidCo := node.Nameid + "#" + "coordo1"
            //nowCo := time.Now().Format(time.RFC3339)
            //isRootCo := false

            //co := model.NodeRef{
            //    CreatedAt:  & nowCo,
            //    CreatedBy:  & model.UserRef{Username: & uctx.Username},
            //    IsRoot:     & isRootCo,
            //    Type:       & typeCo,
            //    RoleType:   & roleTypeCo,
            //    Name:       & nameCo,
            //    Nameid:     & nameidCo,
            //    Rootnameid: & node.Rootnameid,
            //    Charac: &model.NodeCharacRef{
            //        UserCanJoin: &userCanJoin,
            //        Mode: &charac.Mode, // Inherit mode
            //    },
            //    //Mandate: ... // @Debug: todo default mandate for basic Role
            //    FirstLink:  & model.UserRef{Username: & uctx.Username},
            //}
            //node.Children = []*model.NodeRef{&co}
            //newInput := []*model.AddNodeInput{&node}
            //return newInput, nil
            return input_, nil
        }
    }

    e := fmt.Errorf("contact a coordinator to access this ressource.")
    return nil, tools.LogErr("@addNodeHook", "Access denied", e)
}

// Update Node hook
// * add the nameid field in the context for further inspection in new resolver
func updateNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    nameid_ := filter["nameid"].(model.JsonAtom)["eq"]
    if nameid_ != nil {
        //return nil, tools.LogErr("@updateNodeHook", "not implemented", fmt.Errorf("node identifier unknonw"))
        ctx = context.WithValue(ctx, "nameid", nameid_.(string))
        input_, err := next(ctx)
        return input_, err
    }

    return next(ctx)
}

// Update Post hook (Tensiont, Comment, etc)
// * add the id field in the context for further inspection in new resolver
func updatePostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) > 1 {
        return nil, tools.LogErr("@updatePostHook", "not implemented", fmt.Errorf("multiple post not supported"))
    }

    ctx = context.WithValue(ctx, "id", ids[0].(string))
    input_, err := next(ctx)
    return input_, err
}

// HasRole check the user has the authorisation to update a ressource by checking if it satisfies at least one of 
// 1. user rights
// 2. user ownership (u field)
// 3. check user role, (n r field)
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string, role model.RoleType, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRole/userCtx", "Access denied", err)  // Login or signup
    }

    // If userField is given check if the current user
    // is the owner of the ressource
    var ok bool
    if userField != nil {
        ok, err = checkUserOwnership(ctx, uctx, *userField, obj)
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkOwn", "Access denied", err) 
        }
        if ok {
            return next(ctx) 
        }
    }
    
    // Check that user has the given role on the asked node
    for _, nodeField := range nodeFields {
        ok, err = checkUserRole(ctx, uctx, nodeField, obj, role)
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkRole", "Access denied", err)
        }
        if ok {
            return next(ctx)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    // @DEBUG: get rootnameid from nameid
    e := fmt.Errorf("join this organisation or contact a coordinator to access this ressource")
    return nil, tools.LogErr("@hasRole", "Access denied", e)
}

// HasRoot check the list of node to check if the user has root node in common.
func hasRoot(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRoot/userCtx", "Access denied", err)  // Login or signup 
    }

    // Check that user has the given role on the asked node
    var ok bool
    for _, nodeField := range nodeFields {
        ok, err = checkUserRoot(ctx, uctx, nodeField, obj)
        if err != nil {
            return nil, tools.LogErr("@hasRoot", "Access denied", err) 
        }
        if ok {
            return next(ctx) 
        }
    }

    // Format output to get to be able to format a links in a frontend.
    e := fmt.Errorf("join this organisation or contact a coordinator to access this ressource")
    return nil, tools.LogErr("@hasRoot", "Access denied", e)
}

// Only the onwer of the ibject can edut it.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@isOwner/userCtx", "Access denied", err)
    }

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
    if err != nil {
        return nil, tools.LogErr("@isOwner/auth", "Access denied", err)
    }
    if ok {
        return next(ctx)
    }

    return nil, tools.LogErr("@isOwner", "Access Denied", fmt.Errorf("bad ownership"))
}

func readOnly(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldName := rc.Field.Name
    return nil, tools.LogErr("@ro", "Forbiden", fmt.Errorf("Read only field on `%s'", fieldName))
}

//
// Private auth methods
//

// Check if the an user owns the given object
func checkUserOwnership(ctx context.Context, uctx model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    var username string
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        // Database request
        id := ctx.Value("id").(string)
        if id == "" {
            return false, fmt.Errorf("node target unknown(id), need a database request here...")
        }
        // Request the database to get the field
        // @DEBUG (#xxx): how to get the type of the object to update for a more generic function hre ?
        typeName := "Post" // @DEBUG: in the dgraph graphql schema, @createdBy is in the Post interface ....
        username_, err := db.GetDB().GetSubFieldById(id, typeName, userField, "User", "username")
        if err != nil {
            return false, err
        }
        username = username_.(string)
    } else {
        username = user.(model.JsonAtom)["username"].(string)
    }

    // Check user ID match
    return uctx.Username == username, nil
}

// check if the an user has the given role of the given (nested) node
func checkUserRole(ctx context.Context, uctx model.UserCtx, nodeField string, nodeObj interface{}, role model.RoleType) (bool, error) {
    // Check that nodes are present
    var nameid string
    node := nodeObj.(model.JsonAtom)[nodeField]
    id_ := ctx.Value("id")
    nameid_ := ctx.Value("nameid")
    if  id_ != nil {
        // Tension here
        // Request the database to get the field
        // @DEBUG (#xxx): how to get the type of the object to update for a more generic function here ?
        typeName := "Tension"
        nameid_, err := db.GetDB().GetSubFieldById(id_.(string), typeName, nodeField, "Node", "nameid")
        if err != nil {
            return false, err
        }
        nameid = nameid_.(string)
        if isRole(nameid) {
            nameid, _ = nid2pid(nameid)
        }
    } else if (nameid_ != nil) {
        // Node Here
        typeName := "Node"
        nameid_, err := db.GetDB().GetSubFieldByEq("nameid", nameid_.(string), typeName, nodeField, "Node", "nameid")
        if err != nil {
            return false, err
        }
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

    // Search for rights
    ok := userHasRole(uctx, role, nameid)
    return ok, nil
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
        // @DEBUG (#xxx): how to get the type of the object to update for a more generic function hre ?
        //typeName := tools.ToTypeName(reflect.TypeOf(nodeObj).String())
        typeName := "Tension"
        rootnameid_, err := db.GetDB().GetSubFieldById(id, typeName, nodeField, "Node", "rootnameid")
        if err != nil {
            return false, err
        }
        if rootnameid_ != nil {
            rootnameid = rootnameid_.(string)
        }
    } else {
        // Extract node identifiers
        nameid_ := nodeTarget_.(model.JsonAtom)["nameid"]
        if nameid_ == nil {
            return false, fmt.Errorf("node target unknown (nameid), need a database request here ...")
        }
        rootnameid, err = nid2rootid(nameid_.(string))
        if err != nil {
            panic(err.Error())
        }
    }

    // Search for rights
    ok := userHasRoot(uctx, rootnameid)
    return ok, nil
}

func doAddNodeHook(uctx model.UserCtx, node model.AddNodeInput, parentid string, charac model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    isRoot := node.IsRoot
    nameid := node.Nameid // @TODO (nameid @codec): verify that nameid match parentid
    rootnameid := node.Rootnameid
    name := node.Name
    parent_ := node.Parent

    err = auth.ValidateNameid(nameid, rootnameid, name)
    if err != nil {
        return false, err
    }

    //
    // Create new organisation Hook
    // 
    if isRoot{
        if uctx.Rights.CanCreateRoot {
            if parent_ != nil {
                err = fmt.Errorf("root node can't have a parent")
            } else if nameid != rootnameid {
                err = fmt.Errorf("root node nameid and rootnameid are different")
            } else {
                ok = true
            }
        } else {
            err = fmt.Errorf("you are not authorized to create new organisation")
        }

        return ok, err
    } 

    //
    // New member hook (Guest)
    // 
    nodeType := node.Type
    roleType := node.RoleType
    if roleType != nil && *roleType == model.RoleTypeGuest {
        if !charac.UserCanJoin {
            err = fmt.Errorf("this organisation does not accept new members")
        } else if rootnameid != parentid {
            err = fmt.Errorf("guest user can only join the root circle")
        } else if nodeType != model.NodeTypeRole {
            // @DEBUG; this will be obsolete with union schema
            err = fmt.Errorf("circle with role_type defined should be of type RoleType")
        } else {
            ok = true
        }

        return ok, err
    }

    //
    // New Role hook
    // 
    if nodeType == model.NodeTypeRole {
        if roleType == nil {
            err = fmt.Errorf("role should have a RoleType")
        }

        // Add node Policies
        if charac.Mode == model.NodeModeChaos {
            ok = userIsMember(uctx, parentid)
        } else if charac.Mode == model.NodeModeCoordinated {
            ok = userIsCoordo(uctx, parentid)
        }

        // Change Guest to member if user got its first role
        if ok && node.FirstLink != nil {
            err = maybeUpdateGuest2Peer(uctx, rootnameid, *node.FirstLink)
        }

        return ok, err
    }

    //
    // New sub-circle hook
    // 
    if nodeType == model.NodeTypeCircle {
        // Add node Policies
        if charac.Mode == model.NodeModeChaos {
            ok = userIsMember(uctx, parentid)
        } else if charac.Mode == model.NodeModeCoordinated {
            ok = userIsCoordo(uctx, parentid)
        }

        for _, child := range node.Children {
            if ok && child.FirstLink != nil {
                err = maybeUpdateGuest2Peer(uctx, rootnameid, *child.FirstLink)
            }
        }

        return ok, err
    }

    return false, fmt.Errorf("not implemented addNode request")
}

//
// User Rights Seeker
//

// useHasRoot return true if the user has at least one role in above given node
func userHasRoot(uctx model.UserCtx, rootnameid string) bool {
    for _, ur := range uctx.Roles {
        if ur.Rootnameid == rootnameid  {
            return true
        }
    }
    return false
}

// useHasRole return true if the user has the given role on the given node
func userHasRole(uctx model.UserCtx, role model.RoleType, nameid string) bool {
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

// useIsCoordo return true if the user has at least one role in the given node
func userIsMember(uctx model.UserCtx, nameid string) bool {
    for _, ur := range uctx.Roles {
        pid, err := nid2pid(ur.Nameid)
        if err != nil {
            panic(err.Error())
        }
        if pid == nameid {
            return true
        }
    }
    return false
}

// useIsCoordo return true if the user has at least one role of Coordinator in the given node
func userIsCoordo(uctx model.UserCtx, nameid string) bool {
    for _, ur := range uctx.Roles {
        pid, err := nid2pid(ur.Nameid)
        if err != nil {
            panic("bad nameid format for coordo test: "+ ur.Nameid)
        }
        if pid == nameid && ur.RoleType == model.RoleTypeCoordinator {
            return true
        }
    }

    return false
}

// userIsGuest return true if the user is a guest (has only one role) in the given organisation
func userIsGuest(uctx model.UserCtx, rootnameid string) int {
    for i, r := range uctx.Roles {
        if r.Rootnameid == rootnameid && r.RoleType == model.RoleTypeGuest {
            return i
        }
    }

    return -1
}

// maybeUpdateGuest2Peer check if Guest should be upgrade to Member role type
func maybeUpdateGuest2Peer(uctx model.UserCtx, rootnameid string, firstLink model.UserRef) error {
    if uctx.Username == *(firstLink).Username {
        i := userIsGuest(uctx, rootnameid)
        if i >= 0 {
            // Update RoleType to Member
            err := db.GetDB().UpgradeGuest(uctx.Roles[i].Nameid, model.RoleTypeMember)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

//
// User Codecs
//

// Get the parent nameid from the given nameid
func nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else {
        pid = strings.Join(parts[:len(parts)-1],  "#")
    }
    return pid, nil
}

// Get the rootnameid from the given nameid
func nid2rootid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("bad nameid format for nid2pid: " + nid)
    }

    return parts[0], nil
}

func isCircle(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 1 || len(parts) == 2
}
func isRole(nid string) (bool) {
    parts := strings.Split(nid, "#")
    return len(parts) == 3
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
        if target == nil {
            return nil
        }
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
