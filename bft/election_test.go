package bft

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestIsCandidateDistribution(t *testing.T) {
	const trials = 100000
	tolerance := 0.01 // Allow ± 1% deviation

	tests := []struct {
		name               string
		votingPower        uint64
		totalVotingPower   uint64
		expectedCandidates uint64
	}{
		{"Low Stake", 10, 1000, 100},
		{"Medium Stake", 500, 1000, 100},
		{"High Stake", 900, 1000, 100},
		{"Single Candidate", 500, 1000, 1},
		{"All Candidates", 500, 500, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedProb := float64(tt.votingPower*tt.expectedCandidates) / float64(tt.totalVotingPower)
			if expectedProb > 1 {
				expectedProb = 1 // Since probabilities can't exceed 1
			}
			selectedCount := 0
			for i := 0; i < trials; i++ {
				// generate random 256 bit (32 bytes)
				n, _ := crand.Int(crand.Reader, new(big.Int).Lsh(big.NewInt(1), 256))
				// use VRF Out
				vrfOut := n.Bytes()
				if IsCandidate(tt.votingPower, tt.totalVotingPower, tt.expectedCandidates, vrfOut) {
					selectedCount++
				}
			}
			observedProb := float64(selectedCount) / float64(trials)
			require.False(t, observedProb < expectedProb*(1-tolerance) || observedProb > expectedProb*(1+tolerance))
		})
	}
}

func TestSortitionAndVerifyCandidate(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		totalVals   uint64
		isCandidate bool
	}{
		{
			name:        "isCandidate",
			detail:      "deterministic key set ensures sortition results in a candidate in a set of 6 validators",
			totalVals:   6,
			isCandidate: true,
		},
		{
			name:        "isNotCandidate",
			detail:      "deterministic key set ensures sortition results in not a candidate in a set of 5 validators",
			totalVals:   5,
			isCandidate: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Election, int(test.totalVals))
			privateKey := c.valKeys[0]
			sortitionData := newTestSortitionData(t, c)
			out, vrf, isCandidate := Sortition(&SortitionParams{
				SortitionData: sortitionData,
				PrivateKey:    privateKey,
			})
			require.Equal(t, VRF(sortitionData.LastProposerAddresses, sortitionData.RootHeight, sortitionData.Height, sortitionData.Round, privateKey), vrf)
			require.Equal(t, crypto.Hash(vrf.Signature), out)
			require.Equal(t, test.isCandidate, isCandidate)
			outVerify, isCandidateFromVerify := VerifyCandidate(&SortitionVerifyParams{
				SortitionData: sortitionData,
				Signature:     vrf.Signature,
				PublicKey:     c.valKeys[0].PublicKey(),
			})
			require.Equal(t, out, outVerify)
			require.Equal(t, test.isCandidate, isCandidateFromVerify)
		})
	}
}

func TestWhenIsCandidate(t *testing.T) {
	for i := 1; i < 8; i++ {
		c := newTestConsensus(t, Election, i)
		for j := 0; j < len(c.valKeys); j++ {
			privateKey := c.valKeys[j]
			sortitionData := newTestSortitionData(t, c)
			_, _, isCandidate := Sortition(&SortitionParams{
				SortitionData: sortitionData,
				PrivateKey:    privateKey,
			})
			if isCandidate {
				fmt.Printf("%d,", j)
			}
		}
		fmt.Println()
	}
}

func TestSortitionValidity(t *testing.T) {
	privateKey, _ := crypto.NewBLS12381PrivateKey()
	lastNProposers := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	power, totalPower := 1000000, 3000000
	expectedAvg := float64(power) / float64(totalPower)
	totalIterations := 1000
	errorThreshold := .07
	isCandCount := uint64(0)
	for i := 0; i < totalIterations; i++ {
		if isCand := vrfAndCDF(SortitionParams{
			SortitionData: &lib.SortitionData{
				LastProposerAddresses: lastNProposers,
				Height:                uint64(rand.Intn(math.MaxUint32)),
				VotingPower:           uint64(power),
				TotalPower:            uint64(totalPower),
			},
			PrivateKey: privateKey,
		}); isCand {
			isCandCount++
		}
	}
	e := math.Abs(float64(isCandCount)/float64(totalIterations) - expectedAvg)
	require.True(t, e < errorThreshold)
}

