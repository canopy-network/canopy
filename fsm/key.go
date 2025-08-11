package fsm

import (
	"encoding/binary"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
)

// ReservedIds ensures Validators can't stake for 'reserved ids'
var ReservedIDs = []uint64{
	lib.UnknownChainId,
	lib.DAOPoolID, // NOTE: DAOPoolId cannot be staked for as the max chain Id is the EscrowPoolAddend
}

// EscrowPoolAddend is used to translate a chainId into the id for the 'swap' escrow pool
// Example: pools[chainId] -> stake pool && pools[chainId+EscrowPoolAddend] -> escrow pool for token swaps

var (
	MaxChainId          = math.MaxUint16 / 4
	HoldingPoolAddend   = 1 * math.MaxUint16 / 4
	LiquidityPoolAddend = 2 * math.MaxUint16 / 4
	Unused1PoolAddend   = 3 * math.MaxUint16 / 4
	EscrowPoolAddend    = 4 * math.MaxUint16 / 4
	Unused2PoolAddend   = 5 * math.MaxUint16 / 4
	Unused3PoolAddend   = 6 * math.MaxUint16 / 4
	Unused4PoolAddend   = 7 * math.MaxUint16 / 4
)

var (
	accountPrefix          = []byte{1}  // store key prefix for accounts
	poolPrefix             = []byte{2}  // store key prefix for pools
	validatorPrefix        = []byte{3}  // store key prefix for validators
	committeePrefix        = []byte{4}  // store key prefix for validators in committees
	unstakePrefix          = []byte{5}  // store key prefix for validators currently unstaking
	pausedPrefix           = []byte{6}  // store key prefix for validators currently paused
	paramsPrefix           = []byte{7}  // store key prefix for governance parameters
	nonSignerPrefix        = []byte{8}  // store key prefix for validators who have missed signing QCs
	lastProposersPrefix    = []byte{9}  // store key prefix for the last proposers
	supplyPrefix           = []byte{10} // store key prefix for the supply count
	delegatePrefix         = []byte{11} // store key prefix for the validators who are delegating for committees
	committeesDataPrefix   = []byte{12} // store key prefix for Quorum Certificate proposals before they are paid
	orderBookPrefix        = []byte{13} // store key prefix for 'sell orders' before they are bid on
	retiredCommitteePrefix = []byte{14} // store key prefix for 'retired' (dead) committees
)

/*
- Prefixes are used to allow 'grouping' and organization in a schemaless key-value database environment

- Segments are used to allow 'sub-groupings' ex. CommitteeKey allows iteration over all committees (CommitteePrefix)
or a specific committee (CommitteePrefix+CommitteeId)

- Iterating over a prefix enables operations over groups of similar datastructures (accounts, validators, params etc.)

- Length prefixed append is used to be able to easily separate the segments of a key

- BigEndianEncoding is used for uint64 to accommodate the 'lexicographical' sorting nature of the key-value database
*/
func AccountPrefix() []byte             { return lib.JoinLenPrefix(accountPrefix) }
func PoolPrefix() []byte                { return lib.JoinLenPrefix(poolPrefix) }
func SupplyPrefix() []byte              { return lib.JoinLenPrefix(supplyPrefix) }
func ValidatorPrefix() []byte           { return lib.JoinLenPrefix(validatorPrefix) }
func NonSignerPrefix() []byte           { return lib.JoinLenPrefix(nonSignerPrefix) }
func UnstakingPrefix(h uint64) []byte   { return lib.JoinLenPrefix(unstakePrefix, formatUint64(h)) }
func PausedPrefix(height uint64) []byte { return lib.JoinLenPrefix(pausedPrefix, formatUint64(height)) }
func LastProposersPrefix() []byte       { return lib.JoinLenPrefix(lastProposersPrefix) }
func CommitteePrefix(id uint64) []byte  { return lib.JoinLenPrefix(committeePrefix, formatUint64(id)) }
func DelegatePrefix(id uint64) []byte   { return lib.JoinLenPrefix(delegatePrefix, formatUint64(id)) }
func CommitteesDataPrefix() []byte      { return lib.JoinLenPrefix(committeesDataPrefix) }
func RetiredCommitteesPrefix() []byte   { return lib.JoinLenPrefix(retiredCommitteePrefix) }
func KeyForPool(n uint64) []byte        { return lib.JoinLenPrefix(poolPrefix, formatUint64(n)) }
func KeyForNonSigner(a []byte) []byte   { return lib.JoinLenPrefix(nonSignerPrefix, a) }
func OrderBookPrefix(cId uint64) []byte { return lib.JoinLenPrefix(orderBookPrefix, formatUint64(cId)) }
func KeyForOrder(chainId uint64, orderId []byte) []byte {
	return append(OrderBookPrefix(chainId), lib.JoinLenPrefix(orderId)...)
}
func KeyForUnstaking(height uint64, address crypto.AddressI) []byte {
	return append(UnstakingPrefix(height), lib.JoinLenPrefix(address.Bytes())...)
}
func KeyForPaused(maxPausedHeight uint64, address crypto.AddressI) []byte {
	return append(PausedPrefix(maxPausedHeight), lib.JoinLenPrefix(address.Bytes())...)
}
func KeyForCommittee(chainId uint64, addr crypto.AddressI, stake uint64) []byte {
	return append(CommitteePrefix(chainId), lib.JoinLenPrefix(formatUint64(stake), addr.Bytes())...)
}
func KeyForDelegate(chainId uint64, addr crypto.AddressI, stake uint64) []byte {
	return append(DelegatePrefix(chainId), lib.JoinLenPrefix(formatUint64(stake), addr.Bytes())...)
}
func KeyForRetiredCommittee(cId uint64) []byte {
	return lib.JoinLenPrefix(retiredCommitteePrefix, formatUint64(cId))
}
func KeyForAccount(addr crypto.AddressI) []byte {
	return lib.JoinLenPrefix(accountPrefix, addr.Bytes())
}
func KeyForValidator(addr crypto.AddressI) []byte {
	return lib.JoinLenPrefix(validatorPrefix, addr.Bytes())
}
func KeyForParams(s string) []byte {
	return lib.JoinLenPrefix(paramsPrefix, []byte(prefixForParamSpace(s)))
}

func AddressFromKey(k []byte) (crypto.AddressI, lib.ErrorI) {
	segments := lib.DecodeLengthPrefixed(k)
	return crypto.NewAddressFromBytes(segments[len(segments)-1]), nil
}

func IdFromKey(k []byte) (uint64, lib.ErrorI) {
	segments := lib.DecodeLengthPrefixed(k)
	if len(segments) < 2 {
		return 0, ErrInvalidKey(k)
	}
	return binary.BigEndian.Uint64(segments[1]), nil
}

func formatUint64(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}
