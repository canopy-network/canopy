package contract

// ---------------------------------------------------------------------------
// Pure custody arithmetic helpers.
//
// These do NOT read state, write state, emit events, or touch anything
// beyond the two struct pointers passed in. This is deliberate: deposit.go
// and withdraw.go have confirmed-different shapes for WHEN custody math
// runs, WHAT value feeds it (msg.Amount vs. a computed actualWithdrawn),
// and HOW the resulting write gets appended (a fixed Sets slice vs. a
// conditional writeReq with Deletes). Sharing the StateRead/StateWrite
// plumbing would force those genuinely different call sites into one
// artificial shape and risk silently changing behavior at one of them.
// Sharing ONLY the arithmetic -- the one piece that IS identical, and the
// one piece where a future silent divergence (e.g. one call site getting a
// checked-add and another getting a bare +=) would be a real bug -- changes
// nothing about protocol behavior versus each site writing this inline; it
// only removes the risk of the two copies drifting apart.
//
// Each function mutates its *Account/*Pool argument in place on success and
// returns nil, or leaves the argument UNCHANGED and returns a *PluginError
// on failure. Callers must check the returned error before using the
// mutated struct, exactly as every existing inline check in this codebase
// already does.
// ---------------------------------------------------------------------------

// debitAccountAmount subtracts amount from account.Amount. Compare-before-
// subtract (same discipline as PoolSub/AccountSub in fsm/account.go) --
// never underflow-then-check. Returns ErrInsufficientFunds() on shortfall.
func debitAccountAmount(account *Account, amount uint64) *PluginError {
	if account.Amount < amount {
		return ErrInsufficientFunds()
	}
	account.Amount -= amount
	return nil
}

// creditAccountAmount adds amount to account.Amount. Checked-add -- same
// discipline as every other accumulator in this codebase. Returns
// ErrAccountBalanceOverflow (already defined, contract/error.go code 225)
// on overflow.
func creditAccountAmount(addr []byte, account *Account, amount uint64) *PluginError {
	if amount > (^uint64(0) - account.Amount) {
		return ErrAccountBalanceOverflow(addr, account.Amount, amount)
	}
	account.Amount += amount
	return nil
}

// debitPoolAmount subtracts amount from pool.Amount. Compare-before-
// subtract, mirrors fsm/account.go's PoolSub() exactly. Returns
// ErrInsufficientFunds() on shortfall.
func debitPoolAmount(pool *Pool, amount uint64) *PluginError {
	if pool.Amount < amount {
		return ErrInsufficientFunds()
	}
	pool.Amount -= amount
	return nil
}

// creditPoolAmount adds amount to pool.Amount. Checked-add, mirrors
// fsm/account.go's PoolAdd() exactly -- NOT MintToPool(), which mints new
// supply. Returns ErrMarketPoolOverflow (new, code 230) on overflow.
func creditPoolAmount(marketId string, pool *Pool, amount uint64) *PluginError {
	if amount > (^uint64(0) - pool.Amount) {
		return ErrMarketPoolOverflow(marketId, pool.Amount, amount)
	}
	pool.Amount += amount
	return nil
}

// creditAssetBalanceAmount adds amount to bal.Amount ({37} AssetBalance).
// Checked-add, identical shape to creditPoolAmount above. Used exclusively
// by the devnet faucet (faucet.go) as of this addition -- deposit_collateral
// and other handlers do not yet read/write against AssetBalance (KNOWN GAP,
// see AssetBalance's own doc comment in arbor_state.proto).
func creditAssetBalanceAmount(assetID string, addr []byte, bal *AssetBalance, amount uint64) *PluginError {
	if amount > (^uint64(0) - bal.Amount) {
		return ErrAssetBalanceOverflow(assetID, addr, bal.Amount, amount)
	}
	bal.Amount += amount
	return nil
}
