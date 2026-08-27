package lib

const (
	// RLPIndicator is the legacy indicator for RLP-backed transactions.
	RLPIndicator = "RLP"
	// RLPV2Indicator identifies nonce-backed RLP transactions.
	RLPV2Indicator = "RLP.V2"
)

// IsRLPMemo reports whether the memo indicates an RLP-backed Ethereum transaction.
func IsRLPMemo(memo string) bool {
	return memo == RLPIndicator || memo == RLPV2Indicator
}
