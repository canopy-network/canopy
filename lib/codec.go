package lib

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// MarshalAnypbJSON() marshals an anypb to JSON with protobuf bytes fields represented as hex.
func MarshalAnypbJSON(any *anypb.Any) (json.RawMessage, error) {
	if any == nil {
		return nil, nil
	}
	// Native transaction codecs already implement the global hex contract and
	// also retain message-specific representations for non-bytes fields.
	payload, payloadErr := FromAny(any)
	if payloadErr == nil {
		if msgI, ok := payload.(MessageI); ok {
			msg, err := MarshalJSON(msgI)
			return msg, err
		}
		jsonBytes, err := protojson.MarshalOptions{}.Marshal(payload)
		if err == nil {
			return transformProtoJSONBytes(jsonBytes, payload.ProtoReflect().Descriptor(), base64ToHex)
		}
	}
	// Dynamically registered plugin messages are not available through the
	// process-wide protobuf type registry, so resolve their descriptors here.
	desc := messageDescriptorForTypeURL(any.TypeUrl)
	if desc != nil && len(any.Value) > 0 {
		dynamic := dynamicpb.NewMessage(desc)
		if err := proto.Unmarshal(any.Value, dynamic); err == nil {
			jsonBytes, e := protojson.MarshalOptions{}.Marshal(dynamic)
			if e == nil {
				return transformProtoJSONBytes(jsonBytes, desc, base64ToHex)
			}
		}
	}
	// exit
	return nil, fmt.Errorf("unable to marshal any payload type %s to json", any.TypeUrl)
}

// MarshalAnyProtoJSON() converts an 'any proto' into JSON
func MarshalAnyProtoJSON(any *anypb.Any) (json.RawMessage, error) {
	if any == nil {
		return nil, nil
	}
	jsonBytes, err := protojson.MarshalOptions{}.Marshal(any)
	if err != nil {
		return nil, ErrJSONMarshal(err)
	}
	return jsonBytes, nil
}

// AnyFromProtoJSON() converts JSON into a protobuf.Any
func AnyFromProtoJSON(msg json.RawMessage) (a *anypb.Any, e ErrorI) {
	if len(msg) == 0 {
		return nil, ErrJSONUnmarshal(fmt.Errorf("empty json payload"))
	}
	a = new(anypb.Any)
	if err := protojson.Unmarshal(msg, a); err != nil {
		return nil, ErrJSONUnmarshal(err)
	}
	return a, nil
}

// AnyFromJSONForMessageType() converts JSON into anypb
func AnyFromJSONForMessageType(messageType string, msg json.RawMessage) (*anypb.Any, ErrorI) {
	if messageType == "" {
		return nil, ErrUnknownMessageName(messageType)
	}
	desc := globalPluginSchemaRegistry.FindMessageDescriptorForMessageType(messageType)
	typeURL := messageType
	if strings.Contains(messageType, "/") {
		desc = messageDescriptorForTypeURL(messageType)
	} else if desc != nil {
		typeURL = "type.googleapis.com/" + string(desc.FullName())
	}
	if desc == nil {
		return nil, ErrUnknownMessageName(messageType)
	}
	dynamic := dynamicpb.NewMessage(desc)
	protoJSON, err := transformProtoJSONBytes(msg, desc, hexToBase64)
	if err != nil {
		return nil, ErrJSONUnmarshal(err)
	}
	if err = protojson.Unmarshal(protoJSON, dynamic); err != nil {
		return nil, ErrJSONUnmarshal(err)
	}
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(dynamic)
	if err != nil {
		return nil, ErrToAny(err)
	}
	return &anypb.Any{TypeUrl: typeURL, Value: bz}, nil
}

func messageDescriptorForTypeURL(typeURL string) protoreflect.MessageDescriptor {
	if desc := globalPluginSchemaRegistry.FindMessageDescriptorForTypeURL(typeURL); desc != nil {
		return desc
	}
	messageType, err := protoregistry.GlobalTypes.FindMessageByURL(typeURL)
	if err != nil {
		return nil
	}
	return messageType.Descriptor()
}

