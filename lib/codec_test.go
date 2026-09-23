package lib

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPluginJSONBytesUseHex(t *testing.T) {
	originalRegistry := globalPluginSchemaRegistry
	globalPluginSchemaRegistry = NewPluginSchemaRegistry()
	defer func() { globalPluginSchemaRegistry = originalRegistry }()

	desc := (&PluginConfig{}).ProtoReflect().Descriptor()
	typeURL := "type.googleapis.com/" + string(desc.FullName())
	globalPluginSchemaRegistry.byCommonMessageName["plugin-config"] = desc
	globalPluginSchemaRegistry.byTypeURL[typeURL] = desc

	expected := []byte{0x00, 0x11, 0xaa, 0xff}
	input := json.RawMessage(`{"name":"test","fileDescriptorProtos":["0011aaff"]}`)
	message, err := AnyFromJSONForMessageType("plugin-config", input)
	require.NoError(t, err)

	decoded := new(PluginConfig)
	require.NoError(t, message.UnmarshalTo(decoded))
	require.Equal(t, [][]byte{expected}, decoded.FileDescriptorProtos)
	expectedBytes, marshalProtoErr := proto.MarshalOptions{Deterministic: true}.Marshal(&PluginConfig{
		Name:                 "test",
		FileDescriptorProtos: [][]byte{expected},
	})
	require.NoError(t, marshalProtoErr)
	require.Equal(t, expectedBytes, message.Value)

	output, marshalErr := MarshalAnypbJSON(message)
	require.NoError(t, marshalErr)
	require.JSONEq(t, string(input), string(output))
}

func TestTransactionJSONUsesHexForPluginBytes(t *testing.T) {
	originalRegistry := globalPluginSchemaRegistry
	globalPluginSchemaRegistry = NewPluginSchemaRegistry()
	defer func() { globalPluginSchemaRegistry = originalRegistry }()

	desc := (&Signature{}).ProtoReflect().Descriptor()
	globalPluginSchemaRegistry.byCommonMessageName[testMessageName] = desc

	expected := bytes.Repeat([]byte{0xab}, 20)
	jsonBytes := []byte(`{"type":"signature","msg":{"publicKey":"` + BytesToString(expected) + `"}}`)
	transaction := new(Transaction)
	require.NoError(t, json.Unmarshal(jsonBytes, transaction))

	message := new(Signature)
	require.NoError(t, transaction.Msg.UnmarshalTo(message))
	require.Equal(t, expected, message.PublicKey)
}

func TestPluginJSONBytesRejectInvalidHex(t *testing.T) {
	desc := (&Signature{}).ProtoReflect().Descriptor()
	_, err := transformPluginJSONBytes([]byte(`{"publicKey":"not-hex"}`), desc, hexToBase64)
	require.Error(t, err)
}
