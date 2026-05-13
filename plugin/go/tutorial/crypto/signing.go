package crypto

import (
	"github.com/canopy-network/go-plugin/tutorial/contract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// GetSignBytes returns the canonical bytes for signing a transaction
// This uses the contract.Transaction proto which must match lib.Transaction exactly
func GetSignBytes(msgType string, msg *anypb.Any, time, createdHeight, fee uint64, memo string, networkID, chainID uint64) ([]byte, error) {
	// Create a Transaction with all fields EXCEPT signature (nil for signing)
	tx := &contract.Transaction{
		MessageType:   msgType,
		Msg:           msg,
		Signature:     nil, // Omitted for sign bytes
		CreatedHeight: createdHeight,
		Time:          time,
		Fee:           fee,
		Memo:          memo,
		NetworkId:     networkID,
		ChainId:       chainID,
	}

	// Use deterministic marshaling to match the server's GetSignBytes
	return proto.MarshalOptions{Deterministic: true}.Marshal(tx)
}
