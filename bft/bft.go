package bft

import (
	"bytes"
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"sort"
	"sync/atomic"
	"time"
)

// BFT is a structure that holds data for a Hotstuff BFT instance
type BFT struct {
	*lib.View                            // the current period during which the BFT is occurring (Height/Round/Phase)
	Votes         VotesForHeight         // 'votes' received from Replica (non-leader) Validators
	Proposals     ProposalsForHeight     // 'proposals' received from the Leader Validator(s)
	ProposerKey   []byte                 // the public key of the proposer
	ValidatorSet  ValSet                 // the current set of Validators
	HighQC        *QC                    // the highest PRECOMMIT quorum certificate the node is aware of for this Height
	Block         []byte                 // the current Block being voted on (the foundational unit of the blockchain)
	Results       *lib.CertificateResult // the current Result being voted on (reward and slash recipients)
	SortitionData *SortitionData         // the current data being used for VRF+CDF Leader Election
	VDFService    *lib.VDFService        // the verifiable delay service, run once per block as a deterrent against long-range-attacks
	HighVDF       *crypto.VDF            // the highest VDF among replicas - if the chain is using VDF for long-range-attack protection

	ByzantineEvidence *ByzantineEvidence // evidence of faulty or malicious Validators collected during the BFT process
	PartialQCs        PartialQCs         // potentially implicating evidence that may turn into ByzantineEvidence if paired with an equivocating QC
	PacemakerMessages PacemakerMessages  // View messages from the current ValidatorSet allowing the node to synchronize to the highest +2/3 seen Round

	Controller               // reference to the Controller for callbacks like producing and validating the proposal via the plugin or gossiping commit message
	resetBFT   chan ResetBFT // trigger that resets the BFT due to a new Target block or a new Canopy block
	syncing    *atomic.Bool  // if chain for this committee is currently catching up to latest height

	PhaseTimer      *time.Timer // ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
	OptimisticTimer *time.Timer // enables 'Optimistic Responsiveness Mode' starting from Round 10, allowing faster consensus while sacrificing voter participation

	PublicKey  []byte             // self consensus public key
	PrivateKey crypto.PrivateKeyI // self consensus private key
	Config     lib.Config         // self configuration
	log        lib.LoggerI        // logging
}

// New() creates a new instance of HotstuffBFT for a specific Committee
func New(c lib.Config, valKey crypto.PrivateKeyI, committeeID, canopyHeight, height uint64, vs ValSet,
	con Controller, vdfEnabled bool, l lib.LoggerI) (*BFT, lib.ErrorI) {
	// determine if using a Verifiable Delay Function for long-range-attack protection
	var vdf *lib.VDFService
	// calculate the targetTime from commitProcess and set the VDF
	if vdfEnabled {
		vdfTargetTime := time.Duration(float64(c.CommitProcessMS)*CommitProcessToVDFTargetCoefficient) * time.Millisecond
		vdf = lib.NewVDFService(vdfTargetTime, l)
	}
	return &BFT{
		View: &lib.View{
			Height:       height,
			CanopyHeight: canopyHeight,
			NetworkId:    c.NetworkID,
			CommitteeId:  committeeID,
		},
		Votes:        make(VotesForHeight),
		Proposals:    make(ProposalsForHeight),
		ValidatorSet: vs,
		ByzantineEvidence: &ByzantineEvidence{
			DSE: DoubleSignEvidences{},
		},
		PartialQCs:        make(PartialQCs),
		PacemakerMessages: make(PacemakerMessages),
		PublicKey:         valKey.PublicKey().Bytes(),
		PrivateKey:        valKey,
		Config:            c,
		log:               l,
		Controller:        con,
		resetBFT:          make(chan ResetBFT, 1),
		syncing:           con.Syncing(committeeID),
		PhaseTimer:        lib.NewTimer(),
		OptimisticTimer:   lib.NewTimer(),
		VDFService:        vdf,
		HighVDF:           new(crypto.VDF),
	}, nil
}

