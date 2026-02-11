package fsm

import (
	"bytes"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

func TestDeltaIndexerBlobs_AccountsChangedAddedRemoved_PoolsRemainFull(t *testing.T) {
	prevAcc1 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{1}, 20), Amount: 10})
	prevAcc2 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{2}, 20), Amount: 20})
	currAcc2 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{2}, 20), Amount: 21})
	currAcc3 := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{3}, 20), Amount: 30})

	prevPool1 := mustMarshalProto(t, &Pool{Id: 1, Amount: 100})
	prevPool2 := mustMarshalProto(t, &Pool{Id: 2, Amount: 200})
	currPool2 := mustMarshalProto(t, &Pool{Id: 2, Amount: 201})
	currPool3 := mustMarshalProto(t, &Pool{Id: 3, Amount: 300})

	prevVal1 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{4}, 20), StakedAmount: 400})
	prevVal2 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{5}, 20), StakedAmount: 500})
	currVal1 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{4}, 20), StakedAmount: 400})
	currVal3 := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{6}, 20), StakedAmount: 600})

	prev := &IndexerBlob{
		Block:      mustMarshalProto(t, &lib.BlockResult{}),
		Accounts:   [][]byte{prevAcc1, prevAcc2},
		Pools:      [][]byte{prevPool1, prevPool2},
		Validators: [][]byte{prevVal1, prevVal2},
	}
	curr := &IndexerBlob{
		Block:      mustMarshalProto(t, &lib.BlockResult{}),
		Accounts:   [][]byte{currAcc2, currAcc3},
		Pools:      [][]byte{currPool2, currPool3},
		Validators: [][]byte{currVal1, currVal3},
	}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: curr, Previous: prev})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, currAcc2, currAcc3)
	requireEntriesAsSet(t, delta.Previous.Accounts, prevAcc1, prevAcc2)
	requireEntriesAsSet(t, delta.Current.Pools, currPool2, currPool3)
	requireEntriesAsSet(t, delta.Previous.Pools, prevPool1, prevPool2)
	requireEntriesAsSet(t, delta.Current.Validators, currVal1, currVal3)
	requireEntriesAsSet(t, delta.Previous.Validators, prevVal1, prevVal2)
}

func TestDeltaIndexerBlobs_UnchangedAccountsBecomeEmpty_PoolsRemainFull(t *testing.T) {
	acc := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{7}, 20), Amount: 1})
	pool := mustMarshalProto(t, &Pool{Id: 7, Amount: 7})
	val := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{8}, 20), StakedAmount: 8})
	block := mustMarshalProto(t, &lib.BlockResult{})
	current := &IndexerBlob{Block: block, Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}}
	previous := &IndexerBlob{Block: block, Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}}

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{Current: current, Previous: previous})
	require.NoError(t, err)
	require.Empty(t, delta.Current.Accounts)
	require.Empty(t, delta.Previous.Accounts)
	requireEntriesAsSet(t, delta.Current.Pools, pool)
	requireEntriesAsSet(t, delta.Previous.Pools, pool)
	requireEntriesAsSet(t, delta.Current.Validators, val)
	requireEntriesAsSet(t, delta.Previous.Validators, val)
}

func TestDeltaIndexerBlobs_ForceIncludeRewardSlashAccounts(t *testing.T) {
	addr := bytes.Repeat([]byte{9}, 20)
	accPrev := mustMarshalProto(t, &Account{Address: addr, Amount: 100})
	accCurr := mustMarshalProto(t, &Account{Address: addr, Amount: 100})

	block := &lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: 10},
		Events: []*lib.Event{{
			EventType: string(lib.EventTypeReward),
			Address:   addr,
		}},
	}
	blockBz := mustMarshalProto(t, block)
	emptyBlockBz := mustMarshalProto(t, &lib.BlockResult{})

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current:  &IndexerBlob{Block: blockBz, Accounts: [][]byte{accCurr}},
		Previous: &IndexerBlob{Block: emptyBlockBz, Accounts: [][]byte{accPrev}},
	})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, accCurr)
	requireEntriesAsSet(t, delta.Previous.Accounts, accPrev)
}

func TestDeltaIndexerBlobs_NoPreviousKeepsCurrent(t *testing.T) {
	acc := mustMarshalProto(t, &Account{Address: bytes.Repeat([]byte{10}, 20), Amount: 42})
	pool := mustMarshalProto(t, &Pool{Id: 42, Amount: 42})
	val := mustMarshalProto(t, &Validator{Address: bytes.Repeat([]byte{11}, 20), StakedAmount: 11})

	delta, err := DeltaIndexerBlobs(&IndexerBlobs{
		Current: &IndexerBlob{Accounts: [][]byte{acc}, Pools: [][]byte{pool}, Validators: [][]byte{val}},
	})
	require.NoError(t, err)
	requireEntriesAsSet(t, delta.Current.Accounts, acc)
	requireEntriesAsSet(t, delta.Current.Pools, pool)
	requireEntriesAsSet(t, delta.Current.Validators, val)
	require.Nil(t, delta.Previous)
}

func mustMarshalProto(t *testing.T, message any) []byte {
	t.Helper()
	bz, err := lib.Marshal(message)
	require.NoError(t, err)
	return bz
}

func requireEntriesAsSet(t *testing.T, got [][]byte, expected ...[]byte) {
	t.Helper()
	gotSet := make(map[string]int, len(got))
	for _, entry := range got {
		gotSet[string(entry)]++
	}
	expSet := make(map[string]int, len(expected))
	for _, entry := range expected {
		expSet[string(entry)]++
	}
	require.Equal(t, expSet, gotSet)
}
