package bft

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
)

// testConsensus is a mocked structure used by the testing suite
type testConsensus struct {
	// the testable BFT object
	bft *BFT
	// below is data used to feed into the BFT object
	valKeys []crypto.PrivateKeyI // the private keys of the deterministic validators
	valSet  ValSet               // the validator set made of the deterministic private keys
	cont    *testController      // the mocked controller
}

// newTestConsensus() creates a Consensus test suite containing a testable BFT object using a mocked controller and a deterministic validator set
func newTestConsensus(t *testing.T, phase Phase, numValidators int) (tc *testConsensus) {
	// initialize variables
	tc, proposers, err := new(testConsensus), lib.Proposers{}, error(nil)
	// create a validator set for testing
	tc.valSet, tc.valKeys, proposers = newTestValSet(t, numValidators)
	// create the test controller
	tc.cont = &testController{
		Mutex:              sync.Mutex{},
		proposers:          &proposers,
		valSet:             map[uint64]ValSet{lib.CanopyChainId: tc.valSet},
		gossipCertChan:     make(chan *lib.QuorumCertificate),
		sendToProposerChan: make(chan lib.Signable),
		sendToReplicasChan: make(chan lib.Signable),
	}
	// Disable VDF service for testing
	config := lib.DefaultConfig()
	config.RunVDF = false
	// create the bft object using the mocks
	tc.bft, err = New(config, tc.valKeys[0], 1, 1, tc.cont, config.RunVDF, lib.NewDefaultLogger())
	tc.bft.ValidatorSet = tc.valSet
	require.NoError(t, err)
	// set the bft phase
	tc.bft.Phase = phase
	return
}

// newTestValSet() creates a validator set with matching private keys from a 'hard coded / deterministic' list of private keys
func newTestValSet(t *testing.T, numValidators int) (valSet ValSet, valKeys []crypto.PrivateKeyI, mockedPrevProposers lib.Proposers) {
	const TestValidatorVotingPower = 1000000
	var (
		err                 error
		consensusValidators lib.ConsensusValidators
		// DETERMINISTIC / HARDCODED KEYS USED FOR TESTING - MODIFICATION MAY BREAK TESTS
		keys = []string{
			"00453a101301cd7019b78ffa1186842dd93923e563b8ae22e2ab33ae889b23ee",
			"1b6b244fbdf614acb5f0d00a2b56ffcbe2aa23dabd66365dffcd3f06491ae50a",
			"2ee868f74134032eacba191ca529115c64aa849ac121b75ca79b37420a623036",
			"3e3ab94c10159d63a12cb26aca4b0e76070a987d49dd10fc5f526031e05801da",
			"479839d3edbd0eefa60111db569ded6a1a642cc84781600f0594bd8d4a429319",
			"51eb5eb6eca0b47c8383652a6043aadc66ddbcbe240474d152f4d9a7439eae42",
			"637cb8e916bba4c1773ed34d89ebc4cb86e85c145aea5653a58de930590a2aa4",
			"7235e5757e6f52e6ae4f9e20726d9c514281e58e839e33a7f667167c524ff658"}
	)
	// for each deterministic private key
	for i := 0; i < numValidators; i++ {
		// initialize voting power object
		votingPower := TestValidatorVotingPower
		// slightly weight the first validator so 2/3 validators can pass the +2/3 maj
		if i == 0 {
			votingPower += 2
		}
		// convert the string private key into a private key object
		privateKey, e := crypto.StringToBLS12381PrivateKey(keys[i])
		require.NoError(t, e)
		// add the private key to the list of keys
		valKeys = append(valKeys, privateKey)
		// add the address to the list of 'mocked previous proposers' for sortition seed data
		mockedPrevProposers.Addresses = append(mockedPrevProposers.Addresses, privateKey.PublicKey().Address().Bytes())
		// create the consensus validator object and add it to the validator set
		consensusValidators.ValidatorSet = append(consensusValidators.ValidatorSet, &lib.ConsensusValidator{
			PublicKey:   privateKey.PublicKey().Bytes(),
			VotingPower: uint64(votingPower),
		})
	}
	// create a validator set out of the validator objects
	valSet, err = lib.NewValidatorSet(&consensusValidators)
	require.NoError(t, err)
	return
}