// Start() initiates the HotStuff BFT service.
// - Phase Timeout ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
// - Optimistic Timeout enables 'Optimistic Responsiveness Mode' starting from Round 10, allowing faster consensus
// This design balances synchronization speed during adverse conditions with maximizing voter participation under normal conditions
// - ResetBFT occurs upon receipt of a Quorum Certificate
//   - (a) Canopy committeeID <committeeSet changed, reset but keep locks to prevent conflicting validator sets between peers during a view change>
//   - (b) Target committeeID <mission accomplished, move to next height>
func (b *BFT) Start() {
	for {
		select {
		// PHASE TIMEOUT
		// - This triggers when the phase's sleep time has expired, indicating that all expected messages for this phase should have already been received
		case <-b.PhaseTimer.C:
			func() {
				b.Controller.Lock()
				defer b.Controller.Unlock()
				b.HandlePhase()
			}()

		// OPTIMISTIC TIMEOUT
		// - This triggers when Round 10 phase sleep has expired, this functionality only works well after Round 10
		// - Allows an intermittent 'Optimistic' check to see if node can move on before PhaseTimer actually triggers
		case <-b.OptimisticTimer.C:
			func() {
				b.Controller.Lock()
				defer b.Controller.Unlock()
				// if self doesn't have +2/3rds (or leader msg) already, reset the timer and sleep again
				if !b.PhaseHas23Maj() {
					lib.ResetTimer(b.OptimisticTimer, b.WaitTime(b.Phase, 10))
					return
				}
				// if self has +2/3 (or leader msg) already, move forward Optimistically
				b.HandlePhase()
			}()

		// RESET BFT
		// - This triggers when receiving a new Commit Block (QC) from either Canopy (a) or a sub-chain (b)
		case resetBFT := <-b.resetBFT:
			func() {
				b.Controller.Lock()
				defer b.Controller.Unlock()
				if resetBFT.UpdatedCanopyHeight != 0 { // (a) Canopy block reset
					if b.CommitteeId == lib.CanopyCommitteeId {
						return // ignore if Canopy Committee, as the Target reset notification is the correct reset path
					}
					b.log.Info("Resetting BFT timers after receiving a new Canopy block")
					// update the new committee
					b.CanopyHeight, b.ValidatorSet = resetBFT.UpdatedCanopyHeight, resetBFT.UpdatedCommitteeSet
					// reset back to round 0 but maintain locks to prevent 'fork attacks'
					b.NewHeight(true)
					// immediately reset and start the height over again with the new Validator set
					b.SetWaitTimers(0, b.WaitTime(CommitProcess, 10), 0)
				} else { // (b) Target block reset
					b.log.Info("Resetting BFT timers after receiving a new Target block (NEW_HEIGHT)")
					// reset BFT variables and start VDF
					b.NewHeight()
					// start BFT over after sleeping CommitProcessMS
					b.SetWaitTimers(b.WaitTime(CommitProcess, 0), b.WaitTime(CommitProcess, 10), resetBFT.ProcessTime)
				}
			}()
		}
	}
}

// HandlePhase() is the main BFT Phase stepping loop
func (b *BFT) HandlePhase() {
	stopTimers := func() { b.PhaseTimer.Stop(); b.OptimisticTimer.Stop() }
	// if currently catching up to latest height, pause the BFT loop
	if isSyncing := b.syncing.Load(); isSyncing {
		b.log.Info("Paused BFT loop as currently syncing")
		stopTimers()
		return
	}
	// if not a validator, wait until the next block to check if became a validator
	if !b.SelfIsValidator() {
		b.log.Info("Not currently a validator, waiting for a new block")
		stopTimers()
		return
	}
	// measure process time to have the most accurate timer timeouts
	startTime := time.Now()
	switch b.Phase {
	case Election:
		b.StartElectionPhase()
	case ElectionVote:
		b.StartElectionVotePhase()
	case Propose:
		b.StartProposePhase()
	case ProposeVote:
		b.StartProposeVotePhase()
	case Precommit:
		b.StartPrecommitPhase()
	case PrecommitVote:
		b.StartPrecommitVotePhase()
	case Commit:
		b.StartCommitPhase()
	case CommitProcess:
		b.StartCommitProcessPhase()
	case Pacemaker:
		b.Pacemaker()
	}
	// after each phase, set the timers for the next phase
	b.SetTimerForNextPhase(time.Since(startTime))
	return
}

// StartElectionPhase() begins the ElectionPhase after the CommitProcess (normal) or Pacemaker (previous Round failure) timeouts
// ELECTION PHASE:
// - Replicas run the Cumulative Distribution Function and a 'practical' Verifiable Random Function
// - If they are a candidate they send the VRF Out to the replicas
func (b *BFT) StartElectionPhase() {
	b.log.Infof(b.View.ToString())
	// retrieve Validator object from the ValidatorSet
	selfValidator, err := b.ValidatorSet.GetValidator(b.PublicKey)
	if err != nil {
		b.log.Error(err.Error())
		return
	}
	// initialize the sortition parameters
	b.SortitionData = &SortitionData{
		LastProposerAddresses: b.LoadLastProposers().Addresses, // LastProposers ensures defense against Grinding Attacks
		Height:                b.Height,                        // height ensures a unique sortition seed for each height
		Round:                 b.Round,                         // round ensures a unique sortition seed for each round
		TotalValidators:       b.ValidatorSet.NumValidators,    // validator count is required for CDF
		TotalPower:            b.ValidatorSet.TotalPower,       // total power between all validators is required for CDF
		VotingPower:           selfValidator.VotingPower,       // self voting power is required for CDF
	}
	// SORTITION (CDF + VRF)
	_, vrf, isCandidate := Sortition(&SortitionParams{
		SortitionData: b.SortitionData,
		PrivateKey:    b.PrivateKey,
	})
	// if is a possible proposer candidate, then send the VRF to other Replicas for the ElectionVote
	if isCandidate {
		b.SendToReplicas(b.CommitteeId, b.ValidatorSet, &Message{
			Header: b.View.Copy(),
			Vrf:    vrf,
		})
	}
}

