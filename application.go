package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/mostlygeek/arp"
	"github.com/spf13/viper"
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

	ServersByMACAddress map[string]compute.Server

	stateLock     *sync.Mutex
	refreshTimer  *time.Timer
	cancelRefresh chan bool
}

// NewApplication creates new application state.
func NewApplication() *Application {
	return &Application{
		ServersByMACAddress: make(map[string]compute.Server),
		stateLock:           &sync.Mutex{},
	}
}

// Initialize performs initial configuration of the application.
func (app *Application) Initialize() error {
	viper.BindEnv("MCP_USER", "mcp.user")
	viper.BindEnv("MCP_PASSWORD", "mcp.password")
	viper.BindEnv("MCP_REGION", "mcp.region")
	viper.BindEnv("MCP_VLAN_ID", "network.vlan_id")

	viper.SetConfigType("YAML")
	viper.SetConfigFile("dd-rancheros-cc.yml")

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	app.McpRegion = viper.GetString("mcp.region")
	app.McpUser = viper.GetString("mcp.user")
	app.McpPassword = viper.GetString("mcp.password")
	app.Client = compute.NewClient(app.McpRegion, app.McpUser, app.McpPassword)

	vlanID := viper.GetString("network.vlan_id")
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

// Start polling CloudControl for server metadata.
func (app *Application) Start() {
	app.stateLock.Lock()
	defer app.stateLock.Unlock()

	// Warm up caches.
	arp.CacheUpdate()
	err := app.RefreshServerMetadata()
	if err != nil {
		log.Printf("Error refreshing servers: %s",
			err.Error(),
		)
	}

	// Periodically scan the ARP cache so we can resolve MAC addresses from client IPs.
	arp.AutoRefresh(5 * time.Second)

	app.cancelRefresh = make(chan bool, 1)
	app.refreshTimer = time.NewTimer(10 * time.Second)

	go func() {
		cancelRefresh := app.cancelRefresh
		refreshTimer := app.refreshTimer.C

		for {
			select {
			case <-cancelRefresh:
				return // Stopped

			case <-refreshTimer:
				log.Printf("Refreshing server MAC addresses...")

				err := app.RefreshServerMetadata()
				if err != nil {
					log.Printf("Error refreshing servers: %s",
						err.Error(),
					)
				}

				log.Printf("Refreshed server MAC addresses.")
			}
		}
	}()
}

// Stop polling CloudControl for server metadata.
func (app *Application) Stop() {
	app.stateLock.Lock()
	defer app.stateLock.Unlock()

	if app.cancelRefresh != nil {
		app.cancelRefresh <- true
	}
	app.cancelRefresh = nil

	app.refreshTimer.Stop()
	app.refreshTimer = nil
}
