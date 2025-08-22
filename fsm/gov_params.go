package fsm

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
	"strconv"
	"strings"
)

const (
	ParamPrefixCons = "/c/" // store key prefix for Consensus param space
	ParamPrefixVal  = "/v/" // store key prefix for Validators param space
	ParamPrefixFee  = "/f/" // store key prefix for Fee param space
	ParamPrefixGov  = "/g/" // store key prefix for Gov param space

	ParamSpaceCons = "cons" // name of Consensus param space
	ParamSpaceVal  = "val"  // name of Validator param space
	ParamSpaceFee  = "fee"  // name of Fee param space
	ParamSpaceGov  = "gov"  // name of Governance param space

	Delimiter = "/" // standard delimiter for protocol version

	AcceptAllProposals  = GovProposalVoteConfig_ACCEPT_ALL
	ProposalApproveList = GovProposalVoteConfig_APPROVE_LIST
	RejectAllProposals  = GovProposalVoteConfig_REJECT_ALL
)

// ParamSpace is a distinct, isolated category within the overarching Params structure
type ParamSpace interface {
	Check() lib.ErrorI
	SetString(paramName string, value string) lib.ErrorI // SetString() update a string parameter in the structure
	SetUint64(paramName string, value uint64) lib.ErrorI // SetUint64() update a uint64 parameter in the structure
}

// DefaultParams() returns the developer set params
func DefaultParams() *Params {
	return &Params{
		Consensus: &ConsensusParams{
			BlockSize:       uint64(units.MB),
			ProtocolVersion: NewProtocolVersion(0, 1),
			RootChainId:     1,
			Retired:         0,
		},
		Validator: &ValidatorParams{
			UnstakingBlocks:                    2,
			MaxPauseBlocks:                     4380,
			DoubleSignSlashPercentage:          10,
			NonSignSlashPercentage:             1,
			MaxNonSign:                         3,
			NonSignWindow:                      5,
			MaxCommittees:                      15,
			MaxCommitteeSize:                   100,
			EarlyWithdrawalPenalty:             20,
			DelegateUnstakingBlocks:            2,
			MinimumOrderSize:                   1000000000,
			StakePercentForSubsidizedCommittee: 33,
			MaxSlashPerCommittee:               15,
			DelegateRewardPercentage:           10,
			BuyDeadlineBlocks:                  60,
			LockOrderFeeMultiplier:             2,
			MinimumStakeForValidators:          0,
			MinimumStakeForDelegates:           0,
			MaximumDelegatesPerCommittee:       0,
		},
		Fee: &FeeParams{
			SendFee:                 10000,
			StakeFee:                10000,
			EditStakeFee:            10000,
			UnstakeFee:              10000,
			PauseFee:                10000,
			UnpauseFee:              10000,
			ChangeParameterFee:      10000,
			DaoTransferFee:          10000,
			CertificateResultsFee:   0,
			SubsidyFee:              10000,
			CreateOrderFee:          10000,
			EditOrderFee:            10000,
			DeleteOrderFee:          10000,
			DexLimitOrderFee:        0,
			DexLiquidityDepositFee:  0,
			DexLiquidityWithdrawFee: 0,
		},
		Governance: &GovernanceParams{
			DaoRewardPercentage: 5,
		},
	}
}

// Check() validates the Params object
func (x *Params) Check() lib.ErrorI {
	if err := x.Consensus.Check(); err != nil {
		return err
	}
	if err := x.Fee.Check(); err != nil {
		return err
	}
	if err := x.Validator.Check(); err != nil {
		return err
	}
	return x.Governance.Check()
}

// consensus param space

const (
	ParamBlockSize       = "blockSize"       // size of the block - header
	ParamProtocolVersion = "protocolVersion" // current protocol version (upgrade enforcement)
	ParamRetired         = "retired"         // if the chain is marking itself as 'retired' to the root-chain making it forever un-subsidized
	ParamRootChainId     = "rootChainID"     // the chain id of the root chain (source of the validator set)
)

var _ ParamSpace = &ConsensusParams{}

// Check() validates the consensus params
func (x *ConsensusParams) Check() lib.ErrorI {
	if _, err := x.ParseProtocolVersion(); err != nil {
		return err
	}
	return nil
}