// StartElectionVotePhase() begins the ElectionVotePhase after the ELECTION phase timeout
// ELECTION-VOTE PHASE:
// - Replicas review messages from Candidates and determine the 'Leader' by the highest VRF
// - If no Candidate messages received, fallback to stake weighted random 'Leader' selection
// - Replicas send a signed (aggregable) ELECTION vote to the Leader (Proposer)
// - With this vote, the Replica attaches any Byzantine evidence or 'Locked' QC they have collected as well as their VDF output
func (b *BFT) StartElectionVotePhase() {
	b.log.Info(b.View.ToString())
	// select Proposer (set is required for self-send)
	b.ProposerKey = SelectProposerFromCandidates(b.GetElectionCandidates(), b.SortitionData, b.ValidatorSet.ValidatorSet)
	defer func() { b.ProposerKey = nil }()
	b.log.Debugf("Voting %s as the proposer", lib.BytesToTruncatedString(b.ProposerKey))
	// get locally produced Verifiable delay function
	b.HighVDF = b.VDFService.Finish()
	// sign and send vote to Proposer
	b.SendToProposer(b.CommitteeId, &Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      b.View.Copy(),
			ProposerKey: b.ProposerKey, // using voting power, authorizes Candidate to act as the 'Leader'
		},
		HighQc:                 b.HighQC,                         // forward highest known 'Lock' for this Height, so the new Proposer may satisfy SAFE-NODE-PREDICATE
		LastDoubleSignEvidence: b.ByzantineEvidence.DSE.Evidence, // forward any evidence of DoubleSigning
		Vdf:                    b.HighVDF,                        // forward local VDF to the candidate
	})
}

// StartProposePhase() begins the ProposePhase after the ELECTION-VOTE phase timeout
// PROPOSE PHASE:
// - Leader reviews the collected vote messages from Replicas
//   - Determines the highest 'lock' (HighQC) if one exists for this Height
//   - Combines any ByzantineEvidence sent from Replicas into their own
//   - Aggregates the signatures from the Replicas to form a +2/3 threshold multi-signature
//
// - If a HighQC exists, use that as the Proposal - if not, the Leader produces a Proposal with ByzantineEvidence using the specific plugin
// - Leader creates a PROPOSE message from the Proposal and justifies the message with the +2/3 threshold multi-signature
func (b *BFT) StartProposePhase() {
	b.log.Info(b.View.ToString())
	vote, as, err := b.GetMajorityVote()
	if err != nil {
		return
	}
	b.log.Info("Self is the proposer")
	// produce new proposal or use highQC as the proposal
	if b.HighQC == nil {
		b.Block, b.Results, err = b.ProduceProposal(b.CommitteeId, b.ByzantineEvidence, b.HighVDF)
		if err != nil {
			b.log.Error(err.Error())
			return
		}
	}
	// send PROPOSE message to the replicas
	b.SendToReplicas(b.CommitteeId, b.ValidatorSet, &Message{
		Header: b.View.Copy(),
		Qc: &QC{
			Header:      vote.Qc.Header, // the current view
			Results:     b.Results,      // the proposed `certificate results`
			ResultsHash: b.Results.Hash(),
			Block:       b.Block,
			BlockHash:   crypto.Hash(b.Block),
			ProposerKey: vote.Qc.ProposerKey, // self-public-key, Replicas use this to validate the Aggregate (multi) Signature
			Signature:   as,                  // justifies them as the leader
		},
		HighQc:                 b.HighQC,                         // nil or justifies the proposal
		LastDoubleSignEvidence: b.ByzantineEvidence.DSE.Evidence, // evidence is attached (if any) to validate the Proposal
	})
}

