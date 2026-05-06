FROM golang:1.25-alpine AS builder

RUN apk update && apk add --no-cache make bash nodejs npm

WORKDIR /go/src/github.com/canopy-network/canopy
COPY . /go/src/github.com/canopy-network/canopy


RUN make build/wallet
RUN make build/explorer
RUN go build -a -o bin ./cmd/main/...

# Build the canoLiq plugin process and the canoliqctl operator CLI from the
# sibling Go module under plugin/go. The plugin binary is launched by the
# canopy controller via plugin/go/pluginctl.sh; canoliqctl is a host-side tool
# that signs and submits canoLiq plugin transactions to /v1/tx.
WORKDIR /go/src/github.com/canopy-network/canopy/plugin/go
RUN go build -o /go-plugin .
RUN go build -o /canoliqctl ./canoliqctl/

FROM alpine:3.19
# bash for pluginctl.sh; procps for full GNU ps (busybox ps lacks the -p flag
# pluginctl.sh::is_running relies on, so without procps the launcher mistakes
# a healthy plugin for a dead one).
RUN apk add --no-cache bash procps
WORKDIR /app
COPY --from=builder /go/src/github.com/canopy-network/canopy/bin ./

# Install the plugin where controller.resolvePluginCtlPath looks for it
# (relative path "plugin/<name>/pluginctl.sh"). Config.Plugin = "go" causes
# the controller to exec /app/plugin/go/pluginctl.sh, which launches the
# binary inheriting CANOPY_PLUGIN_MODE from the canopy process env.
RUN mkdir -p /app/plugin/go/canoliq
COPY --from=builder /go-plugin /app/plugin/go/go-plugin
COPY plugin/go/pluginctl.sh /app/plugin/go/pluginctl.sh
COPY plugin/go/canoliq/genesis.json /app/plugin/go/canoliq/genesis.json
# Plugin config consumed by plugin/go/main.go when CANOLIQ_CONFIG is set in
# the canopy process env. Points the canoLiq plugin at the bundled genesis
# file so its one-shot Genesis() can mint the 100M CLIQ supply.
RUN printf '{"chainId":2,"dataDirPath":"/tmp/plugin","genesisPath":"/app/plugin/go/canoliq/genesis.json"}\n' \
    > /app/plugin/go/canoliq/canoliq-config.json
RUN chmod +x /app/plugin/go/pluginctl.sh /app/plugin/go/go-plugin

# Install canoliqctl on PATH so operators can run it inside the container or
# via `docker exec`. Hosts can also build it directly from plugin/go/canoliqctl.
COPY --from=builder /canoliqctl /usr/local/bin/canoliqctl

ENTRYPOINT ["/app/bin"]
