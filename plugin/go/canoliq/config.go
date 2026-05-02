package canoliq

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/canopy-network/go-plugin/contract"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// CanoliqConfig is the plugin identity advertised to the FSM during handshake.
// Registers all canoLiq-specific transaction types alongside the standard
// 'send' message reused from the existing tx.proto.
var CanoliqConfig = &contract.PluginConfig{
	Name:    "canoliq_plugin",
	Id:      2,
	Version: 1,
	SupportedTransactions: []string{
		"send",
		"canoliq_deposit",
		"canoliq_redeem",
		"canoliq_claim_redemption",
		"cliq_transfer",
		"cliq_claim_vested",
		"cliq_stake",
		"cliq_unstake",
		"cliq_claim_unstake",
		"cliq_proposal_create",
		"cliq_vote",
		"buyback_execute",
		"dao_treasury_spend",
		"multisig_approve",
	},
	TransactionTypeUrls: []string{
		"type.googleapis.com/types.MessageSend",
		"type.googleapis.com/types.MessageCanoliqDeposit",
		"type.googleapis.com/types.MessageCanoliqRedeem",
		"type.googleapis.com/types.MessageCanoliqClaimRedemption",
		"type.googleapis.com/types.MessageCLIQTransfer",
		"type.googleapis.com/types.MessageCLIQClaimVested",
		"type.googleapis.com/types.MessageCLIQStake",
		"type.googleapis.com/types.MessageCLIQUnstake",
		"type.googleapis.com/types.MessageCLIQClaimUnstake",
		"type.googleapis.com/types.MessageCLIQProposalCreate",
		"type.googleapis.com/types.MessageCLIQVote",
		"type.googleapis.com/types.MessageBuybackExecute",
		"type.googleapis.com/types.MessageDAOTreasurySpend",
		"type.googleapis.com/types.MessageMultisigApprove",
	},
	EventTypeUrls: nil,
}

// init seeds CanoliqConfig.FileDescriptorProtos with the proto files the FSM
// needs to decode canoLiq transactions. Mirrors the registration pattern in
// contract/contract.go::init.
func init() {
	var fds [][]byte
	for _, file := range []protoreflect.FileDescriptor{
		anypb.File_google_protobuf_any_proto,
		contract.File_account_proto,
		contract.File_event_proto,
		contract.File_plugin_proto,
		contract.File_tx_proto,
		contract.File_canoliq_proto,
	} {
		fd, _ := proto.Marshal(protodesc.ToFileDescriptorProto(file))
		fds = append(fds, fd)
	}
	CanoliqConfig.FileDescriptorProtos = fds
}

// Config is the canoLiq plugin runtime configuration. It mirrors
// contract.Config but adds the chainId of the canoLiq committee (used to
// derive the committee reward pool key) and the path to genesis.json.
type Config struct {
	ChainId     uint64 `json:"chainId"`
	DataDirPath string `json:"dataDirPath"`
	GenesisPath string `json:"genesisPath"`
}

// DefaultConfig returns reasonable defaults: the next free committee chainId
// and the standard plugin socket directory.
func DefaultConfig() Config {
	return Config{
		ChainId:     2,
		DataDirPath: filepath.Join("/tmp/plugin/"),
		GenesisPath: "",
	}
}

// NewConfigFromFile populates a Config from a JSON file, falling back to
// DefaultConfig values for any missing fields.
func NewConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err = json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// DefaultParams returns the canonical canoLiq fee/split parameters from the
// whitepaper: 12% protocol fee with a 40/30/15/15 split. Phase 2 governance
// defaults (voting period, quorum, multisig, etc.) are pinned in the plan
// at docs/plans/canoliq-implementation-plan.md.
func DefaultParams() *contract.CanoliqParams {
	return &contract.CanoliqParams{
		FeeBps:             1200,
		UserRebateBps:      4000,
		TreasuryBps:        3000,
		ValidatorBps:       1500,
		BuybackBps:         1500,
		DepositFee:         10_000,
		RedeemFee:          10_000,
		ClaimFee:           10_000,
		CliqTransferFee:    10_000,
		InsuranceBps:       1500,        // 15% of treasury slice ≈ 1.5% of fee — within WP §11
		TreasuryThreshold:  1_000_000_000, // 1k CNPY-equivalent in uCNPY
		MultisigSigners:    nil,         // populated at genesis (genesis.json) or via param-change vote
		MultisigThreshold:  3,
		VotingPeriodBlocks: 100_800, // ~7d at 6s blocks
		QuorumBps:          3300,    // 33% of snapshot staked CLIQ
		PassThresholdBps:   5001,    // just-above 50% of (yes+no)
		TimelockBlocks:     28_800,  // ~48h at 6s blocks
		CliqUnstakingBlocks: 100_800, // ~7d at 6s — must be ≥ voting period
		ProposalFee:        10_000,
		VoteFee:            10_000,
		StakeFee:           10_000,
		MultisigApproveFee: 10_000,
		MinStakeToPropose:  1_000_000, // 1 CLIQ minimum to deter spam
	}
}

// ValidateParams enforces invariants on a CanoliqParams record before it is
// stored or used. The four fee-split bps fields must total 10000; ranges and
// monotonic relationships for the Phase 2 fields are enforced when present.
func ValidateParams(p *contract.CanoliqParams) *contract.PluginError {
	if p == nil {
		return ErrInvalidParams()
	}
	if p.FeeBps > 10_000 {
		return ErrInvalidParams()
	}
	if p.UserRebateBps+p.TreasuryBps+p.ValidatorBps+p.BuybackBps != 10_000 {
		return ErrInvalidParams()
	}
	if p.InsuranceBps > 10_000 {
		return ErrInvalidParams()
	}
	if p.QuorumBps > 10_000 {
		return ErrInvalidParams()
	}
	if p.PassThresholdBps > 10_000 {
		return ErrInvalidParams()
	}
	if signers := uint64(len(p.MultisigSigners)); signers > 0 && p.MultisigThreshold > signers {
		return ErrInvalidParams()
	}
	// Unstaking window must be ≥ voting period so a voter cannot stake → vote
	// → unstake → unwind their position before tally. Skip the check if either
	// is unset (zero) so DefaultParams loaded by older state still validates.
	if p.VotingPeriodBlocks > 0 && p.CliqUnstakingBlocks > 0 && p.CliqUnstakingBlocks < p.VotingPeriodBlocks {
		return ErrInvalidParams()
	}
	return nil
}
