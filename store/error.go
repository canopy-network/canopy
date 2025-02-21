package store

import (
	"fmt"

	"github.com/canopy-network/canopy/lib"
)

func ErrDecompactProof(err error) lib.ErrorI {
	return lib.NewError(lib.CodeDecompactProof, lib.StorageModule, fmt.Sprintf("decompactProof failed with err: %s", err.Error()))
}

func ErrTxWrite(err error) lib.ErrorI {
	return lib.NewError(lib.CodeWriteTxn, lib.StorageModule, fmt.Sprintf("tx write failed with err: %s", err.Error()))
}

func ErrCopyStore(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCopyStore, lib.StorageModule, fmt.Sprintf("copy store failed with err: %s", err.Error()))
}

func ErrOpenDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeOpenDB, lib.StorageModule, fmt.Sprintf("openDB() failed with err: %s", err.Error()))
}

func ErrCloseDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCloseDB, lib.StorageModule, fmt.Sprintf("closeDB() failed with err: %s", err.Error()))
}

func ErrCommitDB(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCommitDB, lib.StorageModule, fmt.Sprintf("commitDB() failed with err: %s", err.Error()))
}

func ErrCommitTree(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCommitTree, lib.StorageModule, fmt.Sprintf("commitTree() failed with err: %s", err.Error()))
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

func ErrStoreIter(err error) lib.ErrorI {
	return lib.NewError(lib.CodeStoreIter, lib.StorageModule, fmt.Sprintf("store.Iter() failed with err: %s", err.Error()))
}

func ErrStoreRevIter(err error) lib.ErrorI {
	return lib.NewError(lib.CodeStoreRevIter, lib.StorageModule, fmt.Sprintf("store.RevIter() failed with err: %s", err.Error()))
}

func ErrProve(err error) lib.ErrorI {
	return lib.NewError(lib.CodeProve, lib.StorageModule, fmt.Sprintf("prove() failed with err: %s", err.Error()))
}

func ErrCompactProof(err error) lib.ErrorI {
	return lib.NewError(lib.CodeCompactProof, lib.StorageModule, fmt.Sprintf("compactProof() failed with err: %s", err.Error()))
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
