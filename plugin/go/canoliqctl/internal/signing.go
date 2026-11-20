package internal

import (
	"github.com/canopy-network/go-plugin/contract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// SignBytes returns the canonical bytes the FSM hashes for signature
// verification. The shape mirrors lib.Transaction with Signature omitted.
// Deterministic marshaling is required so client and server agree byte-for-byte.
func SignBytes(msgType string, msg *anypb.Any, time, createdHeight, fee uint64, memo string, networkID, chainID uint64) ([]byte, error) {
	tx := &contract.Transaction{
		MessageType:   msgType,
		Msg:           msg,
		Signature:     nil,
		CreatedHeight: createdHeight,
		Time:          time,
		Fee:           fee,
		Memo:          memo,
		NetworkId:     networkID,
		ChainId:       chainID,
	}
	return proto.MarshalOptions{Deterministic: true}.Marshal(tx)
}
