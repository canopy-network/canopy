package controller

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/stretchr/testify/require"
)

/*
This file covers controller.GetAccountDelta's REPLAY branch (the fall-through when the tip
cache misses), which the cache-hit tests in account_delta_test.go deliberately never reach:
the TimeMachine snapshot reads, the above-tip clamp guard, the block fetch + height check,
the QC fetch (and its blockHeight>1 gating), the RootDexBatch restore, and a full happy-path
replay against a REAL committed chain.

The chain fixture mirrors fsm/indexer_test.go's newTestAccountDeltaChain (which controller
tests cannot call -- test helpers don't cross packages), built here from exported fsm APIs
plus the same reflection seam cmd/rpc/query_test.go uses to construct a StateMachine outside
the fsm package.
*/

// testChainTimestamp is the fixed base block time the chain fixture uses (same date as the
// fsm package's fixtures)
var testChainTimestamp = uint64(time.Date(2024, 02, 01, 0, 0, 0, 0, time.UTC).UnixMicro())

// testChainBLSKeys are the same deterministic BLS12381 private keys the fsm package's test
// fixtures use (fsm/state_test.go newTestKeyGroup), so this fixture's validator set and
// QC-signing behavior mirror the proven-good fsm chain fixture exactly
var testChainBLSKeys = []string{
	"01553a101301cd7019b78ffa1186842dd93923e563b8ae22e2ab33ae889b23ee",
	"1b6b244fbdf614acb5f0d00a2b56ffcbe2aa23dabd66365dffcd3f06491ae50a",
	"2ee868f74134032eacba191ca529115c64aa849ac121b75ca79b37420a623036",
	"3e3ab94c10159d63a12cb26aca4b0e76070a987d49dd10fc5f526031e05801da",
	"479839d3edbd0eefa60111db569ded6a1a642cc84781600f0594bd8d4a429319",
	"51eb5eb6eca0b47c8383652a6043aadc66ddbcbe240474d152f4d9a7439eae42",
	"637cb8e916bba4c1773ed34d89ebc4cb86e85c145aea5653a58de930590a2aa4",
	"7235e5757e6f52e6ae4f9e20726d9c514281e58e839e33a7f667167c524ff658",
}

func testChainKeyGroup(t *testing.T, i int) *crypto.KeyGroup {
	t.Helper()
	key, err := crypto.StringToBLS12381PrivateKey(testChainBLSKeys[i])
	require.NoError(t, err)
	return crypto.NewKeyGroup(key)
}

func testChainAddress(t *testing.T, i int) crypto.AddressI {
	t.Helper()
	return testChainKeyGroup(t, i).Address
}

func testChainAddressBytes(t *testing.T, i int) []byte {
	t.Helper()
	return testChainAddress(t, i).Bytes()
}

