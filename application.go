package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/gin-gonic/gin"
	"github.com/mostlygeek/arp"
)

// Application represents the state for the cloud-config generator.
type Application struct {
	McpUser      string
	McpPassword  string
	McpRegion    string
	SSHPublicKey string

	Client        *compute.Client
	NetworkDomain *compute.NetworkDomain
	VLAN          *compute.VLAN
}

// GetCloudConfig handles HTTP GET for /cloud-config.yml
func (app *Application) GetCloudConfig(context *gin.Context) {
	clientIP := context.ClientIP()

	log.Printf("Received cloud-config request from %s", clientIP)

	var server *compute.Server

	if clientIP != "127.0.0.1" {
		remoteMACAddress := arp.Search(context.Request.RemoteAddr)
		if remoteMACAddress == "" {
			context.String(http.StatusBadRequest,
				"Sorry, I can't figure out your MAC address from your IPv4 address (%s).", clientIP,
			)

			return
		}

		var err error
		server, err = app.FindServerByMACAddress(remoteMACAddress)
		if err != nil {
			context.Error(err)

			return
		}
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

	// The nested cloud-config.yml is the one that actually drives the installed RancherOS.
	nestedCloudConfig, err := app.GenerateInnerCloudConfig(server)
	if err != nil {
		context.Error(err)

		return
	}

	// The outer cloud-config.yml exists only to kick off the RancherOS installation with the inner cloud-config.yml.
	context.YAML(http.StatusOK,
		app.GenerateCloudConfig(nestedCloudConfig),
	)
}

// GenerateInnerCloudConfig creates customised cloud-config.yml content for the specified server.
func (app *Application) GenerateInnerCloudConfig(server *compute.Server) (cloudConfig string, err error) {
	var serializedCloudConfig []byte

	serializedCloudConfig, err = yaml.Marshal(gin.H{
		"hostname": server.Name,
		"rancher": gin.H{
			"network": gin.H{
				"interfaces": gin.H{
					"eth*": gin.H{"dhcp": false},
					"eth0": gin.H{
						"addresses": []string{
							*server.Network.PrimaryAdapter.PrivateIPv4Address,
							*server.Network.PrimaryAdapter.PrivateIPv6Address,
						},
						"gateway":      app.VLAN.IPv4GatewayAddress,
						"gateway_ipv6": app.VLAN.IPv6GatewayAddress,
						"mtu":          1500,
					},
				},
			},
		},
		"ssh_authorized_keys": []string{app.SSHPublicKey},
	})
	if err != nil {
		return
	}

	cloudConfig = string(serializedCloudConfig)

	return
}

// GenerateCloudConfig creates the generic "boilerplate" cloud-config.yml content that directs the iPXE-booted image to install RancherOS.
func (app *Application) GenerateCloudConfig(nestedCloudConfig string) gin.H {
	return gin.H{
		"write_files": []gin.H{
			gin.H{
				"path":        "/opt/rancher/bin/install.yml",
				"permissions": "0700",
				"content":     nestedCloudConfig,
			},
			gin.H{
				"path":        "/opt/rancher/bin/start.sh",
				"permissions": "0700",
				"content":     startScript,
			},
		},
	}
}

// Initialize performs initial configuration of the application.
func (app *Application) Initialize() error {
	app.McpUser = os.Getenv("MCP_USER")
	app.McpPassword = os.Getenv("MCP_PASSWORD")
	app.McpRegion = os.Getenv("MCP_REGION")
	app.Client = compute.NewClient(app.McpRegion, app.McpUser, app.McpPassword)

	var err error
	vlanID := os.Getenv("MCP_VLAN_ID")
	app.VLAN, err = app.Client.GetVLAN(vlanID)
	if err != nil {
		return err
	} else if app.VLAN == nil {
		return fmt.Errorf("Cannot find VLAN with Id '%s'", vlanID)
	}
	app.NetworkDomain, err = app.Client.GetNetworkDomain(app.VLAN.NetworkDomain.ID)
	if err != nil {
		return err
	} else if app.NetworkDomain == nil {
		return fmt.Errorf("Cannot find network domain with Id '%s'", app.VLAN.NetworkDomain.ID)
	}

	err = app.loadSSHPublicKey()
	if err != nil {
		return err
	}

	return nil
}

func (app *Application) loadSSHPublicKey() error {
	sshPublicKeyFile, err := os.Open(
		os.Getenv("HOME") + "/.ssh/id_rsa.pub",
	)
	if err != nil {
		return err
	}
	defer sshPublicKeyFile.Close()

	sshPublicKeyData, err := ioutil.ReadAll(sshPublicKeyFile)
	if err != nil {
		return err
	}

	app.SSHPublicKey = string(sshPublicKeyData)

	return nil
}
