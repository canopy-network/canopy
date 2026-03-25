FROM node:24-alpine AS builder 

ARG BRANCH='latest'
ARG CLI_DIR
ARG BUILD_PATH=cmd/cli
ARG GO_BIN_DIR
ARG BUILD_LOCAL=False
ARG BIN_PATH=/bin/cli

# downloads git and clones selected version
RUN apk add --no-cache git ca-certificates alpine-sdk
WORKDIR /go/src/github.com/canopy-network/canopy
RUN echo "Building from BRANCH=${BRANCH}" && \
    if [ "$BRANCH" = "latest" ]; then \
        echo "Fetching latest tag..."; \
        git clone https://github.com/canopy-network/canopy.git . && \
        LATEST_TAG=$(git describe --tags `git rev-list --tags --max-count=1`) && \
        echo "Checking out tag $LATEST_TAG" && \
        git checkout $LATEST_TAG; \
    else \
        echo "Cloning branch $BRANCH" && \
        git clone -b "$BRANCH" https://github.com/canopy-network/canopy.git .; \
    fi
# copy golang 
COPY --from=golang:1.24-alpine  /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go version
# Builds 
RUN apk update && apk add --no-cache make bash nodejs npm 
RUN make build/wallet
RUN make build/explorer

# Builds auto-update CLI
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bin ./cmd/auto-update/.

# Only build if the file at ${BIN_PATH} doesn't already exist
RUN if [ ! -f "${BIN_PATH}" ]; then \
    echo "File ${BIN_PATH} not found. Building it..."; \
    CGO_ENABLED=0 GOOS=linux go build -a -o "${BIN_PATH}" ./cmd/main/...; \
  else \
    echo "File ${BIN_PATH} already exists. Skipping build."; \
  fi

FROM alpine:3.23
WORKDIR /app
ARG BIN_PATH=/bin/cli

# Coying for command
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./canopy
# Copying cli for auto update
COPY --from=builder ${BIN_PATH} ${BIN_PATH}
# Copying entrypoint for persisting update cli
COPY entrypoint.sh /app/entrypoint.sh

RUN apk add --no-cache pigz ca-certificates

RUN chmod +x ${BIN_PATH}
RUN chmod +x /app/canopy
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
