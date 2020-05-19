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
    c.Directives.Alter_maxLength = inputMaxLength
    c.Directives.Alter_assertType = assertType
    c.Directives.Alter_hasRole = hasRole 
    c.Directives.Alter_hasRoot = hasRoot
    c.Directives.Add_isOwner = isOwner
    c.Directives.Patch_isOwner = isOwner
    c.Directives.Patch_RO = readOnly
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
    db := db.GetDB()
    v := db.Count(id, typeName, field)
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

// HasRole search for authorized user by checking if user satisfy at least one of 
// 1. user rights
// 2. user ownership (u field)
// 3. check user role, (n r field)
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string, role model.RoleType, userField *string) (interface{}, error) {
    // Retrive userCtx from token
    userCtx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRole/userCtx", "Access denied", err)  // Login or signup
    }

    // Special HOOK for:
    // * addNode: New Orga (Root creation) -> check UserRight.CanCreateRoot
    // * addNode/updateNode?: Join orga (push Node) -> check if NodeCharac.UserCanJoin is True and if user is not already a member
    // * addNode/updateNode: Add role and subcircle
    // Check user right for special query
    rc := graphql.GetResolverContext(ctx)
    queryName := rc.Field.Name 
    if queryName == "addNode" {
        ok, err := addNodeHook(userCtx, obj)
        if ok {
            return next(ctx)
        }
        if err != nil {
            return nil, tools.LogErr("@hasRole/addNode", "Access denied", err)
        }
    }

    // If userField is given check if the current user
    // is the owner of the ressource
    var ok bool
    if userField != nil {
        ok, err = checkUserOwnership(userCtx, *userField, obj)
        if ok {
            return next(ctx)
        }
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkOwn", "Access denied", err)
        }
    }
    
    // Check that user has the given role on the asked node
    for _, nodeField := range nodeFields {
        ok, err = checkUserRole(userCtx, nodeField, obj, role)
        if ok {
            return next(ctx)
        }
        if err != nil {
            return nil, tools.LogErr("@hasRole/checkRole", "Access denied", err)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    // @DEBUG: get rootnameid from nameid
    e := fmt.Errorf("Please, join this organisation or contact a coordinator to access this ressource.")
    return nil, tools.LogErr("@hasRole", "Access denied", e)
}

// HasRoot check the list of node to check if the user has root node in common.
func hasRoot(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string) (interface{}, error) {
    // Retrive userCtx from token
    userCtx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRoot/userCtx", "Access denied", err)  // Login or signup
    }

    // Check that user has the given role on the asked node
    var ok bool
    for _, nodeField := range nodeFields {
        ok, err = checkUserRoot(userCtx, nodeField, obj)
        if ok {
            return next(ctx)
        }
        if err != nil {
            return nil, tools.LogErr("@hasRoot", "Access denied", err)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    e := fmt.Errorf("Please, join this organisation or contact a coordinator to access this ressource.")
    return nil, tools.LogErr("@hasRoot", "Access denied", e)
}

// Only the onwer of the ibject can edut it.
func isOwner(ctx context.Context, obj interface{}, next graphql.Resolver, userField *string) (interface{}, error) {
    //rc := graphql.GetResolverContext(ctx)
    //fieldName := rc.Field.Name // for input is like "updateUser"
    
    // Retrive userCtx from token
    userCtx, err := auth.UserCtxFromContext(ctx)
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

    ok, err := checkUserOwnership(userCtx, f, userObj)
    if ok {
        return next(ctx)
    }
    if err != nil {
        return nil, tools.LogErr("@isOwner/auth", "Access denied", err)
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

func checkUserOwnership(userCtx model.UserCtx, userField string, userObj interface{}) (bool, error) {
    user := userObj.(model.JsonAtom)[userField]
    var userid string
    if user == nil || user.(model.JsonAtom)["username"] == nil  {
        println("User unknown, need a database request here...")
        return false, nil
    } else {
        userid = user.(model.JsonAtom)["username"].(string)
    }

    if userCtx.Username == userid {
        return true, nil
    }
    return false, nil
}

func checkUserRole(userCtx model.UserCtx, nodeField string, nodeObj interface{}, role model.RoleType) (bool, error) {
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
    nameid := nameid_.(string)
    for _, ur := range userCtx.Roles {
        if ur.Nameid == nameid && ur.RoleType == role {
            return true, nil
        }
    }
    return false, nil
}

func checkUserRoot(userCtx model.UserCtx, nodeField string, nodeObj interface{}) (bool, error) {
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
    rootnameid := rootnameid_.(string)
    for _, ur := range userCtx.Roles {
        if ur.Rootnameid == rootnameid  {
            return true, nil
        }
    }

    return false, nil
}

func addNodeHook(u model.UserCtx, nodeObj interface{}) (bool, error) {
    var ok bool = false
    var err error

    node := nodeObj.(model.JsonAtom)
    isRoot := node["isRoot"]
    if isRoot != nil && isRoot.(bool) {
        // Create new organisation
        if u.Rights.CanCreateRoot {
            if node["parent"] != nil {
                err = fmt.Errorf("Root node can't have a parent.")
            } else if node["nameid"] != node["rootnameid"] {
                err = fmt.Errorf("Root node nameid and rootnameid are different.")
            } else {
                ok = true
            }
        } else {
            err = fmt.Errorf("You are not authorized to create new organisation.")
        }
    } else {
        // Push a new node
        // pass
    }

    return ok, err
}

//
// Utils
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
