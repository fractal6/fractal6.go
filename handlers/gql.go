package handlers

import (
    "github.com/labstack/echo/v4"
	"github.com/99designs/gqlgen/handler"

	gen "zerogov/fractal6.go/graph/generated"
	"zerogov/fractal6.go/graph"
	//"zerogov/fractal6.go/internal"
)




// Defining the Graphql handler
func GraphqlHandler() echo.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.GraphQL(
        gen.NewExecutableSchema(graph.Init()),
        handler.ComplexityLimit(50),
    )

	return func(c echo.Context) error {
        // cc := c.(*internal.GlobalContext)
        h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// Defining the Playground handler
func PlaygroundHandler(path string) echo.HandlerFunc {
	h := handler.Playground("Go GraphQL Playground", path)

	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
        return nil
	}
}
