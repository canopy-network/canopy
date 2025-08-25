package codec

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// BinaryCodec is an interface model that defines the requirements for binary encoding and decoding
// A binary encoder converts data into a compact, non-human-readable binary format, which is highly
// efficient in terms of both storage size and speed for serialization and deserialization
type BinaryCodec interface {
	Marshal(message any) ([]byte, error)
	Unmarshal(data []byte, ptr any) error
	ToAny(proto.Message) (*anypb.Any, error)
	FromAny(*anypb.Any) (proto.Message, error)
}

// JSONCodec is an interface model that defines requirements for json encoding and decoding
// converts data into a human-readable text format (JSON) which can be more friendly but less performant
type JSONCodec interface {
	json.Marshaler
	json.Unmarshaler
}

// ensure the protobuf codec implements the BinaryCodec interface
var _ BinaryCodec = &Protobuf{}

// Protobuf is an encoding implementation for protobuf
type Protobuf struct{}

// Marshal() converts a message to bytes
func (p *Protobuf) Marshal(message any) ([]byte, error) {
	return proto.Marshal(message.(proto.Message))
}

// Unmarshal() converts bytes to a protobuf structure
func (p *Protobuf) Unmarshal(data []byte, ptr any) error {
	return proto.Unmarshal(data, ptr.(proto.Message))
}

// A Protobuf Any is a special type that allows a protocol buffer message to encapsulate another message generically

// ToAny() packs a protobuf message to a generic any
func (p *Protobuf) ToAny(message proto.Message) (*anypb.Any, error) {
	return anypb.New(message)
}

// FromAny() converts a proto any to the protobuf message
func (p *Protobuf) FromAny(any *anypb.Any) (proto.Message, error) {
	return anypb.UnmarshalNew(any, proto.UnmarshalOptions{})
}

// GetRawProtoField extracts the raw bytes for field number from a proto message
func GetRawProtoField(protoBytes []byte, fieldNumber int) ([]byte, error) {
	var offset int
	// parse the proto bytes to find a field
	for offset < len(protoBytes) {
		// decode the field tag (field number + wire type)
		fieldNum, wireType, tagLen := protowire.ConsumeTag(protoBytes[offset:])
		if tagLen < 0 {
			return nil, fmt.Errorf("invalid tag at offset %d", offset)
		}
		offset += tagLen
		// check if this is the field we're looking for
		if int(fieldNum) == fieldNumber {
			// for length-delimited fields (like messages), we need to read the length
			if wireType == protowire.BytesType {
				// read the length of the field value
				valueLen, lenBytes := protowire.ConsumeVarint(protoBytes[offset:])
				if lenBytes < 0 {
					return nil, fmt.Errorf("invalid length at offset %d", offset)
				}
				// calculate the new offset
				offset += lenBytes
				// extract the field value bytes
				if offset+int(valueLen) > len(protoBytes) {
					return nil, fmt.Errorf("field value exceeds buffer bounds")
				}
				// make buffer to return
				fieldBytes := make([]byte, valueLen)
				// copy into the buffer
				copy(fieldBytes, protoBytes[offset:offset+int(valueLen)])
				// return the value
				return fieldBytes, nil
			} else {
				// for other wire types, consume the value directly
				valueLen := protowire.ConsumeFieldValue(fieldNum, wireType, protoBytes[offset:])
				if valueLen < 0 {
					return nil, fmt.Errorf("invalid field value at offset %d", offset)
				}
				if offset+valueLen > len(protoBytes) {
					return nil, fmt.Errorf("field value exceeds buffer bounds")
				}
				fieldBytes := make([]byte, valueLen)
				copy(fieldBytes, protoBytes[offset:offset+valueLen])
				return fieldBytes, nil
			}
		} else {
			// skip this field
			skipLen := protowire.ConsumeFieldValue(fieldNum, wireType, protoBytes[offset:])
			if skipLen < 0 {
				return nil, fmt.Errorf("invalid field value at offset %d", offset)
			}
			offset += skipLen
		}
	}
	return nil, fmt.Errorf("field number %d not found", fieldNumber)
}

// NullifyProtoField removes a field from protobytes without unmarshalling
func NullifyProtoField(protoBytes []byte, fieldNumber int) ([]byte, error) {
	var offset int
	// create a buffer to store the result
	result := make([]byte, 0, len(protoBytes))
	// iterate through the bytes
	for offset < len(protoBytes) {
		// remember the start position of this field
		fieldStart := offset
		// decode the field tag
		fieldNum, wireType, tagLen := protowire.ConsumeTag(protoBytes[offset:])
		if tagLen < 0 {
			return nil, fmt.Errorf("invalid tag at offset %d", offset)
		}
		// update the offset
		offset += tagLen
		// calculate the field value length
		valueLen := protowire.ConsumeFieldValue(fieldNum, wireType, protoBytes[offset:])
		if valueLen < 0 {
			return nil, fmt.Errorf("invalid field value at offset %d", offset)
		}
		// if this is the field to nullify, skip it entirely
		if int(fieldNum) == fieldNumber {
			offset += valueLen
			continue
		}
		// for all other fields, copy them to the result
		fieldEnd := offset + valueLen
		// copy to the result
		result = append(result, protoBytes[fieldStart:fieldEnd]...)
		// update the offset
		offset = fieldEnd
	}
	// return the result
	return result, nil
}

// HasUnknownFields() returns true if the protobuf message (or any nested message) has unknown fields
// NOTE: this version doesn't support maps in protobuf structure
func HasUnknown(msg proto.Message) bool {
	if msg == nil {
		return false
	}
	m := msg.ProtoReflect()
	// check for this message's unknowns
	if len(m.GetUnknown()) > 0 {
		return true
	}
	// recursively check nested messages
	hasUnknownFound := false
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Message() != nil {
			if fd.IsList() {
				l := v.List()
				for i := 0; i < l.Len(); i++ {
					if HasUnknown(l.Get(i).Message().Interface()) {
						hasUnknownFound = true
						return false // stop iteration
					}
				}
			} else { // single message
				if HasUnknown(v.Message().Interface()) {
					hasUnknownFound = true
					return false // stop iteration
				}
			}
		}
		return true // continue iteration
	})
	return hasUnknownFound
}
