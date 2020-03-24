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

    "zerogov/fractal6.go/handlers"
    "zerogov/fractal6.go/tools"
)

var queryPath, buildMode string


func init() {
    rootCmd.AddCommand(runCmd)
    queryPath = "/api"

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

	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	cors := cors.New(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins:   []string{"*"},
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
    //r.Use(tools.RequestContextMiddleware)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Set a timeout value on the request context (ctx), that will signal
    // through ctx.Done() that the request has timed out and further
    // processing should be stopped.
    r.Use(middleware.Timeout(60 * time.Second))

    //r.POST("signup", handler.Signup)
    //r.POST("signin", handlers.Signin)
    //r.POST("signout", handler.Signout)

    if buildMode == "DEV" {
        // Serve Graphql Playground
        r.Get("/playground", handlers.PlaygroundHandler(queryPath))
        r.Get("/ping", handlers.Ping)

        // Serve frontend static files
        tools.FileServer(r, "/", "./web/public")

        // Overwrite gql config
        gqlConfig["introspection"] = true
    }

    // Serve Graphql Api
    r.Post(queryPath, handlers.GraphqlHandler(gqlConfig))

    log.Println("Running @ http://" + HOST + ":" + PORT)
    http.ListenAndServe(HOST + ":" + PORT, r)
}


