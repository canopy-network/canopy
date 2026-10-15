// canoliqctl is a small CLI for building, signing, and submitting canoLiq
// plugin transactions to a running Canopy node. Remaining rollout work is
// tracked in docs/plans/canoliq-release-plan.md.
//
// Each subcommand fetches the signer's BLS key from the node's admin keystore,
// constructs the canoliq message proto, signs it with deterministic bytes that
// match the FSM's verifier, and POSTs the envelope to /v1/tx.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// globalFlags are accepted by every subcommand and have sane localnet defaults.
type globalFlags struct {
	rpcURL    string // node query RPC (height, /v1/tx)
	adminURL  string // node admin RPC (keystore-get)
	networkID uint64
	chainID   uint64
	fee       uint64
	password  string
	noWait    bool          // skip confirmation; print the hash and exit
	waitFor   time.Duration // how long to wait for a terminal outcome
}

// commands lists every subcommand alongside its handler. Adding a new command
// is one entry plus a new file (deposit.go is the canonical template). Usage
// strings live in a separate map (commandUsages) so handler files can
// reference them without creating an initialization cycle through `commands`.
var commands = map[string]func([]string, globalFlags) error{
	"deposit":            cmdDeposit,
	"redeem":             cmdRedeem,
	"claim":              cmdClaim,
	"canoliq-transfer":   cmdCanoliqTransfer,
	"cplq-transfer":      cmdCPLQTransfer,
	"cplq-claim-vested":  cmdCPLQClaimVested,
	"cplq-stake":         cmdCPLQStake,
	"cplq-unstake":       cmdCPLQUnstake,
	"cplq-claim-unstake": cmdCPLQClaimUnstake,
	"vote":               cmdVote,
	"buyback-execute":    cmdBuybackExecute,
	"spend-execute":      cmdSpendExecute,
	"multisig-approve":   cmdMultisigApprove,
	"proposal-create":    cmdProposalCreate,
	"otc-lock":           cmdOTCLockCreate,
	"otc-lock-claim":     cmdOTCLockClaim,
	"otc-lock-cancel":    cmdOTCLockCancel,
}

var commandUsages = map[string]string{
	"deposit":            "deposit <address> <amount-uCNPY>",
	"redeem":             "redeem <address> <ccnpy-amount>",
	"claim":              "claim <address> <redemption-id>",
	"canoliq-transfer":   "canoliq-transfer <from-address> <to-address-hex> <amount-uCCNPY>",
	"cplq-transfer":      "cplq-transfer <from-address> <to-address-hex> <amount-uCPLQ>",
	"cplq-claim-vested":  "cplq-claim-vested <address>",
	"cplq-stake":         "cplq-stake <address> <amount-uCPLQ> [--lock none|3m|6m|12m|24m]",
	"cplq-unstake":       "cplq-unstake <address> <amount-uCPLQ>",
	"cplq-claim-unstake": "cplq-claim-unstake <address> <unstake-id>",
	"vote":               "vote <address> <proposal-id> <yes|no|abstain>",
	"buyback-execute":    "buyback-execute <address> <proposal-id>",
	"spend-execute":      "spend-execute <address> <proposal-id>",
	"multisig-approve":   "multisig-approve <signer-address> <spend-id>",
	"proposal-create":    "proposal-create <param-change|buyback|treasury-spend|validator-eject|emergency|otc-program-fund> <args> [--description …]",
	"otc-lock":           "otc-lock <address> <ccnpy-amount> <90d|120d>",
	"otc-lock-claim":     "otc-lock-claim <address> <lock-id>",
	"otc-lock-cancel":    "otc-lock-cancel <address> <lock-id>   (forfeits the CPLQ reward)",
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
	// The chain id a transaction is SIGNED for is the chain id of the node it is
	// submitted to — fsm/transaction.go::CheckReplay rejects any mismatch with
	// ErrWrongChainId before the plugin is ever consulted. It is NOT the canoLiq
	// committee id from the plugin config, which only ever scopes fee-pool keys
	// and committee membership; the two are designed to differ (testnet pairs
	// node chain 1 with committee 42, mainnet with committee 19). Defaulting
	// this to the committee id made every command fail silently: the RPC runs
	// only CheckBasic, so it accepts the tx and returns a hash, then the mempool
	// re-check drops it with nothing printed.
	fs.Uint64Var(&gf.chainID, "chain-id", uint64(envDefaultInt("CANOLIQCTL_CHAIN_ID", 1)), "chain id of the NODE being submitted to (not the canoLiq committee id)")
	fs.Uint64Var(&gf.fee, "fee", uint64(envDefaultInt("CANOLIQCTL_FEE", 10_000)), "tx fee (uCNPY)")
	fs.StringVar(&gf.password, "password", os.Getenv("CANOLIQCTL_PASSWORD"), "keystore password (or set CANOLIQCTL_PASSWORD)")
	// Submitting is not the same as succeeding: /v1/tx admits a transaction on
	// CheckBasic alone, so it returns a hash for transactions the next mempool
	// re-check will reject and drop. Every command therefore waits for a
	// terminal outcome by default and exits non-zero if the transaction failed.
	// --no-wait restores the old fire-and-forget behaviour for scripts that do
	// their own tracking.
	fs.BoolVar(&gf.noWait, "no-wait", envDefaultBool("CANOLIQCTL_NO_WAIT", false), "print the tx hash and exit without confirming the outcome")
	fs.DurationVar(&gf.waitFor, "wait-timeout", 45*time.Second, "how long to wait for a submitted tx to be included or rejected")
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
	fmt.Fprintln(os.Stderr, "  --rpc-url, --admin-url, --network-id, --chain-id, --fee, --password, --no-wait, --wait-timeout")
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDefaultBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
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
