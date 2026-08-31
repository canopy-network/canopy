package fsm

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"math"
	"math/big"
	"time"
)

/* This file implements an Ethereum translation layer over Canopy transactions in order to allow popular tooling to interact with Canopy */

// Flow
// 1. An Ethereum wallet creates the transaction
// 2. Canopy translates the RLP to a standard Canopy transaction and gossips it like normal
// 3. If RLP detected during tx processing, verify the signature and generate a new transaction from the RLP to verify equality

// CNPYContractAddress: (CNPY) a fake contract address that allows tools to send/receive CNPY as if it's an ERC20
const CNPYContractAddress = `0x0000000000000000000000000000000000000001`
const SendSelector = "a9059cbb"    // transfer(address,uint256) # Canopy expects valid ABI encoding to be 100% compatible with send tooling
const SubsidySelector = "16d68b09" // while the signature is subsidy(bytes), Canopy expects (selector + proto-bytes)

// StakedCNPYContractAddress: (stCNPY) a fake contract address that allows tools to stake, edit, and unstake CNPY as a delegator
const StakedCNPYContractAddress = `0x0000000000000000000000000000000000000002`
const StakeSelector = "2d1e0c02"     // while the signature is stake(bytes), Canopy expects (selector + proto-bytes)
const EditStakeSelector = "8c71a515" // while the signature is editStake(bytes), Canopy expects (selector + proto-bytes)
const UnstakeSelector = "3c3653e2"   // while the signature is unstake(bytes), Canopy expects (selector + proto-bytes)

// SwapCNPYContractAddress: (swCNPY) a fake contract address that allows tools to create a sell order, edit a sell order, and delete a sell order
const SwapCNPYContractAddress = `0x0000000000000000000000000000000000000003`
const CreateOrderSelector = "bc2e8e5f" // while the signature is createOrder(bytes), Canopy expects (selector + proto-bytes)
const EditOrderSelector = "74e78d6f"   // while the signature is editOrder(bytes), Canopy expects (selector + proto-bytes)
const DeleteOrderSelector = "6c4650e7" // while the signature is deleteOrder(bytes), Canopy expects (selector + proto-bytes)

// RLPIndicator is the legacy indicator for RLP-backed transactions that map Ethereum nonce onto CreatedHeight.
const RLPIndicator = lib.RLPIndicator

// RLPV2Indicator uses the dedicated tx nonce field and a canonical CreatedHeight sentinel.
const RLPV2Indicator = lib.RLPV2Indicator

// RLPV2CreatedHeight is a canonical sentinel. RLP.V2 replay protection is provided by the
// account nonce, so its wrapper must not contain mutable, unsigned height metadata.
const RLPV2CreatedHeight uint64 = 1

// EthereumBaseFeePerGas is the fixed base fee advertised by Canopy's Ethereum RPC.
const EthereumBaseFeePerGas int64 = 10_000_000_000

// legacyRLPDisabledProtocolVersion disables new legacy RLP wrappers while retaining historical replay.
const legacyRLPDisabledProtocolVersion = 2

// IsRLPMemo reports whether the memo indicates an RLP-backed Ethereum transaction.
func IsRLPMemo(memo string) bool { return lib.IsRLPMemo(memo) }

func ethereumTxHashFromRawBytes(txBytes []byte) ([]byte, lib.ErrorI) {
	var tx ethTypes.Transaction
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		return nil, ErrInvalidRLPTx(err)
	}
	return tx.Hash().Bytes(), nil
}

// RLPToCanopyTransaction() converts a legacy-domain RLP transaction into a Canopy transaction.
func RLPToCanopyTransaction(txBytes []byte) (transaction *lib.Transaction, e lib.ErrorI) {
	return rlpToCanopyTransaction(txBytes, RLPIndicator)
}

// RLPToCanopyTransactionV2() converts an RLP encoded transaction into a nonce-backed Canopy transaction.
func RLPToCanopyTransactionV2(txBytes []byte) (transaction *lib.Transaction, e lib.ErrorI) {
	return rlpToCanopyTransaction(txBytes, RLPV2Indicator)
}

