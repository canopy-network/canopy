package contract

import "math/big"

// CompoundExact implements AYIS Section 3's exact (non-modular)
// compounding formula, used when delta_t > MAX_DELTA_T_LINEAR (1000
// blocks, AYIS Section 13). Verified independently in Python against this
// exact formula before being written here; see conversation record for the
// cross-check against the linear approximation at small delta_t.
//
// base = RAY + per_block_rate
// B_index(t) = b_index * base^delta_t / RAY^delta_t
//
// Uses big.Int.Exp directly, matching AYIS's own pseudocode exactly (no
// modulus -- the third Exp argument is nil, meaning ordinary
// exponentiation, not modular exponentiation).
func CompoundExact(bIndex *big.Int, perBlockRate *big.Int, deltaT uint64) *big.Int {
	base := new(big.Int).Add(RAY, perBlockRate)
	numerator := new(big.Int).Exp(base, new(big.Int).SetUint64(deltaT), nil)
	denominator := new(big.Int).Exp(RAY, new(big.Int).SetUint64(deltaT), nil)
	result := new(big.Int).Mul(bIndex, numerator)
	result.Div(result, denominator)
	return result
}
