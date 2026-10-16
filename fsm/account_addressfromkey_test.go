package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

// TestGetAccountsRecoversAddressFromKey ensures the bulk account getters report
// an address even when the stored value does not carry one.
//
// Accounts are keyed by address, so the key is authoritative. Records written
// without the address in the value still exist in live state (observed on the
// canoLiq devnet: untouched genesis accounts returned "address": "" from
// /v1/query/accounts, while the same accounts resolved correctly through the
// single-account query, which derives the address from the request). Those
// bytes are never rewritten unless the account transacts, so the read path has
// to tolerate them.
func TestGetAccountsRecoversAddressFromKey(t *testing.T) {
	addr := newTestAddress(t)

	// write an account record whose marshalled value omits the address,
	// reproducing the legacy on-disk shape
	writeAddresslessAccount := func(t *testing.T, sm StateMachine, a crypto.AddressI, amount uint64) {
		t.Helper()
		bz, err := sm.marshalAccount(&Account{Amount: amount}) // no Address
		require.NoError(t, err)
		require.NoError(t, sm.Set(KeyForAccount(a), bz))
	}

	t.Run("GetAccounts", func(t *testing.T) {
		sm := newTestStateMachine(t)
		writeAddresslessAccount(t, sm, addr, 100000000)

		got, err := sm.GetAccounts()
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, addr.Bytes(), got[0].Address,
			"address must be recovered from the state key")
		require.EqualValues(t, 100000000, got[0].Amount)
	})

	t.Run("GetAccountsPaginated", func(t *testing.T) {
		sm := newTestStateMachine(t)
		writeAddresslessAccount(t, sm, addr, 100000000)

		page, err := sm.GetAccountsPaginated(lib.PageParams{PageNumber: 1, PerPage: 10})
		require.NoError(t, err)
		results, ok := page.Results.(*AccountPage)
		require.True(t, ok)
		require.Len(t, *results, 1)
		require.Equal(t, addr.Bytes(), (*results)[0].Address,
			"address must be recovered from the state key")
	})
}
