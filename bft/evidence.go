package bft

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
)

// ByzantineEvidence represents a collection of evidence that supports byzantine behavior during the BFT lifecycle
// this Evidence is circulated to the Leader of a Round and is processed in the execution of Reward Transactions
type ByzantineEvidence struct {
	DSE DoubleSignEvidences // Evidence of `DoubleSigning`: Signing two different messages for the same View is against protocol rules (breaks protocol safety)
}

// ValidateByzantineEvidence() ensures the DoubleSigners in the Proposal are supported by the ByzantineEvidence
func (b *BFT) ValidateByzantineEvidence(slashRecipients *lib.SlashRecipients, be *ByzantineEvidence) lib.ErrorI {
	if slashRecipients == nil {
		return nil
	}
	if len(slashRecipients.DoubleSigners) != 0 {
		// locally generate a Double Signers list from the provided evidence
		doubleSigners, err := b.ProcessDSE(be.DSE.Evidence...)
		if err != nil {
			return err
		}
		// this validation ensures that the DoubleSigners is justified, but there may be additional evidence included without any error
		for _, ds := range slashRecipients.DoubleSigners {
			if ds == nil {
				return lib.ErrEmptyDoubleSigner()
			}
			// check if the Double Signer in the Proposal is within our locally generated Double Signers list
			if !slices.ContainsFunc(doubleSigners, func(signer *lib.DoubleSigner) bool {
				if signer == nil || !bytes.Equal(ds.Id, signer.Id) {
					return false
				}
				// validate each height slash per double signer is also justified
				for _, height := range ds.Heights {
					if !slices.Contains(signer.Heights, height) {
						return false
					}
				}
				return true
			}) {
				return lib.ErrMismatchEvidenceAndHeader()
			}
		}
	}
	return nil
}

// DOUBLE SIGN EVIDENCE

// NewDSE() creates a list of DoubleSignEvidences with a built-in DeDuplicator
func NewDSE(dse ...[]*DoubleSignEvidence) DoubleSignEvidences {
	ds := make([]*DoubleSignEvidence, 0)
	if dse != nil {
		ds = dse[0]
	}
	return DoubleSignEvidences{
		Evidence:     ds,
		DeDuplicator: make(map[string]bool),
	}
}

// ProcessDSE() validates each piece of double sign evidence and returns a list of double signers
func (b *BFT) ProcessDSE(dse ...*DoubleSignEvidence) (results []*lib.DoubleSigner, e lib.ErrorI) {
	results = make([]*lib.DoubleSigner, 0)
	for _, x := range dse {
		// sanity check the evidence
		if err := x.CheckBasic(); err != nil {
			return nil, err
		}
		// load the Validator set for this Committee at that height
		committeeHeight := x.VoteA.Header.RootHeight
		// load the root chain id for the committeeHeight - 1; this ensures we are never using the latest
		// height which would require logic to handle 'switch root' in the currently played block
		rootChainId := b.Controller.LoadRootChainId(committeeHeight - 1)
		// load the committee from the root chain id using the n-1 height because state machine heights are 'end state' once committed
		vs, err := b.LoadCommittee(rootChainId, committeeHeight)
		if err != nil {
			return nil, err
		}
		// ensure the evidence isn't expired
		minEvidenceHeight, err := b.LoadMinimumEvidenceHeight(rootChainId, committeeHeight)
		if err != nil {
			return nil, err
		}
		// validate the piece of evidence
		if err = x.Check(vs, b.View, *minEvidenceHeight); err != nil {
			return nil, err
		}
		// if the votes are identical - it's not a double sign...
		if bytes.Equal(x.VoteB.SignBytes(), x.VoteA.SignBytes()) {
			return nil, lib.ErrNonEquivocatingVote() // same payloads
		}
		// take the signatures from the two
		sig1, sig2 := x.VoteA.Signature, x.VoteB.Signature
		// extract the double signers between the two
		doubleSigners, err := sig1.GetDoubleSigners(sig2, vs)
		if err != nil {
			return nil, err
		}
		// the evidence may include double signers who were already slashed for that height
		// if so, ignore those double signers but still process the rest of the bad actors
	out:
		for _, pubKey := range doubleSigners {
			pk, er := crypto.NewPublicKeyFromBytes(pubKey)
			if er != nil {
				return nil, lib.ErrPubKeyFromBytes(er)
			}
			if b.IsValidDoubleSigner(rootChainId, committeeHeight, pk.Address().Bytes()) {
				b.log.Infof("DoubleSigner %s is valid", lib.BytesToTruncatedString(pubKey))
				// check to see if double signer included in the results already
				for i, doubleSigner := range results {
					if bytes.Equal(doubleSigner.Id, pubKey) {
						// simply update the height
						results[i].AddHeight(committeeHeight)
						continue out
					}
				}
				// add to the results
				results = append(results, &lib.DoubleSigner{
					Id:      pubKey,
					Heights: []uint64{committeeHeight},
				})
			} else {
				b.log.Warnf("DoubleSigner %s is not valid", lib.BytesToTruncatedString(pubKey))
			}
		}
	}
	return
}