// setUnexportedField / setFSMHeight / setFSMCacheMaps are the same reflection seam
// cmd/rpc/query_test.go uses to build an fsm.StateMachine outside the fsm package
func setUnexportedField(t *testing.T, target any, name string, value any) {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(name)
	require.True(t, field.IsValid(), name)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func setFSMHeight(t *testing.T, sm *fsm.StateMachine, height uint64) {
	t.Helper()
	setUnexportedField(t, sm, "height", height)
}

func setFSMCacheMaps(t *testing.T, sm *fsm.StateMachine) {
	t.Helper()
	field := reflect.ValueOf(sm).Elem().FieldByName("cache")
	require.True(t, field.IsValid())
	cacheValue := reflect.New(field.Type().Elem())
	// ApplyBlock touches both the account and the pool caches, so both maps must exist
	for _, name := range []string{"accounts", "pools"} {
		mapField := cacheValue.Elem().FieldByName(name)
		require.True(t, mapField.IsValid(), name)
		reflect.NewAt(mapField.Type(), unsafe.Pointer(mapField.UnsafeAddr())).Elem().Set(reflect.MakeMap(mapField.Type()))
	}
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(cacheValue)
}

// newReflectedFSM builds a StateMachine over db at the given height, mirroring the fsm
// package's newTestStateMachine. The config's P2P NetworkID must agree with the hand-set
// NetworkID: TimeMachine snapshots derive theirs from Config (newStateMachine), and a
// mismatch would fail every replayed transaction's network-id check.
func newReflectedFSM(t *testing.T, db lib.StoreI, log lib.LoggerI, height uint64) *fsm.StateMachine {
	t.Helper()
	sm := &fsm.StateMachine{
		ProtocolVersion: 1,
		NetworkID:       1,
		Config: lib.Config{
			MainConfig:         lib.DefaultMainConfig(),
			StateMachineConfig: lib.DefaultStateMachineConfig(),
		},
	}
	sm.Config.P2PConfig.NetworkID = 1
	setUnexportedField(t, sm, "store", db)
	setUnexportedField(t, sm, "height", height)
	setUnexportedField(t, sm, "slashTracker", fsm.NewSlashTracker())
	setUnexportedField(t, sm, "proposeVoteConfig", fsm.AcceptAllProposals)
	setUnexportedField(t, sm, "events", new(lib.EventsTracker))
	setUnexportedField(t, sm, "log", log)
	setFSMCacheMaps(t, sm)
	return sm
}

// newBareReplayController builds a Controller over an FSM whose in-memory store has
// `versions` committed versions but no real chain -- just enough for GetAccountDelta's
// snapshot reads to run. blockAtLastVersion, if non-nil, is indexed into the final
// version's commit. NOTE on heights: the store package keeps a GLOBAL block cache keyed
// only by height, so fixtures in this package must use heights no other fixture indexes
// when they rely on a block being MISSING.
func newBareReplayController(t *testing.T, versions uint64, blockAtLastVersion *lib.BlockResult) *Controller {
	t.Helper()
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	sm := newReflectedFSM(t, db, log, 0)
	for v := uint64(1); v <= versions; v++ {
		if v == versions && blockAtLastVersion != nil {
			require.NoError(t, db.IndexBlock(blockAtLastVersion))
		}
		_, e := db.Commit()
		require.NoError(t, e)
	}
	setFSMHeight(t, sm, db.Version())
	sm.Reset()
	return &Controller{FSM: sm, log: log}
}

// signTestChainQC signs qc with 3 of the fixture's 4 committee validators (mirrors the fsm
// fixtures' QC signing)
func signTestChainQC(t *testing.T, sm *fsm.StateMachine, qc *lib.QuorumCertificate) {
	t.Helper()
	committee, err := sm.GetCommitteeMembers(lib.CanopyChainId)
	require.NoError(t, err)
	mk := committee.MultiKey.Copy()
	for i := 0; i < 3; i++ {
		privateKey := testChainKeyGroup(t, i).PrivateKey
		for j, pubKey := range mk.PublicKeys() {
			if privateKey.PublicKey().Equals(pubKey) {
				require.NoError(t, mk.AddSigner(privateKey.Sign(qc.SignBytes()), j))
			}
		}
	}
	aggSig, e := mk.AggregateSignatures()
	require.NoError(t, e)
	qc.Signature = &lib.AggregateSignature{Signature: aggSig, Bitmap: mk.Bitmap()}
}

// testChainSendTx builds a signed zero-fee send transaction from test key group
// `fromKeyGroup`, valid at the state machine's current height
func testChainSendTx(t *testing.T, sm *fsm.StateMachine, fromKeyGroup int, to crypto.AddressI, amount uint64) []byte {
	t.Helper()
	kg := testChainKeyGroup(t, fromKeyGroup)
	tx, err := fsm.NewSendTransaction(kg.PrivateKey, to, amount, uint64(sm.NetworkID), sm.Config.ChainId, 0, sm.Height(), "")
	require.NoError(t, err)
	bz, mErr := lib.Marshal(tx)
	require.NoError(t, mErr)
	return bz
}

// applyAndCommitTestChainBlock applies a block at sm.Height() with the given transactions,
// indexes the resulting BlockResult and a signed certificate for that height, commits, and
// advances the state machine -- the same order controller.CommitCertificate uses. qcMutate,
// if non-nil, edits the certificate BEFORE signing (used to vary Results for the
// RootDexBatch restore tests).
func applyAndCommitTestChainBlock(t *testing.T, sm *fsm.StateMachine, db lib.StoreI, txs [][]byte, qcMutate func(*lib.QuorumCertificate)) {
	t.Helper()
	height := sm.Height()
	block := &lib.Block{
		BlockHeader: &lib.BlockHeader{
			Height:          height,
			Time:            testChainTimestamp + height*1_000_000,
			ProposerAddress: testChainAddressBytes(t, 0),
		},
		Transactions: txs,
	}
	header, result, err := sm.ApplyBlock(context.Background(), block, false, nil, false)
	require.NoError(t, err)
	for _, f := range result.Failed {
		t.Logf("FAILED TX at height %d: %s", height, f.Error.Error())
	}
	require.Empty(t, result.Failed, "block at height %d had failed transactions", height)
	// ChainId is what keys the committee data the reward machinery pays out from; the
	// recipient is validator #3 (non-compounding), so its reward lands on its OUTPUT
	// account (key group 0) -- mirrors fsm's testAccountDeltaQC
	qc := &lib.QuorumCertificate{
		Header: &lib.View{Height: height, ChainId: lib.CanopyChainId},
		Results: &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{
				Address: testChainAddressBytes(t, 3),
				Percent: 100,
				ChainId: lib.CanopyChainId,
			}},
		}},
	}
	if qcMutate != nil {
		qcMutate(qc)
	}
	signTestChainQC(t, sm, qc)
	require.NoError(t, db.IndexQC(qc))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader:  header,
		Transactions: result.Results,
		Events:       result.Events,
	}))
	_, e := db.Commit()
	require.NoError(t, e)
	setFSMHeight(t, sm, height+1)
	sm.Reset()
	require.Equal(t, sm.Height(), db.Version())
}

