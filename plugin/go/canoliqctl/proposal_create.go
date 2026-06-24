package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

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
//	canoliqctl proposal-create param-change    <nickname> <params-json-file> [--description ...]
//	canoliqctl proposal-create buyback         <nickname> <cnpy-amount> <price-micro-cnpy-per-cplq> <burn|distribute> [--description ...]
//	canoliqctl proposal-create treasury-spend  <nickname> <recipient-hex> <amount> <cnpy|cplq> [--description ...]
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
	case "help", "-h", "--help":
		return printProposalCreateHelp()
	default:
		return fmt.Errorf("unknown proposal-create subcommand %q (want param-change|buyback|treasury-spend|validator-eject|emergency)", args[0])
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
	return nil
}

// cmdProposalParamChange submits a ProposalParamChange. Loads the new
// CanoliqParams from a JSON file (typically a copy of the genesis "params"
// block, edited). The plugin runs ValidateParams on the payload at
// dispatchPassed, so invalid bps sums or signer/threshold mismatches surface
// only after the proposal passes — operators should pre-validate.
func cmdProposalParamChange(args []string, gf globalFlags) error {
	usage := "proposal-create param-change <nickname> <params-json-file> [--description \"…\"]"
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
	usage := "proposal-create buyback <nickname> <cnpy-amount> <price-micro-cnpy-per-cplq> <burn|distribute> [--description \"…\"]"
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
	usage := "proposal-create treasury-spend <nickname> <recipient-hex> <amount> <cnpy|cplq> [--description \"…\"]"
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
	usage := "proposal-create validator-eject <nickname> <validator-hex> [--description \"…\"]"
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

// cmdProposalEmergency submits a ProposalEmergency on the fast-track tier
// (ACTION_EMERGENCY: 8% quorum / 67% / 24h vote / no timelock). An optional
// params-json-file is included as the emergency param diff applied on pass.
func cmdProposalEmergency(args []string, gf globalFlags) error {
	usage := "proposal-create emergency <nickname> [params-json-file] [--description \"…\"]"
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
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_proposal_create", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("proposal-create %s submitted: tx_hash=%s from=%s description=%q\n",
		kind, hash, signer.Address, description)
	fmt.Printf("  payload typeUrl=%s bytes=%d\n", payload.TypeUrl, len(payload.Value))
	return nil
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
func loadParamsFromFile(path string) (*contract.CanoliqParams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read params file %q: %w", path, err)
	}
	// We intentionally use the proto's own JSON decoding rather than
	// stdlib because the proto has @gotags that match the wire shape.
	// Using stdlib json with proto messages is acceptable here because
	// CanoliqParams is a flat scalar+repeated-bytes message; the bytes
	// fields (`multisig_signers`) are encoded as base64 in protojson but
	// hex in the genesis convention, so we accept both forms.
	var raw paramsJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse params JSON: %w", err)
	}
	return raw.toContract()
}

// paramsJSON mirrors the on-disk shape used by genesis files. Field
// names are camelCase to match @gotags. multisigSigners are hex strings
// (with or without 0x), to match the genesis file convention.
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
}

func (p paramsJSON) toContract() (*contract.CanoliqParams, error) {
	signers := make([][]byte, 0, len(p.MultisigSigners))
	for _, s := range p.MultisigSigners {
		b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
		if err != nil {
			return nil, fmt.Errorf("multisig signer %q: %w", s, err)
		}
		if len(b) != 20 {
			return nil, fmt.Errorf("multisig signer %q must be 20 bytes", s)
		}
		signers = append(signers, b)
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
