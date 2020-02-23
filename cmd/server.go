package cmd

import (
	"log"

	"github.com/gin-gonic/gin"

	"fractal6/gin/handlers"
)

var HOST, PORT string

func init() {
	HOST = "localhost"
	PORT = "8888"
}

// Run web server
func Run() {
	r := gin.Default()

	// Setup routes
	r.GET("/ping", handlers.Ping())

	log.Println("Running @ http://" + HOST + ":" + PORT)
	log.Fatalln(r.Run(HOST + ":" + PORT))
}
