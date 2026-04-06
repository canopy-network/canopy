package eth

import (
	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var _ types.BlockI = &Block{} // Ensures *Block implements BlockI

// Block represents an ethereum block that implements BlockI interface
type Block struct {
	hash         string         // block hash as hex string
	parentHash   string         // parent hash
	number       uint64         // block number
	transactions []*Transaction // array of transactions in this block
}

// NewBlock creates a new Block from an ethereum block
func NewBlock(ethBlock *ethtypes.Block) (*Block, error) {
	// validate input block is not nil
	if ethBlock == nil {
		return nil, lib.ErrNilBlock()
	}
	// create new block instance
	block := &Block{
		hash:         ethBlock.Hash().Hex(),       // convert block hash to hex string
		parentHash:   ethBlock.ParentHash().Hex(), // convert parent block hash to hex string
		number:       ethBlock.NumberU64(),        // get block number as uint64
		transactions: make([]*Transaction, 0),     // initialize empty transaction slice
	}
	return block, nil // return successfully created block
}

// Hash returns the block hash as a string
func (b *Block) Hash() string {
	return b.hash // return the stored block hash
}

// ParentHash returns the parent block hash as a string
func (b *Block) ParentHash() string {
	return b.parentHash // return the parent hash
}

// Number returns the block number
func (b *Block) Number() uint64 {
	return b.number // return the stored block number
}

// Transactions returns all transactions in this block
func (b *Block) Transactions() []types.TransactionI {
	// create slice to hold transaction interfaces
	txs := make([]types.TransactionI, len(b.transactions))
	// convert each transaction to interface type
	for i, tx := range b.transactions {
		txs[i] = tx // assign transaction to interface slice
	}
	return txs // return slice of transaction interfaces
}
