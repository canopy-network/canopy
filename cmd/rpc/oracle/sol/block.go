package sol

import "github.com/canopy-network/canopy/cmd/rpc/oracle/types"

var _ types.BlockI = &Block{}

// Block wraps a Solana block (identified by slot) and implements types.BlockI.
type Block struct {
	slot         uint64
	hash         string // blockhash
	parentHash   string // previousBlockhash
	transactions []*Transaction
}

// newBlock creates a Block from decoded fields.
func newBlock(slot uint64, hash, parentHash string, txs []*Transaction) *Block {
	return &Block{slot: slot, hash: hash, parentHash: parentHash, transactions: txs}
}

// Hash returns the block hash (blockhash).
func (b *Block) Hash() string { return b.hash }

// ParentHash returns the previous block hash.
func (b *Block) ParentHash() string { return b.parentHash }

// Number returns the slot number.
func (b *Block) Number() uint64 { return b.slot }

// Transactions returns the block's transactions as interface values.
func (b *Block) Transactions() []types.TransactionI {
	txs := make([]types.TransactionI, len(b.transactions))
	for i, tx := range b.transactions {
		txs[i] = tx
	}
	return txs
}
