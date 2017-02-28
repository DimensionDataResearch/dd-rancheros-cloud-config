package main

import (
	"os"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
)

// RefreshServerMetadata refreshes the map of MAC addresses to server metadata.
func (app *Application) RefreshServerMetadata() error {
	app.stateLock.Lock()
	defer app.stateLock.Unlock()

	serversByMACAddress := make(map[string]compute.Server)

	page := compute.DefaultPaging()
	page.PageSize = 50

	for {
		servers, err := app.Client.ListServersInNetworkDomain(app.NetworkDomain.ID, page)
		if err != nil {
			return err
		}
		if servers.IsEmpty() {
			break
		}

		for _, server := range servers.Items {
			// Ignore servers that are being deployed or destroyed.
			if server.Network.PrimaryAdapter.PrivateIPv4Address == nil {
				continue
			}

			primaryMACAddress := *server.Network.PrimaryAdapter.MACAddress
			app.ServersByMACAddress[primaryMACAddress] = server

			for _, additionalNetworkAdapter := range server.Network.AdditionalNetworkAdapters {
				// Ignore network adapters that are being deployed or destroyed.
				if additionalNetworkAdapter.PrivateIPv4Address == nil {
					continue
				}

				additionalMACAddress := *additionalNetworkAdapter.MACAddress
				app.ServersByMACAddress[additionalMACAddress] = server
			}
		}

		page.Next()
	}

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
