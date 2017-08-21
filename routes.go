package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/gin-gonic/gin"
	"github.com/mostlygeek/arp"
	yaml "gopkg.in/yaml.v2"
)

// GetCloudConfig handles HTTP GET for /cloud-config.yml
func (app *Application) GetCloudConfig(context *gin.Context) {
	clientIP := context.ClientIP()

	log.Printf("Received cloud-config request from %s", clientIP)

	var server *compute.Server

	if clientIP != "127.0.0.1" {
		remoteMACAddress := arp.Search(clientIP)
		if remoteMACAddress == "" {
			context.String(http.StatusBadRequest,
				"Sorry, I can't figure out your MAC address from your IPv4 address (%s).", clientIP,
			)

			return
		}

		server = app.FindServerByMACAddress(remoteMACAddress)
		if server == nil {
			context.String(http.StatusBadRequest,
				"Sorry, %s, I can't find the server your MAC address corresponds to.",
				remoteMACAddress,
			)

			return
		}
	} else {
		log.Printf("Request originates from local machine; treating this as a test request.")

		server = createTestServer()
	}

	cloudConfig, err := app.GenerateCloudConfig(*server)
	if err != nil {
		context.Error(err)

		return
	}
	//context.String(http.StatusOK, "#cloud-config\n%s", cloudConfig)
	//context.YAML(http.StatusOK, cloudConfig)
	cloudConfigYaml, err := yaml.Marshal(cloudConfig)

	if err != nil {

		context.Error(err)

		return

	}

	context.String(http.StatusOK, fmt.Sprintf("#cloud-config\n%s",

		string(cloudConfigYaml),
	))
}
