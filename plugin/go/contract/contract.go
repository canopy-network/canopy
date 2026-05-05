package contract

import (
"google.golang.org/protobuf/proto"
"google.golang.org/protobuf/reflect/protodesc"
"google.golang.org/protobuf/reflect/protoreflect"
"google.golang.org/protobuf/types/known/anypb"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — Contract Wiring
// Spec authority: ADLMSR v5.6.6-r2-CORRECTED + PORS v1.0-r2-CORRECTED
//
// This file contains:
//   - ContractConfig  (FSM handshake — tx type registration)
//   - Contract struct (application context)
//   - init()          (file descriptor registration + PANEL_ENTROPY_KEY init)
//   - Lifecycle       (Genesis, BeginBlock, EndBlock)
//   - CheckTx router  (stateless validation — zero StateRead calls)
//   - DeliverTx router(stateful execution)
//
// Never modify plugin.go, main.go, or plugin.proto.
// ═══════════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// CONTRACT CONFIG
// SupportedTransactions[i] MUST exactly match TransactionTypeUrls[i].
// Any mismatch causes silent misrouting — transactions fail with no error.
// Phase 1: ADLMSR only (5 types including send).
// Phase 2: add PORS types at indices 5-11.
// ─────────────────────────────────────────────────────────────────────────────

var ContractConfig = &PluginConfig{
Name:    "praxis_contract",
Id:      1,
Version: 1,
SupportedTransactions: []string{
"send",               // index 0
"create_market",      // index 1
"submit_prediction",  // index 2
"resolve_market",     // index 3
"claim_winnings",     // index 4
"register_resolver",  // index 5
"propose_outcome",    // index 6
"file_dispute",       // index 7
"commit_vote",        // index 8
"reveal_vote",        // index 9
"tally_votes",        // index 10
"finalize_market",    // index 11
"claim_slash",        // index 12
},
TransactionTypeUrls: []string{
"type.googleapis.com/types.MessageSend",
"type.googleapis.com/types.MessageCreateMarket",
"type.googleapis.com/types.MessageSubmitPrediction",
"type.googleapis.com/types.MessageResolveMarket",
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
EventTypeUrls: nil,
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTRACT STRUCT
// Plugin and fsmId are set by plugin.go — do not set manually.
// Do NOT add currentHeight here — use GetGlobalHeight() instead (AUDIT-9).
// ─────────────────────────────────────────────────────────────────────────────

type Contract struct {
Config    Config
FSMConfig *PluginFSMConfig
plugin    *Plugin
fsmId     uint64
}

// ─────────────────────────────────────────────────────────────────────────────
// INIT
// Registers all protobuf file descriptors with the FSM during startup handshake.
// Also initialises PANEL_ENTROPY_KEY which depends on JoinLenPrefix.
// ─────────────────────────────────────────────────────────────────────────────

func init() {
// Initialise the singleton entropy key now that JoinLenPrefix is available.
PANEL_ENTROPY_KEY = KeyForPanelEntropy()

file_account_proto_init()
file_event_proto_init()
file_plugin_proto_init()
file_tx_proto_init()

var fds [][]byte
for _, file := range []protoreflect.FileDescriptor{
anypb.File_google_protobuf_any_proto,
File_account_proto,
File_event_proto,
File_plugin_proto,
File_tx_proto,
} {
fd, err := proto.Marshal(protodesc.ToFileDescriptorProto(file))
if err != nil {
panic("praxis: failed to marshal file descriptor: " + err.Error())
}
fds = append(fds, fd)
}
ContractConfig.FileDescriptorProtos = fds
}

// ─────────────────────────────────────────────────────────────────────────────
// LIFECYCLE
// ─────────────────────────────────────────────────────────────────────────────

// Genesis imports initial state at chain launch. No-op for now.
func (c *Contract) Genesis(_ *PluginGenesisRequest) *PluginGenesisResponse {
return &PluginGenesisResponse{}
}

// BeginBlock is called at the start of every block before any transactions.
// Sets the global height so DeliverTx handlers can reference it (AUDIT-9).
func (c *Contract) BeginBlock(req *PluginBeginRequest) *PluginBeginResponse {
SetGlobalHeight(req.Height)
return &PluginBeginResponse{}
}

// EndBlock is called at the end of every block after all transactions.
func (c *Contract) EndBlock(_ *PluginEndRequest) *PluginEndResponse {
return &PluginEndResponse{}
}

// ─────────────────────────────────────────────────────────────────────────────
// CHECKTX ROUTER
// Stateless validation only — ZERO StateRead calls permitted here (AUDIT-8).
// Validates message fields and returns AuthorizedSigners for FSM sig verification.
// Fee check happens first — rejects obviously underfunded transactions fast.
// ─────────────────────────────────────────────────────────────────────────────

func (c *Contract) CheckTx(req *PluginCheckRequest) *PluginCheckResponse {
msg, err := FromAny(req.Tx.Msg)
if err != nil {
return &PluginCheckResponse{Error: err}
}

// Route to the correct CheckTx handler.
switch x := msg.(type) {
case *MessageSend:
return c.CheckMessageSend(x)
case *MessageCreateMarket:
return c.CheckMessageCreateMarket(x)
case *MessageSubmitPrediction:
return c.CheckMessageSubmitPrediction(x)
case *MessageResolveMarket:
return c.CheckMessageResolveMarket(x)
case *MessageClaimWinnings:
return c.CheckMessageClaimWinnings(x)
case *MessageRegisterResolver:
return c.CheckMessageRegisterResolver(x)
case *MessageProposeOutcome:
return c.CheckMessageProposeOutcome(x)
case *MessageFileDispute:
return c.CheckMessageFileDispute(x)
case *MessageCommitVote:
return c.CheckMessageCommitVote(x)
case *MessageRevealVote:
return c.CheckMessageRevealVote(x)
case *MessageTallyVotes:
return c.CheckMessageTallyVotes(x)
case *MessageFinalizeMarket:
return c.CheckMessageFinalizeMarket(x)
case *MessageClaimSlash:
return c.CheckMessageClaimSlash(x)
default:
return &PluginCheckResponse{Error: ErrInvalidMessageCast()}
}
}

// ─────────────────────────────────────────────────────────────────────────────
// DELIVERTX ROUTER
// Stateful execution — called exactly once per transaction in block order.
// If this returns an error the tx is recorded as failed and fee still charged.
// CheckTx must catch everything recoverable before a tx enters a block.
// ─────────────────────────────────────────────────────────────────────────────

func (c *Contract) DeliverTx(req *PluginDeliverRequest) *PluginDeliverResponse {
msg, err := FromAny(req.Tx.Msg)
if err != nil {
return &PluginDeliverResponse{Error: err}
}

switch x := msg.(type) {
case *MessageSend:
return c.DeliverMessageSend(x, req.Tx.Fee)
case *MessageCreateMarket:
return c.DeliverMessageCreateMarket(x, req.Tx.Fee)
case *MessageSubmitPrediction:
return c.DeliverMessageSubmitPrediction(x, req.Tx.Fee)
case *MessageResolveMarket:
return c.DeliverMessageResolveMarket(x, req.Tx.Fee)
case *MessageClaimWinnings:
return c.DeliverMessageClaimWinnings(x, req.Tx.Fee)
case *MessageRegisterResolver:
return c.DeliverMessageRegisterResolver(x, req.Tx.Fee)
case *MessageProposeOutcome:
return c.DeliverMessageProposeOutcome(x, req.Tx.Fee)
case *MessageFileDispute:
return c.DeliverMessageFileDispute(x, req.Tx.Fee)
case *MessageCommitVote:
return c.DeliverMessageCommitVote(x, req.Tx.Fee)
case *MessageRevealVote:
return c.DeliverMessageRevealVote(x, req.Tx.Fee)
case *MessageTallyVotes:
return c.DeliverMessageTallyVotes(x, req.Tx.Fee)
case *MessageFinalizeMarket:
return c.DeliverMessageFinalizeMarket(x, req.Tx.Fee)
case *MessageClaimSlash:
return c.DeliverMessageClaimSlash(x, req.Tx.Fee)
default:
return &PluginDeliverResponse{Error: ErrInvalidMessageCast()}
}
}