// transformProtoJSONBytes changes the JSON representation of every protobuf bytes field.
func transformProtoJSONBytes(jsonBytes []byte, desc protoreflect.MessageDescriptor, transform func(string) (string, error)) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("invalid JSON after plugin message")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("protobuf message JSON must be an object")
	}
	if err := transformProtoJSONMessage(object, desc, transform); err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

func transformProtoJSONMessage(object map[string]any, desc protoreflect.MessageDescriptor, transform func(string) (string, error)) error {
	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := field.JSONName()
		value, found := object[name]
		if !found {
			name = string(field.Name())
			value, found = object[name]
		}
		if !found {
			continue
		}
		transformed, err := transformProtoJSONField(value, field, transform)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.FullName(), err)
		}
		object[name] = transformed
	}
	return nil
}

func transformProtoJSONField(value any, field protoreflect.FieldDescriptor, transform func(string) (string, error)) (any, error) {
	if field.IsMap() {
		object, ok := value.(map[string]any)
		if !ok {
			return value, nil
		}
		for key, entry := range object {
			transformed, err := transformProtoJSONScalar(entry, field.MapValue(), transform)
			if err != nil {
				return nil, err
			}
			object[key] = transformed
		}
		return object, nil
	}
	if field.IsList() {
		list, ok := value.([]any)
		if !ok {
			return value, nil
		}
		for i, entry := range list {
			transformed, err := transformProtoJSONScalar(entry, field, transform)
			if err != nil {
				return nil, err
			}
			list[i] = transformed
		}
		return list, nil
	}
	return transformProtoJSONScalar(value, field, transform)
}

func transformProtoJSONScalar(value any, field protoreflect.FieldDescriptor, transform func(string) (string, error)) (any, error) {
	if field.Kind() == protoreflect.BytesKind {
		text, ok := value.(string)
		if !ok {
			return value, nil
		}
		return transform(text)
	}
	if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
		object, ok := value.(map[string]any)
		if ok {
			return object, transformProtoJSONMessage(object, field.Message(), transform)
		}
	}
	return value, nil
}

func hexToBase64(value string) (string, error) {
	bz, err := StringToBytes(value)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bz), nil
}

func base64ToHex(value string) (string, error) {
	bz, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return BytesToString(bz), nil
}

var globalPluginSchemaRegistry = NewPluginSchemaRegistry()

// PluginSchemaRegistry() acts as a global registry for plugin proto schemas to implement the json.Marshal interface
type PluginSchemaRegistry struct {
	mu                   sync.RWMutex
	byFullyQualifiedName map[string]protoreflect.MessageDescriptor
	byTypeURL            map[string]protoreflect.MessageDescriptor
	byCommonMessageName  map[string]protoreflect.MessageDescriptor
}

// NewPluginSchemaRegistry()
func NewPluginSchemaRegistry() *PluginSchemaRegistry {
	return &PluginSchemaRegistry{
		byFullyQualifiedName: make(map[string]protoreflect.MessageDescriptor),
		byTypeURL:            make(map[string]protoreflect.MessageDescriptor),
		byCommonMessageName:  make(map[string]protoreflect.MessageDescriptor),
	}
}