// simElectionPhase() simulates the election phase of the BFT lifecycle
func (tc *testConsensus) simElectionPhase(t *testing.T) {
	// set the sortition data for the election
	tc.setSortitionData(t)
	// for each validator, act like a Leader Candidate
	// NOTE: in real execution, the Validator checks if
	// it's a Candidate before sending messages out
	for _, k := range tc.valKeys {
		// signs and sends a message with their VRF
		msg := &Message{
			Header: tc.view(Election),
			Vrf:    VRF(tc.cont.proposers.Addresses, 0, 1, 0, k),
		}
		require.NoError(t, msg.Sign(k))
		require.NoError(t, tc.bft.HandleMessage(msg))
	}
	return
}

// simElectionVotePhase() simulates the ELECTION-VOTE phase of the BFT lifecycle
func (tc *testConsensus) simElectionVotePhase(t *testing.T, propIdx int, BE, liveHQC, safeHQC bool, round uint64) (mk crypto.MultiPublicKeyI) {
	// simulate a vote in the ElectionVote phase
	mk, _, _ = tc.simVote(t, round, ElectionVote, func(idx int, m *Message) bool {
		// in ElectionVote, there's no proposal
		m.Qc.Block, m.Qc.Results = nil, nil
		// instead set the proposer key at the specified index
		m.Qc.ProposerKey = tc.valKeys[propIdx].PublicKey().Bytes()
		// test the aggregation of messages by only using the replica at index 1 to report BE and HQC
		// in the real world, it's unlikely that only one replica would have BE and HQC
		if idx == 1 {
			// if 'has byzantine evidence' then fill the DoubleSigners for the message
			if BE {
				m.LastDoubleSignEvidence = tc.newTestDoubleSignEvidence(t)
			}
			// if has 'highQc, fill the highQC appropriately
			if liveHQC || safeHQC {
				highQC := tc.setupTestableHighQC(t, liveHQC, true)
				m.Qc.Block, m.Qc.Results, m.HighQc = highQC.Block, highQC.Results, highQC
			}
		}
		return true
	})
	return
}

// simProposePhase() simulates the PROPOSE phase of the BFT lifecycle
func (tc *testConsensus) simProposePhase(t *testing.T, propIdx int, validProp bool, be ByzantineEvidence, hQC *QC, round uint64) (block []byte, results *lib.CertificateResult) {
	// generate a justification for the ProposePhase
	justifyPropose := tc.simElectionVotePhase(t, propIdx, false, false, false, round)
	// simulate the PROPOSE phase with a callback
	block, results = tc.simLead(t, justifyPropose, round, Propose, func(m *Message) {
		// set the hQC, BE, and proposer key
		m.HighQc, m.LastDoubleSignEvidence, m.Qc.ProposerKey = hQC, be.DSE.Evidence, tc.valKeys[propIdx].PublicKey().Bytes()
		// if proposal not valid, generate an invalid proposal and hash pair
		if !validProp {
			m.Qc.Block, m.Qc.BlockHash = tc.cont.NewTestBlock2(), tc.cont.NewTestBlockHash2()
		}
	}, propIdx)
	return
}

// simProposeVotePhase() simulates the PROPOSE-VOTE phase of the BFT lifecycle
func (tc *testConsensus) simProposeVotePhase(t *testing.T, isProp, maj23 bool, round uint64) (mk crypto.MultiPublicKeyI, blkHash []byte, resHash []byte) {
	// if self-identified as the Leader during the Election-Vote phase
	if isProp {
		tc.bft.ProposerKey = tc.bft.PublicKey
	}
	// simulate the voting for PROPOSE-VOTE
	return tc.simVote(t, round, ProposeVote, func(idx int, m *Message) bool {
		return maj23 || idx != 0
	})
}

// simPrecommitPhase() simulates the PRECOMMIT phase of the BFT lifecycle
func (tc *testConsensus) simPrecommitPhase(t *testing.T, round uint64) (block []byte, results *lib.CertificateResult) {
	// generate a justification for leading the PRECOMMIT phase
	justifyPrecommit, _, _ := tc.simProposeVotePhase(t, true, true, round)
	// simulate leading the PRECOMMIT phase
	return tc.simLead(t, justifyPrecommit, round, Precommit, func(m *Message) {})
}

