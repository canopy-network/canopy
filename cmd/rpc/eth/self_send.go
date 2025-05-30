package eth

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SelfSend self sends an ethereum transaction the data is input data
func SelfSend(client *ethclient.Client, address string, key string, data []byte) error {
	// parse the private key from hex string
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	// derive the public key from private key
	publicKey := privateKey.Public()
	// cast public key to ecdsa public key
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("failed to cast public key to ecdsa")
	}
	// get the from address from public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	// parse the to address from string
	toAddress := common.HexToAddress(address)
	// get the nonce for the from address
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	// set the value to zero for self send
	value := big.NewInt(0)
	// get the gas limit estimate
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From: fromAddress,
		To:   &toAddress,
		Data: data,
	})
	if err != nil {
		return fmt.Errorf("failed to estimate gas: %w", err)
	}
	// get the gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}
	// create the transaction
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)
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
