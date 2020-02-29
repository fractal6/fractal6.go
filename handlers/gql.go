package handlers

import (
    "net/http"
	"github.com/99designs/gqlgen/handler"

	gen "zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph"
	//"zerogov/fractal6.go/internal"
)




// Defining the Graphql handler
func GraphqlHandler() http.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.GraphQL(
        gen.NewExecutableSchema(graph.Init()),
        handler.ComplexityLimit(50),
    )

    return h.ServeHTTP

}

// Defining the Playground handler
func PlaygroundHandler(path string) http.HandlerFunc {
	h := handler.Playground("Go GraphQL Playground", path)

    return h.ServeHTTP
}
