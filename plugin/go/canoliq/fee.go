package canoliq

import (
	"math/big"
)

// computeMint returns the amount of cCNPY to mint for a CNPY deposit.
//
// The exchange rate uses a +1 virtual-shares/virtual-assets offset (ERC-4626
// style, M5): mint = amount * (total_ccnpy + 1) / (total_pooled_cnpy + 1). The
// virtual unit anchors the rate so the first depositor cannot start it at a
// manipulable 1:1 against an empty pool and skew rounding against later small
// depositors. The first deposit still mints ~1:1 (both totals 0 -> *1/1), and
// at a 1:1 rate the offset cancels. Flooring favors the pool (no over-mint).
func computeMint(amount, totalCcnpy, totalPooled uint64) uint64 {
	if amount == 0 {
		return 0
	}
	return mulDiv(amount, totalCcnpy+1, totalPooled+1)
}

// computeRedeem returns the amount of CNPY owed for a cCNPY burn at the current
// exchange rate, with the same +1 virtual offset as computeMint (M5):
// cnpy = ccnpy_amount * (total_pooled_cnpy + 1) / (total_ccnpy + 1). Flooring
// favors the pool (no over-pay); a few uCNPY of dust is permanently retained,
// which is what keeps the pool from draining to a manipulable empty state.
func computeRedeem(ccnpyAmount, totalCcnpy, totalPooled uint64) uint64 {
	if ccnpyAmount == 0 {
		return 0
	}
	return mulDiv(ccnpyAmount, totalPooled+1, totalCcnpy+1)
}

// mulDiv returns (a*b)/c using big.Int internally to avoid uint64 overflow.
// Mirrors the safety properties of lib.SafeMulDiv from Canopy core.
func mulDiv(a, b, c uint64) uint64 {
	if c == 0 {
		return 0
	}
	bigA := new(big.Int).SetUint64(a)
	bigB := new(big.Int).SetUint64(b)
	bigC := new(big.Int).SetUint64(c)
	res := new(big.Int).Mul(bigA, bigB)
	res.Quo(res, bigC)
	if !res.IsUint64() {
		return 0
	}
	return res.Uint64()
}

// FeeSplit holds the four destinations for a single block's fee skim.
// Sum of (UserRebate, Treasury, Validators, Buyback) == feeAmount minus
// any integer-truncation residual, which is sent to the treasury.
type FeeSplit struct {
	UserRebate uint64
	Treasury   uint64
	Validators uint64
	Buyback    uint64
}

// SplitFee divides feeAmount into the four canonical buckets according to the
// bps weights in params. Truncation residual is added to the treasury bucket
// so the four parts always sum to feeAmount exactly.
func SplitFee(feeAmount uint64, p *FeeSplitParams) FeeSplit {
	user := mulDiv(feeAmount, p.UserRebateBps, 10_000)
	treasury := mulDiv(feeAmount, p.TreasuryBps, 10_000)
	validators := mulDiv(feeAmount, p.ValidatorBps, 10_000)
	buyback := mulDiv(feeAmount, p.BuybackBps, 10_000)
	allocated := user + treasury + validators + buyback
	if allocated < feeAmount {
		treasury += feeAmount - allocated
	}
	return FeeSplit{
		UserRebate: user,
		Treasury:   treasury,
		Validators: validators,
		Buyback:    buyback,
	}
}

// FeeSplitParams is the slim subset of CanoliqParams needed for fee math.
type FeeSplitParams struct {
	UserRebateBps uint64
	TreasuryBps   uint64
	ValidatorBps  uint64
	BuybackBps    uint64
}

// FeeOnReward returns the protocol fee carved out of a block's reward delta.
func FeeOnReward(delta uint64, feeBps uint64) uint64 {
	return mulDiv(delta, feeBps, 10_000)
}
