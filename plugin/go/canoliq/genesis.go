package canoliq

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/canopy-network/go-plugin/contract"
)

// CLIQTotalSupply is the hard-coded fixed CLIQ supply (100M CLIQ in uCLIQ).
// 1 CLIQ = 1_000_000 uCLIQ for parity with uCNPY micro-units.
const CLIQTotalSupply uint64 = 100_000_000 * 1_000_000

// GenesisFile is the JSON shape persisted at plugin/go/canoliq/genesis.json.
// Each Bucket lists the recipient addresses and the bps weights summing to
// 10_000 across the seven canonical buckets.
//
// ValidatorRegistry seeds the canoLiq committee validator set used by
// reward.go::distributeValidatorShare for per-validator pro-rata. When
// present and non-empty, the 15% validator-incentive slice is split
// proportional to per-validator stake. When absent or empty, the legacy
// committee aggregator path stays in effect (Phase 1 baseline).
type GenesisFile struct {
	BlocksPerYear     uint64                       `json:"blocksPerYear"`
	Buckets           []GenesisBucket              `json:"buckets"`
	Params            *GenesisParamsJSON           `json:"params,omitempty"`
	ValidatorRegistry []GenesisValidatorRegistryEntry `json:"validatorRegistry,omitempty"`
}

// GenesisValidatorRegistryEntry pairs a validator address with the stake
// weight used to share out the per-validator reward slice. Stake is in
// the same uCNPY-equivalent unit as Canopy's `Validator.StakedAmount`,
// so operators can copy it directly from the chain's existing validator
// set or seed it manually.
type GenesisValidatorRegistryEntry struct {
	Address string `json:"address"` // hex-encoded 20-byte address (with or without 0x)
	Stake   uint64 `json:"stake"`   // share-out weight
}

// GenesisBucket describes one of the CLIQ allocation tranches.
type GenesisBucket struct {
	Name        string                 `json:"name"`
	Bps         uint64                 `json:"bps"`
	CliffMonths uint64                 `json:"cliffMonths"`
	VestMonths  uint64                 `json:"vestMonths"`
	Recipients  []GenesisAllocation    `json:"recipients"`
}

// GenesisAllocation is a single (address, share) pair within a bucket.
// Share is in bps within the bucket; shares within a bucket must sum to 10_000.
type GenesisAllocation struct {
	Address string `json:"address"`
	Bps     uint64 `json:"bps"`
}

// GenesisParamsJSON optionally overrides DefaultParams() at genesis time.
// Phase 2 fields (insurance, governance, multisig) are honored when present
// and fall back to DefaultParams() values otherwise.
type GenesisParamsJSON struct {
	FeeBps              uint64   `json:"feeBps"`
	UserRebateBps       uint64   `json:"userRebateBps"`
	TreasuryBps         uint64   `json:"treasuryBps"`
	ValidatorBps        uint64   `json:"validatorBps"`
	BuybackBps          uint64   `json:"buybackBps"`
	DepositFee          uint64   `json:"depositFee"`
	RedeemFee           uint64   `json:"redeemFee"`
	ClaimFee            uint64   `json:"claimFee"`
	CliqTransferFee     uint64   `json:"cliqTransferFee"`
	InsuranceBps        uint64   `json:"insuranceBps"`
	TreasuryThreshold   uint64   `json:"treasuryThreshold"`
	MultisigSigners     []string `json:"multisigSigners"` // hex-encoded 20-byte addresses
	MultisigThreshold   uint64   `json:"multisigThreshold"`
	VotingPeriodBlocks  uint64   `json:"votingPeriodBlocks"`
	QuorumBps           uint64   `json:"quorumBps"`
	PassThresholdBps    uint64   `json:"passThresholdBps"`
	TimelockBlocks      uint64   `json:"timelockBlocks"`
	CliqUnstakingBlocks uint64   `json:"cliqUnstakingBlocks"`
	ProposalFee         uint64   `json:"proposalFee"`
	VoteFee             uint64   `json:"voteFee"`
	StakeFee            uint64   `json:"stakeFee"`
	MultisigApproveFee  uint64   `json:"multisigApproveFee"`
	MinStakeToPropose   uint64   `json:"minStakeToPropose"`
}

// runGenesis is the body of Canoliq.Genesis. It is a no-op once the globals
// record reports genesis_complete=true. Loads the genesis distribution from
// the configured path (Config.GenesisPath) or req.GenesisJson if non-nil.
func (c *Canoliq) runGenesis(req *contract.PluginGenesisRequest) *contract.PluginError {
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if g.GenesisComplete {
		return nil
	}
	gf, e := loadGenesisFile(c.Config.GenesisPath, req)
	if e != nil {
		return e
	}
	if err := validateGenesis(gf); err != nil {
		return err
	}
	params := DefaultParams()
	if gf.Params != nil {
		params = paramsFromJSON(gf.Params)
		if err := ValidateParams(params); err != nil {
			return err
		}
	}
	if err := c.SaveParams(params); err != nil {
		return err
	}
	g.CliqTotalSupply = CLIQTotalSupply
	if err := c.applyGenesisBuckets(gf, g); err != nil {
		return err
	}
	if err := c.applyGenesisValidatorRegistry(gf); err != nil {
		return err
	}
	g.GenesisComplete = true
	return c.SaveGlobals(g)
}

