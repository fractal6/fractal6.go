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

    // User defined directives
    c.Directives.Hidden = hidden
    c.Directives.Count = count

    // for add inputs...
    c.Directives.Add_isOwner = isOwner
    // for update or remove inputs...
    c.Directives.Patch_isOwner = isOwner
    c.Directives.Patch_RO = readOnly
    // for add, update and remove inputs...
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_assertType = assertType
    c.Directives.Alter_hasRole = hasRole 
    c.Directives.Alter_hasRoot = hasRoot
    // For mutation hook
    c.Directives.Add_addNodeHook = addNodeHook

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

func inputMaxLength(ctx context.Context, obj interface{}, next graphql.Resolver, field string, max int) (interface{}, error) {
    v := obj.(model.JsonAtom)[field].(string)
    if len(v) > max {
        return nil, fmt.Errorf("`%s' to long. Maximum length is %d", field, max)
    }
    return next(ctx)
}

func assertType(ctx context.Context, obj interface{}, next graphql.Resolver, field string, type_ model.NodeType) (interface{}, error) {
    v := obj.(model.JsonAtom)[field].(model.JsonAtom)
    return nil, fmt.Errorf("Only Node type is supported, this is not: %T", v )
}

// Special HOOK for:
// * addNode: New Orga (Root creation) -> check UserRight.CanCreateRoot
// * addNode/updateNode?: Join orga (push Node) -> check if NodeCharac.UserCanJoin is True and if user is not already a member
// * addNode/updateNode: Add role and subcircle
// Check user right for special query
func addNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRole/userCtx", "Access denied", err)  // Login or signup
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
    if queryName == "addNode" { 

        // Get the Node Characteristics of the **Parent Node**
        if node.Parent == nil && (*node.Parent).Nameid == nil {
            return nil, tools.LogErr("@addNodeHook", "Access denied", fmt.Errorf("Parent node not found"))
        }
        parentid := *(*node.Parent).Nameid
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

    e := fmt.Errorf("Operation not allowed. Please, contact a coordinator to access this ressource.")
    return nil, tools.LogErr("@addNodeHook", "Access denied", e)
}