// StartProposeVotePhase() begins the ProposeVote after the PROPOSE phase timeout
// PROPOSE-VOTE PHASE:
// - Replica reviews the message from the Leader by validating the justification (+2/3 multi-sig) proving that they are in-fact the leader
// - If the Replica is currently Locked on a previous Proposal for this Height, the new Proposal must pass the SAFE-NODE-PREDICATE
// - Replica Validates the proposal using the byzantine evidence and the specific plugin
// - Replicas send a signed (aggregable) PROPOSE vote to the Leader
func (b *BFT) StartProposeVotePhase() {
	b.log.Info(b.View.ToString())
	msg := b.GetProposal()
	if msg == nil {
		b.log.Warn("no valid message received from Proposer")
		b.RoundInterrupt()
		return
	}
	b.ProposerKey = msg.Signature.PublicKey
	b.log.Infof("Proposer is %s 👑", lib.BytesToTruncatedString(b.ProposerKey))
	// if locked, confirm safe to unlock
	if b.HighQC != nil {
		if err := b.SafeNode(msg); err != nil {
			b.log.Error(err.Error())
			b.RoundInterrupt()
			return
		}
	}
	// aggregate any evidence submitted from the replicas
	byzantineEvidence := &ByzantineEvidence{
		DSE: NewDSE(msg.LastDoubleSignEvidence),
	}
	// check candidate block against plugin
	if err := b.ValidateCertificate(b.CommitteeId, msg.Qc, byzantineEvidence); err != nil {
		b.log.Error(err.Error())
		b.RoundInterrupt()
		return
	}
	// Store the proposal data to enforce consistency during this voting round
	// Note: This is not the same as a `lock`, since a `lock` would keep the data even after the round changes
	b.Block, b.Results = msg.Qc.Block, msg.Qc.Results
	b.ByzantineEvidence = byzantineEvidence // BE stored in case of round interrupt and replicas locked on a proposal with BE
	// send vote to the proposer
	b.SendToProposer(b.CommitteeId, &Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      b.View.Copy(),
			BlockHash:   crypto.Hash(b.Block),
			ResultsHash: b.Results.Hash(),
			ProposerKey: b.ProposerKey,
		},
	})
}

// StartPrecommitPhase() begins the PrecommitPhase after the PROPOSE-VOTE phase timeout
// PRECOMMIT PHASE:
// - Leader reviews the collected Replica PROPOSE votes (votes signing off on the validity of the Leader's Proposal)
//   - Aggregates the signatures from the Replicas to form a +2/3 threshold multi-signature
//
// - Leader creates a PRECOMMIT message with the Proposal hashes and justifies the message with the +2/3 threshold multi-signature
func (b *BFT) StartPrecommitPhase() {
	b.log.Info(b.View.ToString())
	if !b.SelfIsProposer() {
		return
	}
	// get the VoteSet and aggregate signature that has +2/3 majority (by voting power) signatures from Replicas
	vote, as, err := b.GetMajorityVote()
	if err != nil {
		b.log.Error(err.Error())
		return
	}
	// send PRECOMMIT msg to Replicas
	b.SendToReplicas(b.CommitteeId, b.ValidatorSet, &Message{
		Header: b.Copy(),
		Qc: &QC{
			Header:      vote.Qc.Header,       // vote view
			BlockHash:   crypto.Hash(b.Block), // vote block payload
			ResultsHash: b.Results.Hash(),     // vote certificate results payload
			ProposerKey: b.ProposerKey,
			Signature:   as,
		},
	})
}

// StartPrecommitVotePhase() begins the Precommit vote after the PRECOMMIT phase timeout
// PRECOMMIT-VOTE PHASE:
// - Replica reviews the message from the Leader by validating the justification (+2/3 multi-sig) proving that +2/3rds of Replicas approved the Proposal
// - Replica `Locks` on the Proposal to protect those who may commit as a consequence of providing the aggregable signature
// - Replicas send a signed (aggregable) PROPOSE vote to the Leader
func (b *BFT) StartPrecommitVotePhase() {
	b.log.Info(b.View.ToString())
	msg := b.GetProposal()
	if msg == nil {
		b.log.Warn("no valid message received from Proposer")
		b.RoundInterrupt()
		return
	}
	// validate the proposer and proposal against local variables
	if interrupt := b.CheckProposerAndProposal(msg); interrupt {
		b.RoundInterrupt()
		return
	}
	// `lock` on the proposal (only by satisfying the SAFE-NODE-PREDICATE or COMMIT can this node unlock)
	b.HighQC = msg.Qc
	// send vote to the proposer
	b.SendToProposer(b.CommitteeId, &Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      b.View.Copy(),
			BlockHash:   crypto.Hash(b.Block),
			ResultsHash: b.Results.Hash(),
			ProposerKey: b.ProposerKey,
		},
	})
}

