package bft

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
	"time"
)

// PROPOSALS RECEIVED FROM PROPOSER FOR CURRENT HEIGHT

// NOTE: A 'Proposal' is a message from the Leader (Proposer) asking for votes from Replicas.
// The Proposal is valid (justified) if it is backed by either:
//   - A Verifiable Random Function (VRF)
//   - A Quorum Certificate, votes from +2/3rds of the Replica for the previous Phase
// The Leader must gather these votes to organize consensus in the current Phase and lead the set through the stages of BFT.

// ProposalsForHeight are an in-memory list of messages received from the Leader
// NOTE: an array of Messages is required for the ELECTION phase as there can be multiple proposals if there is multiple Candidates
type ProposalsForHeight map[uint64]map[string][]*Message // [ROUND][PHASE] -> PROPOSAL(s)

// AddProposal() saves a validated proposal from the Leader in memory
func (b *BFT) AddProposal(m *Message) lib.ErrorI {
	b.Controller.Lock()
	defer b.Controller.Unlock()
	// load the list of Proposals for the round
	// initialize if not yet created
	roundProposal, found := b.Proposals[m.Header.Round]
	if !found {
		roundProposal = make(map[string][]*Message)
	}
	// define convenience variables
	phase := m.Header.Phase
	phaseProposal := roundProposal[phaseToString(phase)]
	// if it's an Election Proposal, add to the list (multiple candidates)
	// else overwrite the Proposal (this is ok because Proposer ID is previously authenticated after the ELECTION phase)
	if m.Header.Phase == Election {
		// check to see if the candidate is already on the list
		// this may happen during a NEW_COMMITTEE reset
		alreadyExists := slices.ContainsFunc(roundProposal[phaseToString(phase)], func(compare *Message) bool {
			if compare == nil || compare.Qc == nil || m == nil || m.Qc == nil {
				return false
			}
			return bytes.Equal(m.Qc.ProposerKey, compare.Qc.ProposerKey)
		})
		// if candidate not already on the list
		if !alreadyExists {
			// add to the list
			roundProposal[phaseToString(phase)] = append(phaseProposal, m)
		}
	} else {
		// overwrite the proposal
		roundProposal[phaseToString(phase)] = []*Message{m}
	}
	// add to the global list
	b.Proposals[m.Header.Round] = roundProposal
	return nil
}

// ProposalsResetForNewCommittee resets proposals when the root chain sends a 'NewCommittee' reset command
func (b *BFT) ProposalsResetForNewCommittee() {
	// initialize a new proposals map for the current height
	propsForHeight := make(ProposalsForHeight)
	propsForHeight[0] = make(map[string][]*Message)
	// preserve R0 election candidates to ensure messages sent before the reset are not lost
	propsForHeight[0][phaseToString(Election)] = b.Proposals[0][phaseToString(Election)]
	// replace existing proposals with the new structure
	b.Proposals = propsForHeight
}

// GetProposal() retrieves a proposal from the leader at the latest View
func (b *BFT) GetProposal() *Message { return b.getProposal(b.Round, b.Phase) }

// getProposal() retrieves a proposal from the leader at the Round.Phase
func (b *BFT) getProposal(round uint64, phase Phase) *Message {
	proposal, found := b.Proposals[round][phaseToString(phase-1)]
	if !found {
		return nil
	}
	return proposal[0]
}

// GetElectionCandidates() retrieves ELECTION messages, verifies, and returns the candidate(s)
func (b *BFT) GetElectionCandidates() (candidates []VRFCandidate) {
	defer lib.TimeTrack(b.log, time.Now())
	roundProposal := b.Proposals[b.Round]
	// for each Election proposal message
	// validate the VRF and verify is a candidate
	for _, m := range roundProposal[phaseToString(Election)] {
		// define convenience variable
		vrf := m.GetVrf()
		// get the validator from the set
		v, err := b.ValidatorSet.GetValidator(vrf.PublicKey)
		if err != nil {
			b.log.Errorf("an error occurred retrieving the Validator from the ValSet: %s", err.Error())
			continue
		}
		publicKey, _ := crypto.NewPublicKeyFromBytes(v.PublicKey)
		// validate the sortition and verify if is a candidate
		b.SortitionData.VotingPower = v.VotingPower
		out, isCandidate := VerifyCandidate(&SortitionVerifyParams{
			SortitionData: b.SortitionData,
			Signature:     vrf.Signature,
			PublicKey:     publicKey,
		})
		// add to candidates list
		if isCandidate {
			candidates = append(candidates, VRFCandidate{
				PublicKey: publicKey,
				Out:       out,
			})
		}
	}
	// return list of candidates
	return candidates
}
