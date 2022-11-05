/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package handlers

import (
    "net/http"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
    "github.com/99designs/gqlgen/graphql/playground"

	gen "fractale/fractal6.go/graph/generated"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/web/middleware"
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
	h.SetRecoverFunc(middleware.GqlRecover)

    return h.ServeHTTP

}

// Defining the Playground handler
func PlaygroundHandler(path string) http.HandlerFunc {
    h := playground.Handler("Fractale playground", path)
    return h.ServeHTTP
}
