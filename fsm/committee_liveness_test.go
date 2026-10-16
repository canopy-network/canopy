package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

// enableCommitteeLiveness sets the governance protocol version so IsFeatureEnabled(3) returns true
func enableCommitteeLiveness(t *testing.T, sm *StateMachine) {
	consParams, err := sm.GetParamsCons()
	require.NoError(t, err)
	consParams.ProtocolVersion = NewProtocolVersion(0, 3)
	require.NoError(t, sm.SetParamsCons(consParams))
	require.True(t, sm.IsFeatureEnabled(3))
}

// livenessTestValidator creates a non-delegate validator staked for chainId using test key group `idx`
func livenessTestValidator(t *testing.T, idx int, stake uint64, chainId uint64) *Validator {
	kg := newTestKeyGroup(t, idx)
	return &Validator{
		Address:      kg.Address.Bytes(),
		PublicKey:    kg.PublicKey.Bytes(),
		StakedAmount: stake,
		Committees:   []uint64{chainId},
		Output:       kg.Address.Bytes(),
	}
}

// TestCommitteeLivenessBootstrap ensures that, with the feature enabled but no member having proven
// liveness yet, the committee is built with every member at full voting power (so a new chain can start)
func TestCommitteeLivenessBootstrap(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	chainId := lib.CanopyChainId
	a := livenessTestValidator(t, 0, 100, chainId)
	b := livenessTestValidator(t, 1, 100, chainId)
	require.NoError(t, sm.SetValidator(a))
	require.NoError(t, sm.SetValidator(b))
	// no active members yet -> bootstrap -> full power for everyone
	set, err := sm.GetCommitteeMembers(chainId)
	require.NoError(t, err)
	require.Equal(t, uint64(2), set.NumValidators)
	require.Equal(t, uint64(200), set.TotalPower)
	for _, m := range set.ValidatorSet.ValidatorSet {
		require.Equal(t, uint64(100), m.VotingPower)
	}
}

// TestCommitteeLivenessProvisionalHasNoPower is the core fix: a large validator that joins a running
// committee but has not proven liveness must join as a provisional (zero-power) member so it cannot
// inflate the quorum threshold and halt the chain.
func TestCommitteeLivenessProvisionalHasNoPower(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	chainId := lib.CanopyChainId
	// two small, liveness-proven validators
	a := livenessTestValidator(t, 0, 100, chainId)
	b := livenessTestValidator(t, 1, 100, chainId)
	// one large validator that has NOT proven liveness (just staked/joined)
	big := livenessTestValidator(t, 2, 10_000, chainId)
	for _, v := range []*Validator{a, b, big} {
		require.NoError(t, sm.SetValidator(v))
	}
	// mark the two small validators as active (they are actually running the chain and signing)
	require.NoError(t, sm.SetCommitteeMemberActive(chainId, crypto.NewAddressFromBytes(a.Address)))
	require.NoError(t, sm.SetCommitteeMemberActive(chainId, crypto.NewAddressFromBytes(b.Address)))
	// build the committee
	set, err := sm.GetCommitteeMembers(chainId)
	require.NoError(t, err)
	// all three are members (so the big one can vote to prove liveness)...
	require.Equal(t, uint64(3), set.NumValidators)
	// ...but only the active stake counts toward voting power and the quorum threshold
	require.Equal(t, uint64(200), set.TotalPower)
	require.Equal(t, uint64((2*200)/3+1), set.MinimumMaj23)
	// the active members (200 power) can reach quorum entirely on their own -> no halt
	require.GreaterOrEqual(t, uint64(200), set.MinimumMaj23)
	// verify the big validator is present but powerless
	bigVal, err := set.GetValidator(big.PublicKey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), bigVal.VotingPower)
}

