package main

import (
	"testing"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/canoliqctl/internal"
)

// TestMsgTypeURLMatchesPluginRegistration pins internal.MsgTypeURL against the
// plugin's own SupportedTransactions / TransactionTypeUrls registration.
//
// The CLI map is a hand-maintained duplicate of CanoliqConfig and nothing
// enforced the sync before this test. Two ways it can go wrong, both quiet:
//
//   - A message registered by the plugin but missing from the CLI map fails at
//     runtime with "unknown canoliq message type", long after the build passed.
//   - A short name that differs between the two is worse than a routing bug:
//     the short name is part of the signed bytes, so the mismatch surfaces as a
//     signature verification failure, which reads like a key problem.
//
// "send" is excluded: it is inherited from the tutorial contract plugin and is
// not a canoLiq message the CLI submits.
func TestMsgTypeURLMatchesPluginRegistration(t *testing.T) {
	cfg := canoliq.CanoliqConfig
	if len(cfg.SupportedTransactions) != len(cfg.TransactionTypeUrls) {
		t.Fatalf("plugin registration is itself inconsistent: %d names vs %d type urls",
			len(cfg.SupportedTransactions), len(cfg.TransactionTypeUrls))
	}
	want := make(map[string]string, len(cfg.SupportedTransactions))
	for i, name := range cfg.SupportedTransactions {
		if name == "send" {
			continue
		}
		want[name] = cfg.TransactionTypeUrls[i]
	}
	for name, url := range want {
		got, ok := internal.MsgTypeURL[name]
		if !ok {
			t.Errorf("internal.MsgTypeURL is missing %q — the CLI cannot submit this message", name)
			continue
		}
		if got != url {
			t.Errorf("internal.MsgTypeURL[%q] = %q, plugin registers %q", name, got, url)
		}
	}
	for name := range internal.MsgTypeURL {
		if _, ok := want[name]; !ok {
			t.Errorf("internal.MsgTypeURL has %q, which the plugin does not register", name)
		}
	}
}
