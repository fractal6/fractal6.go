/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
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
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/spf13/viper"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web"
	"fractale/fractal6.go/web/auth"
	handle6 "fractale/fractal6.go/web/handlers"
	middle6 "fractale/fractal6.go/web/middleware"
)

var (
	tkMaster    *auth.Jwt
	buildMode   string
	buildBranch string
)

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
	if buildMode == "PROD" && buildBranch != "prod" { // @DEBUG: prod branch is for public build...
		allowedOrigins = append(allowedOrigins, "https://"+DOMAIN, "https://api."+DOMAIN, "https://staging."+DOMAIN)
	} else {
		allowedOrigins = append(allowedOrigins, "http://localhost:3000")
		for port := 8000; port <= 8888; port++ {
			allowedOrigins = append(allowedOrigins, fmt.Sprintf("http://localhost:%d", port))
		}
	}

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler)
	// r.Use(middle6.RequestContextMiddleware) // Set context info
	// JWT   //r.Use(jwtauth.Verifier(tkMaster.GetAuth()))
	r.Use(middle6.JwtVerifier(tkMaster.GetAuth())) // Seek, verify and validate JWT token
	r.Use(middle6.JwtDecode)                       // Set user claims
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
		secured.Use(middle6.CheckBearerProm)
		// secured.Handle("/metrics", promhttp.Handler()) // inclue Go collection metrics
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
		// r.Use(middle6.EnsurePostMethod)
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
			r.Post("/updatepassword", handle6.UpdatePassword)

			// Organisation
			r.Post("/createorga", handle6.CreateOrga)
			r.Post("/createorga/spreadsheet", handle6.ImportOrga)
			r.Post("/setusercanjoin", handle6.SetUserCanJoin)
			r.Post("/setguestcancreatetension", handle6.SetGuestCanCreateTension)
			r.Post("/setlexicon", handle6.SetLexicon)
			r.Post("/setistemplatetensiononly", handle6.SetIsTemplateTensionOnly)
			r.Post("/setispinnedtensionfetchrecursively", handle6.SetisPinnedTensionfetchRecursively)

			// Special
			r.Post("/makeowner", handle6.MakeOwner)
		})
	})

	// Rest API
	r.Group(func(r chi.Router) {
		r.Route("/q", func(r chi.Router) {
			// Two-phase visibility filtering: see web/handlers/q_nodes.go.
			top := db.GetDB().GetTopNodeVisibilities
			sub := db.GetDB().GetSubNodeVisibilities
			r.Route("/nodes", func(r chi.Router) {
				r.Post("/sub", handle6.SubNodes)
				r.Post("/subauth", handle6.SubNodeAuth)
			})
			r.Route("/members", func(r chi.Router) {
				r.Post("/sub", handle6.SubMembers)
			})
			r.Route("/labels", func(r chi.Router) {
				r.Post("/top", handle6.NodeHolderHandler(top, db.GetDB().GetLabelsIn))
				r.Post("/sub", handle6.NodeHolderHandler(sub, db.GetDB().GetLabelsIn))
			})
			r.Route("/roles", func(r chi.Router) {
				r.Post("/top", handle6.NodeHolderHandler(top, db.GetDB().GetRolesIn))
				r.Post("/sub", handle6.NodeHolderHandler(sub, db.GetDB().GetRolesIn))
			})
			r.Route("/tension_templates", func(r chi.Router) {
				r.Post("/top", handle6.NodeHolderHandler(top, db.GetDB().GetTopTensionTemplatesIn))
				r.Post("/sub", handle6.NodeHolderHandler(sub, db.GetDB().GetTensionTemplatesIn))
			})
			r.Route("/project_templates", func(r chi.Router) {
				r.Post("/top", handle6.NodeHolderHandler(top, db.GetDB().GetTopProjectTemplatesIn))
				r.Post("/sub", handle6.NodeHolderHandler(sub, db.GetDB().GetProjectTemplatesIn))
			})
			r.Route("/projects", func(r chi.Router) {
				r.Post("/sub", handle6.NodeHolderHandler(sub, db.GetDB().GetProjectsIn))
			})

			// Special tension query (nested filters and counts)
			r.Route("/tensions", func(r chi.Router) {
				r.Post("/light", handle6.TensionsHandler("light"))
				r.Post("/int", handle6.TensionsHandler("int"))
				r.Post("/ext", handle6.TensionsHandler("ext"))
				r.Post("/all", handle6.TensionsHandler("all"))
				r.Post("/count", handle6.TensionsCount)
			})
		})
	})

	// File attachments (S3-backed). See docs/file-storage.md.
	// /file/<id>            : auth-checked 302 to a presigned URL (read)
	// POST /file/upload     : multipart upload, comment-author only
	// DELETE /file/<id>     : comment-author only
	//
	// Storage is optional: handlers receive nil and respond 503 when [storage]
	// is unset. See checkServices in cmd/health.go for the startup healthchecks.
	storageCli := initStorage()
	checkServices(storageCli)
	r.Group(func(r chi.Router) {
		r.Route("/file", func(r chi.Router) {
			r.Post("/upload", handle6.FileUploadHandler(storageCli))
			r.Get("/{id}", handle6.FileGetHandler(storageCli))
			r.Delete("/{id}", handle6.FileDeleteHandler(storageCli))
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
	assetsCacheControl := "max-age=3600"
	if buildMode == "DEV" {
		assetsCacheControl = "no-store, no-cache, must-revalidate"
	}
	web.FileServer(r, "/assets/", "./assets", assetsCacheControl)
	// Serve static frontend files
	web.FileServer(r, "/", "./public", "")

	address := HOST + ":" + PORT
	log.Printf("Running API (%s) @ http://%s", buildMode, address)
	err := http.ListenAndServe(address, r)
	if err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
}
