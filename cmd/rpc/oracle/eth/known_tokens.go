package eth

import "github.com/ethereum/go-ethereum/common"

// Known ERC20 token contracts on Ethereum mainnet
// These are reference addresses for commonly supported tokens
// The oracle is token-agnostic and works with any ERC20 contract
var (
	// USDCMainnet is the USDC token contract on Ethereum mainnet
	// Decimals: 6
	// Standard ERC20 implementation
	USDCMainnet = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")

	// USDTMainnet is the USDT token contract on Ethereum mainnet
	// Decimals: 6
	// Non-standard ERC20 implementation:
	//   - transfer() and transferFrom() return void instead of bool
	//   - However, this does NOT affect the oracle since it observes transactions
	//     rather than calling these functions
	//   - The oracle validates transfers via transaction receipt status,
	//     which works identically for both standard and non-standard tokens
	USDTMainnet = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
)

// TokenInfo provides metadata about known tokens
type TokenInfo struct {
	Name     string
	Symbol   string
	Decimals uint8
	Address  common.Address
	Notes    string
}

// KnownTokens maps contract addresses to token information
var KnownTokens = map[common.Address]TokenInfo{
	USDCMainnet: {
		Name:     "USD Coin",
		Symbol:   "USDC",
		Decimals: 6,
		Address:  USDCMainnet,
		Notes:    "Standard ERC20 implementation",
	},
	USDTMainnet: {
		Name:     "Tether USD",
		Symbol:   "USDT",
		Decimals: 6,
		Address:  USDTMainnet,
		Notes:    "Non-standard ERC20: returns void instead of bool. This doesn't affect the oracle's passive observation approach.",
	},
}
