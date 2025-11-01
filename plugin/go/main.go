package main

import (
	"context"
	"github.com/canopy-network/go-plugin/contract"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// start the plugin
	contract.StartPlugin(contract.DefaultConfig())
	// create a cancellable context that listens for kill signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
