package contract

import (
"math/rand"

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

entropyQId := rand.Uint64()
entropyResp, readErr := c.plugin.StateRead(c, &PluginStateReadRequest{
Keys: []*PluginKeyRead{
{QueryId: entropyQId, Key: PANEL_ENTROPY_KEY},
},
})

var acc uint64
if readErr == nil && entropyResp != nil {
for _, r := range entropyResp.Results {
if r.QueryId == entropyQId && len(r.Entries) > 0 {
if len(r.Entries[0].Value) >= 8 {
for i := 0; i < 8; i++ {
acc = (acc << 8) | uint64(r.Entries[0].Value[i])
}
}
}
}
}

acc ^= req.Height * 0x9e3779b97f4a7c15

buf := make([]byte, 8)
for i := 7; i >= 0; i-- {
buf[i] = byte(acc & 0xFF)
acc >>= 8
}

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
default:
return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
}
}

var _ = (*anypb.Any)(nil)
