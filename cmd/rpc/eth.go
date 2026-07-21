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
	var raw json.RawMessage
	if ok := unmarshal(w, r, &raw); !ok {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(string(raw)), "[") {
		var requests []ethRPCRequest
		if err := json.Unmarshal(raw, &requests); err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		responses := make([]ethRPCResponse, 0, len(requests))
		for i := range requests {
			responses = append(responses, s.handleEthereumRPCRequest(&requests[i]))
		}
		write(w, responses, http.StatusOK)
		return
	}
	ptr := new(ethRPCRequest)
	if err := json.Unmarshal(raw, ptr); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, s.handleEthereumRPCRequest(ptr), http.StatusOK)
}

func (s *Server) handleEthereumRPCRequest(ptr *ethRPCRequest) ethRPCResponse {
	var err error
	var args []any
	if len(ptr.Params) != 0 && string(ptr.Params) != "null" {
		if err = json.Unmarshal(ptr.Params, &args); err != nil {
			return ethRPCResponse{
				ID:      ptr.ID,
				JSONRPC: "2.0",
				Error:   &ethereumRPCError{Code: -32603, Message: err.Error()},
			}
		}
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
	case `eth_protocolVersion`:
		ethResponse, err = s.EthProtocolVersion(args)
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
		err = ethMethodNotFound(ptr.Method)
	}
	return ethRPCResponse{
		ID:      ptr.ID,
		JSONRPC: "2.0",
		Result:  ethResponse,
		Error:   ethereumRPCErrorFrom(err),
	}
}

// startEthRPCService() runs the needed routines for the eth rpc wrapper
func (s *Server) startEthRPCService() {
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

// EthProtocolVersion() returns a compatibility protocol version string
func (s *Server) EthProtocolVersion(_ []any) (any, error) { return "0x41", nil }

// EthSyncing() returns the syncing status of the node
func (s *Server) EthSyncing(_ []any) (any, error) {
	if !s.controller.Syncing().Load() {
		return false, nil
	}
	currentBlock := s.currentEthBlockNumber()
	// return the syncing response
	return ethSyncingResponse{
		StartingBlock: hexutil.Uint64(1),
		CurrentBlock:  hexutil.Uint64(currentBlock),
		HighestBlock:  hexutil.Uint64(currentBlock),
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
	err = s.readOnlyState(height, func(state *fsm.StateMachine) (e lib.ErrorI) {
		// get the balance for the address
		balance, e := state.GetAccountSpendableBalance(address)
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

// EthGetTransactionCount() returns the next nonce for Ethereum accounts and a pseudo-nonce fallback for read-only addresses.
func (s *Server) EthGetTransactionCount(args []any) (any, error) {
	address, err := addressFromArgs(args)
	if err != nil {
		return nil, err
	}
	blockTag := latestBlockTag
	if len(args) >= 2 {
		blockTag, err = strFromArgs(args, 1)
		if err != nil {
			return nil, err
		}
	}
	return s.withStore(func(st *store.Store) (any, error) {
		base := s.ethereumNonceFloor(s.currentEthBlockNumber())
		hasMinedHistory := false
		if minedNonce, ok, nonceErr := s.latestMinedNonceForAddress(st, address); nonceErr != nil {
			return nil, nonceErr
		} else if ok {
			base = minedNonce
			hasMinedHistory = true
		}
		switch blockTag {
		case pendingBlockTag:
			pending := base
			maxNonce := s.maximumAcceptedEthereumNonce()
			if highestPending, ok, pendingErr := s.highestPendingNonceForAddress(st, address.String()); pendingErr != nil {
				return nil, pendingErr
			} else if ok && highestPending >= pending {
				if highestPending >= maxNonce {
					return nil, ethInvalidParams("no acceptable pending nonce available")
				}
				if highestPending == math.MaxUint64 {
					pending = math.MaxUint64
				} else {
					pending = highestPending + 1
				}
			}
			if pending > maxNonce {
				pending = maxNonce
			}
			return hexutil.Uint64(pending), nil
		case latestBlockTag, safeBlockTag, finalizedBlockTag:
			return hexutil.Uint64(base), nil
		case earliestBlockTag:
			return hexutil.Uint64(0), nil
		default:
			// Explicit historical block-number queries are compatibility-oriented: validate the tag but
			// return the latest confirmed nonce for Ethereum-derived accounts and the height fallback otherwise.
			height, parseErr := parseBlockTag(blockTag)
			if parseErr != nil {
				return nil, parseErr
			}
			if hasMinedHistory {
				return hexutil.Uint64(base), nil
			}
			return hexutil.Uint64(s.ethereumNonceFloor(height)), nil
		}
	})
}

// EthGetBlockTransactionCountByHash() returns the number of transactions in a block by hash
func (s *Server) EthGetBlockTransactionCountByHash(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		blockHash, err := bytesFromArgs(args)
		if err != nil {
			return nil, err
		}
		block, err := blockByHashOrNil(st, blockHash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return ethNullResult(), nil
		}
		return hexutil.Uint64(block.BlockHeader.NumTxs), nil
	})
}

// EthGetBlockTransactionCountByNumber() returns the number of transactions in a block by height
func (s *Server) EthGetBlockTransactionCountByNumber(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		blockHeight, err := s.blockHeightFromNumberArg(args, 0)
		if err != nil {
			return nil, err
		}
		block, err := blockByHeightOrNil(st, blockHeight)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return ethNullResult(), nil
		}
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
	var ethTx types.Transaction
	if err = ethTx.UnmarshalBinary(rawTx); err != nil {
		return nil, err
	}
	// convert it to a Canopy send transaction
	transaction, err := fsm.RLPToCanopyTransaction(rawTx)
	if err != nil {
		return nil, err
	}
	// ensure created height stays inside Canopy's accepted replay window
	if transaction.CreatedHeight < s.minimumAcceptedEthereumNonce() {
		return nil, lib.ErrInvalidTxHeight()
	}
	if transaction.CreatedHeight > s.maximumAcceptedEthereumNonce() {
		return nil, lib.ErrInvalidTxHeight()
	}
	// marshal the transaction to protobuf
	bz, err := lib.Marshal(transaction)
	if err != nil {
		return nil, err
	}
	// send transaction to controller
	if err = s.controller.SendTxMsgs([][]byte{bz}); err != nil {
		return nil, err
	}
	// track the pending transaction using the canonical ethereum tx hash
	txHashString := strings.ToLower(ethTx.Hash().Hex())
	registerPendingEthTx(txHashString, transaction)
	// return the transaction hash
	return txHashString, nil
}

// EthCall() simulates a call to a 'smart contract' for Canopy
func (s *Server) EthCall(args []any) (any, error) {
	if len(args) < 1 {
		return nil, ethInvalidParams("missing call arguments")
	}
	// handle the block tag
	height, err := blockTagFromArgs(args)
	if err != nil {
		return nil, err
	}
	// extract the call data
	callParams, ok := args[0].(map[string]any)
	if !ok {
		return nil, ethInvalidParams("invalid call argument format")
	}
	// get the sender address hex
	fromHex, ok := callParams["from"].(string)
	if !ok {
		fromHex = "0x" + strings.Repeat("0", 20)
	}
	// parse the `data` field from the call data
	dataHex, ok := callParams["data"].(string)
	if !ok {
		return nil, ethInvalidParams("invalid or missing 'data' field")
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
		return nil, ethInvalidParams(fmt.Sprintf("invalid hex: %v", err))
	}
	// validate the data length
	if len(data) < 4 {
		return nil, ethInvalidParams("insufficient data length")
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
				balance, e = state.GetAccountSpendableBalance(address)
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
			balance, e := state.GetAccountSpendableBalance(fromAddress)
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
		blockHash, e := bytesFromArgs(args)
		if e != nil {
			return nil, e
		}
		fullTxs := boolFromArgs(args)
		block, e := blockByHashOrNil(st, blockHash)
		if e != nil {
			return nil, e
		}
		if block == nil {
			return ethNullResult(), nil
		}
		return s.blockToEIP1559Block(block, fullTxs)
	})
}

// EthGetBlockByNumber() returns a dummy-ish block (based on the actual Canopy block) that is EIP-1559 compatible
func (s *Server) EthGetBlockByNumber(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		blockHeight, err := s.blockHeightFromNumberArg(args, 0)
		if err != nil {
			return nil, err
		}
		fullTxs := boolFromArgs(args)
		block, err := blockByHeightOrNil(st, blockHeight)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return ethNullResult(), nil
		}
		return s.blockToEIP1559Block(block, fullTxs)
	})
}

