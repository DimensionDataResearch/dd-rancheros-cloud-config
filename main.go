package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	app := NewApplication()
	err := app.Initialize()
	if err != nil {
		panic(err)
	}

	// Start polling CloudControl for server metadata.
	app.Start()

	server := gin.Default()
	server.GET("/cloud-config.yml", app.GetCloudConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "19123"
		os.Setenv("PORT", port)
	}

	fmt.Printf("Server listens on port %s.\n", port)
	server.Run()
}