// SetUint64() update a uint64 parameter in the structure
func (x *ConsensusParams) SetUint64(paramName string, value uint64) lib.ErrorI {
	switch paramName {
	case ParamBlockSize:
		x.BlockSize = value
	case ParamRetired:
		x.Retired = value
	case ParamRootChainId:
		x.RootChainId = value
	default:
		return ErrUnknownParam()
	}
	return x.Check()
}

// SetString() update a string parameter in the structure
func (x *ConsensusParams) SetString(paramName string, value string) lib.ErrorI {
	switch paramName {
	case ParamProtocolVersion:
		// get new protocol version
		newVersion, err := checkProtocolVersion(value)
		if err != nil {
			return err
		}
		// get old protocol version
		oldVersion, err := checkProtocolVersion(x.ProtocolVersion)
		if err != nil {
			return err
		}
		// ensure new version isn't less than old version
		if newVersion.Version <= oldVersion.Version || newVersion.Height <= oldVersion.Height {
			return ErrInvalidProtocolVersion()
		}
		x.ProtocolVersion = value
	default:
		return ErrUnknownParam()
	}
	return x.Check()
}

// ParseProtocolVersion() validates the format of the Protocol version string and returns the ProtocolVersion object
func (x *ConsensusParams) ParseProtocolVersion() (*ProtocolVersion, lib.ErrorI) {
	return checkProtocolVersion(x.ProtocolVersion)
}

// checkProtocolVersion (helper) validates the format of the Protocol version string and returns the ProtocolVersion object
func checkProtocolVersion(v string) (*ProtocolVersion, lib.ErrorI) {
	ptr := new(ProtocolVersion)
	arr := strings.Split(v, Delimiter)
	if len(arr) != 2 {
		return nil, ErrInvalidProtocolVersion()
	}
	version, err := strconv.Atoi(arr[0])
	if err != nil {
		return nil, ErrInvalidProtocolVersion()
	}
	height, err := strconv.Atoi(arr[1])
	if err != nil {
		return nil, ErrInvalidProtocolVersion()
	}
	ptr.Height, ptr.Version = uint64(height), uint64(version)
	return ptr, nil
}

// NewProtocolVersion() creates a properly formatted protocol version string
func NewProtocolVersion(height uint64, version uint64) string {
	return fmt.Sprintf("%d%s%d", version, Delimiter, height)
}

// FormatParamSpace() converts a user inputted ParamSpace string into the proper ParamSpace name
func FormatParamSpace(paramSpace string) string {
	paramSpace = strings.ToLower(paramSpace)
	switch {
	case strings.Contains(paramSpace, "con"):
		return ParamSpaceCons
	case strings.Contains(paramSpace, "gov"):
		return ParamSpaceGov
	case strings.Contains(paramSpace, "fee"):
		return ParamSpaceFee
	case strings.Contains(paramSpace, "val"):
		return ParamSpaceVal
	}
	return paramSpace
}

// validator param space

var _ ParamSpace = &ValidatorParams{}

const (
	ParamUnstakingBlocks                    = "unstakingBlocks"                    // number of blocks a committee member must be 'unstaking' for
	ParamMaxPauseBlocks                     = "maxPauseBlocks"                     // maximum blocks a validator may be paused for before force-unstaking
	ParamNonSignSlashPercentage             = "nonSignSlashPercentage"             // how much a non-signer is slashed if exceeds threshold in window (% of stake)
	ParamMaxNonSign                         = "maxNonSign"                         // how much a committee member can not sign before being slashed
	ParamNonSignWindow                      = "nonSignWindow"                      // how frequently the non-sign-count is reset
	ParamDoubleSignSlashPercentage          = "doubleSignSlashPercentage"          // how much a double signer is slashed (% of stake)
	ParamMaxCommittees                      = "maxCommittees"                      // maximum number of committees a single validator may participate in
	ParamMaxCommitteeSize                   = "maxCommitteeSize"                   // maximum number of members a committee may have
	ParamEarlyWithdrawalPenalty             = "earlyWithdrawalPenalty"             // reduction percentage of non-compounded rewards
	ParamDelegateUnstakingBlocks            = "delegateUnstakingBlocks"            // number of blocks a delegator must be 'unstaking' for
	ParamMinimumOrderSize                   = "minimumOrderSize"                   // minimum sell tokens in a sell order
	ParamStakePercentForSubsidizedCommittee = "stakePercentForSubsidizedCommittee" // the minimum percentage of total stake needed to be a 'paid committee'
	ParamMaxSlashPerCommittee               = "maxSlashPerCommittee"               // the maximum validator slash per committee per block
	ParamDelegateRewardPercentage           = "delegateRewardPercentage"           // the percentage of the block reward that is awarded to the delegates
	ParamBuyDeadlineBlocks                  = "buyDeadlineBlocks"                  // the amount of blocks a 'buyer' has to complete an order they reserved
	ParamLockOrderFeeMultiplier             = "lockOrderFeeMultiplier"             // the fee multiplier of the 'send' fee that is required to execute a lock order
	ParamMinimumStakeForValidators          = "minimumStakeForValidators"          // minimum stake required to be a validator
	ParamMinimumStakeForDelegates           = "minimumStakeForDelegates"           // minimum stake required to be a delegate
	ParamMaximumDelegatesPerCommittee       = "maximumDelegatesPerCommittee"       // maximum number of delegates per committee
)