// HasRole check the user has the authorisatiion to access a ressource by checking if it satisfies at least one of 
// 1. user rights
// 2. user ownership (u field)
// 3. check user role, (n r field)
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string, role model.RoleType, userField *string) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRole/userCtx", "Access denied", err)  // Login or signup
    }

    rc := graphql.GetResolverContext(ctx)
    queryName := rc.Field.Name
    if queryName == "addNode" {
        // Manage this authorization in the addNodeHook
        return next(ctx)
    } // else => Update or Remove queries

    // If userField is given check if the current user
    // is the owner of the ressource
    var ok bool
    if userField != nil {
        ok, err = checkUserOwnership(uctx, *userField, obj)
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkOwn", "Access denied", err)
        }
        if ok {
            return next(ctx)
        }
    }
    
    // Check that user has the given role on the asked node
    for _, nodeField := range nodeFields {
        ok, err = checkUserRole(uctx, nodeField, obj, role)
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkRole", "Access denied", err)
        }
        if ok {
            return next(ctx)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    // @DEBUG: get rootnameid from nameid
    e := fmt.Errorf("Please, join this organisation or contact a coordinator to access this ressource.")
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
        ok, err = checkUserRoot(uctx, nodeField, obj)
        if err != nil {
            return nil, tools.LogErr("@hasRoot", "Access denied", err)
        }
        if ok {
            return next(ctx)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    e := fmt.Errorf("Please, join this organisation or contact a coordinator to access this ressource.")
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

    ok, err := checkUserOwnership(uctx, f, userObj)
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

func checkUserOwnership(uctx model.UserCtx, userField string, userObj interface{}) (bool, error) {
    // Get user ID
    user := userObj.(model.JsonAtom)[userField]
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        println("User unknown, need a database request here...")
        return false, nil
    } 

    // Check user ID match
    userid := user.(model.JsonAtom)["username"].(string)
    if uctx.Username == userid {
        return true, nil
    }
    return false, nil
}

func checkUserRole(uctx model.UserCtx, nodeField string, nodeObj interface{}, role model.RoleType) (bool, error) {
    // Check that nodes are present
    nodeTarget_ := getNestedObj(nodeObj, nodeField)
    if nodeTarget_ == nil {
        return false, fmt.Errorf("Node target unknown(%s), need a database request here...", nodeField)
    }

    // Extract node identifier
    nameid_ := nodeTarget_.(model.JsonAtom)["nameid"]
    if nameid_ == nil {
        return false, fmt.Errorf("Node target unknown(nameid), need a database request here...")
    }

    // Search for rights
    ok := userHasRole(uctx, role, nameid_.(string))
    return ok, nil
}

func checkUserRoot(uctx model.UserCtx, nodeField string, nodeObj interface{}) (bool, error) {
    // Check that nodes are present
    nodeTarget_ := getNestedObj(nodeObj, nodeField)
    if nodeTarget_ == nil {
        return false, fmt.Errorf("Node target unknown(%s), need a database request here...", nodeField)
    }

    // Extract node identifiers
    rootnameid_ := nodeTarget_.(model.JsonAtom)["rootnameid"]
    if rootnameid_ == nil {
        return false, fmt.Errorf("node target unknown (rootnameid), need a database request here !!!")
    }

    // Search for rights
    ok := userHasRoot(uctx, rootnameid_.(string))
    return ok, nil
}

func doAddNodeHook(uctx model.UserCtx, node model.AddNodeInput, parentid string, charac model.NodeCharac) (bool, error) {
    var ok bool = false
    var err error

    isRoot := node.IsRoot
    nameid := node.Nameid
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
                err = fmt.Errorf("Root node can't have a parent.")
            } else if nameid != rootnameid {
                err = fmt.Errorf("Root node nameid and rootnameid are different.")
            } else {
                ok = true
            }
        } else {
            err = fmt.Errorf("You are not authorized to create new organisation.")
        }
        return ok, err
    } 

    //
    // New member hook
    // 
    nodeType := node.Type
    roleType := node.RoleType
    if roleType != nil && *roleType == model.RoleTypeGuest {
        if !charac.UserCanJoin {
            err = fmt.Errorf("This organisation does not accept new members.")
        } else if rootnameid != parentid {
            err = fmt.Errorf("Guest user can only join the root circle.")
        } else if nodeType != model.NodeTypeRole {
            // @DEBUG; this will be obsolete with union schema
            err = fmt.Errorf("Circle with role_type defined should be of type RoleType.")
        } else {
            ok = true
        }
        return ok, err
    }

    //
    // New sub-circle hook
    // 
    if nodeType == model.NodeTypeCircle {
        // @TODO (nameid @codec): verify that nameid match parentid
        if charac.Mode == model.NodeModeChaos {
            ok = userIsMember(uctx, parentid)
        } else if charac.Mode == model.NodeModeCoordinated {
            ok = userIsCoordo(uctx, parentid)
        }

        return ok, err
    }

    return false, fmt.Errorf("Not implemented addNode request.")
}

//
// User Rights Seeker
//


// useHasRoot return true if the user has at least one rool in above given node
func userHasRoot(uctx model.UserCtx, nameid string) bool {
    for _, ur := range uctx.Roles {
        if ur.Rootnameid == nameid  {
            return true
        }
    }
    return false
}

// useHasRole return true if the user has the given role on the given node
func userHasRole(uctx model.UserCtx, role model.RoleType, nameid string) bool {
    for _, ur := range uctx.Roles {
        if ur.Nameid == nameid && ur.RoleType == role {
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
            panic("Bad nameid format for coordo test: "+ ur.Nameid)
        }
        if pid == nameid && ur.RoleType == model.RoleTypeCoordinator {
            return true
        }
    }
    return false
}

//
// User Codecs
//
func nid2pid(nid string) (string, error) {
    var pid string
    parts := strings.Split(nid, "#")
    if !(len(parts) == 3 || len(parts) == 1 || len(parts) == 2) {
        return pid, fmt.Errorf("Bad nameid format for nid2pid: " + nid)
    }

    if len(parts) == 1 || parts[1] == "" {
        pid = parts[0]
    } else {
        pid = strings.Join(parts[:len(parts)-1],  "#")
    }
    return pid, nil
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
