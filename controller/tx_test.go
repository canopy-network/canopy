package controller

import (
	"sync"
	"testing"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestGetPendingTxByHash(t *testing.T) {
	c := &Controller{
		Mempool: &Mempool{
			cachedResults: lib.TxResults{
				&lib.TxResult{TxHash: "abcdef1234"},
				&lib.TxResult{TxHash: "1234567890"},
			},
		},
		Mutex: &sync.Mutex{},
	}

	tx, found := c.GetPendingTxByHash("ABCDEF1234")
	require.True(t, found)
	require.NotNil(t, tx)
	require.Equal(t, "abcdef1234", tx.TxHash)

	tx, found = c.GetPendingTxByHash("0x1234567890")
	require.True(t, found)
	require.NotNil(t, tx)
	require.Equal(t, "1234567890", tx.TxHash)

	tx, found = c.GetPendingTxByHash("missing")
	require.False(t, found)
	require.Nil(t, tx)
}

func TestGetProposalBlockFromMempool(t *testing.T) {
	c := &Controller{Mempool: &Mempool{}}

	p, ok := c.GetProposalBlockFromMempool()
	require.False(t, ok)
	require.Nil(t, p)

	expected := &CachedProposal{dirtyVersion: 1}
	c.Mempool.cachedProposal.Store(expected)

	p, ok = c.GetProposalBlockFromMempool()
	require.True(t, ok)
	require.Equal(t, expected, p)
}

func TestHandleTransactionsOnlyMarksDirtyOnSuccessfulNewTx(t *testing.T) {
	key, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	tx, errI := fsm.NewSendTransaction(key, key.PublicKey().Address(), 1, 1, 1, 1, 1, "")
	require.NoError(t, errI)

	txBytes, errI := lib.Marshal(tx)
	require.NoError(t, errI)

	m := &Mempool{
		Mempool: lib.NewMempool(lib.DefaultMempoolConfig()),
		L:       &sync.Mutex{},
	}

	require.NoError(t, m.HandleTransactions(txBytes))
	require.EqualValues(t, 1, m.dirtyVersion.Load())

	require.NoError(t, m.HandleTransactions(txBytes))
	require.EqualValues(t, 1, m.dirtyVersion.Load())

	require.Error(t, m.HandleTransactions([]byte("bad-tx")))
	require.EqualValues(t, 1, m.dirtyVersion.Load())
}
