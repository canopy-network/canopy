// NEW — cmd/rpc/oracle/sol/error.go
package sol

import "errors"

var (
	// ErrSlotSkipped indicates getBlock reported the slot was skipped (no block produced).
	// Solana returns this via jsonrpc codes -32007 (skipped/snapshot) and -32009 (missing in long-term storage).
	ErrSlotSkipped = errors.New("solana slot was skipped")
	// ErrNilTransaction indicates a nil transaction was passed to NewTransaction.
	ErrNilTransaction = errors.New("transaction is nil")
	// ErrMintAccountTooSmall indicates a Mint account's data was shorter than the decimals offset.
	ErrMintAccountTooSmall = errors.New("mint account data too small to decode decimals")
	// ErrSourceSlot indicates the observed finalized slot went backwards below nextSlot.
	ErrSourceSlot = errors.New("solana finalized slot lower than expected")
)
