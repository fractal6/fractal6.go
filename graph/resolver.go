//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "log"
    "context"
    "reflect"
    "github.com/99designs/gqlgen/graphql"
    "github.com/mitchellh/mapstructure"

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
    c.Directives.Auth = authorize
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
    fieldDef := rc.Field.Name
    return nil, fmt.Errorf("`%s' field is hidden", fieldDef)
    }

func count(ctx context.Context, obj interface{}, next graphql.Resolver, field string) (interface{}, error) {
    rc := graphql.GetResolverContext(ctx)
    fieldDef := rc.Field.Name
    goFieldfDef := tools.ToGoNameFormat(fieldDef)

    // Reflect to get obj data info
    // DEBUG: use type switch instead ? (less modular but faster?)
    id := reflect.ValueOf(obj).Elem().FieldByName("ID").String()
	if id == "" {
        err := fmt.Errorf("`id' field is needed to query `%s'", fieldDef)
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

// Authorize Search for authorized user by comparing the node source agains the roles of the user.
func authorize(ctx context.Context, obj interface{}, next graphql.Resolver, nodeField string, role model.RoleType ) (interface{}, error) {
    // Retrive userCtx from token
    userCtx, err := auth.UserCtxFromContext(ctx)
    if err != nil {
        e := fmt.Errorf("Access denied: %s", err.Error())
        log.Println("@auth/"+e.Error())
        return nil, e
        ///return nil, fmt.Errorf("Access denied: Login or signup to perform this action.")
    }

    // Verify format is correct
    nodeSource_ :=  obj.(model.JsonAtom)
    nodeTarget_ :=  nodeSource_[nodeField]
    var nodeTarget model.Node
    if nodeTarget_ == nil {
        e := fmt.Errorf("Access denied: Node undefined for input %s", nodeField)
        log.Println("@auth/"+e.Error())
        return nil, e
    }
    mapstructure.Decode(nodeTarget_, &nodeTarget)
    rootnameid := nodeSource_["rootnameid"].(string)
    nameid := nodeTarget.Nameid
    if nameid == "" || rootnameid == "" {
        e := fmt.Errorf("Access denied: Node IDs undefined for input %s", nodeField)
        log.Println("@auth/"+e.Error())
        return nil, e
    }

    // Search for rights
    for _, ur := range userCtx.Roles {
        if ur.Rootnameid == rootnameid && ur.Nameid == nameid && ur.RoleType == role {
            return next(ctx)
        }
    }

    // Format output to get to be able to format a links in a frontend.
    e := fmt.Errorf("Access denied: Please contact the %s's coordinator to perform this action.", rootnameid +"/"+ nameid)
    log.Println("@auth/"+e.Error())
    return nil, e
}

