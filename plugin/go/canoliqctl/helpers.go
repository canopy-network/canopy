package main

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/proto"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
)

// fetchSigner resolves a signer for the given account. The parameter is named
// nickOrAddr because the intent was to accept either form, but a node's admin
// keystore endpoint hex-decodes whatever it is given, so a nickname comes back
// as `encoding/hex: invalid byte`. Pass the hex address until nickname
// resolution is actually implemented; every usage string says <address> for
// that reason.
//
// Resolves a hex address into the corresponding
// keystore record by first treating the input as a hex address, then falling
// key material via the admin keystore.
func fetchSigner(adminURL, nickOrAddr, password string) (*internal.Key, error) {
	k, err := internal.KeystoreGet(adminURL, nickOrAddr, password)
	if err != nil {
		return nil, fmt.Errorf("keystore lookup for %q: %w", nickOrAddr, err)
	}
	if k.PrivateKey == "" {
		return nil, fmt.Errorf("keystore returned no private key for %q (wrong password?)", nickOrAddr)
	}
	return k, nil
}

// addrFromHex decodes a hex address string. The plugin protos take addresses
// as raw bytes, so we accept the same hex encoding used everywhere else in
// Canopy (validator_key.json, RPC responses, etc.).
func addrFromHex(s string) ([]byte, error) {
	addr, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex address %q: %w", s, err)
	}
	return addr, nil
}

// parseUint converts an unsigned-integer arg, returning a friendly error.
func parseUint(s, label string) (uint64, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", label, s, err)
	}
	return n, nil
}

// txParams converts globalFlags into the param bundle the submit helper takes.
func txParams(gf globalFlags) internal.TxParams {
	return internal.TxParams{
		NetworkID: gf.networkID,
		ChainID:   gf.chainID,
		Fee:       gf.fee,
	}
}

// requireArgs fails fast if positional args don't match the command's needs.
func requireArgs(args []string, n int, usage string) error {
	if len(args) < n {
		return fmt.Errorf("expected %d positional arg(s), got %d (usage: %s)", n, len(args), usage)
	}
	return nil
}

// submitAndReport submits a plugin transaction and reports what actually
// happened to it.
//
// This exists because "submitted" was never the same as "succeeded": /v1/tx
// admits a transaction on CheckBasic alone, so it returns a hash for
// transactions the next mempool re-check will reject and drop. Printing that
// hash and exiting 0 told operators a treasury spend had executed when it had
// not. Every command now resolves a terminal outcome before returning, and a
// rejected transaction becomes a non-zero exit carrying the node's own reason.
//
// summary is the command-specific detail appended to the outcome line, e.g.
// "from=… ccnpy=… tier=90d".
func submitAndReport(gf globalFlags, signer *internal.Key, msgType string, msg proto.Message, label, summary string) error {
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, msgType, msg, txParams(gf))
	if err != nil {
		return err
	}
	if gf.noWait {
		fmt.Printf("%s submitted (unconfirmed): tx_hash=%s %s\n", label, hash, summary)
		return nil
	}
	outcome, err := internal.ConfirmTx(gf.rpcURL, signer.Address, hash, gf.waitFor)
	if err != nil {
		// Timed out without a terminal answer. The transaction is not known to
		// have failed, so say exactly that rather than implying either result.
		return fmt.Errorf("%s: %w", label, err)
	}
	if outcome.Failed {
		return fmt.Errorf("%s REJECTED: %s (tx_hash=%s %s)", label, outcome.Error(), hash, summary)
	}
	fmt.Printf("%s confirmed in block %d: tx_hash=%s %s\n", label, outcome.Height, hash, summary)
	return nil
}
