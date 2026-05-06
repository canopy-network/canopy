package main

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
)

// fetchSigner resolves a nickname OR raw hex address into the corresponding
// keystore record by first treating the input as a hex address, then falling
// back to nickname (the admin keystore accepts either form).
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
