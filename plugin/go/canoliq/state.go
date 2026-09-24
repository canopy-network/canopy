package canoliq

import (
	"bytes"
	"encoding/binary"
)

// State key layout for the canoLiq plugin.
//
// All keys live under prefix []byte{20} to stay clear of the Canopy core
// prefixes, which reserve the contiguous single-byte range 1-15 (accounts,
// pools, validators, ... supply=10; see fsm/key.go and lib.CoreReservedPrefixMax).
// Writing under a reserved prefix makes FSM.StateWrite panic, so this MUST stay
// outside 1-15. It is also declared in CanoliqConfig.CustomStatePrefixes so the
// handshake validates it at startup. Subdomains use single-byte discriminators
// inside JoinLenPrefix segments to keep keys compact and unambiguous.
var (
	canoliqPrefix = []byte{20}

	domainGlobals       = []byte{1}
	domainCcnpyBal      = []byte{2}
	domainCplqBal       = []byte{3}
	domainVesting       = []byte{4}
	domainVestIndex     = []byte{5}
	domainRedemption    = []byte{6}
	domainTreasury      = []byte{7}
	domainBuyback       = []byte{8}
	domainValIncent     = []byte{9}
	domainParams        = []byte{11}
	domainCplqStake     = []byte{12}
	domainCplqUnstaking = []byte{13}
	domainProposal      = []byte{14}
	domainVote          = []byte{15}
	domainBuybackOrder  = []byte{16}
	domainSpend         = []byte{17}
	domainMultisig      = []byte{18}
	domainInsurance     = []byte{19}
	domainStakeIndex    = []byte{20}
	domainRedeemIndex   = []byte{21}
	domainUnstakeIndex  = []byte{22}
	domainAlertState    = []byte{23}
	domainMatureRedeem  = []byte{24}
	domainEscrow        = []byte{25}
	domainTxFeeAccrual  = []byte{26}
	domainEjected       = []byte{27}
	domainOTCLock       = []byte{28}
	domainOTCLockIndex  = []byte{29}
	domainOTCBudget     = []byte{30}

	treasuryCanopy = []byte("canopy")
	treasuryCplq   = []byte("cplq")
	buybackPool    = []byte("pool")
	indexSingleton = []byte("index")
	insuranceSlot  = []byte("pool")
	otcAvailable   = []byte("available")
	otcReserved    = []byte("reserved")
)

// JoinLenPrefix mirrors contract.JoinLenPrefix to avoid an import cycle for
// trivial key-building. Each segment is encoded as 1-byte length + segment.
func JoinLenPrefix(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		if p != nil {
			total += 1 + len(p)
		}
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		if p == nil {
			continue
		}
		out = append(out, byte(len(p)))
		out = append(out, p...)
	}
	return out
}

// FormatUint64 returns the big-endian 8-byte encoding of n.
func FormatUint64(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// KeyForGlobals returns the singleton globals record key.
func KeyForGlobals() []byte {
	return JoinLenPrefix(canoliqPrefix, domainGlobals)
}

// KeyForParams returns the canoLiq parameters key.
func KeyForParams() []byte {
	return JoinLenPrefix(canoliqPrefix, domainParams)
}

// KeyForCCNPYBalance returns the cCNPY balance key for an address.
func KeyForCCNPYBalance(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainCcnpyBal, addr)
}

// KeyForCPLQBalance returns the liquid CPLQ balance key for an address.
func KeyForCPLQBalance(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainCplqBal, addr)
}

// KeyForVesting returns the vesting schedule key for an (address, schedule_id) pair.
func KeyForVesting(addr []byte, scheduleID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainVesting, addr, FormatUint64(scheduleID))
}

// KeyForVestingIndex returns the vesting index key listing schedule_ids per address.
func KeyForVestingIndex(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainVestIndex, addr)
}

// KeyForRedemption returns the redemption record key for an (address, redemption_id) pair.
func KeyForRedemption(addr []byte, redemptionID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainRedemption, addr, FormatUint64(redemptionID))
}

// KeyForRedemptionIndex returns the per-address redemption index key listing
// pending redemption ids.
func KeyForRedemptionIndex(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainRedeemIndex, addr)
}

// KeyForTreasuryCNPY returns the canoLiq DAO CNPY treasury key.
func KeyForTreasuryCNPY() []byte {
	return JoinLenPrefix(canoliqPrefix, domainTreasury, treasuryCanopy)
}

// KeyForTreasuryCPLQ returns the canoLiq DAO CPLQ treasury key.
func KeyForTreasuryCPLQ() []byte {
	return JoinLenPrefix(canoliqPrefix, domainTreasury, treasuryCplq)
}

// KeyForBuybackPool returns the buyback pool (CNPY held for CPLQ buyback) key.
func KeyForBuybackPool() []byte {
	return JoinLenPrefix(canoliqPrefix, domainBuyback, buybackPool)
}

