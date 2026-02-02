# Variables
GO_BIN_DIR := ~/go/bin
CLI_DIR := ./cmd/main/...
WALLET_DIR := ./cmd/rpc/web/wallet
EXPLORER_DIR := ./cmd/rpc/web/explorer
DOCKER_DIR := ./.docker/compose.yaml

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print each command's help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# Targets, this is a list of all available commands which can be executed using the make command.
.PHONY: build/canopy build/canopy-full build/wallet build/explorer test/all dev/deps docker/up \
	docker/down docker/build docker/up-fast docker/down docker/logs \
	build/plugin build/kotlin-plugin build/go-plugin build/all-plugins docker/plugin \
	docker/run docker/run-kotlin docker/run-go docker/run-typescript docker/run-python docker/run-csharp

# ==================================================================================== #
# BUILDING
# ==================================================================================== #

## build/canopy: build the canopy binary into the GO_BIN_DIR
build/canopy:
	go build -o $(GO_BIN_DIR)/canopy $(CLI_DIR)

## build/canopy-full: build the canopy binary and its wallet and explorer altogether
build/canopy-full: build/wallet build/explorer build/canopy

## build/wallet: build the canopy's wallet project
build/wallet:
	npm install --prefix $(WALLET_DIR) && npm run build --prefix $(WALLET_DIR)

## build/explorer: build the canopy's explorer project
build/explorer:
	npm install --prefix $(EXPLORER_DIR) && npm run build --prefix $(EXPLORER_DIR)

# ==================================================================================== #
# TESTING
# ==================================================================================== #

## test/all: run all canopy tests
test/all:
	go test ./... -p=1

## test/fuzz: run all canopy fuzz tests individually
test/fuzz:
	# Golang currently does not support multiple fuzz targets, so each need to be called individually
	# For more information check the open issue: https://github.com/golang/go/issues/46312
	go test -fuzz=FuzzKeyDecodeEncode ./store -fuzztime=5s
	go test -fuzz=FuzzBytesToBits ./store -fuzztime=5s

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## dev/deps: install all dependencies on the project's directory
dev/deps:
	go mod vendor

# Detect OS to run the docker compose command, this is because Docker for MacOS does not support the
# modern docker compose command and still uses the legacy docker-compose
ifeq ($(shell uname -s),Darwin)
    DOCKER_COMPOSE_CMD = docker-compose
else
    DOCKER_COMPOSE_CMD = docker compose
endif

## docker/build: build the compose containers
docker/build:
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) build

## docker/up: build and start the compose containers in detached mode
docker/up:
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) down && \
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) up --build -d

## docker/down: stop the compose containers
docker/down:
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) down

## docker/up-fast: build and start the compose containers in detached mode without rebuilding
docker/up-fast:
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) down && \
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) up -d

## docker/logs: show the latest logs of the compose containers
docker/logs:
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_DIR) logs -f --tail=1000

# ==================================================================================== #
# PLUGINS
# ==================================================================================== #

# Plugin selection: make build/plugin PLUGIN=kotlin
PLUGIN ?= kotlin

## build/plugin: build a specific plugin (PLUGIN=kotlin|go|typescript|python|csharp|all)
build/plugin:
ifeq ($(PLUGIN),kotlin)
	cd plugin/kotlin && ./gradlew fatJar --no-daemon
else ifeq ($(PLUGIN),go)
	$(MAKE) -C plugin/go build
else ifeq ($(PLUGIN),typescript)
	cd plugin/typescript && npm ci && npm run build
else ifeq ($(PLUGIN),python)
	cd plugin/python && pip install -e ".[dev]" 2>/dev/null || true
else ifeq ($(PLUGIN),csharp)
	cd plugin/csharp && dotnet publish -c Release -o out
else ifeq ($(PLUGIN),all)
	$(MAKE) build/plugin PLUGIN=go
	$(MAKE) build/plugin PLUGIN=kotlin
	$(MAKE) build/plugin PLUGIN=typescript
	$(MAKE) build/plugin PLUGIN=python
	$(MAKE) build/plugin PLUGIN=csharp
else
	@echo "Unknown plugin: $(PLUGIN). Options: kotlin, go, typescript, python, csharp, all"
	@exit 1
endif

## build/kotlin-plugin: build the Kotlin plugin
build/kotlin-plugin:
	$(MAKE) build/plugin PLUGIN=kotlin

## build/go-plugin: build the Go plugin
build/go-plugin:
	$(MAKE) build/plugin PLUGIN=go

## build/typescript-plugin: build the TypeScript plugin
build/typescript-plugin:
	$(MAKE) build/plugin PLUGIN=typescript

## build/python-plugin: build the Python plugin
build/python-plugin:
	$(MAKE) build/plugin PLUGIN=python

## build/csharp-plugin: build the C# plugin
build/csharp-plugin:
	$(MAKE) build/plugin PLUGIN=csharp

## build/all-plugins: build all plugins
build/all-plugins:
	$(MAKE) build/plugin PLUGIN=all

## docker/plugin: build Docker image with specific plugin (PLUGIN=kotlin|go|typescript|python|csharp)
docker/plugin:
	docker build -f plugin/$(PLUGIN)/Dockerfile -t canopy-$(PLUGIN) .

## docker/run: run Docker container with specific plugin (PLUGIN=kotlin|go|typescript|python|csharp)
docker/run:
	docker run -v ~/.canopy:/root/.canopy canopy-$(PLUGIN)

## docker/run-kotlin: run Kotlin plugin container
docker/run-kotlin:
	docker run -v ~/.canopy:/root/.canopy canopy-kotlin

## docker/run-go: run Go plugin container
docker/run-go:
	docker run -v ~/.canopy:/root/.canopy canopy-go

## docker/run-typescript: run TypeScript plugin container
docker/run-typescript:
	docker run -v ~/.canopy:/root/.canopy canopy-typescript

## docker/run-python: run Python plugin container
docker/run-python:
	docker run -v ~/.canopy:/root/.canopy canopy-python

## docker/run-csharp: run C# plugin container
docker/run-csharp:
	docker run -v ~/.canopy:/root/.canopy canopy-csharp
