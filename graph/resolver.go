// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    //"fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"
    //"golang.org/x/crypto/bcrypt" 

    //"zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/tools"
)

/*
*
* Data structures initialisation
*
*/

type Resolver struct{
    // pointer on dgraph
    db tools.Dgraph
    
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    db := tools.Dgraph{
        Url: "http://localhost:8080/graphql",
    }
    r := Resolver{db:db}
    c := gen.Config{Resolvers: &r}
    //c.Directives.HasRole = hasRoleMiddleware
    c.Directives.Id = nothing
    return c
}

/*
*
* Business Logic layer methods
*
*/

func nothing (ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    return next(ctx)
}
