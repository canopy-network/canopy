package canoliq

import (
	"bytes"
	"testing"

	"github.com/canopy-network/go-plugin/contract"
)

// accountaddress_test.go covers the pinned-height switch that makes credited
// account records carry their address, and the one-time migration that fixes
// the records written before it (accountaddress.go).

var (
	addrSender   = parityHex("a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1")
	addrNewRecip = parityHex("b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2")
	addrOldRecip = parityHex("c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3")
)

func loadAccountRecord(t *testing.T, s *fakeStore, addr []byte) *contract.Account {
	t.Helper()
	bz := s.get(contract.KeyForAccount(addr))
	if bz == nil {
		t.Fatalf("no account record for %x", addr)
	}
	a := new(contract.Account)
	if err := contract.Unmarshal(bz, a); err != nil {
		t.Fatal(err)
	}
	return a
}

func seedAccountRecord(t *testing.T, s *fakeStore, addr []byte, acct *contract.Account) {
	t.Helper()
	bz, err := contract.Marshal(acct)
	if err != nil {
		t.Fatal(err)
	}
	s.set(contract.KeyForAccount(addr), bz)
}

func sendAt(t *testing.T, c *Canoliq, h uint64, from, to []byte, amount uint64) {
	t.Helper()
	c.plugin.setHeight(h)
	r := c.deliverMessageSend(&contract.MessageSend{FromAddress: from, ToAddress: to, Amount: amount}, 10_000)
	if r.Error != nil {
		t.Fatalf("send at %d: %v", h, r.Error)
	}
}

// Before the activation height a send must write exactly the bytes the
// running release writes: a new recipient's record has no address.
func TestSendBeforeAddressFixHeightLeavesRecordUnchanged(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedAccountRecord(t, s, addrSender, &contract.Account{Amount: 100_000_000})
	sendAt(t, c, MainnetAccountAddressFixHeight-1, addrSender, addrNewRecip, 1_000_000)

	got := loadAccountRecord(t, s, addrNewRecip)
	if len(got.Address) != 0 || got.Amount != 1_000_000 {
		t.Fatalf("pre-fix recipient record changed: %+v", got)
	}
	want, _ := contract.Marshal(&contract.Account{Amount: 1_000_000})
	if !bytes.Equal(s.get(contract.KeyForAccount(addrNewRecip)), want) {
		t.Fatal("pre-fix recipient bytes differ from the running release")
	}
	if a := loadAccountRecord(t, s, addrSender); len(a.Address) != 0 {
		t.Fatalf("pre-fix sender record gained an address: %x", a.Address)
	}
}

func TestSendFromAddressFixHeightSetsAddresses(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedAccountRecord(t, s, addrSender, &contract.Account{Amount: 100_000_000})
	sendAt(t, c, MainnetAccountAddressFixHeight, addrSender, addrNewRecip, 1_000_000)

	to := loadAccountRecord(t, s, addrNewRecip)
	if !bytes.Equal(to.Address, addrNewRecip) || to.Amount != 1_000_000 {
		t.Fatalf("recipient: %+v", to)
	}
	from := loadAccountRecord(t, s, addrSender)
	if !bytes.Equal(from.Address, addrSender) || from.Amount != 100_000_000-1_000_000-10_000 {
		t.Fatalf("sender: %+v", from)
	}
}

func TestSendToSelfKeepsOneRecord(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	seedAccountRecord(t, s, addrSender, &contract.Account{Amount: 100_000_000})
	sendAt(t, c, MainnetAccountAddressFixHeight, addrSender, addrSender, 1_000_000)
	a := loadAccountRecord(t, s, addrSender)
	if !bytes.Equal(a.Address, addrSender) || a.Amount != 100_000_000-10_000 {
		t.Fatalf("self send: %+v", a)
	}
}

func TestNonMainnetProfilesSetAddressFromGenesis(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedAccountRecord(t, s, addrSender, &contract.Account{Amount: 100_000_000})
	sendAt(t, c, 1, addrSender, addrNewRecip, 1_000_000)
	if a := loadAccountRecord(t, s, addrNewRecip); !bytes.Equal(a.Address, addrNewRecip) {
		t.Fatalf("recipient address %x", a.Address)
	}
}

