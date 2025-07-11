package vm

import (
	"fmt"
	"math"

	"github.com/canopy-network/canopy/lib"
)

const (
	NoCode lib.ErrorCode = math.MaxUint32

	// VM Module
	VMModule lib.ErrorModule = "vm"

	// VM Module Error Codes
	CodeVMClosed            lib.ErrorCode = 1
	CodeCreateDataDir       lib.ErrorCode = 2
	CodeInitCache           lib.ErrorCode = 3
	CodeCreateVM            lib.ErrorCode = 4
	CodeStoreCode           lib.ErrorCode = 5
	CodeGetCode             lib.ErrorCode = 6
	CodeInstantiateContract lib.ErrorCode = 7
	CodeExecuteContract     lib.ErrorCode = 8
	CodeQueryContract       lib.ErrorCode = 9
	CodeValidateContract    lib.ErrorCode = 10
	CodeCreateTempVM        lib.ErrorCode = 11
	CodeCreateTempDir       lib.ErrorCode = 12

	// Cache Module
	CacheModule lib.ErrorModule = "vm_cache"

	// Cache Module Error Codes
	CodeCacheMaxSize    lib.ErrorCode = 1
	CodeCacheNotFound   lib.ErrorCode = 2
	CodeCacheStoreGet   lib.ErrorCode = 3
	CodeCacheStoreSet   lib.ErrorCode = 4
	CodeCacheStoreCheck lib.ErrorCode = 5

	// Bridge Module
	BridgeModule lib.ErrorModule = "vm_bridge"

	// Bridge Module Error Codes
	CodeOutOfGas             lib.ErrorCode = 1
	CodeInvalidAddress       lib.ErrorCode = 2
	CodeUnsupportedQuery     lib.ErrorCode = 3
	CodeInvalidCustomQuery   lib.ErrorCode = 4
	CodeContractStorageQuery lib.ErrorCode = 5
	CodeQueryNotImplemented  lib.ErrorCode = 6
)

// Error functions for VM module
func ErrVMClosed() lib.ErrorI {
	return lib.NewError(CodeVMClosed, VMModule, "VM is closed")
}

func ErrCreateDataDir(err error) lib.ErrorI {
	return lib.NewError(CodeCreateDataDir, VMModule, "failed to create VM data directory: "+err.Error())
}

func ErrInitCache(err error) lib.ErrorI {
	return lib.NewError(CodeInitCache, VMModule, "failed to initialize VM cache: "+err.Error())
}

func ErrCreateVM(err error) lib.ErrorI {
	return lib.NewError(CodeCreateVM, VMModule, "failed to create wasmvm instance: "+err.Error())
}

func ErrStoreCode(err error, gasCost uint64) lib.ErrorI {
	return lib.NewError(CodeStoreCode, VMModule, "failed to compile WASM code: "+err.Error()+", gas cost: "+fmt.Sprintf("%d", gasCost))
}

func ErrGetCode(codeID uint64) lib.ErrorI {
	return lib.NewError(CodeGetCode, VMModule, fmt.Sprintf("WASM code not found: code ID %d", codeID))
}

func ErrInstantiateContract(err error) lib.ErrorI {
	return lib.NewError(CodeInstantiateContract, VMModule, "contract instantiation failed: "+err.Error())
}

func ErrExecuteContract(err error) lib.ErrorI {
	return lib.NewError(CodeExecuteContract, VMModule, "contract execution failed: "+err.Error())
}

func ErrQueryContract(err error) lib.ErrorI {
	return lib.NewError(CodeQueryContract, VMModule, "contract query failed: "+err.Error())
}

func ErrValidateContract(err error) lib.ErrorI {
	return lib.NewError(CodeValidateContract, VMModule, "WASM validation failed: "+err.Error())
}

func ErrCreateTempVM(err error) lib.ErrorI {
	return lib.NewError(CodeCreateTempVM, VMModule, "failed to create temp VM: "+err.Error())
}

func ErrCreateTempDir(err error) lib.ErrorI {
	return lib.NewError(CodeCreateTempDir, VMModule, "failed to create temp directory: "+err.Error())
}

// Error functions for Cache module
func ErrCacheMaxSize() lib.ErrorI {
	return lib.NewError(CodeCacheMaxSize, CacheModule, "cache max size must be greater than 0")
}

func ErrCacheNotFound() lib.ErrorI {
	return lib.NewError(CodeCacheNotFound, CacheModule, "code not found in cache")
}

func ErrCacheStoreGet(err error) lib.ErrorI {
	return lib.NewError(CodeCacheStoreGet, CacheModule, "failed to get code from store: "+err.Error())
}

func ErrCacheStoreSet(err error) lib.ErrorI {
	return lib.NewError(CodeCacheStoreSet, CacheModule, "failed to set code in store: "+err.Error())
}

func ErrCacheStoreCheck(err error) lib.ErrorI {
	return lib.NewError(CodeCacheStoreCheck, CacheModule, "failed to check code in store: "+err.Error())
}

// Error functions for Bridge module
func ErrOutOfGas(used, limit uint64) lib.ErrorI {
	return lib.NewError(CodeOutOfGas, BridgeModule, fmt.Sprintf("out of gas: used %d, limit %d", used, limit))
}

func ErrInvalidAddress(addr string, err error) lib.ErrorI {
	return lib.NewError(CodeInvalidAddress, BridgeModule, fmt.Sprintf("invalid address %s: %s", addr, err.Error()))
}

func ErrUnsupportedQuery(queryType string) lib.ErrorI {
	return lib.NewError(CodeUnsupportedQuery, BridgeModule, "unsupported query type: "+queryType)
}

func ErrInvalidCustomQuery(err error) lib.ErrorI {
	return lib.NewError(CodeInvalidCustomQuery, BridgeModule, "invalid custom query: "+err.Error())
}

func ErrContractStorageQuery(err error) lib.ErrorI {
	return lib.NewError(CodeContractStorageQuery, BridgeModule, "failed to query contract storage: "+err.Error())
}

func ErrQueryNotImplemented(queryType string) lib.ErrorI {
	return lib.NewError(CodeQueryNotImplemented, BridgeModule, queryType+" queries not implemented yet")
}
