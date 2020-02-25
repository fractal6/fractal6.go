package handlers

import (
    "github.com/gin-gonic/gin"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"fractal6/gin/graph"
	"fractal6/gin/graph/generated"
)




// Defining the Graphql handler
func GraphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func PlaygroundHandler(path string) gin.HandlerFunc {
	h := playground.Handler("Go GraphQL Playground", path)

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
