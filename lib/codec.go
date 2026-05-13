package lib

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// MarshalAnypbJSON() marshals an anypb to JSON using the globalPluginSchemaRegistry
func MarshalAnypbJSON(any *anypb.Any) (json.RawMessage, error) {
	if any == nil {
		return nil, nil
	}
	// try with the globalPluginSchemaRegistry first
	desc := globalPluginSchemaRegistry.FindMessageDescriptorForTypeURL(any.TypeUrl)
	if desc != nil && len(any.Value) > 0 {
		dynamic := dynamicpb.NewMessage(desc)
		if err := proto.Unmarshal(any.Value, dynamic); err == nil {
			jsonBytes, e := protojson.MarshalOptions{}.Marshal(dynamic)
			if e == nil {
				return jsonBytes, nil
			}
		}
	}
	// fallback to standard anypb.UnmarshalNew()
	payload, payloadErr := FromAny(any)
	if payloadErr == nil {
		if msgI, ok := payload.(MessageI); ok {
			msg, err := MarshalJSON(msgI)
			if err == nil {
				return msg, nil
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
		desc = globalPluginSchemaRegistry.FindMessageDescriptorForTypeURL(messageType)
	} else if desc != nil {
		typeURL = "type.googleapis.com/" + string(desc.FullName())
	}
	if desc == nil {
		return nil, ErrUnknownMessageName(messageType)
	}
	dynamic := dynamicpb.NewMessage(desc)
	if err := protojson.Unmarshal(msg, dynamic); err != nil {
		return nil, ErrJSONUnmarshal(err)
	}
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(dynamic)
	if err != nil {
		return nil, ErrToAny(err)
	}
	return &anypb.Any{TypeUrl: typeURL, Value: bz}, nil
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
