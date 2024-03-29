package crypto

type PublicKeyI interface {
	Address() AddressI
	Bytes() []byte
	VerifyBytes(msg []byte, sig []byte) bool
	String() string
	Equals(PublicKeyI) bool
}

type PrivateKeyI interface {
	Bytes() []byte
	Sign(msg []byte) []byte
	PublicKey() PublicKeyI
	String() string
	Equals(PrivateKeyI) bool
}

type AddressI interface {
	Marshal() ([]byte, error)
	MarshalJSON() ([]byte, error)
	Bytes() []byte
	String() string
	Equals(AddressI) bool
}

type MultiPublicKey interface {
	AggregateSignatures(ordered [][]byte) ([]byte, error)
	VerifyBytes(msg, aggregatedSignature []byte) bool
	AddSigner(index int) error
	PublicKeys() (keys []PublicKeyI)
	SetBitmap(bm []byte) error
	Bitmap() []byte
	Copy() MultiPublicKey
	Reset()
}
