package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/gin-gonic/gin"
	"github.com/mostlygeek/arp"
)

const startScript = `
#!/bin/bash
echo Y | sudo ros install -f -c /opt/rancher/bin/install.yml -d /dev/sda
`

func main() {
	mcpUser := os.Getenv("MCP_USER")
	mcpPassword := os.Getenv("MCP_PASSWORD")
	mcpRegion := os.Getenv("MCP_REGION")
	vlanID := os.Getenv("MCP_VLAN_ID")

	sshPublicKey, err := loadSSHPublicKey()
	if err != nil {
		panic(err)
	}

	// Look up basic information about our environment from CloudControl.
	client := compute.NewClient(mcpRegion, mcpUser, mcpPassword)

	vlan, err := client.GetVLAN(vlanID)
	if err != nil {
		panic(err)
	} else if vlan == nil {
		panic(fmt.Errorf("Cannot find VLAN '%s'.", vlanID))
	}

	networkDomain, err := client.GetNetworkDomain(vlan.NetworkDomain.ID)
	if err != nil {
		panic(err)
	} else if networkDomain == nil {
		panic(fmt.Errorf("Cannot find network domain '%s'.", vlan.NetworkDomain.ID))
	}

	// Periodically scan the ARP cache so we can resolve MAC addresses from IP addresses.
	arp.CacheUpdate()
	arp.AutoRefresh(5 * time.Second)

	app := gin.Default()

	localhost4 := os.Getenv("MCP_TEST_HOST_IPV4")
	if localhost4 == "" {
		localhost4 = "127.0.0.1"
	}

	localhost6 := os.Getenv("MCP_TEST_HOST_IPV6")
	if localhost6 == "" {
		localhost6 = "::1"
	}

	app.GET("/cloud-config.yml", func(context *gin.Context) {
		clientIP := context.ClientIP()

		log.Printf("Received cloud-config request from %s", clientIP)

		var server *compute.Server

		if clientIP == "127.0.0.1" {
			log.Printf("Request originates from local machine; treating this as a test request.")

			server = &compute.Server{
				Name: os.Getenv("HOST"),
				Network: compute.VirtualMachineNetwork{
					PrimaryAdapter: compute.VirtualMachineNetworkAdapter{
						PrivateIPv4Address: &localhost4,
						PrivateIPv6Address: &localhost6,
					},
				},
			}

		} else {
			remoteMACAddress := arp.Search(context.Request.RemoteAddr)
			if remoteMACAddress == "" {
				context.String(http.StatusBadRequest,
					"Sorry, I can't figure out your MAC address from your IPv4 address (%s).", clientIP,
				)

				return
			}

			var err error
			server, err = findServerByMACAddress(remoteMACAddress, client, networkDomain.ID)
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
		}

		// The nested cloud-config.yml is the one that actually drives the installed RancherOS.
		nestedCloudConfig, err := generateInnerCloudConfig(
			server,
			vlan.IPv4GatewayAddress,
			vlan.IPv6GatewayAddress,
			sshPublicKey,
		)
		if err != nil {
			context.Error(err)

			return
		}

		// Render cloud-config.yml
		context.YAML(http.StatusOK, gin.H{
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
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "19123"
		os.Setenv("PORT", port)
	}

	fmt.Printf("Server listens on port %s.\n", port)
	app.Run()
}

func loadSSHPublicKey() (string, error) {
	sshPublicKeyFile, err := os.Open(
		os.Getenv("HOME") + "/.ssh/id_rsa.pub",
	)
	if err != nil {
		return "", err
	}
	defer sshPublicKeyFile.Close()

	sshPublicKeyData, err := ioutil.ReadAll(sshPublicKeyFile)
	if err != nil {
		return "", err
	}

	return string(sshPublicKeyData), nil
}

func findServerByMACAddress(macAddress string, client *compute.Client, networkDomainID string) (*compute.Server, error) {
	page := compute.DefaultPaging()
	page.PageSize = 50

	for {
		servers, err := client.ListServersInNetworkDomain(networkDomainID, page)
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

func generateInnerCloudConfig(server *compute.Server, gatewayIPv4 string, gatewayIPv6 string, sshPublicKey string) (cloudConfig string, err error) {
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
						"gateway":      gatewayIPv4,
						"gateway_ipv6": gatewayIPv6,
						"mtu":          1500,
					},
				},
			},
		},
		"ssh_authorized_keys": []string{sshPublicKey},
	})
	if err != nil {
		return
	}

	cloudConfig = string(serializedCloudConfig)

	return
}
