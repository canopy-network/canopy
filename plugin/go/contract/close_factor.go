package contract

import "math/big"

// CloseFactorBpsForHF implements ARCM Section 7's dynamic close factor,
// tiered by how far underwater a position is at liquidation time. Input is
// HF_scaled (same 1_000_000 scaling as ComputeHealthFactorScaled / Section
// 5) -- callers MUST have already confirmed HF_scaled <=
// HFLiquidatableThresholdScaled before calling this; behavior for a
// healthy position is undefined by this function (ARCM Section 7 only
// defines close factor for an already-liquidatable position).
//
// HF >  0.95 (950_000  < HF_scaled <= 1_000_000): Tier 1, 30% (3000 bps)
// HF >  0.85 (850_000  < HF_scaled <= 950_000):   Tier 2, 60% (6000 bps)
// HF <= 0.85 (HF_scaled <= 850_000):              Tier 3, 100% (10000 bps)
func CloseFactorBpsForHF(hfScaled *big.Int) uint64 {
	tier2Threshold := big.NewInt(950_000)
	tier3Threshold := big.NewInt(850_000)
	switch {
	case hfScaled.Cmp(tier2Threshold) > 0:
		return 3000
	case hfScaled.Cmp(tier3Threshold) > 0:
		return 6000
	default:
		return 10000
	}
}
