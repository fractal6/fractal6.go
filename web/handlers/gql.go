package handlers

import (
    "net/http"
	"github.com/99designs/gqlgen/handler"

	gen "zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph"
)




// Defining the Graphql handler
func GraphqlHandler(c map[string]interface{}) http.HandlerFunc {
    introspection := c["introspection"].(bool)
    complextityLimit := int(c["complexity_limit"].(int64))

	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.GraphQL(
        gen.NewExecutableSchema(graph.Init()),
        handler.IntrospectionEnabled(introspection),
        handler.ComplexityLimit(complextityLimit),
    )

    return h.ServeHTTP

}

// Defining the Playground handler
func PlaygroundHandler(path string) http.HandlerFunc {
	h := handler.Playground("Go GraphQL Playground", path)

    return h.ServeHTTP
}
