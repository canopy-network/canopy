package store

import (
	"fmt"

	"github.com/canopy-network/canopy/lib"
)

func ErrOpenDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeOpenDB, lib.StorageModule, fmt.Sprintf("openDB() failed with err: %s", err.Error()))
}

func ErrCloseDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCloseDB, lib.StorageModule, fmt.Sprintf("closeDB() failed with err: %s", err.Error()))
}

func ErrCommitDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCommitDB, lib.StorageModule, fmt.Sprintf("commitDB() failed with err: %s", err.Error()))
}

func ErrGarbageCollectDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeGarbageCollectDB, lib.StorageModule, fmt.Sprintf("garbageCollectDB() failed with err: %s", err.Error()))
}

func ErrSetEntry(err error) lib.ErrorI {
	return lib.NewError(lib.CodeSetEntry, lib.StorageModule, fmt.Sprintf("setEntry() failed with err: %s", err.Error()))
}

func ErrStoreSet(err error) lib.ErrorI {
	return lib.NewError(lib.CodeStoreSet, lib.StorageModule, fmt.Sprintf("store.set() failed with err: %s", err.Error()))
}

func ErrStoreDelete(err error) lib.ErrorI {
	return lib.NewError(lib.CodeStoreDelete, lib.StorageModule, fmt.Sprintf("store.delete() failed with err: %s", err.Error()))
}

func ErrStoreGet(err error) lib.ErrorI {
	return lib.NewError(lib.CodeStoreGet, lib.StorageModule, fmt.Sprintf("store.get() failed with err: %s", err.Error()))
}

func ErrFlushBatch(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFlushBatch, lib.StorageModule, fmt.Sprintf("flushBatch() failed with err: %s", err.Error()))
}

func ErrInvalidKey() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidKey, lib.StorageModule, "found store key is invalid")
}

func ErrReserveKeyWrite(key string) lib.ErrorI {
	return lib.NewError(lib.CodeReserveKeyWrite, lib.StorageModule, fmt.Sprintf("cannot write a reserve key %s", key))
}

func ErrInvalidMerkleTree() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidMerkleTree, lib.StorageModule, "merkle tree is invalid")
}

func ErrInvalidMerkleTreeProof() lib.ErrorI {
	return lib.NewError(lib.CodeInvalidMerkleTreeProof, lib.StorageModule, "merkle tree proof is invalid")
}
