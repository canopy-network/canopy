package rpc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file wraps Canopy with the Ethereum JSON-RPC interface as specified here: https://ethereum.org/en/developers/docs/apis/json-rpc */

// EthereumHandler is a helper function that abstracts common workflows of ethereum calls using the JSON rpc 2.0 specification
func (s *Server) EthereumHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error
	// create a new ethereumRequest instance to populate with request data
	ptr := new(ethRPCRequest)
	// attempt to unmarshal the request body into the keystoreRequest
	if ok := unmarshal(w, r, ptr); !ok {
		return
	}
	var args []any
	// convert the params to a list
	if err = json.Unmarshal(ptr.Params, &args); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	var ethResponse any
	switch ptr.Method {
	case `web3_clientVersion`:
		ethResponse, err = s.Web3ClientVersion(args)
	case `web3_sha3`:
		ethResponse, err = s.Web3Sha3(args)
	case `net_version`:
		ethResponse, err = s.NetVersion(args)
	case `net_listening`:
		ethResponse, err = s.NetListening(args)
	case `net_peerCount`:
		ethResponse, err = s.NetPeerCount(args)
	case `eth_syncing`:
		ethResponse, err = s.EthSyncing(args)
	case `eth_chainId`:
		ethResponse, err = s.EthChainId(args)
	case `eth_gasPrice`:
		ethResponse, err = s.EthGasPrice(args)
	case `eth_accounts`:
		ethResponse, err = s.EthAccounts(args)
	case `eth_blockNumber`:
		ethResponse, err = s.EthBlockNumber(args)
	case `eth_getBalance`:
		ethResponse, err = s.EthGetBalance(args)
	case `eth_getTransactionCount`:
		ethResponse, err = s.EthGetTransactionCount(args)
	case `eth_getBlockTransactionCountByHash`:
		ethResponse, err = s.EthGetBlockTransactionCountByHash(args)
	case `eth_getBlockTransactionCountByNumber`:
		ethResponse, err = s.EthGetBlockTransactionCountByNumber(args)
	case `eth_getUncleCountByBlockHash`:
		ethResponse, err = s.EthGetUncleCountByBlockHash(args)
	case `eth_getUncleCountByBlockNumber`:
		ethResponse, err = s.EthGetUncleCountByBlockNumber(args)
	case `eth_getCode`:
		ethResponse, err = s.EthGetCode(args)
	case `eth_sendRawTransaction`:
		ethResponse, err = s.EthSendRawTransaction(args)
	case `eth_call`:
		ethResponse, err = s.EthCall(args)
	case `eth_estimateGas`:
		ethResponse, err = s.EthEstimateGas(args)
	case `eth_getBlockByHash`:
		ethResponse, err = s.EthGetBlockByHash(args)
	case `eth_getBlockByNumber`:
		ethResponse, err = s.EthGetBlockByNumber(args)
	case `eth_getTransactionByHash`:
		ethResponse, err = s.EthGetTransactionByHash(args)
	case `eth_getTransactionByBlockHashAndIndex`:
		ethResponse, err = s.EthGetTransactionByBlockHashAndIndex(args)
	case `eth_getTransactionByBlockNumberAndIndex`:
		ethResponse, err = s.EthGetTransactionByBlockNumAndIndex(args)
	case `eth_getTransactionReceipt`:
		ethResponse, err = s.EthGetTransactionReceipt(args)
	case `eth_getUncleByBlockHashAndIndex`:
		ethResponse, err = s.EthGetUncleByBlockHashAndIndex(args)
	case `eth_getUncleByBlockNumberAndIndex`:
		ethResponse, err = s.EthGetUncleByBlockNumAndIndex(args)
	case `eth_newFilter`:
		ethResponse, err = s.EthNewFilter(args)
	case `eth_newBlockFilter`:
		ethResponse, err = s.EthNewBlockFilter(args)
	case `eth_getFilterChanges`:
		ethResponse, err = s.EthGetFilterChanges(args)
	case `eth_getFilterLogs`:
		ethResponse, err = s.EthGetFilterLogs(args)
	case `eth_getLogs`:
		ethResponse, err = s.EthGetLogs(args)
	case `eth_newPendingTransactionFilter`:
		ethResponse, err = s.EthNewPendingTxsFilter(args)
	case `eth_uninstallFilter`:
		ethResponse, err = s.EthUninstallFilter(args)
	case `eth_blobBaseFee`:
		ethResponse, err = s.EthBlobBaseFee(args)
	default:
		// purposefully don't support any method that requires private key unlocks
		err = fmt.Errorf("the method %s does not exist/is not available", ptr.Method)
	}
	// convert the error to ethError
	var ethError *ethereumRPCError
	if err != nil {
		ethError = &ethereumRPCError{
			Code:    -32601,
			Message: err.Error(),
		}
	}
	// write the final result
	write(w, ethRPCResponse{
		ID:      ptr.ID,
		JSONRPC: "2.0",
		Result:  ethResponse,
		Error:   ethError,
	}, http.StatusOK)
}

// startEthRPCService() runs the needed routines for the eth rpc wrapper
func (s *Server) startEthRPCService() {
	go s.startEthPseudoNonceService()
	go s.startEthPendingTxsExpireService()
	go s.startEthFilterExpireService()
}

// Web3ClientVersion() return a dummy string for compatibility
func (s *Server) Web3ClientVersion(_ []any) (any, error) { return "Canopy_Eth_Wrapper", nil }

// Web3Sha3() executes the Keccak-256 hash
func (s *Server) Web3Sha3(args []any) (any, error) {
	strToHash, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	// convert from hex string to bytes
	bzToHash, err := lib.StringToBytes(cleanHex(strToHash))
	if err != nil {
		return nil, err
	}
	// execute the hash
	return hexutil.Bytes(ethCrypto.Keccak256(bzToHash)), nil
}

// NetVersion() returns the network id
func (s *Server) NetVersion(_ []any) (any, error) {
	return strconv.FormatUint(fsm.CanopyIdsToEVMChainId(s.config.ChainId, s.config.NetworkID), 10), nil
}

// NetListening() canopy is always listening for peers
func (s *Server) NetListening(_ []any) (any, error) { return true, nil }

// NetPeerCount() returns the number of peers
func (s *Server) NetPeerCount(_ []any) (any, error) {
	return hexutil.Uint64(s.controller.P2P.PeerCount()), nil
}

// EthSyncing() returns the syncing status of the node
func (s *Server) EthSyncing(_ []any) (any, error) {
	if !s.controller.Syncing().Load() {
		return false, nil
	}
	// return the syncing response
	return ethSyncingResponse{
		StartingBlock: hexutil.Uint64(1),
		CurrentBlock:  hexutil.Uint64(s.controller.ChainHeight()),
		HighestBlock:  hexutil.Uint64(s.controller.ChainHeight()),
	}, nil
}

// EthChainId() returns the chain id of this node
func (s *Server) EthChainId(_ []any) (any, error) {
	return hexutil.Uint64(fsm.CanopyIdsToEVMChainId(s.config.ChainId, s.config.NetworkID)), nil
}

// gas = tx.Fee * 100
// gasPrice = 1e10 (10,000,000,000 wei = 0.01 uCNPY)
// fee = gas * gasPrice = tx.Fee * 100 * 1e10 = tx.Fee * 1e12
var ethGasPrice = int64(10_000_000_000)

// EthGasPrice() returns minimum_fee / eth_gas_limit to be compatible with the
func (s *Server) EthGasPrice(_ []any) (any, error) { return hexutil.Big(*big.NewInt(ethGasPrice)), nil }