// StartCommitPhase() begins the Commit after the PRECOMMIT-VOTE phase timeout
// COMMIT PHASE:
// - Leader reviews the collected Replica PRECOMMIT votes (votes signing off on the validity of the Leader's Proposal)
//   - Aggregates the signatures from the Replicas to form a +2/3 threshold multi-signature
//
// - Leader creates a COMMIT message with the Proposal hashes and justifies the message with the +2/3 threshold multi-signature
func (b *BFT) StartCommitPhase() {
	b.log.Info(b.View.ToString())
	if !b.SelfIsProposer() {
		return
	}
	// get the VoteSet and aggregate signature that has +2/3 majority (by voting power) signatures from Replicas
	vote, as, err := b.GetMajorityVote()
	if err != nil {
		b.log.Error(err.Error())
		return
	}
	// SEND MSG TO REPLICAS
	b.SendToReplicas(b.CommitteeId, b.ValidatorSet, &Message{
		Header: b.Copy(), // header
		Qc: &QC{
			Header:      vote.Qc.Header,       // vote view
			BlockHash:   crypto.Hash(b.Block), // vote block payload
			ResultsHash: b.Results.Hash(),     // vote certificate results payload
			ProposerKey: b.ProposerKey,
			Signature:   as,
		},
	})
}

// StartCommitProcessPhase() begins the COMMIT-PROCESS phase after the COMMIT phase timeout
// COMMIT-PROCESS PHASE:
// - Replica reviews the message from the Leader by validating the justification (+2/3 multi-sig) proving that +2/3rds of Replicas are locked on the Proposal
// - Replica clears Byzantine Evidence
// - Replica gossips the Quorum Certificate message to Peers
// - If Leader, send the Proposal (reward) Transaction
func (b *BFT) StartCommitProcessPhase() {
	b.log.Info(b.View.ToString())
	msg := b.GetProposal()
	if msg == nil {
		b.log.Warn("no valid message received from Proposer")
		b.RoundInterrupt()
		return
	}
	// validate proposer and proposal against local variables
	if interrupt := b.CheckProposerAndProposal(msg); interrupt {
		b.RoundInterrupt()
		return
	}
	msg.Qc.Block, msg.Qc.Results = b.Block, b.Results
	// preset the Byzantine Evidence for the next height
	b.ByzantineEvidence = &ByzantineEvidence{
		DSE: b.GetLocalDSE(),
	}
	// gossip committed block message to peers
	b.GossipBlock(b.CommitteeId, msg.Qc)
	// if leader: send the proposal (reward) transaction
	if b.SelfIsProposer() {
		b.SendCertificateResultsTx(b.CommitteeId, msg.Qc)
	}
}

// RoundInterrupt() begins the ROUND-INTERRUPT phase after any phase errors
// ROUND-INTERRUPT:
// - Replica sends current View message to other replicas (Pacemaker vote)
func (b *BFT) RoundInterrupt() {
	b.log.Warn(b.View.ToString())
	b.Phase = RoundInterrupt
	// send pacemaker message
	b.SendToReplicas(b.CommitteeId, b.ValidatorSet, &Message{
		Qc: &lib.QuorumCertificate{
			Header: b.View.Copy(),
		},
	})
}

// Pacemaker() begins the Pacemaker process after ROUND-INTERRUPT timeout occurs
// - sets the highest round that +2/3rds majority of replicas have seen
func (b *BFT) Pacemaker() {
	b.log.Info(b.View.ToString())
	b.NewRound(false)
	// sort the pacemaker votes from the highest Round to the lowest Round
	var sortedVotes []*Message
	for _, vote := range b.PacemakerMessages {
		sortedVotes = append(sortedVotes, vote)
	}
	sort.Slice(sortedVotes, func(i, j int) bool {
		return sortedVotes[i].Qc.Header.Round >= sortedVotes[j].Qc.Header.Round
	})
	// loop from the highest Round to the lowest Round, summing the voting power until reaching round 0 or getting +2/3rds majority
	totalVotedPower, pacemakerRound := uint64(0), uint64(0)
	for _, vote := range sortedVotes {
		validator, err := b.ValidatorSet.GetValidator(vote.Signature.PublicKey)
		if err != nil {
			b.log.Warn(err.Error())
			continue
		}
		totalVotedPower += validator.VotingPower
		if totalVotedPower >= b.ValidatorSet.MinimumMaj23 {
			pacemakerRound = vote.Qc.Header.Round // set the highest round where +2/3rds have been
			break
		}
	}
	// if +2/3rd Round is larger than local Round - advance to the +2/3rd Round to better join the Majority
	if pacemakerRound > b.Round {
		b.log.Infof("Pacemaker peers set round: %d", pacemakerRound)
		b.Round = pacemakerRound
	}
}

// PacemakerMessages is a collection of 'View' messages keyed by each Replica's public key
// These messages help Replicas synchronize their Rounds more effectively during periods of instability or failure
type PacemakerMessages map[string]*Message // [ public_key_string ] -> View message

