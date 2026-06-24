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
	"cplq_transfer":            "type.googleapis.com/types.MessageCPLQTransfer",
	"cplq_claim_vested":        "type.googleapis.com/types.MessageCPLQClaimVested",
	"cplq_stake":               "type.googleapis.com/types.MessageCPLQStake",
	"cplq_unstake":             "type.googleapis.com/types.MessageCPLQUnstake",
	"cplq_claim_unstake":       "type.googleapis.com/types.MessageCPLQClaimUnstake",
	"cplq_proposal_create":     "type.googleapis.com/types.MessageCPLQProposalCreate",
	"cplq_vote":                "type.googleapis.com/types.MessageCPLQVote",
	"buyback_execute":          "type.googleapis.com/types.MessageBuybackExecute",
	"dao_treasury_spend":       "type.googleapis.com/types.MessageDAOTreasurySpend",
	"multisig_approve":         "type.googleapis.com/types.MessageMultisigApprove",
}