// KeyForValidatorIncentives returns the per-validator infrastructure incentive key.
func KeyForValidatorIncentives(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainValIncent, addr)
}

// KeyForCPLQStake returns the active stake record key for an address.
func KeyForCPLQStake(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainCplqStake, addr)
}

// KeyForCPLQUnstaking returns the queued unstake record key for an
// (address, unstake_id) pair.
func KeyForCPLQUnstaking(addr []byte, unstakeID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainCplqUnstaking, addr, FormatUint64(unstakeID))
}

// KeyForUnstakingIndex returns the per-address unstake index key listing
// pending unstake ids.
func KeyForUnstakingIndex(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainUnstakeIndex, addr)
}

// removeUint64 returns a copy of s with the first occurrence of id removed.
// If id is not present the input is returned unchanged. Used to drop matured
// ids from per-address Redemption / Unstaking indexes.
func removeUint64(s []uint64, id uint64) []uint64 {
	for i, v := range s {
		if v == id {
			return append(s[:i:i], s[i+1:]...)
		}
	}
	return s
}

// KeyForCPLQStakeIndex returns the singleton key listing active staker addresses.
func KeyForCPLQStakeIndex() []byte {
	return JoinLenPrefix(canoliqPrefix, domainStakeIndex, indexSingleton)
}

// KeyForProposal returns the proposal record key for a proposal id.
func KeyForProposal(id uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainProposal, FormatUint64(id))
}

// KeyForProposalIndex returns the singleton key listing active proposal ids.
func KeyForProposalIndex() []byte {
	return JoinLenPrefix(canoliqPrefix, domainProposal, indexSingleton)
}

// KeyForVote returns the per-(proposal, voter) vote record key.
func KeyForVote(proposalID uint64, voter []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainVote, FormatUint64(proposalID), voter)
}

// KeyForBuybackOrder returns the buyback receipt key for a proposal id.
func KeyForBuybackOrder(proposalID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainBuybackOrder, FormatUint64(proposalID))
}

// KeyForTreasurySpend returns the treasury spend record key for a spend id.
func KeyForTreasurySpend(spendID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainSpend, FormatUint64(spendID))
}

// KeyForSpendIndex returns the singleton key listing pending spend ids.
func KeyForSpendIndex() []byte {
	return JoinLenPrefix(canoliqPrefix, domainSpend, indexSingleton)
}

// KeyForMultisigApproval returns the per-(spend_id, signer) approval key.
func KeyForMultisigApproval(spendID uint64, signer []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainMultisig, FormatUint64(spendID), signer)
}

// KeyForInsurancePool returns the insurance pool scalar key.
func KeyForInsurancePool() []byte {
	return JoinLenPrefix(canoliqPrefix, domainInsurance, insuranceSlot)
}

// KeyForValidatorRegistry returns the singleton validator stake registry key.
func KeyForValidatorRegistry() []byte {
	return JoinLenPrefix(canoliqPrefix, domainValIncent, indexSingleton)
}

// KeyForEjectedValidator returns the tombstone key marking a validator that
// governance ejected from the committee registry (F12). The value is the
// big-endian height of the ejection. The registry is reconciled against
// Canopy's live committee membership every block (see registry.go), so
// without a persisted tombstone the very next sync would re-admit an ejected
// operator and silently undo the passed proposal.
func KeyForEjectedValidator(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainEjected, addr)
}

// KeyForOTCLock returns the OTC lock position record key for an
// (address, lock_id) pair. Mirrors KeyForCPLQUnstaking: the position is a
// per-address record addressed by a globally monotonic id.
func KeyForOTCLock(addr []byte, lockID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainOTCLock, addr, FormatUint64(lockID))
}

// KeyForOTCLockIndex returns the per-address index key listing open lock ids
// so the account view can enumerate positions without a state-range scan.
func KeyForOTCLockIndex(addr []byte) []byte {
	return JoinLenPrefix(canoliqPrefix, domainOTCLockIndex, addr)
}

// KeyForOTCBudgetAvailable returns the scalar holding unreserved OTC program
// CPLQ. Funded only by a passed ProposalOTCProgramFund, which moves CPLQ out
// of treasury_cplq — nothing mints, so this is always a transfer from an
// existing allocation.
func KeyForOTCBudgetAvailable() []byte {
	return JoinLenPrefix(canoliqPrefix, domainOTCBudget, otcAvailable)
}

// KeyForOTCBudgetReserved returns the scalar holding CPLQ committed to open
// lock positions. available + reserved is the program's funded total; the
// split is what makes the cap hard, since a lock is rejected unless the
// reward can be moved from available to reserved up front. No matured
// position can therefore exceed what the program can pay.
func KeyForOTCBudgetReserved() []byte {
	return JoinLenPrefix(canoliqPrefix, domainOTCBudget, otcReserved)
}