// AddDSE() validates and adds new DoubleSign Evidence to a list of DoubleSignEvidences
func (b *BFT) AddDSE(e *DoubleSignEvidences, ev *DoubleSignEvidence) (err lib.ErrorI) {
	// basic sanity checks for the evidence
	if err = ev.CheckBasic(); err != nil {
		return
	}
	// nullify the block and results as they are unnecessary bloat in the message for this purpose
	ev.VoteA.Block, ev.VoteA.Results = nil, nil
	ev.VoteB.Block, ev.VoteB.Results = nil, nil
	// process the Double Sign Evidence and save the double signers
	badSigners, err := b.ProcessDSE(ev)
	if err != nil {
		return err
	}
	// ignore if there are no bad actors
	if len(badSigners) == 0 {
		return lib.ErrInvalidEvidence()
	}
	// de duplicate the evidence
	if e.DeDuplicator == nil {
		e.DeDuplicator = make(map[string]bool)
	}
	// NOTE: this de-duplication is only good for 'accidental' duplication
	// evidence could be replayed - but it wouldn't result in additional slashes
	// as each DS is indexed by canopy height
	bz, _ := lib.Marshal(ev)
	key1 := lib.BytesToString(bz)
	if _, isDuplicate := e.DeDuplicator[key1]; isDuplicate {
		return
	}
	b.log.Infof("Adding byzantine evidence: %s", lib.BytesToTruncatedString(bz))
	e.Evidence = append(e.Evidence, ev)
	e.DeDuplicator[key1] = true
	return
}

// GetLocalDSE() returns the double sign evidences collected by the local node
func (b *BFT) GetLocalDSE() DoubleSignEvidences {
	dse := NewDSE()
	// by partial QC: a byzantine Leader sent a 'non +2/3 quorum certificate'
	// and the node holds a correct Quorum Certificate for the same View
	b.addDSEByPartialQC(&dse)
	// by candidate: a good samaritan Leader Candidate node receives ELECTION votes from Replicas
	// that also voted for the 'true Leader' the candidate now holds equivocating signatures by Replicas
	// for the same View
	b.addDSEByCandidate(&dse)
	// log if DSE is found
	if dseLen := len(dse.Evidence); dseLen != 0 {
		b.log.Infof("GetLocalDSE yielded %d pieces of evidence", dseLen)
	}
	return dse
}

// CheckBasic() executes basic sanity checks on the DoubleSign Evidence
// It's important to note that DoubleSign evidence may be processed for any height
// thus it's never validated against 'current height'
func (x *DoubleSignEvidence) CheckBasic() lib.ErrorI {
	if x == nil {
		return lib.ErrEmptyEvidence()
	}
	if x.VoteA == nil || x.VoteB == nil || x.VoteA.Header == nil || x.VoteB.Header == nil {
		return lib.ErrEmptyQuorumCertificate()
	}
	if !x.VoteA.Header.Equals(x.VoteB.Header) {
		return lib.ErrMismatchEvidenceAndHeader()
	}
	return nil
}

// Check() validates the double sign evidence
func (x *DoubleSignEvidence) Check(vs lib.ValidatorSet, view *lib.View, minimumEvidenceHeight uint64) lib.ErrorI {
	// can't be too old
	if x.VoteA.Header.RootHeight < minimumEvidenceHeight {
		return lib.ErrEvidenceTooOld()
	}
	// ensure large payloads are empty as they are unnecessary for this message
	if x.VoteA.Block != nil || x.VoteB.Block != nil {
		return lib.ErrNonNilBlock()
	}
	if x.VoteA.Results != nil || x.VoteB.Results != nil {
		return lib.ErrNonNilCertResults()
	}
	// should be a valid QC for the committee
	// NOTE: CheckBasic() purposefully doesn't return errors on partial QCs
	if _, err := x.VoteA.Check(vs, 0, view, false); err != nil {
		return err
	}
	if _, err := x.VoteB.Check(vs, 0, view, false); err != nil {
		return err
	}
	// ensure it's the same height
	if !x.VoteA.Header.Equals(x.VoteB.Header) {
		return lib.ErrInvalidEvidenceHeights() // different heights
	}
	if bytes.Equal(x.VoteB.SignBytes(), x.VoteA.SignBytes()) {
		return lib.ErrNonEquivocatingVote() // same payloads
	}
	return nil
}

/*
DoubleSignEvidence: By PartialQC

With < 1/3 Byzantine actors, two conflicting Quorum Certs with +2/3 majority cannot exist for the same view

Double signs can be detected when a leader sends a PartialQC (< +2/3 majority) to a replica, and that replica already
holds a +2/3 majority QC for the same view (from another leader or an earlier message)

Correct replicas save PartialQCs to check for double-sign evidence, which they share with the next proposer during the ElectionVote phase
*/
type PartialQCs map[string]*QC // [ PayloadHash ] -> Partial QC