// Check() validates the Validator params
func (x *ValidatorParams) Check() lib.ErrorI {
	if x.UnstakingBlocks == 0 {
		return ErrInvalidParam(ParamUnstakingBlocks)
	}
	if x.MaxPauseBlocks == 0 {
		return ErrInvalidParam(ParamMaxPauseBlocks)
	}
	if x.NonSignSlashPercentage > 100 {
		return ErrInvalidParam(ParamNonSignSlashPercentage)
	}
	if x.NonSignWindow == 0 {
		return ErrInvalidParam(ParamNonSignWindow)
	}
	if x.MaxNonSign > x.NonSignWindow {
		return ErrInvalidParam(ParamMaxNonSign)
	}
	if x.DoubleSignSlashPercentage > 100 {
		return ErrInvalidParam(ParamDoubleSignSlashPercentage)
	}
	if x.MaxCommittees > 100 {
		return ErrInvalidParam(ParamMaxCommittees)
	}
	if x.MaxCommitteeSize == 0 {
		return ErrInvalidParam(ParamMaxCommitteeSize)
	}
	if x.DelegateUnstakingBlocks < 2 {
		return ErrInvalidParam(ParamDelegateUnstakingBlocks)
	}
	if x.EarlyWithdrawalPenalty > 100 {
		return ErrInvalidParam(ParamEarlyWithdrawalPenalty)
	}
	if x.StakePercentForSubsidizedCommittee == 0 || x.StakePercentForSubsidizedCommittee > 100 {
		return ErrInvalidParam(ParamStakePercentForSubsidizedCommittee)
	}
	if x.MaxSlashPerCommittee == 0 || x.MaxSlashPerCommittee > 100 {
		return ErrInvalidParam(ParamMaxSlashPerCommittee)
	}
	if x.DelegateRewardPercentage > 100 {
		return ErrInvalidParam(ParamDelegateRewardPercentage)
	}
	if x.BuyDeadlineBlocks == 0 {
		return ErrInvalidParam(ParamBuyDeadlineBlocks)
	}
	if x.LockOrderFeeMultiplier == 0 {
		return ErrInvalidParam(ParamLockOrderFeeMultiplier)
	}
	return nil
}

