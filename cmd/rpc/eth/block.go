package eth

import (
	"github.com/ethereum/go-ethereum/core/types"
	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
)

// Block represents an ethereum block that implements BlockI interface
type Block struct {
	ethBlock *types.Block // underlying ethereum block
}

// NewBlock creates a new Block from an ethereum block
func NewBlock(ethBlock *types.Block) *Block {
	// create new block wrapper
	return &Block{
		ethBlock: ethBlock, // store the ethereum block
	}
}

// Hash returns the block hash as a string
func (b *Block) Hash() string {
	// get the block hash and convert to string
	return b.ethBlock.Hash().Hex()
}

// Number returns the block number
func (b *Block) Number() uint64 {
	// get the block number
	return b.ethBlock.NumberU64()
}

// Transactions returns all transactions in the block
func (b *Block) Transactions() []wstypes.TransactionI {
	// get ethereum transactions from block
	ethTxs := b.ethBlock.Transactions()
	// create slice to hold wrapped transactions
	transactions := make([]wstypes.TransactionI, 0, len(ethTxs))
	// iterate through ethereum transactions
	for _, ethTx := range ethTxs {
		// create new transaction wrapper
		tx, err := NewTransaction(ethTx)
		// check if transaction creation failed
		if err != nil {
			// skip this transaction if error occurred
			continue
		}
		// add transaction to slice
		transactions = append(transactions, tx)
	}
	// return all wrapped transactions
	return transactions
}