// newAccountDeltaChainController builds a Controller over a REAL committed chain (blocks 3,
// 4 and 5 applied via ApplyBlock + IndexQC + IndexBlock + Commit), mirroring fsm's
// newTestAccountDeltaChain. Block 5 -- the one GetAccountDelta(6) replays -- exercises a
// brand-new account (added), balance changes (changed), a net-unchanged self-send (dropped),
// and an account zeroed out (removed). finalQCMutate, if non-nil, edits block 5's
// certificate before it is signed and indexed.
func newAccountDeltaChainController(t *testing.T, finalQCMutate func(*lib.QuorumCertificate)) *Controller {
	t.Helper()
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	sm := newReflectedFSM(t, db, log, 2)
	// -- base state at heights <= 2 (mirrors fsm's newTestStateMachine + newTestApplyBlockFixture) --
	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	_, e := db.Commit()
	require.NoError(t, e)
	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	// preset the 'last block' so block 3's application can load its predecessor
	require.NoError(t, db.IndexBlock(&lib.BlockResult{BlockHeader: &lib.BlockHeader{
		Height: 2,
		Hash:   crypto.Hash([]byte("block_hash")),
		Time:   testChainTimestamp,
	}}))
	require.NoError(t, sm.UpdateParam("fee", fsm.ParamSendFee, &lib.UInt64Wrapper{Value: 1}))
	require.NoError(t, sm.AccountAdd(testChainAddress(t, 0), 2))
	supply := &fsm.Supply{}
	for i := 0; i < 4; i++ {
		require.NoError(t, sm.SetValidators([]*fsm.Validator{{
			Address:      testChainAddressBytes(t, i),
			PublicKey:    testChainKeyGroup(t, i).PublicKey.Bytes(),
			StakedAmount: 100,
			Committees:   []uint64{lib.CanopyChainId},
			Output:       testChainAddressBytes(t, 0),
		}}, supply))
		require.NoError(t, sm.SetCommitteeMember(testChainAddress(t, i), lib.CanopyChainId, 100))
	}
	require.NoError(t, sm.SetSupply(supply))
	qc2 := &lib.QuorumCertificate{
		Header: &lib.View{Height: 2},
		Results: &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{Address: testChainAddressBytes(t, 0), Percent: 100}},
		}},
	}
	signTestChainQC(t, sm, qc2)
	require.NoError(t, db.IndexQC(qc2))
	setFSMHeight(t, sm, 3)
	_, e = db.Commit()
	require.NoError(t, e)
	// -- the chain itself (mirrors fsm's newTestAccountDeltaChain) --
	// align the store version with sm.Height()
	_, e = db.Commit()
	require.NoError(t, e)
	require.Equal(t, sm.Height(), db.Version())
	// send fee of zero, so the net-unchanged self-send below really is net-unchanged
	require.NoError(t, sm.UpdateParam("fee", fsm.ParamSendFee, &lib.UInt64Wrapper{Value: 0}))
	// funder pays for everything downstream
	require.NoError(t, sm.AccountAdd(testChainAddress(t, 0), 10_000_000))
	// block 3: no transactions -- exists so block 4's BeginBlock has a real committed
	// predecessor block and certificate to load
	applyAndCommitTestChainBlock(t, sm, db, nil, nil)
	// block 4: fund the accounts block 5 will change / zero / self-send
	applyAndCommitTestChainBlock(t, sm, db, [][]byte{
		testChainSendTx(t, sm, 0, testChainAddress(t, 4), 5_000), // net-unchanged self-sender
		testChainSendTx(t, sm, 0, testChainAddress(t, 5), 3_000), // zeroed out in block 5
		testChainSendTx(t, sm, 0, testChainAddress(t, 6), 2_000), // changed in block 5
		testChainSendTx(t, sm, 0, testChainAddress(t, 3), 4_000), // reward-event recipient; untouched by block 5
	}, nil)
	// block 5: the measured block
	brandNew := crypto.NewAddressFromBytes(bytes.Repeat([]byte{0xAB}, crypto.AddressSize))
	applyAndCommitTestChainBlock(t, sm, db, [][]byte{
		testChainSendTx(t, sm, 0, brandNew, 1_500),               // brandNew added; kg0 changed
		testChainSendTx(t, sm, 6, testChainAddress(t, 7), 1_000), // kg6 changed; kg7 added (never funded)
		testChainSendTx(t, sm, 4, testChainAddress(t, 4), 5_000), // kg4 self-send: touched, net-unchanged
		testChainSendTx(t, sm, 5, testChainAddress(t, 6), 3_000), // kg5 sends its whole balance -> removed
	}, finalQCMutate)
	// sm.Height() is now 6; blocks 2..5 and their QCs are indexed, so GetAccountDelta(6)
	// (replay of block 5) resolves entirely from the store
	return &Controller{FSM: sm, Config: sm.Config, log: log}
}

