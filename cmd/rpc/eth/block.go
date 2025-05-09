// Only produce code. No descriptions, no backticks
// Do not create lines that end in whitespace
// Package is "eth"
// Follow these instructions for comments:
// - Do not use inline comments
// - Add a comment for every function
// - Add a comment for most lines of code
// - Do not comment the import statements
// - Follow go guidelines for commenting

// Create a type that can encapsulate an ethereum block.
// It must conform to this interface:
// type BlockI interface {
//  Hash() string
//  Number() uint64
//  Transactions() []wstypes.TransactionI
// }
// This type:
// - Calls an existing function NewTransaction(tx) when returning transactions
// - Is called Block and has a constructor called NewBlock

// Use these imports
// "github.com/ethereum/go-ethereum/core/types"
// wstypes "github.com/canopy-network/canopy/cmd/rpc/types"

// START
package eth

import (
	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
	"github.com/ethereum/go-ethereum/core/types"
)

// Block represents an Ethereum block that conforms to the BlockI interface
type Block struct {
	block *types.Block
}

// NewBlock creates a new Block instance from an Ethereum block
func NewBlock(block *types.Block) *Block {
	return &Block{
		block: block,
	}
}

// Hash returns the block hash as a string
func (b *Block) Hash() string {
	// Convert the hash to string representation
	return b.block.Hash().String()
}

// Number returns the block number
func (b *Block) Number() uint64 {
	// Return the block number as uint64
	return b.block.NumberU64()
}

// Transactions returns all transactions in the block as a slice of TransactionI
func (b *Block) Transactions() []wstypes.TransactionI {
	// Get the underlying ethereum transactions
	ethTxs := b.block.Transactions()

	// Create a slice to hold our wrapped transactions
	txs := make([]wstypes.TransactionI, len(ethTxs))

	// Convert each Ethereum transaction to our transaction type
	for i, tx := range ethTxs {
		txs[i] = NewTransaction(tx)
	}

	// Return the slice of transactions
	return txs
}
