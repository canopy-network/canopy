package plans

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// StateDB is the main interface you need to implement for state management
// This is the primary adapter between your custom state store and the EVM
type StateDB interface {
	// Account operations
	CreateAccount(common.Address)
	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int
	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)
	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	// Storage operations
	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64
	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	// State management
	Suicide(common.Address) bool
	HasSuicided(common.Address) bool
	Exist(common.Address) bool
	Empty(common.Address) bool
	PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)
	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	AddAddressToAccessList(addr common.Address)
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	// Snapshots for transaction rollback
	RevertToSnapshot(int)
	Snapshot() int

	// Logs
	AddLog(*types.Log)
	GetLogs(common.Hash, common.Hash) []*types.Log

	// State root and finalization
	IntermediateRoot(deleteEmptyObjects bool) common.Hash
	Finalise(deleteEmptyObjects bool)
	Commit(deleteEmptyObjects bool) (common.Hash, error)
}

// ChainContext provides blockchain-specific context to the EVM
// You'll need to implement this to provide block and chain information
type ChainContext interface {
	// GetHeader returns the header of a block by hash and number
	GetHeader(common.Hash, uint64) *types.Header
}

// BlockContext provides block-specific information to the EVM
// You'll need to construct this from your custom block format
type BlockContext struct {
	CanTransfer CanTransferFunc
	Transfer    TransferFunc
	GetHash     GetHashFunc

	// Block information
	Coinbase    common.Address // Provides the beneficiary of the block
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
	BaseFee     *big.Int       // Provides information for BASEFEE
	Random      *common.Hash   // Provides information for PREVRANDAO
}

// TxContext provides transaction-specific information to the EVM
// You'll need to construct this from your custom transaction format
type TxContext struct {
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE
}

// Function types used in BlockContext
type CanTransferFunc func(StateDB, common.Address, *big.Int) bool
type TransferFunc func(StateDB, common.Address, common.Address, *big.Int)
type GetHashFunc func(uint64) common.Hash

// Message interface represents a transaction for EVM execution
// You'll need to implement this to wrap your custom transaction format
type Message interface {
	From() common.Address
	To() *common.Address
	GasPrice() *big.Int
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Gas() uint64
	Value() *big.Int
	Nonce() uint64
	IsFake() bool
	Data() []byte
	AccessList() types.AccessList
}

// ChainConfig defines the chain configuration
// You'll need to adapt this for your custom blockchain parameters
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"`

	// Fork block numbers (set to nil to disable)
	HomesteadBlock      *big.Int    `json:"homesteadBlock,omitempty"`
	DAOForkBlock        *big.Int    `json:"daoForkBlock,omitempty"`
	DAOForkSupport      bool        `json:"daoForkSupport,omitempty"`
	EIP150Block         *big.Int    `json:"eip150Block,omitempty"`
	EIP150Hash          common.Hash `json:"eip150Hash,omitempty"`
	EIP155Block         *big.Int    `json:"eip155Block,omitempty"`
	EIP158Block         *big.Int    `json:"eip158Block,omitempty"`
	ByzantiumBlock      *big.Int    `json:"byzantiumBlock,omitempty"`
	ConstantinopleBlock *big.Int    `json:"constantinopleBlock,omitempty"`
	PetersburgBlock     *big.Int    `json:"petersburgBlock,omitempty"`
	IstanbulBlock       *big.Int    `json:"istanbulBlock,omitempty"`
	MuirGlacierBlock    *big.Int    `json:"muirGlacierBlock,omitempty"`
	BerlinBlock         *big.Int    `json:"berlinBlock,omitempty"`
	LondonBlock         *big.Int    `json:"londonBlock,omitempty"`
	ArrowGlacierBlock   *big.Int    `json:"arrowGlacierBlock,omitempty"`
	GrayGlacierBlock    *big.Int    `json:"grayGlacierBlock,omitempty"`
	MergeNetsplitBlock  *big.Int    `json:"mergeNetsplitBlock,omitempty"`
	ShanghaiBlock       *big.Int    `json:"shanghaiBlock,omitempty"`
	CancunBlock         *big.Int    `json:"cancunBlock,omitempty"`

	// Various consensus engines
	Ethash *params.EthashConfig `json:"ethash,omitempty"`
	Clique *params.CliqueConfig `json:"clique,omitempty"`
}

// PrecompiledContract interface for custom precompiled contracts
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64
	Run(input []byte) ([]byte, error)
}

// Logger interface for EVM execution tracing (optional)
type Logger interface {
	CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error)
	CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)
	CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error)
	CaptureEnd(output []byte, gasUsed uint64, err error)
}

// Example of how you would use these interfaces:
//
// func ExecuteTransaction(
//     evm *vm.EVM,
//     msg Message,
//     gp *GasPool,
//     statedb StateDB,
//     header *types.Header,
//     cfg *ChainConfig,
//     vmCfg vm.Config,
// ) (*types.Receipt, error) {
//     // This is where you would implement transaction execution
//     // using your custom blockchain's transaction format
// }
