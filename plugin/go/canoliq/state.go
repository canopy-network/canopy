package canoliq

import (
	"encoding/binary"
)

// State key layout for the canoLiq plugin.
//
// All keys live under prefix []byte{10} to stay clear of the Canopy core
// prefixes documented in plugin/go/AGENTS.md (1=accounts, 2=pools, 7=gov).
// Subdomains use single-byte discriminators inside JoinLenPrefix segments to
// keep keys compact and unambiguous.
var (
	canoliqPrefix = []byte{10}

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
	domainEscrow        = []byte{24}
	domainTxFeeAccrual  = []byte{25}

	treasuryCanopy = []byte("canopy")
	treasuryCplq   = []byte("cplq")
	buybackPool    = []byte("pool")
	indexSingleton = []byte("index")
	insuranceSlot  = []byte("pool")
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

// KeyForAlertState returns the per-kind alert bookkeeping key (T6).
func KeyForAlertState(kind string) []byte {
	return JoinLenPrefix(canoliqPrefix, domainAlertState, []byte(kind))
}

// KeyForTxFeeAccrual returns the singleton scalar accumulating protocol tx-fee
// revenue collected into the committee pool since the last reward sweep (L3).
// ProcessRewards subtracts it from the reward delta (so tx fees are not
// distributed as staking reward) and routes it to the DAO treasury, then zeroes
// it. Bumped centrally in DeliverTx on each successful tx.
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
