package eth

import "github.com/ethereum/go-ethereum/common"

// USDTMainnet is the USDT token contract on Ethereum mainnet (6 decimals).
// USDT is a non-standard ERC20 (transfer() returns void), but the oracle observes
// transactions passively and validates via receipt status, so this does not matter.
// Retained as a reference address used by tests.
var USDTMainnet = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
