//go:build exclude
// +build exclude

// Only produce code. No descriptions, no backticks
// Do not create lines that end in whitespace
// Package is "rpc"
// Follow these instructions for comments:
// - Do not use inline comments
// - Add a comment for every function
// - Add a comment for most lines of code
// - Do not comment the import statements
// - Follow go guidelines for commenting
// You will be creating multiple types:

// Create a type that uses go-ethereum to generate a websocket client that:
// - Connects to an ethereum node
// - Listens for new block headers
// - When receiving a new block header, fetches the block
// - Uses the created iterator to iterate over transactions in the block
// - Uses the created persistor to write the transactions to disk

// Create a type that iterates over transactions and does the following:
// - Identifies transactions that have the same to and from address - self-sent transactions
// - Of these transactions, verify there is valid json in the data field
// - Extracts this JSON

// Use this interface to write transactions to disk:
// type TransactionStorer interface {
//  WriteTx(blockHeight uint64, txID string, data string) error
// }

// Use these imports
// "github.com/ethereum/go-ethereum/core/types"
// "github.com/ethereum/go-ethereum/ethclient"
// "github.com/ethereum/go-ethereum/crypto"

// Extra information:
// - To get the from address for a transaction:
//
//	if from, err := types.Sender(types.LatestSignerForChainID(1), tx); err == nil {
//	    fmt.Println(from.Hex()) // 0x0fD081e3Bb178dc45c0cb23202069ddA57064258
//	}
//
// START
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransactionStorer interface for writing transactions to disk
type TransactionStorer interface {
	// WriteTx writes a transaction with block height, transaction ID, and data
	WriteTx(blockHeight uint64, txID string, data string) error
}

// EthereumClient represents a client connected to an Ethereum node via WebSocket
type EthereumClient struct {
	client    *ethclient.Client
	persistor TransactionStorer
}

// NewEthereumClient creates a new EthereumClient
func NewEthereumClient(rpcURL string, persistor TransactionStorer) (*EthereumClient, error) {
	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return &EthereumClient{client: client, persistor: persistor}, nil
}

// ListenForNewBlocks listens for new block headers and processes them
func (ec *EthereumClient) ListenForNewBlocks(ctx context.Context) {
	headers := make(chan *types.Header)
	sub, err := ec.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			ec.ProcessBlock(header.Number.Uint64())
		}
	}
}

// ProcessBlock retrieves a block using the block number and iterates over its transactions
func (ec *EthereumClient) ProcessBlock(blockNumber uint64) {
	block, err := ec.client.BlockByNumber(context.Background(), big.NewInt(int64(blockNumber)))
	if err != nil {
		log.Fatalf("failed to retrieve block: %v", err)
	}

	txIterator := NewTransactionIterator()
	for _, tx := range block.Transactions() {
		txIterator.ProcessTransaction(block.NumberU64(), tx, ec.persistor)
	}
}

// TransactionIterator iterates over transactions and extracts self-sent transactions
type TransactionIterator struct{}

// NewTransactionIterator creates a new TransactionIterator
func NewTransactionIterator() *TransactionIterator {
	return &TransactionIterator{}
}

// ProcessTransaction checks transactions for self-sent transactions with valid JSON
func (ti *TransactionIterator) ProcessTransaction(blockHeight uint64, tx *types.Transaction, persistor TransactionStorer) {
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

	if from.Hex() == tx.To().Hex() {
		data := tx.Data()
		if json.Valid(data) {
			if err := persistor.WriteTx(blockHeight, tx.Hash().Hex(), string(data)); err != nil {
				log.Println("error writing transaction:", err)
			}
		}
	}
}

// FileTransactionStore writes transactions to a file
type FileTransactionStore struct {
	file *os.File
}

// NewFileTransactionStore creates a new FileTransactionStore
func NewFileTransactionStore(filePath string) (*FileTransactionStore, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileTransactionStore{file: file}, nil
}

// WriteTx writes a transaction to the file
func (fts *FileTransactionStore) WriteTx(blockHeight uint64, txID string, data string) error {
	txData := fmt.Sprintf("Block: %d, TxID: %s, Data: %s\n", blockHeight, txID, data)
	if _, err := fts.file.WriteString(txData); err != nil {
		return err
	}
	return nil
}
