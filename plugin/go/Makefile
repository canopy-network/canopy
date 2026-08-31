.PHONY: build test

build:
	go build -o go-plugin .

# Run the integration tests (requires a running Canopy node with the go plugin).
# This runs the transaction tests AND the custom RPC endpoints test
# (/v1/query/faucets, /v1/query/rewards) — the latter needs the plugin's RPC server on port 50010.
test:
	cd tutorial && go test -v -run 'TestPluginTransactions|TestPluginCustomRPCEndpoints' -timeout 600s