// entryAddressSet collapses delta entries to their address set for order-independent compare
func entryAddressSet(entries []*fsm.AccountChangeEntry) map[string]struct{} {
	set := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		set[string(e.Address)] = struct{}{}
	}
	return set
}

func addressSet(addresses ...[]byte) map[string]struct{} {
	set := make(map[string]struct{}, len(addresses))
	for _, a := range addresses {
		set[string(a)] = struct{}{}
	}
	return set
}

// accountBytesAtVersion reads the raw stored account bytes at a state version through a
// throwaway snapshot -- the independent ground truth the replay's delta is checked against
func accountBytesAtVersion(t *testing.T, c *Controller, version uint64, address []byte) []byte {
	t.Helper()
	snap, err := c.FSM.TimeMachine(version)
	require.NoError(t, err)
	require.NotSame(t, c.FSM, snap)
	defer snap.Discard()
	bz, gErr := snap.Get(fsm.KeyForAccount(crypto.NewAddressFromBytes(address)))
	require.NoError(t, gErr)
	return bz
}

// requireDeltaMatchesCommittedState checks every entry's classification and values against
// the committed state at versions height-1 and height -- nothing is hand-seeded, so a replay
// that drifted from the real block application fails here
func requireDeltaMatchesCommittedState(t *testing.T, c *Controller, delta *fsm.AccountDelta, height uint64) {
	t.Helper()
	for _, e := range delta.Added {
		require.Nil(t, e.PrevValue, "an added account must have no previous value")
		require.Empty(t, accountBytesAtVersion(t, c, height-1, e.Address), "an added account must not exist at the previous version")
		require.Equal(t, accountBytesAtVersion(t, c, height, e.Address), e.FinalValue)
	}
	for _, e := range delta.Changed {
		require.Equal(t, accountBytesAtVersion(t, c, height-1, e.Address), e.PrevValue)
		require.Equal(t, accountBytesAtVersion(t, c, height, e.Address), e.FinalValue)
		require.NotEqual(t, e.PrevValue, e.FinalValue, "a changed account must actually change")
	}
	for _, e := range delta.Removed {
		require.Equal(t, accountBytesAtVersion(t, c, height-1, e.Address), e.PrevValue)
		require.Nil(t, e.FinalValue, "a removed account must have no final value")
		require.Empty(t, accountBytesAtVersion(t, c, height, e.Address), "a removed account must not exist at the current version")
	}
}

