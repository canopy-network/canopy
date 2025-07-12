package crypto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/canopy-network/canopy/lib/codec"
)

// Address represents a short version of a public key that pairs to a users secret private key
// Addresses are the most used identity in the blockchain state due to their hash collision resistant property
type Address []byte

// Address must conform to the AddressI interface
var _ AddressI = Address{}

const (
	// the number of bytes in an address
	AddressSize = 20
)

// NewAddressFromBytes() casts bytes as an AddressI interface
func NewAddressFromBytes(bz []byte) AddressI {
	if bz == nil {
		return nil
	}
	return Address(bz)
}

// NewAddressFromString() returns the hex string implementation of an AddressI interface
func NewAddressFromString(hexString string) (AddressI, error) {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return NewAddressFromBytes(bz), nil
}

// MarshalJSON() is the address implementation of json.Marshaller interface
func (a Address) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }

// UnmarshalJSON() is the address implementation of json.Marshaller interface
func (a *Address) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	// decode the bytes to a hex string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	// decode the string to bytes
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	// assign the bytes to the address object
	*a = bz
	return
}

// Bytes() casts the address value back to a byte slice
func (a Address) Bytes() []byte { return (a)[:] }

// String() returns the hex string representation of an address
func (a Address) String() string { return hex.EncodeToString(a.Bytes()) }

// Equals() compares two address objects and returns true if they're equal
func (a Address) Equals(e AddressI) bool { return bytes.Equal(a.Bytes(), e.Bytes()) }

var cdc = codec.Protobuf{}

// Marshal() implements the proto.Marshaller interface
func (a Address) Marshal() ([]byte, error) {
	return cdc.Marshal(ProtoAddress{Address: a.Bytes()})
}

// NewAddress() creates a new address object from bytes by assigning bytes to the underlying address object
func NewAddress(b []byte) AddressI {
	a := Address(b)
	return &a
}
