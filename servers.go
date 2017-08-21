package main

import (
	"log"
	"os"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
)

// RefreshServerMetadata refreshes the map of MAC addresses to server metadata.
func (app *Application) RefreshServerMetadata(acquireStateLock bool) error {
	if acquireStateLock {
		app.stateLock.Lock()
		defer app.stateLock.Unlock()
	}

	serversByMACAddress := make(map[string]compute.Server)

	page := compute.DefaultPaging()
	page.PageSize = 50

	for {
		servers, err := app.Client.ListServersInNetworkDomain(app.NetworkDomain.ID, page)
		if err != nil {
			log.Printf("Error in ListServersInNetworkDomain: %s", err.Error())

			return err
		}
		if servers.IsEmpty() {
			log.Printf("No more servers in network domain '%s'", app.NetworkDomain.ID)

			break
		}

		for _, server := range servers.Items {
			// Ignore servers that are being deployed or destroyed.
			if server.Network.PrimaryAdapter.PrivateIPv4Address == nil {
				log.Printf("Skipping server '%s' ('%s') because it has no private IPv4 address",
					server.Name,
					server.ID,
				)

				continue
			}

			primaryMACAddress := *server.Network.PrimaryAdapter.MACAddress
			serversByMACAddress[primaryMACAddress] = server

			for _, additionalNetworkAdapter := range server.Network.AdditionalNetworkAdapters {
				// Ignore network adapters that are being deployed or destroyed.
				if additionalNetworkAdapter.PrivateIPv4Address == nil {
					log.Printf("Skipping additional network adapter '%s' (MAC='%s') of server '%s' ('%s') because it has no private IPv4 address",
						*additionalNetworkAdapter.ID,
						*additionalNetworkAdapter.MACAddress,
						server.Name,
						server.ID,
					)

					continue
				}

				additionalMACAddress := *additionalNetworkAdapter.MACAddress
				serversByMACAddress[additionalMACAddress] = server
			}
		}

		page.Next()
	}

	log.Printf("ServersByMACAddress = '%#v'", serversByMACAddress)

	app.ServersByMACAddress = serversByMACAddress

	return nil
}

// FindServerByMACAddress finds the server (if any) posessing a network adapter with the specified MAC address.
func (app *Application) FindServerByMACAddress(macAddress string) *compute.Server {
	app.stateLock.Lock()
	defer app.stateLock.Unlock()

	server, ok := app.ServersByMACAddress[macAddress]
	if ok {
		return &server
	}

	return nil
}

// Create a test server for calls from localhost.
func createTestServer() *compute.Server {
	localhost4 := os.Getenv("MCP_TEST_HOST_IPV4")
	if localhost4 == "" {
		localhost4 = "127.0.0.1"
	}

	localhost6 := os.Getenv("MCP_TEST_HOST_IPV6")
	if localhost6 == "" {
		localhost6 = "::1"
	}

	return &compute.Server{
		Name: os.Getenv("HOST"),
		Network: compute.VirtualMachineNetwork{
			PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
				PrivateIPv4Address: &localhost4,
				PrivateIPv6Address: &localhost6,
			},
		},
	}
}
