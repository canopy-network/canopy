package contract

// ─────────────────────────────────────────────────────────────────────────────
// PRIS STATE KEYS (0x1D – 0x27)
// Spec authority: PRIS v1.0-r3
//
//   0x1D  CreatorFeePool      per market_id
//   0x1E  ResolverFeePool     per market_id
//   0x1F  TreasuryPool        singleton global accumulator
//   0x20  EpochSnapshot       per epoch number
//   0x21  ResolverEpochPool   per epoch number
//   0x22  BuilderPool         singleton
//   0x23  CommunityPool       singleton
//   0x24  InvestorPool        singleton
//   0x25  ProtocolPool        singleton
//   0x26  BuilderLastClaimed  singleton
//   0x27  InvestorLastClaimed singleton
// ─────────────────────────────────────────────────────────────────────────────

var (
creatorFeePoolPrefix      = []byte{0x1D}
resolverFeePoolPrefix     = []byte{0x1E}
treasuryPoolPrefix        = []byte{0x1F}
epochSnapshotPrefix       = []byte{0x20}
resolverEpochPoolPrefix   = []byte{0x21}
builderPoolPrefix         = []byte{0x22}
communityPoolPrefix       = []byte{0x23}
investorPoolPrefix        = []byte{0x24}
protocolPoolPrefix        = []byte{0x25}
builderLastClaimedPrefix  = []byte{0x26}
investorLastClaimedPrefix = []byte{0x27}
)

func KeyForCreatorFeePool(marketId []byte) []byte {
return JoinLenPrefix(creatorFeePoolPrefix, marketId)
}
func KeyForResolverFeePool(marketId []byte) []byte {
return JoinLenPrefix(resolverFeePoolPrefix, marketId)
}
func KeyForEpochSnapshot(epoch uint64) []byte {
return JoinLenPrefix(epochSnapshotPrefix, uint64ToBytes(epoch))
}
func KeyForResolverEpochPool(epoch uint64) []byte {
return JoinLenPrefix(resolverEpochPoolPrefix, uint64ToBytes(epoch))
}
func KeyForBuilderPool() []byte {
return JoinLenPrefix(builderPoolPrefix, []byte("/builder/"))
}
func KeyForCommunityPool() []byte {
return JoinLenPrefix(communityPoolPrefix, []byte("/community/"))
}
func KeyForInvestorPool() []byte {
return JoinLenPrefix(investorPoolPrefix, []byte("/investor/"))
}
func KeyForProtocolPool() []byte {
return JoinLenPrefix(protocolPoolPrefix, []byte("/protocol/"))
}
func KeyForBuilderLastClaimed() []byte {
return JoinLenPrefix(builderLastClaimedPrefix, []byte("/blc/"))
}
func KeyForInvestorLastClaimed() []byte {
return JoinLenPrefix(investorLastClaimedPrefix, []byte("/ilc/"))
}

var globalStatsPrefix     = []byte{0x28}
var unbondingRecordPrefix = []byte{0x29}

// KeyForUnbondingRecord returns the singleton unbonding record for a resolver.
func KeyForUnbondingRecord(addr []byte) []byte {
	return JoinLenPrefix(unbondingRecordPrefix, addr)
}

// KeyForGlobalStats returns the singleton state key for protocol-wide resolution stats.
func KeyForGlobalStats() []byte {
return JoinLenPrefix(globalStatsPrefix, []byte("/stats/"))
}