// EthGetTransactionByHash() returns a canonical Ethereum transaction object for mined or pending transactions
func (s *Server) EthGetTransactionByHash(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		hashString, err := normalizedHashFromArgs(args)
		if err != nil {
			return nil, err
		}
		tx, block, err := s.findIndexedTxByHash(st, hashString)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			clearPendingEthTx(hashString)
			return s.txToEthTransaction(block, tx, false)
		}
		if pending := s.findPendingEthTx(hashString); pending != nil {
			return s.pendingTxToEthTransaction(pending), nil
		}
		return ethNullResult(), nil
	})
}

// EthGetTransactionByBlockHashAndIndex() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionByBlockHashAndIndex(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		blockHash, e := bytesFromArgs(args)
		if e != nil {
			return nil, e
		}
		index, e := intFromArgs(args, 1)
		if e != nil {
			return nil, e
		}
		block, e := blockByHashOrNil(st, blockHash)
		if e != nil {
			return nil, e
		}
		tx, ok := txAtBlockIndex(block, index)
		if !ok {
			return ethNullResult(), nil
		}
		return s.txToEthTransaction(block, tx, false)
	})
}

// EthGetTransactionByBlockNumAndIndex() returns an EIP-1559 compatible tx + receipt
func (s *Server) EthGetTransactionByBlockNumAndIndex(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		blockHeight, e := s.blockHeightFromNumberArg(args, 0)
		if e != nil {
			return nil, e
		}
		index, e := intFromArgs(args, 1)
		if e != nil {
			return nil, e
		}
		block, e := blockByHeightOrNil(st, blockHeight)
		if e != nil {
			return nil, e
		}
		tx, ok := txAtBlockIndex(block, index)
		if !ok {
			return ethNullResult(), nil
		}
		return s.txToEthTransaction(block, tx, false)
	})
}

// EthGetTransactionReceipt() returns a canonical Ethereum receipt only for mined transactions
func (s *Server) EthGetTransactionReceipt(args []any) (any, error) {
	return s.withStore(func(st *store.Store) (any, error) {
		hashString, err := normalizedHashFromArgs(args)
		if err != nil {
			return nil, err
		}
		tx, block, err := s.findIndexedTxByHash(st, hashString)
		if err != nil {
			return nil, err
		}
		if tx == nil {
			return ethNullResult(), nil
		}
		clearPendingEthTx(hashString)
		return s.txToEthReceipt(block, tx)
	})
}

