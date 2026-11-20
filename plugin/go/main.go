package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/contract"
)

// CANOPY_PLUGIN_MODE selects which plugin implementation runs in this binary.
// "" or "contract" → the original send-tutorial plugin (default).
// "canoliq"        → the canoLiq liquid-staking plugin.
const pluginModeEnv = "CANOPY_PLUGIN_MODE"

// CANOLIQ_RPC_ADDR overrides the read-only HTTP query listener address,
// taking precedence over Config.RpcAddress from CANOLIQ_CONFIG. Empty/unset
// keeps whatever the JSON config provides (default: disabled).
const canoliqRpcAddrEnv = "CANOLIQ_RPC_ADDR"

// CANOLIQ_ALERT_URL turns on (or overrides the target of) the push-alert
// webhook, taking precedence over Config.Alerts.WebhookURL. Empty/unset keeps
// whatever the JSON config provides (default: disabled).
const canoliqAlertURLEnv = "CANOLIQ_ALERT_URL"

func main() {
	mode := os.Getenv(pluginModeEnv)
	var canoliqPlugin *canoliq.Plugin
	switch mode {
	case "", "contract":
		log.Println("starting plugin in 'contract' mode (send tutorial)")
		// start the plugin and capture the running instance
		plugin := contract.StartPlugin(contract.DefaultConfig())
		// start the plugin's own HTTP server exposing custom, chain-specific RPC endpoints
		go plugin.StartRPCServer()
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
		if addr := os.Getenv(canoliqRpcAddrEnv); addr != "" {
			cfg.RpcAddress = addr
		}
		if url := os.Getenv(canoliqAlertURLEnv); url != "" {
			if cfg.Alerts == nil {
				cfg.Alerts = &canoliq.AlertConfig{}
			}
			cfg.Alerts.WebhookURL = url
		}
		canoliqPlugin = canoliq.StartPlugin(cfg)
	default:
		log.Fatalf("unknown %s value %q (want '', 'contract', or 'canoliq')", pluginModeEnv, mode)
	}
	// create a cancellable context that listens for kill signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	if canoliqPlugin != nil {
		if rpc := canoliqPlugin.RPC(); rpc != nil {
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := rpc.Shutdown(shutCtx); err != nil {
				log.Printf("canoliq: rpc shutdown error: %v", err)
			}
		}
	}
}
