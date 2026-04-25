package controller

import (
	"sync"
	"testing"

	"github.com/canopy-network/canopy/lib"
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