// simPrecommitVote() simulates the PRECOMMIT-VOTE phase of the BFT lifecycle
func (tc *testConsensus) simPrecommitVotePhase(t *testing.T, proposerIdx int, round ...uint64) (crypto.MultiPublicKeyI, []byte, []byte) {
	// preset the proposer key as it would've been set in an earlier phase
	tc.bft.ProposerKey = tc.valKeys[proposerIdx].PublicKey().Bytes()
	// allow a custom round
	r := uint64(0)
	if len(round) == 1 {
		r = round[0]
	}
	// simulate the voting
	return tc.simVote(t, r, PrecommitVote, func(idx int, m *Message) bool {
		tc.bft.Block, _, tc.bft.Results, _ = tc.proposal(t)
		return true
	})
}

// simCommitPhase() simulates the COMMIT phase of the BFT lifecycle
func (tc *testConsensus) simCommitPhase(t *testing.T, proposerIdx int, round uint64) (multiKey crypto.MultiPublicKeyI, block []byte, results *lib.CertificateResult) {
	// generate a justification for leading the commit phase
	justifyCommit, _, _ := tc.simPrecommitVotePhase(t, proposerIdx, round)
	// simulate the phase
	tc.bft.Block, tc.bft.Results = tc.simLead(t, justifyCommit, round, Commit, func(m *Message) {}, proposerIdx)
	// return the results
	return justifyCommit, tc.bft.Block, tc.bft.Results
}

// simPacemakerPhase() simulates the PACEMAKER phase of the BFT lifecycle
func (tc *testConsensus) simPacemakerPhase(t *testing.T) {
	for i := 1; i < len(tc.valKeys); i++ {
		pacemakerMsg := &Message{Qc: &lib.QuorumCertificate{Header: tc.view(RoundInterrupt, uint64(i+2))}}
		require.NoError(t, pacemakerMsg.Sign(tc.valKeys[i]))
		require.NoError(t, tc.bft.HandleMessage(pacemakerMsg))
	}
}

// setSortitionData, sets the 'mock' sortition data for testing the ELECTION phase
func (tc *testConsensus) setSortitionData(t *testing.T) {
	selfVal, err := tc.valSet.GetValidator(tc.valKeys[0].PublicKey().Bytes())
	require.NoError(t, err)
	tc.bft.SortitionData = &lib.SortitionData{
		LastProposerAddresses: tc.cont.proposers.Addresses,
		Height:                1,
		Round:                 0,
		TotalValidators:       tc.valSet.NumValidators,
		VotingPower:           selfVal.VotingPower,
		TotalPower:            tc.valSet.TotalPower,
	}
}

// simLead() generically simulates a Leader Phase, generating a Proposal, executing a custom callback, signing, and sending the Proposal
func (tc *testConsensus) simLead(t *testing.T, mk crypto.MultiPublicKeyI, round uint64, phase Phase, callback func(m *Message), propIdx ...int) (block []byte, results *lib.CertificateResult) {
	// generate the proposal
	block, blkHash, results, resHash := tc.proposal(t)
	// aggregate the signatures from the multikey
	as, err := mk.AggregateSignatures()
	require.NoError(t, err)
	// create the leader (proposal) message
	msg := &Message{
		Header: tc.view(phase, round),
		Qc: &QC{
			Header:      tc.view(phase-1, round),
			Results:     results,
			ResultsHash: resHash,
			Block:       block,
			BlockHash:   blkHash,
			Signature: &lib.AggregateSignature{
				Signature: as,
				Bitmap:    mk.Bitmap(),
			},
		},
	}
	// execute the callback for custom phase functionality
	callback(msg)
	// allow a custom proposer index for double sign simulation
	proposerIdx := 0
	if len(propIdx) == 1 {
		proposerIdx = propIdx[0]
	}
	// sign the message with the proposer key
	require.NoError(t, msg.Sign(tc.valKeys[proposerIdx]))
	// send the message
	require.NoError(t, tc.bft.HandleMessage(msg))
	return
}

