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

package cmd

import (
    //"fmt"
    "log"
    "time"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
    "github.com/spf13/viper"

    "fractale/fractal6.go/web"
    "fractale/fractal6.go/web/auth"
    handle6 "fractale/fractal6.go/web/handlers"
    middle6 "fractale/fractal6.go/web/middleware"
)

var tkMaster *auth.Jwt
var buildMode string

func init() {
    // Get env mode
    if buildMode == "" {
        buildMode = "DEV"
    } else {
        buildMode = "PROD"
    }

    // Jwt init
    tkMaster = auth.GetTokenMaster()
}

// RunServer launch the server
func RunServer() {
    DOMAIN := viper.GetString("server.domain")
    HOST := viper.GetString("server.hostname")
    PORT := viper.GetString("server.port")
    gqlConfig := viper.GetStringMap("graphql")
    instrumentation := viper.GetBool("server.prometheus_instrumentation")

    r := chi.NewRouter()

    var allowedOrigins []string
    if buildMode == "PROD" {
        allowedOrigins = append(allowedOrigins, "https://"+DOMAIN, "https://api."+DOMAIN, "https://staging."+DOMAIN)
    } else {
        allowedOrigins = append(allowedOrigins, "http://localhost:8000")
    }

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		//AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		//ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

    // Middleware stack
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
	r.Use(cors.Handler)
    //r.Use(middle6.RequestContextMiddleware) // Set context info
    // JWT   //r.Use(jwtauth.Verifier(tkMaster.GetAuth()))
    r.Use(middle6.JwtVerifier(tkMaster.GetAuth())) // Seek, verify and validate JWT token
    r.Use(middle6.JwtDecode) // Set user claims
    // Log request
    r.Use(middleware.Logger)
    // Recover from panic   //r.Use(middleware.Recoverer)
    r.Use(middle6.Recoverer)
    // Set a timeout value on the request context (ctx), that will signal
    // through ctx.Done() that the request has timed out and further
    // processing should be stopped.
    r.Use(middleware.Timeout(60 * time.Second))

    // Serve Prometheus instrumentation
	if instrumentation {
        go func() {
            // Update metrics in a goroutine
            for {
                handle6.InstrumentationMeasures()
                time.Sleep(time.Duration(time.Second * 500))
            }
        }()
        secured := r.Group(nil)
        secured.Use(middle6.CheckBearer)
		//secured.Handle("/metrics", promhttp.Handler()) // inclue Go collection metrics
        secured.Handle("/metrics", handle6.InstruHandler())
	}

    // Serve Graphql Playground & introspection
    if buildMode == "DEV" {
        r.Get("/playground", handle6.PlaygroundHandler("/api"))
        r.Get("/ping", handle6.Ping)

        // Overwrite gql config
        gqlConfig["introspection"] = true
    }

    // Graphql API
    r.Post("/api", handle6.GraphqlHandler(gqlConfig))

    // Auth API
    r.Group(func(r chi.Router) {
        //r.Use(middle6.EnsurePostMethod)
        r.Route("/auth", func(r chi.Router) {
            // User
            r.Post("/signup", handle6.Signup)
            r.Post("/validate", handle6.SignupValidate)
            r.Post("/login", handle6.Login)
            r.Get("/logout", handle6.Logout)
            r.Post("/tokenack", handle6.TokenAck)
            r.Post("/resetpasswordchallenge", handle6.ResetPasswordChallenge)
            r.Post("/resetpassword", handle6.ResetPassword)
            r.Post("/resetpassword2", handle6.ResetPassword2)
            r.Post("/uuidcheck", handle6.UuidCheck)

            // Organisation
            r.Post("/createorga", handle6.CreateOrga)
            r.Post("/setusercanjoin", handle6.SetUserCanJoin)
            r.Post("/setguestcancreatetension", handle6.SetGuestCanCreateTension)
        })
    })

    // Rest API
    r.Group(func(r chi.Router) {
        r.Route("/q", func(r chi.Router) {

            // Special recursive query
            r.Group(func(r chi.Router) {
                // Those data are not secured by now, and anyone can
                // query them recursively, but as there are not sensitive
                // and set them public for now.
                //r.Use(middle6.CheckRecursiveQueryRights)
                r.Post("/sub_nodes", handle6.SubNodes)
                r.Post("/sub_members", handle6.SubMembers)
                r.Post("/top_labels", handle6.TopLabels)
                r.Post("/sub_labels", handle6.SubLabels)
                r.Post("/top_roles", handle6.TopRoles)
                r.Post("/sub_roles", handle6.SubRoles)
            })

            // Special tension query (nested filters and counts)
            r.Group(func(r chi.Router) {
                // The filtering is done directly in the query resolver as
                // doing it here required to rewrite the body, which seems difficult ?!
                //r.Use(middle6.CheckTensionQueryRights)
                r.Post("/tensions_int", handle6.TensionsInt)
                r.Post("/tensions_ext", handle6.TensionsExt)
                r.Post("/tensions_all", handle6.TensionsAll)
                r.Post("/tensions_count", handle6.TensionsCount)
            })
        })
    })

    // MTA communication Endpoints
    // --
    // Notifications endpoint
    r.Post("/notifications", handle6.Notifications)
    // Mailing-list endpoint
    r.Post("/mailing", handle6.Mailing)
    // Postal webhook endpoint
    r.Post("/postal_webhook", handle6.PostalWebhook)

    // Static & Public files
    // --
    // Serve static files
    web.FileServer(r, "/assets/", "./assets", "3600")
    // Serve static frontend files
    web.FileServer(r, "/", "./public", "")

    address := HOST + ":" + PORT
    log.Printf("Running API (%s) @ http://%s", buildMode, address)
    http.ListenAndServe(address, r)
}