func rlpToCanopyTransaction(txBytes []byte, memo string) (transaction *lib.Transaction, e lib.ErrorI) {
	// protect against spam
	if len(txBytes) > int(2*units.KB) {
		return nil, ErrInvalidRLPTx(fmt.Errorf("max transaction size"))
	}
	// decode transaction to ethereum object
	var tx ethTypes.Transaction
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		return nil, ErrInvalidRLPTx(err)
	}
	// get the signer type (supports: Legacy, EIP-155, EIP-1559, EIP-2930, EIP-4844, EIP-7702)
	signer := ethTypes.LatestSignerForChainID(tx.ChainId())
	// recover the public key from the rlp transaction and validate the signature
	publicKey, err := crypto.RecoverPublicKey(signer, tx)
	if err != nil {
		return nil, ErrInvalidPublicKey(err)
	}
	// ensure the EVM chain id fits into Canopy's uint64 translation.
	if tx.ChainId() == nil || !tx.ChainId().IsUint64() {
		return nil, ErrInvalidRLPTx(fmt.Errorf("chain id exceeds uint64"))
	}
	// The signed Ethereum chain ID separates legacy RLP from RLP.V2 even while
	// both wrapper formats remain understood by this binary.
	chainId, networkId, chainIDErr := evmChainIdToCanopyIdsForRLP(tx.ChainId().Uint64(), memo)
	if chainIDErr != nil {
		return nil, chainIDErr
	}
	gasPrice := tx.GasPrice()
	if memo == RLPV2Indicator {
		gasPrice, err = EthereumEffectiveGasPrice(&tx)
		if err != nil {
			return nil, ErrInvalidRLPTx(err)
		}
	}
	// compute fee with unsigned gas to avoid signed narrowing skew
	fee, ok := DownscaleTo6DecimalsChecked(new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), gasPrice))
	if !ok {
		return nil, ErrInvalidRLPTx(fmt.Errorf("invalid fee amount"))
	}
	createdHeight := RLPV2CreatedHeight
	if memo == RLPIndicator {
		createdHeight = tx.Nonce()
	}
	// generate the transaction object
	transaction = &lib.Transaction{
		MessageType: MessageSendName, // fallback default
		Signature: &lib.Signature{
			PublicKey: publicKey.Bytes(),
			Signature: txBytes, // store the raw transaction here
		},
		CreatedHeight: createdHeight,
		Time:          pseudoEthereumTimestamp(tx.Gas()),
		Fee:           fee,
		NetworkId:     networkId,
		Memo:          memo,
		ChainId:       chainId,
	}
	if memo == RLPV2Indicator {
		transaction.Nonce = tx.Nonce()
	}
	// extract a message from the rlp transaction
	msg, e := rlpToMessage(publicKey, transaction, tx)
	// handle any error
	if e != nil {
		return
	}
	// convert the message to an `any`
	transaction.Msg, e = lib.NewAny(msg)
	// exit
	return
}

// EthereumEffectiveGasPrice returns the execution gas price for RLP.V2. Dynamic-fee
// transactions pay min(fee cap, base fee + tip); legacy typed transactions retain gasPrice.
func EthereumEffectiveGasPrice(tx *ethTypes.Transaction) (*big.Int, error) {
	switch tx.Type() {
	case ethTypes.DynamicFeeTxType, ethTypes.BlobTxType, ethTypes.SetCodeTxType:
		baseFee := big.NewInt(EthereumBaseFeePerGas)
		tip, err := tx.EffectiveGasTip(baseFee)
		if err != nil {
			return nil, err
		}
		return tip.Add(tip, baseFee), nil
	default:
		return tx.GasPrice(), nil
	}
}