// EthAccounts() return all keystore addresses
func (s *Server) EthAccounts(_ []any) (any, error) {
	keystore, err := crypto.NewKeystoreFromFile(s.config.DataDirPath)
	if err != nil {
		return nil, err
	}
	// create a list of ethereum compatible addresses
	var ethAddresses []string
	for _, account := range keystore.AddressMap {
		// convert the public key string to an object
		publicKey, e := crypto.NewPublicKeyFromString(account.PublicKey)
		if e != nil {
			return nil, e
		}
		// if the key is an ethereum compatible public key
		if _, ok := publicKey.(*crypto.ETHSECP256K1PublicKey); ok {
			ethAddresses = append(ethAddresses, "0x"+account.KeyAddress)
		}
	}
	return ethAddresses, nil
}

// EthBlobBaseFee() returns the base fee for send transactions
func (s *Server) EthBlobBaseFee(a []any) (any, error) { return s.EthGasPrice(a) }

// EthBlockNumber() returns the height of the chain
func (s *Server) EthBlockNumber(_ []any) (result any, err error) {
	// create a read-only state for the latest block
	_ = s.readOnlyState(0, func(state *fsm.StateMachine) lib.ErrorI {
		result = hexutil.Uint64(state.Height() - 1)
		return nil
	})
	return
}

// EthGetBalance() returns the balance of an address
func (s *Server) EthGetBalance(args []any) (result any, err error) {
	// extract the address from the args
	address, err := addressFromArgs(args)
	if err != nil {
		return
	}
	// handle the block tag
	height, err := blockTagFromArgs(args)
	if err != nil {
		return
	}
	// create a read-only state for the block tag
	_ = s.readOnlyState(height, func(state *fsm.StateMachine) (e lib.ErrorI) {
		// get the balance for the address
		balance, e := state.GetAccountBalance(address)
		if e != nil {
			return
		}
		// upscale to 18 dec in hex string format
		result = hexutil.Big(*fsm.UpscaleTo18Decimals(balance))
		// exit
		return
	})
	return
}

// EthGetTransactionCount() returns a pseudo-nonce in the form of a random 'created_at' height
func (s *Server) EthGetTransactionCount(args []any) (any, error) {
	address, err := addressFromArgs(args)
	if err != nil {
		return nil, err
	}
	return hexutil.Uint64(getAndIncPseudoNonce(address.String())), nil
}

// EthGetBlockTransactionCountByHash() returns the number of transactions in a block by hash
func (s *Server) EthGetBlockTransactionCountByHash(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block hash
		blockHash, err := bytesFromArgs(args)
		if err != nil {
			return nil, err
		}
		// get the block from hash
		block, err := st.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		// check for a nil block
		if block == nil || block.BlockHeader == nil {
			return nil, lib.ErrNilBlock()
		}
		// return the result
		return hexutil.Uint64(block.BlockHeader.NumTxs), nil
	})
}

// EthGetBlockTransactionCountByNumber() returns the number of transactions in a block by height
func (s *Server) EthGetBlockTransactionCountByNumber(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block height
		blockHeight, err := intFromArgs(args, 0)
		if err != nil {
			return nil, err
		}
		// get the block from hash
		block, err := st.GetBlockByHeight(uint64(blockHeight))
		if err != nil {
			return nil, err
		}
		// get and increment the pseudo-nonce
		return hexutil.Uint64(block.BlockHeader.NumTxs), nil
	})
}

// EthGetUncleCountByBlockHash() n/a to Canopy as NestBFT doesn't have the concept of 'uncles'
func (s *Server) EthGetUncleCountByBlockHash(_ []any) (any, error) { return "0x0", nil }

// EthGetUncleCountByBlockNumber() n/a to Canopy as NestBFT doesn't have the concept of 'uncles'
func (s *Server) EthGetUncleCountByBlockNumber(_ []any) (any, error) { return "0x0", nil }

// EthGetCode() returns pseudo-ERC20 code for the CNPY contract and echoes the EOA address for everything else
func (s *Server) EthGetCode(args []any) (any, error) {
	// get the address from the args
	address, err := addressFromArgs(args)
	if err != nil {
		return nil, err
	}
	// get the string for the address
	addressString := "0x" + address.String()
	// if asking about the canopy pseudo-contract address
	switch addressString {
	case fsm.CNPYContractAddress, fsm.SwapCNPYContractAddress, fsm.StakedCNPYContractAddress:
		return CanopyPseudoContractByteCode, nil
	}
	return "0x", nil
}

// EthSendRawTransaction() converts the RLP transaction into a Canopy compatible transaction and submits it
// - a valid RLP signature is considered a valid signature in Canopy for send transactions
func (s *Server) EthSendRawTransaction(args []any) (any, error) {
	// extract the raw transaction bytes
	rawTx, err := bytesFromArgs(args)
	if err != nil {
		return nil, err
	}
	// convert it to a Canopy send transaction
	transaction, err := fsm.RLPToCanopyTransaction(rawTx)
	if err != nil {
		return nil, err
	}
	// ensure created height isn't too close to the limit
	if int64(transaction.CreatedHeight) < int64(s.controller.ChainHeight())-fsm.BlockAcceptanceRange/2 {
		return nil, lib.ErrInvalidTxHeight()
	}
	// extract the public key from the message
	pubKey, err := crypto.NewPublicKeyFromBytes(transaction.Signature.PublicKey)
	if err != nil {
		return nil, err
	}
	// increment the pseudo-nonce
	incPseudoNonce(pubKey.Address().String())
	// marshal the transaction to protobuf
	bz, err := lib.Marshal(transaction)
	if err != nil {
		return nil, err
	}
	// send transaction to controller
	if err = s.controller.SendTxMsgs([][]byte{bz}); err != nil {
		return nil, err
	}
	// get the tx hash string
	txHashString := crypto.HashString(bz)
	// set in pending
	shouldSimultePending(txHashString)
	// return the transaction hash
	return "0x" + txHashString, nil
}

