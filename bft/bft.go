package bft

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"slices"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// 08/25 BYZANTINE FAULT TOLERANCE PROTOCOL BREAKING CHANGE: ADD ROOT-BUILD-HEIGHT TO THE QUORUM CERTIFICATE
//
// NOTE: This is not expected to be a consensus-breaking change. Historical syncers will work fine without embedding an "if height == X then do Y" condition.
//
// 3 PHASES TO THIS UPGRADE:
//   1) Backwards compatible feature add
//   2) Upgrade height which switches to the embedded 'root build height' once Validators all upgraded
//   3) Cleanup the deprecated code

var PROTOCOL_BREAK_UPGRADE_HEIGHT = uint64(109010)

func (b *BFT) BeforeUpgradeHeight() bool {
	if b.LoadIsOwnRoot() {
		return b.Height < PROTOCOL_BREAK_UPGRADE_HEIGHT
	} else {
		return b.RootHeight < PROTOCOL_BREAK_UPGRADE_HEIGHT
	}
}

// BFT is a structure that holds data for a Hotstuff BFT instance
type BFT struct {
	*lib.View                            // the current period during which the BFT is occurring (Height/Round/Phase)
	Votes         VotesForHeight         // 'votes' received from Replica (non-leader) Validators
	Proposals     ProposalsForHeight     // 'proposals' received from the Leader Validator(s)
	ProposerKey   []byte                 // the public key of the proposer
	ValidatorSet  ValSet                 // the current set of Validators
	CommitteeData *lib.CommitteeData     // the data for the committee for this height
	HighQC        *QC                    // the highest PRECOMMIT quorum certificate the node is aware of for this Height
	RCBuildHeight uint64                 // the build height of the locked proposal
	Block         []byte                 // the current Block being voted on (the foundational unit of the blockchain)
	BlockHash     []byte                 // the current hash of the block being voted on
	Results       *lib.CertificateResult // the current Result being voted on (reward and slash recipients)
	BlockResult   *lib.BlockResult       // the cached result of the block - allowing a validator to validate the block only once and directly commit
	SortitionData *lib.SortitionData     // the current data being used for VRF+CDF Leader Election
	VDFService    *lib.VDFService        // the verifiable delay service, run once per block as a deterrent against long-range-attacks
	HighVDF       *crypto.VDF            // the highest VDF among replicas - if the chain is using VDF for long-range-attack protection
	VDFCache      []*Message             // the cache of VDFs used for long-range-attack protection, validated at the PROPOSE phase

	ByzantineEvidence *ByzantineEvidence // evidence of faulty or malicious Validators collected during the BFT process
	PartialQCs        PartialQCs         // potentially implicating evidence that may turn into ByzantineEvidence if paired with an equivocating QC
	PacemakerMessages PacemakerMessages  // View messages from the current ValidatorSet allowing the node to synchronize to the highest +2/3 seen Round

	Controller               // reference to the Controller for callbacks like producing and validating the proposal via the plugin or gossiping commit message
	ResetBFT   chan ResetBFT // trigger that resets the BFT due to a new Target block or a new Canopy block
	syncing    *atomic.Bool  // if chain for this committee is currently catching up to latest height

	PhaseTimer     *time.Timer // ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
	LastCommitTime time.Time   // the last time a block was committed

	PublicKey  []byte             // self consensus public key
	PrivateKey crypto.PrivateKeyI // self consensus private key
	Config     lib.Config         // self configuration
	Metrics    *lib.Metrics       // telemetry
	log        lib.LoggerI        // logging
}

// New() creates a new instance of HotstuffBFT for a specific Committee
func New(c lib.Config, valKey crypto.PrivateKeyI, rootHeight, height uint64, con Controller, vdfEnabled bool, m *lib.Metrics, l lib.LoggerI) (*BFT, lib.ErrorI) {
	// determine if using a Verifiable Delay Function for long-range-attack protection
	var vdf *lib.VDFService
	// calculate the targetTime from commitProcess and set the VDF
	if vdfEnabled {
		vdfTargetTime := time.Duration(float64(c.BlockTimeMS())*BlockTimeToVDFTargetCoefficient) * time.Millisecond
		vdf = lib.NewVDFService(vdfTargetTime, l)
	}
	return &BFT{
		View: &lib.View{
			Height:     height,
			RootHeight: rootHeight,
			NetworkId:  c.NetworkID,
			ChainId:    c.ChainId,
		},
		Votes:     make(VotesForHeight),
		Proposals: make(ProposalsForHeight),
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
		ResetBFT:          make(chan ResetBFT, 100),
		syncing:           con.Syncing(),
		PhaseTimer:        lib.NewTimer(),
		VDFService:        vdf,
		Metrics:           m,
		HighVDF:           new(crypto.VDF),
		VDFCache:          []*Message{},
	}, nil
}

