//go:generate go run github.com/99designs/gqlgen -v

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import (
    "fmt"
    "context"
	"github.com/99designs/gqlgen/graphql"

	"zerogov/fractal6.go/graph/model"
	gen "zerogov/fractal6.go/graph/generated"
)

type Resolver struct {
	count int
	todos []*model.Todo
}

func hasRoleMiddleware (ctx context.Context, obj interface{}, next graphql.Resolver, role model.Role) (interface{}, error) {

    fmt.Println(ctx)
    //if !getCurrentUser(ctx).HasRole(role) {
    //    // block calling the next resolver
    //     fmt.Println(ctx)
    //    return nil, fmt.Errorf("Access denied")
	//}

	// or let it pass through
	return next(ctx)
}

// Init initialize shema config and Directives...
func Init() gen.Config {
    c := gen.Config{Resolvers: &Resolver{}}
	c.Directives.HasRole = hasRoleMiddleware

    return c
}