// AddPacemakerMessage() adds the 'View' message to the list (keyed by public key string)
func (b *BFT) AddPacemakerMessage(msg *Message) (err lib.ErrorI) {
	b.PacemakerMessages[lib.BytesToString(msg.Signature.PublicKey)] = msg
	return
}

// PhaseHas23Maj() returns true if the node received enough messages to optimistically move forward
func (b *BFT) PhaseHas23Maj() bool {
	switch b.Phase {
	case ElectionVote, ProposeVote, PrecommitVote, CommitProcess:
		return b.GetProposal() != nil
	case Propose, Precommit, Commit:
		_, _, err := b.GetMajorityVote()
		return err == nil
	}
	return false
}

// CheckProposerAndProposal() ensures the Leader message has the correct sender public key and correct ProposalHash
func (b *BFT) CheckProposerAndProposal(msg *Message) (interrupt bool) {
	// confirm is expected proposer
	if !b.IsProposer(msg.Signature.PublicKey) {
		b.log.Error(lib.ErrInvalidProposerPubKey().Error())
		return true
	}

	// confirm is expected proposal
	if !bytes.Equal(crypto.Hash(b.Block), msg.Qc.BlockHash) || !bytes.Equal(b.Results.Hash(), msg.Qc.ResultsHash) {
		b.log.Error(ErrMismatchedProposals().Error())
		return true
	}
	return
}

// NewRound() initializes the VoteSet and Proposals cache for the next round
// - increments the round count if not NewHeight (goes to Round 0)
func (b *BFT) NewRound(newHeight bool) {
	if newHeight {
		b.Round = 0
	} else {
		b.Round++
	}
	b.Votes.NewRound(b.Round)
	b.Proposals[b.Round] = make(map[string][]*Message)
}

// NewHeight() initializes / resets consensus variables preparing for the NewHeight
func (b *BFT) NewHeight(keepLocks ...bool) {
	// reset VotesForHeight
	b.Votes = make(VotesForHeight)
	// reset ProposalsForHeight
	b.Proposals = make(ProposalsForHeight)
	// reset PacemakerMessages
	b.PacemakerMessages = make(PacemakerMessages)
	// reset PartialQCs
	b.PartialQCs = make(PartialQCs)
	// if resetting due to new Canopy Block and Validator Set then KeepLocks
	// - protecting any who may have committed against attacks like malicious proposers from withholding
	// COMMIT_MSG and sending it after the next block is produces
	if keepLocks == nil || !keepLocks[0] {
		b.HighQC = nil
		// begin the verifiable delay function for the next height
		if err := b.RunVDF(); err != nil {
			b.log.Errorf("RunVDF() failed with error, %s", err.Error())
		}
		b.Height++
		b.CanopyHeight = b.Controller.GetCanopyHeight()
	}
	// reset ProposerKey, Proposal, and Sortition data
	b.ProposerKey = nil
	b.Block, b.Results = nil, nil
	b.SortitionData = nil
	// initialize Round 0
	b.NewRound(true)
	// set phase to Election
	b.Phase = Election
}

// SafeNode is the codified Hotstuff SafeNodePredicate:
// - Protects replicas who may have committed to a previous value by locking on that value when signing a Precommit Message
// - May unlock if new proposer:
//   - SAFETY: uses the same value the replica is locked on (safe because it will match the value that may have been committed by others)
//   - LIVENESS: uses a lock with a higher round (safe because replica is convinced no other replica committed to their locked value as +2/3rds locked on a higher round)
func (b *BFT) SafeNode(msg *Message) lib.ErrorI {
	if msg == nil || msg.Qc == nil || msg.HighQc == nil {
		return ErrEmptyMessage()
	}
	// ensure the messages' HighQC justifies its proposal (should have the same hashes)
	if !bytes.Equal(crypto.Hash(msg.Qc.Block), msg.HighQc.BlockHash) && !bytes.Equal(msg.Qc.Results.Hash(), msg.HighQc.ResultsHash) {
		return ErrMismatchedProposals()
	}
	// if the hashes of the Locked proposal is the same as the Leader's message
	if bytes.Equal(b.HighQC.BlockHash, msg.HighQc.BlockHash) && bytes.Equal(b.HighQC.ResultsHash, msg.HighQc.ResultsHash) {
		return nil // SAFETY (SAME PROPOSAL AS LOCKED)
	}
	// if the view of the Locked proposal is older than the Leader's message
	if msg.HighQc.Header.CanopyHeight > b.HighQC.Header.CanopyHeight || msg.HighQc.Header.Round > b.HighQC.Header.Round {
		return nil // LIVENESS (HIGHER ROUND v COMMITTEE THAN LOCKED)
	}
	return ErrFailedSafeNodePredicate()
}

