package contract

import (
	"encoding/binary"
	"fmt"
)

// State key prefixes reserved for Arbor within Canopy's {16}-{28} custom range,
// plus {29}+ for Arbor-internal extensions beyond the ARCM/AYIS-audited layout
// (currently: {29} asset tier registry -- see KeyForAssetTier below; {40}
// Arbor protocol treasury -- see KeyForTreasuryArbor below; {41} NASM/NUSD
// protocol treasury -- see KeyForTreasuryNASM below). {30}-{39} remains
// reserved for future NASM/NUSD coordination, deferred until core lending is
// complete. {40} was chosen with deliberate headroom above that reservation,
// not immediately adjacent to it, so NASM can claim additional prefixes in
// {30}-{39} without colliding with Treasury as a next-available neighbor.
// This {30}-{39} reservation was explicitly confirmed (not merely inherited
// from an earlier, unverified comment) during Treasury's addition at {40} --
// see PrefixTreasuryArbor/PrefixTreasuryNASM below. A future session should treat {30}-{39} as a
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

	// [REVERSED] PrefixTreasury was originally a SINGLE fund shared across
	// Arbor lending AND NASM/NUSD (see git history for the superseded
	// comment). That decision was reopened and reversed the same session
	// it was made: a shared pool means a NUSD-side bad-debt event can
	// silently drain the Layer 3 protection Arbor lenders were counting
	// on, and vice versa -- a hidden coupling between two products users
	// would reasonably expect to be risk-independent. Reversed to TWO
	// separate, isolated treasuries, one per protocol, deliberately
	// distinct functions rather than one parameterized accessor -- a
	// caller cannot accidentally read/write the wrong pool at the type
	// level, matching the safety-over-brevity priority this reversal
	// itself was made under.
	//
	// PrefixTreasuryArbor: Arbor lending's own protocol treasury (T_fund),
	// a global (not market-keyed) uint128 balance -- ARCM Section 9.2
	// Layer 3. Kept at {40} (its original key) to minimize churn, since
	// it was already live and referenced elsewhere before this reversal.
	PrefixTreasuryArbor = []byte{40}

	// PrefixTreasuryNASM: NASM/NUSD's own protocol treasury, isolated from
	// PrefixTreasuryArbor above -- a global uint128 balance, same shape,
	// separate key. NASM's own waterfall (not yet built) will read/write
	// this key, never PrefixTreasuryArbor. Placed at {41}, immediately
	// adjacent to {40} -- both are Arbor-repo-owned keys (this codebase
	// implements NASM's plugin too), so adjacency here carries none of
	// the collision risk the {30}-{39} NASM-coordination wall exists to
	// prevent; that wall is about a DIFFERENT, external NASM concern
	// (coordination with NUSD's own future key range), not this key.
	PrefixTreasuryNASM = []byte{41}
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

// KeyForCircuitBreaker is keyed by assetID alone -- oracle trustworthiness
// is a canonical property of the asset's own price feed (mirrors
// KeyForAssetTier's identical rationale immediately below), NOT per-market.
// [FIX, session finding] Originally keyed by marketID, with zero real
// callers ever built against that signature -- re-keyed here before any
// caller was written, matching NASM Consolidated Spec Section 9.2's own
// OracleUntrustworthy(asset_id) predicate exactly, rather than building a
// predicate on top of an inconsistent, market-keyed foundation.
func KeyForCircuitBreaker(assetID string) []byte {
	return JoinLenPrefix(PrefixCircuitBreaker, []byte(assetID))
}

// KeyForAssetTier is keyed by assetID alone -- tier is a canonical property
// of the asset itself (ARCM Section 3.1/3.2), not per-market or
// per-submitter, unlike KeyForPriceRecord's composite key above.
func KeyForAssetTier(assetID string) []byte {
	return JoinLenPrefix(PrefixAssetTier, []byte(assetID))
}

// KeyForEmergencyMode is keyed by assetID alone -- same rationale as
// KeyForCircuitBreaker above. [FIX, session finding] Same re-keying, same
// reason: originally marketID-keyed with zero real callers.
func KeyForEmergencyMode(assetID string) []byte {
	return JoinLenPrefix(PrefixEmergencyMode, []byte(assetID))
}

func KeyForGovernanceParams() []byte {
	return JoinLenPrefix(PrefixGovernanceParams)
}

func KeyForBackstopQueue() []byte {
	return JoinLenPrefix(PrefixBackstopQueue)
}

// [REVERSED] KeyForTreasury (single shared key) is replaced by two
// distinct functions below, matching PrefixTreasuryArbor/PrefixTreasuryNASM's
// own split -- see that comment for the full reversal reasoning.
//
// KeyForTreasuryArbor is NOT keyed by marketID, unlike every other key
// builder in this file -- Arbor's protocol treasury is a single global
// balance, not a per-market one. Mirrors KeyForGovernanceParams/
// KeyForBackstopQueue's existing no-argument shape.
func KeyForTreasuryArbor() []byte {
	return JoinLenPrefix(PrefixTreasuryArbor)
}