// EthCall() simulates a call to a 'smart contract' for Canopy
func (s *Server) EthCall(args []any) (any, error) {
	if len(args) < 1 {
		return nil, errors.New("missing call arguments")
	}
	// handle the block tag
	height, err := blockTagFromArgs(args)
	if err != nil {
		return nil, err
	}
	// extract the call data
	callParams, ok := args[0].(map[string]any)
	if !ok {
		return nil, errors.New("invalid call argument format")
	}
	// get the sender address hex
	fromHex, ok := callParams["from"].(string)
	if !ok {
		fromHex = "0x" + strings.Repeat("0", 20)
	}
	// parse the `data` field from the call data
	dataHex, ok := callParams["data"].(string)
	if !ok {
		return nil, errors.New("invalid or missing 'data' field")
	}
	// parse the `to` field from the call data
	toHex, _ := callParams["to"].(string)
	switch toHex {
	default:
		// exit as it's a non-contract call
		return "0x", nil
	case fsm.CNPYContractAddress, fsm.StakedCNPYContractAddress, fsm.SwapCNPYContractAddress:
		// continue
	}
	// get the sender address
	fromAddress, err := crypto.NewAddressFromString(cleanHex(fromHex))
	if err != nil {
		return nil, err
	}
	// decode the data from hex
	data, err := lib.StringToBytes(cleanHex(dataHex[:]))
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	// validate the data length
	if len(data) < 4 {
		return nil, errors.New("insufficient data length")
	}
	// parse the selector
	selector := lib.BytesToString(data[:4])
	// create a read-only state for the block tag and write the height
	var encoded hexutil.Bytes
	return encoded, s.readOnlyState(height, func(state *fsm.StateMachine) lib.ErrorI {
		switch selector {
		case "95d89b41": // symbol()
			switch toHex {
			case fsm.CNPYContractAddress:
				encoded, err = pack(ABIStringType, "CNPY")
			case fsm.StakedCNPYContractAddress:
				encoded, err = pack(ABIStringType, "stCNPY")
			case fsm.SwapCNPYContractAddress:
				encoded, err = pack(ABIStringType, "swCNPY")
			}
		case "06fdde03": // name()
			switch toHex {
			case fsm.CNPYContractAddress:
				encoded, err = pack(ABIStringType, "Canopy")
			case fsm.StakedCNPYContractAddress:
				encoded, err = pack(ABIStringType, "Staked Canopy")
			case fsm.SwapCNPYContractAddress:
				encoded, err = pack(ABIStringType, "Swap Canopy")
			}
		case "313ce567": // decimals()
			encoded, err = pack(ABIUint8Type, uint8(6))
		case "18160ddd": // totalSupply()
			supply, e := state.GetSupply()
			if e != nil {
				return e
			}
			switch toHex {
			case fsm.CNPYContractAddress:
				encoded, err = pack(ABIUint256Type, new(big.Int).SetUint64(supply.Total))
			case fsm.StakedCNPYContractAddress:
				encoded, err = pack(ABIUint256Type, new(big.Int).SetUint64(supply.Staked))
			case fsm.SwapCNPYContractAddress:
				escrowed, er := state.GetTotalEscrowed(nil)
				if er != nil {
					return er
				}
				encoded, err = pack(ABIUint256Type, new(big.Int).SetUint64(escrowed))
			}
		case "70a08231": // balanceOf(address)
			address, e := parseAddressFromABI(data)
			if e != nil {
				return e
			}
			var balance uint64
			switch toHex {
			case fsm.CNPYContractAddress:
				balance, e = state.GetAccountBalance(address)
				if e != nil {
					return e
				}
			case fsm.StakedCNPYContractAddress:
				val, e := state.GetValidator(address)
				if e != nil {
					return e
				}
				balance = val.StakedAmount
			case fsm.SwapCNPYContractAddress:
				balance, e = state.GetTotalEscrowed(address)
				if e != nil {
					return e
				}
			}
			encoded, err = pack(ABIUint256Type, new(big.Int).SetUint64(balance))
		case "a9059cbb": // transfer(address,uint256)
			if toHex == fsm.StakedCNPYContractAddress || toHex == fsm.SwapCNPYContractAddress {
				return lib.NewError(1, "ethereum", fmt.Sprintf("unsupported selector: 0x%s", selector))
			}
			_, amount, e := parseAddressAndAmountFromABI(data)
			if e != nil {
				return e
			}
			balance, e := state.GetAccountBalance(fromAddress)
			if e != nil {
				return e
			}
			if balance < amount {
				encoded, err = revert("ERC20: transfer amount exceeds balance")
				break
			}
			encoded, err = pack(ABIBoolType, true)
		case "23b872dd", "095ea7b3", "dd62ed3e", "79cc6790", "42966c68", "40c10f19": // unsupported ERC20 methods
			encoded, err = revert("ERC20: method not supported")
		default:
			return lib.NewError(1, "ethereum", fmt.Sprintf("unsupported selector: 0x%s", selector))
		}
		if err != nil {
			return lib.NewError(1, "ethereum", err.Error())
		}
		return nil
	})
}

// EthEstimateGas() returns the corresponding Canopy fee for the message
func (s *Server) EthEstimateGas(args []any) (any, error) {
	if len(args) < 1 {
		return nil, errors.New("missing call arguments")
	}
	// extract the call data
	callParams, ok := args[0].(map[string]any)
	if !ok {
		return nil, errors.New("invalid call argument format")
	}
	// parse the `data` field from the call data
	dataHex, _ := callParams["data"].(string)
	// create a txRequest
	req, err := new(txRequest), error(nil)
	// if invalid selector data return send
	if len(dataHex) < 10 {
		err = s.getFeeFromState(req, fsm.MessageSendName)
	} else {
		switch dataHex[2:10] {
		case fsm.StakeSelector:
			err = s.getFeeFromState(req, fsm.MessageStakeName)
		case fsm.EditStakeSelector:
			err = s.getFeeFromState(req, fsm.MessageEditStakeName)
		case fsm.UnstakeSelector:
			err = s.getFeeFromState(req, fsm.MessageUnstakeName)
		case fsm.CreateOrderSelector:
			err = s.getFeeFromState(req, fsm.MessageCreateOrderName)
		case fsm.EditOrderSelector:
			err = s.getFeeFromState(req, fsm.MessageEditOrderName)
		case fsm.DeleteOrderSelector:
			err = s.getFeeFromState(req, fsm.MessageDeleteOrderName)
		case fsm.SubsidySelector:
			err = s.getFeeFromState(req, fsm.MessageSubsidyName)
		default:
			err = s.getFeeFromState(req, fsm.MessageSendName)
		}
	}
	if err != nil {
		return nil, err
	}
	return hexutil.Uint64(req.Fee * 100), nil
}

// EthGetBlockByHash() returns a dummy-ish block (based on the actual Canopy block) that is EIP-1559 compatible
func (s *Server) EthGetBlockByHash(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block hash
		blockHash, e := bytesFromArgs(args)
		if e != nil {
			return nil, e
		}
		// extract full txs option
		fullTxs := boolFromArgs(args)
		// get the block from hash
		block, e := st.GetBlockByHash(blockHash)
		if e != nil {
			return nil, e
		}
		return s.blockToEIP1559Block(block, fullTxs)
	})
}

// EthGetBlockByNumber() returns a dummy-ish block (based on the actual Canopy block) that is EIP-1559 compatible
func (s *Server) EthGetBlockByNumber(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block height
		blockHeight, err := intFromArgs(args, 0)
		if err != nil {
			return nil, err
		}
		// extract full txs option
		fullTxs := boolFromArgs(args)
		// get the block from hash
		block, err := st.GetBlockByHeight(uint64(blockHeight))
		if err != nil {
			return nil, err
		}
		return s.blockToEIP1559Block(block, fullTxs)
	})
}

// EthGetTransactionByHash() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionByHash(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the tx hash
		txHash, e := bytesFromArgs(args)
		if e != nil {
			return nil, e
		}
		// get the transaction by hash
		tx, e := st.GetTxByHash(txHash)
		if e != nil || tx.TxHash == "" {
			hashString := lib.BytesToString(txHash)
			// check mempool
			if pending := s.controller.Mempool.Contains(hashString); pending {
				return nil, errors.New("tx pending")
			}
			// check eth pending cache
			if isPending := shouldSimultePending(hashString); isPending {
				return nil, errors.New("tx pending")
			}
			// pseudo-failed height
			failedHeight := s.controller.ChainHeight() - 1
			// get the block associated with the transaction height
			b, _ := st.GetBlockHeaderByHeight(failedHeight)
			// return failed to prevent 'resend'
			return ethRPCTransaction{
				BlockHash:         common.BytesToHash(b.BlockHeader.Hash),
				BlockNumber:       hexutil.Big(*big.NewInt(int64(tx.Height))),
				From:              common.BytesToAddress(tx.Sender),
				Gas:               hexutil.Uint64(21000),
				Hash:              common.HexToHash("0x" + lib.BytesToString(txHash)),
				TxHash:            common.HexToHash("0x" + lib.BytesToString(txHash)),
				TransactionIndex:  hexutil.Uint64(tx.Index),
				Type:              types.DynamicFeeTxType,
				ChainID:           hexutil.Big(*big.NewInt(int64(fsm.CanopyIdsToEVMChainId(s.config.ChainId, s.config.NetworkID)))),
				Status:            hexutil.Uint64(types.ReceiptStatusFailed),
				CumulativeGasUsed: hexutil.Uint64(21000 * uint64(math.Min(1, float64(b.BlockHeader.NumTxs)))),
				Bloom:             make([]byte, 256),
				ContractAddress:   common.Address{},
				GasUsed:           hexutil.Uint64(int64(21000)),
			}, nil
		}
		// get the block associated with the transaction height
		block, _ := st.GetBlockHeaderByHeight(tx.Height)
		// convert to eip 1559 transaction
		return s.txToEIP1559(block, tx)
	})
}

