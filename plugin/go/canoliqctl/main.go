// canoliqctl is a small CLI for building, signing, and submitting canoLiq
// plugin transactions to a running Canopy node. It targets the Phase 1.5
// verification matrix in docs/plans/canoliq-implementation-plan.md.
//
// Each subcommand fetches the signer's BLS key from the node's admin keystore,
// constructs the canoliq message proto, signs it with deterministic bytes that
// match the FSM's verifier, and POSTs the envelope to /v1/tx.
package main

import (
	"flag"
	"fmt"
	"os"
)

// globalFlags are accepted by every subcommand and have sane localnet defaults.
type globalFlags struct {
	rpcURL    string // node query RPC (height, /v1/tx)
	adminURL  string // node admin RPC (keystore-get)
	networkID uint64
	chainID   uint64
	fee       uint64
	password  string
}

// commands lists every subcommand alongside its handler. Adding a new command
// is one entry plus a new file (deposit.go is the canonical template). Usage
// strings live in a separate map (commandUsages) so handler files can
// reference them without creating an initialization cycle through `commands`.
var commands = map[string]func([]string, globalFlags) error{
	"deposit":            cmdDeposit,
	"redeem":             cmdRedeem,
	"claim":              cmdClaim,
	"cliq-transfer":      cmdCLIQTransfer,
	"cliq-claim-vested":  cmdCLIQClaimVested,
	"cliq-stake":         cmdCLIQStake,
	"cliq-unstake":       cmdCLIQUnstake,
	"cliq-claim-unstake": cmdCLIQClaimUnstake,
	"vote":               cmdVote,
	"buyback-execute":    cmdBuybackExecute,
	"spend-execute":      cmdSpendExecute,
	"multisig-approve":   cmdMultisigApprove,
	"proposal-create":    cmdProposalCreate,
}

var commandUsages = map[string]string{
	"deposit":            "deposit <nickname> <amount-uCNPY>",
	"redeem":             "redeem <nickname> <ccnpy-amount>",
	"claim":              "claim <nickname> <redemption-id>",
	"cliq-transfer":      "cliq-transfer <from-nickname> <to-address-hex> <amount-uCLIQ>",
	"cliq-claim-vested":  "cliq-claim-vested <nickname>",
	"cliq-stake":         "cliq-stake <nickname> <amount-uCLIQ>",
	"cliq-unstake":       "cliq-unstake <nickname> <amount-uCLIQ>",
	"cliq-claim-unstake": "cliq-claim-unstake <nickname> <unstake-id>",
	"vote":               "vote <nickname> <proposal-id> <yes|no|abstain>",
	"buyback-execute":    "buyback-execute <nickname> <proposal-id>",
	"spend-execute":      "spend-execute <nickname> <proposal-id>",
	"multisig-approve":   "multisig-approve <signer-nickname> <spend-id>",
	"proposal-create":    "proposal-create <param-change|buyback|treasury-spend|validator-eject|emergency> <args> [--description …]",
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	name := os.Args[1]
	run, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", name)
		printUsage()
		os.Exit(2)
	}

	fs := flag.NewFlagSet(name, flag.ExitOnError)
	gf := globalFlags{}
	fs.StringVar(&gf.rpcURL, "rpc-url", envDefault("CANOLIQCTL_RPC_URL", "http://localhost:50002"), "node query RPC URL")
	fs.StringVar(&gf.adminURL, "admin-url", envDefault("CANOLIQCTL_ADMIN_URL", "http://localhost:50003"), "node admin RPC URL (keystore)")
	fs.Uint64Var(&gf.networkID, "network-id", uint64(envDefaultInt("CANOLIQCTL_NETWORK_ID", 1)), "Canopy network id")
	fs.Uint64Var(&gf.chainID, "chain-id", uint64(envDefaultInt("CANOLIQCTL_CHAIN_ID", 2)), "canoLiq committee chain id")
	fs.Uint64Var(&gf.fee, "fee", uint64(envDefaultInt("CANOLIQCTL_FEE", 10_000)), "tx fee (uCNPY)")
	fs.StringVar(&gf.password, "password", os.Getenv("CANOLIQCTL_PASSWORD"), "keystore password (or set CANOLIQCTL_PASSWORD)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	if gf.password == "" {
		fmt.Fprintln(os.Stderr, "error: --password (or CANOLIQCTL_PASSWORD) is required")
		os.Exit(2)
	}
	if err := run(fs.Args(), gf); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: canoliqctl <command> [flags] <args>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "commands:")
	for name := range commands {
		fmt.Fprintf(os.Stderr, "  %-20s %s\n", name, commandUsages[name])
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "global flags (all commands):")
	fmt.Fprintln(os.Stderr, "  --rpc-url, --admin-url, --network-id, --chain-id, --fee, --password")
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}