// TestCommitteeLivenessDisabledUnchanged ensures pre-feature behavior is byte-identical: full power
// regardless of any activation state.
func TestCommitteeLivenessDisabledUnchanged(t *testing.T) {
	sm := newTestStateMachine(t)
	require.False(t, sm.IsFeatureEnabled(3))
	chainId := lib.CanopyChainId
	a := livenessTestValidator(t, 0, 100, chainId)
	big := livenessTestValidator(t, 2, 10_000, chainId)
	require.NoError(t, sm.SetValidator(a))
	require.NoError(t, sm.SetValidator(big))
	// even if some (irrelevant) activation key exists, feature-off ignores it
	require.NoError(t, sm.SetCommitteeMemberActive(chainId, crypto.NewAddressFromBytes(a.Address)))
	set, err := sm.GetCommitteeMembers(chainId)
	require.NoError(t, err)
	require.Equal(t, uint64(2), set.NumValidators)
	require.Equal(t, uint64(10_100), set.TotalPower)
}

// TestActivateCommitteeSigners verifies that signing a QC promotes a provisional member to active
func TestActivateCommitteeSigners(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	chainId := lib.CanopyChainId
	keyGroups := newTestKeyGroups(t, 3)
	committee := make([]*Validator, 0, 3)
	for i, kg := range keyGroups {
		committee = append(committee, &Validator{
			Address:      kg.Address.Bytes(),
			PublicKey:    kg.PublicKey.Bytes(),
			StakedAmount: 100,
			Committees:   []uint64{chainId},
			Output:       kg.Address.Bytes(),
		})
		require.NoError(t, sm.SetValidator(committee[i]))
	}
	// build a QC signed only by members 0 and 1 (member 2 does not sign)
	qc := newTestQC(t, testQCParams{
		height:        2,
		idxSigned:     map[int]bool{0: true, 1: true},
		committeeKeys: keyGroups,
		committee:     committee,
		results:       &lib.CertificateResult{RewardRecipients: &lib.RewardRecipients{}, SlashRecipients: new(lib.SlashRecipients)},
	})
	// build the validator set the QC was signed against
	var cvs []*lib.ConsensusValidator
	for _, m := range committee {
		cvs = append(cvs, &lib.ConsensusValidator{PublicKey: m.PublicKey, VotingPower: m.StakedAmount})
	}
	vs, err := lib.NewValidatorSet(&lib.ConsensusValidators{ValidatorSet: cvs})
	require.NoError(t, err)
	// activate signers
	require.NoError(t, sm.ActivateCommitteeSigners(qc, &vs))
	// members 0 and 1 should now be active; member 2 should not be
	for i, kg := range keyGroups {
		active, e := sm.IsCommitteeMemberActive(chainId, kg.Address)
		require.NoError(t, e)
		if i == 2 {
			require.False(t, active, "non-signer should not be active")
		} else {
			require.True(t, active, "signer should be active")
		}
	}
}

// TestDeleteActiveCommitteesOnLeave ensures activation is dropped only for committees the validator leaves
func TestDeleteActiveCommitteesOnLeave(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	kg := newTestKeyGroup(t, 0)
	addr := kg.Address
	// validator staked for chains 1 and 2, active in both
	val := &Validator{
		Address:      addr.Bytes(),
		PublicKey:    kg.PublicKey.Bytes(),
		StakedAmount: 100,
		Committees:   []uint64{lib.CanopyChainId, 2},
		Output:       addr.Bytes(),
	}
	require.NoError(t, sm.SetValidator(val))
	// seed committee supply so DeleteCommittees (inside UpdateCommittees) has balances to subtract
	require.NoError(t, sm.SetCommittees(addr, val.StakedAmount, val.Committees))
	require.NoError(t, sm.SetCommitteeMemberActive(lib.CanopyChainId, addr))
	require.NoError(t, sm.SetCommitteeMemberActive(2, addr))
	// edit-stake dropping committee 2 (keeping canopy)
	require.NoError(t, sm.UpdateCommittees(addr, val, val.StakedAmount, []uint64{lib.CanopyChainId}))
	// canopy activation kept, committee-2 activation dropped
	active1, err := sm.IsCommitteeMemberActive(lib.CanopyChainId, addr)
	require.NoError(t, err)
	require.True(t, active1)
	active2, err := sm.IsCommitteeMemberActive(2, addr)
	require.NoError(t, err)
	require.False(t, active2)
}

