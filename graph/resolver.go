//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
    "reflect"
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
    c.Directives.Input_maxLength = inputMaxLength
    c.Directives.Input_ensureType = ensureType
    c.Directives.Input_hasRole = hasRole 
    c.Directives.Input_hasRoot = hasRoot
    c.Directives.InputP_isOwner = isOwner
    c.Directives.InputP_RO = readOnly
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

func ensureType(ctx context.Context, obj interface{}, next graphql.Resolver, field string, type_ model.NodeType) (interface{}, error) {
    v := obj.(model.JsonAtom)[field].(model.JsonAtom)
    return nil, fmt.Errorf("Only Node type is supported, this is not: %T", v )
}

// HasRole search for authorized user by checking if user has the required role on the given node (or is owner)
func hasRole(ctx context.Context, obj interface{}, next graphql.Resolver, nodeFields []string, role model.RoleType, userField *string) (interface{}, error) {
    // Retrive userCtx from token
    userCtx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        return nil, tools.LogErr("@hasRole/userCtx", "Access denied", err)  // Login or signup
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
    }

    if err != nil {
        return nil, tools.LogErr("@hasRole/checkRole", "Access denied", err)
    }
    // Format output to get to be able to format a links in a frontend.
    // @DEBUG: get rootnameid from nameid
    e := fmt.Errorf("Contaact a coordinator to grant rights")
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
    }

    if err != nil {
        return nil, tools.LogErr("@hasRoot", "Access denied", err)
    }
    // Format output to get to be able to format a links in a frontend.
    e := fmt.Errorf("Contact a coordinator to access this ressource")
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
    var userObj map[string]interface{}
    var f string
    if userField == nil {
        f = "user"
        userObj[f] = obj
    } else {
        f = *userField
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
        println("user unknown, need a database request here !!!")
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
    objSource :=  nodeObj.(model.JsonAtom)
    nodeTarget_ :=  objSource[nodeField]
    if nodeTarget_ == nil {
        err := fmt.Errorf("node target  unknown, need a database request here !!!")
        return false, tools.LogErr(fmt.Sprintf("@hasRole/node undefined (n:%s)", nodeField), "Access denied", err)
    }
    nodeTarget := nodeTarget_.(model.JsonAtom)

    // Extract node identifier
    nameid := nodeTarget["nameid"]
    if nameid == "" {
        err := fmt.Errorf("node target  unknown, need a database request here !!!")
        return false, tools.LogErr(fmt.Sprintf("@hasRole/fieldid undefined (n:%s)", nodeField), "Access denied", err)
    }

    // Search for rights
    for _, ur := range userCtx.Roles {
        if ur.Nameid == nameid && ur.RoleType == role {
            return true, nil
        }
    }
    return false, nil
}

func checkUserRoot(userCtx model.UserCtx, nodeField string, nodeObj interface{}) (bool, error) {
    // Check that nodes are present
    objSource :=  nodeObj.(model.JsonAtom)
    nodeTarget_ :=  objSource[nodeField]
    if nodeTarget_ == nil {
        err := fmt.Errorf("node target  unknown, need a database request here !!!")
        return false, tools.LogErr(fmt.Sprintf("@hasRole/node undefined (n:%s)", nodeField), "Access denied", err)
    }
    nodeTarget := nodeTarget_.(model.JsonAtom)

    // Extract node identifiers
    rootnameid := nodeTarget["rootnameid"].(string)
    if rootnameid == "" {
        err := fmt.Errorf("node target  unknown, need a database request here !!!")
        return false, tools.LogErr(fmt.Sprintf("@hasRole/fieldid undefined (n:%s)", nodeField), "Access denied", err)
    }

    // Search for rights
    for _, ur := range userCtx.Roles {
        if ur.Rootnameid == rootnameid  {
            return true, nil
        }
    }

    return false, nil
}