// simVote() generically simulates a Replica Vote Phase, generating the vote message, executing a custom callback, signing and sending the Vote
func (tc *testConsensus) simVote(t *testing.T, round uint64, phase Phase, callback func(idx int, m *Message) bool) (mk crypto.MultiPublicKeyI, blkHash []byte, resHash []byte) {
	// generate the proposal
	tc.bft.Block, blkHash, tc.bft.Results, resHash = tc.proposal(t)
	mk = tc.valSet.MultiKey.Copy()
	for idx, privateKey := range tc.valKeys {
		// the last signer is skipped to make it just barely +2/3rds majority
		// this is purely to stress test under the 'harshest' majority conditions
		if idx == len(tc.valKeys)-1 {
			break
		}
		// create a consensus message
		msg := &Message{
			Qc: &lib.QuorumCertificate{
				Header:      tc.view(phase, round),
				BlockHash:   blkHash,
				ResultsHash: resHash,
			},
		}
		// execute callback on the message to allow custom phase functionality
		sign := callback(idx, msg)
		// allow the callback to dictate which indecies sign the message
		if !sign {
			continue
		}
		// sign it with the validator key
		require.NoError(t, msg.Sign(privateKey))
		// mark the validator as signed on the multi-public key
		require.NoError(t, mk.AddSigner(msg.Signature.Signature, idx))
		// route the message through the BFT module as if 'self' just received the message
		require.NoError(t, tc.bft.HandleMessage(msg))
	}
	return
}

// setupTestableHighQC() creates a testable (liveness, safety, or invalid) highQC and
// sets up the BFT to accept or reject it under various conditions
func (tc *testConsensus) setupTestableHighQC(t *testing.T, liveness, shouldUnlock bool) (highQc *QC) {
	// setup a test consensus instance to fabricate a highQC
	round, c := uint64(0), newTestConsensus(t, ProposeVote, 3)
	// if the highQC should use 'Liveness' to pass the SafeNodePredicate, then set the round to 1
	if liveness {
		round = 1
	}
	// create a justification for a highQC
	justifyHQC, _, _ := c.simProposeVotePhase(t, false, true, round)
	aggSig, e := justifyHQC.AggregateSignatures()
	require.NoError(t, e)
	// create the actual highQC
	blk, blkHash, results, resHash := tc.proposal(t)
	highQc = &QC{
		Header:      tc.view(ProposeVote, round),
		Block:       blk,
		Results:     results,
		BlockHash:   blkHash,
		ResultsHash: resHash,
		Signature: &lib.AggregateSignature{
			Signature: aggSig,
			Bitmap:    justifyHQC.Bitmap(),
		},
	}
	// lock the node on some arbitrary hQC to be able to test the 'Unlocking' situations (only for liveness)
	tc.bft.HighQC = &QC{
		Header:  tc.view(ProposeVote, 0),
		Results: tc.bft.Results,
		Block:   tc.bft.Block,
	}
	// setup 'Unlocking' situations
	if shouldUnlock {
		// there's only two paths to unlock - liveness or safety
		if liveness {
			// if liveness, the 'View' must be higher than the BFTs current 'Lock'
			highQc.Header.Round = 1
		} else {
			// if safety, the Proposal must be the same as the current 'Lock'
			tc.bft.HighQC = &QC{
				Header:      tc.view(ProposeVote, 0),
				Results:     results,
				ResultsHash: resHash,
				Block:       blk,
				BlockHash:   blkHash,
			}
			tc.bft.Block, tc.bft.Results = blk, results
		}
	}
	return
}

// newPartialQCDoubleSign() simulates a partial QC being sent to the replica
func (tc *testConsensus) newPartialQCDoubleSign(t *testing.T, phase Phase) {
	// create an equivocating QC
	qc := &lib.QuorumCertificate{
		Header:      tc.view(phase-1, 1),
		BlockHash:   crypto.Hash([]byte("some proposal")),
		ResultsHash: crypto.Hash([]byte("some results")),
	}
	// create the bytes to be signed by the 'double signers'
	sb := qc.SignBytes()
	// create a multikey to hold the partial justification
	partialJustify := tc.valSet.MultiKey.Copy()
	// 2/3 sign the proposal, just missing the +2/3 majority that is needed
	for i, pk := range tc.valKeys {
		if i == 0 {
			continue
		}
		// sign it and add it to the justification
		_, idx, e := tc.valSet.GetValidatorAndIdx(pk.PublicKey().Bytes())
		require.NoError(t, e)
		require.NoError(t, partialJustify.AddSigner(pk.Sign(sb), idx))
	}
	// aggregate the partial justification signature
	aggSig, e := partialJustify.AggregateSignatures()
	require.NoError(t, e)
	// finalize the partial QC
	qc.Signature = &lib.AggregateSignature{
		Signature: aggSig,
		Bitmap:    partialJustify.Bitmap(),
	}
	// wrap it in a message and send it to the test replica
	msg := &Message{
		Header: tc.view(phase, 1),
		Qc:     qc,
	}
	// ironically, use the test replica as the 'organizer' who omitted their signature
	// this is a fun use-case because it shows how malicious proposers may not be trusted
	require.NoError(t, msg.Sign(tc.valKeys[0]))
	// send it to the test replica
	require.NoError(t, tc.bft.HandleMessage(msg))
}