// EthGetTransactionByBlockHashAndIndex() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionByBlockHashAndIndex(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block hash
		blockHash, e := bytesFromArgs(args)
		if e != nil {
			return nil, e
		}
		// get the index
		index, e := intFromArgs(args, 1)
		if e != nil {
			return nil, e
		}
		// get the tx from hash
		block, e := st.GetBlockByHash(blockHash)
		if e != nil {
			return nil, e
		}
		// ensure index isn't out of bounds
		if len(block.Transactions) <= int(index) {
			return nil, fmt.Errorf("index %d invalid for block 0x%s", index, lib.BytesToString(blockHash))
		}
		// convert to eip 1559 transaction
		return s.txToEIP1559(block, block.Transactions[index])
	})
}

// EthGetTransactionByBlockNumAndIndex() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionByBlockNumAndIndex(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		// get the block height
		blockHeight, e := intFromArgs(args, 0)
		if e != nil {
			return nil, e
		}
		// get the index
		index, e := intFromArgs(args, 1)
		if e != nil {
			return nil, e
		}
		// get the tx from hash
		block, e := st.GetBlockByHeight(uint64(blockHeight))
		if e != nil {
			return nil, e
		}
		// ensure index isn't out of bounds
		if len(block.Transactions) <= int(index) {
			return nil, fmt.Errorf("index %d invalid for block height: %d", index, blockHeight)
		}
		// convert to eip 1559 transaction
		return s.txToEIP1559(block, block.Transactions[index])
	})
}

// EthGetTransactionReceipt() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionReceipt(args []any) (any, error) {
	return s.EthGetTransactionByHash(args)
}

// EthGetUncleByBlockHashAndIndex() returns null (no uncles) as expected
func (s *Server) EthGetUncleByBlockHashAndIndex(_ []any) (any, error) { return nil, nil }

// EthGetUncleByBlockNumAndIndex() returns null (no uncles) as expected
func (s *Server) EthGetUncleByBlockNumAndIndex(_ []any) (any, error) { return nil, nil }

// EthNewFilter() creates a filter object, based on filter options, to notify when the state changes
func (s *Server) EthNewFilter(args []any) (any, error) {
	// convert the args to filter params
	params, err := filterParamsFromArgs(args)
	if err != nil {
		return nil, err
	}
	// create the filter
	return s.newEthFilter(params)
}

// EthNewBlockFilter() creates a filter object, updating when a new block is produced
func (s *Server) EthNewBlockFilter(_ []any) (any, error) {
	return s.newEthFilter(newFilterParams{filter: ethFilter{Blocks: true}})
}

// EthNewPendingTxsFilter() creates a filter object, updating when a new transaction is processed
func (s *Server) EthNewPendingTxsFilter(_ []any) (any, error) {
	return s.newEthFilter(newFilterParams{filter: ethFilter{PendingTxs: true}})
}

// EthUninstallFilter() deletes a filter object
func (s *Server) EthUninstallFilter(args []any) (any, error) {
	// get the filter id
	id, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	// delete the filter from the sync.map
	_, deleted := ethFilters.LoadAndDelete(id)
	return deleted, nil
}

// EthGetFilterChanges() gets the latest logs since the last filter call (simulated using lastReadHeight
func (s *Server) EthGetFilterChanges(args []any) (any, error) {
	// get the filter id
	id, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	// get the filter
	filter, lastReadHeight, err := s.getEthFilter(id)
	if err != nil {
		return nil, err
	}
	// call ethGetLogs
	return s.ethGetLogs(filter, lastReadHeight)
}

// EthGetFilterLogs() returns an array of all logs matching a pre-created filter with given id
func (s *Server) EthGetFilterLogs(args []any) (any, error) {
	// get the filter id
	id, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	// get the filter
	filter, _, err := s.getEthFilter(id)
	if err != nil {
		return nil, err
	}
	// call ethGetLogs
	return s.ethGetLogs(filter, 1)
}

// EthGetLogs() returns an array of all logs matching the passed filter argument
func (s *Server) EthGetLogs(args []any) (any, error) {
	// convert the args to filter params
	params, err := filterParamsFromArgs(args)
	if err != nil {
		return nil, err
	}
	// call ethGetLogs
	return s.ethGetLogs(&params.filter, 1)
}

// MAJOR HELPER FUNCTIONS BELOW

// blockToEIP1559Block() attempts to convert a Canopy block to a EIP1559Block for display only
func (s *Server) blockToEIP1559Block(block *lib.BlockResult, fullTx bool) (ethRPCBlock, error) {
	// ensure the block exists
	if block.BlockHeader.Hash == nil {
		return ethRPCBlock{}, errors.New("block not found")
	}
	// get the minimum send tx fee
	tx := new(txRequest)
	if err := s.getFeeFromState(tx, fsm.MessageSendName); err != nil {
		return ethRPCBlock{}, err
	}
	// tx.Fee x 100 to ensure always above 21,000
	sendFee := big.NewInt(int64(tx.Fee * 100))
	// make a structure to capture the EIP-1559 transactions
	var txs []ethRPCTransaction
	transactions := make([]interface{}, len(txs))
	for _, tx := range block.Transactions {
		if fullTx {
			eip1559Tx, e := s.txToEIP1559(block, tx)
			if e != nil {
				return ethRPCBlock{}, e
			}
			transactions = append(transactions, eip1559Tx)
		} else {
			transactions = append(transactions, "0x"+tx.TxHash)
		}
	}
	// create the EIP-1559 block
	return ethRPCBlock{
		Number:                hexutil.Big(*big.NewInt(int64(block.BlockHeader.Height))),
		Hash:                  common.BytesToHash(block.BlockHeader.Hash),
		ParentHash:            common.BytesToHash(block.BlockHeader.LastBlockHash),
		Sha3Uncles:            types.EmptyUncleHash,
		LogsBloom:             types.Bloom{},
		StateRoot:             common.BytesToHash(block.BlockHeader.StateRoot),
		Miner:                 common.BytesToAddress(block.BlockHeader.ProposerAddress),
		ExtraData:             hexutil.Bytes("Canopy EIP1559 Wrapper is for display only"),
		GasLimit:              50_000_000,
		GasUsed:               hexutil.Uint64(new(big.Int).Mul(sendFee, big.NewInt(int64(len(block.Transactions)))).Uint64()),
		Timestamp:             hexutil.Uint64(time.UnixMicro(int64(block.BlockHeader.Time)).Unix()),
		TransactionsRoot:      common.BytesToHash(block.BlockHeader.TransactionRoot),
		ReceiptsRoot:          common.BytesToHash(block.BlockHeader.TransactionRoot),
		BaseFeePerGas:         hexutil.Big(*big.NewInt(ethGasPrice)),
		WithdrawalsRoot:       types.EmptyWithdrawalsHash,
		ParentBeaconBlockRoot: common.BytesToHash(block.BlockHeader.ValidatorRoot),
		RequestsHash:          types.EmptyRequestsHash,
		Size:                  hexutil.Uint64(block.Meta.Size),
		Transactions:          transactions,
		Uncles:                make([]common.Hash, 0),
	}, nil
}