func TestTreasurySpendSetsRecipientAddressFromFixHeight(t *testing.T) {
	for _, tc := range []struct {
		h    uint64
		want []byte
	}{
		{MainnetAccountAddressFixHeight - 1, nil},
		{MainnetAccountAddressFixHeight, addrNewRecip},
	} {
		c, s := newMainnetFixCanoliq(t)
		s.set(KeyForTreasuryCNPY(), EncodeUint64(50_000_000))
		c.plugin.setHeight(tc.h)
		spend := &contract.TreasurySpend{Id: 1, Payload: &contract.ProposalTreasurySpend{
			Recipient: addrNewRecip, Amount: 5_000_000, Denomination: contract.SpendDenomination_SPEND_CNPY,
		}}
		if err := c.applySpend(spend); err != nil {
			t.Fatalf("height %d: %v", tc.h, err)
		}
		a := loadAccountRecord(t, s, addrNewRecip)
		if !bytes.Equal(a.Address, tc.want) || a.Amount != 5_000_000 {
			t.Fatalf("height %d: recipient %+v", tc.h, a)
		}
	}
}

func TestAccountAddressMigration(t *testing.T) {
	c, s := newMainnetFixCanoliq(t)
	// Credited before the fix: no address. Already correct: written by core.
	seedAccountRecord(t, s, addrNewRecip, &contract.Account{Amount: 76_000_000_000})
	seedAccountRecord(t, s, addrOldRecip, &contract.Account{Address: addrOldRecip, Amount: 42})
	untouched := append([]byte(nil), s.get(contract.KeyForAccount(addrOldRecip))...)

	// Any other height is a no-op.
	for _, h := range []uint64{MainnetAccountAddressFixHeight - 1, MainnetAccountAddressFixHeight + 1} {
		if err := c.applyMainnetAccountAddressMigration(h); err != nil {
			t.Fatal(err)
		}
		if a := loadAccountRecord(t, s, addrNewRecip); len(a.Address) != 0 {
			t.Fatalf("migration ran at height %d", h)
		}
	}

	c.plugin.setHeight(MainnetAccountAddressFixHeight)
	if r := c.BeginBlock(&contract.PluginBeginRequest{Height: MainnetAccountAddressFixHeight}); r.Error != nil {
		t.Fatal(r.Error)
	}
	a := loadAccountRecord(t, s, addrNewRecip)
	if !bytes.Equal(a.Address, addrNewRecip) || a.Amount != 76_000_000_000 {
		t.Fatalf("migrated record: %+v", a)
	}
	if !bytes.Equal(s.get(contract.KeyForAccount(addrOldRecip)), untouched) {
		t.Fatal("a record that already had its address was rewritten")
	}

	// Idempotent.
	before := append([]byte(nil), s.get(contract.KeyForAccount(addrNewRecip))...)
	if err := c.applyMainnetAccountAddressMigration(MainnetAccountAddressFixHeight); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s.get(contract.KeyForAccount(addrNewRecip)), before) {
		t.Fatal("second run changed the record")
	}
}

func TestMigrationOnlyRunsOnMainnet(t *testing.T) {
	c, s := newTestCanoliq()
	seedParams(t, c, DefaultParams())
	seedAccountRecord(t, s, addrNewRecip, &contract.Account{Amount: 1})
	if err := c.applyMainnetAccountAddressMigration(MainnetAccountAddressFixHeight); err != nil {
		t.Fatal(err)
	}
	if a := loadAccountRecord(t, s, addrNewRecip); len(a.Address) != 0 {
		t.Fatal("migration ran on a non-mainnet profile")
	}
}

func TestAddressFromAccountKey(t *testing.T) {
	if got := addressFromAccountKey(contract.KeyForAccount(addrSender)); !bytes.Equal(got, addrSender) {
		t.Fatalf("got %x", got)
	}
	for _, k := range [][]byte{
		accountsPrefix(),
		contract.KeyForAccount(addrSender[:19]),
		append(contract.KeyForAccount(addrSender), 0),
		KeyForTreasuryCNPY(),
	} {
		if addressFromAccountKey(k) != nil {
			t.Fatalf("accepted %x", k)
		}
	}
}
