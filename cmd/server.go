package cmd

import (
    "log"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/contrib/static"

    "zerogov/fractal6.go/handlers"
    "zerogov/fractal6.go/utils"
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
    r := gin.Default()
	r.Use(utils.GinContextToContextMiddleware())

    HOST := viper.GetString("server.host")
    PORT := viper.GetString("server.port")


    // Serve Graphql Api
    r.POST(queryPath, handlers.GraphqlHandler())


	if buildMode == "DEV" {
		r.GET("/ping", handlers.Ping())
		r.GET("/playground", handlers.PlaygroundHandler(queryPath))
		// Serve frontend static files
		r.Use(static.Serve("/", static.LocalFile("./web/public", false)))
	}


    log.Println("Running @ http://" + HOST + ":" + PORT)
    log.Fatalln(r.Run(HOST + ":" + PORT))
}