// txToEIP1559() attempts to convert a Canopy transaction to EIP1559 for display only
func (s *Server) txToEIP1559(b *lib.BlockResult, tx *lib.TxResult) (ethRPCTransaction, error) {
	var amount uint64
	var logs []string
	// extract the hash and numTxs from the block result
	// extract the amount if it's send message
	if tx.MessageType == fsm.MessageSendName {
		sendMsg, err := msgToSend(tx.Transaction.Msg)
		if err != nil {
			return ethRPCTransaction{}, err
		}
		amount = sendMsg.Amount
		// generate a pseudo log for it
		logs = []string{transferEventFilterHash,
			fmt.Sprintf("0x%064s", lib.BytesToString(sendMsg.FromAddress)),
			fmt.Sprintf("0x%064s", lib.BytesToString(sendMsg.ToAddress)),
		}
	}
	// get the recipient if applicable
	var to common.Address
	if len(tx.Recipient) != 0 {
		to = common.BytesToAddress(tx.Recipient)
	}
	// convert to an EIP1559Tx and add to the list
	return ethRPCTransaction{
		BlockHash:         common.BytesToHash(b.BlockHeader.Hash),
		BlockNumber:       hexutil.Big(*big.NewInt(int64(tx.Height))),
		From:              common.BytesToAddress(tx.Sender),
		Gas:               hexutil.Uint64(tx.Transaction.Fee),
		GasPrice:          hexutil.Big(*big.NewInt(ethGasPrice)),
		GasFeeCap:         hexutil.Big(*big.NewInt(ethGasPrice)),
		GasTipCap:         hexutil.Big(*big.NewInt(0)),
		Hash:              common.HexToHash("0x" + tx.TxHash),
		TxHash:            common.HexToHash("0x" + tx.TxHash),
		Nonce:             hexutil.Uint64(tx.Transaction.CreatedHeight),
		To:                to,
		TransactionIndex:  hexutil.Uint64(tx.Index),
		Value:             hexutil.Big(*fsm.UpscaleTo18Decimals(amount)),
		Type:              types.DynamicFeeTxType,
		ChainID:           hexutil.Big(*big.NewInt(int64(tx.Transaction.ChainId))),
		Status:            hexutil.Uint64(types.ReceiptStatusSuccessful),
		CumulativeGasUsed: hexutil.Uint64(tx.Transaction.Fee * uint64(math.Min(1, float64(b.BlockHeader.NumTxs)))),
		Bloom:             make([]byte, 256),
		Logs:              logs,
		ContractAddress:   common.Address{},
		GasUsed:           hexutil.Uint64(int64(tx.Transaction.Fee)),
		EffectiveGasPrice: hexutil.Big(*big.NewInt(ethGasPrice)),
	}, nil
}

// ethGetLogs() simulates eth_getLogs call by executing queries over the indexer and mempool
// - canopy only has 1 pseudo-smart contract and only 1 event (transfer) the implementation is simple
// - canopy generates the logs in real-time upon call by using the TxIndexer
func (s *Server) ethGetLogs(filter *ethFilter, lastReadHeight uint64) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		var strResults []string
		// handle pending txs filter
		if filter.PendingTxs {
			s.controller.Mempool.L.Lock()
			transactions := s.controller.Mempool.GetTransactions(math.MaxUint64)
			s.controller.Mempool.L.Unlock()
			for _, tx := range transactions {
				strResults = append(strResults, "0x"+crypto.HashString(tx))
			}
			return strResults, nil
		}
		// handle new blocks filter
		if filter.Blocks {
			// from the last read height to the chain height
			for i := lastReadHeight; i < s.controller.ChainHeight(); i++ {
				block, e := st.GetBlockHeaderByHeight(i)
				if e != nil {
					return nil, e
				}
				strResults = append(strResults, "0x"+lib.BytesToString(block.BlockHeader.Hash))
			}
			return strResults, nil
		}
		// set the start height
		startHeight := filter.StartHeight
		if lastReadHeight != 0 {
			startHeight = lastReadHeight
		}
		// set height at latest block
		endHeight := filter.EndHeight
		if endHeight == 0 {
			endHeight = s.controller.ChainHeight()
		}
		// parse blocks looking for an appropriate response
		response := make([]ethGetLogsResponse, 0)
		for i := startHeight; i <= endHeight; i++ {
			// get the block
			block, err := st.GetBlockByHeight(i)
			if err != nil {
				return nil, err
			}
			// for each send transaction in the block
			for _, tx := range block.Transactions {
				// ignore non applicable txs
				if tx.MessageType != fsm.MessageSendName ||
					!s.passesAddressFilter(tx.Sender, filter.Sender) ||
					!s.passesAddressFilter(tx.Recipient, filter.Recipient) {
					continue
				}
				// convert the transaction to a getLogs response
				converted, e := s.txToGetLogsResp(block.BlockHeader.Hash, tx)
				if e != nil {
					return nil, e
				}
				// add to the list
				response = append(response, converted)
			}
		}
		return response, nil
	})
}

// txToGetLogsResp() converts a send message into an ethGetLogsResponse
func (s *Server) txToGetLogsResp(blockHash []byte, tx *lib.TxResult) (ethGetLogsResponse, error) {
	sendMessage, err := msgToSend(tx.Transaction.Msg)
	if err != nil {
		return ethGetLogsResponse{}, err
	}
	return ethGetLogsResponse{
		LogIndex:              fmt.Sprintf("0x%x", tx.Index),
		BlockNumber:           fmt.Sprintf("0x%x", tx.Height),
		BlockHash:             fmt.Sprintf("0x%s", lib.BytesToString(blockHash)),
		TransactionHash:       fmt.Sprintf("0x%s", tx.TxHash),
		TransactionIndex:      fmt.Sprintf("0x%x", tx.Index),
		PseudoContractAddress: strings.ToLower(fsm.CNPYContractAddress),
		Amount:                fmt.Sprintf("0x%x", sendMessage.Amount),
		Topics: []string{transferEventFilterHash,
			fmt.Sprintf("0x%064s", lib.BytesToString(sendMessage.FromAddress)),
			fmt.Sprintf("0x%064s", lib.BytesToString(sendMessage.ToAddress))},
	}, nil
}

// passesAddressFilter() ensures the address is in the slice (nil slice means all)
func (s *Server) passesAddressFilter(addr []byte, addresses []string) (ok bool) {
	if addresses == nil {
		return true
	}
	for _, sender := range addresses {
		// remove the prefix
		padded := strings.TrimPrefix(sender, "0x")
		// take the last 40 hex characters (20 bytes)
		last20Hex := padded[len(padded)-40:]
		if strings.ToLower(last20Hex) == strings.ToLower(lib.BytesToString(addr)) {
			return true
		}
	}
	return
}

// ethFilters holds all active filters
var ethFilters = sync.Map{}

// startEthFilterExpireService() expires filters not read in ~ 5 minutes
func (s *Server) startEthFilterExpireService() {
	for range time.Tick(time.Minute) {
		ethFilters.Range(func(key, value any) bool {
			filter := value.(*ethFilter)
			// expire the filter after ~ 5 minutes of no read
			if filter.LastReadHeight.Load()+15 < s.controller.ChainHeight() {
				ethFilters.Delete(key)
			}
			return true
		})
	}
}

// getEthFilter() returns a filter by ID
func (s *Server) getEthFilter(id string) (filter *ethFilter, lastReadHeight uint64, err error) {
	// retrieve the filter from the list
	got, ok := ethFilters.Load(id)
	if !ok {
		return nil, 0, fmt.Errorf("filter with id %s not found", id)
	}
	// cast the filter
	filter = got.(*ethFilter)
	// get and update the last read height
	lastReadHeight = filter.LastReadHeight.Swap(s.controller.ChainHeight())
	// update the read height
	ethFilters.Store(id, filter)
	// return the filter
	return
}

