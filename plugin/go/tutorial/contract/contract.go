package contract

import (
"crypto/sha256"

"google.golang.org/protobuf/types/known/anypb"
)

var (
PREFIX_MARKET_STATE    = []byte{0x10}
PREFIX_POSITION_STATE  = []byte{0x11}
PREFIX_OUTCOME_STATE   = []byte{0x12}
PREFIX_RESOLVER_STATE  = []byte{0x13}
PREFIX_TREASURY        = []byte{0x14}
PREFIX_RESOLVER_RECORD = []byte{0x16}
PREFIX_PROPOSAL_RECORD = []byte{0x17}
PREFIX_DISPUTE_RECORD  = []byte{0x18}
PREFIX_VOTE_COMMIT     = []byte{0x19}
PREFIX_VOTE_REVEAL     = []byte{0x1A}
PREFIX_SLASH_RECORD    = []byte{0x1B}
)

var (
PREFIX_ACCOUNT  = []byte{0x01}
PREFIX_FEE_POOL = []byte{0x02}
)

// Issue-12: assert MIN_B0 > FINALIZATION_BOUNTY at startup.
// If this ever fails, create_market would seed a TreasuryReserve that cannot
// cover the finalization bounty, silently breaking permissionless finalization.
func init() {
if MIN_B0 <= FINALIZATION_BOUNTY {
panic("invariant violated: MIN_B0 must be greater than FINALIZATION_BOUNTY")
}
}

var ContractConfig = &PluginConfig{
Name:    "praxis_prediction_market",
Id:      1,
Version: 1,
SupportedTransactions: []string{
"create_market",
"submit_prediction",
"claim_winnings",
"register_resolver",
"propose_outcome",
"file_dispute",
"commit_vote",
"reveal_vote",
"tally_votes",
"finalize_market",
"claim_slash",
"reclaim_stake",
"forfeit_position",
	"claim_builder_reward",
	"claim_creator_fee",
	"claim_resolver_reward",
		"claim_community_reward",
		"claim_investor_reward",
		"claim_protocol_reward",
		"unstake_resolver",
		"cancel_market",
		"claim_unbonded_stake",
},
TransactionTypeUrls: []string{
	"type.googleapis.com/types.MessageCreateMarket",
	"type.googleapis.com/types.MessageSubmitPrediction",
	"type.googleapis.com/types.MessageClaimWinnings",
	"type.googleapis.com/types.MessageRegisterResolver",
	"type.googleapis.com/types.MessageProposeOutcome",
	"type.googleapis.com/types.MessageFileDispute",
	"type.googleapis.com/types.MessageCommitVote",
	"type.googleapis.com/types.MessageRevealVote",
	"type.googleapis.com/types.MessageTallyVotes",
	"type.googleapis.com/types.MessageFinalizeMarket",
	"type.googleapis.com/types.MessageClaimSlash",
	"type.googleapis.com/types.MessageReclaimStake",
	"type.googleapis.com/types.MessageForfeitPosition",
	"type.googleapis.com/types.MessageClaimBuilderReward",
	"type.googleapis.com/types.MessageClaimCreatorFee",
	"type.googleapis.com/types.MessageClaimResolverReward",
		"type.googleapis.com/types.MessageClaimCommunityReward",
		"type.googleapis.com/types.MessageClaimInvestorReward",
		"type.googleapis.com/types.MessageClaimProtocolReward",
		"type.googleapis.com/types.MessageUnstakeResolver",
		"type.googleapis.com/types.MessageCancelMarket",
		"type.googleapis.com/types.MessageClaimUnbondedStake",
},
}



type Contract struct {
Config    Config
FSMConfig *PluginFSMConfig
plugin    *Plugin
fsmId     uint64
}

func marketKey(prefix, marketId []byte) []byte {
return append(append([]byte{}, prefix...), marketId...)
}
func positionKey(marketId, addr []byte) []byte {
k := append(append([]byte{}, PREFIX_POSITION_STATE...), marketId...)
return append(k, addr...)
}
func addrKey(prefix, addr []byte) []byte {
return append(append([]byte{}, prefix...), addr...)
}

func (c *Contract) Genesis(req *PluginGenesisRequest) *PluginGenesisResponse {
return &PluginGenesisResponse{}
}

func (c *Contract) BeginBlock(req *PluginBeginRequest) *PluginBeginResponse {
SetGlobalHeight(req.Height)

entropyQId := nextQueryId()
entropyResp, readErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: entropyQId, Key: PANEL_ENTROPY_KEY},
},
})

// Issue-17 fix: replace XOR accumulator with SHA256 hash chain.
// XOR with a deterministic height function is fully predictable by chain
// observers — they can compute the accumulator value at any future block and
// time a file_dispute call to influence panel selection.
// SHA256(prev || height_bytes) is a one-way function: knowing the output
// does not allow computing a height that produces a desired panel seed.
var prev [8]byte
if readErr == nil && entropyResp != nil {
for _, r := range entropyResp.Results {
if r.QueryId == entropyQId && len(r.Entries) > 0 && len(r.Entries[0].Value) >= 8 {
copy(prev[:], r.Entries[0].Value[:8])
}
}
}

heightBytes := make([]byte, 8)
for i := 7; i >= 0; i-- {
heightBytes[i] = byte(req.Height & 0xFF)
req.Height >>= 8
}
input := append(prev[:], heightBytes...)
hash := sha256.Sum256(input)
buf := hash[:8]