// SetTimerForNextPhase() calculates the wait time for a specific phase/Round, resets the Phase and Optimistic timers
func (b *BFT) SetTimerForNextPhase(processTime time.Duration) {
	waitTime, optimisticTime := b.WaitTime(b.Phase, b.Round), b.WaitTime(b.Phase, 10)
	switch b.Phase {
	default:
		b.Phase++
	case CommitProcess:
		// no op
	case Pacemaker:
		b.Phase = Election
	}
	b.SetWaitTimers(waitTime, optimisticTime, processTime)
}

// WaitTime() returns the wait time (wait and receive consensus messages) for a specific Phase.Round
func (b *BFT) WaitTime(phase Phase, round uint64) (waitTime time.Duration) {
	switch phase {
	case Election:
		waitTime = b.waitTime(b.Config.ElectionTimeoutMS, round)
	case ElectionVote:
		waitTime = b.waitTime(b.Config.ElectionVoteTimeoutMS, round)
	case Propose:
		waitTime = b.waitTime(b.Config.ProposeTimeoutMS, round)
	case ProposeVote:
		waitTime = b.waitTime(b.Config.ProposeVoteTimeoutMS, round)
	case Precommit:
		waitTime = b.waitTime(b.Config.PrecommitTimeoutMS, round)
	case PrecommitVote:
		waitTime = b.waitTime(b.Config.PrecommitVoteTimeoutMS, round)
	case Commit:
		waitTime = b.waitTime(b.Config.CommitTimeoutMS, round)
	case CommitProcess:
		waitTime = b.waitTime(b.Config.CommitProcessMS, round)
	case RoundInterrupt:
		waitTime = b.waitTime(b.Config.RoundInterruptTimeoutMS, round)
	case Pacemaker:
		waitTime = b.waitTime(b.Config.CommitProcessMS, round)
	}
	return
}

// waitTime() calculates the waiting time for a specific sleepTime configuration and Round number (helper)
func (b *BFT) waitTime(sleepTimeMS int, round uint64) time.Duration {
	return time.Duration(uint64(sleepTimeMS)*(2*round+1)) * time.Millisecond
}

// SetWaitTimers() sets the phase and optimistic timers
// - Phase Timeout ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
// - Optimistic Timeout enables 'Optimistic Responsiveness Mode' starting from Round 10, allowing faster consensus
// This design balances synchronization speed during adverse conditions with maximizing voter participation under normal conditions
func (b *BFT) SetWaitTimers(phaseWaitTime, optimisticWaitTIme, processTime time.Duration) {
	subtract := func(wt, pt time.Duration) (t time.Duration) {
		if pt > 700*time.Hour {
			return wt
		}
		if wt <= pt {
			return 0
		}
		return wt - pt
	}
	// calculate the phase timer and the optimistic timer by subtracting the process time
	phaseWaitTime, optimisticWaitTime := subtract(phaseWaitTime, processTime), subtract(optimisticWaitTIme, processTime)
	b.log.Debugf("Setting consensus timer: %.2fS", phaseWaitTime.Seconds())
	// set Phase and Optimistic timers to go off in their respective timeouts
	lib.ResetTimer(b.PhaseTimer, phaseWaitTime)
	lib.ResetTimer(b.OptimisticTimer, optimisticWaitTime)
}

// SelfIsPropose() returns true if this node is the Leader
func (b *BFT) SelfIsProposer() bool { return b.IsProposer(b.PublicKey) }

// IsProposer() returns true if specific public key is the expected Leader public key
func (b *BFT) IsProposer(id []byte) bool { return bytes.Equal(id, b.ProposerKey) }

// SelfIsValidator() returns true if this node is part of the ValSet
func (b *BFT) SelfIsValidator() bool {
	selfValidator, _ := b.ValidatorSet.GetValidator(b.PublicKey)
	return selfValidator != nil
}

// RunVDF() runs the verifiable delay service
func (b *BFT) RunVDF() lib.ErrorI {
	// generate the VDF seed
	seed, err := b.VDFSeed(b.PublicKey)
	if err != nil {
		return err
	}
	// run the VDF generation
	go b.VDFService.Run(seed)
	return nil
}

// VDFSeed() generates the seed for the verifiable delay service
func (b *BFT) VDFSeed(publicKey []byte) ([]byte, lib.ErrorI) {
	lastQuorumCertificate, err := b.LoadCertificate(b.CommitteeId, b.Height-1)
	if err != nil {
		return nil, err
	}
	if lastQuorumCertificate == nil {
		return nil, lib.ErrEmptyQuorumCertificate()
	}
	return append(lastQuorumCertificate.BlockHash, publicKey...), nil
}