// TestGetAccountDelta_ReplayHappyPath replays block 5 of a real committed chain through the
// full production path and checks the classified delta against the committed state at
// versions 5 and 6
func TestGetAccountDelta_ReplayHappyPath(t *testing.T) {
	c := newAccountDeltaChainController(t, nil)
	delta, err := c.GetAccountDelta(context.Background(), 6)
	require.NoError(t, err)
	require.NotNil(t, delta)
	brandNew := bytes.Repeat([]byte{0xAB}, crypto.AddressSize)
	// exact address sets: kg4's net-unchanged self-send must NOT appear anywhere, and kg3
	// (funded in block 4, untouched by block 5) must not leak in either. kg0 is changed by
	// both its sends and the block's committee reward (validator #3 is non-compounding, so
	// its reward lands on its output account, kg0).
	require.Equal(t, addressSet(brandNew, testChainAddressBytes(t, 7)), entryAddressSet(delta.Added))
	require.Equal(t, addressSet(testChainAddressBytes(t, 0), testChainAddressBytes(t, 6)), entryAddressSet(delta.Changed))
	require.Equal(t, addressSet(testChainAddressBytes(t, 5)), entryAddressSet(delta.Removed))
	requireDeltaMatchesCommittedState(t, c, delta, 6)
}

// TestGetAccountDelta_ReplayCommittedQCVariants exercises the RootDexBatch restore branch:
// the committed certificate's Results feed SetRootDexCache before the replay, exactly as
// the live commit path does. A full nested-chain fixture (where the restored batch changes
// HandleDexBatch's behavior) is not constructible in-process without a root chain to sync
// against, so this covers the QC-read/restore decision logic on a root chain: a certificate
// CARRYING a RootDexBatch must be tolerated and restored without disturbing the delta (the
// cache only alters nested-chain replay), and a certificate with nil Results is normal and
// must not error.
func TestGetAccountDelta_ReplayCommittedQCVariants(t *testing.T) {
	expectDelta := func(t *testing.T, c *Controller) {
		delta, err := c.GetAccountDelta(context.Background(), 6)
		require.NoError(t, err)
		require.NotNil(t, delta)
		brandNew := bytes.Repeat([]byte{0xAB}, crypto.AddressSize)
		require.Equal(t, addressSet(brandNew, testChainAddressBytes(t, 7)), entryAddressSet(delta.Added))
		require.Equal(t, addressSet(testChainAddressBytes(t, 0), testChainAddressBytes(t, 6)), entryAddressSet(delta.Changed))
		require.Equal(t, addressSet(testChainAddressBytes(t, 5)), entryAddressSet(delta.Removed))
		requireDeltaMatchesCommittedState(t, c, delta, 6)
	}
	t.Run("certificate carrying a RootDexBatch is restored and tolerated", func(t *testing.T) {
		c := newAccountDeltaChainController(t, func(qc *lib.QuorumCertificate) {
			qc.Results.RootDexBatch = &lib.DexBatch{Committee: lib.CanopyChainId}
		})
		expectDelta(t, c)
	})
	t.Run("certificate with nil Results is normal", func(t *testing.T) {
		c := newAccountDeltaChainController(t, func(qc *lib.QuorumCertificate) {
			qc.Results = nil
		})
		expectDelta(t, c)
	})
}

// TestGetAccountDelta_ReplayMissingBlock: a request for a height whose block was never
// indexed must fail loudly. NOTE: the store returns a zero-value (non-nil) header for a
// missing block -- lib.Unmarshal(nil) fills nothing -- so the miss surfaces through the
// height-mismatch guard (got=0), not the nil-header guard, which is defense-in-depth for a
// nil blockResult the store API cannot actually produce.
func TestGetAccountDelta_ReplayMissingBlock(t *testing.T) {
	// heights in the 40s: the store's global block cache is keyed only by height, so this
	// fixture must use heights no other fixture in this package indexes (see
	// newBareReplayController)
	c := newBareReplayController(t, 42, nil)
	delta, err := c.GetAccountDelta(context.Background(), 42)
	require.Nil(t, delta)
	require.NotNil(t, err)
	require.Equal(t, lib.CodeWrongBlockHeight, err.Code())
	// got=0 pins WHICH guard fired: the missing block's zero-value header, not the clamp
	require.Contains(t, err.Error(), "got=0 | wanted=41")
}

