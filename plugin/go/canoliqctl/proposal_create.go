package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/canopy-network/go-plugin/canoliq"
	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdProposalCreate dispatches MessageCPLQProposalCreate to one of three
// payload sub-commands. The plugin's `MessageCPLQProposalCreate.Payload`
// is a `google.protobuf.Any` resolved by TypeUrl in
// governance.go::unwrapPayload, so each sub-command builds its
// concrete payload, anypb.New-wraps it, and submits the same outer
// message.
//
// Usage:
//
//	canoliqctl proposal-create param-change    <address> <params-json-file> [--description ...]
//	canoliqctl proposal-create buyback         <address> <cnpy-amount> <price-micro-cnpy-per-cplq> <burn|distribute> [--description ...]
//	canoliqctl proposal-create treasury-spend  <address> <recipient-hex> <amount> <cnpy|cplq> [--description ...]
func cmdProposalCreate(args []string, gf globalFlags) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: %s", commandUsages["proposal-create"])
	}
	switch args[0] {
	case "param-change":
		return cmdProposalParamChange(args[1:], gf)
	case "buyback":
		return cmdProposalBuyback(args[1:], gf)
	case "treasury-spend":
		return cmdProposalTreasurySpend(args[1:], gf)
	case "validator-eject":
		return cmdProposalValidatorEject(args[1:], gf)
	case "emergency":
		return cmdProposalEmergency(args[1:], gf)
	case "otc-program-fund":
		return cmdProposalOTCProgramFund(args[1:], gf)
	case "help", "-h", "--help":
		return printProposalCreateHelp()
	default:
		return fmt.Errorf("unknown proposal-create subcommand %q (want param-change|buyback|treasury-spend|validator-eject|emergency|otc-program-fund)", args[0])
	}
}

func printProposalCreateHelp() error {
	fmt.Println(commandUsages["proposal-create"])
	fmt.Println()
	fmt.Println("subcommands:")
	fmt.Println("  param-change    full-set CanoliqParams replacement (read from JSON file)")
	fmt.Println("  buyback         CNPY → CPLQ buyback at a vote-set price (BURN or DISTRIBUTE_STAKERS)")
	fmt.Println("  treasury-spend  authorize a transfer from treasury_canoliq (CNPY) or treasury_cplq (CPLQ)")
	fmt.Println("  validator-eject remove a validator from the committee registry (F12)")
	fmt.Println("  emergency       security-critical fast-track action with optional param diff (F13)")
	fmt.Println("  otc-program-fund move CPLQ from treasury_cplq into the OTC lock program budget")
	return nil
}

