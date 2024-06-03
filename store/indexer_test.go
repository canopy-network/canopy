package store

import (
	"bytes"
	"github.com/ginchuco/ginchu/lib"
	"github.com/ginchuco/ginchu/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

const testHeight = 1

func TestGetTxByHash(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	txRes, _, hash, _, _ := newTestTxResult(t)
	require.NoError(t, store.IndexTx(txRes))
	txResult, err := store.GetTxByHash(hash)
	require.NoError(t, err)
	gotBytes, err := txResult.GetBytes()
	require.NoError(t, err)
	wantedBytes, err := txRes.GetBytes()
	require.NoError(t, err)
	require.True(t, bytes.Equal(gotBytes, wantedBytes))
}

func TestGetTxByHeight(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	txRes, _, _, _, _ := newTestTxResult(t)
	require.NoError(t, store.IndexTx(txRes))
	txResults, err := store.GetTxsByHeightNonPaginated(testHeight, true)
	require.NoError(t, err)
	require.Len(t, txResults, 1)
	gotBytes, err := txResults[0].GetBytes()
	require.NoError(t, err)
	wantedBytes, err := txRes.GetBytes()
	require.NoError(t, err)
	require.True(t, bytes.Equal(gotBytes, wantedBytes))
}

func newTestTxResult(t *testing.T) (r *lib.TxResult, tx *lib.Transaction, hash []byte, msg *CommitID, address crypto.AddressI) {
	pk, err := crypto.NewEd25519PrivateKey()
	require.NoError(t, err)
	msg = &CommitID{
		Height: 1,
		Root:   []byte("root"),
	}
	address = pk.PublicKey().Address()
	a, err := lib.NewAny(msg)
	require.NoError(t, err)
	tx = &lib.Transaction{
		Type:     "commit_id",
		Msg:      a,
		Sequence: 1,
		Fee:      1,
	}
	require.NoError(t, tx.Sign(pk))
	hash, err = tx.GetHash()
	require.NoError(t, err)
	r = &lib.TxResult{
		Sender:      address.Bytes(),
		Recipient:   address.Bytes(),
		MessageType: "commit_id",
		Height:      testHeight,
		Index:       0,
		Transaction: tx,
		TxHash:      lib.BytesToString(hash),
	}
	return
}