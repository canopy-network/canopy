package contract

import "math/big"

// RAY is AYIS's fixed-point precision unit (1e18). B_index, S_rate, and
// loss_factor are all RAY-scaled uint128 values (AYIS Section 2).
var RAY = big.NewInt(1_000_000_000_000_000_000)

// TryEncodeUint128 encodes a non-negative big.Int into a 16-byte big-endian
// buffer, returning ok=false rather than panicking if v is negative or
// exceeds 128 bits. This is the ONLY safe way to encode a uint128-scale value
// from a BeginBlock context, where there is no transaction to revert (AYIS
// v1.10 Section 9, M1; Principle 14). Callers in a BeginBlock-context write
// path (B_index, S_rate, R_fund's interest-routing leg) MUST use this
// function directly and handle ok=false by freezing the single affected
// market (setting index_overflow_halted), never by panicking or silently
// truncating.
func TryEncodeUint128(v *big.Int) ([]byte, bool) {
	if v.Sign() < 0 || v.BitLen() > 128 {
		return nil, false
	}
	b := v.Bytes()
	out := make([]byte, 16)
	copy(out[16-len(b):], b)
	return out, true
}

// EncodeUint128 is a thin reverting wrapper around TryEncodeUint128, for the
// DeliverTx-context call sites where a clean transaction revert IS the
// correct response to an out-of-range value (loss_factor, which Invariant I8
// proves is structurally bounded and never actually needs the freeze
// treatment; R_fund's repay/liquidation legs, which execute inside a
// transaction that can safely abort). Panics via Go's error-return
// convention -- callers MUST check the returned error and revert the
// transaction, never ignore it.
func EncodeUint128(v *big.Int) ([]byte, *PluginError) {
	encoded, ok := TryEncodeUint128(v)
	if !ok {
		return nil, ErrUint128EncodingOverflow(v.String())
	}
	return encoded, nil
}

// DecodeUint128 decodes a 16-byte big-endian buffer into a big.Int.
func DecodeUint128(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

// EncodeUint64 encodes a uint64 as 8-byte big-endian, matching AYIS Section
// 9's SupplyIndexRecord layout convention.
func EncodeUint64(v uint64) []byte {
	out := make([]byte, 8)
	out[0] = byte(v >> 56)
	out[1] = byte(v >> 48)
	out[2] = byte(v >> 40)
	out[3] = byte(v >> 32)
	out[4] = byte(v >> 24)
	out[5] = byte(v >> 16)
	out[6] = byte(v >> 8)
	out[7] = byte(v)
	return out
}

// DecodeUint64 decodes an 8-byte big-endian buffer into a uint64.
func DecodeUint64(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

// EncodeSupplyIndexRecord implements {26}'s explicit 24-byte layout (AYIS
// v1.9 Section 9, M3): bytes 0-15 = s_rate (uint128, big-endian), bytes
// 16-23 = total_shares_outstanding (uint64, big-endian). sRateEncoded is
// passed in already encoded by the caller, which chose TryEncodeUint128 or
// the reverting wrapper as appropriate for its own call-site context -- this
// function does not re-decide that policy, only the byte layout.
func EncodeSupplyIndexRecord(sRateEncoded []byte, totalShares uint64) []byte {
	out := make([]byte, 24)
	copy(out[0:16], sRateEncoded)
	copy(out[16:24], EncodeUint64(totalShares))
	return out
}

// DecodeSupplyIndexRecord reverses EncodeSupplyIndexRecord.
func DecodeSupplyIndexRecord(b []byte) (sRate *big.Int, totalShares uint64) {
	sRate = DecodeUint128(b[0:16])
	totalShares = DecodeUint64(b[16:24])
	return
}

// EncodeAssetTierRecord implements {29}'s 1-byte layout: the asset's tier
// (0-3, ARCM Section 3.1). Tier validity (0-3, never 4/Blacklisted) is
// checked by the caller at the write site, not here -- matching this
// file's existing convention (EncodeSupplyIndexRecord does not itself
// re-decide the DeliverTx/BeginBlock encoding-context policy its caller
// already chose; this function likewise only defines the byte layout).
func EncodeAssetTierRecord(tier uint8) []byte {
	return []byte{tier}
}

// DecodeAssetTierRecord reverses EncodeAssetTierRecord.
func DecodeAssetTierRecord(b []byte) uint8 {
	return b[0]
}
