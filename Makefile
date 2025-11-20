# Variables
GO_BIN_DIR := ~/go/bin
CLI_DIR := ./cmd/main/...
WALLET_DIR := ./cmd/rpc/web/wallet-new
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
	docker/down docker/build docker/up-fast docker/down docker/logs

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

## build/new-wallet: alias for build/wallet (for backward compatibility)
build/new-wallet: build/wallet

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
