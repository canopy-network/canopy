package fsm

import (
	"encoding/json"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGlobalHexPreservesNativeTransactionJSON(t *testing.T) {
	t.Run("stake field names", func(t *testing.T) {
		message := &MessageStake{
			PublicKey:     []byte{0x01, 0x02},
			Amount:        10,
			Committees:    []uint64{1},
			NetAddress:    "tcp://example.com",
			OutputAddress: []byte{0x03, 0x04},
		}
		anyMessage, err := lib.NewAny(message)
		require.NoError(t, err)

		encoded, marshalErr := json.Marshal(&lib.Transaction{MessageType: MessageStakeName, Msg: anyMessage})
		require.NoError(t, marshalErr)
		require.Contains(t, string(encoded), `"publickey":"0102"`)
		require.NotContains(t, string(encoded), `"publicKey"`)

		decoded := new(lib.Transaction)
		require.NoError(t, json.Unmarshal(encoded, decoded))
		decodedMessage := new(MessageStake)
		require.NoError(t, decoded.Msg.UnmarshalTo(decodedMessage))
		require.True(t, proto.Equal(message, decodedMessage))
	})

	t.Run("change parameter value", func(t *testing.T) {
		parameterValue, err := lib.NewAny(&lib.StringWrapper{Value: "v3.2"})
		require.NoError(t, err)
		message := &MessageChangeParameter{
			ParameterSpace: "cons",
			ParameterKey:   ParamProtocolVersion,
			ParameterValue: parameterValue,
			StartHeight:    100,
			EndHeight:      200,
			Signer:         []byte{0x05, 0x06},
		}
		anyMessage, err := lib.NewAny(message)
		require.NoError(t, err)

		encoded, marshalErr := json.Marshal(&lib.Transaction{MessageType: MessageChangeParameterName, Msg: anyMessage})
		require.NoError(t, marshalErr)
		require.Contains(t, string(encoded), `"parameterValue":"v3.2"`)
		require.Contains(t, string(encoded), `"signer":"0506"`)

		decoded := new(lib.Transaction)
		require.NoError(t, json.Unmarshal(encoded, decoded))
		decodedMessage := new(MessageChangeParameter)
		require.NoError(t, decoded.Msg.UnmarshalTo(decodedMessage))
		decodedValue := new(lib.StringWrapper)
		require.NoError(t, decodedMessage.ParameterValue.UnmarshalTo(decodedValue))
		require.Equal(t, "v3.2", decodedValue.Value)
	})
}