// newElectionVoteDoubleSign() simulates an election candidate receiving a conflicting Votes with the real proposers' election vote QC
func (tc *testConsensus) newElectionVoteDoubleSign(t *testing.T) {
	// create the equivocating Vote
	msg := &Message{
		Qc: &lib.QuorumCertificate{
			Header:      tc.view(ElectionVote, 1),
			ProposerKey: tc.valKeys[0].PublicKey().Bytes(),
		},
	}
	// have replicas 1 and 2 sign and send it to the honest Candidate
	// 2/3 is just shy of +2/3 needed to justify the Candidate as a leader
	for i, pk := range tc.valKeys {
		if i != 0 {
			require.NoError(t, msg.Sign(pk))
			require.NoError(t, tc.bft.HandleMessage(msg))
		}
	}
}

// newTestDoubleSignEvidence() fabricates double sign evidence for the testing suite
func (tc *testConsensus) newTestDoubleSignEvidence(t *testing.T) []*DoubleSignEvidence {
	// create two equivocating Quorum Certificates with the same View
	qcA := &lib.QuorumCertificate{
		Header:      tc.view(0),
		BlockHash:   crypto.Hash([]byte("some proposal")),
		ResultsHash: crypto.Hash([]byte("some results")),
	}
	qcB := &lib.QuorumCertificate{
		Header:      tc.view(0),
		BlockHash:   crypto.Hash([]byte("some other proposal")),
		ResultsHash: crypto.Hash([]byte("some other results")),
	}
	// generate the sign bytes of both
	sbA, sbB := qcA.SignBytes(), qcB.SignBytes()
	// generate the justifications for both
	fullJustify, partialJustify := tc.valSet.MultiKey.Copy(), tc.valSet.MultiKey.Copy()
	// full justify has all signers
	for _, pk := range tc.valKeys {
		_, idx, e := tc.valSet.GetValidatorAndIdx(pk.PublicKey().Bytes())
		require.NoError(t, e)
		require.NoError(t, fullJustify.AddSigner(pk.Sign(sbA), idx))
	}
	// partial justify only has 2/3 signers
	for i, pk := range tc.valKeys {
		if i != 0 {
			_, idx, e := tc.valSet.GetValidatorAndIdx(pk.PublicKey().Bytes())
			require.NoError(t, e)
			require.NoError(t, partialJustify.AddSigner(pk.Sign(sbB), idx))
		}
	}
	// finalize the full QC by adding the aggregated signature
	aggSig, e := fullJustify.AggregateSignatures()
	require.NoError(t, e)
	qcA.Signature = &lib.AggregateSignature{
		Signature: aggSig,
		Bitmap:    fullJustify.Bitmap(),
	}
	// finalize the partial QC by adding the aggregated signature
	aggSig, e = partialJustify.AggregateSignatures()
	require.NoError(t, e)
	qcB.Signature = &lib.AggregateSignature{
		Signature: aggSig,
		Bitmap:    partialJustify.Bitmap(),
	}
	// wrap in the Evidence structure
	return []*DoubleSignEvidence{{
		VoteA: qcA,
		VoteB: qcB,
	}}
}

// view() creates a standardized view for consensus module testing
// - required phase param to set the phase in the view
// - optional round param, as most test will simply use round 0
func (tc *testConsensus) view(phase Phase, round ...uint64) *lib.View {
	r := uint64(0)
	if len(round) == 1 {
		r = round[0]
	}
	return &lib.View{
		NetworkId:  lib.CanopyMainnetNetworkId,
		ChainId:    lib.CanopyChainId,
		Height:     1,
		Round:      r,
		RootHeight: 1,
		Phase:      phase,
	}
}

