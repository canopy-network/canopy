package crypto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
)

type Address []byte

var _ AddressI = &Address{}

const (
	AddressSize = 20
)

func (a *Address) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }
func (a *Address) UnmarshalJSON(b []byte) (err error) {
	var hexString string
	if err = json.Unmarshal(b, &hexString); err != nil {
		return
	}
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		return
	}
	*a = bz
	return
}
func (a *Address) Bytes() []byte          { return (*a)[:] }
func (a *Address) String() string         { return hex.EncodeToString(a.Bytes()) }
func (a *Address) Equals(e AddressI) bool { return bytes.Equal(a.Bytes(), e.Bytes()) }
func (a *Address) Marshal() ([]byte, error) {
	return cdc.Marshal(ProtoAddress{
		Address: a.Bytes(),
	})
}
func NewAddress(b []byte) AddressI {
	a := Address(b)
	return &a
}