// VerifyVDF() validates the VDF from a Replica
func (b *BFT) VerifyVDF(vote *Message) (bool, lib.ErrorI) {
	seed, err := b.VDFSeed(vote.Signature.PublicKey)
	if err != nil {
		return false, err
	}
	return b.VDFService.VerifyVDF(seed, vote.Vdf), nil
}

// ResetBFTChan() is a callback trigger that allows the caller to Reset the BFT to Round 0 with a varying sleep
func (b *BFT) ResetBFTChan() chan ResetBFT { return b.resetBFT }

// ResetBFT is a structure that allows the Controller to reset the BFT either due to a Target chain block or Canopy block
type ResetBFT struct {
	UpdatedCanopyHeight uint64        // new Canopy height
	UpdatedCommitteeSet ValSet        // new Committee from the Canopy block
	ProcessTime         time.Duration // process Target block time
}

// phaseToString() converts the phase object to a human-readable string
func phaseToString(p Phase) string {
	return fmt.Sprintf("%d_%s", p, lib.Phase_name[int32(p)])
}

type (
	// aliases for easy library variable access
	QC     = lib.QuorumCertificate
	Phase  = lib.Phase
	ValSet = lib.ValidatorSet

	// Controller defines the expected parent interface for the BFT structure, providing various callback functions
	// that manage interactions with BFT and other parts of the application like Plugins, P2P and Storage
	Controller interface {
		Lock()
		Unlock()
		// GetCanopyHeight returns the height of the base-chain
		GetCanopyHeight() uint64
		// ProduceProposal() is a plugin call to produce a Proposal object as a Leader
		ProduceProposal(committeeID uint64, be *ByzantineEvidence, vdf *crypto.VDF) (block []byte, results *lib.CertificateResult, err lib.ErrorI)
		// ValidateCertificate() is a plugin call to validate a Certificate object as a Replica
		ValidateCertificate(committeeID uint64, qc *lib.QuorumCertificate, evidence *ByzantineEvidence) lib.ErrorI
		// LoadCommittee() loads the ValidatorSet operating under CommitteeID
		LoadCommittee(committeeID uint64, canopyHeight uint64) (lib.ValidatorSet, lib.ErrorI)
		// LastCommitteeRewardHeight() loads the last height a committee member executed a Proposal (reward) transaction
		LoadCommitteeHeightInState(committeeID uint64) uint64
		// LoadCertificate() gets the Quorum Certificate from the committeeID-> plugin at a certain height
		LoadCertificate(committeeID uint64, height uint64) (*lib.QuorumCertificate, lib.ErrorI)
		// LoadLastProposers() loads the last Canopy committee proposers for sortition data
		LoadLastProposers() *lib.Proposers
		// LoadMinimumEvidenceHeight() loads the Canopy enforced minimum height for valid Byzantine Evidence
		LoadMinimumEvidenceHeight() (uint64, lib.ErrorI)
		// IsValidDoubleSigner() checks to see if the double signer is valid for this specific height
		IsValidDoubleSigner(height uint64, address []byte) bool
		// SendCertMsg() is a P2P call to gossip a completed Quorum Certificate with a Proposal
		GossipBlock(committeeID uint64, certificate *lib.QuorumCertificate)
		// SendCertificateResultsTx() is a P2P call that allows a Leader to submit their CertificateResults (reward) transaction
		SendCertificateResultsTx(committeeID uint64, certificate *lib.QuorumCertificate)
		// SendConsMsgToReplicas() is a P2P call to directly send a Consensus message to all Replicas
		SendToReplicas(committeeID uint64, replicas lib.ValidatorSet, msg lib.Signable)
		// SendConsMsgToProposer() is a P2P call to directly send a Consensus message to the Leader
		SendToProposer(committeeID uint64, msg lib.Signable)
		// Syncing() returns true if the plugin is currently syncing
		Syncing(committeeID uint64) *atomic.Bool
	}
)

const (
	Election       = lib.Phase_ELECTION
	ElectionVote   = lib.Phase_ELECTION_VOTE
	Propose        = lib.Phase_PROPOSE
	ProposeVote    = lib.Phase_PROPOSE_VOTE
	Precommit      = lib.Phase_PRECOMMIT
	PrecommitVote  = lib.Phase_PRECOMMIT_VOTE
	Commit         = lib.Phase_COMMIT
	CommitProcess  = lib.Phase_COMMIT_PROCESS
	RoundInterrupt = lib.Phase_ROUND_INTERRUPT
	Pacemaker      = lib.Phase_PACEMAKER

	CommitProcessToVDFTargetCoefficient = .80 // how much the commit process time is reduced for VDF processing
)
