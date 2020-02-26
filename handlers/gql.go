package handlers

import (
    "github.com/gin-gonic/gin"
	"github.com/99designs/gqlgen/handler"

	"fractal6/gin/graph"
	gen "fractal6/gin/graph/generated"
)




// Defining the Graphql handler
func GraphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.GraphQL(
        gen.NewExecutableSchema(graph.Init()),
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