// Start() initiates the HotStuff BFT service.
// - Phase Timeout ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
// This design balances synchronization speed during adverse conditions with maximizing voter participation under normal conditions
// - ResetBFT occurs upon receipt of a Quorum Certificate
//   - (a) Canopy chainId <committeeSet changed, reset but keep locks to prevent conflicting validator sets between peers during a view change>
//   - (b) Target chainId <mission accomplished, move to next height>
func (b *BFT) Start() {
	var err lib.ErrorI
	var resetOnRootHeight uint64
	// load the committee from the base chain
	b.ValidatorSet, err = b.Controller.LoadCommittee(b.LoadRootChainId(b.ChainHeight()), b.Controller.RootChainHeight())
	if err != nil {
		b.log.Warn(err.Error())
	}
	// load the committee data
	b.CommitteeData, err = b.Controller.LoadCommitteeData()
	if err != nil {
		b.log.Warn(err.Error())
	}
	for {
		select {
		// EXECUTE PHASE
		// - This triggers when the phase's sleep time has expired, indicating that all expected messages for this phase should have already been received
		case <-b.PhaseTimer.C:
			func() {
				b.Controller.Lock()
				defer b.Controller.Unlock()
				// Update BFT metrics
				defer b.Metrics.UpdateBFTMetrics(b.Height, b.RootHeight, b.Round, b.Phase, time.Now())
				// handle the phase
				b.HandlePhase()
			}()

		// RESET BFT
		// - This triggers when receiving a new Commit Block (QC) from either root-chainId (a) or the Target-ChainId (b)
		case resetBFT := <-b.ResetBFT:
			var processTime time.Duration
			func() {
				b.Controller.Lock()
				defer b.Controller.Unlock()
				// get the last commit time from the meta
				lastCommitTime := time.UnixMicro(int64(resetBFT.BFTMeta.GetLastCommitTime()))
				// calculate time since
				since := time.Since(lastCommitTime)
				// allow if 'since' is less than 1 block old
				if int(since.Milliseconds()) < b.Config.BlockTimeMS() {
					b.log.Infof("Using last commit timestamp for bft timing: %s", lastCommitTime.Format(time.StampMilli))
					processTime = since
				}
				// if is a root-chain update reset back to round 0 but maintain locks to prevent 'fork attacks'
				// else increment the height and don't maintain locks
				if !resetBFT.IsRootChainUpdate {
					b.log.Info("RESET BFT (NEW_HEIGHT)")
					b.NewHeight(false)
					b.SetWaitTimers(b.NewHeightWaitTime(), processTime)
					b.LastCommitTime = time.UnixMicro(int64(resetBFT.BFTMeta.GetLastCommitTime()))
					resetOnRootHeight = resetBFT.BFTMeta.GetResetOnRootHeight()
				} else {
					if b.BeforeUpgradeHeight() { // TODO DEPRECATE (11)
						b.NewHeight(true)
						b.SetWaitTimers(time.Duration(b.Config.NewHeightTimeoutMs)*time.Millisecond, processTime)
					} else {
						if b.LoadIsOwnRoot() || resetOnRootHeight != 0 && b.Round == 0 && resetOnRootHeight == resetBFT.RootHeight {
							b.log.Infof("RESET BFT (ROOT_HEIGHT: %d)", resetBFT.RootHeight)
							b.NewHeight(true)
							b.SetWaitTimers(b.NewHeightWaitTime(), processTime)
						} else {
							b.log.Infof("NOTIFICATION (NEW_ROOT_HEIGHT: %d)", resetBFT.RootHeight)
						}
					}
				}
			}()
		}
	}
}