// rlpToMessage() converts an ethereum RLP transaction to a message
func rlpToMessage(publicKey crypto.PublicKeyI, transaction *lib.Transaction, tx ethTypes.Transaction) (msg lib.MessageI, e lib.ErrorI) {
	// get the relevant tx fields
	to, from, data := tx.To(), publicKey.Address().Bytes(), tx.Data()
	// ensure non-nil to
	if to == nil {
		return nil, ErrRecipientAddressEmpty()
	}
	// switch on the 'recipient'
	switch tx.To().Hex() {
	// if the recipient is a pseudo-contract call
	case CNPYContractAddress, StakedCNPYContractAddress, SwapCNPYContractAddress:
		// ensure enough data for a selector
		if len(data) < 4 {
			return nil, ErrInvalidERC20Tx(fmt.Errorf("data too short"))
		}
		// switch on the selector
		switch selector := lib.BytesToString(data[:4]); selector {
		case SendSelector:
			msg, e = ethDataToMsgSend(from, data)
		case StakeSelector:
			m := new(MessageStake)
			msg, e = ethDataToMsg(MessageStakeName, transaction, m, data, func() {
				// allow the omission of the public key because it may be difficult to get the public key from the wallet
				if len(m.PublicKey) == 0 {
					m.PublicKey = publicKey.Bytes()
				}
			})
		case EditStakeSelector:
			m := new(MessageEditStake)
			msg, e = ethDataToMsg(MessageEditStakeName, transaction, m, data, nil)
		case UnstakeSelector:
			m := new(MessageUnstake)
			msg, e = ethDataToMsg(MessageUnstakeName, transaction, m, data, nil)
		case CreateOrderSelector:
			m := new(MessageCreateOrder)
			msg, e = ethDataToMsg(MessageCreateOrderName, transaction, m, data, nil)
		case EditOrderSelector:
			msg, e = ethDataToMsg(MessageEditOrderName, transaction, new(MessageEditOrder), data, nil)
		case DeleteOrderSelector:
			msg, e = ethDataToMsg(MessageDeleteOrderName, transaction, new(MessageDeleteOrder), data, nil)
		case SubsidySelector:
			m := new(MessageSubsidy)
			msg, e = ethDataToMsg(MessageSubsidyName, transaction, m, data, nil)
		default:
			e = ErrInvalidERC20Tx(fmt.Errorf("unsupported selector: 0x%s", selector))
		}
	default: // non-contract call (transfer() only)
		if transaction.Memo == RLPV2Indicator && new(big.Int).Mod(tx.Value(), scaleFactor).Sign() != 0 {
			return nil, ErrInvalidAmount()
		}
		amount, ok := DownscaleTo6DecimalsChecked(tx.Value())
		if !ok || amount == 0 {
			return nil, ErrInvalidAmount()
		}
		msg = &MessageSend{
			FromAddress: from,
			ToAddress:   tx.To().Bytes(),
			Amount:      amount,
		}
	}
	return
}

// ethDataToMsgSend() translates the 'data' from an RLP transaction into a MessageSend
// - not protobuf, actual ABI encoding to allow native ERC20 transfer() calls
func ethDataToMsgSend(fromAddress, data []byte) (*MessageSend, lib.ErrorI) {
	// check input length (4 selector + 32 address + 32 amount = 136 hex chars + 2 for "0x")
	if len(data) < 4+32+32 {
		return nil, ErrInvalidERC20Tx(fmt.Errorf("input too short"))
	}
	// amount: full 32-byte uint256
	amount := new(big.Int).SetBytes(data[36:68])
	// sanity check the amount
	if !amount.IsUint64() || amount.Uint64() == 0 {
		return nil, ErrInvalidAmount()
	}
	// return the message send type
	return &MessageSend{
		ToAddress:   data[16:36],
		FromAddress: fromAddress,
		Amount:      amount.Uint64(), // take amount as is - because decimals is specified at the 'contract level'
	}, nil
}

// ethDataToMsg() converts the ethereum tx data (message proto bytes) to a message
func ethDataToMsg(messageType string, transaction *lib.Transaction, msg lib.MessageI, data []byte, callback func()) (lib.MessageI, lib.ErrorI) {
	// set the message type
	transaction.MessageType = messageType
	// convert the data after the selector to a message
	if err := lib.Unmarshal(data[4:], msg); err != nil {
		return nil, err
	}
	// execute the callback
	if callback != nil {
		callback()
	}
	// sanity check the message
	return msg, msg.Check()
}

// VerifyRLPBytes() implements special 'signature verification logic' that allows a MessageSend to be authenticated using a signed RLP transaction
func (s *StateMachine) VerifyRLPBytes(tx *lib.Transaction) lib.ErrorI {
	// create a compare transaction from the signature field
	compare, err := RLPToCanopyTransaction(tx.Signature.Signature)
	if tx.Memo == RLPV2Indicator {
		compare, err = RLPToCanopyTransactionV2(tx.Signature.Signature)
	}
	if err != nil {
		return err
	}
	// get the transaction hash (includes the raw RLP) for the compare tx
	compareHash, err := compare.GetHash()
	if err != nil {
		return err
	}
	// get the transaction hash (includes the raw RLP) for the raw tx
	originalHash, err := tx.GetHash()
	if err != nil {
		return err
	}
	// check the equality of the two transactions
	if !bytes.Equal(compareHash, originalHash) {
		return ErrInvalidSignature()
	}
	// exit without error
	return nil
}