// Register() registers a plugin with the global schema registry
func (r *PluginSchemaRegistry) Register(config *PluginConfig) ErrorI {
	if config == nil || len(config.FileDescriptorProtos) == 0 {
		return nil
	}

	// Unmarshal the FileDescriptorProtos
	fileProtos := make([]*descriptorpb.FileDescriptorProto, 0, len(config.FileDescriptorProtos))
	for _, bz := range config.FileDescriptorProtos {
		fd := new(descriptorpb.FileDescriptorProto)
		if err := proto.Unmarshal(bz, fd); err != nil {
			return ErrInvalidPluginSchema(err)
		}
		fileProtos = append(fileProtos, fd)
	}

	// Unmarshal the file protos into a 'proto.Files' object
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: fileProtos})
	if err != nil {
		return ErrInvalidPluginSchema(err)
	}

	byFullyQualifiedName := make(map[string]protoreflect.MessageDescriptor)
	byTypeURL := make(map[string]protoreflect.MessageDescriptor)
	byCommonMessageName := make(map[string]protoreflect.MessageDescriptor)

	// for each transaction type URL - register it
	for _, typeURL := range config.TransactionTypeUrls {
		name := typeURL
		if idx := strings.LastIndex(typeURL, "/"); idx >= 0 {
			name = typeURL[idx+1:]
		}
		if name == "" {
			return ErrInvalidPluginSchema(fmt.Errorf("empty message name in type url %q", typeURL))
		}
		desc, e := files.FindDescriptorByName(protoreflect.FullName(name))
		if e != nil {
			return ErrInvalidPluginSchema(fmt.Errorf("message %s: %w", name, e))
		}
		md, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return ErrInvalidPluginSchema(fmt.Errorf("descriptor %s is not a message", name))
		}
		byFullyQualifiedName[name] = md
		byTypeURL[typeURL] = md
	}

	// for each transaction (message) common name - register it
	if len(config.SupportedTransactions) > 0 {
		if len(config.SupportedTransactions) != len(config.TransactionTypeUrls) {
			return ErrInvalidPluginSchema(fmt.Errorf("supported transactions count %d does not match transaction type urls count %d", len(config.SupportedTransactions), len(config.TransactionTypeUrls)))
		}
		for i, messageType := range config.SupportedTransactions {
			md := byTypeURL[config.TransactionTypeUrls[i]]
			if md == nil {
				return ErrInvalidPluginSchema(fmt.Errorf("transaction type url %s not found for type %s", config.TransactionTypeUrls[i], messageType))
			}
			byCommonMessageName[messageType] = md
		}
	}

	// for each event type URL - register it
	for _, typeURL := range config.EventTypeUrls {
		name := typeURL
		if idx := strings.LastIndex(typeURL, "/"); idx >= 0 {
			name = typeURL[idx+1:]
		}
		if name == "" {
			return ErrInvalidPluginSchema(fmt.Errorf("empty message name in type url %q", typeURL))
		}
		desc, e := files.FindDescriptorByName(protoreflect.FullName(name))
		if e != nil {
			return ErrInvalidPluginSchema(fmt.Errorf("message %s: %w", name, e))
		}
		md, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return ErrInvalidPluginSchema(fmt.Errorf("descriptor %s is not a message", name))
		}
		byFullyQualifiedName[name] = md
		byTypeURL[typeURL] = md
	}

	r.mu.Lock()
	r.byFullyQualifiedName = byFullyQualifiedName
	r.byTypeURL = byTypeURL
	r.byCommonMessageName = byCommonMessageName
	r.mu.Unlock()

	return nil
}

// FindMessageDescriptorForTypeURL() uses the formal type url to ID the proto message descriptor
func (r *PluginSchemaRegistry) FindMessageDescriptorForTypeURL(typeURL string) protoreflect.MessageDescriptor {
	if typeURL == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if desc, ok := r.byTypeURL[typeURL]; ok {
		return desc
	}
	name := typeURL
	if idx := strings.LastIndex(typeURL, "/"); idx >= 0 {
		name = typeURL[idx+1:]
	}
	return r.byFullyQualifiedName[name]
}

// FindMessageDescriptorForMessageType() uses the short message name (common name) to ID the proto message descriptor
func (r *PluginSchemaRegistry) FindMessageDescriptorForMessageType(messageType string) protoreflect.MessageDescriptor {
	if messageType == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byCommonMessageName[messageType]
}
