package contract

// ═══════════════════════════════════════════════════════════════════════════════
// Praxis Prediction Market — Error Definitions
// Spec authority:
//   ADLMSR v5.6.6-r2-CORRECTED
//   PORS   v1.0-r2-CORRECTED
//
// Built-in Canopy error codes 1–14 are reserved — never reuse them.
// Praxis error codes start at 100.
// ═══════════════════════════════════════════════════════════════════════════════

const errModule = "praxis"

// ─────────────────────────────────────────────────────────────────────────────
// CANOPY BUILT-IN ERRORS (1–14)
// These constructors wrap the built-in codes from plugin.go.
// ─────────────────────────────────────────────────────────────────────────────

func ErrPluginTimeout() *PluginError {
return &PluginError{Code: 1, Module: errModule, Msg: "plugin timeout"}
}
func ErrMarshal(err error) *PluginError {
return &PluginError{Code: 2, Module: errModule, Msg: "marshal error: " + err.Error()}
}
func ErrUnmarshal(err error) *PluginError {
return &PluginError{Code: 3, Module: errModule, Msg: "unmarshal error: " + err.Error()}
}
func ErrPluginRead(err error) *PluginError {
return &PluginError{Code: 4, Module: errModule, Msg: "state read error: " + err.Error()}
}
func ErrPluginWrite(err error) *PluginError {
return &PluginError{Code: 5, Module: errModule, Msg: "state write error: " + err.Error()}
}
func ErrInvalidResponseId() *PluginError {
return &PluginError{Code: 6, Module: errModule, Msg: "invalid response id"}
}
func ErrUnexpectedType() *PluginError {
return &PluginError{Code: 7, Module: errModule, Msg: "unexpected message type"}
}
func ErrInvalidMessage() *PluginError {
return &PluginError{Code: 8, Module: errModule, Msg: "invalid message"}
}
func ErrInsufficientFunds() *PluginError {
return &PluginError{Code: 9, Module: errModule, Msg: "insufficient funds"}
}
func ErrFromAny(err error) *PluginError {
return &PluginError{Code: 10, Module: errModule, Msg: "from any error: " + err.Error()}
}
func ErrInvalidMessageCast() *PluginError {
return &PluginError{Code: 11, Module: errModule, Msg: "invalid message cast"}
}
func ErrInvalidAddress() *PluginError {
return &PluginError{Code: 12, Module: errModule, Msg: "invalid address: must be exactly 20 bytes"}
}
func ErrInvalidAmount() *PluginError {
return &PluginError{Code: 13, Module: errModule, Msg: "invalid amount: must be non-zero"}
}
func ErrTxFeeBelowStateLimit() *PluginError {
return &PluginError{Code: 14, Module: errModule, Msg: "tx fee below minimum"}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRAXIS ERRORS — GENERAL (100–119)
// ─────────────────────────────────────────────────────────────────────────────

func ErrStateReadFailed() *PluginError {
return &PluginError{Code: 100, Module: errModule, Msg: "state read failed"}
}
func ErrMarshalFailed() *PluginError {
return &PluginError{Code: 101, Module: errModule, Msg: "marshal failed"}
}
func ErrUnmarshalFailed() *PluginError {
return &PluginError{Code: 102, Module: errModule, Msg: "unmarshal failed"}
}
func ErrHeightNotSet() *PluginError {
return &PluginError{Code: 103, Module: errModule, Msg: "global height not set: BeginBlock has not been called"}
}
func ErrInternal() *PluginError {
return &PluginError{Code: 104, Module: errModule, Msg: "internal error"}
}
func ErrUnauthorized() *PluginError {
return &PluginError{Code: 105, Module: errModule, Msg: "unauthorized: caller is not the registered resolver"}
}
func ErrInvalidParam() *PluginError {
return &PluginError{Code: 106, Module: errModule, Msg: "invalid parameter"}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRAXIS ERRORS — MARKET (120–139)
// ─────────────────────────────────────────────────────────────────────────────

func ErrMarketNotFound() *PluginError {
return &PluginError{Code: 120, Module: errModule, Msg: "market not found"}
}
func ErrMarketNotOpen() *PluginError {
return &PluginError{Code: 121, Module: errModule, Msg: "market is not open"}
}
func ErrMarketCancelled() *PluginError {
return &PluginError{Code: 122, Module: errModule, Msg: "market has been cancelled"}
}
func ErrMarketNotResolved() *PluginError {
return &PluginError{Code: 123, Module: errModule, Msg: "market has not been resolved"}
}
func ErrMarketNotExpired() *PluginError {
return &PluginError{Code: 124, Module: errModule, Msg: "market has not expired yet"}
}
func ErrResolutionTooEarly() *PluginError {
return &PluginError{Code: 125, Module: errModule, Msg: "resolution window has not opened yet"}
}
func ErrExpiryTooLarge() *PluginError {
return &PluginError{Code: 126, Module: errModule, Msg: "expiry_time exceeds MAX_EXPIRY_TIME"}
}
func ErrInvalidNonce() *PluginError {
return &PluginError{Code: 127, Module: errModule, Msg: "nonce must be non-zero"}
}
func ErrInvalidQuestion() *PluginError {
return &PluginError{Code: 128, Module: errModule, Msg: "question must be non-empty"}
}
func ErrInvalidB0() *PluginError {
return &PluginError{Code: 129, Module: errModule, Msg: "b0 must be >= MIN_B0"}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRAXIS ERRORS — POSITION / PREDICTION (140–159)
// ─────────────────────────────────────────────────────────────────────────────

func ErrNoPosition() *PluginError {
return &PluginError{Code: 140, Module: errModule, Msg: "no position found for this address in this market"}
}
func ErrAlreadyClaimed() *PluginError {
return &PluginError{Code: 141, Module: errModule, Msg: "winnings already claimed"}
}
func ErrCostExceedsMaxCost() *PluginError {
return &PluginError{Code: 142, Module: errModule, Msg: "computed cost exceeds max_cost slippage limit"}
}
func ErrSharesBelowMinimum() *PluginError {
return &PluginError{Code: 143, Module: errModule, Msg: "shares must be >= PRECISION_SCALE"}
}
func ErrInsufficientPoolFunds() *PluginError {
return &PluginError{Code: 144, Module: errModule, Msg: "insufficient funds in market pool"}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRAXIS ERRORS — PORS RESOLVER (160–179)
// ─────────────────────────────────────────────────────────────────────────────

func ErrResolverNotRegistered() *PluginError {
return &PluginError{Code: 160, Module: errModule, Msg: "resolver is not registered"}
}
func ErrResolverSuspended() *PluginError {
return &PluginError{Code: 161, Module: errModule, Msg: "resolver RRS score is below minimum threshold"}
}
func ErrNoResolverRegistered() *PluginError {
return &PluginError{Code: 162, Module: errModule, Msg: "no resolver registered for this market"}
}
func ErrAlreadyProposed() *PluginError {
return &PluginError{Code: 163, Module: errModule, Msg: "outcome already proposed for this market"}
}
func ErrInsufficientBond() *PluginError {
return &PluginError{Code: 164, Module: errModule, Msg: "proposal bond is below minimum required"}
}

// ─────────────────────────────────────────────────────────────────────────────
// PRAXIS ERRORS — PORS DISPUTE (180–199)
// ─────────────────────────────────────────────────────────────────────────────

func ErrDisputeWindowOpen() *PluginError {
return &PluginError{Code: 181, Module: errModule, Msg: "dispute window is still open — too early to finalize"}
}

func ErrDisputeWindowClosed() *PluginError {
return &PluginError{Code: 180, Module: errModule, Msg: "dispute window has closed"}
}
func ErrAlreadyDisputed() *PluginError {
return &PluginError{Code: 181, Module: errModule, Msg: "market is already disputed"}
}
func ErrNotDisputed() *PluginError {
return &PluginError{Code: 182, Module: errModule, Msg: "market is not in disputed state"}
}
func ErrNotAPanelMember() *PluginError {
return &PluginError{Code: 183, Module: errModule, Msg: "caller is not a panel member for this dispute"}
}
func ErrCommitPhaseOver() *PluginError {
return &PluginError{Code: 184, Module: errModule, Msg: "commit phase has ended"}
}
func ErrRevealPhaseNotOpen() *PluginError {
return &PluginError{Code: 185, Module: errModule, Msg: "reveal phase has not started yet"}
}
func ErrRevealPhaseOver() *PluginError {
return &PluginError{Code: 186, Module: errModule, Msg: "reveal phase has ended"}
}
func ErrCommitHashMismatch() *PluginError {
return &PluginError{Code: 187, Module: errModule, Msg: "revealed vote does not match committed hash"}
}
func ErrAlreadyCommitted() *PluginError {
return &PluginError{Code: 188, Module: errModule, Msg: "vote already committed"}
}
func ErrAlreadyRevealed() *PluginError {
return &PluginError{Code: 189, Module: errModule, Msg: "vote already revealed"}
}
func ErrTallyNotReady() *PluginError {
return &PluginError{Code: 190, Module: errModule, Msg: "reveal phase has not ended yet"}
}
func ErrAlreadyTallied() *PluginError {
return &PluginError{Code: 191, Module: errModule, Msg: "votes already tallied"}
}
func ErrNotFinalized() *PluginError {
return &PluginError{Code: 192, Module: errModule, Msg: "market has not been finalized"}
}
func ErrNoSlashToClaim() *PluginError {
return &PluginError{Code: 193, Module: errModule, Msg: "no slash proceeds to claim"}
}
func ErrInvalidCommitHash() *PluginError {
return &PluginError{Code: 194, Module: errModule, Msg: "commit hash must be exactly 32 bytes"}
}
func ErrInsufficientPanelCandidates() *PluginError {
return &PluginError{Code: 195, Module: errModule, Msg: "insufficient eligible panel candidates after position exclusion"}
}
func ErrMarketNotReclaimable() *PluginError {
return &PluginError{Code: 196, Module: errModule, Msg: "market is not eligible for stake reclaim"}
}
func ErrReclaimWindowClosed() *PluginError {
return &PluginError{Code: 197, Module: errModule, Msg: "reclaim window has not opened yet"}
}
func ErrNoStakeToReclaim() *PluginError {
return &PluginError{Code: 198, Module: errModule, Msg: "no stake or position to reclaim"}
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

// SafeMarshal wraps Marshal and returns a typed *PluginError on failure.
// Every marshal in a handler must use this — never discard marshal errors.
func SafeMarshal(m interface{}) ([]byte, *PluginError) {
b, err := Marshal(m)
if err != nil {
return nil, ErrMarshalFailed()
}
return b, nil
}

// errCheckWrite checks both the transport error and the embedded response error
// from a StateWrite call. Both must be checked — per the Canopy plugin spec.
func errCheckWrite(wr *PluginStateWriteResponse, err *PluginError) *PluginError {
if err != nil {
return err
}
if wr != nil && wr.Error != nil {
return wr.Error
}
return nil
}

// ErrCheckResp is a convenience wrapper for returning errors from CheckTx handlers.
func ErrResolverHasPosition() *PluginError {
return &PluginError{Code: 199, Module: errModule, Msg: "resolver holds a position in this market"}
}

func ErrCreatorCannotResolve() *PluginError {
return &PluginError{Code: 200, Module: errModule, Msg: "market creator cannot be the resolver for the same market"}
}

func ErrPositionCapExceeded() *PluginError {
return &PluginError{Code: 201, Module: errModule, Msg: "position would exceed per-address cap (20% of pool)"}
}
func ErrInsufficientResolverStake() *PluginError {
return &PluginError{Code: 202, Module: errModule, Msg: "resolver stake below minimum (500,000 PRX required)"}
}

func ErrCheckResp(err *PluginError) *PluginCheckResponse {
return &PluginCheckResponse{Error: err}
}

// ─────────────────────────────────────────────────────────────────────────────
// PLUGIN INFRASTRUCTURE ERRORS — required by plugin.go (never modify plugin.go)
// ─────────────────────────────────────────────────────────────────────────────

// Error() satisfies the error interface so plugin.go can call err.Error().
func (e *PluginError) Error() string {
if e == nil {
return ""
}
return e.Module + ": " + e.Msg
}

func ErrUnexpectedFSMToPlugin(t interface{}) *PluginError {
return &PluginError{Code: 7, Module: errModule, Msg: "unexpected FSM-to-plugin message type"}
}

func ErrInvalidFSMToPluginMMessage(t interface{}) *PluginError {
return &PluginError{Code: 8, Module: errModule, Msg: "invalid FSM-to-plugin message"}
}

func ErrInvalidPluginRespId() *PluginError {
return &PluginError{Code: 6, Module: errModule, Msg: "invalid plugin response id"}
}

func ErrFailedPluginWrite(err error) *PluginError {
return &PluginError{Code: 5, Module: errModule, Msg: "plugin socket write failed: " + err.Error()}
}

func ErrFailedPluginRead(err error) *PluginError {
return &PluginError{Code: 4, Module: errModule, Msg: "plugin socket read failed: " + err.Error()}
}

// ErrCorruptState is returned when a required state key exists but contains
// zero-length data — distinguishable from ErrMarketNotFound (key absent).
// Issue-13: improves debuggability of storage corruption scenarios.
func ErrCorruptState() *PluginError {
return &PluginError{Code: 4010, Msg: "state key exists but value is empty — possible storage corruption"}
}
