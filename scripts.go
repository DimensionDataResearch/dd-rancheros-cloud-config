package main

const rancherOSInstallScript = `#!/bin/bash
echo Y | sudo ros install -f -c /opt/rancher/bin/install.yml -d /dev/sda
`
