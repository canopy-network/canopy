package eth

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransferERC20WithData calls and ERC20 contract but also appends data to the end of the ABI encoding
func TransferERC20WithData(client *ethclient.Client, contractAddress, receiveAddress string, key string, transferAmount int, data []byte) error {
	// validate the contract address
	if !common.IsHexAddress(contractAddress) {
		return &ERC20TransferError{Message: "invalid contract address format"}
	}

	// validate the receiver address
	if !common.IsHexAddress(receiveAddress) {
		return &ERC20TransferError{Message: "invalid receiver address format"}
	}

	// parse the private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(key, "0x"))
	if err != nil {
		return &ERC20TransferError{Message: "failed to parse private key", Err: err}
	}

	// get the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return &ERC20TransferError{Message: "error casting public key to ECDSA"}
	}

	// derive the sender address from the public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// get the nonce for the sender address
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return &ERC20TransferError{Message: "failed to get account nonce", Err: err}
	}

	// convert transfer amount to wei
	value := big.NewInt(0) // we're not sending ETH, just tokens
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return &ERC20TransferError{Message: "failed to suggest gas price", Err: err}
	}

	// create the token contract address
	tokenAddress := common.HexToAddress(contractAddress)
	// create the receiver address
	toAddress := common.HexToAddress(receiveAddress)

	// create the transfer function signature
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := crypto.Keccak256(transferFnSignature)
	methodID := hash[:4]

	// pad the address to 32 bytes
	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)

	// convert the transfer amount to big int and pad to 32 bytes
	amount := big.NewInt(int64(transferAmount))
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)

	// combine the method ID, padded address, padded amount, and additional data
	var input []byte
	input = append(input, methodID...)
	input = append(input, paddedAddress...)
	input = append(input, paddedAmount...)
	input = append(input, data...)

	// estimate gas limit
	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		From:     fromAddress,
		To:       &tokenAddress,
		GasPrice: gasPrice,
		Value:    value,
		Data:     input,
	})
	if err != nil {
		return &ERC20TransferError{Message: "failed to estimate gas limit", Err: err}
	}

	// increase gas limit by 20% to be safe
	gasLimit = gasLimit * 6 / 5

	// create the transaction
	tx := types.NewTransaction(nonce, tokenAddress, value, gasLimit, gasPrice, input)

	// get the chain ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return &ERC20TransferError{Message: "failed to get chain ID", Err: err}
	}

	// sign the transaction
	signedTx, err := types.SignTx(tx, types.NewLondonSigner(chainID), privateKey)
	if err != nil {
		return &ERC20TransferError{Message: "failed to sign transaction", Err: err}
	}

	// send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return &ERC20TransferError{Message: "failed to send transaction", Err: err}
	}

	return nil
}