// newEthFilter() creates a new filter and returns the id
func (s *Server) newEthFilter(params newFilterParams) (id string, err error) {
	uuid := make([]byte, 16)
	if _, err = rand.Read(uuid); err != nil {
		return "", err
	}
	id = "0x" + hex.EncodeToString(uuid[:])
	params.filter.FilterId = id
	// create the latest read height
	params.filter.LastReadHeight = &atomic.Uint64{}
	params.filter.LastReadHeight.Store(s.controller.ChainHeight())
	// add a new filter to the list
	ethFilters.Store(id, &params.filter)
	return
}

// Canopy-specific pseudo-nonce logic for eth_getTransactionCount: enables replay protection using
// created_at_height+timestamp, with per-address pending tx counters that decay over ±4320 blocks

var pseudoNonceMap = sync.Map{}    // [address] -> count
var latestHeight = atomic.Uint64{} // latest known height

// getAndIncPseudoNonce() retrieves and increments the pseudo-nonce
func getAndIncPseudoNonce(addr string) uint64 { return latestHeight.Load() + incPseudoNonce(addr) }

// incPseudoNonce() increments the pseudo-nonce and returns the old value
func incPseudoNonce(addr string) (old uint64) {
	v, _ := pseudoNonceMap.LoadOrStore(addr, new(atomic.Int64))
	got := v.(*atomic.Int64).Add(1) - 1
	return uint64(math.Max(float64(got), 0)) // defensive
}

// startEthPseudoNonceService() ensures if a new block is processed the nonce map is updated appropriately
func (s *Server) startEthPseudoNonceService() {
	for range time.Tick(time.Second) {
		currentHeight := s.controller.ChainHeight()
		prevHeight := latestHeight.Load()
		// only proceed if block height has increased
		if currentHeight > prevHeight {
			latestHeight.Store(currentHeight)
			// for each key in the map, decrement the count
			pseudoNonceMap.Range(func(key, value any) bool {
				count := value.(*atomic.Int64)
				// decrement count
				if count.Add(-1) <= 0 {
					// delete when count reaches 0
					pseudoNonceMap.Delete(key)
				}
				return true
			})
		}
	}
}

// Canopy only saves valid transactions in blocks - so the RPC mocks 'failed or dropped' txs for ethereum tooling compatibility
//
// - Node maps a txHash to the latest block height either when queried through eth_getTransactionReceipt or sent through eth_sendRawTransaction
// - After 15 blocks, if not found in the indexer - return status_failed
// - After appx 6 hours evict the txHash from the map
var pseudoPendingTxsMap = sync.Map{} // [hash] -> height

// shouldSimultePending() sets a pending tx in the map and returns the status of the transaction
func shouldSimultePending(txHash string) (pending bool) {
	storedHeight, found := pseudoPendingTxsMap.LoadOrStore(txHash, latestHeight.Load())
	// if not found after 5 minutes
	if found && storedHeight.(uint64)+15 < latestHeight.Load() {
		return false
	}
	return true
}

// startEthPendingTxsExpireService() evicts pending txs after 6 hours
func (s *Server) startEthPendingTxsExpireService() {
	for range time.Tick(time.Second) {
		// for each key in the map, decrement the count
		pseudoPendingTxsMap.Range(func(key, value any) bool {
			// evict if older than 6 hours
			if value.(uint64)+1080 < latestHeight.Load() {
				pseudoPendingTxsMap.Delete(key)
			}
			return true
		})
	}
}

// TYPES BELOW

