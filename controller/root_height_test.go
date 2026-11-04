package controller

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/store"
)

// testRCManager is a minimal root chain manager that only reports the latest root height
type testRCManager struct {
	lib.RCManagerI
	height uint64
}

// GetHeight() returns the configured latest root height
func (t *testRCManager) GetHeight(_ uint64) uint64 { return t.height }

// rootHeightTestChain is a minimal committed chain used to exercise the active root height logic
type rootHeightTestChain struct {
	t  *testing.T
	db lib.StoreI
	c  *Controller
	rc *testRCManager
}

// newRootHeightTestChain() creates a chain with a committed genesis state that sets the root chain id
func newRootHeightTestChain(t *testing.T, chainId, rootChainId, latestRootHeight uint64) *rootHeightTestChain {
	t.Helper()
	log := lib.NewNullLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	// write the root chain id into the genesis state so every historical version has one
	genesis := &fsm.StateMachine{}
	genesis.SetStore(db)
	consParams := fsm.DefaultParams().Consensus
	consParams.RootChainId = rootChainId
	require.NoError(t, genesis.SetParamsCons(consParams))
	_, err = db.Commit()
	require.NoError(t, err)
	// create the state machine on top of the committed genesis
	config := lib.DefaultConfig()
	config.ChainId = chainId
	sm, err := fsm.New(config, db, nil, nil, log)
	require.NoError(t, err)
	rc := &testRCManager{height: latestRootHeight}
	return &rootHeightTestChain{t: t, db: db, rc: rc, c: &Controller{FSM: sm, Config: config, RCManager: rc, log: log}}
}

// commit() commits the next block with a certificate signed at rootHeight that records rootBuildHeight
func (r *rootHeightTestChain) commit(rootHeight, rootBuildHeight uint64) {
	r.t.Helper()
	height := r.c.FSM.Height()
	require.NoError(r.t, r.db.IndexQC(&lib.QuorumCertificate{
		Header:  &lib.View{Height: height, RootHeight: rootHeight, ChainId: r.c.Config.ChainId},
		Results: &lib.CertificateResult{RootBuildHeight: rootBuildHeight},
	}))
	require.NoError(r.t, r.db.IndexBlock(&lib.BlockResult{BlockHeader: &lib.BlockHeader{Height: height}}))
	_, err := r.db.Commit()
	require.NoError(r.t, err)
	_, err = r.c.FSM.Initialize(r.db)
	require.NoError(r.t, err)
}

// setRootChainId() changes the root chain id in the state of the block being built
func (r *rootHeightTestChain) setRootChainId(rootChainId uint64) {
	r.t.Helper()
	consParams, err := r.c.FSM.GetParamsCons()
	require.NoError(r.t, err)
	consParams.RootChainId = rootChainId
	require.NoError(r.t, r.c.FSM.SetParamsCons(consParams))
}

func TestActiveRootHeight(t *testing.T) {
	const nestedId, rootId, latest = uint64(2), uint64(1), uint64(110)
	tests := []struct {
		name           string
		detail         string
		chainId        uint64
		build          func(r *rootHeightTestChain)
		expectedHeight uint64
		expectedPinned bool
	}{
		{
			name:           "own root",
			detail:         "an own root chain always uses its latest root height",
			chainId:        rootId,
			build:          func(r *rootHeightTestChain) { r.commit(100, 103) },
			expectedHeight: latest,
		},
		{
			name:           "no committed block",
			detail:         "the first height has no record to pin to",
			chainId:        nestedId,
			build:          func(r *rootHeightTestChain) {},
			expectedHeight: latest,
		},
		{
			name:           "pre-upgrade block",
			detail:         "a last block without a root build height falls back to the latest root height",
			chainId:        nestedId,
			build:          func(r *rootHeightTestChain) { r.commit(100, 0) },
			expectedHeight: latest,
		},
		{
			name:           "pinned to the recorded root build height",
			detail:         "the recorded root build height pins the committee of the next height",
			chainId:        nestedId,
			build:          func(r *rootHeightTestChain) { r.commit(100, 103) },
			expectedHeight: 103,
			expectedPinned: true,
		},
		{
			name:           "never below the last signing committee",
			detail:         "if the last certificate was signed at a newer root height, use it",
			chainId:        nestedId,
			build:          func(r *rootHeightTestChain) { r.commit(105, 100) },
			expectedHeight: 105,
			expectedPinned: true,
		},
		{
			name:    "root chain switch",
			detail:  "a record from the previous root chain is ignored after a root chain switch",
			chainId: nestedId,
			build: func(r *rootHeightTestChain) {
				r.commit(100, 103)
				r.setRootChainId(3)
				r.commit(104, 106)
			},
			expectedHeight: latest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newRootHeightTestChain(t, test.chainId, rootId, latest)
			test.build(r)
			height, pinned := r.c.ActiveRootHeight()
			require.Equal(t, test.expectedHeight, height, test.detail)
			require.Equal(t, test.expectedPinned, pinned, test.detail)
		})
	}
}