// SetUint64() update a uint64 parameter in the structure
func (x *ValidatorParams) SetUint64(paramName string, value uint64) lib.ErrorI {
	switch paramName {
	case ParamUnstakingBlocks:
		x.UnstakingBlocks = value
	case ParamMaxPauseBlocks:
		x.MaxPauseBlocks = value
	case ParamNonSignWindow:
		x.NonSignWindow = value
	case ParamMaxNonSign:
		x.MaxNonSign = value
	case ParamNonSignSlashPercentage:
		x.NonSignSlashPercentage = value
	case ParamDoubleSignSlashPercentage:
		x.DoubleSignSlashPercentage = value
	case ParamMaxCommittees:
		x.MaxCommittees = value
	case ParamMaxCommitteeSize:
		x.MaxCommitteeSize = value
	case ParamEarlyWithdrawalPenalty:
		x.EarlyWithdrawalPenalty = value
	case ParamDelegateUnstakingBlocks:
		x.DelegateUnstakingBlocks = value
	case ParamMinimumOrderSize:
		x.MinimumOrderSize = value
	case ParamStakePercentForSubsidizedCommittee:
		x.StakePercentForSubsidizedCommittee = value
	case ParamMaxSlashPerCommittee:
		x.MaxSlashPerCommittee = value
	case ParamDelegateRewardPercentage:
		x.DelegateRewardPercentage = value
	case ParamBuyDeadlineBlocks:
		x.BuyDeadlineBlocks = value
	case ParamLockOrderFeeMultiplier:
		x.LockOrderFeeMultiplier = value
	case ParamMinimumStakeForValidators:
		x.MinimumStakeForValidators = value
	case ParamMinimumStakeForDelegates:
		x.MinimumStakeForDelegates = value
	case ParamMaximumDelegatesPerCommittee:
		x.MaximumDelegatesPerCommittee = value
	default:
		return ErrUnknownParam()
	}
	return x.Check()
}

// SetString() update a string parameter in the structure
func (x *ValidatorParams) SetString(_ string, _ string) lib.ErrorI {
	return ErrUnknownParam()
}

// fee param space

var _ ParamSpace = &FeeParams{}

const (
	ParamSendFee                 = "sendFee"                 // transaction fee for MessageSend
	ParamStakeFee                = "stakeFee"                // transaction fee for MessageStake
	ParamEditStakeFee            = "editStakeFee"            // transaction fee for MessageEditStake
	ParamUnstakeFee              = "unstakeFee"              // transaction fee for MessageUnstake
	ParamPauseFee                = "pauseFee"                // transaction fee for MessagePause
	ParamUnpauseFee              = "unpauseFee"              // transaction fee for MessageUnpause
	ParamChangeParameterFee      = "changeParameterFee"      // transaction fee for MessageChangeParameter
	ParamDAOTransferFee          = "daoTransferFee"          // transaction fee for MessageDAOTransfer
	ParamCertificateResultsFee   = "certificateResultsFee"   // transaction fee for MessageCertificateResults
	ParamSubsidyFee              = "subsidyFee"              // transaction fee for MessageSubsidy
	ParamCreateOrderFee          = "createOrderFee"          // transaction fee for MessageCreateOrder
	ParamEditOrderFee            = "editOrderFee"            // transaction fee for MessageEditOrder
	ParamDeleteOrderFee          = "deleteOrderFee"          // transaction fee for MessageDeleteOrder
	ParamDexLimitOrderFee        = "dexLimitOrderFee"        // transaction fee for MessageDexLimitOrder
	ParamDexLiquidityDepositFee  = "dexLiquidityDepositFee"  // transaction fee for MessageDexLiquidityDeposit
	ParamDexLiquidityWithdrawFee = "dexLiquidityWithdrawFee" // transaction fee for MessageDexLiquidityWithdraw
)

// Check() validates the Fee params
func (x *FeeParams) Check() lib.ErrorI {
	return nil
}

// SetString() update a string parameter in the structure
func (x *FeeParams) SetString(_ string, _ string) lib.ErrorI {
	return ErrUnknownParam()
}

// SetUint64() update a uint64 parameter in the structure
func (x *FeeParams) SetUint64(paramName string, value uint64) lib.ErrorI {
	switch paramName {
	case ParamSendFee:
		x.SendFee = value
	case ParamStakeFee:
		x.StakeFee = value
	case ParamEditStakeFee:
		x.EditStakeFee = value
	case ParamUnstakeFee:
		x.UnstakeFee = value
	case ParamPauseFee:
		x.PauseFee = value
	case ParamUnpauseFee:
		x.UnpauseFee = value
	case ParamChangeParameterFee:
		x.ChangeParameterFee = value
	case ParamDAOTransferFee:
		x.DaoTransferFee = value
	case ParamCertificateResultsFee:
		x.CertificateResultsFee = value
	case ParamSubsidyFee:
		x.SubsidyFee = value
	case ParamCreateOrderFee:
		x.CreateOrderFee = value
	case ParamDeleteOrderFee:
		x.DeleteOrderFee = value
	case ParamEditOrderFee:
		x.EditOrderFee = value
	case ParamDexLimitOrderFee:
		x.DexLimitOrderFee = value
	case ParamDexLiquidityDepositFee:
		x.DexLiquidityDepositFee = value
	case ParamDexLiquidityWithdrawFee:
		x.DexLiquidityWithdrawFee = value
	default:
		return ErrUnknownParam()
	}
	return x.Check()
}