// ethRPCRequest is the JSON RPC 2.0 request structure
type ethRPCRequest struct {
	ID      any             `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// ethRPCResponse is the JSON RPC 2.0 response structure
type ethRPCResponse struct {
	ID      any               `json:"id"`
	JSONRPC string            `json:"jsonrpc"`
	Result  any               `json:"result,omitempty"`
	Error   *ethereumRPCError `json:"error,omitempty"`
}

// ethereumRPCError is the expected JSON RPC 2.0 error structure
type ethereumRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ethRPCBlock matches the ethereum block header (which isn't exposed)
type ethRPCBlock struct {
	Number                hexutil.Big    `json:"number"`
	Hash                  common.Hash    `json:"hash"`
	ParentHash            common.Hash    `json:"parentHash"`
	Sha3Uncles            common.Hash    `json:"sha3Uncles"`
	LogsBloom             types.Bloom    `json:"logsBloom"`
	StateRoot             common.Hash    `json:"stateRoot"`
	Miner                 common.Address `json:"miner"`
	ExtraData             hexutil.Bytes  `json:"extraData"`
	GasLimit              hexutil.Uint64 `json:"gasLimit"`
	GasUsed               hexutil.Uint64 `json:"gasUsed"`
	Timestamp             hexutil.Uint64 `json:"timestamp"`
	TransactionsRoot      common.Hash    `json:"transactionsRoot"`
	ReceiptsRoot          common.Hash    `json:"receiptsRoot"`
	BaseFeePerGas         hexutil.Big    `json:"baseFeePerGas,omitempty"`
	WithdrawalsRoot       common.Hash    `json:"withdrawalsRoot,omitempty"`
	ParentBeaconBlockRoot common.Hash    `json:"parentBeaconBlockRoot,omitempty"`
	RequestsHash          common.Hash    `json:"requestsHash,omitempty"`
	Size                  hexutil.Uint64 `json:"size,omitempty"`
	Transactions          []interface{}  `json:"transactions"`
	Uncles                []common.Hash  `json:"uncles"`
}

// ethRPCTransaction matches the ethereum rpc transaction (which isn't exposed)
// - combines receipt and transaction into 1 object for simplicity
type ethRPCTransaction struct {
	BlockHash         common.Hash    `json:"blockHash"`
	BlockNumber       hexutil.Big    `json:"blockNumber"`
	From              common.Address `json:"from"`
	Gas               hexutil.Uint64 `json:"gas"`
	GasPrice          hexutil.Big    `json:"gasPrice"`
	GasFeeCap         hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap         hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Hash              common.Hash    `json:"hash"`
	Nonce             hexutil.Uint64 `json:"nonce"`
	To                common.Address `json:"to"`
	TransactionIndex  hexutil.Uint64 `json:"transactionIndex"`
	Value             hexutil.Big    `json:"value"`
	Type              hexutil.Uint64 `json:"type"`
	ChainID           hexutil.Big    `json:"chainId,omitempty"`
	Status            hexutil.Uint64 `json:"status"`
	CumulativeGasUsed hexutil.Uint64 `json:"cumulativeGasUsed"`
	Bloom             hexutil.Bytes  `json:"logsBloom"`
	Logs              []string       `json:"logs"`
	TxHash            common.Hash    `json:"transactionHash"`
	ContractAddress   common.Address `json:"contractAddress"`
	GasUsed           hexutil.Uint64 `json:"gasUsed"`
	EffectiveGasPrice hexutil.Big    `json:"effectiveGasPrice"`
}

// newFilterParams() is the params object for eth_newFilter()
type newFilterParams struct {
	StartBlock string    `json:"fromBlock"`
	EndBlock   string    `json:"toBlock"`
	Address    any       `json:"address"` // ignore as it's always going to be null or the pseudo-canopy contract address
	Topics     []any     `json:"topics"`
	filter     ethFilter // internal
}

// ethFilter is an internal object used to track active eth filters
type ethFilter struct {
	FilterId       string // hex string
	StartHeight    uint64
	EndHeight      uint64
	LastReadHeight *atomic.Uint64 // track the height last read for eth_getFilterChanges
	Blocks         bool
	PendingTxs     bool
	Topic          []string
	Sender         []string
	Recipient      []string
}

// ethGetLogsResponse is the response structure to an eth_getLogs request
type ethGetLogsResponse struct {
	LogIndex              string   `json:"logIndex"` // always tx index
	BlockNumber           string   `json:"blockNumber"`
	BlockHash             string   `json:"blockHash"`
	TransactionHash       string   `json:"transactionHash"`
	TransactionIndex      string   `json:"transactionIndex"`
	PseudoContractAddress string   `json:"address"`
	Amount                string   `json:"data"`   // amount
	Topics                []string `json:"topics"` // [0]=event signature, [1]=from (indexed), [2]=to (indexed)
}

// ethSyncingResponse is the response structure to an eth_syncing request
type ethSyncingResponse struct {
	StartingBlock hexutil.Uint64 `json:"startingBlock"`
	CurrentBlock  hexutil.Uint64 `json:"currentBlock"`
	HighestBlock  hexutil.Uint64 `json:"highestBlock"`
}

// HELPERS BELOW

// filterParamsFromArgs() creates newFilterParams from args
func filterParamsFromArgs(args []any) (params newFilterParams, err error) {
	params.filter.StartHeight, params.filter.EndHeight = uint64(1), uint64(0)
	// convert first argument into the params structure
	if len(args) > 0 {
		bz, e := json.Marshal(args[0])
		if e != nil {
			return newFilterParams{}, e
		}
		if err = json.Unmarshal(bz, &params); err != nil {
			return newFilterParams{}, fmt.Errorf("failed to unmarshal filter params: %w", err)
		}
	}
	// parse start block
	if params.StartBlock != "" {
		params.filter.StartHeight, err = parseBlockTag(params.StartBlock)
		if err != nil {
			return newFilterParams{}, err
		}
	}
	// parse end block
	if params.EndBlock != "" {
		params.filter.EndHeight, err = parseBlockTag(params.EndBlock)
		if err != nil {
			return newFilterParams{}, err
		}
	}
	// handle topics
	for i, topic := range params.Topics {
		res, e := stringArrayFromAny(topic)
		if e != nil {
			return newFilterParams{}, e
		}
		switch i {
		case 0:
			params.filter.Topic = res
		case 1:
			params.filter.Sender = res
		case 2:
			params.filter.Recipient = res
		}
	}
	// populate the default if empty
	if len(params.filter.Topic) == 0 {
		params.filter.Topic = []string{transferEventFilterHash}
	}
	return params, nil
}

// stringArrayFromAny() extracts a string array from the argument
func stringArrayFromAny(arg any) (res []string, err error) {
	if arg == nil {
		return nil, nil
	}
	switch t := arg.(type) {
	case string:
		return []string{strings.ToLower(t)}, nil
	case []any:
		for _, sub := range t {
			if s, ok := sub.(string); ok {
				res = append(res, strings.ToLower(s))
			} else {
				return nil, fmt.Errorf("invalid argument: expected string but got %T", sub)
			}
		}
	default:
		return nil, fmt.Errorf("invalid argument type: expected string or []any but got %T", arg)
	}
	return
}

// addressFromArgs() extracts the address from the first argument
func addressFromArgs(args []any) (crypto.AddressI, error) {
	str, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	return crypto.NewAddressFromString(cleanHex(str))
}

// bytesFromArgs() extracts a hash from the first argument
func bytesFromArgs(args []any) ([]byte, error) {
	str, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	return lib.StringToBytes(cleanHex(str))
}

// intFromArgs() extracts an integer from the first argument
func intFromArgs(args []any, position int) (int64, error) {
	str, err := strFromArgs(args, position)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(str, 0, 64)
}

// strFromArgs() extracts a string from the first argument
func strFromArgs(args []any, position int) (string, error) {
	if len(args) <= position {
		return "", errors.New("missing arguments")
	}
	str, ok := args[position].(string)
	if !ok {
		return "", errors.New("invalid argument format")
	}
	return str, nil
}

// boolFromArgs() extracts a bool from the second argument
func boolFromArgs(args []any) bool {
	// missing args check
	if len(args) < 2 {
		return false
	}
	got, _ := args[1].(bool)
	return got
}

// blockTagFromArgs() handles the optional block tag ethereum parameter
func blockTagFromArgs(args []any) (height uint64, err error) {
	// handle optional block tag
	blockTag := latestBlockTag
	if len(args) >= 2 {
		blockTag, err = strFromArgs(args, 1)
		if err != nil {
			return 0, err
		}
	}
	// convert blockTag to height
	return parseBlockTag(blockTag)
}

// ABI Encoder helpers below
const latestBlockTag, pendingBlockTag, safeBlockTag, finalizedBlockTag, earliestBlockTag = "latest", "pending", "safe", "finalized", "earliest"

var (
	ABIUint8Type, _   = abi.NewType("uint8", "", nil)
	ABIUint256Type, _ = abi.NewType("uint256", "", nil)
	ABIStringType, _  = abi.NewType("string", "", nil)
	ABIBoolType, _    = abi.NewType("bool", "", nil)
)

// revert() is a helper function for reverting ABI
func revert(error string) (encoded []byte, i lib.ErrorI) {
	revertData, err := pack(ABIStringType, error)
	if err != nil {
		return nil, lib.NewError(1, "ethereum", err.Error())
	}
	return append(common.FromHex("08c379a0"), revertData...), nil
}

// pack() is a helper function for packing ABI arguments
func pack(abiType abi.Type, args ...any) ([]byte, error) {
	return abi.Arguments{{Type: abiType}}.Pack(args...)
}

// parseBlockTag() converts Ethereum block tags to heights
func parseBlockTag(tag string) (uint64, error) {
	switch tag {
	case latestBlockTag, pendingBlockTag, safeBlockTag, finalizedBlockTag:
		return 0, nil
	case earliestBlockTag:
		return 1, nil
	}
	if strings.HasPrefix(tag, "0x") {
		n, err := strconv.ParseUint(tag[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid block number: %w", err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("unsupported block tag: %s", tag)
}

// parseAddressFromABI() extracts the address from the ABI data
func parseAddressFromABI(data []byte) (crypto.AddressI, lib.ErrorI) {
	if len(data) < 36 {
		return nil, lib.NewError(1, "ethereum", "malformed balanceOf input")
	}
	return crypto.NewAddress(common.BytesToAddress(data[16:36]).Bytes()), nil
}

// parseAddressFromABI() extracts the address and amount from the ABI data
func parseAddressAndAmountFromABI(data []byte) (crypto.AddressI, uint64, lib.ErrorI) {
	if len(data) != 68 {
		return nil, 0, lib.NewError(2, "ethereum", "malformed transfer input")
	}
	address, err := parseAddressFromABI(data)
	if err != nil {
		return nil, 0, err
	}
	// no upscaling on this amount - as decimals are specified in the 'pseudo-contract' logic
	amount := new(big.Int).SetBytes(data[36:68])
	if amount.Cmp(big.NewInt(0)) == -1 {
		return nil, 0, lib.NewError(2, "ethereum", "malformed transfer amount")
	}
	return address, amount.Uint64(), nil
}

// msgToSend() converts an any to MessageSend
func msgToSend(msg *anypb.Any) (*fsm.MessageSend, error) {
	a, err := lib.FromAny(msg)
	if err != nil {
		return nil, err
	}
	got, ok := a.(*fsm.MessageSend)
	if !ok {
		return nil, lib.ErrInvalidMessageCast()
	}
	return got, nil
}

// cleanHex() strips the 0x prefix from a hex string
func cleanHex(s string) string {
	s, _ = strings.CutPrefix(s, "0x")
	if s == "0" {
		s = "00"
	}
	return s
}

// The only supported event Transfer(address indexed from, address indexed to, uint256 value)
// 0xddf252... = Keccak-256(Transfer(address,address,uint256))
const transferEventFilterHash = `0xddf252ad0be3b87a1f7f5b73dfd3f49b8ff24c3e3a20713da75dd84c6d4c2c7c`

// Fake ERC20 byte code for CNPY
const CanopyPseudoContractByteCode = "0x608060405234801561000f575f5ffd5b506040518060400160405280600681526020017f43616e6f70790000000000000000000000000000000000000000000000000000815250600290816100549190610471565b506040518060400160405280600481526020017f434e505900000000000000000000000000000000000000000000000000000000815250600390816100999190610471565b50600660045f6101000a81548160ff021916908360ff1602179055506100ee3360045f9054906101000a900460ff16600a6100d491906106a8565b631e0a6e006100e391906106f2565b6100f360201b60201c565b610806565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610161576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101589061078d565b60405180910390fd5b8060015f82825461017291906107ab565b92505081905550805f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546101c491906107ab565b925050819055508173ffffffffffffffffffffffffffffffffffffffff165f73ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8360405161022891906107ed565b60405180910390a35050565b5f81519050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f60028204905060018216806102af57607f821691505b6020821081036102c2576102c161026b565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f600883026103247fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826102e9565b61032e86836102e9565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f61037261036d61036884610346565b61034f565b610346565b9050919050565b5f819050919050565b61038b83610358565b61039f61039782610379565b8484546102f5565b825550505050565b5f5f905090565b6103b66103a7565b6103c1818484610382565b505050565b5b818110156103e4576103d95f826103ae565b6001810190506103c7565b5050565b601f821115610429576103fa816102c8565b610403846102da565b81016020851015610412578190505b61042661041e856102da565b8301826103c6565b50505b505050565b5f82821c905092915050565b5f6104495f198460080261042e565b1980831691505092915050565b5f610461838361043a565b9150826002028217905092915050565b61047a82610234565b67ffffffffffffffff8111156104935761049261023e565b5b61049d8254610298565b6104a88282856103e8565b5f60209050601f8311600181146104d9575f84156104c7578287015190505b6104d18582610456565b865550610538565b601f1984166104e7866102c8565b5f5b8281101561050e578489015182556001820191506020850194506020810190506104e9565b8683101561052b5784890151610527601f89168261043a565b8355505b6001600288020188555050505b505050505050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f8160011c9050919050565b5f5f8291508390505b60018511156105c25780860481111561059e5761059d610540565b5b60018516156105ad5780820291505b80810290506105bb8561056d565b9450610582565b94509492505050565b5f826105da5760019050610695565b816105e7575f9050610695565b81600181146105fd576002811461060757610636565b6001915050610695565b60ff84111561061957610618610540565b5b8360020a9150848211156106305761062f610540565b5b50610695565b5060208310610133831016604e8410600b841016171561066b5782820a90508381111561066657610665610540565b5b610695565b6106788484846001610579565b9250905081840481111561068f5761068e610540565b5b81810290505b9392505050565b5f60ff82169050919050565b5f6106b282610346565b91506106bd8361069c565b92506106ea7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff84846105cb565b905092915050565b5f6106fc82610346565b915061070783610346565b925082820261071581610346565b9150828204841483151761072c5761072b610540565b5b5092915050565b5f82825260208201905092915050565b7f45524332303a206d696e7420746f20746865207a65726f2061646472657373005f82015250565b5f610777601f83610733565b915061078282610743565b602082019050919050565b5f6020820190508181035f8301526107a48161076b565b9050919050565b5f6107b582610346565b91506107c083610346565b92508282019050808211156107d8576107d7610540565b5b92915050565b6107e781610346565b82525050565b5f6020820190506108005f8301846107de565b92915050565b6109ef806108135f395ff3fe608060405234801561000f575f5ffd5b5060043610610060575f3560e01c806306fdde031461006457806318160ddd14610082578063313ce567146100a057806370a08231146100be57806395d89b41146100ee578063a9059cbb1461010c575b5f5ffd5b61006c61013c565b60405161007991906105a9565b60405180910390f35b61008a6101cc565b60405161009791906105e1565b60405180910390f35b6100a86101d5565b6040516100b59190610615565b60405180910390f35b6100d860048036038101906100d3919061068c565b6101ea565b6040516100e591906105e1565b60405180910390f35b6100f661022f565b60405161010391906105a9565b60405180910390f35b610126600480360381019061012191906106e1565b6102bf565b6040516101339190610739565b60405180910390f35b60606002805461014b9061077f565b80601f01602080910402602001604051908101604052809291908181526020018280546101779061077f565b80156101c25780601f10610199576101008083540402835291602001916101c2565b820191905f5260205f20905b8154815290600101906020018083116101a557829003601f168201915b5050505050905090565b5f600154905090565b5f60045f9054906101000a900460ff16905090565b5f5f5f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20549050919050565b60606003805461023e9061077f565b80601f016020809104026020016040519081016040528092919081815260200182805461026a9061077f565b80156102b55780601f1061028c576101008083540402835291602001916102b5565b820191905f5260205f20905b81548152906001019060200180831161029857829003601f168201915b5050505050905090565b5f5f3390506102cf8185856102da565b600191505092915050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610348576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161033f9061081f565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036103b6576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103ad906108ad565b60405180910390fd5b5f5f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905081811015610439576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016104309061093b565b60405180910390fd5b8181035f5f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2081905550815f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546104c79190610986565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8460405161052b91906105e1565b60405180910390a350505050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f61057b82610539565b6105858185610543565b9350610595818560208601610553565b61059e81610561565b840191505092915050565b5f6020820190508181035f8301526105c18184610571565b905092915050565b5f819050919050565b6105db816105c9565b82525050565b5f6020820190506105f45f8301846105d2565b92915050565b5f60ff82169050919050565b61060f816105fa565b82525050565b5f6020820190506106285f830184610606565b92915050565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61065b82610632565b9050919050565b61066b81610651565b8114610675575f5ffd5b50565b5f8135905061068681610662565b92915050565b5f602082840312156106a1576106a061062e565b5b5f6106ae84828501610678565b91505092915050565b6106c0816105c9565b81146106ca575f5ffd5b50565b5f813590506106db816106b7565b92915050565b5f5f604083850312156106f7576106f661062e565b5b5f61070485828601610678565b9250506020610715858286016106cd565b9150509250929050565b5f8115159050919050565b6107338161071f565b82525050565b5f60208201905061074c5f83018461072a565b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061079657607f821691505b6020821081036107a9576107a8610752565b5b50919050565b7f45524332303a207472616e736665722066726f6d20746865207a65726f2061645f8201527f6472657373000000000000000000000000000000000000000000000000000000602082015250565b5f610809602583610543565b9150610814826107af565b604082019050919050565b5f6020820190508181035f830152610836816107fd565b9050919050565b7f45524332303a207472616e7366657220746f20746865207a65726f20616464725f8201527f6573730000000000000000000000000000000000000000000000000000000000602082015250565b5f610897602383610543565b91506108a28261083d565b604082019050919050565b5f6020820190508181035f8301526108c48161088b565b9050919050565b7f45524332303a207472616e7366657220616d6f756e74206578636565647320625f8201527f616c616e63650000000000000000000000000000000000000000000000000000602082015250565b5f610925602683610543565b9150610930826108cb565b604082019050919050565b5f6020820190508181035f83015261095281610919565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610990826105c9565b915061099b836105c9565b92508282019050808211156109b3576109b2610959565b5b9291505056fea26469706673582212206d60a31558a9b9652ea881b1666d44b771e665cc38d4fb41fbb8357d8ab608f964736f6c634300081e0033"