// EjectedValidatorPrefix returns the prefix used to range-scan the ejection
// tombstones during the per-block registry sync.
func EjectedValidatorPrefix() []byte {
	return JoinLenPrefix(canoliqPrefix, domainEjected)
}

// KeyForAlertState returns the per-kind alert bookkeeping key (T6).
func KeyForAlertState(kind string) []byte {
	return JoinLenPrefix(canoliqPrefix, domainAlertState, []byte(kind))
}

// KeyForTxFeeAccrual returns the singleton scalar accumulating protocol tx-fee
// revenue collected since the last reward sweep (L3). ProcessRewards routes it
// straight to the DAO treasury — it is protocol revenue, not committee reward,
// so it is kept out of the 12% fee + 40/30/15/15 split applied to the observed
// stake-growth reward — then zeroes it. Bumped centrally in DeliverTx on each
// successful tx.
func KeyForTxFeeAccrual() []byte {
	return JoinLenPrefix(canoliqPrefix, domainTxFeeAccrual)
}

// KeyForEscrowPool returns the singleton CNPY escrow pool key. This pool holds
// the real CNPY backing live cCNPY plus pending redemptions — the H1 fix
// custodies deposited principal and the cCNPY-holder reward slice here, kept
// distinct from the committee fee pool (KeyForFeePool) that ProcessRewards
// sweeps. Invariant: escrow.Amount == TotalPooledCnpy + PendingRedemptionCnpy.
func KeyForEscrowPool() []byte {
	return JoinLenPrefix(canoliqPrefix, domainEscrow)
}

// KeyForMatureRedemption returns the global mature-redemption index entry for
// a queued redemption. Layout is `prefix | domainMatureRedeem | matureHeight |
// addr | redemptionID` so lexicographic order matches maturity order — a
// range scan up to current height efficiently lists all mature unclaimed
// redemptions for the stuck-redemption alert (T6 follow-up). Value is the
// presence marker `matureRedemptionMarker`; the full Redemption record
// remains at `KeyForRedemption(addr, id)`.
func KeyForMatureRedemption(matureHeight uint64, addr []byte, redemptionID uint64) []byte {
	return JoinLenPrefix(canoliqPrefix, domainMatureRedeem, FormatUint64(matureHeight), addr, FormatUint64(redemptionID))
}

// MatureRedemptionPrefix returns the prefix used to range-scan the
// mature-redemption index (for the stuck-redemption alert evaluator).
func MatureRedemptionPrefix() []byte {
	return JoinLenPrefix(canoliqPrefix, domainMatureRedeem)
}

// matureRedemptionMarker is a one-byte presence marker stored at each
// mature-redemption index key. Non-empty so StateRead can disambiguate
// "present with empty value" from "absent" — though for this index we only
// ever care about presence.
var matureRedemptionMarker = []byte{0x01}

// ParseMatureRedemptionHeight extracts the encoded `matureHeight` from a
// mature-redemption index key produced by KeyForMatureRedemption. Returns
// `(height, true)` on a recognised key; `(0, false)` when the key does not
// belong to the mature-redemption index. Used by the stuck-redemption alert
// evaluator to skip entries whose maturity has not yet arrived.
func ParseMatureRedemptionHeight(key []byte) (uint64, bool) {
	pre := MatureRedemptionPrefix()
	if len(key) < len(pre)+9 {
		return 0, false
	}
	if !bytes.Equal(key[:len(pre)], pre) {
		return 0, false
	}
	// After the prefix sits a length-prefixed 8-byte big-endian height.
	if key[len(pre)] != 8 {
		return 0, false
	}
	return binary.BigEndian.Uint64(key[len(pre)+1 : len(pre)+9]), true
}

// ParseEjectedValidator extracts the 20-byte validator address from an
// ejection tombstone key produced by KeyForEjectedValidator. Returns
// `(addr, true)` on a recognised key; `(nil, false)` otherwise.
func ParseEjectedValidator(key []byte) ([]byte, bool) {
	pre := EjectedValidatorPrefix()
	if len(key) != len(pre)+21 {
		return nil, false
	}
	if !bytes.Equal(key[:len(pre)], pre) {
		return nil, false
	}
	// After the prefix sits the length-prefixed 20-byte address.
	if key[len(pre)] != 20 {
		return nil, false
	}
	return key[len(pre)+1:], true
}

// EncodeUint64 returns the 8-byte big-endian encoding of n. Used for storing
// scalar uint64 values directly under their key.
func EncodeUint64(n uint64) []byte {
	return FormatUint64(n)
}

// DecodeUint64 parses an 8-byte big-endian uint64. Returns 0 for nil/short input
// so unset keys read as zero.
func DecodeUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}