// governance param space

const (
	ParamDAORewardPercentage = "daoRewardPercentage" // percent of rewards the DAO fund receives
)

var _ ParamSpace = &GovernanceParams{}

// Check() validates the Governance params
func (x *GovernanceParams) Check() lib.ErrorI {
	if x.DaoRewardPercentage > 100 {
		return ErrInvalidParam(ParamDAORewardPercentage)
	}
	return nil
}

// SetUint64() update a uint64 parameter in the structure
func (x *GovernanceParams) SetUint64(paramName string, value uint64) lib.ErrorI {
	switch paramName {
	case ParamDAORewardPercentage:
		x.DaoRewardPercentage = value
	default:
		return ErrUnknownParam()
	}
	return x.Check()
}

// SetString() update a string parameter in the structure
func (x *GovernanceParams) SetString(_ string, _ string) lib.ErrorI {
	return ErrUnknownParam()
}

// prefixForParamSpace() converts the ParamSpace name into the ParamSpace prefix
func prefixForParamSpace(space string) string {
	switch space {
	case ParamSpaceCons:
		return ParamPrefixCons
	case ParamSpaceVal:
		return ParamPrefixVal
	case ParamSpaceFee:
		return ParamPrefixFee
	case ParamSpaceGov:
		return ParamPrefixGov
	default:
		panic("unknown param space")
	}
}

// POLLING CODE BELOW

/*
	Chain Polling Feature:
	- Canopy straw polling
	- Internal chain straw polling
	- External chain straw polling

	On-Chain
	1. Memo field signal 'START POLL' {Hash, Opt:URL, Start, End}
	2. Memo field signal 'VOTE' {Hash, Y/N}

	Off-Chain
	1. 'poll.json' maintains status of poll at each height
	2. 'poll.json' maintains historical poll stats for 1000 blocks
	3. Query the state of addresses & validators each height to update polling power
	4. '/v1/gov/poll/' shows what's in 'poll.json' + voting power
*/

const (
	minPollEmbedSize     = 80    // minPollEmbedSize is (below) the minimum length string a memo must be to be a poll
	prunePollAfterBlocks = 40320 // prunePollAfterBlocks is the amount of blocks a poll is maintained before being pruned
	maxPollLengthBlocks  = 10000 // maxPollLengthBlocks is the maximum length a poll may run in blocks
)

// ActivePolls is the in-memory representation of the polls.json file
// Contains a list of all active polls
type ActivePolls struct {
	Polls    map[string]map[string]bool `json:"activePolls"` // [poll_hash] -> [address hex] -> Vote
	PollMeta map[string]*StartPoll      `json:"pollMeta"`    // [poll_hash] -> StartPoll structure
}

// CheckForPollTransaction() populates the poll.json file from embeds if the embed exists in the memo field
func (p *ActivePolls) CheckForPollTransaction(sender crypto.AddressI, memo string, height uint64) lib.ErrorI {
	if len(memo) < minPollEmbedSize {
		return nil
	}
	// check for start poll embed
	if startPoll, err := checkMemoForStartPoll(height, memo); err == nil {
		p.NewPoll(startPoll)
		return nil
	}
	// check for vote poll embed
	if votePoll, err := checkMemoForVotePoll(memo); err == nil {
		p.VotePoll(sender, votePoll, height)
		return nil
	}
	// no embed
	return nil
}

// NewPoll() creates a new poll from the start poll embed
func (p *ActivePolls) NewPoll(startPoll *StartPoll) {
	if _, exists := p.Polls[startPoll.StartPoll]; exists {
		return
	}
	p.Polls[startPoll.StartPoll] = make(map[string]bool)
	p.PollMeta[startPoll.StartPoll] = startPoll
}