// TestCompoundingKeepsActivation ensures an edit-stake that does not change committees (e.g. reward
// auto-compounding) does not reset a validator's proven liveness.
func TestCompoundingKeepsActivation(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	kg := newTestKeyGroup(t, 0)
	addr := kg.Address
	val := &Validator{
		Address:      addr.Bytes(),
		PublicKey:    kg.PublicKey.Bytes(),
		StakedAmount: 100,
		Committees:   []uint64{lib.CanopyChainId},
		Output:       addr.Bytes(),
	}
	require.NoError(t, sm.SetValidator(val))
	// seed committee supply so DeleteCommittees (inside UpdateCommittees) has balances to subtract
	require.NoError(t, sm.SetCommittees(addr, val.StakedAmount, val.Committees))
	require.NoError(t, sm.SetCommitteeMemberActive(lib.CanopyChainId, addr))
	// same committees, higher stake (compounding)
	require.NoError(t, sm.UpdateCommittees(addr, val, val.StakedAmount+50, []uint64{lib.CanopyChainId}))
	active, err := sm.IsCommitteeMemberActive(lib.CanopyChainId, addr)
	require.NoError(t, err)
	require.True(t, active, "compounding must not reset liveness")
}

// TestHandleStalledCommitteeDemotesBlocker verifies the liveness watchdog: an already-active majority
// validator that goes dark (committee stalls) is demoted back to provisional so the remaining online
// members can reach quorum again.
func TestHandleStalledCommitteeDemotesBlocker(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	// ensure self is its own root (so the watchdog runs) and pick a nested committee id
	cons, err := sm.GetParamsCons()
	require.NoError(t, err)
	cons.RootChainId = sm.Config.ChainId
	require.NoError(t, sm.SetParamsCons(cons))
	nestedChainId := sm.Config.ChainId + 1
	// three validators all actively signing: a big (majority) one and two small ones
	big := livenessTestValidator(t, 2, 10_000, nestedChainId)
	a := livenessTestValidator(t, 0, 100, nestedChainId)
	b := livenessTestValidator(t, 1, 100, nestedChainId)
	for _, v := range []*Validator{big, a, b} {
		require.NoError(t, sm.SetValidator(v))
		require.NoError(t, sm.SetCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(v.Address)))
	}
	// sanity: before the stall, the big validator dominates the committee power
	set, err := sm.GetCommitteeMembers(nestedChainId)
	require.NoError(t, err)
	require.Equal(t, uint64(10_200), set.TotalPower)
	// record committee data as last updated at height 1, then advance well past the liveness window
	require.NoError(t, sm.SetCommitteesData(&lib.CommitteesData{List: []*lib.CommitteeData{
		{ChainId: nestedChainId, LastRootHeightUpdated: 1, LastChainHeightUpdated: 1},
	}}))
	sm.height = 1 + CommitteeLivenessWindow + 5
	// run the watchdog
	require.NoError(t, sm.HandleStalledCommittees())
	// the big (blocking) validator is demoted; the two small ones remain active
	bigActive, err := sm.IsCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(big.Address))
	require.NoError(t, err)
	require.False(t, bigActive, "the stalled majority validator should be demoted")
	for _, v := range []*Validator{a, b} {
		act, e := sm.IsCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(v.Address))
		require.NoError(t, e)
		require.True(t, act, "small online validators should keep their active status")
	}
	// the rebuilt committee now excludes the offline majority's power -> the online set can reach quorum
	set, err = sm.GetCommitteeMembers(nestedChainId)
	require.NoError(t, err)
	require.Equal(t, uint64(200), set.TotalPower)
	require.GreaterOrEqual(t, uint64(200), set.MinimumMaj23)
	bigVal, err := set.GetValidator(big.PublicKey)
	require.NoError(t, err)
	require.Equal(t, uint64(0), bigVal.VotingPower)
}

