package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mostlygeek/arp"
)

const startScript = `
#!/bin/bash
echo Y | sudo ros install -f -c /opt/rancher/bin/install.yml -d /dev/sda
`

func main() {
	app := &Application{}
	err := app.Initialize()
	if err != nil {
		panic(err)
	}

	// Periodically scan the ARP cache so we can resolve MAC addresses from IP addresses.
	arp.CacheUpdate()
	arp.AutoRefresh(5 * time.Second)

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