// VotePoll() upserts a vote to a specific poll
func (p *ActivePolls) VotePoll(sender crypto.AddressI, votePoll *VotePoll, height uint64) {
	poll, exists := p.Polls[votePoll.VotePoll]
	if !exists {
		return
	}
	if height > p.PollMeta[votePoll.VotePoll].EndHeight {
		return
	}
	poll[sender.String()] = votePoll.Approve
	p.Polls[votePoll.VotePoll] = poll
}

// Cleanup() adds polls to 'closed' section if past end height and removes any polls that are older than 4 weeks
func (p *ActivePolls) Cleanup(height uint64) {
	// close any polls that are past the end_height
	for hash, poll := range p.PollMeta {
		if int(height-poll.EndHeight) >= prunePollAfterBlocks {
			delete(p.PollMeta, hash)
			delete(p.Polls, hash) // defensive
		}
	}
}

// NewFromFile() creates a new polls object from a file
func (p *ActivePolls) NewFromFile(dataDirPath string) lib.ErrorI {
	return lib.NewJSONFromFile(p, dataDirPath, lib.PollsFilePath)
}

// SaveToFile() persists the polls object to a json file
func (p *ActivePolls) SaveToFile(dataDirPath string) lib.ErrorI {
	return lib.SaveJSONToFile(p, dataDirPath, lib.PollsFilePath)
}

// StartPoll represents the structure for initiating a new poll
// It is used to encode data into JSON format for storing in memo fields
type StartPoll struct {
	StartPoll string `json:"startPoll"`
	Url       string `json:"url,omitempty"`
	EndHeight uint64 `json:"endHeight"`
}

// NewStartPollTransaction() isn't an actual transaction type - rather it's a protocol built on top of send transactions to allow simple straw polling on Canopy.
// This model is plugin specific and does not need to be followed for other chains.
func NewStartPollTransaction(from crypto.PrivateKeyI, pollJSON json.RawMessage, networkId, chainId, fee, height uint64) (lib.TransactionI, lib.ErrorI) {
	// extract the params from the pollJSON
	extract := struct {
		URL      string `json:"URL"`      // optional
		EndBlock uint64 `json:"endBlock"` // required
	}{}
	if err := lib.UnmarshalJSON(pollJSON, &extract); err != nil {
		return nil, err
	}
	// encode the structure to the memo
	memoBytes, err := lib.MarshalJSON(StartPoll{
		StartPoll: crypto.HashString(pollJSON),
		Url:       extract.URL,
		EndHeight: extract.EndBlock,
	})
	if err != nil {
		return nil, err
	}
	// return the transaction object
	return NewSendTransaction(from, from.PublicKey().Address(), 1, networkId, chainId, fee, height, string(memoBytes))
}

// VotePoll represents the structure of a voting action on a poll
// It is used to encode data into JSON format for storing in memo fields
type VotePoll struct {
	VotePoll string `json:"votePoll"`
	Approve  bool   `json:"approve"`
}

// NewVotePollTransaction() isn't an actual transaction type - rather it's a protocol built on top of send transactions to allow simple straw polling on Canopy.
// This model is plugin specific and does not need to be followed for other chains.
func NewVotePollTransaction(from crypto.PrivateKeyI, pollJSON json.RawMessage, approve bool, networkId, chainId, fee, height uint64) (lib.TransactionI, lib.ErrorI) {
	// encode the structure to the memo
	memoBytes, err := lib.MarshalJSON(VotePoll{
		VotePoll: crypto.HashString(pollJSON),
		Approve:  approve,
	})
	if err != nil {
		return nil, err
	}
	// return the transaction object
	return NewSendTransaction(from, from.PublicKey().Address(), 1, networkId, chainId, fee, height, string(memoBytes))
}

// validatePollHash() ensures a poll hash is valid for a poll transaction
func validatePollHash(pollHash string) lib.ErrorI {
	hash, err := lib.StringToBytes(pollHash)
	if err != nil {
		return err
	}
	if len(hash) != crypto.HashSize {
		return lib.ErrHashSize()
	}
	return nil
}