// cmdProposalParamChange submits a ProposalParamChange. Loads the new
// CanoliqParams from a JSON file (typically a copy of the genesis "params"
// block, edited). The plugin runs ValidateParams on the payload at
// dispatchPassed, so invalid bps sums or signer/threshold mismatches surface
// only after the proposal passes — operators should pre-validate.
func cmdProposalParamChange(args []string, gf globalFlags) error {
	usage := "proposal-create param-change <address> <params-json-file> [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 2 {
		return fmt.Errorf("expected 2 positional args (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	params, err := loadParamsFromFile(rest[1])
	if err != nil {
		return err
	}
	payload, err := anypb.New(&contract.ProposalParamChange{Params: params})
	if err != nil {
		return fmt.Errorf("wrap param-change payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "param-change")
}

// cmdProposalBuyback submits a ProposalBuyback authorizing a single
// CNPY → CPLQ extraction at a vote-set price.
//
// price-micro-cnpy-per-cplq is "how many uCNPY = 1 CPLQ × 10^6"; the plugin
// computes `cplq_acquired = cnpy_amount * 10^6 / price`.
func cmdProposalBuyback(args []string, gf globalFlags) error {
	usage := "proposal-create buyback <address> <cnpy-amount> <price-micro-cnpy-per-cplq> <burn|distribute> [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 4 {
		return fmt.Errorf("expected 4 positional args (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	cnpyAmount, err := parseUint(rest[1], "cnpy-amount")
	if err != nil {
		return err
	}
	price, err := parseUint(rest[2], "price-micro-cnpy-per-cplq")
	if err != nil {
		return err
	}
	if price == 0 {
		return fmt.Errorf("price must be greater than zero")
	}
	mode, err := parseBuybackMode(rest[3])
	if err != nil {
		return err
	}
	payload, err := anypb.New(&contract.ProposalBuyback{
		CnpyAmount:            cnpyAmount,
		PriceMicroCnpyPerCplq: price,
		Mode:                  mode,
	})
	if err != nil {
		return fmt.Errorf("wrap buyback payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "buyback")
}

// cmdProposalTreasurySpend submits a ProposalTreasurySpend authorizing a
// transfer from treasury_canoliq (CNPY) or treasury_cplq (CPLQ).
// Above-threshold spends additionally require multisig + timelock — those
// are enforced at execution, not at proposal create.
func cmdProposalTreasurySpend(args []string, gf globalFlags) error {
	usage := "proposal-create treasury-spend <address> <recipient-hex> <amount> <cnpy|cplq> [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 4 {
		return fmt.Errorf("expected 4 positional args (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	recipient, err := addrFromHex(strings.TrimPrefix(rest[1], "0x"))
	if err != nil {
		return fmt.Errorf("recipient: %w", err)
	}
	if len(recipient) != 20 {
		return fmt.Errorf("recipient must be 20 bytes (40 hex chars), got %d bytes", len(recipient))
	}
	amount, err := parseUint(rest[2], "amount")
	if err != nil {
		return err
	}
	denom, err := parseSpendDenomination(rest[3])
	if err != nil {
		return err
	}
	payload, err := anypb.New(&contract.ProposalTreasurySpend{
		Recipient:    recipient,
		Amount:       amount,
		Denomination: denom,
	})
	if err != nil {
		return fmt.Errorf("wrap treasury-spend payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "treasury-spend")
}

// cmdProposalValidatorEject submits a ProposalValidatorEject removing a
// validator from the committee registry on pass (F12). The plugin infers the
// ACTION_VALIDATOR_EJECT tier (5% quorum / 51% / 48h) from the payload type.
func cmdProposalValidatorEject(args []string, gf globalFlags) error {
	usage := "proposal-create validator-eject <address> <validator-hex> [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 2 {
		return fmt.Errorf("expected 2 positional args (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	validator, err := addrFromHex(strings.TrimPrefix(rest[1], "0x"))
	if err != nil {
		return fmt.Errorf("validator: %w", err)
	}
	if len(validator) != 20 {
		return fmt.Errorf("validator must be 20 bytes (40 hex chars), got %d bytes", len(validator))
	}
	payload, err := anypb.New(&contract.ProposalValidatorEject{ValidatorAddress: validator})
	if err != nil {
		return fmt.Errorf("wrap validator-eject payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "validator-eject")
}

// cmdProposalOTCProgramFund submits a ProposalOTCProgramFund, moving CPLQ from
// treasury_cplq into the OTC lock program's unreserved budget on pass. This is
// the only path by which the program is funded: nothing mints, and the amount
// is capped at the treasury balance at execution time.
func cmdProposalOTCProgramFund(args []string, gf globalFlags) error {
	usage := "proposal-create otc-program-fund <address> <amount-uCPLQ> [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 2 {
		return fmt.Errorf("expected 2 positional args (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	amount, err := parseUint(rest[1], "amount-uCPLQ")
	if err != nil {
		return err
	}
	if amount == 0 {
		return fmt.Errorf("amount-uCPLQ must be non-zero")
	}
	payload, err := anypb.New(&contract.ProposalOTCProgramFund{Amount: amount})
	if err != nil {
		return fmt.Errorf("wrap otc-program-fund payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "otc-program-fund")
}

// cmdProposalEmergency submits a ProposalEmergency on the fast-track tier
// (ACTION_EMERGENCY: 8% quorum / 67% / 24h vote / no timelock). An optional
// params-json-file is included as the emergency param diff applied on pass.
func cmdProposalEmergency(args []string, gf globalFlags) error {
	usage := "proposal-create emergency <address> [params-json-file] [--description \"…\"]"
	rest, description := parseDescriptionFlag(args)
	if len(rest) < 1 {
		return fmt.Errorf("expected at least 1 positional arg (usage: %s)", usage)
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}
	emergency := &contract.ProposalEmergency{Description: description}
	if len(rest) >= 2 {
		params, err := loadParamsFromFile(rest[1])
		if err != nil {
			return err
		}
		emergency.ParamChange = &contract.ProposalParamChange{Params: params}
	}
	payload, err := anypb.New(emergency)
	if err != nil {
		return fmt.Errorf("wrap emergency payload: %w", err)
	}
	return submitProposalCreate(gf, signer, from, payload, description, "emergency")
}

// submitProposalCreate is the shared submit path: build the outer
// MessageCPLQProposalCreate, sign, POST. The kind label is only used for
// the success log line.
func submitProposalCreate(gf globalFlags, signer *internal.Key, from []byte,
	payload *anypb.Any, description, kind string) error {
	msg := &contract.MessageCPLQProposalCreate{
		FromAddress: from,
		Payload:     payload,
		Description: description,
	}
	return submitAndReport(gf, signer, "cplq_proposal_create", msg, "proposal-create "+kind,
		fmt.Sprintf("from=%s description=%q payload=%s (%d bytes)",
			signer.Address, description, payload.TypeUrl, len(payload.Value)))
}

// parseDescriptionFlag pulls "--description X" out of args. We use a
// hand-rolled parser instead of the FlagSet because the outer command
// already consumed the global flags and these sub-args are positional
// with one optional --description trailing.
func parseDescriptionFlag(args []string) (positional []string, description string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--description":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		case "-d":
			if i+1 < len(args) {
				description = args[i+1]
				i++
			}
		default:
			if strings.HasPrefix(args[i], "--description=") {
				description = strings.TrimPrefix(args[i], "--description=")
			} else {
				positional = append(positional, args[i])
			}
		}
	}
	return
}

// loadParamsFromFile reads a CanoliqParams JSON file. Accepts the same
// shape as the genesis "params" block so operators can copy/paste from
// genesis.localnet.json or genesis.testnet.json and edit.
// expectedParamKeys returns the JSON key for every paramsJSON field, derived
// from the struct tags so it cannot drift from the struct itself.
func expectedParamKeys() []string {
	rt := reflect.TypeOf(paramsJSON{})
	keys := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}

// requireCompleteParams rejects a params file that does not carry every key.
//
// This matters because ProposalParamChange is a full-set replacement with no
// merge semantics: dispatchPassed calls SaveParams(p.Params) wholesale, so an
// omitted key is not "leave it alone", it is "set it to zero". Nothing
// downstream catches that — ValidateParams never inspects tvlCapBps, the
// graduation thresholds, or insuranceTargetBps, and accepts an empty governance
// tier list.
//
// Two of those zeros are outright security regressions:
//
//   - omit tvlCapBps and the TVL cap is lifted without anyone voting to lift
//     it, which WP §9.4 makes a DAO decision;
//   - omit multisigSigners AND multisigThreshold together and ValidateParams'
//     threshold check is skipped (it only runs when signers are present), so
//     the threshold lands at 0 and treasury.go's `approvals < threshold` gate
//     passes with zero approvals — the multisig requirement on above-threshold
//     treasury spends silently disappears.
//
// Unknown keys are rejected for the same reason: `"tvlCap"` for `"tvlCapBps"`
// would otherwise be silently ignored and the real field zeroed.
func requireCompleteParams(data []byte, path string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse params JSON: %w", err)
	}
	var missing []string
	for _, k := range expectedParamKeys() {
		if _, ok := raw[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("params file %q is missing %d required key(s): %s\n"+
			"a param-change replaces the WHOLE parameter set, so a missing key is written as zero, not left unchanged.\n"+
			"start from a complete dump: curl -s <plugin-rpc>/v1/params > %s",
			path, len(missing), strings.Join(missing, ", "), path)
	}
	return nil
}

func loadParamsFromFile(path string) (*contract.CanoliqParams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read params file %q: %w", path, err)
	}
	if err := requireCompleteParams(data, path); err != nil {
		return nil, err
	}
	// We intentionally use the proto's own JSON decoding rather than
	// stdlib because the proto has @gotags that match the wire shape.
	// Using stdlib json with proto messages is acceptable here because
	// CanoliqParams is a flat scalar+repeated-bytes message; the bytes
	// fields (`multisig_signers`) are encoded as base64 in protojson but
	// hex in the genesis convention, so we accept both forms.
	//
	// DisallowUnknownFields catches a misspelled key, which would otherwise be
	// dropped silently and leave the real field at zero.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var raw paramsJSON
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse params JSON: %w", err)
	}
	params, err := raw.toContract()
	if err != nil {
		return nil, err
	}
	// Pre-validate locally. The plugin runs ValidateParams at dispatchPassed,
	// i.e. only after the proposal has passed — an invalid payload would burn a
	// full voting period and then fail to apply. Catch it before submitting.
	if perr := canoliq.ValidateParams(params); perr != nil {
		return nil, fmt.Errorf("params file %q fails ValidateParams: %s", path, perr.Msg)
	}
	return params, nil
}

// paramsJSON mirrors the on-disk shape used by genesis files. Field
// names are camelCase to match @gotags. multisigSigners accept either hex
// (the genesis file convention) or base64 (what `GET /v1/params` emits,
// since encoding/json renders the proto's [][]byte that way).
//
// IMPORTANT: this struct must cover EVERY field of contract.CanoliqParams.
// ProposalParamChange is a full-set replacement — dispatchPassed calls
// SaveParams(p.Params) wholesale — so any field missing here is written as its
// Go zero value when the proposal passes. ValidateParams does not catch that:
// it never inspects tvlCapBps, the graduation thresholds, or
// insuranceTargetBps, and it explicitly documents an empty governance tier
// list as valid. A missing field is therefore a silent state corruption, not a
// rejected proposal. TestParamsJSONCoversEveryParamField guards this.
type paramsJSON struct {
	FeeBps              uint64   `json:"feeBps"`
	UserRebateBps       uint64   `json:"userRebateBps"`
	TreasuryBps         uint64   `json:"treasuryBps"`
	ValidatorBps        uint64   `json:"validatorBps"`
	BuybackBps          uint64   `json:"buybackBps"`
	DepositFee          uint64   `json:"depositFee"`
	RedeemFee           uint64   `json:"redeemFee"`
	ClaimFee            uint64   `json:"claimFee"`
	CplqTransferFee     uint64   `json:"cplqTransferFee"`
	CanoliqTransferFee  uint64   `json:"canoliqTransferFee"`
	InsuranceBps        uint64   `json:"insuranceBps"`
	TreasuryThreshold   uint64   `json:"treasuryThreshold"`
	MultisigSigners     []string `json:"multisigSigners"`
	MultisigThreshold   uint64   `json:"multisigThreshold"`
	VotingPeriodBlocks  uint64   `json:"votingPeriodBlocks"`
	QuorumBps           uint64   `json:"quorumBps"`
	PassThresholdBps    uint64   `json:"passThresholdBps"`
	TimelockBlocks      uint64   `json:"timelockBlocks"`
	CplqUnstakingBlocks uint64   `json:"cplqUnstakingBlocks"`
	ProposalFee         uint64   `json:"proposalFee"`
	VoteFee             uint64   `json:"voteFee"`
	StakeFee            uint64   `json:"stakeFee"`
	MultisigApproveFee  uint64   `json:"multisigApproveFee"`
	MinStakeToPropose   uint64   `json:"minStakeToPropose"`

	// Fields below were absent before and were therefore zeroed by every
	// param-change this tool submitted.
	TvlCapBps                 uint64               `json:"tvlCapBps"`
	InsuranceTargetBps        uint64               `json:"insuranceTargetBps"`
	GraduationMinTvlUcnpy     uint64               `json:"graduationMinTvlUcnpy"`
	GraduationMinValidators   uint64               `json:"graduationMinValidators"`
	GraduationMinTurnoutBps   uint64               `json:"graduationMinTurnoutBps"`
	GraduationMinDailyTx      uint64               `json:"graduationMinDailyTx"`
	GraduationMinRunwayMonths uint64               `json:"graduationMinRunwayMonths"`
	Governance                []governanceTierJSON `json:"governance"`
	RestakingPolicy           []restakingEntryJSON `json:"restakingPolicy"`

	// Reward ownership. stakeOutputAddresses lists the canoLiq-controlled
	// output addresses that identify which Canopy bonds belong to the protocol;
	// only their stake growth becomes reward. Omitting the key sends an empty
	// set, which turns reward attribution off — a param-change is a full-set
	// replacement, so carry the current value forward unless you mean to clear
	// it.
	StakeOutputAddresses []string `json:"stakeOutputAddresses"`
	MaxRewardBpsPerBlock uint64   `json:"maxRewardBpsPerBlock"`

	// OTC lock program tier rates, minimum position size, and tier terms.
	OtcTier90Bps     uint64 `json:"otcTier90Bps"`
	OtcTier120Bps    uint64 `json:"otcTier120Bps"`
	OtcMinLockUccnpy uint64 `json:"otcMinLockUccnpy"`
	OtcTier90Blocks  uint64 `json:"otcTier90Blocks"`
	OtcTier120Blocks uint64 `json:"otcTier120Blocks"`
}

// governanceTierJSON is one row of the per-action governance matrix. Action is
// the numeric ActionType, matching what encoding/json emits for the enum.
type governanceTierJSON struct {
	Action             int32  `json:"action"`
	QuorumBps          uint64 `json:"quorumBps"`
	ApprovalBps        uint64 `json:"approvalBps"`
	TimelockBlocks     uint64 `json:"timelockBlocks"`
	VotingPeriodBlocks uint64 `json:"votingPeriodBlocks"`
}

// restakingEntryJSON is one committee's restaking allocation (WP §7).
type restakingEntryJSON struct {
	CommitteeId     uint64 `json:"committeeId"`
	TargetWeightBps uint64 `json:"targetWeightBps"`
	MinStakeUcnpy   uint64 `json:"minStakeUcnpy"`
	MaxStakeUcnpy   uint64 `json:"maxStakeUcnpy"`
}

// decodeSigner accepts a 20-byte address as hex (genesis convention, with or
// without 0x) or base64 (what GET /v1/params emits). Hex is tried first; a
// 20-byte address base64-encodes to 28 chars ending in '=', which hex always
// rejects, so there is no ambiguity between the two forms.
func decodeSigner(s string) ([]byte, error) {
	if b, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err == nil && len(b) == 20 {
		return b, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("multisig signer %q: not valid hex or base64", s)
	}
	if len(b) != 20 {
		return nil, fmt.Errorf("multisig signer %q must be 20 bytes, got %d", s, len(b))
	}
	return b, nil
}

func (p paramsJSON) toContract() (*contract.CanoliqParams, error) {
	signers := make([][]byte, 0, len(p.MultisigSigners))
	for _, s := range p.MultisigSigners {
		b, err := decodeSigner(s)
		if err != nil {
			return nil, err
		}
		signers = append(signers, b)
	}
	// Stake output addresses take the same encodings as multisig signers (hex,
	// 0x-hex, base64) and the same 20-byte shape check.
	outputs := make([][]byte, 0, len(p.StakeOutputAddresses))
	for _, a := range p.StakeOutputAddresses {
		b, err := decodeSigner(a)
		if err != nil {
			return nil, fmt.Errorf("stake output address: %w", err)
		}
		outputs = append(outputs, b)
	}
	var tiers []*contract.GovernanceTier
	for _, t := range p.Governance {
		tiers = append(tiers, &contract.GovernanceTier{
			Action:             contract.ActionType(t.Action),
			QuorumBps:          t.QuorumBps,
			ApprovalBps:        t.ApprovalBps,
			TimelockBlocks:     t.TimelockBlocks,
			VotingPeriodBlocks: t.VotingPeriodBlocks,
		})
	}
	var restaking []*contract.RestakingPolicyEntry
	for _, e := range p.RestakingPolicy {
		restaking = append(restaking, &contract.RestakingPolicyEntry{
			CommitteeId:     e.CommitteeId,
			TargetWeightBps: e.TargetWeightBps,
			MinStakeUcnpy:   e.MinStakeUcnpy,
			MaxStakeUcnpy:   e.MaxStakeUcnpy,
		})
	}
	return &contract.CanoliqParams{
		FeeBps:              p.FeeBps,
		UserRebateBps:       p.UserRebateBps,
		TreasuryBps:         p.TreasuryBps,
		ValidatorBps:        p.ValidatorBps,
		BuybackBps:          p.BuybackBps,
		DepositFee:          p.DepositFee,
		RedeemFee:           p.RedeemFee,
		ClaimFee:            p.ClaimFee,
		CplqTransferFee:     p.CplqTransferFee,
		CanoliqTransferFee:  p.CanoliqTransferFee,
		InsuranceBps:        p.InsuranceBps,
		TreasuryThreshold:   p.TreasuryThreshold,
		MultisigSigners:     signers,
		MultisigThreshold:   p.MultisigThreshold,
		VotingPeriodBlocks:  p.VotingPeriodBlocks,
		QuorumBps:           p.QuorumBps,
		PassThresholdBps:    p.PassThresholdBps,
		TimelockBlocks:      p.TimelockBlocks,
		CplqUnstakingBlocks: p.CplqUnstakingBlocks,
		ProposalFee:         p.ProposalFee,
		VoteFee:             p.VoteFee,
		StakeFee:            p.StakeFee,
		MultisigApproveFee:  p.MultisigApproveFee,
		MinStakeToPropose:   p.MinStakeToPropose,

		StakeOutputAddresses: outputs,
		MaxRewardBpsPerBlock: p.MaxRewardBpsPerBlock,

		TvlCapBps:                 p.TvlCapBps,
		InsuranceTargetBps:        p.InsuranceTargetBps,
		GraduationMinTvlUcnpy:     p.GraduationMinTvlUcnpy,
		GraduationMinValidators:   p.GraduationMinValidators,
		GraduationMinTurnoutBps:   p.GraduationMinTurnoutBps,
		GraduationMinDailyTx:      p.GraduationMinDailyTx,
		GraduationMinRunwayMonths: p.GraduationMinRunwayMonths,
		Governance:                tiers,
		RestakingPolicy:           restaking,
		OtcTier90Bps:              p.OtcTier90Bps,
		OtcTier120Bps:             p.OtcTier120Bps,
		OtcMinLockUccnpy:          p.OtcMinLockUccnpy,
		OtcTier90Blocks:           p.OtcTier90Blocks,
		OtcTier120Blocks:          p.OtcTier120Blocks,
	}, nil
}

func parseBuybackMode(s string) (contract.BuybackMode, error) {
	switch strings.ToLower(s) {
	case "burn":
		return contract.BuybackMode_BUYBACK_BURN, nil
	case "distribute", "distribute-stakers", "distribute_stakers":
		return contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS, nil
	default:
		return contract.BuybackMode_BUYBACK_UNKNOWN,
			fmt.Errorf("invalid buyback mode %q (want burn|distribute)", s)
	}
}

func parseSpendDenomination(s string) (contract.SpendDenomination, error) {
	switch strings.ToLower(s) {
	case "cnpy":
		return contract.SpendDenomination_SPEND_CNPY, nil
	case "cplq":
		return contract.SpendDenomination_SPEND_CPLQ, nil
	default:
		return contract.SpendDenomination_SPEND_UNKNOWN,
			fmt.Errorf("invalid denomination %q (want cnpy|cplq)", s)
	}
}

// guard against unused-import lint when adapting this file in the future.
var _ proto.Message = (*contract.MessageCPLQProposalCreate)(nil)