// proposal() generates a mock block and result objects using the mock controller and their corresponding hashes
func (tc *testConsensus) proposal(t *testing.T) (blk, blkHash []byte, results *lib.CertificateResult, resultsHash []byte) {
	blk, results, err := tc.cont.ProduceProposal(nil, nil)
	require.NoError(t, err)
	blkHash, resultsHash = tc.cont.NewTestBlockHash(), results.Hash()
	return
}

// Below is a Controller Mock to enable the testing of the BFT module

var _ Controller = &testController{}

type testController struct {
	sync.Mutex
	proposers          *lib.Proposers
	valSet             map[uint64]ValSet // height -> id -> valset
	gossipCertChan     chan *lib.QuorumCertificate
	sendToProposerChan chan lib.Signable
	sendToReplicasChan chan lib.Signable
}

func (t *testController) LoadRootChainId(height uint64) (rootChainId uint64) {
	return lib.CanopyChainId
}

func (t *testController) LoadIsOwnRoot() bool {
	return true
}

func (t *testController) SelfSendBlock(qc *lib.QuorumCertificate) {
	t.GossipBlock(qc, nil)
}

func (t *testController) ChainHeight() uint64 {
	return t.RootChainHeight()
}

func (t *testController) ProduceProposal(_ *ByzantineEvidence, _ *crypto.VDF) (block []byte, results *lib.CertificateResult, err lib.ErrorI) {
	block = t.NewTestBlock()
	results = &lib.CertificateResult{
		RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{
				Address: crypto.Hash([]byte("mock"))[:20],
				ChainId: lib.CanopyChainId,
				Percent: 100,
			}},
		},
	}
	return
}

func (t *testController) ValidateProposal(qc *lib.QuorumCertificate, _ *ByzantineEvidence) lib.ErrorI {
	if len(qc.Block) == expectedCandidateLen {
		return nil
	}
	return ErrEmptyMessage()
}

func (t *testController) LoadCommittee(rootChainId, rootHeight uint64) (lib.ValidatorSet, lib.ErrorI) {
	return t.valSet[rootHeight], nil
}
func (t *testController) LoadCertificate(_ uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	return nil, nil
}

func (t *testController) SendCertificateResultsTx(certificate *lib.QuorumCertificate) {
	t.gossipCertChan <- certificate
}

func (t *testController) SendToReplicas(_ lib.ValidatorSet, msg lib.Signable) {
	t.sendToReplicasChan <- msg
}

func (t *testController) SendToProposer(msg lib.Signable) {
	t.sendToProposerChan <- msg
}

func (t *testController) LoadMinimumEvidenceHeight() (uint64, lib.ErrorI) { return 0, nil }
func (t *testController) IsValidDoubleSigner(_ uint64, _ []byte) bool     { return true }
func (t *testController) Syncing() *atomic.Bool                           { return &atomic.Bool{} }
func (t *testController) LoadCommitteeData() (*lib.CommitteeData, lib.ErrorI) {
	return &lib.CommitteeData{}, nil
}
func (t *testController) RootChainHeight() uint64 { return 0 }
func (t *testController) LoadLastProposers(_ uint64) (*lib.Proposers, lib.ErrorI) {
	return t.proposers, nil
}
func (t *testController) GossipBlock(cert *lib.QuorumCertificate, _ []byte) {
	t.gossipCertChan <- cert
}

func (t *testController) NewTestBlock() []byte {
	blockBytes, err := lib.Marshal(t.newTestBlock())
	if err != nil {
		panic(err)
	}
	return blockBytes
}

func (t *testController) NewTestBlock2() []byte {
	blockBytes, err := lib.Marshal(t.newTestBlock2())
	if err != nil {
		panic(err)
	}
	return blockBytes
}

func (t *testController) NewTestBlockHash() []byte {
	blk := t.newTestBlock()
	hash, _ := blk.BlockHeader.SetHash()
	return hash
}

func (t *testController) NewTestBlockHash2() []byte {
	blk := t.newTestBlock2()
	hash, _ := blk.BlockHeader.SetHash()
	return hash
}

func (t *testController) newTestBlock() *lib.Block {
	return &lib.Block{BlockHeader: &lib.BlockHeader{Height: 1}}
}

func (t *testController) newTestBlock2() *lib.Block {
	return &lib.Block{BlockHeader: &lib.BlockHeader{Height: 1, TransactionRoot: crypto.Hash([]byte("entropy"))}}
}

const expectedCandidateLen = 4
