package bft

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
)

// PROPOSALS RECEIVED FROM PROPOSER FOR CURRENT HEIGHT

// NOTE: A 'Proposal' is a message from the Leader (Proposer) asking for votes from Replicas.
// The Proposal is valid when backed by a Quorum Certificate with votes from +2/3rds of Replicas for the previous phase.
// The Leader must gather these votes to organize consensus in the current Phase and lead the set through the stages of BFT.

// ProposalsForHeight are an in-memory list of messages received from the Leader
type ProposalsForHeight map[uint64]map[string][]*Message // [ROUND][PHASE] -> PROPOSAL(s)

// AddProposal() saves a validated proposal from the Leader in memory
func (b *BFT) AddProposal(m *Message) lib.ErrorI {
	if m == nil {
		return ErrEmptyMessage()
	}
	if m.Header == nil {
		return lib.ErrEmptyView()
	}
	// Never retain proposals from the retired VRF election protocol, even if an internal caller bypasses HandleMessage.
	if b.usesDeterministicProposers() && m.Header.Phase == Election {
		return lib.ErrWrongPhase()
	}
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
	if m.Header.Phase == Election {
		phaseProposal := roundProposal[phaseToString(phase)]
		alreadyExists := slices.ContainsFunc(phaseProposal, func(compare *Message) bool {
			return compare != nil && compare.Qc != nil && m.Qc != nil && bytes.Equal(m.Qc.ProposerKey, compare.Qc.ProposerKey)
		})
		if !alreadyExists {
			roundProposal[phaseToString(phase)] = append(phaseProposal, m)
		}
	} else {
		// The proposer is already authenticated, so there is one proposal per phase.
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
	if !b.usesDeterministicProposers() {
		propsForHeight[0][phaseToString(Election)] = b.Proposals[0][phaseToString(Election)]
	}
	// replace existing proposals with the new structure
	b.Proposals = propsForHeight
}

// GetProposal() retrieves a proposal from the leader at the latest View
func (b *BFT) GetProposal() *Message { return b.getProposal(b.Round, b.Phase) }

// getProposalAtPhase() returns a proposal whose header exactly matches phase for event-driven advancement across a corresponding leader phase.
func (b *BFT) getProposalAtPhase(round uint64, phase Phase) *Message {
	proposals := b.Proposals[round][phaseToString(phase)]
	if len(proposals) == 0 {
		return nil
	}
	return proposals[0]
}

// getProposal() retrieves a proposal from the leader at the Round.Phase
func (b *BFT) getProposal(round uint64, phase Phase) *Message {
	proposal, found := b.Proposals[round][phaseToString(phase-1)]
	if !found || len(proposal) == 0 {
		return nil
	}
	return proposal[0]
}

// GetElectionCandidates() retrieves ELECTION messages, verifies, and returns the candidate(s)
func (b *BFT) GetElectionCandidates() (candidates []VRFCandidate) {
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
