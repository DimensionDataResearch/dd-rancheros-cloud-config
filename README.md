# cloud-config server for Dimension Data CloudControl and RancherOS

A simple implementation of a server to return customised cloud-config.yml (to drive RancherOS installation via iPXE) for servers in CloudControl.

It's functional, but should be considered a work in progress; feel free to create an issue if you have questions or would like to contribute.


## Configuration

* Supply credentials and CloudControl region via environment variables (`MCP_USER`, `MCP_PASSWORD`, `MCP_REGION`).
* Specify the Id (from CloudControl) of the local VLAN via the `MCP_VLAN_ID` environment variable.