wr, writeErr := c.plugin.StateWrite(c, &PluginStateWriteRequest{
Sets: []*PluginSetOp{
{Key: PANEL_ENTROPY_KEY, Value: buf},
},
})
if writeErr != nil || (wr != nil && wr.Error != nil) {
_ = wr
}

return &PluginBeginResponse{}
}

func (c *Contract) EndBlock(req *PluginEndRequest) *PluginEndResponse {
	height := GetGlobalHeight()
	if height > 0 && height%PRIS_EPOCH_BLOCKS == 0 {
		_ = c.processEpochBoundary(height)
	}
	return &PluginEndResponse{}
}

func (c *Contract) CheckTx(req *PluginCheckRequest) *PluginCheckResponse {
msg, err := FromAny(req.Tx.Msg)
if err != nil {
return &PluginCheckResponse{Error: err}
}
switch m := msg.(type) {
case *MessageSend:
return c.CheckMessageSend(m)
case *MessageCreateMarket:
return c.CheckMessageCreateMarket(m)
case *MessageSubmitPrediction:
return c.CheckMessageSubmitPrediction(m)
case *MessageClaimWinnings:
return c.CheckMessageClaimWinnings(m)
case *MessageRegisterResolver:
return c.CheckMessageRegisterResolver(m)
case *MessageProposeOutcome:
return c.CheckMessageProposeOutcome(m)
case *MessageFileDispute:
return c.CheckMessageFileDispute(m)
case *MessageCommitVote:
return c.CheckMessageCommitVote(m)
case *MessageRevealVote:
return c.CheckMessageRevealVote(m)
case *MessageTallyVotes:
return c.CheckMessageTallyVotes(m)
case *MessageFinalizeMarket:
return c.CheckMessageFinalizeMarket(m)
case *MessageClaimSlash:
return c.CheckMessageClaimSlash(m)
case *MessageReclaimStake:
return c.CheckMessageReclaimStake(m)
case *MessageForfeitPosition:
return c.CheckMessageForfeitPosition(m)
case *MessageClaimBuilderReward:
return c.CheckMessageClaimBuilderReward(m)
case *MessageClaimCreatorFee:
return c.CheckMessageClaimCreatorFee(m)
case *MessageClaimResolverReward:
return c.CheckMessageClaimResolverReward(m)
case *MessageClaimCommunityReward:
return c.CheckMessageClaimCommunityReward(m)
case *MessageClaimInvestorReward:
return c.CheckMessageClaimInvestorReward(m)
case *MessageClaimProtocolReward:
return c.CheckMessageClaimProtocolReward(m)
case *MessageUnstakeResolver:
return c.CheckMessageUnstakeResolver(m)
case *MessageCancelMarket:
return c.CheckMessageCancelMarket(m)
case *MessageClaimUnbondedStake:
return c.CheckMessageClaimUnbondedStake(m)
default:
return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
}
}

func (c *Contract) DeliverTx(req *PluginDeliverRequest) *PluginDeliverResponse {
msg, err := FromAny(req.Tx.Msg)
if err != nil {
return &PluginDeliverResponse{Error: err}
}
fee := req.Tx.Fee
switch m := msg.(type) {
case *MessageSend:
return c.DeliverMessageSend(m, fee)
case *MessageCreateMarket:
return c.DeliverMessageCreateMarket(m, fee)
case *MessageSubmitPrediction:
return c.DeliverMessageSubmitPrediction(m, fee)
case *MessageClaimWinnings:
return c.DeliverMessageClaimWinnings(m, fee)
case *MessageRegisterResolver:
return c.DeliverMessageRegisterResolver(m, fee)
case *MessageProposeOutcome:
return c.DeliverMessageProposeOutcome(m, fee)
case *MessageFileDispute:
return c.DeliverMessageFileDispute(m, fee)
case *MessageCommitVote:
return c.DeliverMessageCommitVote(m, fee)
case *MessageRevealVote:
return c.DeliverMessageRevealVote(m, fee)
case *MessageTallyVotes:
return c.DeliverMessageTallyVotes(m, fee)
case *MessageFinalizeMarket:
return c.DeliverMessageFinalizeMarket(m, fee)
case *MessageClaimSlash:
return c.DeliverMessageClaimSlash(m, fee)
case *MessageReclaimStake:
return c.DeliverMessageReclaimStake(m, fee)
case *MessageForfeitPosition:
return c.DeliverMessageForfeitPosition(m, fee)
case *MessageClaimBuilderReward:
return c.DeliverMessageClaimBuilderReward(m, fee)
case *MessageClaimCreatorFee:
return c.DeliverMessageClaimCreatorFee(m, fee)
case *MessageClaimResolverReward:
return c.DeliverMessageClaimResolverReward(m, fee)
case *MessageClaimCommunityReward:
return c.DeliverMessageClaimCommunityReward(m, fee)
case *MessageClaimInvestorReward:
return c.DeliverMessageClaimInvestorReward(m, fee)
case *MessageClaimProtocolReward:
return c.DeliverMessageClaimProtocolReward(m, fee)
case *MessageUnstakeResolver:
return c.DeliverMessageUnstakeResolver(m, fee)
case *MessageCancelMarket:
return c.DeliverMessageCancelMarket(m, fee)
case *MessageClaimUnbondedStake:
return c.DeliverMessageClaimUnbondedStake(m, fee)
default:
return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
}
}

var _ = (*anypb.Any)(nil)
