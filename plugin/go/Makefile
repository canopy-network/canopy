.PHONY: build test test-rpc

build:
	go build -o go-plugin .

# Run the transaction integration tests (requires a running Canopy node with the go plugin)
test:
	cd tutorial && go test -v -run TestPluginTransactions -timeout 120s

# Run ONLY the custom RPC endpoints test (/v1/query/faucets, /v1/query/rewards)
# Requires a running Canopy node with the go plugin and its RPC server on port 50010
test-rpc:
	cd tutorial && go test -v -run TestPluginCustomRPCEndpoints -timeout 600s
