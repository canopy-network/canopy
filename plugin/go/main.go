package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/contract"
)

// CANOPY_PLUGIN_MODE selects which plugin implementation runs in this binary.
// "" or "contract" → the original send-tutorial plugin (default).
// "canoliq"        → the canoLiq liquid-staking plugin.
const pluginModeEnv = "CANOPY_PLUGIN_MODE"

func main() {
	mode := os.Getenv(pluginModeEnv)
	switch mode {
	case "", "contract":
		log.Println("starting plugin in 'contract' mode (send tutorial)")
		contract.StartPlugin(contract.DefaultConfig())
	case "canoliq":
		log.Println("starting plugin in 'canoliq' mode (liquid staking)")
		cfg := canoliq.DefaultConfig()
		if path := os.Getenv("CANOLIQ_CONFIG"); path != "" {
			loaded, err := canoliq.NewConfigFromFile(path)
			if err != nil {
				log.Fatalf("failed to load canoliq config %q: %v", path, err)
			}
			cfg = loaded
		}
		canoliq.StartPlugin(cfg)
	default:
		log.Fatalf("unknown %s value %q (want '', 'contract', or 'canoliq')", pluginModeEnv, mode)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
