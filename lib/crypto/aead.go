package crypto

import (
	"bytes"
	"crypto/cipher"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"io"
)

// MaxDataSize is the largest plaintext in one variable-length authenticated transport record.
const (
	LengthHeaderSize   = 4
	MaxDataSize        = 64 * 1024
	ChallengeSize      = 32
	Poly1305TagSize    = 16
	FrameSize          = MaxDataSize + LengthHeaderSize
	EncryptedFrameSize = Poly1305TagSize + FrameSize
	AEADKeySize        = chacha20poly1305.KeySize
	AEADNonceSize      = chacha20poly1305.NonceSize
	TwoAEADKeySize     = 2 * AEADKeySize
	HKDFSize           = TwoAEADKeySize + ChallengeSize // 2 keys and challenge
)

const (
	// LegacyMaxDataSize is the fixed plaintext payload used by transport V1.
	LegacyMaxDataSize        = 1024
	LegacyFrameSize          = LegacyMaxDataSize + LengthHeaderSize
	LegacyEncryptedFrameSize = Poly1305TagSize + LegacyFrameSize
)

// Big picture: DH is used to establish a shared secret, and then HKDF is used to derive multiple keys from that secret for encryption

// HKDFSecretsAndChallenge generates shared encryption keys and a unique challenge
// using HKDF (HMAC-based Key Derivation Function) from a Diffie-Hellman shared secret
//
// Parameters:
// - dhSecret: the shared secret derived from a Diffie-Hellman key exchange
// - ePub: the ephemeral local public key, this should just be a temporary public key only used for authentication
// - ePeerPub: the ephemeral peer's public key, this should just be a temporary public key only used for authentication
//
// Returns:
// - send: an AEAD (Authenticated Encryption with Associated Data) cipher for outgoing messages
// - receive: an AEAD (Authenticated Encryption with Associated Data) cipher for incoming messages
// - challenge: a unique 32-byte challenge used for authentication purposes
// - err: error if key derivation or cipher creation fails
//
// Process:
// 1. Uses the shared DH secret as input to HKDF to derive multiple cryptographic keys
// 2. Compares local and peer public keys to determine which key is for sending vs. receiving
// 3. Populates `challenge`, `sendSecret`, and `receiveSecret` from the HKDF buffer
// 4. Initializes AEAD ciphers (ChaCha20-Poly1305) for sending and receiving messages
func HKDFSecretsAndChallenge(dhSecret []byte, ePub, ePeerPub []byte) (send cipher.AEAD, receive cipher.AEAD, challenge *[32]byte, err error) {
	// (HMAC-based Key Derivation Function) function is a secure key derivation method used to derive multiple
	// cryptographic keys from a single shared secret
	hkdfReader := hkdf.New(Hasher, dhSecret, nil, nil)
	// create an array of bytes to populate with the key derivation
	buffer := new([HKDFSize]byte)
	// populate with the HKDF function
	// this populates the buffer with two pseudorandom bytes from which any portion can be extracted and used as desired for cryptographic purposes
	// the important part with an HKDF is that both sides agree to which bytes each party is using
	if _, err = io.ReadFull(hkdfReader, buffer[:]); err != nil {
		return
	}
	// create byte arrays for the challenge and two AEAD private keys
	// two sets of keys help keep each communication direction secure and independently managed,
	// adding an extra layer of robustness to the encryption scheme
	challenge, receiveSecret, sendSecret := new([ChallengeSize]byte), new([AEADKeySize]byte), new([AEADKeySize]byte)
	// Use a basic comparison protocol to assign send and receive channels:
	// The actor with the smaller public key (ePub < ePeerPub) will use the buffer's first derived secret as the receive key
	// and the second as the send key. The other actor (ePub >= ePeerPub) does the reverse, ensuring consistent
	// send/receive key pairing between both parties
	if bytes.Compare(ePub, ePeerPub) < 0 {
		getTwoSecretsFromBuffer(buffer, receiveSecret, sendSecret)
	} else {
		getTwoSecretsFromBuffer(buffer, sendSecret, receiveSecret)
	}
	// copy the last part of the HKDF into the challenge object
	copy(challenge[:], buffer[TwoAEADKeySize:HKDFSize])
	// using the arbitrary bytes as 32 byte secret inputs into the chacha20ply1305 scheme,
	// the peers are able to encrypt and decrypt each message
	// Unlike AES, ChaCha20 is resistant to timing attacks in software implementations
	send, err = chacha20poly1305.New(sendSecret[:])
	if err != nil {
		return
	}
	// use the same AEAD protocol for the receive secret
	receive, err = chacha20poly1305.New(receiveSecret[:])
	if err != nil {
		return
	}
	return
}

// getTwoSecretsFromBuffer() takes an HKDF buffer and
func getTwoSecretsFromBuffer(buffer *[HKDFSize]byte, first, second *[32]byte) {
	copy(first[:], buffer[0:AEADKeySize])
	copy(second[:], buffer[AEADKeySize:TwoAEADKeySize])
}