func TestSelectProposerFromCandidates(t *testing.T) {
	tests := []struct {
		name                string
		detail              string
		totalVals           uint64
		totalCandidates     uint64
		expectedProposerIdx int
	}{
		{
			name:                "no candidates, weighted pseudorandom",
			detail:              "deterministic key set ensures that the weighted leader id is deterministic",
			totalVals:           3,
			totalCandidates:     0,
			expectedProposerIdx: 2,
		},
		{
			name:                "3 candidates, lowest index (0) is the proposer",
			detail:              "since out is set to index and the lowest out is the proposer, candidate 0 should be the proposer",
			totalVals:           3,
			totalCandidates:     3,
			expectedProposerIdx: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestConsensus(t, Election, int(test.totalVals))
			var vrfCandidates []VRFCandidate
			for i := uint64(0); i < test.totalCandidates; i++ {
				out := make([]byte, 8)
				binary.BigEndian.PutUint64(out, i)
				vrfCandidates = append(vrfCandidates, VRFCandidate{
					PublicKey: c.valKeys[i].PublicKey(),
					Out:       out,
				})
			}
			expectedProposerPubKey := c.valKeys[test.expectedProposerIdx].PublicKey().Bytes()
			require.Equal(t, expectedProposerPubKey, SelectProposerFromCandidates(vrfCandidates, newTestSortitionData(t, c), c.valSet.ValidatorSet))
		})
	}
}

// TestSelectProposerWeightedRoundRobin() verifies deterministic stake-weighted proposer selection.
func TestSelectProposerWeightedRoundRobin(t *testing.T) {
	validators := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{
		{PublicKey: []byte("validator-a"), VotingPower: 1},
		{PublicKey: []byte("validator-b"), VotingPower: 2},
		{PublicKey: []byte("validator-c"), VotingPower: 3},
	}}
	const totalPower = uint64(6)
	const samples = uint64(60_000)
	const tolerance = float64(samples) / 100
	view := &lib.View{NetworkId: 1, ChainId: 1, Height: 1}

	// The immutable 64-bit schedule converges on each validator's stake share.
	heightCounts := make(map[string]int)
	for height := uint64(1); height <= samples; height++ {
		view.Height, view.Round = height, 0
		proposer := SelectProposerWeightedRoundRobin(view, validators, totalPower)
		heightCounts[string(proposer)]++
	}
	require.InDelta(t, float64(samples/6), heightCounts["validator-a"], tolerance)
	require.InDelta(t, float64(samples/3), heightCounts["validator-b"], tolerance)
	require.InDelta(t, float64(samples/2), heightCounts["validator-c"], tolerance)

	// Recovery rounds use an independent immutable stride with the same weighting.
	roundCounts := make(map[string]int)
	view.Height = 10
	for round := uint64(0); round < samples; round++ {
		view.Round = round
		proposer := SelectProposerWeightedRoundRobin(view, validators, totalPower)
		roundCounts[string(proposer)]++
	}
	require.InDelta(t, float64(samples/6), roundCounts["validator-a"], tolerance)
	require.InDelta(t, float64(samples/3), roundCounts["validator-b"], tolerance)
	require.InDelta(t, float64(samples/2), roundCounts["validator-c"], tolerance)

	view.Height, view.Round = 42, 3
	first := SelectProposerWeightedRoundRobin(view, validators, totalPower)
	second := SelectProposerWeightedRoundRobin(view, validators, totalPower)
	require.Equal(t, first, second)
	require.Nil(t, SelectProposerWeightedRoundRobin(view, validators, totalPower+1))
	require.Nil(t, SelectProposerWeightedRoundRobin(nil, validators, totalPower))
}

// TestSelectProposerWeightedRoundRobinStakeEditDoesNotRekeySchedule() verifies stake edits do not re-key the proposer schedule.
func TestSelectProposerWeightedRoundRobinStakeEditDoesNotRekeySchedule(t *testing.T) {
	view := &lib.View{NetworkId: 1, ChainId: 1}
	for height := uint64(1); height <= 1_000; height++ {
		view.Height, view.Round = height, 0
		before := weightedRoundRobinPowerIndex(view, 2_000_000_000)
		after := weightedRoundRobinPowerIndex(view, 2_000_000_001)
		require.GreaterOrEqual(t, after, before)
		require.LessOrEqual(t, after-before, uint64(1))
	}
}