// Ethereum chain ID layout:
//   - high 32 bits: Canopy network ID
//   - next 2 bits: signed RLP domain (0 = legacy, 1 = RLP.V2)
//   - low 30 bits: Canopy chain ID
//
// RLP.V2 uses a different Ethereum signing domain without changing Canopy's
// internal network or chain identifiers.
const (
	evmChainIDLowMask     = uint64(math.MaxUint32)
	evmChainIDDomainShift = 30
	evmChainIDDomainMask  = uint64(3) << evmChainIDDomainShift
	evmChainIDChainMask   = uint64(1<<evmChainIDDomainShift) - 1
	rlpV2EVMChainIDDomain = uint64(1)
	rlpV2EVMChainIDMarker = rlpV2EVMChainIDDomain << evmChainIDDomainShift
)

// EvmChainIdToCanopyIds() decodes the legacy EVM chain ID layout.
func EvmChainIdToCanopyIds(evmChainId uint64) (chainId, networkId uint64) {
	networkId = evmChainId >> 32
	chainId = evmChainId & evmChainIDLowMask
	return
}

// CanopyIdsToEVMChainId() encodes the legacy EVM chain ID layout.
func CanopyIdsToEVMChainId(chainId, networkId uint64) uint64 {
	return (networkId << 32) | (chainId & evmChainIDLowMask)
}

// CanopyIdsToEVMChainIdV2 returns the domain-separated chain ID advertised by
// this binary. The boolean is false rather than silently aliasing oversized IDs.
func CanopyIdsToEVMChainIdV2(chainId, networkId uint64) (uint64, bool) {
	if chainId == 0 || chainId > evmChainIDChainMask || networkId == 0 || networkId > math.MaxUint32 {
		return 0, false
	}
	return (networkId << 32) | rlpV2EVMChainIDMarker | chainId, true
}

func evmChainIdToCanopyIdsForRLP(evmChainId uint64, memo string) (chainId, networkId uint64, err lib.ErrorI) {
	low := evmChainId & evmChainIDLowMask
	domain := (low & evmChainIDDomainMask) >> evmChainIDDomainShift
	expectedDomain := uint64(0)
	if memo == RLPV2Indicator {
		expectedDomain = rlpV2EVMChainIDDomain
	}
	if domain != expectedDomain {
		return 0, 0, ErrInvalidRLPTx(fmt.Errorf("ethereum chain id uses RLP domain %d, expected %d", domain, expectedDomain))
	}
	chainId, networkId = low&evmChainIDChainMask, evmChainId>>32
	if chainId == 0 {
		return 0, 0, ErrInvalidRLPTx(fmt.Errorf("invalid zero chain id"))
	}
	return chainId, networkId, nil
}

// scaleFactor allows conversion from 6 decimal places to 18 10^12
var scaleFactor = big.NewInt(1_000_000_000_000)

// UpscaleTo18Decimals converts a 6-decimal unit (Canopy native) to 18-decimal (Ethereum RPC)
func UpscaleTo18Decimals(amount uint64) *big.Int {
	return new(big.Int).Mul(new(big.Int).SetUint64(amount), scaleFactor)
}

// DownscaleTo6Decimals converts from 18-decimal unit (Ethereum RPC) to 6-decimal (Canopy native)
func DownscaleTo6Decimals(amount *big.Int) uint64 {
	return new(big.Int).Div(amount, scaleFactor).Uint64()
}

// DownscaleTo6DecimalsChecked converts from 18-decimal unit (Ethereum RPC) to 6-decimal (Canopy native)
// and reports overflow/invalid input instead of wrapping through Uint64().
func DownscaleTo6DecimalsChecked(amount *big.Int) (uint64, bool) {
	if amount == nil || amount.Sign() < 0 {
		return 0, false
	}
	downscaled := new(big.Int).Div(amount, scaleFactor)
	if !downscaled.IsUint64() {
		return 0, false
	}
	return downscaled.Uint64(), true
}

// pseudoEthereumTimestamp() creates a fake timestamp to ensure collision resistance using the gas limit variable
// - note: the gas-limit may be used to extend the pseudo-nonce functionality if needed
func pseudoEthereumTimestamp(gasLimit uint64) uint64 {
	return uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(gasLimit) * time.Second).Unix())
}
