VERSION = 0.0.1-beta4
VERSION_INFO_FILE = ./version-info.go

BIN_DIRECTORY   = _bin
EXECUTABLE_NAME = cloud-config-server
DIST_ZIP_PREFIX = $(EXECUTABLE_NAME).v$(VERSION)

REPO_BASE     = github.com/DimensionDataResearch
REPO_ROOT     = $(REPO_BASE)/dd-rancheros-cloud-config
VENDOR_ROOT   = $(REPO_ROOT)/vendor

default: fmt build

fmt:
	go fmt $(REPO_ROOT)

clean:
	rm -rf $(BIN_DIRECTORY) $(VERSION_INFO_FILE)
	go clean $(REPO_ROOT)/...

# Peform a development (current-platform-only) build.
dev: version fmt
	go build -o $(BIN_DIRECTORY)/$(EXECUTABLE_NAME)

# Perform a full (all-platforms) build.
build: version build-linux64 build-mac64

build-linux64: version
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIRECTORY)/linux-amd64/$(EXECUTABLE_NAME)

build-mac64: version
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIRECTORY)/darwin-amd64/$(EXECUTABLE_NAME)

# Produce archives for a GitHub release.
dist: build
	cd $(BIN_DIRECTORY)/linux-amd64 && \
		zip -9 ../$(DIST_ZIP_PREFIX).linux-amd64.zip $(EXECUTABLE_NAME)
	cd $(BIN_DIRECTORY)/darwin-amd64 && \
		zip -9 ../$(DIST_ZIP_PREFIX)-darwin-amd64.zip $(EXECUTABLE_NAME)

version: $(VERSION_INFO_FILE)

$(VERSION_INFO_FILE): Makefile
	@echo "Update version info: v$(VERSION)"
	@echo "package main\n\n// ProductVersion is the current version of the CloudControl server for RancherOS cloud-config.\nconst ProductVersion = \"v$(VERSION) (`git rev-parse HEAD`)\"" > $(VERSION_INFO_FILE)
