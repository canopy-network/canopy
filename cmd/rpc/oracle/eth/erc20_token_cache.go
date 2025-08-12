package eth

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
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
	// context timeout for call contract calls
	callContractTimeoutS = 5
	// default cache size for token information
	defaultCacheSize = 1000
)

// ContractCaller interface defines the method needed to call ethereum contracts
type ContractCaller interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, height *big.Int) ([]byte, error)
}

// ERC20TokenCache caches token information for ERC20 contracts
type ERC20TokenCache struct {
	// client is the ethereum client used to make contract calls
	client ContractCaller
	// cache stores token information by contract address using LRU eviction
	cache *lib.LRUCache[types.TokenInfo]
	// metrics for telemetry
	metrics *lib.Metrics
}

// NewERC20TokenCache creates a new ERC20TokenCache instance
func NewERC20TokenCache(client ContractCaller, metrics *lib.Metrics) *ERC20TokenCache {
	return &ERC20TokenCache{
		client:  client,
		cache:   lib.NewLRUCache[types.TokenInfo](defaultCacheSize),
		metrics: metrics,
	}
}

// TokenInfo fetches an erc20's name, symbol and decimals from the contract
func (m *ERC20TokenCache) TokenInfo(ctx context.Context, contractAddress string) (types.TokenInfo, error) {
	// check if token info is already cached
	if info, exists := m.cache.Get(contractAddress); exists {
		// record cache hit metrics
		if m.metrics != nil {
			m.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 1, 0, 0, 0, 0, 0)
		}
		return info, nil
	}
	// fetch name from contract
	nameBytes, err := callContract(ctx, m.client, contractAddress, erc20NameFunction)
	if err != nil {
		return types.TokenInfo{}, ErrTokenInfo
	}
	// decode name from bytes
	name := decodeString(nameBytes)
	// fetch symbol from contract
	symbolBytes, err := callContract(ctx, m.client, contractAddress, erc20SymbolFunction)
	if err != nil {
		return types.TokenInfo{}, ErrTokenInfo
	}
	// decode symbol from bytes
	symbol := decodeString(symbolBytes)
	// fetch decimals from contract
	decimalsBytes, err := callContract(ctx, m.client, contractAddress, erc20DecimalsFunction)
	if err != nil {
		return types.TokenInfo{}, ErrTokenInfo
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
	m.cache.Put(contractAddress, tokenInfo)
	// record cache miss metrics
	if m.metrics != nil {
		m.metrics.UpdateEthBlockProviderMetrics(0, 0, 0, 0, 1, 0, 0, 0, 0)
	}
	return tokenInfo, nil
}

// callContract uses client to call the specified function at address
func callContract(ctx context.Context, client ContractCaller, address, function string) ([]byte, error) {
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
	// create fresh context with timeout that respects the parent context
	callCtx, cancel := context.WithTimeout(ctx, callContractTimeoutS*time.Second)
	defer cancel()
	// make the contract call
	result, err := client.CallContract(callCtx, msg, nil)
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