// checkMemoForStartPoll() checks a memo for a start poll embed
func checkMemoForStartPoll(height uint64, memo string) (start *StartPoll, err lib.ErrorI) {
	start = new(StartPoll)
	if err = lib.UnmarshalJSON([]byte(memo), start); err != nil {
		return
	}
	if err = validatePollHash(start.StartPoll); err != nil {
		return
	}
	heightDiff := start.EndHeight - height
	if heightDiff > maxPollLengthBlocks || heightDiff <= 0 {
		return nil, ErrInvalidStartPollHeight()
	}
	return
}

// checkMemoForVotePoll() checks a memo for a vote poll embed
func checkMemoForVotePoll(memo string) (vote *VotePoll, err lib.ErrorI) {
	vote = new(VotePoll)
	if err = lib.UnmarshalJSON([]byte(memo), vote); err != nil {
		return
	}
	if err = validatePollHash(vote.VotePoll); err != nil {
		return
	}
	return
}

// Poll is a list of PollResults keyed by the hash of the proposal
type Poll map[string]PollResult

// PollResult is a structure that represents the current state of a 'Poll' for a 'Proposal'
type PollResult struct {
	ProposalHash string    `json:"proposalHash"` // the hash of the proposal
	ProposalURL  string    `json:"proposalURL"`  // the url of the proposal
	Accounts     VoteStats `json:"accounts"`     // vote statistics for accounts
	Validators   VoteStats `json:"validators"`   // vote statistics for validators
}

// VoteStats: are demonstrative statistics about poll voting
type VoteStats struct {
	ApproveTokens     uint64 `json:"approveTokens"`    // power (tokens) that voted 'yay'
	RejectTokens      uint64 `json:"rejectTokens"`     // power (tokens)  that voted 'nay'
	TotalVotedTokens  uint64 `json:"totalVotedTokens"` // total power (tokens) that already voted
	TotalTokens       uint64 `json:"totalTokens"`      // total power (tokens)  that could possibly vote
	ApprovePercentage uint64 `json:"approvedPercent"`  // percent representation of power (tokens) that voted 'yay' out of total power
	RejectPercentage  uint64 `json:"rejectPercent"`    // percent representation of power (tokens) that voted 'nay' out of total power
	VotedPercentage   uint64 `json:"votedPercent"`     // percent representation of power (tokens) voted out of total power
}

// PROPOSAL CODE BELOW

// GovProposal is an interface that all proposals that may be polled for and voted on must conform to
type GovProposal interface {
	proto.Message
	GetStartHeight() uint64
	GetEndHeight() uint64
	GetProposalHash() string
}

// GovProposalWithVote is a wrapper over a GovProposal but contains an approval / disapproval boolean
type GovProposalWithVote struct {
	Proposal json.RawMessage `json:"proposal"`
	Approve  bool            `json:"approve"`
}

// GovProposals is a list of GovProposalsWithVote keyed by the transaction hash of the underlying proposal transaction
type GovProposals map[string]GovProposalWithVote

// Add() adds a GovProposalWithVote to the list
func (p GovProposals) Add(proposalTransaction json.RawMessage, approve bool) (err lib.ErrorI) {
	// get the transaction hash from the proposal transaction json
	txHash, err := TxHashFromJSON(proposalTransaction)
	// if an error occurred during the extraction
	if err != nil {
		// exit with error
		return
	}
	// add to the proposals list keyed by the transaction hash
	p[txHash] = GovProposalWithVote{proposalTransaction, approve}
	// exit
	return
}

// Del() removes a GovProposalWithVote from the list
func (p GovProposals) Del(proposalTransaction json.RawMessage) (err error) {
	// get the transaction hash from the proposal transaction json
	txHash, err := TxHashFromJSON(proposalTransaction)
	// if an error occurred during the extraction
	if err != nil {
		// exit with error
		return
	}
	// removed from the proposals list keyed by the transaction hash
	delete(p, txHash)
	// exit
	return
}

// NewFromFile() creates a new polls object from a file
func (p *GovProposals) NewFromFile(dataDirPath string) lib.ErrorI {
	return lib.NewJSONFromFile(p, dataDirPath, lib.ProposalsFilePath)
}

// SaveToFile() persists the polls object to a json file
func (p *GovProposals) SaveToFile(dataDirPath string) lib.ErrorI {
	return lib.SaveJSONToFile(p, dataDirPath, lib.ProposalsFilePath)
}
