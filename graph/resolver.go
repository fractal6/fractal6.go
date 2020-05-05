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
    //c.Directives.HasRole = hasRoleMiddleware
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
    fmt.Println(v)
    fmt.Println("Sould be a list of Node (checl that type_ == v.type_ !")
    panic("not implemented")
    return next(ctx)
}

//func hasRoleMiddleware(ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {
//
//    fmt.Println(ctx)
//    //if !getCurrentUser(ctx).HasRole(role) {
//    //    // block calling the next resolver
//    //     fmt.Println(ctx)
//    //    return nil, fmt.Errorf("Access denied")
//    //}
//
//    // or let it pass through
//    return next(ctx)
//}
