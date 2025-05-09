// Only produce code. No descriptions, no backticks
// Do not create lines that end in whitespace
// Package is "eth"
// Follow these instructions for comments:
// - Do not use inline comments
// - Add a comment for every function
// - Add a comment for most lines of code
// - Do not comment the import statements
// - Follow go guidelines for commenting

// Create a type that can encapsulate an ethereum transaction.
// This type:
// - Is called Transaction and has a constructor called NewTransaction
// - Has a method to get the From address for a transaction. It should not have a return error. On error it should return an empty string
// - Has a method to determine if this transaction is a self-send, where the from and to addresses are the same
// - Has a method to get the data from the data field

// Use these imports
// "github.com/ethereum/go-ethereum/core/types"

// Extra information:
// - To get the from address for a transaction:
//
//	if from, err := types.Sender(types.LatestSignerForChainID(big.NewInt(1)), tx); err == nil {
//	    fmt.Println(from.Hex()) // 0x0fD081e3Bb178dc45c0cb23202069ddA57064258
//	}
//
// START
package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// Transaction encapsulates an ethereum transaction
type Transaction struct {
	tx *types.Transaction
}

// NewTransaction creates a new Transaction from a types.Transaction
func NewTransaction(tx *types.Transaction) *Transaction {
	return &Transaction{
		tx: tx,
	}
}

// From returns the sender address for the transaction
// If the sender cannot be determined, an empty string is returned
func (t *Transaction) From() string {
	// Use the latest signer for chain ID 1 (Ethereum mainnet)
	signer := types.LatestSignerForChainID(big.NewInt(1))

	// Attempt to extract the sender address from the transaction
	from, err := types.Sender(signer, t.tx)

	// If there was an error determining the sender, return empty string
	if err != nil {
		return ""
	}

	// Return the sender address as a hex string
	return from.Hex()
}

// IsSelfSend determines if the transaction is sent from an address to itself
func (t *Transaction) IsSelfSend() bool {
	// Get the from address
	from := t.From()

	// If from is empty (error occurred), return false
	if from == "" {
		return false
	}

	// Get the to address
	var to string

	// Check if the to address is nil (contract creation)
	if t.tx.To() == nil {
		to = ""
	} else {
		// Convert the to address to hex string
		to = t.tx.To().Hex()
	}

	// Return true if from and to are the same and not empty
	return from != "" && to != "" && from == to
}

// To returns the recipient address of the transaction
// For contract creation transactions, returns an empty string
func (t *Transaction) To() string {
	// Check if the to address is nil (contract creation)
	if t.tx.To() == nil {
		return ""
	}

	// Return the recipient address as a hex string
	return t.tx.To().Hex()
}

// Data returns the data payload of the transaction
func (t *Transaction) Data() []byte {
	// Return a copy of the data to prevent modification of the original
	return t.tx.Data()
}

// Hash returns the transaction hash
func (t *Transaction) Hash() string {
	// Return the transaction hash as a hex string
	return t.tx.Hash().Hex()
}
