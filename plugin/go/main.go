package main

import (
	"context"
	"github.com/canopy-network/go-plugin/contract"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// start the plugin and capture the running instance
	plugin := contract.StartPlugin(contract.DefaultConfig())
	// start the plugin's own HTTP server exposing custom, chain-specific RPC endpoints
	go plugin.StartRPCServer()
	// create a cancellable context that listens for kill signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
