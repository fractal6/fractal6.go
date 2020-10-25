package cmd

import (
    "log"
    "time"
    "net/http"
    "github.com/go-chi/chi"
    "github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"

    //"zerogov/fractal6.go/web"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/web/middleware/jwtauth"
    handle6 "zerogov/fractal6.go/web/handlers"
    middle6 "zerogov/fractal6.go/web/middleware"
)

var tkMaster *auth.Jwt
var buildMode string

func init() {
    // Jwt init
    tkMaster = auth.GetTokenMaster()

    // Cli init
    rootCmd.AddCommand(runCmd)

    // Get env mode
    if buildMode == "" {
        buildMode = "DEV"
    } else {
        buildMode = "PROD"
    }
}

var runCmd = &cobra.Command{
    Use:   "run",
    Short: "run server.",
    Long:  `run server.`,
    Run: func(cmd *cobra.Command, args []string) {
        RunServer()
    },
}

// RunServer launch the server
func RunServer() {
    HOST := viper.GetString("server.host")
    PORT := viper.GetString("server.port")
    gqlConfig := viper.GetStringMap("graphql")

    r := chi.NewRouter()

    var allowedOrigins []string
    if buildMode == "PROD" {
        allowedOrigins = append(allowedOrigins,  "https://fractale.co")
    } else {
        allowedOrigins = append(allowedOrigins,  "http://localhost:8000")
    }

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		//AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		//AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		//ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		//MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

    // Middleware stack
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
	r.Use(cors.Handler)
    //r.Use(middle6.RequestContextMiddleware) // Set context info
    // JWT
    r.Use(jwtauth.Verifier(tkMaster.GetAuth())) // Seek, verify and validate JWT token
    r.Use(middle6.JwtDecode) // Set user claims
    // Log request
    r.Use(middleware.Logger)
    // Recover from panic
    r.Use(middleware.Recoverer)
    // Set a timeout value on the request context (ctx), that will signal
    // through ctx.Done() that the request has timed out and further
    // processing should be stopped.
    r.Use(middleware.Timeout(60 * time.Second))

    // Auth handlers
    r.Group(func(r chi.Router) {
        //r.Use(middle6.EnsurePostMethod)
        r.Route("/auth", func(r chi.Router) {
            // User
            r.Post("/signup", handle6.Signup)
            r.Post("/login", handle6.Login)
            r.Post("/tokenack", handle6.TokenAck)

            // Organisation
            r.Post("/createorga", handle6.CreateOrga)
        })
    })

    // Http/Rest API
    r.Group(func(r chi.Router) {
        r.Route("/q", func(r chi.Router) {
            // query
            r.Post("/sub_children", handle6.SubChildren)
            r.Post("/sub_members", handle6.SubMembers)
        })
    })

    // Graphql API
    r.Post("/api", handle6.GraphqlHandler(gqlConfig))

    if buildMode == "DEV" {
        // Serve Graphql Playground
        r.Get("/playground", handle6.PlaygroundHandler("/api"))
        r.Get("/ping", handle6.Ping)

        // Serve frontend static files
        //web.FileServer(r, "/", "./public")

        // Overwrite gql config
        gqlConfig["introspection"] = true
    }

    address := HOST + ":" + PORT
    log.Printf("Running (%s) @ http://%s", buildMode, address)
    http.ListenAndServe(address, r)
}


