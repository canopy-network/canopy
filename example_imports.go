// This file imports common types so the LLM knows which imports to use
package main

import (
	"fmt"

	"github.com/canopy-network/canopy/lib"
)

func main() {
	lockOrder := lib.LockOrder{}
	closeOrder := lib.CloseOrder{}
	fmt.Println(lockOrder, closeOrder)

	block := lib.Block{}

	// Get all transactions in this block
	// Returns [][]byte
	txs := block.GetTransactions()
	fmt.Println(txs)
}