// TestSelectProposerWeightedRoundRobinIgnoresValidatorInputOrder() verifies validator input order does not affect selection.
func TestSelectProposerWeightedRoundRobinIgnoresValidatorInputOrder(t *testing.T) {
	firstOrder := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{
		{PublicKey: []byte("validator-c"), VotingPower: 3, NetAddress: "tcp://one.example"},
		{PublicKey: []byte("validator-a"), VotingPower: 1, NetAddress: "tcp://one.example"},
		{PublicKey: []byte("validator-b"), VotingPower: 2, NetAddress: "tcp://one.example"},
	}}
	secondOrder := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{
		{PublicKey: []byte("validator-b"), VotingPower: 2, NetAddress: "tcp://two.example"},
		{PublicKey: []byte("validator-c"), VotingPower: 3, NetAddress: "tcp://two.example"},
		{PublicKey: []byte("validator-a"), VotingPower: 1, NetAddress: "tcp://two.example"},
	}}
	view := &lib.View{NetworkId: 1, ChainId: 1}
	for height := uint64(1); height <= 12; height++ {
		for round := uint64(0); round < 6; round++ {
			view.Height, view.Round = height, round
			require.Equal(t,
				SelectProposerWeightedRoundRobin(view, firstOrder, 6),
				SelectProposerWeightedRoundRobin(view, secondOrder, 6),
			)
		}
	}
}

// TestSelectProposerWeightedRoundRobinDoesNotPinSubThirdValidator() verifies adjacent tickets do not pin a sub-third validator.
func TestSelectProposerWeightedRoundRobinDoesNotPinSubThirdValidator(t *testing.T) {
	validators := &lib.ConsensusValidators{ValidatorSet: []*lib.ConsensusValidator{
		{PublicKey: []byte("validator-a"), VotingPower: 1},
		{PublicKey: []byte("validator-b"), VotingPower: 1},
		{PublicKey: []byte("validator-c"), VotingPower: 1},
		{PublicKey: []byte("validator-d"), VotingPower: 1},
	}}
	// These chain IDs produced runs of 13,323 rounds and 3,452 heights with hash-derived odd strides; a quarter-power validator must now rotate on every adjacent ticket along either schedule dimension.
	for _, chainID := range []uint64{6_212, 13_210} {
		view := &lib.View{NetworkId: 1, ChainId: chainID, Height: 1}
		var previous []byte
		for round := uint64(0); round < 20_000; round++ {
			view.Round = round
			proposer := SelectProposerWeightedRoundRobin(view, validators, 4)
			if previous != nil {
				require.NotEqual(t, previous, proposer, "chain %d pinned proposer at round %d", chainID, round)
			}
			previous = proposer
		}

		view.Round = 0
		previous = nil
		for height := uint64(1); height <= 20_000; height++ {
			view.Height = height
			proposer := SelectProposerWeightedRoundRobin(view, validators, 4)
			if previous != nil {
				require.NotEqual(t, previous, proposer, "chain %d pinned proposer at height %d", chainID, height)
			}
			previous = proposer
		}
	}
}

func newTestSortitionData(t *testing.T, c *testConsensus) *lib.SortitionData {
	var lastNProposers [][]byte
	for _, k := range c.valKeys {
		lastNProposers = append(lastNProposers, k.PublicKey().Address().Bytes())
	}
	val, err := c.valSet.GetValidator(c.valKeys[0].PublicKey().Bytes())
	require.NoError(t, err)
	sortitionData := &lib.SortitionData{
		LastProposerAddresses: lastNProposers,
		RootHeight:            1,
		Height:                1,
		Round:                 0,
		TotalValidators:       uint64(len(c.valKeys)),
		VotingPower:           val.VotingPower,
		TotalPower:            c.valSet.TotalPower,
	}
	return sortitionData
}

func vrfAndCDF(p SortitionParams) bool {
	vrf := VRF(p.LastProposerAddresses, p.RootHeight, p.Height, p.Round, p.PrivateKey)
	return IsCandidate(p.VotingPower, p.TotalPower, 1, crypto.Hash(vrf.Signature))
}