// applyGenesisValidatorRegistry writes the seeded validator set when
// genesis carries one. No-op for empty/missing entries — the legacy
// aggregator path in distributeValidatorShare keeps working.
func (c *Canoliq) applyGenesisValidatorRegistry(gf *GenesisFile) *contract.PluginError {
	if len(gf.ValidatorRegistry) == 0 {
		return nil
	}
	reg := &contract.ValidatorRegistry{
		Entries: make([]*contract.ValidatorRegistryEntry, 0, len(gf.ValidatorRegistry)),
	}
	for _, e := range gf.ValidatorRegistry {
		raw := e.Address
		if len(raw) >= 2 && (raw[:2] == "0x" || raw[:2] == "0X") {
			raw = raw[2:]
		}
		addr, err := hex.DecodeString(raw)
		if err != nil {
			return ErrStateUnmarshal(fmt.Errorf("validator address %q: %w", e.Address, err))
		}
		if len(addr) != 20 {
			return ErrStateUnmarshal(fmt.Errorf("validator address %q must be 20 bytes", e.Address))
		}
		reg.Entries = append(reg.Entries, &contract.ValidatorRegistryEntry{
			Address: addr,
			Stake:   e.Stake,
		})
	}
	bz, e := contract.Marshal(reg)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForValidatorRegistry(), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}

func loadGenesisFile(path string, req *contract.PluginGenesisRequest) (*GenesisFile, *contract.PluginError) {
	var data []byte
	if req != nil && len(req.GenesisJson) > 0 {
		data = req.GenesisJson
	} else if path != "" {
		bz, err := os.ReadFile(path)
		if err != nil {
			return nil, ErrStateUnmarshal(err)
		}
		data = bz
	} else {
		return nil, ErrStateUnmarshal(fmt.Errorf("no genesis source configured"))
	}
	gf := new(GenesisFile)
	if err := json.Unmarshal(data, gf); err != nil {
		return nil, ErrStateUnmarshal(err)
	}
	return gf, nil
}

func validateGenesis(gf *GenesisFile) *contract.PluginError {
	if gf == nil || len(gf.Buckets) == 0 {
		return ErrStateUnmarshal(fmt.Errorf("genesis must list at least one bucket"))
	}
	bpsSum := uint64(0)
	for _, b := range gf.Buckets {
		bpsSum += b.Bps
		recBps := uint64(0)
		for _, r := range b.Recipients {
			recBps += r.Bps
			if _, err := hex.DecodeString(r.Address); err != nil {
				return ErrStateUnmarshal(fmt.Errorf("invalid hex address %q: %w", r.Address, err))
			}
		}
		if recBps != 10_000 {
			return ErrStateUnmarshal(fmt.Errorf("bucket %q: recipients bps sum %d (want 10000)", b.Name, recBps))
		}
	}
	if bpsSum != 10_000 {
		return ErrStateUnmarshal(fmt.Errorf("bucket bps sum %d (want 10000)", bpsSum))
	}
	return nil
}

