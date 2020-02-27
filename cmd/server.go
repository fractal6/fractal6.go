package cmd

import (
    "log"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    "zerogov/fractal6.go/handlers"
    "zerogov/fractal6.go/internal"
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

    r := echo.New()

    // Middleware
    //r.Use(middleware.Logger())
    r.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
        Format: `[${time_rfc3339}]  ${method}  ${status}  ${uri}  ${error}  ${latency_human}` + "\n",
    }))
    r.Use(middleware.Recover())
	r.Use(internal.RouterContextToContextMiddleware)

    //r.POST("signup", handler.Signup)
    r.POST("signin", handler.Signin)
    //r.POST("signout", handler.Signout)

	if buildMode == "DEV" {
		r.GET("/ping", handlers.Ping)
		r.GET("/playground", handlers.PlaygroundHandler(queryPath))
		// Serve frontend static files
		r.Static("/", "./web/public")
	}

    // Serve Graphql Api
    r.POST(queryPath, handlers.GraphqlHandler())



    log.Println("Running @ http://" + HOST + ":" + PORT)
    r.Logger.Fatal(r.Start(HOST + ":" + PORT))
}


