VERSION ?= v6.1.0
IMAGE   ?= mydaemon
HOST_PORT ?= 8080
GOOS   ?= linux
GOARCH ?= amd64

# Detect OS
ifeq ($(OS),Windows_NT)
    SHELL := pwsh
    IS_WINDOWS := 1
else
    IS_WINDOWS := 0
endif

# Only define SNET_CONFIG for docker-run on host
ifeq ($(IS_WINDOWS),1)
    SNET_CONFIG := $(shell pwsh -Command "Write-Output $$PWD")/snet-config
else
    SNET_CONFIG := $(PWD)/snet-config
endif

.PHONY: build release run test clean build_in_docker gen lint help

define RUN_SCRIPT
$(if $(IS_WINDOWS),pwsh -File ./scripts/powershell/$(1).ps1 $(2),bash ./scripts/$(1) $(2))
endef

gen:
	$(call RUN_SCRIPT,install_deps)

gen-artifacts:gen
install-deps:gen


# Build the project
build:
	gen
	$(call RUN_SCRIPT,build,$(GOOS) $(GOARCH) $(VERSION))


build-in-docker:
	$(call RUN_SCRIPT,build_in_docker)

# Build a Docker image from source
docker-build:
	docker build -f docker/Dockerfile.build -t $(IMAGE):$(VERSION) .

# Build a Docker image from GitHub releases
docker-release:
	docker build -f docker/Dockerfile.release -t $(IMAGE):$(VERSION) .

docker-run:
ifeq ($(IS_WINDOWS),1)
	SNET_CONFIG := $(shell pwsh -Command "Write-Output $$PWD")/snet-config
else
	SNET_CONFIG := $(PWD)/snet-config
endif
ifeq ($(wildcard $(SNET_CONFIG)/snetd.config.json),)
	$(error "snet-config/snetd.config.json not found! Create config before docker-run")
endif
	docker run --rm -it -v "$(SNET_CONFIG):/etc/singnet:ro" -p $(HOST_PORT):$(HOST_PORT) $(IMAGE):$(VERSION)

lint:
	golangci-lint run

# Run tests
test:
	go test ./...

clean:
ifeq ($(OS),Windows_NT)
	del /Q /S build
else
	rm -rf build
endif


help:
	@echo "Available make targets:"
	@echo "  build            - Build the daemon binary, Example: make build GOOS=linux GOARCH=amd64 VERSION=v6.1.0"
	@echo "  release          - Build release version"
	@echo "  run              - Run the daemon locally"
	@echo "  build_in_docker  - Build the daemon inside Docker"
	@echo "  docker-build     - Build Docker image from source"
	@echo "  docker-release   - Build Docker image from GitHub release"
	@echo "  docker-run       - Run Docker container with daemon"
	@echo "  gen              - Generate proto and smart-contract bindings"
	@echo "  lint             - Run linter"
	@echo "  test             - Run tests"
	@echo "  clean            - Remove build artifacts"
	@echo "  gen-artifacts    - Alias for gen"
	@echo "  install-deps     - Alias for gen"