// applyGenesisBuckets allocates CLIQ to recipients and writes either a liquid
// balance (cliff_months==0) or a VestingSchedule with linear unlock between
// cliff and end. Tranches with vest_months==0 unlock the full amount at the
// cliff and are stored as a one-instant schedule for accounting clarity.
func (c *Canoliq) applyGenesisBuckets(gf *GenesisFile, g *contract.CanoliqGlobals) *contract.PluginError {
	blocksPerYear := gf.BlocksPerYear
	if blocksPerYear == 0 {
		blocksPerYear = 5_256_000 // assume ~6s blocks
	}
	blocksPerMonth := blocksPerYear / 12
	sets := make([]*contract.PluginSetOp, 0)
	indexUpdates := make(map[string]*contract.VestingIndex)
	scheduleCounter := uint64(0)
	for _, b := range gf.Buckets {
		bucketTotal := mulDiv(CLIQTotalSupply, b.Bps, 10_000)
		for _, r := range b.Recipients {
			addrBytes, _ := hex.DecodeString(r.Address)
			amount := mulDiv(bucketTotal, r.Bps, 10_000)
			if amount == 0 {
				continue
			}
			if b.CliffMonths == 0 && b.VestMonths == 0 {
				// fully liquid at TGE
				key := KeyForCLIQBalance(addrBytes)
				existing, exists := liquidExisting(sets, key)
				existing += amount
				if exists {
					replaceSet(sets, key, EncodeUint64(existing))
				} else {
					sets = append(sets, &contract.PluginSetOp{Key: key, Value: EncodeUint64(existing)})
				}
				g.CliqCirculatingSupply += amount
				continue
			}
			scheduleCounter++
			scheduleID := scheduleCounter
			cliff := b.CliffMonths * blocksPerMonth
			end := cliff + b.VestMonths*blocksPerMonth
			sched := &contract.VestingSchedule{
				Address:      addrBytes,
				ScheduleId:   scheduleID,
				TotalAmount:  amount,
				CliffHeight:  cliff,
				StartHeight:  cliff,
				EndHeight:    end,
				ClaimedAmount: 0,
			}
			bz, e := contract.Marshal(sched)
			if e != nil {
				return e
			}
			sets = append(sets, &contract.PluginSetOp{
				Key:   KeyForVesting(addrBytes, scheduleID),
				Value: bz,
			})
			idx, ok := indexUpdates[r.Address]
			if !ok {
				idx = &contract.VestingIndex{}
				indexUpdates[r.Address] = idx
			}
			idx.ScheduleIds = append(idx.ScheduleIds, scheduleID)
		}
	}
	for addr, idx := range indexUpdates {
		addrBytes, _ := hex.DecodeString(addr)
		bz, e := contract.Marshal(idx)
		if e != nil {
			return e
		}
		sets = append(sets, &contract.PluginSetOp{
			Key:   KeyForVestingIndex(addrBytes),
			Value: bz,
		})
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets}); err != nil {
		return err
	}
	_ = rand.Uint64()
	return nil
}

func paramsFromJSON(p *GenesisParamsJSON) *contract.CanoliqParams {
	d := DefaultParams()
	if p.FeeBps != 0 {
		d.FeeBps = p.FeeBps
	}
	if p.UserRebateBps+p.TreasuryBps+p.ValidatorBps+p.BuybackBps != 0 {
		d.UserRebateBps = p.UserRebateBps
		d.TreasuryBps = p.TreasuryBps
		d.ValidatorBps = p.ValidatorBps
		d.BuybackBps = p.BuybackBps
	}
	if p.DepositFee != 0 {
		d.DepositFee = p.DepositFee
	}
	if p.RedeemFee != 0 {
		d.RedeemFee = p.RedeemFee
	}
	if p.ClaimFee != 0 {
		d.ClaimFee = p.ClaimFee
	}
	if p.CliqTransferFee != 0 {
		d.CliqTransferFee = p.CliqTransferFee
	}
	if p.InsuranceBps != 0 {
		d.InsuranceBps = p.InsuranceBps
	}
	if p.TreasuryThreshold != 0 {
		d.TreasuryThreshold = p.TreasuryThreshold
	}
	if len(p.MultisigSigners) > 0 {
		signers := make([][]byte, 0, len(p.MultisigSigners))
		for _, hexAddr := range p.MultisigSigners {
			b, err := hex.DecodeString(hexAddr)
			if err == nil && len(b) == 20 {
				signers = append(signers, b)
			}
		}
		d.MultisigSigners = signers
	}
	if p.MultisigThreshold != 0 {
		d.MultisigThreshold = p.MultisigThreshold
	}
	if p.VotingPeriodBlocks != 0 {
		d.VotingPeriodBlocks = p.VotingPeriodBlocks
	}
	if p.QuorumBps != 0 {
		d.QuorumBps = p.QuorumBps
	}
	if p.PassThresholdBps != 0 {
		d.PassThresholdBps = p.PassThresholdBps
	}
	if p.TimelockBlocks != 0 {
		d.TimelockBlocks = p.TimelockBlocks
	}
	if p.CliqUnstakingBlocks != 0 {
		d.CliqUnstakingBlocks = p.CliqUnstakingBlocks
	}
	if p.ProposalFee != 0 {
		d.ProposalFee = p.ProposalFee
	}
	if p.VoteFee != 0 {
		d.VoteFee = p.VoteFee
	}
	if p.StakeFee != 0 {
		d.StakeFee = p.StakeFee
	}
	if p.MultisigApproveFee != 0 {
		d.MultisigApproveFee = p.MultisigApproveFee
	}
	if p.MinStakeToPropose != 0 {
		d.MinStakeToPropose = p.MinStakeToPropose
	}
	return d
}

// liquidExisting returns the running CLIQ balance for `key` that is already
// staged in `sets`. Used so multiple recipients in the same bucket sharing an
// address get aggregated into one set op rather than overwriting each other.
func liquidExisting(sets []*contract.PluginSetOp, key []byte) (uint64, bool) {
	for _, s := range sets {
		if string(s.Key) == string(key) {
			return DecodeUint64(s.Value), true
		}
	}
	return 0, false
}

func replaceSet(sets []*contract.PluginSetOp, key, value []byte) {
	for _, s := range sets {
		if string(s.Key) == string(key) {
			s.Value = value
			return
		}
	}
}
