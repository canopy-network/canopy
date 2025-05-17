package fsm

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
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
const SendSelector = "a9059cbb"    // transfer(address,uint256)
const SubsidySelector = "5b82335c" // subsidy(bytes msgSubsidyProtoBytes)

// StakedCNPYContractAddress: (stCNPY) a fake contract address that allows tools to stake, edit, and unstake CNPY as a delegator
const StakedCNPYContractAddress = `0x0000000000000000000000000000000000000002`
const StakeSelector = "b3de648b"     // stake(bytes msgStakeProtoBytes)
const EditStakeSelector = "ec8ac4b9" // editStake(bytes msgEditStakeBytes)
const UnstakeSelector = "89d8c832"   // unstake(bytes msgUnstakeBytes)

// SwapCNPYContractAddress: (swCNPY) a fake contract address that allows tools to create a sell order, edit a sell order, and delete a sell order
const SwapCNPYContractAddress = `0x0000000000000000000000000000000000000003`
const CreateOrderSelector = "2f6c493c" // createOrder(bytes msgCreateOrderProtoBytes)
const EditOrderSelector = "04d4ee2e"   // editOrder(bytes msgEditOrderProtoBytes)
const DeleteOrderSelector = "af2cf05c" // deleteOrder(bytes msgDeleteOrderProtoBytes)

// RLPIndicator is a human-readable indicator the tx is translated from RLP
const RLPIndicator = "RLP"

// RLPToCanopyTransaction() converts an RLP encoded transaction into a Canopy transaction
func RLPToCanopyTransaction(txBytes []byte) (transaction *lib.Transaction, e lib.ErrorI) {
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
	// extract chain id and network id from the evm chain id
	chainId, networkId := EvmChainIdToCanopyIds(tx.ChainId().Uint64())
	// generate the transaction object
	transaction = &lib.Transaction{
		MessageType: MessageSendName, // fallback default
		Signature: &lib.Signature{
			PublicKey: publicKey.Bytes(),
			Signature: txBytes, // store the raw transaction here
		},
		CreatedHeight: tx.Nonce(), // the rpc ensures a proper value that satisfies replay protection
		Time:          pseudoEthereumTimestamp(tx.Gas()),
		Fee:           DownscaleTo6Decimals(new(big.Int).Mul(big.NewInt(int64(tx.Gas())), tx.GasPrice())),
		NetworkId:     networkId,
		Memo:          RLPIndicator,
		ChainId:       chainId,
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
				if len(m.PublicKey) == 0 {
					m.PublicKey = publicKey.Bytes()
				}
			})
		case EditStakeSelector:
			m := new(MessageEditStake)
			msg, e = ethDataToMsg(MessageEditStakeName, transaction, m, data, func() {
				if len(m.Address) == 0 {
					m.Address = from
				}
			})
		case UnstakeSelector:
			m := new(MessageUnstake)
			msg, e = ethDataToMsg(MessageUnstakeName, transaction, m, data, func() {
				if len(m.Address) == 0 {
					m.Address = from
				}
			})
		case CreateOrderSelector:
			m := new(MessageCreateOrder)
			msg, e = ethDataToMsg(MessageCreateOrderName, transaction, m, data, func() { m.SellersSendAddress = from })
		case EditOrderSelector:
			msg, e = ethDataToMsg(MessageEditOrderName, transaction, new(MessageEditOrder), data, func() {})
		case DeleteOrderSelector:
			msg, e = ethDataToMsg(MessageDeleteOrderName, transaction, new(MessageDeleteOrder), data, func() {})
		case SubsidySelector:
			m := new(MessageSubsidy)
			msg, e = ethDataToMsg(MessageSubsidyName, transaction, m, data, func() { m.Address = from })
		default:
			return nil, ErrInvalidERC20Tx(fmt.Errorf("unsupported selector: 0x%s", selector))
		}
	default: // non-contract call (transfer() only)
		msg = &MessageSend{
			FromAddress: from,
			ToAddress:   tx.To().Bytes(),
			Amount:      DownscaleTo6Decimals(tx.Value()),
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
	if !amount.IsUint64() {
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
	callback()
	// sanity check the message
	return msg, msg.Check()
}

// VerifyRLPBytes() implements special 'signature verification logic' that allows a MessageSend to be authenticated using a signed RLP transaction
func (s *StateMachine) VerifyRLPBytes(tx *lib.Transaction) lib.ErrorI {
	// create a compare transaction from the signature field
	compare, err := RLPToCanopyTransaction(tx.Signature.Signature)
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

// ChainId translation design:
// evmChainId is High 32 bits = networkId and Low 32 bits = chainId
// - avoids conflicts with existing Ethereum chain IDs
// - merges Canopy's dual network ID scheme into a single combined identifier

// EvmChainIdToCanopyIds() converts an EVM chainId to a Canopy chain id
func EvmChainIdToCanopyIds(evmChainId uint64) (chainId, networkId uint64) {
	networkId = evmChainId >> 32
	chainId = evmChainId & 0xFFFFFFFF
	return
}

// CanopyIdsToEVMChainId() converts a chainId and networkId to an evm chain ID
func CanopyIdsToEVMChainId(chainId, networkId uint64) uint64 {
	return (networkId << 32) | (chainId & 0xFFFFFFFF)
}

// scaleFactor allows conversion from 6 decimal places to 18 10^12
var scaleFactor = big.NewInt(1_000_000_000_000)

// UpscaleTo18Decimals converts a 6-decimal unit (Canopy native) to 18-decimal (Ethereum RPC)
func UpscaleTo18Decimals(amount uint64) *big.Int {
	return new(big.Int).Mul(big.NewInt(int64(amount)), scaleFactor)
}

// DownscaleTo6Decimals converts from 18-decimal unit (Ethereum RPC) to 6-decimal (Canopy native)
func DownscaleTo6Decimals(amount *big.Int) uint64 {
	return new(big.Int).Div(amount, scaleFactor).Uint64()
}

// pseudoEthereumTimestamp() creates a fake timestamp to ensure collision resistance using the gas limit variable
// - note: the gas-limit may be used to extend the pseudo-nonce functionality if needed
func pseudoEthereumTimestamp(gasLimit uint64) uint64 {
	return uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(gasLimit) * time.Second).Unix())
}