// TestHandleStalledCommitteeKeepsSoloOperator verifies the watchdog does not strand a committee: if
// demotion would leave no active members it is skipped (governance ResetCommittee is required instead).
func TestHandleStalledCommitteeKeepsSoloOperator(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	cons, err := sm.GetParamsCons()
	require.NoError(t, err)
	cons.RootChainId = sm.Config.ChainId
	require.NoError(t, sm.SetParamsCons(cons))
	nestedChainId := sm.Config.ChainId + 1
	// a single active operator plus provisional (never-signed) members
	solo := livenessTestValidator(t, 2, 10_000, nestedChainId)
	prov := livenessTestValidator(t, 0, 100, nestedChainId)
	for _, v := range []*Validator{solo, prov} {
		require.NoError(t, sm.SetValidator(v))
	}
	require.NoError(t, sm.SetCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(solo.Address)))
	require.NoError(t, sm.SetCommitteesData(&lib.CommitteesData{List: []*lib.CommitteeData{
		{ChainId: nestedChainId, LastRootHeightUpdated: 1, LastChainHeightUpdated: 1},
	}}))
	sm.height = 1 + CommitteeLivenessWindow + 5
	require.NoError(t, sm.HandleStalledCommittees())
	// the sole active operator is NOT demoted (that would leave zero online validators)
	act, err := sm.IsCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(solo.Address))
	require.NoError(t, err)
	require.True(t, act, "sole active operator must not be demoted")
}

// TestHandleStalledCommitteeIgnoresHealthy ensures a committee still within the liveness window is untouched.
func TestHandleStalledCommitteeIgnoresHealthy(t *testing.T) {
	sm := newTestStateMachine(t)
	enableCommitteeLiveness(t, &sm)
	cons, err := sm.GetParamsCons()
	require.NoError(t, err)
	cons.RootChainId = sm.Config.ChainId
	require.NoError(t, sm.SetParamsCons(cons))
	nestedChainId := sm.Config.ChainId + 1
	big := livenessTestValidator(t, 2, 10_000, nestedChainId)
	a := livenessTestValidator(t, 0, 100, nestedChainId)
	for _, v := range []*Validator{big, a} {
		require.NoError(t, sm.SetValidator(v))
		require.NoError(t, sm.SetCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(v.Address)))
	}
	// committee updated recently (within the window)
	sm.height = 100
	require.NoError(t, sm.SetCommitteesData(&lib.CommitteesData{List: []*lib.CommitteeData{
		{ChainId: nestedChainId, LastRootHeightUpdated: sm.height - 1, LastChainHeightUpdated: 1},
	}}))
	require.NoError(t, sm.HandleStalledCommittees())
	// nothing demoted
	bigActive, err := sm.IsCommitteeMemberActive(nestedChainId, crypto.NewAddressFromBytes(big.Address))
	require.NoError(t, err)
	require.True(t, bigActive)
}

// TestFilterProvisionalNonSigners ensures zero-power (provisional) members are excluded from non-sign
// penalty accounting.
func TestFilterProvisionalNonSigners(t *testing.T) {
	sm := newTestStateMachine(t)
	kg0, kg1 := newTestKeyGroup(t, 0), newTestKeyGroup(t, 1)
	vs, err := lib.NewValidatorSet(&lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{
		{PublicKey: kg0.PublicKey.Bytes(), VotingPower: 100}, // active
		{PublicKey: kg1.PublicKey.Bytes(), VotingPower: 0},   // provisional
	}})
	require.NoError(t, err)
	// both are non-signers; only the active one should survive the filter
	nonSigners := [][]byte{kg0.PublicKey.Bytes(), kg1.PublicKey.Bytes()}
	filtered := sm.filterProvisionalNonSigners(&vs, nonSigners)
	require.Len(t, filtered, 1)
	require.Equal(t, kg0.PublicKey.Bytes(), filtered[0])
}