func TestCheckRootBuildHeight(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		chainId          uint64
		latest           uint64
		rcBuildHeight    uint64
		lockedReproposal bool
		expectErr        bool
	}{
		{name: "valid", detail: "between the record and the latest root height", chainId: 2, latest: 110, rcBuildHeight: 105},
		{name: "equal to the record", detail: "the record may stay the same", chainId: 2, latest: 110, rcBuildHeight: 100},
		{name: "zero", detail: "an empty root build height can't pin a committee", chainId: 2, latest: 110, rcBuildHeight: 0, expectErr: true},
		{name: "backwards", detail: "the record must never go backwards", chainId: 2, latest: 110, rcBuildHeight: 99, expectErr: true},
		{name: "ahead of replica", detail: "the replica can't load a committee it doesn't know", chainId: 2, latest: 110, rcBuildHeight: 111, expectErr: true},
		{name: "too stale", detail: "lagging more than the maximum delays rotation", chainId: 2, latest: 100 + lib.MaxRootBuildHeightLag + 1, rcBuildHeight: 100, expectErr: true},
		{name: "stale locked reproposal", detail: "a locked proposal can't change its contents", chainId: 2, latest: 100 + lib.MaxRootBuildHeightLag + 1, rcBuildHeight: 100, lockedReproposal: true},
		{name: "own root", detail: "own root chains aren't checked", chainId: 1, latest: 110, rcBuildHeight: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newRootHeightTestChain(t, test.chainId, 1, test.latest)
			r.commit(100, 100)
			err := r.c.CheckRootBuildHeight(test.rcBuildHeight, test.lockedReproposal)
			require.Equal(t, test.expectErr, err != nil, test.detail)
		})
	}
}

func TestCheckActiveRootHeight(t *testing.T) {
	tests := []struct {
		name       string
		detail     string
		record     uint64
		rootHeight uint64
		expectErr  bool
	}{
		{name: "pinned committee", detail: "signed by the pinned committee", record: 103, rootHeight: 103},
		{name: "newer committee", detail: "a newer committee is accepted", record: 103, rootHeight: 110},
		{name: "older committee", detail: "an older committee than the pinned one is rejected", record: 103, rootHeight: 102, expectErr: true},
		{name: "not pinned", detail: "without a record nothing is enforced", record: 0, rootHeight: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newRootHeightTestChain(t, 2, 1, 110)
			r.commit(100, test.record)
			err := r.c.CheckActiveRootHeight(&lib.QuorumCertificate{Header: &lib.View{RootHeight: test.rootHeight}})
			require.Equal(t, test.expectErr, err != nil, test.detail)
		})
	}
}

func TestUpdateRootChainInfoReset(t *testing.T) {
	validators := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{{VotingPower: 1}}}
	tests := []struct {
		name          string
		detail        string
		chainId       uint64
		lastValidator *lib.ConsensusValidators
		expectReset   bool
		expectNewComm bool
	}{
		{name: "nested", detail: "root updates never reset a nested chain's bft", chainId: 2, lastValidator: validators},
		{name: "nested bootstrap", detail: "a newly formed committee still kicks off consensus", chainId: 2, lastValidator: nil, expectReset: true},
		{name: "own root", detail: "own root chains keep resetting on their own updates", chainId: 1, lastValidator: validators, expectReset: true, expectNewComm: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newRootHeightTestChain(t, test.chainId, 1, 110)
			r.c.Mempool = &Mempool{}
			r.c.Consensus = &bft.BFT{ResetBFT: make(chan bft.ResetBFT, 1)}
			r.c.UpdateRootChainInfo(&lib.RootChainInfo{
				RootChainId:      1,
				Height:           110,
				ValidatorSet:     &lib.ConsensusValidators{},
				LastValidatorSet: test.lastValidator,
			})
			require.Equal(t, test.expectReset, len(r.c.Consensus.ResetBFT) == 1, test.detail)
			if test.expectReset {
				require.Equal(t, test.expectNewComm, (<-r.c.Consensus.ResetBFT).IsRootChainUpdate, test.detail)
			}
		})
	}
}
