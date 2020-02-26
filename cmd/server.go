package cmd

import (
    "log"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/contrib/static"

    "fractal6/gin/handlers"
    "fractal6/gin/utils"
)


func init() {
    rootCmd.AddCommand(runCmd)
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

    queryPath := "/api"

    // Serve Graphql Api
    r.POST(queryPath, handlers.GraphqlHandler())

    // Serve frontend static files
    r.Use(static.Serve("/", static.LocalFile("./web/public", true)))

    r.GET("/ping", handlers.Ping())
    r.GET("/", handlers.PlaygroundHandler(queryPath))


    log.Println("Running @ http://" + HOST + ":" + PORT)
    log.Fatalln(r.Run(HOST + ":" + PORT))
}


