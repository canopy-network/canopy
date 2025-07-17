package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// gasLimitDefault is the default gas limit for ethereum transactions
	gasLimitDefault = uint64(21000)
	// gasLimitWithData is the gas limit for ethereum transactions with data
	gasLimitWithData = uint64(100000)
)

// EthereumClient interface defines methods for interacting with ethereum blockchain
type EthereumClient interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	NetworkID(ctx context.Context) (*big.Int, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

// // SendTransaction sends an ethereum transaction, appending any data
// func SendTransaction(client EthereumClient, to common.Address, key string, value *big.Int, data []byte) error {
// 	// create context for ethereum client operations
// 	ctx := context.Background()
// 	// parse the private key from hex string
// 	privateKey, err := crypto.HexToECDSA(key)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse private key: %w", err)
// 	}
// 	// derive public key from private key
// 	publicKey := privateKey.Public()
// 	// cast public key to ecdsa public key
// 	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
// 	if !ok {
// 		return fmt.Errorf("failed to cast public key to ECDSA")
// 	}
// 	// get ethereum address from public key
// 	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
// 	// get pending nonce for the sender address
// 	nonce, err := client.PendingNonceAt(ctx, fromAddress)
// 	if err != nil {
// 		return fmt.Errorf("failed to get pending nonce: %w", err)
// 	}
// 	// get suggested gas price from network
// 	gasPrice, err := client.SuggestGasPrice(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to get suggested gas price: %w", err)
// 	}
// 	// calculate gas limit based on whether data is included
// 	gasLimit := gasLimitDefault
// 	if len(data) > 0 {
// 		// increase gas limit for contract interactions with data
// 		gasLimit = gasLimitDefault + uint64(len(data)*68)
// 	}
// 	// create new ethereum transaction
// 	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
// 	// get network chain id for transaction signing
// 	chainID, err := client.NetworkID(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to get network ID: %w", err)
// 	}
// 	// sign the transaction with private key and chain id
// 	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
// 	if err != nil {
// 		return fmt.Errorf("failed to sign transaction: %w", err)
// 	}
// 	// send the signed transaction to the network
// 	err = client.SendTransaction(ctx, signedTx)
// 	if err != nil {
// 		return fmt.Errorf("failed to send transaction: %w", err)
// 	}
// 	return nil
// }

// SendTransaction sends an ethereum transaction, optionally appending data
func SendTransaction(client EthereumClient, to common.Address, key string, value *big.Int, data []byte) error {
	// parse the private key from hex string
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	// get the public key from private key
	publicKey := privateKey.Public()
	// cast public key to ecdsa public key
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to cast public key to ecdsa")
	}
	// get the from address from public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	// get the nonce for the from address
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	// get the suggested gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}
	// determine gas limit based on whether data is present
	gasLimit := gasLimitDefault
	if len(data) > 0 {
		gasLimit = gasLimitWithData
	}
	// create the transaction
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)
	// get the chain id
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain id: %w", err)
	}
	// sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	// send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}
	return nil
}
