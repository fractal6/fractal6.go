package handlers

import (
    "fmt"
    "context"
    "net/http"
    "runtime/debug"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
    "github.com/99designs/gqlgen/graphql/playground"
    "github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/99designs/gqlgen/graphql"

	gen "fractale/fractal6.go/graph/generated"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/web/email"
)


// Defining the Graphql handler
func GraphqlHandler(c map[string]interface{}) http.HandlerFunc {
    introspection := c["introspection"].(bool)
    complextityLimit := int(c["complexity_limit"].(int64))

	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.New(gen.NewExecutableSchema(graph.Init()))

    // Enable transport layers
    h.AddTransport(transport.Options{})
    h.AddTransport(transport.POST{})
    //h.AddTransport(transport.Websocket{
    //    KeepAlivePingInterval: 10 * time.Second,
    //})
    //h.AddTransport(transport.MultipartForm{})

    // Limit query complexity
	h.Use(extension.FixedComplexityLimit(complextityLimit))

    // Enable introspection
    if introspection {
        h.Use(extension.Introspection{})
    }

	// Set the default behavior to handle non implemented query
	h.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
        qn := graphql.GetResolverContext(ctx).Field.Name

        // Log error
        fmt.Printf("panic on `%s`:\n%s\n", qn, string(debug.Stack()))
        email.SendMaintainerEmail(
            fmt.Sprintf("[fractal6-api/error] %v", err),
            string(debug.Stack()),
        )

        return gqlerror.Errorf("Internal error on '%s': %v", qn, err)
	})

    return h.ServeHTTP

}

// Defining the Playground handler
func PlaygroundHandler(path string) http.HandlerFunc {
    h := playground.Handler("Fractale playground", path)
    return h.ServeHTTP
}
