package store

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/ristretto/v2/z"
)

func TestStoreStream(t *testing.T) {
	config := lib.Config{
		StoreConfig: lib.StoreConfig{
			CleanupBlockInterval: 200,
			InMemory:             false,
		},
	}
	db, err := NewStore(config, "canopy", nil, lib.NewDefaultLogger())
	if err != nil {
		t.Fatal(err)
	}

	readOnlyStore, err := db.NewReadOnly(db.Version() - 2)

	if err != nil {
		t.Fatal(err)
	}
	chainId := uint64(1)

	stream := db.DB().NewStreamAt(db.Version() - 2)
	stream.NumGo = 2
	prefix := lib.Append([]byte(historicStatePrefix), fsm.CommitteePrefix(chainId))
	stream.Prefix = prefix

	stream.ChooseKey = func(item *badger.Item) bool {
		return true
	}

	count := 0
	stream.KeyToList = nil
	stream.Send = func(buf *z.Buffer) error {
		count++
		kvList, err := badger.BufferToKVList(buf)
		if err != nil {
			return err
		}

		for i, kv := range kvList.GetKv() {
			cleaned := bytes.TrimPrefix(kv.GetKey(), prefix)
			address, err := fsm.AddressFromKey(cleaned)
			if err != nil {
				return err
			}
			// load the validator from the state using the address
			_, err = GetValidator(readOnlyStore, address)
			if err != nil {
				return err
			}
			count++
			fmt.Printf("Validator [%d] %s\n", i, address.String())
		}

		return nil
	}

	start := time.Now()
	if err := stream.Orchestrate(context.Background()); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("[0] Count: %d Elapsed: %s\n", count, time.Since(start))

	// MaxCommitteeSize := uint64(1_000_000_000)
	// i := uint64(0)
	// for ; it.Valid() && i < MaxCommitteeSize; func() { it.Next(); i++ }() {
	// 	address, e := fsm.AddressFromKey(it.Key())
	// 	if e != nil {
	// 		t.Fatal(e)
	// 	}
	// 	// load the validator from the state using the address
	// 	_, e = GetValidator(readOnlyStore, address)
	// 	if e != nil {
	// 		t.Fatal(e)
	// 	}
	// }
	// it.Close()
	// readOnlyStore.Discard()
	// }
}

func TestStore(t *testing.T) {
	config := lib.Config{
		StoreConfig: lib.StoreConfig{
			CleanupBlockInterval: 200,
			InMemory:             false,
		},
	}
	db, err := NewStore(config, "canopy", nil, lib.NewDefaultLogger())
	if err != nil {
		t.Fatal(err)
	}

	newStore, err := db.NewReadOnly(db.Version() - 2)
	if err != nil {
		t.Fatal(err)
	}

	chainId := uint64(1)
	it, err := newStore.RevIterator(fsm.CommitteePrefix(chainId))
	if err != nil {
		return
	}
	defer it.Close()

	start := time.Now()
	MaxCommitteeSize := uint64(1_000_000_000)
	i := uint64(0)
	for ; it.Valid() && i < MaxCommitteeSize; func() { it.Next(); i++ }() {
		fmt.Println("Non stream Key:", lib.BytesToString(it.Key()))
		address, e := fsm.AddressFromKey(it.Key())
		if e != nil {
			t.Fatal(e)
		}
		// load the validator from the state using the address
		val, e := GetValidator(newStore, address)
		if e != nil {
			t.Fatal(e)
		}
		fmt.Printf("%d %s\n", i, lib.BytesToString(val.Address))
	}
	fmt.Printf("Count: %d Elapsed: %s\n", i, time.Since(start))
}

func GetValidator(store lib.StoreI, address crypto.AddressI) (*fsm.Validator, lib.ErrorI) {
	// get the bytes from state using the key for a validator at a specific address
	bz, err := store.Get(fsm.KeyForValidator(address))
	if err != nil {
		return nil, err
	}
	// if the bytes are empty, return 'validator doesn't exist'
	if bz == nil {
		return nil, fsm.ErrValidatorNotExists()
	}
	// convert the bytes into a validator object reference
	val, err := unmarshalValidator(bz)
	if err != nil {
		return nil, err
	}
	// update the validator structure address
	val.Address = address.Bytes()
	// return the validator
	return val, nil
}

func unmarshalValidator(bz []byte) (*fsm.Validator, lib.ErrorI) {
	// create a new validator object reference to ensure a non-nil result
	val := new(fsm.Validator)
	// populate the object reference with validator bytes
	if err := lib.Unmarshal(bz, val); err != nil {
		return nil, err
	}
	// return the object ref
	return val, nil
}
