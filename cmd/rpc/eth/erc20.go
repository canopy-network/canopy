package eth

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	wstypes "github.com/canopy-network/canopy/cmd/rpc/types"
)

const (
	// erc20TransferMethodID is the method signature for transfer(address,uint256)
	erc20TransferMethodID = "a9059cbb"
)

// ERC20Contract represents an ERC20 token contract with metadata
type ERC20Contract struct {
	Symbol   string // token symbol
	Name     string // descriptive name
	Decimals int    // number of decimal places
}

// erc20Contracts contains popular ERC20 contract addresses and their metadata
var erc20Contracts = map[string]ERC20Contract{
	"0xdac17f958d2ee523a2206206994597c13d831ec7": { // USDT
		Symbol:   "USDT",
		Name:     "Tether USD",
		Decimals: 6,
	},
	"0xa0b86a33e6441e6c7d3e4081f7567f8b8e4c3c2e": { // USDC
		Symbol:   "USDC",
		Name:     "USD Coin",
		Decimals: 6,
	},
	"0x514910771af9ca656af840dff83e8264ecf986ca": { // LINK
		Symbol:   "LINK",
		Name:     "Chainlink Token",
		Decimals: 18,
	},
	"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984": { // UNI
		Symbol:   "UNI",
		Name:     "Uniswap",
		Decimals: 18,
	},
	"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": { // WBTC
		Symbol:   "WBTC",
		Name:     "Wrapped BTC",
		Decimals: 8,
	},
	"0x6b175474e89094c44da98b954eedeac495271d0f": { // DAI
		Symbol:   "DAI",
		Name:     "Dai Stablecoin",
		Decimals: 18,
	},
	"0x95ad61b0a150d79219dcf64e1e6cc01f0b64c4ce": { // SHIB
		Symbol:   "SHIB",
		Name:     "SHIBA INU",
		Decimals: 18,
	},
	"0x7d1afa7b718fb893db30a3abc0cfc608aacfebb0": { // MATIC
		Symbol:   "MATIC",
		Name:     "Polygon",
		Decimals: 18,
	},
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": { // WETH
		Symbol:   "WETH",
		Name:     "Wrapped Ether",
		Decimals: 18,
	},
	"0x4fabb145d64652a948d72533023f6e7a623c7c53": { // BUSD
		Symbol:   "BUSD",
		Name:     "Binance USD",
		Decimals: 18,
	},
}

// ParseERC20Transfer parses the transaction data looking for ERC20 transfers and any extra data
func ParseERC20Transfer(tx *types.Transaction) (wstypes.TokenTransfer, []byte, error) {
	// check if transaction has data
	if len(tx.Data()) < 4 {
		return wstypes.TokenTransfer{}, nil, ErrNotERC20Transfer
	}
	// extract method signature from first 4 bytes
	methodSig := hex.EncodeToString(tx.Data()[:4])
	// check if this is an ERC20 transfer method
	if methodSig != erc20TransferMethodID {
		return wstypes.TokenTransfer{}, nil, ErrNotERC20Transfer
	}
	// check if we have enough data for a complete transfer call
	if len(tx.Data()) < 68 {
		return wstypes.TokenTransfer{}, nil, ErrInvalidTransactionData
	}
	// extract recipient address from bytes 4-36
	recipientBytes := tx.Data()[16:36]
	recipientAddress := common.BytesToAddress(recipientBytes).Hex()
	// extract amount from bytes 36-68
	amountBytes := tx.Data()[36:68]
	amount := new(big.Int).SetBytes(amountBytes)
	// get contract address from transaction recipient
	contractAddress := tx.To().Hex()
	// look up contract metadata
	contract, exists := erc20Contracts[strings.ToLower(contractAddress)]
	if !exists {
		return wstypes.TokenTransfer{}, nil, ErrContractNotFound
	}
	// convert amount to float64 considering decimals
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(contract.Decimals)), nil))
	amountFloat, _ := new(big.Float).Quo(new(big.Float).SetInt(amount), divisor).Float64()
	// create token transfer struct
	tokenTransfer := wstypes.TokenTransfer{
		Blockchain:       "Ethereum",
		TransactionID:    tx.Hash().Hex(),
		SenderAddress:    "", // will be set by caller if needed
		RecipientAddress: recipientAddress,
		TokenSymbol:      contract.Symbol,
		TokenAmount:      amountFloat,
		TokenDecimals:    contract.Decimals,
		ContractAddress:  contractAddress,
	}
	// extract any extra data beyond the standard transfer call
	var extraData []byte
	if len(tx.Data()) > 68 {
		extraData = tx.Data()[68:]
	}
	return tokenTransfer, extraData, nil
}

// TransferERC20WithData calls an ERC20 contract, appending data to the end of the ABI encoding
func TransferERC20WithData(client *ethclient.Client, contractAddress, receiveAddress string, key string, transferAmount int, data []byte) error {
	// parse private key
	privateKey, err := crypto.HexToECDSA(key)
	if err != nil {
		return ErrInvalidPrivateKey
	}
	// get public key and address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrInvalidKey
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	// validate contract address
	if !common.IsHexAddress(contractAddress) {
		return ErrInvalidAddress
	}
	contractAddr := common.HexToAddress(contractAddress)
	// validate receive address
	if !common.IsHexAddress(receiveAddress) {
		return ErrInvalidAddress
	}
	receiveAddr := common.HexToAddress(receiveAddress)
	// get nonce
	nonce, err := client.PendingNonceAt(nil, fromAddress)
	if err != nil {
		return err
	}
	// get gas price
	gasPrice, err := client.SuggestGasPrice(nil)
	if err != nil {
		return err
	}
	// create ERC20 ABI
	erc20ABI, err := abi.JSON(strings.NewReader(`[{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","type":"function"}]`))
	if err != nil {
		return err
	}
	// encode transfer function call
	transferData, err := erc20ABI.Pack("transfer", receiveAddr, big.NewInt(int64(transferAmount)))
	if err != nil {
		return err
	}
	// append extra data if provided
	if len(data) > 0 {
		transferData = append(transferData, data...)
	}
	// estimate gas limit
	gasLimit, err := client.EstimateGas(nil, ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddr,
		Data: transferData,
	})
	if err != nil {
		return err
	}
	// create transaction
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), gasLimit, gasPrice, transferData)
	// get chain ID
	chainID, err := client.NetworkID(nil)
	if err != nil {
		return err
	}
	// sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return err
	}
	// send transaction
	err = client.SendTransaction(nil, signedTx)
	if err != nil {
		return ErrTransactionFailed
	}
	return nil
}
