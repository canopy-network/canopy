package canoliq

import (
	"math"

	"github.com/canopy-network/go-plugin/contract"
)

// safetychecks.go adds defensive range checks on amounts and fees, from a
// pinned mainnet height.
//
// Amounts and fees are uint64. From the activation height every handler
// rejects a transaction whose amount plus fee, or whose credit to a balance,
// pool or running total, would not fit in a uint64, instead of letting the
// sum wrap. The fee is also bounded by MaxTxFeeUcnpy. These checks mirror the
// ones in contract/contract.go's send handler.
//
// CheckTx runs during block application as well as in the mempool (see
// fsm/transaction.go), so a reject there is consensus-visible too. Both the
// Check and the Deliver checks are therefore gated, and every block before
// the activation height replays byte-identically (see
// mainnet_replay_parity_test.go).

// MainnetSafetyChecksHeight is the committee-29 block from which the range
// checks apply on mainnet. It must be above the height at which every
// committee-29 node runs this release; pick it with margin for
// pluginAutoUpdate.
const MainnetSafetyChecksHeight uint64 = 351_500

// MaxTxFeeUcnpy bounds the fee of a single canoLiq transaction once the
// checks are active: 1,000,000 CNPY, orders of magnitude above any configured
// minimum fee.
const MaxTxFeeUcnpy uint64 = 1_000_000_000_000

// safetyChecksActive reports whether the range checks apply at `height`.
// Other profiles have them from genesis.
func (c *Canoliq) safetyChecksActive(height uint64) bool {
	if c.Config.Profile != ProfileMainnet {
		return true
	}
	return height >= MainnetSafetyChecksHeight
}

// addOverflows reports whether a+b does not fit in a uint64.
func addOverflows(a, b uint64) bool { return a > math.MaxUint64-b }

// guardAdds returns ErrAmountOutOfRange when the checks are active and any
// (a, b) pair in `pairs` would overflow on a+b. `pairs` is a flat list of
// operands: a1, b1, a2, b2, ... Before activation it always returns nil.
func (c *Canoliq) guardAdds(pairs ...uint64) *contract.PluginError {
	if !c.safetyChecksActive(c.currentHeight()) {
		return nil
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if addOverflows(pairs[i], pairs[i+1]) {
			return ErrAmountOutOfRange()
		}
	}
	return nil
}

// exceedsCap reports whether pooled+amount > cap without computing the sum,
// so a large amount cannot wrap past the comparison.
func exceedsCap(pooled, amount, cap uint64) bool {
	return pooled > cap || amount > cap-pooled
}

// checkTxRanges is the stateless part of the checks, run from CheckTx: the
// fee is bounded and the message's amount plus the fee must fit in a uint64.
// Before activation it always returns nil.
func (c *Canoliq) checkTxRanges(msg any, fee uint64) *contract.PluginError {
	if !c.safetyChecksActive(c.currentHeight()) {
		return nil
	}
	if fee > MaxTxFeeUcnpy {
		return ErrFeeAboveMaximum()
	}
	var amount uint64
	switch x := msg.(type) {
	case *contract.MessageSend:
		amount = x.Amount
	case *contract.MessageCanoliqDeposit:
		amount = x.Amount
	case *contract.MessageCanoliqRedeem:
		amount = x.CcnpyAmount
	case *contract.MessageCanoliqTransfer:
		amount = x.Amount
	case *contract.MessageCPLQTransfer:
		amount = x.Amount
	case *contract.MessageCPLQStake:
		amount = x.Amount
	case *contract.MessageCPLQUnstake:
		amount = x.Amount
	case *contract.MessageOTCLockCreate:
		amount = x.CcnpyAmount
	}
	if addOverflows(amount, fee) {
		return ErrAmountOutOfRange()
	}
	return nil
}

// poolExceedsCap reports whether adding `amount` to `pooled` would exceed
// `cap`. Once the checks are active the comparison cannot wrap; before that
// it is the original sum-then-compare, kept byte-for-byte for replay.
func (c *Canoliq) poolExceedsCap(pooled, amount, cap uint64) bool {
	if c.safetyChecksActive(c.currentHeight()) {
		return exceedsCap(pooled, amount, cap)
	}
	return pooled+amount > cap
}
