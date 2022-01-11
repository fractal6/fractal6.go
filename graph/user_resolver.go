package graph

import (
	//"fmt"
	"context"

	"github.com/99designs/gqlgen/graphql"

	. "zerogov/fractal6.go/tools"
	webauth "zerogov/fractal6.go/web/auth"
)

////////////////////////////////////////////////
// User Resolver
////////////////////////////////////////////////

// Update User - Hook
func updateUserHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    _, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    return next(ctx)
}
