package canoliq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

// Config is the canoLiq plugin runtime configuration.
//
// Profile selects the deployment environment so genesis/parameter values
// can be sanity-checked at startup. Recognized values: "localnet"
// (placeholder addresses are OK), "testnet" / "mainnet" (loading the
// localnet placeholder address into a bucket recipient is refused). Empty
// is treated as "localnet" for backwards compatibility with existing
// docker compose deployments.
//
// RedemptionUnstakingBlocks is the cooldown applied to a queued redemption
// before it can be claimed. Localnet uses 5 (~30s) for fast testing;
// testnet/mainnet should set this to match Canopy's
// `valParams.UnstakingBlocks` (typically thousands of blocks). Reading
// the live value from FSM gov-params is tracked as future work.
//
// RpcAddress optionally turns on the read-only HTTP query layer. Empty
// disables it. CANOLIQ_RPC_ADDR env var overrides this at startup.
type Config struct {
	Profile                   string `json:"profile"`
	ChainId                   uint64 `json:"chainId"`
	DataDirPath               string `json:"dataDirPath"`
	GenesisPath               string `json:"genesisPath"`
	RpcAddress                string `json:"rpcAddress"`
	RedemptionUnstakingBlocks uint64 `json:"redemptionUnstakingBlocks"`
}

// localnetPlaceholderAddress is the single hex address every bundled
// genesis.localnet.json bucket points at — the validator key shipped in
// .docker/volumes. Refusing this address under non-localnet profiles
// prevents the most common foot-gun: shipping the localnet genesis.json
// into a real environment.
const localnetPlaceholderAddress = "851e90eaef1fa27debaee2c2591503bdeec1d123"

// Profile constants. Empty string is normalized to ProfileLocalnet for
// backwards compatibility.
const (
	ProfileLocalnet = "localnet"
	ProfileTestnet  = "testnet"
	ProfileMainnet  = "mainnet"
)

// DefaultConfig returns reasonable defaults for localnet: chainId 2, the
// standard plugin socket directory, the 5-block fast redemption window.
// The HTTP query layer ships disabled by default — operators opt in by
// setting RpcAddress.
func DefaultConfig() Config {
	return Config{
		Profile:                   ProfileLocalnet,
		ChainId:                   2,
		DataDirPath:               filepath.Join("/tmp/plugin/"),
		GenesisPath:               "",
		RpcAddress:                "",
		RedemptionUnstakingBlocks: 5,
	}
}

// NewConfigFromFile populates a Config from a JSON file, falling back to
// DefaultConfig values for any missing fields. The Profile field is
// normalized to ProfileLocalnet when empty for backwards compatibility.
func NewConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err = json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	if c.Profile == "" {
		c.Profile = ProfileLocalnet
	}
	if c.RedemptionUnstakingBlocks == 0 {
		c.RedemptionUnstakingBlocks = 5
	}
	return c, nil
}

// LogProfileBanner prints a single line at startup announcing which
// profile is loaded and which genesis file the plugin will read. Helps
// operators catch a "wrong file mounted" mistake before genesis runs.
func (c Config) LogProfileBanner() {
	log.Printf("canoliq: profile=%q chainId=%d genesis=%q rpc=%q redemptionUnstakingBlocks=%d",
		c.Profile, c.ChainId, c.GenesisPath, c.RpcAddress, c.RedemptionUnstakingBlocks)
}

// SafetyCheck validates a non-localnet profile against the genesis file's
// recipient addresses. The single-address localnet placeholder
// (851e90…d123) being routed 100% of every bucket is safe on localnet
// (it's just a test key), but disastrous on testnet/mainnet — every CLIQ
// bucket would mint to one external party. This check refuses to proceed
// when any bucket recipient matches the placeholder under a non-localnet
// profile.
//
// Localnet (or empty/unrecognized) profiles are skipped — this is a
// guard rail, not a strict validator. Genesis schema correctness is
// enforced separately by validateGenesis.
func (c Config) SafetyCheck() error {
	if c.Profile == ProfileLocalnet || c.Profile == "" {
		return nil
	}
	if c.GenesisPath == "" {
		return nil
	}
	data, err := os.ReadFile(c.GenesisPath)
	if err != nil {
		// Genesis loading errors surface later in runGenesis with a
		// clearer message; don't double-report here.
		return nil
	}
	var gf GenesisFile
	if err := json.Unmarshal(data, &gf); err != nil {
		return nil
	}
	for _, b := range gf.Buckets {
		for _, r := range b.Recipients {
			if strings.EqualFold(strings.TrimPrefix(r.Address, "0x"), localnetPlaceholderAddress) {
				return fmt.Errorf("canoliq: refusing to start profile=%q with localnet placeholder address in bucket %q (set real bucket recipient addresses in %s)",
					c.Profile, b.Name, c.GenesisPath)
			}
		}
	}
	return nil
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
		InsuranceBps:       500,         // 5% of treasury slice — matches Tokenomics v1.1 §8 "5% of DAO treasury inflow" reading
		InsuranceTargetBps: 500,         // T4: reserve target = 5% of peak TVL (WP §9.2); skim auto-off at target
		// T5 autonomy-graduation thresholds (WP §10). TVL is a flat uCNPY
		// placeholder (~$50M at $1/CNPY) pending a real price oracle.
		GraduationMinTvlUcnpy:     50_000_000_000_000,
		GraduationMinValidators:   30,
		GraduationMinTurnoutBps:   1_500,
		GraduationMinDailyTx:      10_000,
		GraduationMinRunwayMonths: 12,
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
		Governance:         defaultGovernanceTiers(),
	}
}