// EthGetUncleByBlockHashAndIndex() returns null (no uncles) as expected
func (s *Server) EthGetUncleByBlockHashAndIndex(_ []any) (any, error) {
	return ethNullResult(), nil
}

// EthGetUncleByBlockNumAndIndex() returns null (no uncles) as expected
func (s *Server) EthGetUncleByBlockNumAndIndex(_ []any) (any, error) {
	return ethNullResult(), nil
}

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
	return s.ethGetLogs(filter, 0)
}

// EthGetLogs() returns an array of all logs matching the passed filter argument
func (s *Server) EthGetLogs(args []any) (any, error) {
	// convert the args to filter params
	params, err := filterParamsFromArgs(args)
	if err != nil {
		return nil, err
	}
	// call ethGetLogs
	return s.ethGetLogs(&params.filter, 0)
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
	transactions := make([]interface{}, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		if fullTx {
			eip1559Tx, e := s.txToEthTransaction(block, tx, false)
			if e != nil {
				return ethRPCBlock{}, e
			}
			transactions = append(transactions, eip1559Tx)
		} else {
			transactions = append(transactions, ethHashStringFromTxResult(tx))
		}
	}
	// create the EIP-1559 block
	return ethRPCBlock{
		Number:                hexutil.Big(*big.NewInt(int64(block.BlockHeader.Height))),
		Difficulty:            hexutil.Big(*big.NewInt(0)),
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

// txToEthTransaction() converts a Canopy transaction into a canonical Ethereum transaction object
func (s *Server) txToEthTransaction(b *lib.BlockResult, tx *lib.TxResult, pending bool) (any, error) {
	if ethTx, ok := ethTransactionFromCanopyTx(tx.Transaction); ok {
		return marshalCanonicalEthTransaction(ethTx, common.BytesToAddress(tx.Sender), b, tx, pending)
	}
	hash := common.HexToHash(ethHashStringFromTxResult(tx))
	from := common.BytesToAddress(tx.Sender)
	gasPrice := big.NewInt(ethGasPrice)
	gasFeeCap := big.NewInt(ethGasPrice)
	gasTipCap := big.NewInt(0)
	gas := uint64(tx.Transaction.Fee)
	nonce := tx.Transaction.CreatedHeight
	input := []byte{}
	value := fsm.UpscaleTo18Decimals(sendAmountFromTxResult(tx))
	chainID := big.NewInt(int64(evmChainIDFromTx(tx.Transaction)))
	var to *common.Address
	if len(tx.Recipient) != 0 {
		recipient := common.BytesToAddress(tx.Recipient)
		to = &recipient
	}
	result := ethRPCTransaction{
		BlockHash:        nil,
		BlockNumber:      nil,
		From:             from,
		Gas:              hexutil.Uint64(gas),
		GasPrice:         (*hexutil.Big)(gasPrice),
		GasFeeCap:        (*hexutil.Big)(gasFeeCap),
		GasTipCap:        (*hexutil.Big)(gasTipCap),
		Hash:             hash,
		Input:            hexutil.Bytes(input),
		Nonce:            hexutil.Uint64(nonce),
		To:               to,
		TransactionIndex: nil,
		Value:            hexutil.Big(*value),
		Type:             types.DynamicFeeTxType,
		ChainID:          (*hexutil.Big)(chainID),
	}
	if !pending {
		blockHash := common.BytesToHash(b.BlockHeader.Hash)
		blockNumber := hexutil.Big(*big.NewInt(int64(tx.Height)))
		txIndex := hexutil.Uint64(tx.Index)
		result.BlockHash = &blockHash
		result.BlockNumber = &blockNumber
		result.TransactionIndex = &txIndex
	}
	return result, nil
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
				transaction := new(lib.Transaction)
				if err := lib.Unmarshal(tx, transaction); err == nil {
					if ethHash := ethHashStringFromTransaction(transaction); ethHash != "" {
						strResults = append(strResults, ethHash)
						continue
					}
				}
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
		if !filterSupportsPseudoTransferLogs(filter) {
			return make([]ethRPCLog, 0), nil
		}
		if filter.BlockHash != "" {
			if lastReadHeight != 0 {
				return make([]ethRPCLog, 0), nil
			}
			blockHash, err := lib.StringToBytes(cleanHex(filter.BlockHash))
			if err != nil {
				return nil, ethInvalidParams(err.Error())
			}
			block, err := st.GetBlockByHash(blockHash)
			if err != nil {
				if isNotFoundErr(err) {
					return make([]ethRPCLog, 0), nil
				}
				return nil, err
			}
			if isNilBlock(block) {
				return make([]ethRPCLog, 0), nil
			}
			response := make([]ethRPCLog, 0)
			for _, tx := range block.Transactions {
				if tx.MessageType != fsm.MessageSendName ||
					!s.passesAddressFilter(tx.Sender, filter.Sender) ||
					!s.passesAddressFilter(tx.Recipient, filter.Recipient) {
					continue
				}
				converted, e := s.txToGetLogsResp(block.BlockHeader.Hash, tx)
				if e != nil {
					return nil, e
				}
				response = append(response, converted...)
			}
			return response, nil
		}
		// set the start height
		startHeight := filter.StartHeight
		if lastReadHeight != 0 {
			startHeight = lastReadHeight
		} else if startHeight == 0 {
			startHeight = s.currentEthBlockNumber()
		}
		// set height at latest block
		endHeight := filter.EndHeight
		if endHeight == 0 {
			endHeight = s.currentEthBlockNumber()
		}
		if startHeight > endHeight {
			return make([]ethRPCLog, 0), nil
		}
		// parse blocks looking for an appropriate response
		response := make([]ethRPCLog, 0)
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
				response = append(response, converted...)
			}
		}
		return response, nil
	})
}