// AddPartialQC() saves a non-majority Quorum Certificate which is a big hint of faulty behavior
func (b *BFT) AddPartialQC(m *Message) (err lib.ErrorI) {
	b.Controller.Lock()
	defer b.Controller.Unlock()
	bz, err := lib.Marshal(m.Qc)
	if err != nil {
		return
	}
	b.PartialQCs[lib.BytesToString(bz)] = m.Qc
	return
}

// addDSEByPartialQC() attempts to convert all partial QCs to DoubleSignEvidence by finding a saved conflicting valid QC
func (b *BFT) addDSEByPartialQC(dse *DoubleSignEvidences) {
	// REPLICA with two proposer messages for same (H,R,P) - the partial is the malicious one
	for _, pQC := range b.PartialQCs {
		evidenceHeight := pQC.Header.Height
		// check if evidence height is current (non-historical)
		if evidenceHeight == b.Height {
			// get the round of the partial QC to try to find a conflicting majority QC
			roundProposal := b.Proposals[pQC.Header.Round]
			if roundProposal == nil {
				continue
			}
			// try to find a conflicting QC
			// NOTE: proposals with conflicting QC is 1 phase above as it's used by the leader as justification
			proposal, found := roundProposal[phaseToString(pQC.Header.Phase+1)]
			if !found {
				continue
			}
			// add the double sign evidence to the list
			if err := b.AddDSE(dse, &DoubleSignEvidence{
				VoteA: proposal[0].Qc, // if both a partial and full exists
				VoteB: pQC,
			}); err != nil {
				b.log.Error(err.Error())
			}
			b.log.Infof("Added byzantine evidence by partial QC for phase %s", phaseToString(pQC.Header.Phase))
		} else { // this partial QC is historical
			// historically can only process precommit vote as the other non Commit QCs are pruned
			if pQC.Header.Phase != PrecommitVote {
				continue
			}
			// Load the certificate that contains the competing QC
			certificate, err := b.LoadCertificate(evidenceHeight)
			if err != nil {
				continue
			}
			// add the double sign evidence to the list
			if err = b.AddDSE(dse, &DoubleSignEvidence{
				VoteA: certificate, // if both a partial and full exists
				VoteB: pQC,
			}); err != nil {
				b.log.Error(err.Error())
			}
			b.log.Infof("Added byzantine evidence by historical partial QC at height %d ", evidenceHeight)
		}
	}
}

/*
DoubleSignEvidence: By LeaderCandidate

If a Leader Candidate receives ELECTION votes from Replicas that also voted for the true Leader, when the true Leader proves
their legitimacy with an ELECTION_VOTE_QC, the candidate ends up holding equivocating signatures (evidence) that some Replicas
voted for both Leaders in the same View

This allows the Leader Candidate to `out` the validators who `double signed` for themselves and the true leader for the same View.
*/

// addDSEByCandidate() node looks through any ELECTION-VOTES they received to see if any signers conflict with signers of the true Leader
func (b *BFT) addDSEByCandidate(dse *DoubleSignEvidences) {
	// ELECTION CANDIDATE exposing double sign election
	if !b.SelfIsProposer() {
		// for each Round until present
		for r := uint64(0); r <= b.Round; r++ {
			// get votes for round
			rvs := b.Votes[r]
			if rvs == nil {
				continue
			}
			// get votes for Election Vote phase
			ps := phaseToString(ElectionVote)
			electionVotes := rvs[ps]
			// if didn't receive any election votes
			if electionVotes == nil {
				continue
			}
			// get the true Leaders' messages for this same round
			roundProposal := b.Proposals[r]
			if roundProposal == nil {
				continue
			}
			// specifically the Propose message which contains the ElectionVote justification signatures
			proposal, found := roundProposal[phaseToString(Propose)]
			if !found || len(proposal) == 0 {
				continue
			}
			// for each election vote payload received (likely only 1 payload)
			for _, voteSet := range electionVotes {
				// if the vote exists and the QC is not empty
				if voteSet.Vote != nil && voteSet.Vote.Qc != nil {
					// aggregate the signers of this vote
					signature, e := voteSet.multiKey.AggregateSignatures()
					if e != nil {
						continue
					}
					// build into an ElectionQC
					qc := &QC{
						Header:      voteSet.Vote.Qc.Header,
						ProposerKey: voteSet.Vote.Qc.ProposerKey,
						Signature: &lib.AggregateSignature{
							Signature: signature,
							Bitmap:    voteSet.multiKey.Bitmap(),
						},
					}
					// attempt to add it as DoubleSignEvidence
					// (will be rejected if there are no conflicting signers)
					if err := b.AddDSE(dse, &DoubleSignEvidence{
						VoteA: proposal[0].Qc,
						VoteB: qc,
					}); err != nil {
						b.log.Warn(err.Error())
					}
				}
			}
		}
	}
}
