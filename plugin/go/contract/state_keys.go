package contract

import "fmt"

// State key prefixes reserved for Arbor within Canopy's {16}-{28} custom range,
// plus {29}+ for Arbor-internal extensions beyond the ARCM/AYIS-audited layout
// (currently: {29} asset tier registry -- see KeyForAssetTier below; {40}
// protocol treasury (T_fund) -- see KeyForTreasury below). {30}-{39} remains
// reserved for future NASM/NUSD coordination, deferred until core lending is
// complete. {40} was chosen with deliberate headroom above that reservation,
// not immediately adjacent to it, so NASM can claim additional prefixes in
// {30}-{39} without colliding with Treasury as a next-available neighbor.
// This {30}-{39} reservation was explicitly confirmed (not merely inherited
// from an earlier, unverified comment) during Treasury's addition at {40} --
// see PrefixTreasury below. A future session should treat {30}-{39} as a
// real, intentional boundary that was checked against ARCM/AYIS and found to
// predate implementation, not as a stale placeholder open for renegotiation.
// Canopy-reserved {1}-{15} are untouched. See ARCM v3.11.1 Section 19.1 / AYIS
// v1.11 Section 9 for the canonical, audited layout this file implements.

// ARCM-owned prefixes
var (
	PrefixMarkets           = []byte{16} // market records: status, index_overflow_halted, layer4_* counters, total_borrowed, total_supplied, last_accrual_block
	PrefixBorrowerPositions = []byte{17} // borrower positions: collateral_quantity, debt_principal, borrow_index_at_open
	PrefixReserveFund       = []byte{18} // R_fund per market, uint128
	PrefixPriceCache        = []byte{19} // oracle price records
	PrefixCircuitBreaker    = []byte{20} // circuit breaker state
	PrefixEmergencyMode     = []byte{21} // emergency mode flags
	PrefixGovernanceParams  = []byte{22} // governance parameter store
	PrefixBackstopQueue     = []byte{23} // backstop liquidation queue
)

// AYIS-owned prefixes
var (
	PrefixLenderPositions = []byte{24} // lender positions: shares, deposit_block
	PrefixBorrowIndex     = []byte{25} // B_index per market, uint128
	PrefixSupplyIndex     = []byte{26} // SupplyIndexRecord: s_rate (uint128) + total_shares_outstanding (uint64)
	PrefixLossFactor      = []byte{27} // loss_factor per market, uint128
	PrefixLossFactorQueue = []byte{28} // {28} loss-factor-application queue
)

// Arbor-internal prefixes, beyond ARCM/AYIS's audited {16}-{28} layout.
var (
	// PrefixAssetTier: per-asset tier registry (0=Tier0/CNPY .. 3=Tier3/Restricted).
	// Tier 4 (Blacklisted, ARCM Section 3.1) is never stored here -- an asset
	// with no registry entry is treated as unregistered/ineligible, not as
	// Tier 4, since Tier 4's defining property is that it CANNOT be used as
	// collateral or borrowed asset at all (Section 3.1), making a stored
	// "tier 4" record meaningless -- there is nothing a market could ever
	// legitimately do with that asset to read it for. This registry is the
	// single source of truth for asset tier, replacing Market.AssetTier's
	// prior self-declared, unvalidated field (see create_market.go).
	// Population path: set_asset_tier transaction (admin/risk-committee
	// gated), not yet implemented -- same disclosed-deferral pattern as
	// interest_rate.go's governance-parameter-store gap.
	PrefixAssetTier = []byte{29}

	// PrefixTreasury: protocol treasury (T_fund), a single global uint128
	// balance -- NOT market-keyed, unlike every other accessor in this
	// codebase. Placed at {40}, well clear of the {30}-{39} NASM/NUSD
	// reservation (see header comment), rather than at the next free
	// integer immediately adjacent to that wall.
	PrefixTreasury = []byte{40}
)

// MaxMarketIDLen bounds market_id length. JoinLenPrefix encodes each segment's
// length as a single byte (byte(len(item))), which silently wraps mod 256 for
// any segment over 255 bytes -- a real key-collision path between two market_ids
// whose lengths differ by a multiple of 256. Neither ARCM nor AYIS states a
// market_id length bound anywhere; this constant and ValidateMarketID close that
// gap at the one place market_id is ever created (create_market), rather than
// guarding every downstream key-builder call site individually.
const MaxMarketIDLen = 64

// ErrMarketIDTooLong is returned when a market_id exceeds MaxMarketIDLen.
type ErrMarketIDTooLong struct {
	Len int
}

func (e ErrMarketIDTooLong) Error() string {
	return fmt.Sprintf("market_id length %d exceeds maximum of %d bytes", e.Len, MaxMarketIDLen)
}

// ValidateMarketID enforces MaxMarketIDLen. MUST be called at create_market's
// DeliverTx admission check, before market_id is ever written to state or used
// in any JoinLenPrefix-based key. No other call site needs this guard: every
// key-builder function below assumes market_id was already validated at
// creation time (Principle 9 -- single mandatory write path owns the guard).
func ValidateMarketID(marketID string) error {
	if len(marketID) == 0 {
		return fmt.Errorf("market_id must not be empty")
	}
	if len(marketID) > MaxMarketIDLen {
		return ErrMarketIDTooLong{Len: len(marketID)}
	}
	return nil
}

