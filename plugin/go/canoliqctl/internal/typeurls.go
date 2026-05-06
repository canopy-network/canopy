package internal

// MsgTypeURL maps the short message-type name (matching the plugin's
// SupportedTransactions registration) to the fully-qualified
// google.protobuf.Any type URL the FSM uses to resolve the message proto.
//
// Keep in sync with plugin/go/canoliq/config.go::CanoliqConfig.
var MsgTypeURL = map[string]string{
	"canoliq_deposit":          "type.googleapis.com/types.MessageCanoliqDeposit",
	"canoliq_redeem":           "type.googleapis.com/types.MessageCanoliqRedeem",
	"canoliq_claim_redemption": "type.googleapis.com/types.MessageCanoliqClaimRedemption",
	"cliq_transfer":            "type.googleapis.com/types.MessageCLIQTransfer",
	"cliq_claim_vested":        "type.googleapis.com/types.MessageCLIQClaimVested",
	"cliq_stake":               "type.googleapis.com/types.MessageCLIQStake",
	"cliq_unstake":             "type.googleapis.com/types.MessageCLIQUnstake",
	"cliq_claim_unstake":       "type.googleapis.com/types.MessageCLIQClaimUnstake",
	"cliq_proposal_create":     "type.googleapis.com/types.MessageCLIQProposalCreate",
	"cliq_vote":                "type.googleapis.com/types.MessageCLIQVote",
	"buyback_execute":          "type.googleapis.com/types.MessageBuybackExecute",
	"dao_treasury_spend":       "type.googleapis.com/types.MessageDAOTreasurySpend",
	"multisig_approve":         "type.googleapis.com/types.MessageMultisigApprove",
}
