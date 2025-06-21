package eth

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// erc20NameFunction is the function signature for name()
	erc20NameFunction = "0x06fdde03"
	// erc20SymbolFunction is the function signature for symbol()
	erc20SymbolFunction = "0x95d89b41"
	// erc20DecimalsFunction is the function signature for decimals()
	erc20DecimalsFunction = "0x313ce567"
)

// ContractCaller interface defines the method needed to call ethereum contracts
type ContractCaller interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, height *big.Int) ([]byte, error)
}

// ERC20TokenCache caches token information for ERC20 contracts
type ERC20TokenCache struct {
	// client is the ethereum client used to make contract calls
	client ContractCaller
	// cache stores token information by contract address
	cache map[string]types.TokenInfo
}

// NewERC20TokenCache creates a new ERC20TokenCache instance
func NewERC20TokenCache(client ContractCaller) *ERC20TokenCache {
	return &ERC20TokenCache{
		client: client,
		cache:  make(map[string]types.TokenInfo),
	}
}

// TokenInfo fetches an erc20's name, symbol and decimals from the contract
func (m *ERC20TokenCache) TokenInfo(contractAddress string) (types.TokenInfo, error) {
	// check if token info is already cached
	if info, exists := m.cache[contractAddress]; exists {
		return info, nil
	}
	// validate contract address format
	if !common.IsHexAddress(contractAddress) {
		return types.TokenInfo{}, ErrInvalidAddress
	}
	// fetch name from contract
	nameBytes, err := callContract(m.client, contractAddress, erc20NameFunction)
	if err != nil {
		return types.TokenInfo{}, err
	}
	// decode name from bytes
	name := decodeString(nameBytes)
	// fetch symbol from contract
	symbolBytes, err := callContract(m.client, contractAddress, erc20SymbolFunction)
	if err != nil {
		return types.TokenInfo{}, err
	}
	// decode symbol from bytes
	symbol := decodeString(symbolBytes)
	// fetch decimals from contract
	decimalsBytes, err := callContract(m.client, contractAddress, erc20DecimalsFunction)
	if err != nil {
		return types.TokenInfo{}, err
	}
	// decode decimals from bytes
	decimals := decodeUint8(decimalsBytes)
	// create token info struct
	tokenInfo := types.TokenInfo{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}
	// cache the token info
	m.cache[contractAddress] = tokenInfo
	return tokenInfo, nil
}

// callContract uses client to call the specified function at address
func callContract(client ContractCaller, address, function string) ([]byte, error) {
	// create context for the call
	ctx := context.Background()
	// convert address string to common.Address
	contractAddr := common.HexToAddress(address)
	// decode function signature from hex
	data, err := hex.DecodeString(strings.TrimPrefix(function, "0x"))
	if err != nil {
		return nil, ErrInvalidTransactionData
	}
	// create call message
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}
	// make the contract call
	result, err := client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, ErrContractNotFound
	}
	return result, nil
}

// decodeString decodes a string from ethereum contract call result
func decodeString(data []byte) string {
	// check if data is long enough for offset and length
	if len(data) < 64 {
		return ""
	}
	// get offset (first 32 bytes)
	offset := new(big.Int).SetBytes(data[0:32]).Uint64()
	// check if offset is valid
	if offset >= uint64(len(data)) {
		return ""
	}
	// get length from offset position
	if offset+32 > uint64(len(data)) {
		return ""
	}
	length := new(big.Int).SetBytes(data[offset : offset+32]).Uint64()
	// extract string data
	if offset+32+length > uint64(len(data)) {
		return ""
	}
	stringData := data[offset+32 : offset+32+length]
	return string(stringData)
}

// decodeUint8 decodes a uint8 from ethereum contract call result
func decodeUint8(data []byte) uint8 {
	// check if data is long enough
	if len(data) < 32 {
		return 0
	}
	// convert last byte to uint8
	return uint8(data[31])
}