// HandlePhase() is the main BFT Phase stepping loop
func (b *BFT) HandlePhase() {
	stopTimers := func() { b.PhaseTimer.Stop() }
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
	lastProposers, err := b.LoadLastProposers(b.RootHeight)
	if err != nil {
		b.log.Error(err.Error())
		return
	}
	// initialize the sortition parameters
	b.SortitionData = &lib.SortitionData{
		LastProposerAddresses: lastProposers.Addresses,      // LastProposers ensures defense against Grinding Attacks
		RootHeight:            b.RootHeight,                 // the height of the root ensures a unique sortition for each root height
		Height:                b.Height,                     // height ensures a unique sortition seed for each height
		Round:                 b.Round,                      // round ensures a unique sortition seed for each round
		TotalValidators:       b.ValidatorSet.NumValidators, // validator count is required for CDF
		TotalPower:            b.ValidatorSet.TotalPower,    // total power between all validators is required for CDF
		VotingPower:           selfValidator.VotingPower,    // self voting power is required for CDF
	}
	// SORTITION (CDF + VRF)
	_, vrf, isCandidate := Sortition(&SortitionParams{
		SortitionData: b.SortitionData,
		PrivateKey:    b.PrivateKey,
	})
	// if is a possible proposer candidate, then send the VRF to other Replicas for the ElectionVote
	if isCandidate {
		b.log.Info("Self is a leader candidate 🗳️")
		b.SendToReplicas(b.ValidatorSet, &Message{
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
	// get the candidates from messages received
	candidates := b.GetElectionCandidates()
	if len(candidates) == 0 {
		b.log.Warn("No election candidates, falling back to weighted pseudorandom")
	}
	// select Proposer (set is required for self-send)
	b.ProposerKey = SelectProposerFromCandidates(candidates, b.SortitionData, b.ValidatorSet.ValidatorSet)
	defer func() { b.ProposerKey = nil }()
	if b.SelfIsProposer() {
		b.log.Info("Voting SELF as the proposer")
	} else {
		b.log.Infof("Voting %s as the proposer", lib.BytesToTruncatedString(b.ProposerKey))
	}
	// get locally produced Verifiable delay function
	b.HighVDF = b.VDFService.Finish()
	// TODO DEPRECATE (1)
	var rootBuildHeight uint64
	if b.BeforeUpgradeHeight() {
		rootBuildHeight = b.RCBuildHeight
	} else {
		rootBuildHeight = b.RootBuildHeight
	}
	// sign and send vote to Proposer
	b.SendToProposer(&Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      b.View.Copy(),
			ProposerKey: b.ProposerKey, // using voting power, authorizes Candidate to act as the 'Leader'
		},
		HighQc:                 b.HighQC,                         // forward highest known 'Lock' for this Height, so the new Proposer may satisfy SAFE-NODE-PREDICATE
		LastDoubleSignEvidence: b.ByzantineEvidence.DSE.Evidence, // forward any evidence of DoubleSigning
		Vdf:                    b.HighVDF,                        // forward local VDF to the candidate
		RcBuildHeight:          rootBuildHeight,                  // forward the highQC build height (if applicable)
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
	// select the highest VDF from the cache
	highVDF, err := b.selectHighestVDF()
	if err != nil {
		return
	}
	b.HighVDF = highVDF
	// TODO DEPRECATE (2)
	var rootBuildHeight uint64
	if b.BeforeUpgradeHeight() {
		rootBuildHeight = b.RCBuildHeight
	} else {
		rootBuildHeight = b.RootBuildHeight
	}
	// produce new proposal or use highQC as the proposal
	if b.HighQC == nil {
		rootBuildHeight, b.Block, b.Results, err = b.ProduceProposal(b.ByzantineEvidence, b.HighVDF)
		if err != nil {
			b.log.Error(err.Error())
			return
		}
	} else {
		b.Block, b.Results = b.HighQC.Block, b.HighQC.Results
	}
	// get view
	view := b.View.Copy()
	// TODO DEPRECATE (3)
	if b.BeforeUpgradeHeight() {
		b.RCBuildHeight = rootBuildHeight
	} else {
		view.RootBuildHeight = rootBuildHeight
		b.RootBuildHeight = rootBuildHeight
	}
	// send PROPOSE message to the replicas
	b.SendToReplicas(b.ValidatorSet, &Message{
		Header: view,
		Qc: &QC{
			Header:      vote.Qc.Header, // the current view
			Results:     b.Results,      // the proposed `certificate results`
			ResultsHash: b.Results.Hash(),
			Block:       b.Block,
			BlockHash:   b.GetBlockHash(),
			ProposerKey: vote.Qc.ProposerKey, // self-public-key, Replicas use this to validate the Aggregate (multi) Signature
			Signature:   as,                  // justifies them as the leader
		},
		HighQc:                 b.HighQC,                         // nil or justifies the proposal
		LastDoubleSignEvidence: b.ByzantineEvidence.DSE.Evidence, // evidence is attached (if any) to validate the Proposal
		RcBuildHeight:          rootBuildHeight,                  // the root chain height when the block was built
	})
}

// StartProposeVotePhase() begins the ProposeVote after the PROPOSE phase timeout
// PROPOSE-VOTE PHASE:
// - Replica reviews the message from the Leader by validating the justification (+2/3 multi-sig) proving that they are in-fact the leader
// - If the Replica is currently Locked on a previous Proposal for this Height, the new Proposal must pass the SAFE-NODE-PREDICATE
// - Replica Validates the proposal using the byzantine evidence and the specific plugin
// - Replicas send a signed (aggregable) PROPOSE vote to the Leader
func (b *BFT) StartProposeVotePhase() {
	var err lib.ErrorI
	b.log.Info(b.View.ToString())
	msg := b.GetProposal()
	if msg == nil {
		b.log.Warn("no valid message received from Proposer")
		b.RoundInterrupt()
		return
	}
	b.ProposerKey = msg.Signature.PublicKey
	if b.SelfIsProposer() {
		b.log.Infof("Proposer is SELF 👑")
	} else {
		b.log.Infof("Proposer is %s 👑", lib.BytesToTruncatedString(b.ProposerKey))
	}
	// if locked, confirm safe to unlock
	if b.HighQC != nil {
		if err := b.SafeNode(msg); err != nil {
			b.log.Error(err.Error())
			b.RoundInterrupt()
			return
		}
	}
	// TODO DEPRECATE (4)
	var rootBuildHeight uint64
	if b.BeforeUpgradeHeight() {
		rootBuildHeight = msg.RcBuildHeight
	} else {
		rootBuildHeight = msg.Header.RootBuildHeight
	}
	// ensure the build height isn't too old
	if rootBuildHeight < b.CommitteeData.LastRootHeightUpdated {
		b.log.Error(lib.ErrInvalidRCBuildHeight(rootBuildHeight, b.CommitteeData.LastRootHeightUpdated).Error())
		b.RoundInterrupt()
		return
	}
	// aggregate any evidence submitted from the replicas
	byzantineEvidence := &ByzantineEvidence{
		DSE: NewDSE(msg.LastDoubleSignEvidence),
	}
	// check candidate block against FSM
	if b.BlockResult, err = b.ValidateProposal(rootBuildHeight, msg.Qc, byzantineEvidence); err != nil {
		b.log.Error(err.Error())
		b.RoundInterrupt()
		return
	}
	// Store the proposal data to enforce consistency during this voting round
	// Note: This is not the same as a `lock`, since a `lock` would keep the data even after the round changes
	b.Block, b.Results = msg.Qc.Block, msg.Qc.Results
	b.ByzantineEvidence = byzantineEvidence // BE stored in case of round interrupt and replicas locked on a proposal with BE
	// start the VDF service on this block hash
	if err := b.RunVDF(b.GetBlockHash()); err != nil {
		b.log.Errorf("RunVDF() failed with error, %s", err.Error())
	}
	// get view and set Root chain build height
	view := b.View.Copy()
	// set RootBuildHeight
	view.RootBuildHeight = rootBuildHeight
	// send vote to the proposer
	b.SendToProposer(&Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      view,
			BlockHash:   b.GetBlockHash(),
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
		b.RoundInterrupt()
		return
	}
	// TODO DEPRECATE (5)
	// get view
	view := b.View.Copy()
	var rootBuildHeight uint64
	if b.BeforeUpgradeHeight() {
		rootBuildHeight = b.RCBuildHeight
	} else {
		view.RootBuildHeight = vote.Qc.Header.RootBuildHeight
		rootBuildHeight = vote.Qc.Header.RootBuildHeight
	}
	// send PRECOMMIT msg to Replicas
	b.SendToReplicas(b.ValidatorSet, &Message{
		Header: view,
		Qc: &QC{
			Header:      vote.Qc.Header,   // vote view
			BlockHash:   b.GetBlockHash(), // vote block payload
			ResultsHash: b.Results.Hash(), // vote certificate results payload
			ProposerKey: b.ProposerKey,
			Signature:   as,
		},
		RcBuildHeight: rootBuildHeight,
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
	// TODO DEPRECATE (6)
	if b.BeforeUpgradeHeight() {
		b.RCBuildHeight = msg.RcBuildHeight
	} else {
		b.RootBuildHeight = msg.Header.RootBuildHeight
	}
	b.HighQC.Block = b.Block
	b.HighQC.Results = b.Results
	b.log.Infof("🔒 Locked on proposal %s", lib.BytesToTruncatedString(b.HighQC.BlockHash))
	// send vote to the proposer
	b.SendToProposer(&Message{
		Qc: &QC{ // NOTE: Replicas use the QC to communicate important information so that it's aggregable by the Leader
			Header:      b.View.Copy(),
			BlockHash:   b.GetBlockHash(),
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
		b.RoundInterrupt()
		return
	}
	// SEND MSG TO REPLICAS
	b.SendToReplicas(b.ValidatorSet, &Message{
		Header: b.Copy(), // header
		Qc: &QC{
			Header:      vote.Qc.Header,   // vote view
			BlockHash:   b.GetBlockHash(), // vote block payload
			ResultsHash: b.Results.Hash(), // vote certificate results payload
			ProposerKey: b.ProposerKey,
			Signature:   as,
		},
		BftCoordinationMeta: b.GetBFTCoordinationMeta(),                                      // provide the next replicas BFT coordination information to help prevent partitions
		Timestamp:           uint64(time.Now().Add(b.WaitTime(Commit, b.Round)).UnixMicro()), // TODO deprecate (9)
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
	// non-blocking
	go func() {
		// TODO deprecate msg.Timestamp (10)
		// send the block to self for committing
		b.SelfSendBlock(msg.Qc, msg.Timestamp, msg.BftCoordinationMeta)
		// wait to allow for CommitProcess to finish
		<-time.After(time.Duration(b.Config.CommitTimeoutMS) * time.Millisecond)
		// gossip committed block message to peers
		b.GossipBlock(msg.Qc, b.PublicKey, msg.Timestamp, msg.BftCoordinationMeta)
	}()
}

// RoundInterrupt() begins the ROUND-INTERRUPT phase after any phase errors
// ROUND-INTERRUPT:
// - Replica sends current View message to other replicas (Pacemaker vote)
func (b *BFT) RoundInterrupt() {
	_ = b.VDFService.Finish() // stop VDF service because the block hash that was being used as a seed will change
	b.Config.RoundInterruptTimeoutMS = b.msLeftInRound()
	b.log.Warnf("Starting next round in %.2f secs", (time.Duration(b.Config.RoundInterruptTimeoutMS) * time.Millisecond).Seconds())
	b.Phase = RoundInterrupt
	b.BlockResult = nil
	b.VDFCache = []*Message{}
	b.ResetFSM()
	view := b.View.Copy()
	// determine the best faction based on what is currently known
	var votingPower uint64
	votingPower, view.RootHeight, view.Round = b.DetermineNextRootHeightAndRound(b.Round + 1)
	if b.ValidatorSet.MinimumMaj23 > 0 {
		b.log.Infof("Round Interrupt set round (%d) and root height (%d) with VP: %.2f%%",
			view.Round, view.RootHeight, float64(votingPower)/float64(b.ValidatorSet.TotalPower)*100)
	}
	// send pacemaker message
	b.SendToReplicas(b.ValidatorSet, &Message{
		Qc: &lib.QuorumCertificate{Header: view},
	})
}

// Pacemaker() begins the Pacemaker process after ROUND-INTERRUPT timeout occurs
//   - When advancing to the next round, there may be no reliable leader to coordinate a reset.
//   - If the root chain notification arrives while half the validators have already switched to the new round and the other half haven’t,
//     the set can split on root height and root info.
//   - Identify the largest voting-power group (“faction”) that agrees on (RootHeight X, Round Y).
//   - Wait until there’s a ≥2/3 majority voting for that faction or timeout after 10 seconds
func (b *BFT) Pacemaker() {
	b.log.Info(b.View.ToString())
	b.NewRound(false)
	// determine largest faction
	totalVP, rootHeight, nextRound := b.DetermineNextRootHeightAndRound(b.Round)
	// set round
	b.Round = nextRound
	// set root height and refresh root chain info
	b.RefreshRootChainInfo(rootHeight)
	// log with div 0 protection
	if b.ValidatorSet.MinimumMaj23 > 0 {
		b.log.Infof("Pacemaker set round (%d) and root height (%d) with VP: %.2f%%",
			nextRound, rootHeight, float64(totalVP)/float64(b.ValidatorSet.TotalPower)*100)
	}
	// clear the pacemaker messages
	b.PacemakerMessages = make(PacemakerMessages)
	// exit
	return
}

// PacemakerMessages is a collection of 'View' messages keyed by each Replica's public key
// These messages help Replicas synchronize their Rounds more effectively during periods of instability or failure
type PacemakerMessages map[string]*Message // [ public_key_string ] -> View message

// DetermineNextRootHeightAndRound() identifies the largest voting-power group (“faction”) that agrees on
// (RootHeight X, Round Y) where Round Y is >= self.NextRound
func (b *BFT) DetermineNextRootHeightAndRound(round uint64) (totalVP, rootHeight, nextRound uint64) {
	rootHeights, rounds := make(map[uint64]uint64), make(map[uint64]uint64) // rootHeight/round -> votingPower
	// helper to track voting power
	addVote := func(pubKey []byte, r, rh uint64) {
		if v, err := b.ValidatorSet.GetValidator(pubKey); err == nil {
			rounds[r] += v.VotingPower
			rootHeights[rh] += v.VotingPower
		}
	}
	// add self during the RoundInterrupt() phase to determine the best faction to vote on
	// but omit self during the PacemakerPhase() in order to determine if a +2/3 majority may be reached if self joins the best faction
	isPrePacemakerPhase := round == b.Round+1
	if isPrePacemakerPhase {
		b.log.Infof("Round %d Latest root chain height detected at: %d", round, b.Controller.RootChainHeight())
		addVote(b.PublicKey, round, b.Controller.RootChainHeight())
	}
	// for each pacemaker vote
	for _, msg := range b.PacemakerMessages {
		addVote(msg.Signature.PublicKey, msg.Qc.Header.Round, msg.Qc.Header.RootHeight)
	}
	// find max round by voting power
	for r, vp := range rounds {
		if r >= round {
			if nextRound == 0 || vp > rounds[nextRound] {
				nextRound = r
				totalVP = vp
			}
		}
	}
	// find max root height by voting power
	for rh, vp := range rootHeights {
		if rootHeight == 0 || vp > rootHeights[rootHeight] {
			rootHeight = rh
		}
	}
	return
}

// AddPacemakerMessage() adds the 'View' message to the list (keyed by public key string)
func (b *BFT) AddPacemakerMessage(msg *Message) (err lib.ErrorI) {
	b.Controller.Lock()
	defer b.Controller.Unlock()
	b.PacemakerMessages[lib.BytesToString(msg.Signature.PublicKey)] = msg
	return
}

// DetermineResetOnRoot() provide coordination information to help prevent network partitions especially for nested chains
func (b *BFT) GetBFTCoordinationMeta() (meta *lib.BFTCoordinationMeta) {
	// next commit time
	nextCommitTime := time.Now().Add(b.WaitTime(Commit, b.Round))
	// initialize the meta
	meta = &lib.BFTCoordinationMeta{LastCommitTime: uint64(nextCommitTime.UnixMicro())}
	// get the root chain block timing
	if lastRCBlkTime, _ := b.Controller.LoadRootBlockTime(); lastRCBlkTime != nil {
		// calculate if the timestamps are within 2 seconds of each other
		estNextRCBlkTime := time.UnixMicro(int64(lastRCBlkTime.EstNextBlockTime))
		b.log.Infof("Next RootBlock expected: %s NextCommitTime: %s",
			estNextRCBlkTime.Format("15:04:05"), nextCommitTime.Format("15:04:05"))
		// if within 2 seconds
		if estNextRCBlkTime.Sub(nextCommitTime).Abs() <= time.Second*2 {
			// calculate the RC height after 'next commit'
			rcHeightAfterNextCommit := lastRCBlkTime.Height + 2
			b.log.Infof("ResetOnRootHeight set to %d", rcHeightAfterNextCommit)
			// set the reset root on height to the next root block
			meta.ResetOnRootHeight = rcHeightAfterNextCommit
		}
	}
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
		b.log.Error(lib.ErrInvalidProposerPubKey(b.ProposerKey).Error())
		return true
	}

	// confirm is expected proposal
	if !bytes.Equal(b.GetBlockHash(), msg.Qc.BlockHash) || !bytes.Equal(b.Results.Hash(), msg.Qc.ResultsHash) {
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
		// defensive: clear byzantine evidence
		b.ByzantineEvidence = &ByzantineEvidence{DSE: DoubleSignEvidences{}}
	}
	b.RefreshRootChainInfo()
	// reset ProposerKey, Proposal, and Sortition data
	b.ProposerKey = nil
	b.Block, b.BlockHash, b.Results = nil, nil, nil
	b.SortitionData = nil
	b.VDFCache = []*Message{}
}

// RefreshRootChainInfo() updates the cached root chain info with the latest known
func (b *BFT) RefreshRootChainInfo(rootHeight ...uint64) {
	var err lib.ErrorI
	// update heights
	b.Height = b.Controller.ChainHeight()
	if len(rootHeight) != 1 {
		b.RootHeight = b.Controller.RootChainHeight()
	} else {
		b.RootHeight = rootHeight[0]
	}
	// update the validator set
	b.ValidatorSet, err = b.Controller.LoadCommittee(b.LoadRootChainId(b.Height), b.RootHeight)
	if err != nil {
		b.log.Errorf("LoadCommittee() failed with err: %s", err.Error())
	}
	// update the committee data
	b.CommitteeData, err = b.Controller.LoadCommitteeData()
	if err != nil {
		b.log.Errorf("LoadCommitteeData() failed with err: %s", err.Error())
	}
}

// NewHeight() initializes / resets consensus variables preparing for the NewHeight
func (b *BFT) NewHeight(keepLocks ...bool) {
	// reset VotesForHeight
	b.Votes = make(VotesForHeight)
	// reset ProposalsForHeight
	b.ProposalsResetForNewCommittee()
	// reset PacemakerMessages
	b.PacemakerMessages = make(PacemakerMessages)
	// initialize Round 0
	b.NewRound(true)
	// set phase to Election
	b.Phase = Election
	// if resetting due to new Canopy Block and Validator Set then KeepLocks
	// - protecting any who may have committed against attacks like malicious proposers from withholding
	// COMMIT_MSG and sending it after the next block is produces
	if keepLocks == nil || !keepLocks[0] {
		// fully reset the proposals
		b.Proposals = make(ProposalsForHeight)
		// reset PartialQCs
		b.PartialQCs = make(PartialQCs)
		b.HighQC = nil
		b.RCBuildHeight = 0
		b.RootBuildHeight = 0
	}
}

// SafeNode is the codified Hotstuff SafeNodePredicate:
// - Protects replicas who may have committed to a previous value by locking on that value when signing a Precommit Message
// - May unlock if new proposer:
//   - SAFETY: uses the same value the replica is locked on (safe because it will match the value that may have been committed by others)
//   - LIVENESS: uses a lock with a higher round (safe because replica is convinced no other replica committed to their locked value as +2/3rds locked on a higher round)
func (b *BFT) SafeNode(msg *Message) lib.ErrorI {
	if msg == nil || msg.Qc == nil || msg.HighQc == nil {
		return ErrNoSafeNodeJustification()
	}
	// ensure the messages' HighQC justifies its proposal (should have the same hashes)
	if !bytes.Equal(b.BlockToHash(msg.Qc.Block), msg.HighQc.BlockHash) && !bytes.Equal(msg.Qc.Results.Hash(), msg.HighQc.ResultsHash) {
		return ErrMismatchedProposals()
	}
	// if the hashes of the Locked proposal is the same as the Leader's message
	if bytes.Equal(b.HighQC.BlockHash, msg.HighQc.BlockHash) && bytes.Equal(b.HighQC.ResultsHash, msg.HighQc.ResultsHash) {
		b.log.Infof("Proposal %s satisfied the safe node predicate with SAFETY", lib.BytesToTruncatedString(b.HighQC.BlockHash))
		return nil // SAFETY (SAME PROPOSAL AS LOCKED)
	}
	// if the view of the Locked proposal is older than the Leader's message
	if msg.HighQc.Header.Round > b.HighQC.Header.Round {
		b.log.Infof("Proposal %s satisfied the safe node predicate with LIVENESS", lib.BytesToTruncatedString(b.HighQC.BlockHash))
		return nil // LIVENESS (HIGHER ROUND v COMMITTEE THAN LOCKED)
	}
	return ErrFailedSafeNodePredicate()
}

// SetTimerForNextPhase() calculates the wait time for a specific phase/Round, resets the Phase wait timer
func (b *BFT) SetTimerForNextPhase(processTime time.Duration) {
	waitTime := b.WaitTime(b.Phase, b.Round)
	switch b.Phase {
	default:
		b.Phase++
	case CommitProcess:
		return // don't set a timer
	case Pacemaker:
		b.Phase = Election
	}
	b.SetWaitTimers(waitTime, processTime)
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
		// arbitrarily sleep for 1 minute -- the BFT should be reset by an inbound block
		waitTime = b.waitTime(60000, round)
	case RoundInterrupt:
		// don't pass again through 'wait time' as it's already calculated at the msLeftInRound()
		waitTime = time.Duration(b.Config.RoundInterruptTimeoutMS) * time.Millisecond
	case Pacemaker:
		waitTime = 0
	}
	return
}

// waitTime() calculates the waiting time for a specific sleepTime configuration and Round number (helper)
func (b *BFT) waitTime(sleepTimeMS int, round uint64) time.Duration {
	extraTimeMs := int(math.Pow(float64(2), float64(round))) * 1000 // ms
	sleepTimeMS += extraTimeMs
	return time.Duration(sleepTimeMS) * time.Millisecond
}

// NewHeightWaitTime() calculates the waiting time between last commit and election
func (b *BFT) NewHeightWaitTime() (waitTime time.Duration) {
	return time.Duration(b.Config.NewHeightTimeoutMs) * time.Millisecond
}

// msLeftInRound() calculates the milliseconds left in the round
func (b *BFT) msLeftInRound() int {
	// calculate the ms for each phase
	electionMs := b.WaitTime(Election, b.Round).Milliseconds()
	electionVoteMs := b.WaitTime(ElectionVote, b.Round).Milliseconds()
	proposeMs := b.WaitTime(Propose, b.Round).Milliseconds()
	proposeVoteMs := b.WaitTime(ProposeVote, b.Round).Milliseconds()
	precommitMs := b.WaitTime(Precommit, b.Round).Milliseconds()
	precommitVoteMs := b.WaitTime(PrecommitVote, b.Round).Milliseconds()
	commitMs := b.WaitTime(Commit, b.Round).Milliseconds()
	// ms left in round = RoundWaitTime - TimeSpentInRound
	switch b.Phase {
	case Election:
		return int(electionMs + electionVoteMs + proposeMs + proposeVoteMs + precommitMs + precommitVoteMs + commitMs)
	case ElectionVote:
		return int(electionVoteMs + proposeMs + proposeVoteMs + precommitMs + precommitVoteMs + commitMs)
	case Propose:
		return int(proposeMs + proposeVoteMs + precommitMs + precommitVoteMs + commitMs)
	case ProposeVote:
		return int(proposeVoteMs + precommitMs + precommitVoteMs + commitMs)
	case Precommit:
		return int(precommitMs + precommitVoteMs + commitMs)
	case PrecommitVote:
		return int(precommitVoteMs + commitMs)
	case Commit:
		return int(commitMs)
	default:
		return 0
	}
}

// SetWaitTimers() sets the phase wait timer
// - Phase Timeout ensures the node waits for a configured duration (Round x phaseTimeout) to allow for full voter participation
// This design balances synchronization speed during adverse conditions with maximizing voter participation under normal conditions
func (b *BFT) SetWaitTimers(phaseWaitTime, processTime time.Duration) {
	b.log.Debugf("Process time: %.2fs, Wait time: %.2fs", processTime.Seconds(), phaseWaitTime.Seconds())
	subtract := func(wt, pt time.Duration) (t time.Duration) {
		if pt > 24*time.Hour || pt < 0 {
			return wt
		}
		if wt <= pt {
			return 0
		}
		return wt - pt
	}
	// calculate the phase timer by subtracting the process time
	phaseWaitTime = subtract(phaseWaitTime, processTime)
	b.log.Debugf("Setting consensus timer: %.2f sec", phaseWaitTime.Seconds())
	// set Phase timers to go off in their respective timeouts
	lib.ResetTimer(b.PhaseTimer, phaseWaitTime)
}

// SelfIsPropose() returns true if this node is the Leader
func (b *BFT) SelfIsProposer() bool { return b.IsProposer(b.PublicKey) }

// IsProposer() returns true if specific public key is the expected Leader public key
func (b *BFT) IsProposer(id []byte) bool { return bytes.Equal(id, b.ProposerKey) }

// SelfIsValidator() returns true if this node is part of the ValSet
func (b *BFT) SelfIsValidator() bool {
	selfValidator, _ := b.ValidatorSet.GetValidator(b.PublicKey)
	// if not a validator
	if selfValidator == nil {
		// defensively check to make sure there wasn't a race between
		// NEW_HEIGHT and NEW_COMMITTEE, worst case is extra consensus
		// participation
		b.RefreshRootChainInfo()
		// double check
		selfValidator, _ = b.ValidatorSet.GetValidator(b.PublicKey)
	}
	// return 'self is validator'
	return selfValidator != nil
}

// RunVDF() runs the verifiable delay service
func (b *BFT) RunVDF(seed []byte) (err lib.ErrorI) {
	if !b.Config.RunVDF {
		b.log.Infof("VDF disabled")
		return
	}
	// if the vdf seed is nil
	if seed == nil {
		// get the vdf seed from disk
		seed, err = b.VDFSeed()
		if err != nil {
			return
		}
	}
	// run the VDF generation
	go b.VDFService.Run(seed)
	return
}

// VDFSeed() generates the seed for the verifiable delay service
func (b *BFT) VDFSeed() ([]byte, lib.ErrorI) {
	lastQuorumCertificate, err := b.LoadCertificate(b.Height - 1)
	if err != nil {
		return nil, err
	}
	if lastQuorumCertificate == nil {
		return nil, lib.ErrEmptyQuorumCertificate()
	}
	return lastQuorumCertificate.BlockHash, nil
}

// VerifyVDF() validates the VDF from a Replica
func (b *BFT) VerifyVDF(vote *Message) (bool, lib.ErrorI) {
	seed, err := b.VDFSeed()
	if err != nil {
		return false, err
	}
	return b.VDFService.VerifyVDF(seed, vote.Vdf), nil
}

// GetBlockHash() retrieves the hash from the block
func (b *BFT) GetBlockHash() (hash []byte) {
	if b.BlockHash == nil {
		b.BlockHash = b.BlockToHash(b.Block)
	}
	return b.BlockHash
}

// BlockToHash() converts block bytes into a hash
func (b *BFT) BlockToHash(blk []byte) (hash []byte) {
	block := new(lib.Block)
	hash, err := block.BytesToBlockHash(blk)
	if err != nil {
		b.log.Errorf("bft.BlockToHash failed: %s", err.Error())
	}
	return
}

// selectHighestVDF validates the previously sent VDFs by replicas and selects the highest valid one
func (b *BFT) selectHighestVDF() (*crypto.VDF, lib.ErrorI) {
	// clean cache upon exit
	defer func() {
		b.VDFCache = []*Message{}
	}()
	// sort the cache by the number of VDF iterations from highest to lowest
	slices.SortFunc(b.VDFCache, func(a, b *Message) int {
		return cmp.Compare(b.Vdf.Iterations, a.Vdf.Iterations)
	})
	for _, vote := range b.VDFCache {
		// validate VDF
		ok, err := b.VerifyVDF(vote)
		if err != nil {
			// b.log.Warnf("failed to Verify VDF by replica %s: %s", lib.BytesToTruncatedString(vote.Signature.PublicKey), err.Error())
			continue
		}
		if !ok {
			continue
		}
		b.log.Infof("Replica %s submitted a highVDF", lib.BytesToTruncatedString(vote.Signature.PublicKey))
		// return the highest valid VDF
		return vote.Vdf, nil
	}
	// exit, no valid VDF found
	return nil, nil
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
	// that manage interactions with BFT and other parts of the application like FSM, P2P and Storage
	Controller interface {
		Lock()
		Unlock()
		// ChainHeight returns the height of the target-chain
		ChainHeight() uint64
		// RootChainHeight returns the height of the root-chain
		RootChainHeight() uint64
		// ProduceProposal() as a Leader, create a Proposal in the form of a block and certificate results
		ProduceProposal(be *ByzantineEvidence, vdf *crypto.VDF) (rcBuildHeight uint64, block []byte, results *lib.CertificateResult, err lib.ErrorI)
		// ValidateCertificate() as a Replica, validates the leader proposal
		ValidateProposal(rcBuildHeight uint64, qc *lib.QuorumCertificate, evidence *ByzantineEvidence) (*lib.BlockResult, lib.ErrorI)
		// LoadCertificate() gets the Quorum Certificate from the chainId-> plugin at a certain height
		LoadCertificate(height uint64) (*lib.QuorumCertificate, lib.ErrorI)
		// CommitCertificate() commits a block to persistence
		CommitCertificate(qc *lib.QuorumCertificate, block *lib.Block, blockResult *lib.BlockResult) (err lib.ErrorI)
		// GossipBlock() is a P2P call to gossip a completed Quorum Certificate with a Proposal
		GossipBlock(certificate *lib.QuorumCertificate, sender []byte, timestamp uint64, bftMeta *lib.BFTCoordinationMeta)
		// SendToSelf() is a P2P call to directly send  a completed Quorum Certificate to self
		SelfSendBlock(qc *lib.QuorumCertificate, timestamp uint64, bftMeta *lib.BFTCoordinationMeta)
		// SendToReplicas() is a P2P call to directly send a Consensus message to all Replicas
		SendToReplicas(replicas lib.ValidatorSet, msg lib.Signable)
		// SendToProposer() is a P2P call to directly send a Consensus message to the Leader
		SendToProposer(msg lib.Signable)
		// LoadRootChainId() returns the unique identifier of the root chain
		LoadRootChainId(height uint64) (rootChainId uint64)
		// IsOwnRoot() returns a boolean if self chain is root
		LoadIsOwnRoot() bool
		// Syncing() returns true if the plugin is currently syncing
		Syncing() *atomic.Bool
		// ResetFSM() resets the finite state machine
		ResetFSM()

		/* root-chain Functionality Below*/

		// SendCertificateResultsTx() is a P2P call that allows a Leader to submit their CertificateResults (reward) transaction
		SendCertificateResultsTx(certificate *lib.QuorumCertificate)
		// LoadCommittee() loads the ValidatorSet operating under ChainId
		LoadCommittee(rootChainId, rootHeight uint64) (lib.ValidatorSet, lib.ErrorI)
		// LoadRootBlockTime() loads the latest root block time information
		LoadRootBlockTime() (*lib.BlockTimeInfo, lib.ErrorI)
		// LoadCommitteeHeightInState() loads the committee information from state as updated by the quorum certificates
		LoadCommitteeData() (*lib.CommitteeData, lib.ErrorI)
		// LoadLastProposers() loads the last Canopy committee proposers for sortition data
		LoadLastProposers(rootHeight uint64) (*lib.Proposers, lib.ErrorI)
		// LoadMinimumEvidenceHeight() loads the Canopy enforced minimum height for valid Byzantine Evidence
		LoadMinimumEvidenceHeight(rootChainId, rootHeight uint64) (*uint64, lib.ErrorI)
		// IsValidDoubleSigner() checks to see if the double signer is valid for this specific height
		IsValidDoubleSigner(rootChainId, rootHeight uint64, address []byte) bool
	}
)

type ResetBFT struct {
	IsRootChainUpdate bool
	RootHeight        uint64
	BFTMeta           *lib.BFTCoordinationMeta
}

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

	BlockTimeToVDFTargetCoefficient = .50 // how much the commit process time is reduced for VDF processing
)