// KeyForTreasuryNASM is the NASM/NUSD-owned analog of
// KeyForTreasuryArbor above -- same no-argument, global-balance shape,
// separate underlying key ({41}), fully isolated from Arbor's own {40}.
func KeyForTreasuryNASM() []byte {
	return JoinLenPrefix(PrefixTreasuryNASM)
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

// ─────────────────────────────────────────────
// NASM (NUSD / Arbor Stability Module) prefixes -- claims {30}, {31}, {32},
// {34} from the {30}-{39} wall reserved above. {33} is deliberately SKIPPED,
// not renumbered: R_nusd (NASM Consolidated Spec Section 12's "{33} NUSD
// Reserve Fund") was already built earlier, under a different number and
// name, as part of the Treasury-split reversal -- see PrefixTreasuryNASM at
// {41} above, with GetTreasuryNASM/SetTreasuryNASMTry/SetTreasuryNASM in
// state_accessors.go. Compacting {34}->{33} to fill the gap was considered
// and rejected: once anything writes to a state key on mainnet, that number
// is permanent, and a silent renumbering away from the spec doc's own
// Section 12 table would leave a mismatch future readers have no way to
// discover except by re-deriving this exact reasoning. A documented gap
// costs nothing and stays traceable; a compacted range does not.
var (
	// PrefixNasmVaults: NUSD-backing collateral vaults (NASM Spec Section
	// 4, 12). Keyed by vault_id, a caller-supplied string identifier --
	// NOT by owner address, unlike BorrowerPosition/LenderPosition's
	// (market_id, address) composite keys. This is a deliberate departure
	// from that pattern: NASM Spec Section 5.2 requires vault ownership to
	// be "a transferable claim, not a bound identity" (the arbitrage loop's
	// second leg is buying a vault position outright on the secondary
	// market). Keying by owner address would make transfer impossible
	// without a close-and-reopen, which is circular -- the buyer would
	// need to already own the collateral being transferred. vault_id
	// therefore follows market_id's precedent (a stable, caller-supplied
	// string handle, validated but not derived from any address) rather
	// than BorrowerPosition's. The NasmVault record itself carries a
	// mutable owner field; the key does not encode ownership.
	PrefixNasmVaults = []byte{30}

	// PrefixNusdSupply: global NUSD total_supply ledger (NASM Spec Section
	// 12). No-argument key, matching KeyForGovernanceParams/
	// KeyForBackstopQueue/KeyForTreasuryArbor's existing no-argument shape
	// for singleton, non-market-keyed values.
	PrefixNusdSupply = []byte{31}

	// PrefixStabilityFeeIndex: SF_index(t), a single global uint128 (RAY)
	// value (NASM Spec Section 6.2, 12) -- NOT per-market, unlike AYIS's
	// PrefixBorrowIndex/PrefixSupplyIndex. NASM has one pooled stability
	// fee across all vaults, not one per collateral asset or market.
	PrefixStabilityFeeIndex = []byte{32}

	// {33} intentionally not defined here. See block comment above --
	// R_nusd lives at PrefixTreasuryNASM ({41}), not here.

	// PrefixRwaYieldVault: RWA Yield Vault share records, keyed by
	// depositor_address (NASM Spec Section 8, 12). Explicitly separate
	// from NUSD and from NasmVaults above -- RYV shares are not NUSD, not
	// 1:1 redeemable on demand, and this product's failure or success
	// must never touch NUSD's own backing or peg mechanics (Section 8.1's
	// structural-mismatch rationale for excluding RWA from NUSD backing
	// in the first place).
	PrefixRwaYieldVault = []byte{34}
)

func KeyForNasmVault(vaultID string) []byte {
	return JoinLenPrefix(PrefixNasmVaults, []byte(vaultID))
}

func KeyForNusdSupply() []byte {
	return JoinLenPrefix(PrefixNusdSupply)
}

func KeyForStabilityFeeIndex() []byte {
	return JoinLenPrefix(PrefixStabilityFeeIndex)
}

// KeyForRwaYieldVaultPosition is keyed by depositor address alone -- RYV
// positions are NOT transferable claims the way NasmVaults are (Section 8.2
// describes ordinary deposit/withdraw against the depositor's own position,
// with no secondary-market transfer mechanic in the spec), so this correctly
// follows LenderPosition's address-keyed precedent rather than NasmVault's
// vault_id precedent above.
func KeyForRwaYieldVaultPosition(addr []byte) []byte {
	return JoinLenPrefix(PrefixRwaYieldVault, addr)
}

// MaxVaultIDLen bounds vault_id length. Same collision rationale as
// MaxMarketIDLen/MaxAssetIDLen: an unbounded string segment inside
// JoinLenPrefix risks collision between two vault_ids whose lengths differ
// by a multiple of 256. Matches MaxMarketIDLen's value -- vault_id and
// market_id are both caller-supplied identifiers of the same practical
// shape, no reason for a different ceiling.
const MaxVaultIDLen = 64

// ErrVaultIDTooLong is returned when a vault_id exceeds MaxVaultIDLen.
type ErrVaultIDTooLong struct{ Len int }

func (e ErrVaultIDTooLong) Error() string {
	return fmt.Sprintf("vault_id length %d exceeds maximum of %d bytes", e.Len, MaxVaultIDLen)
}

// ValidateVaultID enforces MaxVaultIDLen and non-emptiness. MUST be called
// at CheckMessageMintNusd's stateless admission check, before vault_id is
// ever written to state or used in KeyForNasmVault. Mirrors ValidateMarketID
// exactly.
func ValidateVaultID(vaultID string) error {
	if len(vaultID) == 0 {
		return fmt.Errorf("vault_id must not be empty")
	}
	if len(vaultID) > MaxVaultIDLen {
		return ErrVaultIDTooLong{Len: len(vaultID)}
	}
	return nil
}

// PrefixNusdBalance: a holder's independent NUSD balance ({35}), keyed by
// address alone -- see NusdBalance's own doc comment in arbor_state.proto
// for the full rationale on why this is structurally required and cannot
// reuse Account.Amount. Claims {35} from the {30}-{39} NASM-coordination
// wall -- the next fresh number after {30},{31},{32},{34} ({33} remains
// permanently skipped, see the block comment above). Address-keyed,
// matching LenderPosition/KeyForRwaYieldVaultPosition's precedent, not
// vault_id-keyed like NasmVault -- a holder's NUSD balance is independent
// of any specific vault, exactly as a bank account balance is independent
// of any specific loan.
var PrefixNusdBalance = []byte{35}

func KeyForNusdBalance(addr []byte) []byte {
	return JoinLenPrefix(PrefixNusdBalance, addr)
}

// PrefixNasmTierBacking: NASM Spec Section 3.3's per-tier mint concentration
// cap accumulator (see NasmTierBacking's own doc comment in
// arbor_state.proto for the full design rationale). Claims {36} from the
// {30}-{39} NASM-coordination wall -- the next fresh number after
// {30},{31},{32},{34},{35} ({33} remains permanently skipped, see the block
// comment above). No-argument key, matching KeyForNusdSupply/
// KeyForStabilityFeeIndex's single-global-record convention -- there is
// exactly one NasmTierBacking record for the whole chain, not one per tier
// or per vault.
var PrefixNasmTierBacking = []byte{36}

func KeyForNasmTierBacking() []byte {
	return JoinLenPrefix(PrefixNasmTierBacking)
}

// PrefixWaterfallLog: durable, queryable rolling log of every Layer 2/3/4
// bad-debt waterfall step (ARCM Section 9.2), added to close the exact gap
// the Arbor frontend's Events panel flags directly: waterfall events are
// emitted on-chain (via the existing per-layer Event/anypb payloads in
// arbor_events.proto) but, prior to this prefix, had no persisted, queryable
// history of their own -- see apply_loss_factor.go's "EVENT EMISSION,
// RETURNED NOT EMITTED" doc comment for why the existing Event mechanism
// alone cannot serve this. Claims {42}, the next free number in the
// Arbor-internal-extensions range (beyond ARCM/AYIS's audited {16}-{28}
// layout) -- {40} and {41} are Arbor's and NASM's own treasuries
// respectively (PrefixTreasuryArbor/PrefixTreasuryNASM above); {42} does
// not encroach on either, nor on the {30}-{39} NASM-coordination wall.
//
// Keyed (block_height, seq), both big-endian-encoded via formatUint64 so
// byte-lexicographic key ordering matches numeric chronological ordering --
// this is what lets /v1/query/waterfall-events serve "most recent N events"
// as a single Reverse=true, Limit=N range scan (see handleQueryAllMarkets's
// existing PluginRangeRead precedent in rpc.go) with no separate index
// needed. seq disambiguates multiple waterfall steps landing in the same
// block (a single liquidate_position call can emit a Layer 2 miss followed
// by a Layer 3 hit, or a Layer 2 miss, Layer 3 miss, and Layer 4 application,
// all in one DeliverTx -- see liquidate_position.go's fallthrough chain).
var PrefixWaterfallLog = []byte{42}

// KeyForWaterfallEvent builds the composite (height, seq) key described
// above. seq is caller-assigned per waterfall step within a single
// DeliverTx call (0, 1, 2, ...) -- AppendWaterfallEvent (waterfall_log.go)
// does not assign it itself, since a single liquidation may need to write
// more than one entry and the caller already tracks how many waterfall
// steps it has emitted in the current call via len(events) or an explicit
// counter.
func KeyForWaterfallEvent(blockHeight uint64, seq uint32) []byte {
	seqBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(seqBytes, seq)
	return JoinLenPrefix(PrefixWaterfallLog, formatUint64(blockHeight), seqBytes)
}
