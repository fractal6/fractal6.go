package handlers

import (
    "github.com/gin-gonic/gin"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"

	"fractal6/gin/graph"
	gen "fractal6/gin/graph/generated"
)


func hasRoleMiddleware (ctx context.Context, obj interface{}, next graphql.Resolver, role Role) (interface{}, error) {
	if !getCurrentUser(ctx).HasRole(role) {
		// block calling the next resolver
		return nil, fmt.Errorf("Access denied")
	}

	// or let it pass through
	return next(ctx)
}


// Defining the Graphql handler
func GraphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
    c := gen.Config{Resolvers: &graph.Resolver{}}
	c.Directives.HasRole = hasRoleMiddleware

	h := handler.GraphQL(
        gen.NewExecutableSchema(c),
        handler.ComplexityLimit(50),
    )

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func PlaygroundHandler(path string) gin.HandlerFunc {
	h := handler.Playground("Go GraphQL Playground", path)

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
