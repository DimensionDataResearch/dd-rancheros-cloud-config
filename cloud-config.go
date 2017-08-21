package main

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

// GenerateCloudConfig creates the outer cloud-config (customised for the specified server) that directs the iPXE-booted image to install RancherOS.
//
// The outer cloud-config writes the inner cloud-config to a file and then uses that to drive the RancherOS setup.
func (app *Application) GenerateCloudConfig(server compute.Server) (cloudConfig gin.H, err error) {
	var innerCloudConfig string
	innerCloudConfig, err = app.GenerateInnerCloudConfig(server)
	if err != nil {
		return
	}

	cloudConfig = gin.H{
		"write_files": []gin.H{
			gin.H{
				"path":        "/opt/rancher/bin/install.yml",
				"permissions": "0700",
				"content":     innerCloudConfig,
			},
			gin.H{
				"path":        "/opt/rancher/bin/start.sh",
				"permissions": "0700",
				"content":     rancherOSInstallScript,
			},
		},
	}

	return
}

// GenerateInnerCloudConfig creates customised (host-specific) cloud-config for the specified server.
func (app *Application) GenerateInnerCloudConfig(server compute.Server) (cloudConfig string, err error) {
	var serializedCloudConfig []byte

	serializedCloudConfig, err = yaml.Marshal(gin.H{
		"hostname": server.Name,
		"rancher": gin.H{
			"network": gin.H{
				"interfaces": gin.H{
					"eth*": gin.H{"dhcp": false},
					"eth0": gin.H{
						"addresses": []string{
							*server.Network.PrimaryAdapter.PrivateIPv4Address + "/24",
							*server.Network.PrimaryAdapter.PrivateIPv6Address + "/64",
						},
						"gateway":      app.VLAN.IPv4GatewayAddress,
						"gateway_ipv6": app.VLAN.IPv6GatewayAddress,
						"mtu":          1500,
					},
					"dns": gin.H{
						"nameservers": []string{
							app.RancherOSDNS,
						},
					},
				},
			},
			"services": gin.H{
				"rancher-agent1": gin.H{
					"image":      app.RancherAgentVersion,
					"command":    app.RancherAgentURL,
					"privileged": true,
					"volumes": []string{
						"/var/run/docker.sock:/var/run/docker.sock",
						"/var/lib/rancher:/var/lib/rancher",
					},
					"environment": gin.H{
						"CATTLE_AGENT_IP": *server.Network.PrimaryAdapter.PrivateIPv4Address,
					},
				},
			},
		},
		"ssh_authorized_keys": []string{app.SSHPublicKeyFromYML},
	})
	if err != nil {
		return
	}
	cloudConfig = "#cloud-config\n" + string(serializedCloudConfig)
	//	cloudConfig = string(serializedCloudConfig)
	return
}
