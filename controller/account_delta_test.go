package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// heights below 2 have no committed block to pair with and must be rejected before any
// store or FSM access (matching fsm.IndexerBlob's own height<=1 guard)
func TestGetAccountDelta_RejectsHeightsBelowTwo(t *testing.T) {
	c := &Controller{}
	require.Nil(t, c.FSM)
	for _, height := range []uint64{0, 1} {
		t.Run(fmt.Sprintf("height %d", height), func(t *testing.T) {
			delta, err := c.GetAccountDelta(context.Background(), height)
			require.NotNil(t, err)
			require.Equal(t, lib.CodeWrongBlockHeight, err.Code())
			require.Nil(t, delta)
		})
	}
}
