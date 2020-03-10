// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
   // "fmt"
   // "context"
    //"github.com/99designs/gqlgen/graphql"
    //"golang.org/x/crypto/bcrypt" 

    //"zerogov/fractal6.go/graph/model"
    gen "zerogov/fractal6.go/graph/generated"
)

/*
*
* Data structures initialisation
*
*/

type Resolver struct{
    // pointer on dgraph
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    c := gen.Config{Resolvers: &Resolver{}}
    //c.Directives.HasRole = hasRoleMiddleware
    return c
}

/*
*
* Business Logic layer methods
*
*/
