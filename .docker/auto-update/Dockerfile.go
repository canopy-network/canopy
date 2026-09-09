# Pin the builder to the native build platform (e.g. amd64 on CI) so the
# Node/vite frontend build never runs under QEMU emulation. Emulating the
# rollup/esbuild native code for a foreign arch (linux/arm64) crashes with
# "qemu: uncaught target signal 4 (Illegal instruction)". The frontend output
# is architecture-independent and the Go binaries are cross-compiled below.
FROM --platform=$BUILDPLATFORM node:24-alpine AS builder

# Populated automatically by buildx for the requested target platform.
ARG TARGETOS
ARG TARGETARCH

ARG BRANCH='latest'
ARG BIN_PATH=/bin/cli

# Install build dependencies
RUN apk add --no-cache git ca-certificates alpine-sdk

WORKDIR /go/src/github.com/canopy-network/canopy

# Clone repository
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

# Copy golang
COPY --from=golang:1.26-alpine /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go version

# Install build tools
RUN apk update && apk add --no-cache make bash nodejs npm

# Build wallet and explorer
RUN make build/wallet
RUN make build/explorer

# Build auto-update coordinator (cross-compiled for the target arch)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin ./cmd/auto-update/.

# Build CLI (cross-compiled for the target arch)
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o "${BIN_PATH}" ./cmd/main/...

# =============================================================================
# Final image for Go plugin
# =============================================================================
FROM alpine:3.23
WORKDIR /app
ARG BIN_PATH=/bin/cli

# Install runtime dependencies
# - bash: required for pluginctl.sh scripts
# - procps: provides pkill for plugin process cleanup
# - ca-certificates: for HTTPS requests to GitHub API
# - pigz: for fast tarball extraction
RUN apk add --no-cache bash procps ca-certificates pigz

# Copy auto-update coordinator binary
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./canopy

# Copy CLI binary
COPY --from=builder ${BIN_PATH} ${BIN_PATH}

# Create plugin directory and copy only pluginctl.sh
# Plugin binary will be downloaded from upstream release and extracted on first start
RUN mkdir -p /app/plugin/go
COPY --from=builder /go/src/github.com/canopy-network/canopy/plugin/go/pluginctl.sh /app/plugin/go/pluginctl.sh

# Copy entrypoint
COPY entrypoint.sh /app/entrypoint.sh

# Set permissions
RUN chmod +x ${BIN_PATH} && \
    chmod +x /app/canopy && \
    chmod +x /app/entrypoint.sh && \
    chmod +x /app/plugin/go/pluginctl.sh

# Create plugin temp directory
RUN mkdir -p /tmp/plugin

ENTRYPOINT ["/app/entrypoint.sh"]