// TestGetAccountDelta_ReplayAboveTipRejected: TimeMachine silently clamps a height above
// the tip to the tip; the explicit height check must convert that clamp into an error
// instead of replaying a different height's block
func TestGetAccountDelta_ReplayAboveTipRejected(t *testing.T) {
	c := newBareReplayController(t, 3, nil)
	delta, err := c.GetAccountDelta(context.Background(), 9)
	require.Nil(t, delta)
	require.NotNil(t, err)
	require.Equal(t, lib.CodeWrongBlockHeight, err.Code())
	// got=3 (the clamped tip) pins the above-tip guard, not the missing-block guard
	require.Contains(t, err.Error(), "got=3 | wanted=9")
}

// TestGetAccountDelta_ReplayMissingQC: a block whose certificate cannot be loaded must fail
// loudly (a missing QC unmarshals into an empty, non-nil certificate with a nil Header) --
// whether a RootDexBatch existed is unknowable, so the delta could be silently incomplete
func TestGetAccountDelta_ReplayMissingQC(t *testing.T) {
	// height 60: unique to this fixture (global block cache, see newBareReplayController)
	c := newBareReplayController(t, 61, &lib.BlockResult{BlockHeader: &lib.BlockHeader{
		Height: 60,
		Hash:   crypto.Hash([]byte("qc-missing-block-60")),
		Time:   testChainTimestamp,
	}})
	delta, err := c.GetAccountDelta(context.Background(), 61)
	require.Nil(t, delta)
	require.NotNil(t, err)
	require.Equal(t, lib.CodeEmptyQuorumCertificate, err.Code())
}

// TestGetAccountDelta_HeightTwoBoundarySkipsQC: height 2 is the lowest servable height and
// replays block 1, the only block whose certificate is legitimately absent (BeginBlock
// skips certificate handling entirely at height <= 1). The QC fetch must be gated off for
// blockHeight 1 -- this fixture indexes NO certificate anywhere, so a regression that reads
// the QC unconditionally fails with ErrEmptyQuorumCertificate.
func TestGetAccountDelta_HeightTwoBoundarySkipsQC(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	sm := newReflectedFSM(t, db, log, 1)
	// hand-written pre-block-1 state: params for tx/block machinery, a validator set for
	// ApplyBlock's validator-root calculation
	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	supply := &fsm.Supply{}
	for i := 0; i < 4; i++ {
		require.NoError(t, sm.SetValidators([]*fsm.Validator{{
			Address:      testChainAddressBytes(t, i),
			PublicKey:    testChainKeyGroup(t, i).PublicKey.Bytes(),
			StakedAmount: 100,
			Committees:   []uint64{lib.CanopyChainId},
			Output:       testChainAddressBytes(t, 0),
		}}, supply))
		require.NoError(t, sm.SetCommitteeMember(testChainAddress(t, i), lib.CanopyChainId, 100))
	}
	require.NoError(t, sm.SetSupply(supply))
	// stub block 1 so ApplyBlock's last-block load (clamped to height 1) has a header
	require.NoError(t, db.IndexBlock(&lib.BlockResult{BlockHeader: &lib.BlockHeader{
		Height: 1,
		Hash:   crypto.Hash([]byte("boundary-stub-1")),
		Time:   testChainTimestamp,
	}}))
	_, e := db.Commit()
	require.NoError(t, e)
	sm.Reset()
	// apply and commit a REAL (empty) block 1; deliberately index NO certificate
	block1 := &lib.Block{BlockHeader: &lib.BlockHeader{
		Height:          1,
		Time:            testChainTimestamp + 1_000_000,
		ProposerAddress: testChainAddressBytes(t, 0),
	}}
	header, result, aErr := sm.ApplyBlock(context.Background(), block1, false, nil, false)
	require.NoError(t, aErr)
	require.Empty(t, result.Failed)
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader:  header,
		Transactions: result.Results,
		Events:       result.Events,
	}))
	_, e = db.Commit()
	require.NoError(t, e)
	setFSMHeight(t, sm, 2)
	sm.Reset()
	c := &Controller{FSM: sm, log: log}
	// must succeed -- in particular it must NOT be ErrEmptyQuorumCertificate
	delta, dErr := c.GetAccountDelta(context.Background(), 2)
	require.NoError(t, dErr)
	require.NotNil(t, delta)
	// an empty block 1 with no certificate to pay rewards from touches no accounts
	require.Empty(t, delta.Added)
	require.Empty(t, delta.Changed)
	require.Empty(t, delta.Removed)
}
