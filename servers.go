package main

import (
	"os"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
)

// FindServerByMACAddress finds the server (if any) posessing a network adapter with the specified MAC address.
func (app *Application) FindServerByMACAddress(macAddress string) (*compute.Server, error) {
	page := compute.DefaultPaging()
	page.PageSize = 50

	for {
		servers, err := app.Client.ListServersInNetworkDomain(app.NetworkDomain.ID, page)
		if err != nil {
			return nil, err
		}
		if servers.IsEmpty() {
			break
		}

		for _, server := range servers.Items {
			if doesServerHaveMACAddress(server, macAddress) {
				return &server, nil
			}
		}

		page.Next()
	}

	return nil, nil
}

func doesServerHaveMACAddress(server compute.Server, macAddress string) bool {
	if *server.Network.PrimaryAdapter.MACAddress == macAddress {
		return true
	}

	for _, networkAdapter := range server.Network.AdditionalNetworkAdapters {
		if *networkAdapter.MACAddress == macAddress {
			return true
		}
	}

	return false
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