// txToGetLogsResp() converts a send message into canonical Ethereum log objects
func (s *Server) txToGetLogsResp(blockHash []byte, tx *lib.TxResult) ([]ethRPCLog, error) {
	gethLogs, err := ethLogsFromTx(blockHash, tx)
	if err != nil {
		return nil, err
	}
	response := make([]ethRPCLog, 0, len(gethLogs))
	for _, log := range gethLogs {
		response = append(response, ethRPCLog{
			Address:     log.Address,
			Topics:      log.Topics,
			Data:        hexutil.Bytes(log.Data),
			BlockNumber: hexutil.Uint64(log.BlockNumber),
			TxHash:      log.TxHash,
			TxIndex:     hexutil.Uint64(log.TxIndex),
			BlockHash:   log.BlockHash,
			Index:       hexutil.Uint64(log.Index),
			Removed:     log.Removed,
		})
	}
	return response, nil
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

// Canopy only saves valid transactions in blocks, so the RPC keeps a lightweight local pending cache for
// canonical eth_sendRawTransaction / eth_getTransactionByHash behavior until the tx is mined or expires.
var pseudoPendingTxsMap = sync.Map{}   // [ethHash] -> *ethPendingTx
var pendingSenderNonceMap = sync.Map{} // [sender:nonce] -> ethHash

const ethPendingTxTTL = 15 * time.Minute

// startEthPendingTxsExpireService() evicts locally tracked pending txs after a short wall-clock grace period.
func (s *Server) startEthPendingTxsExpireService() {
	for range time.Tick(time.Second) {
		pseudoPendingTxsMap.Range(func(key, value any) bool {
			pending := value.(*ethPendingTx)
			if pendingTxExpired(pending) {
				clearPendingEthTx(key.(string))
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
	Difficulty            hexutil.Big    `json:"difficulty"`
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

// ethRPCTransaction matches the Ethereum transaction object
type ethRPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice,omitempty"`
	GasFeeCap        *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            hexutil.Big     `json:"value"`
	Type             hexutil.Uint64  `json:"type"`
	ChainID          *hexutil.Big    `json:"chainId,omitempty"`
}

// ethRPCReceipt matches the Ethereum receipt object
type ethRPCReceipt struct {
	Type              hexutil.Uint64  `json:"type,omitempty"`
	Status            hexutil.Uint64  `json:"status"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	Bloom             types.Bloom     `json:"logsBloom"`
	Logs              []ethRPCLog     `json:"logs"`
	TxHash            common.Hash     `json:"transactionHash"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	ContractAddress   *common.Address `json:"contractAddress"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	EffectiveGasPrice *hexutil.Big    `json:"effectiveGasPrice,omitempty"`
	BlockHash         common.Hash     `json:"blockHash"`
	BlockNumber       hexutil.Big     `json:"blockNumber"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
}

// ethRPCLog matches the Ethereum log object
type ethRPCLog struct {
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        hexutil.Bytes  `json:"data"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	TxHash      common.Hash    `json:"transactionHash"`
	TxIndex     hexutil.Uint64 `json:"transactionIndex"`
	BlockHash   common.Hash    `json:"blockHash"`
	Index       hexutil.Uint64 `json:"logIndex"`
	Removed     bool           `json:"removed"`
}

// newFilterParams() is the params object for eth_newFilter()
type newFilterParams struct {
	StartBlock string    `json:"fromBlock"`
	EndBlock   string    `json:"toBlock"`
	BlockHash  string    `json:"blockHash"`
	Address    any       `json:"address"`
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
	BlockHash      string
	Address        []string
	Topic          []string
	Sender         []string
	Recipient      []string
}

// ethSyncingResponse is the response structure to an eth_syncing request
type ethSyncingResponse struct {
	StartingBlock hexutil.Uint64 `json:"startingBlock"`
	CurrentBlock  hexutil.Uint64 `json:"currentBlock"`
	HighestBlock  hexutil.Uint64 `json:"highestBlock"`
}

// HELPERS BELOW

type ethPendingTx struct {
	AcceptedAt time.Time
	Tx         *lib.Transaction
}

func ethNullResult() json.RawMessage { return json.RawMessage("null") }

// isNilBlock() reports whether the fetched block result is usable.
func isNilBlock(block *lib.BlockResult) bool {
	return block == nil || block.BlockHeader == nil || len(block.BlockHeader.Hash) == 0
}

// isNotFoundErr() identifies backend lookup misses that should serialize as null/empty results.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func blockByHashOrNil(st *store.Store, hash []byte) (*lib.BlockResult, error) {
	block, err := st.GetBlockByHash(hash)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if isNilBlock(block) {
		return nil, nil
	}
	return block, nil
}

func blockByHeightOrNil(st *store.Store, height uint64) (*lib.BlockResult, error) {
	block, err := st.GetBlockByHeight(height)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if isNilBlock(block) {
		return nil, nil
	}
	return block, nil
}

func txAtBlockIndex(block *lib.BlockResult, index int64) (*lib.TxResult, bool) {
	if isNilBlock(block) || index < 0 || len(block.Transactions) <= int(index) {
		return nil, false
	}
	return block.Transactions[index], true
}

type ethRPCMethodError struct {
	code    int
	message string
}

func (e *ethRPCMethodError) Error() string { return e.message }

// ethMethodNotFound() wraps an unsupported method in the expected JSON-RPC error code.
func ethMethodNotFound(method string) error {
	return &ethRPCMethodError{
		code:    -32601,
		message: fmt.Sprintf("the method %s does not exist/is not available", method),
	}
}

// ethInvalidParams() wraps invalid input in the expected JSON-RPC error code.
func ethInvalidParams(message string) error {
	return &ethRPCMethodError{code: -32602, message: message}
}

// ethereumRPCErrorFrom() converts internal errors into JSON-RPC error payloads.
func ethereumRPCErrorFrom(err error) *ethereumRPCError {
	if err == nil {
		return nil
	}
	var rpcErr *ethRPCMethodError
	if errors.As(err, &rpcErr) {
		return &ethereumRPCError{Code: rpcErr.code, Message: rpcErr.message}
	}
	return &ethereumRPCError{Code: -32603, Message: err.Error()}
}

// normalizedHashFromArgs() returns a lower-cased 0x-prefixed hash argument.
func normalizedHashFromArgs(args []any) (string, error) {
	hash, err := strFromArgs(args, 0)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(hash, "0x") {
		hash = "0x" + hash
	}
	return strings.ToLower(hash), nil
}

// evmChainIDFromTx() returns the packed EVM chain id for a Canopy transaction.
func evmChainIDFromTx(tx *lib.Transaction) uint64 {
	return fsm.CanopyIdsToEVMChainId(tx.ChainId, tx.NetworkId)
}

// marshalCanonicalEthTransaction() overlays Ethereum RPC inclusion fields onto a signed tx JSON object.
func marshalCanonicalEthTransaction(ethTx *types.Transaction, from common.Address, block *lib.BlockResult, tx *lib.TxResult, pending bool) (map[string]any, error) {
	bz, err := ethTx.MarshalJSON()
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	if err = json.Unmarshal(bz, &result); err != nil {
		return nil, err
	}
	result["from"] = from.Hex()
	// Ethereum RPC transaction objects report gasPrice for dynamic-fee txs too; MetaMask expects it during polling.
	if result["gasPrice"] == nil {
		result["gasPrice"] = (*hexutil.Big)(ethTx.GasPrice())
	}
	if pending {
		result["blockHash"] = nil
		result["blockNumber"] = nil
		result["transactionIndex"] = nil
		return result, nil
	}
	result["blockHash"] = common.BytesToHash(block.BlockHeader.Hash).Hex()
	result["blockNumber"] = hexutil.EncodeUint64(tx.Height)
	result["transactionIndex"] = hexutil.EncodeUint64(tx.Index)
	return result, nil
}

// ethTransactionFromCanopyTx() decodes the original signed Ethereum tx from an RLP-backed Canopy tx.
func ethTransactionFromCanopyTx(tx *lib.Transaction) (*types.Transaction, bool) {
	if tx == nil || tx.Memo != fsm.RLPIndicator || tx.Signature == nil || len(tx.Signature.Signature) == 0 {
		return nil, false
	}
	var ethTx types.Transaction
	if err := ethTx.UnmarshalBinary(tx.Signature.Signature); err != nil {
		return nil, false
	}
	return &ethTx, true
}

// ethHashStringFromTransaction() returns the canonical Ethereum tx hash for an RLP-backed tx.
func ethHashStringFromTransaction(tx *lib.Transaction) string {
	if ethTx, ok := ethTransactionFromCanopyTx(tx); ok {
		return strings.ToLower(ethTx.Hash().Hex())
	}
	return ""
}

// ethHashStringFromTxResult() returns the Ethereum hash when available, else the stored Canopy tx hash.
func ethHashStringFromTxResult(tx *lib.TxResult) string {
	if hash := ethHashStringFromTransaction(tx.Transaction); hash != "" {
		return hash
	}
	return "0x" + tx.TxHash
}

// sendAmountFromTxResult() extracts the transfer amount for send transactions.
func sendAmountFromTxResult(tx *lib.TxResult) uint64 {
	if tx.MessageType != fsm.MessageSendName {
		return 0
	}
	sendMsg, err := msgToSend(tx.Transaction.Msg)
	if err != nil {
		return 0
	}
	return sendMsg.Amount
}

// gasLimitFromTxResult() reports the Ethereum gas limit for RLP-backed txs or the Canopy fee field otherwise.
func gasLimitFromTxResult(tx *lib.TxResult) uint64 {
	if ethTx, ok := ethTransactionFromCanopyTx(tx.Transaction); ok {
		return ethTx.Gas()
	}
	return tx.Transaction.Fee
}

// effectiveGasPriceFromEthTx() returns the gas price Canopy actually charges for the Ethereum tx.
func effectiveGasPriceFromEthTx(ethTx *types.Transaction) *big.Int {
	return new(big.Int).Set(ethTx.GasPrice())
}

// cumulativeGasUsedForTx() sums gas usage through the target tx index within the block.
func cumulativeGasUsedForTx(block *lib.BlockResult, index uint64) uint64 {
	var total uint64
	for _, blockTx := range block.Transactions {
		total += gasLimitFromTxResult(blockTx)
		if blockTx.Index == index {
			break
		}
	}
	return total
}

// senderFromTransaction() derives the sender address from the transaction signature payload.
func senderFromTransaction(tx *lib.Transaction) string {
	if tx == nil || tx.Signature == nil {
		return ""
	}
	pubKey, err := crypto.NewPublicKeyFromBytes(tx.Signature.PublicKey)
	if err == nil {
		return pubKey.Address().String()
	}
	return ""
}

// pendingTxExpired() reports whether a locally tracked pending tx has exceeded the wallet-compatibility grace window.
func pendingTxExpired(pending *ethPendingTx) bool {
	return pending == nil || time.Since(pending.AcceptedAt) > ethPendingTxTTL
}

// pendingSenderNonceKey() keys pending entries by sender and nonce for replacement cleanup.
func pendingSenderNonceKey(sender string, nonce uint64) string {
	return fmt.Sprintf("%s:%d", strings.ToLower(sender), nonce)
}

// registerPendingEthTx() stores a locally accepted pending Ethereum tx for short-lived lookup compatibility.
func registerPendingEthTx(hash string, tx *lib.Transaction) {
	hash = strings.ToLower(hash)
	if tx != nil {
		if sender := senderFromTransaction(tx); sender != "" {
			key := pendingSenderNonceKey(sender, tx.CreatedHeight)
			if existing, loaded := pendingSenderNonceMap.Load(key); loaded {
				oldHash := strings.ToLower(existing.(string))
				if oldHash != hash {
					pseudoPendingTxsMap.Delete(oldHash)
				}
			}
			pendingSenderNonceMap.Store(key, hash)
		}
	}
	pseudoPendingTxsMap.Store(hash, &ethPendingTx{AcceptedAt: time.Now(), Tx: tx})
}

// clearPendingEthTx() removes a pending tx from the local lookup maps.
func clearPendingEthTx(hash string) {
	hash = strings.ToLower(hash)
	if pending, ok := pseudoPendingTxsMap.Load(hash); ok {
		tx := pending.(*ethPendingTx).Tx
		if tx != nil {
			if sender := senderFromTransaction(tx); sender != "" {
				pendingSenderNonceMap.Delete(pendingSenderNonceKey(sender, tx.CreatedHeight))
			}
		}
	}
	pseudoPendingTxsMap.Delete(hash)
}

// highestPendingNonceForAddress() returns the highest still-live pending nonce for the sender.
func (s *Server) highestPendingNonceForAddress(st *store.Store, address string) (highest uint64, ok bool, err error) {
	address = strings.ToLower(address)
	pseudoPendingTxsMap.Range(func(key, value any) bool {
		hash := strings.ToLower(key.(string))
		pending := value.(*ethPendingTx)
		if pendingTxExpired(pending) {
			clearPendingEthTx(hash)
			return true
		}
		if tx, _, lookupErr := s.findIndexedTxByHash(st, hash); lookupErr != nil {
			err = lookupErr
			return false
		} else if tx != nil {
			clearPendingEthTx(hash)
			return true
		}
		if pending.Tx == nil || strings.ToLower(senderFromTransaction(pending.Tx)) != address {
			return true
		}
		nonce := pending.Tx.CreatedHeight
		if !ok || nonce > highest {
			highest, ok = nonce, true
		}
		return true
	})
	if err != nil {
		return 0, false, err
	}
	s.controller.Mempool.L.Lock()
	defer s.controller.Mempool.L.Unlock()
	for _, txBz := range s.controller.Mempool.GetTransactions(math.MaxUint64) {
		tx := new(lib.Transaction)
		if err := lib.Unmarshal(txBz, tx); err != nil {
			continue
		}
		if strings.ToLower(senderFromTransaction(tx)) != address {
			continue
		}
		nonce := tx.CreatedHeight
		if !ok || nonce > highest {
			highest, ok = nonce, true
		}
	}
	return
}

// findPendingEthTx() resolves a pending tx by hash from the live mempool, with a short local grace fallback after eviction.
func (s *Server) findPendingEthTx(hash string) *ethPendingTx {
	hash = strings.ToLower(hash)
	s.controller.Mempool.L.Lock()
	defer s.controller.Mempool.L.Unlock()
	for _, txBz := range s.controller.Mempool.GetTransactions(math.MaxUint64) {
		tx := new(lib.Transaction)
		if err := lib.Unmarshal(txBz, tx); err != nil {
			continue
		}
		if ethHashStringFromTransaction(tx) != hash {
			continue
		}
		return &ethPendingTx{AcceptedAt: time.Now(), Tx: tx}
	}
	if pending, ok := pseudoPendingTxsMap.Load(hash); ok {
		cached := pending.(*ethPendingTx)
		if !pendingTxExpired(cached) {
			return cached
		}
		clearPendingEthTx(hash)
	}
	return nil
}

// pendingTxToEthTransaction() serializes a pending tx using the Ethereum transaction object shape.
func (s *Server) pendingTxToEthTransaction(pending *ethPendingTx) any {
	ethTx, ok := ethTransactionFromCanopyTx(pending.Tx)
	if !ok {
		return ethNullResult()
	}
	result, _ := marshalCanonicalEthTransaction(ethTx, common.HexToAddress("0x"+senderFromTransaction(pending.Tx)), nil, nil, true)
	return result
}

// currentEthBlockNumber() returns the Ethereum-facing latest block number.
func (s *Server) currentEthBlockNumber() uint64 {
	height := s.controller.ChainHeight()
	if height == 0 {
		return 0
	}
	return height - 1
}

// minimumAcceptedEthereumNonce() returns the lowest nonce/createdHeight currently accepted by Canopy.
func (s *Server) minimumAcceptedEthereumNonce() uint64 {
	height := s.currentEthBlockNumber()
	if height <= fsm.BlockAcceptanceRange {
		return 0
	}
	return height - fsm.BlockAcceptanceRange
}

// maximumAcceptedEthereumNonce() returns the highest nonce/createdHeight currently accepted by Canopy.
func (s *Server) maximumAcceptedEthereumNonce() uint64 {
	height := s.currentEthBlockNumber()
	if height > math.MaxUint64-fsm.BlockAcceptanceRange {
		return math.MaxUint64
	}
	return height + fsm.BlockAcceptanceRange
}

// ethereumNonceFloor() returns the stable nonce floor used for Ethereum-compatible RPC responses.
func (s *Server) ethereumNonceFloor(height uint64) uint64 {
	return height - (height % fsm.BlockAcceptanceRange)
}

// latestMinedNonceForAddress() returns the next confirmed nonce for an address when it has mined Ethereum-backed tx history.
func (s *Server) latestMinedNonceForAddress(st *store.Store, address crypto.AddressI) (uint64, bool, error) {
	nonce, ok, err := st.GetLatestMinedEthereumNonce(address)
	if err != nil || !ok {
		return 0, ok, err
	}
	nextNonce := nonce + 1
	if floorNonce := s.ethereumNonceFloor(s.currentEthBlockNumber()); nextNonce < floorNonce {
		nextNonce = floorNonce
	}
	return nextNonce, true, nil
}

// blockHeightFromNumberArg() resolves block-number method arguments to indexed Canopy block heights.
func (s *Server) blockHeightFromNumberArg(args []any, position int) (uint64, error) {
	blockTag, err := strFromArgs(args, position)
	if err != nil {
		return 0, err
	}
	switch blockTag {
	case latestBlockTag, pendingBlockTag, safeBlockTag, finalizedBlockTag:
		return s.currentEthBlockNumber(), nil
	case earliestBlockTag:
		return 0, nil
	}
	return parseBlockTag(blockTag)
}

// findIndexedTxByHash() resolves a mined tx by stored Canopy hash or persisted Ethereum-hash alias.
func (s *Server) findIndexedTxByHash(st *store.Store, hash string) (*lib.TxResult, *lib.BlockResult, error) {
	txHash, err := lib.StringToBytes(cleanHex(hash))
	if err == nil {
		tx, txErr := st.GetTxByHash(txHash)
		if txErr == nil && tx.TxHash != "" {
			block, blockErr := st.GetBlockByHeight(tx.Height)
			if blockErr == nil && !isNilBlock(block) {
				return tx, block, nil
			}
			// Treat a partially indexed mined tx as unresolved so callers can fall back to pending/null
			// instead of surfacing a transient backend error during MetaMask polling.
			if blockErr == nil || isNotFoundErr(blockErr) {
				return nil, nil, nil
			}
			return nil, nil, blockErr
		}
	}
	return nil, nil, nil
}

// txToEthReceipt() converts a Canopy tx result into an Ethereum receipt response object.
func (s *Server) txToEthReceipt(block *lib.BlockResult, tx *lib.TxResult) (ethRPCReceipt, error) {
	logs, err := s.txToGetLogsResp(block.BlockHeader.Hash, tx)
	if err != nil {
		return ethRPCReceipt{}, err
	}
	gethLogs, err := ethLogsFromTx(block.BlockHeader.Hash, tx)
	if err != nil {
		return ethRPCReceipt{}, err
	}
	receipt := &types.Receipt{Logs: gethLogs}
	receipt.Bloom = types.CreateBloom(receipt)
	gasPrice := big.NewInt(ethGasPrice)
	txType := uint64(types.DynamicFeeTxType)
	from := common.BytesToAddress(tx.Sender)
	var to *common.Address
	if len(tx.Recipient) != 0 {
		recipient := common.BytesToAddress(tx.Recipient)
		to = &recipient
	}
	if ethTx, ok := ethTransactionFromCanopyTx(tx.Transaction); ok {
		gasPrice = effectiveGasPriceFromEthTx(ethTx)
		txType = uint64(ethTx.Type())
		if recipient := ethTx.To(); recipient != nil {
			recipientCopy := *recipient
			to = &recipientCopy
		} else {
			to = nil
		}
	}
	return ethRPCReceipt{
		Type:              hexutil.Uint64(txType),
		Status:            hexutil.Uint64(types.ReceiptStatusSuccessful),
		CumulativeGasUsed: hexutil.Uint64(cumulativeGasUsedForTx(block, tx.Index)),
		Bloom:             receipt.Bloom,
		Logs:              logs,
		TxHash:            common.HexToHash(ethHashStringFromTxResult(tx)),
		From:              from,
		To:                to,
		ContractAddress:   nil,
		GasUsed:           hexutil.Uint64(gasLimitFromTxResult(tx)),
		EffectiveGasPrice: (*hexutil.Big)(gasPrice),
		BlockHash:         common.BytesToHash(block.BlockHeader.Hash),
		BlockNumber:       hexutil.Big(*big.NewInt(int64(tx.Height))),
		TransactionIndex:  hexutil.Uint64(tx.Index),
	}, nil
}

// ethLogsFromTx() synthesizes supported pseudo-token Transfer logs for a tx result.
func ethLogsFromTx(blockHash []byte, tx *lib.TxResult) ([]*types.Log, error) {
	if tx.MessageType != fsm.MessageSendName {
		return nil, nil
	}
	sendMessage, err := msgToSend(tx.Transaction.Msg)
	if err != nil {
		return nil, err
	}
	amount, err := pack(ABIUint256Type, new(big.Int).SetUint64(sendMessage.Amount))
	if err != nil {
		return nil, err
	}
	return []*types.Log{{
		Address:     common.HexToAddress(fsm.CNPYContractAddress),
		Topics:      []common.Hash{common.HexToHash(transferEventFilterHash), common.BytesToHash(common.LeftPadBytes(sendMessage.FromAddress, 32)), common.BytesToHash(common.LeftPadBytes(sendMessage.ToAddress, 32))},
		Data:        amount,
		BlockNumber: tx.Height,
		TxHash:      common.HexToHash(ethHashStringFromTxResult(tx)),
		TxIndex:     uint(tx.Index),
		BlockHash:   common.BytesToHash(blockHash),
		Index:       uint(tx.Index),
		Removed:     false,
	}}, nil
}

// filterSupportsPseudoTransferLogs() rejects log filters outside the supported pseudo-token Transfer subset.
func filterSupportsPseudoTransferLogs(filter *ethFilter) bool {
	if len(filter.Address) != 0 {
		supported := false
		for _, address := range filter.Address {
			if strings.EqualFold(address, fsm.CNPYContractAddress) {
				supported = true
				break
			}
		}
		if !supported {
			return false
		}
	}
	if len(filter.Topic) != 0 {
		supported := false
		for _, topic := range filter.Topic {
			if strings.EqualFold(topic, transferEventFilterHash) {
				supported = true
				break
			}
		}
		if !supported {
			return false
		}
	}
	return true
}

// filterParamsFromArgs() creates newFilterParams from args
func filterParamsFromArgs(args []any) (params newFilterParams, err error) {
	params.filter.StartHeight, params.filter.EndHeight = uint64(0), uint64(0)
	// convert first argument into the params structure
	if len(args) > 0 {
		bz, e := json.Marshal(args[0])
		if e != nil {
			return newFilterParams{}, e
		}
		if err = json.Unmarshal(bz, &params); err != nil {
			return newFilterParams{}, ethInvalidParams(fmt.Sprintf("failed to unmarshal filter params: %v", err))
		}
	}
	if params.BlockHash != "" {
		if params.StartBlock != "" || params.EndBlock != "" {
			return newFilterParams{}, ethInvalidParams("blockHash is mutually exclusive with fromBlock/toBlock")
		}
		if _, err = lib.StringToBytes(cleanHex(params.BlockHash)); err != nil {
			return newFilterParams{}, ethInvalidParams(err.Error())
		}
		params.filter.BlockHash = strings.ToLower(params.BlockHash)
	}
	// parse start block
	if params.StartBlock != "" {
		params.filter.StartHeight, err = filterBlockHeightFromTag(params.StartBlock)
		if err != nil {
			return newFilterParams{}, err
		}
	}
	// parse end block
	if params.EndBlock != "" {
		params.filter.EndHeight, err = filterBlockHeightFromTag(params.EndBlock)
		if err != nil {
			return newFilterParams{}, err
		}
	}
	params.filter.Address, err = stringArrayFromAny(params.Address)
	if err != nil {
		return newFilterParams{}, err
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
	address, err := crypto.NewAddressFromString(cleanHex(str))
	if err != nil {
		return nil, ethInvalidParams(err.Error())
	}
	return address, nil
}

// bytesFromArgs() extracts a hash from the first argument
func bytesFromArgs(args []any) ([]byte, error) {
	str, err := strFromArgs(args, 0)
	if err != nil {
		return nil, err
	}
	bz, err := lib.StringToBytes(cleanHex(str))
	if err != nil {
		return nil, ethInvalidParams(err.Error())
	}
	return bz, nil
}

// intFromArgs() extracts an integer from the first argument
func intFromArgs(args []any, position int) (int64, error) {
	str, err := strFromArgs(args, position)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		return 0, ethInvalidParams(err.Error())
	}
	return n, nil
}

// strFromArgs() extracts a string from the first argument
func strFromArgs(args []any, position int) (string, error) {
	if len(args) <= position {
		return "", ethInvalidParams("missing arguments")
	}
	str, ok := args[position].(string)
	if !ok {
		return "", ethInvalidParams("invalid argument format")
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
	return stateHeightFromBlockTag(blockTag)
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
			return 0, ethInvalidParams(fmt.Sprintf("invalid block number: %v", err))
		}
		return n, nil
	}
	return 0, ethInvalidParams(fmt.Sprintf("unsupported block tag: %s", tag))
}

func stateHeightFromBlockTag(tag string) (uint64, error) {
	switch tag {
	case latestBlockTag, pendingBlockTag, safeBlockTag, finalizedBlockTag:
		return 0, nil
	case earliestBlockTag:
		return 1, nil
	}
	height, err := parseBlockTag(tag)
	if err != nil {
		return 0, err
	}
	if height == math.MaxUint64 {
		return 0, ethInvalidParams("block number overflow")
	}
	return height + 1, nil
}

// filterBlockHeightFromTag() resolves log filter block tags to indexed Canopy heights.
func filterBlockHeightFromTag(tag string) (uint64, error) {
	switch tag {
	case earliestBlockTag, "0x0":
		return 1, nil
	}
	return parseBlockTag(tag)
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
