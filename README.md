# cloud-config server for Dimension Data CloudControl and RancherOS

A simple implementation of a server to return customised cloud-config.yml (to drive RancherOS installation via iPXE) for servers in CloudControl.

It's functional, but should be considered a work in progress; feel free to create an issue if you have questions or would like to contribute.

Works on Linux and OSX, but not Windows.

Note that it won't (currently) handle the case where the iPXE server is attached to multiple VLANs (but the design could easily be extended to handle this).

## Configuration

`dd-rancheros-cc.yml`:

```yaml
mcp:
  user: "my_user"
  password: "my_password"
  region: "AU"

network:
  vlan_id: "my_vlan_id" # The Id of the VLAN where the iPXE and cloud-config server are running.
```

OR:

```bash
export MCP_USER=my_user
export MCP_PASSWORD=my_password
export MCP_REGION=AU
export MCP_VLAN_ID=my_vlan_id
```