// ─────────────────────────────────────────────
// Key builders -- one per prefix, all market_id-keyed except lender positions
// (which additionally keys by address, matching AYIS {24}'s composite key).
// ─────────────────────────────────────────────

func KeyForMarket(marketID string) []byte {
	return JoinLenPrefix(PrefixMarkets, []byte(marketID))
}

func KeyForBorrowerPosition(marketID string, addr []byte) []byte {
	return JoinLenPrefix(PrefixBorrowerPositions, []byte(marketID), addr)
}

func KeyForReserveFund(marketID string) []byte {
	return JoinLenPrefix(PrefixReserveFund, []byte(marketID))
}

// KeyForPriceRecord is keyed by (assetID, submitter), not assetID alone --
// ARCM Section 10's quorum model (MIN_REPORTERS) requires multiple
// independent submitters' readings to coexist per asset, not one
// overwritable record. See interest_accrual.go-adjacent design note: a
// single-record-per-asset shape would let one submitter silently
// overwrite another's honest reading, defeating quorum/deviation checks
// entirely. Matches KeyForBorrowerPosition/KeyForLenderPosition's existing
// composite-key pattern.
func KeyForPriceRecord(assetID string, submitter []byte) []byte {
	return JoinLenPrefix(PrefixPriceCache, []byte(assetID), submitter)
}

func KeyForCircuitBreaker(marketID string) []byte {
	return JoinLenPrefix(PrefixCircuitBreaker, []byte(marketID))
}

// KeyForAssetTier is keyed by assetID alone -- tier is a canonical property
// of the asset itself (ARCM Section 3.1/3.2), not per-market or
// per-submitter, unlike KeyForPriceRecord's composite key above.
func KeyForAssetTier(assetID string) []byte {
	return JoinLenPrefix(PrefixAssetTier, []byte(assetID))
}

func KeyForEmergencyMode(marketID string) []byte {
	return JoinLenPrefix(PrefixEmergencyMode, []byte(marketID))
}

func KeyForGovernanceParams() []byte {
	return JoinLenPrefix(PrefixGovernanceParams)
}

func KeyForBackstopQueue() []byte {
	return JoinLenPrefix(PrefixBackstopQueue)
}

// KeyForTreasury is NOT keyed by marketID, unlike every other key builder
// in this file -- the protocol treasury is a single global balance, not a
// per-market one (see PrefixTreasury's comment above for why {40} was
// chosen). This mirrors KeyForGovernanceParams/KeyForBackstopQueue's
// existing no-argument shape; it is a deliberate precedent break from the
// marketID-keyed norm, not an oversight.
func KeyForTreasury() []byte {
	return JoinLenPrefix(PrefixTreasury)
}

func KeyForLenderPosition(marketID string, addr []byte) []byte {
	return JoinLenPrefix(PrefixLenderPositions, []byte(marketID), addr)
}

func KeyForBorrowIndex(marketID string) []byte {
	return JoinLenPrefix(PrefixBorrowIndex, []byte(marketID))
}

func KeyForSupplyIndex(marketID string) []byte {
	return JoinLenPrefix(PrefixSupplyIndex, []byte(marketID))
}

func KeyForLossFactor(marketID string) []byte {
	return JoinLenPrefix(PrefixLossFactor, []byte(marketID))
}

// KeyForLossFactorQueue is per-market, matching every other {16}-{28}
// key helper in this file. A market has at most one outstanding queue
// entry at a time (see LossFactorQueueEntry's doc comment in
// arbor_state.proto for why no separate sequence/ordering key is
// needed): EnqueueLossFactorApplication overwrites any existing entry
// for that market_id rather than appending a second one.
func KeyForLossFactorQueue(marketID string) []byte {
	return JoinLenPrefix(PrefixLossFactorQueue, []byte(marketID))
}

// MaxAssetIDLen bounds asset_id length. Asset IDs ("CNPY", "BTC", "USDC")
// are conventionally short, but this must still be an explicit ceiling --
// see MaxMarketIDLen's own comment: an unbounded string segment inside
// JoinLenPrefix is the same key-collision risk regardless of which field
// carries it. Sized generously rather than tightly, matching
// MaxMarketIDLen's own precedent.
const MaxAssetIDLen = 32

// ErrAssetIDTooLong is returned when an asset_id exceeds MaxAssetIDLen.
type ErrAssetIDTooLong struct{ Len int }

func (e ErrAssetIDTooLong) Error() string {
	return fmt.Sprintf("asset_id length %d exceeds maximum of %d bytes", e.Len, MaxAssetIDLen)
}

// ValidateAssetID enforces MaxAssetIDLen. MUST be called at
// CheckMessageSetAssetTier's stateless admission check, before asset_id is
// ever written to state or used in any JoinLenPrefix-based key
// (KeyForAssetTier, KeyForPriceRecord). Mirrors ValidateMarketID exactly.
func ValidateAssetID(assetID string) error {
	if len(assetID) == 0 {
		return fmt.Errorf("asset_id must not be empty")
	}
	if len(assetID) > MaxAssetIDLen {
		return ErrAssetIDTooLong{Len: len(assetID)}
	}
	return nil
}