// Block-count constants for governance timing at the 6s localnet block time.
const (
	blocks24h = 14_400  // ~24h at 6s blocks
	blocks48h = 28_800  // ~48h
	blocks7d  = 100_800 // ~7d
	blocks14d = 201_600 // ~14d
)

// defaultGovernanceTiers returns the 7-row per-action governance matrix from
// Tokenomics v1.1 §7. quorum/approval are in bps; voting/timelock in blocks.
// Non-emergency actions use the standard 7-day voting window; the emergency
// tier uses a 24h fast-track vote with zero timelock (F13).
func defaultGovernanceTiers() []*contract.GovernanceTier {
	return []*contract.GovernanceTier{
		{Action: contract.ActionType_ACTION_FEE_CHANGE, QuorumBps: 500, ApprovalBps: 5100, TimelockBlocks: blocks48h, VotingPeriodBlocks: blocks7d},
		{Action: contract.ActionType_ACTION_TREASURY_SPEND_SMALL, QuorumBps: 500, ApprovalBps: 5100, TimelockBlocks: blocks48h, VotingPeriodBlocks: blocks7d},
		{Action: contract.ActionType_ACTION_TREASURY_SPEND_LARGE, QuorumBps: 1000, ApprovalBps: 6700, TimelockBlocks: blocks7d, VotingPeriodBlocks: blocks7d},
		{Action: contract.ActionType_ACTION_EMERGENCY, QuorumBps: 800, ApprovalBps: 6700, TimelockBlocks: 0, VotingPeriodBlocks: blocks24h},
		{Action: contract.ActionType_ACTION_VALIDATOR_EJECT, QuorumBps: 500, ApprovalBps: 5100, TimelockBlocks: blocks48h, VotingPeriodBlocks: blocks7d},
		{Action: contract.ActionType_ACTION_PROTOCOL_UPGRADE, QuorumBps: 1000, ApprovalBps: 6700, TimelockBlocks: blocks7d, VotingPeriodBlocks: blocks7d},
		{Action: contract.ActionType_ACTION_AUTONOMY_GRADUATE, QuorumBps: 1500, ApprovalBps: 7500, TimelockBlocks: blocks14d, VotingPeriodBlocks: blocks7d},
	}
}

// ValidateParams enforces invariants on a CanoliqParams record before it is
// stored or used. The four fee-split bps fields must total 10000; ranges and
// monotonic relationships for the Phase 2 fields are enforced when present.
func ValidateParams(p *contract.CanoliqParams) *contract.PluginError {
	if p == nil {
		return ErrInvalidParams()
	}
	if p.FeeBps < 500 || p.FeeBps > 2000 {
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
	// Every governance tier must carry a known action and in-range bps. An
	// empty list is valid (callers fall back to the scalar knobs above).
	seen := make(map[contract.ActionType]bool, len(p.Governance))
	for _, t := range p.Governance {
		if t == nil || t.Action == contract.ActionType_ACTION_UNKNOWN {
			return ErrInvalidParams()
		}
		if seen[t.Action] {
			return ErrInvalidParams()
		}
		seen[t.Action] = true
		if t.QuorumBps > 10_000 || t.ApprovalBps > 10_000 {
			return ErrInvalidParams()
		}
	}
	return nil
}

// tierFor returns the governance tier matching action, or nil when no tier is
// configured for it (callers then fall back to the scalar quorum / approval /
// timelock / voting-period knobs on CanoliqParams).
func tierFor(p *contract.CanoliqParams, action contract.ActionType) *contract.GovernanceTier {
	for _, t := range p.Governance {
		if t != nil && t.Action == action {
			return t
		}
	}
	return nil
}
