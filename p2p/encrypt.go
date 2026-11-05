package p2p

import (
	"crypto/cipher"
	"encoding/binary"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	pool "github.com/libp2p/go-buffer-pool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

/*
	Handshake to encrypted connection:
	1) Obtaining shared secret using diffie hellman and x25519 curve (ECDH)
	2) HKDF used to derive the encrypt key, auth key, and nonce (uniqueness IV) from the shared secret
	3) ChaCha20-Poly1305 AEAD scheme uses #2 for encrypt/decrypt/authenticate

	Why handshake and encryption?
	- Authenticate the identity and metadata of the peer
	- Prevent packet sniffing / MITM
	- Guarantee the integrity of a message sent
*/

var (
	// handshakeTimeout is the timeout for each of the handshake network operations
	handshakeTimeout = 1 * time.Second
)

// TransportV2ActivationUnix is the network-wide hard cutover time; zero keeps V2 disabled to prevent accidental peer partitioning.
const TransportV2ActivationUnix int64 = 0

// transportV2ActiveAt() reports whether transport V2 is active at the supplied time.
func transportV2ActiveAt(now time.Time) bool {
	return transportV2ActiveAtUnix(now, TransportV2ActivationUnix)
}

// transportV2ActiveAtUnix() reports whether the supplied activation timestamp has been reached.
func transportV2ActiveAtUnix(now time.Time, activationUnix int64) bool {
	return activationUnix != 0 && now.Unix() >= activationUnix
}

// EncryptedConn is made of the underlying tcp connection, send and receive AEAD ciphers, and the peer address
// NOTE: receiving and sending have two distinct AEAD state objects for key / nonce management and simultaneous send/receive
type EncryptedConn struct {
	conn        net.Conn   // underlying connection
	receive     aeadState  // encryption state for receiving messages
	send        aeadState  // encryption state for sending messages
	mu          sync.Mutex // mutex to prevent unsafe use of net conn
	transportV2 bool       // fixed for the lifetime of this connection

	Address *lib.PeerAddress // authenticated remote peer information
}

// aeadState represents the internal state the encryption protocol
type aeadState struct {
	sync.Mutex
	aead   cipher.AEAD                 // is a cipher mode providing authenticated encryption with associated data
	unread []byte                      // holds extra bytes that weren't read into `data` due to length
	nonce  *[crypto.AEADNonceSize]byte // ensures uniqueness, ensures identical plaintext won't = same ciphertext, and is seed for MAC tag
}

// NewHandshake() executes the authentication protocol between two tcp connections to result in an encryption connection
func NewHandshake(conn net.Conn, meta *lib.PeerMeta, privateKey crypto.PrivateKeyI) (encryptedConn *EncryptedConn, e lib.ErrorI) {
	return newHandshake(conn, meta, privateKey, transportV2ActiveAt(time.Now()))
}

// newHandshake() authenticates a peer and establishes an encrypted connection using the selected transport version.
func newHandshake(conn net.Conn, meta *lib.PeerMeta, privateKey crypto.PrivateKeyI, transportV2 bool) (encryptedConn *EncryptedConn, e lib.ErrorI) {
	// create a temporary keypair to establish a shared secret
	tempPrivateKey, _ := crypto.NewEd25519PrivateKey()
	tempPublicKey := tempPrivateKey.PublicKey().Bytes()
	encryptedConn = &EncryptedConn{conn: conn, transportV2: transportV2}
	// swap temporary keys
	peerTempPublicKey, e := keySwap(encryptedConn, tempPublicKey, handshakeTimeout)
	if e != nil {
		return
	}
	if crypto.PubIsBlacklisted(peerTempPublicKey) {
		return nil, ErrIsBlacklisted()
	}
	// shared Diffie-Hellman secret is computed using the temporary keys
	secret, err := crypto.SharedSecret(peerTempPublicKey, tempPrivateKey.Bytes())
	if err != nil {
		return nil, ErrFailedDiffieHellman(err)
	}
	// locally convert the Diffie-Hellman secret and temporary public keys to
	// send & receive AEAD objects and the corresponding challenge using HMAC-based Key Derivation
	sendAEAD, receiveAEAD, challenge, err := crypto.HKDFSecretsAndChallenge(secret, tempPublicKey, peerTempPublicKey)
	if err != nil {
		return nil, ErrFailedHKDF(err)
	}
	encryptedConn.receive = newInternalState(receiveAEAD)
	encryptedConn.send = newInternalState(sendAEAD)
	// using the newly created encrypted connection, discard the temporary keys and
	// swap signatures with the peer to establish the true public key identity
	peerSig, err := signatureSwap(encryptedConn, &lib.Signature{
		PublicKey: privateKey.PublicKey().Bytes(),
		Signature: privateKey.Sign(challenge[:]),
	}, handshakeTimeout)
	if err != nil {
		return nil, ErrFailedSignatureSwap(err)
	}
	peerPublicKey, err := crypto.NewPublicKeyFromBytes(peerSig.PublicKey)
	if err != nil {
		return nil, ErrInvalidPublicKey(err)
	}
	// verify the peer signature to confirm the identity
	if !peerPublicKey.VerifyBytes(challenge[:], peerSig.Signature) {
		return nil, ErrFailedChallenge()
	}
	// swap peer metadata using the encrypted channel; sign a copy to avoid mutating
	// the shared meta, which would race with other concurrent handshake goroutines
	peerMeta, err := peerMetaSwap(encryptedConn, meta.Copy().Sign(privateKey), handshakeTimeout)
	if err != nil {
		return nil, ErrFailedMetaSwap(err)
	}
	// verify the peer metadata using the peer public key
	if !peerPublicKey.VerifyBytes(peerMeta.SignBytes(), peerMeta.Signature) {
		return nil, ErrFailedChallenge()
	}
	// ensure peer compatibility using the peer metadata
	if peerMeta.NetworkId != meta.NetworkId {
		return nil, ErrIncompatiblePeer()
	}
	// ensure peer chain is the same as ours
	if peerMeta.ChainId != meta.ChainId {
		return nil, ErrIncompatiblePeer()
	}
	// finalize the encrypted connection by setting the exchanged information
	encryptedConn.Address = &lib.PeerAddress{
		PublicKey:  peerSig.PublicKey,
		NetAddress: conn.RemoteAddr().String(),
		PeerMeta:   peerMeta,
	}
	return
}

// Write() writes the data bytes to the encrypted connection
func (c *EncryptedConn) Write(data []byte) (n int, err error) {
	// fallback to regular conn in case encrypted conn is not yet set
	if c.send.aead == nil {
		return c.conn.Write(data)
	}
	// thread safety sends
	c.send.Lock()
	defer c.send.Unlock()
	if !c.transportV2 {
		return c.writeLegacy(data)
	}
	// loop until no more data to send
	for dataSize := len(data); dataSize > 0; dataSize = len(data) {
		chunkSize := min(dataSize, crypto.MaxDataSize)
		chunk := data[:chunkSize]
		data = data[chunkSize:]
		// Authenticate the variable record length as AEAD associated data and bound it before receive-side allocation.
		wireSize := crypto.LengthHeaderSize + chunkSize + crypto.Poly1305TagSize
		wireBuffer := pool.Get(wireSize)
		header := wireBuffer[:crypto.LengthHeaderSize]
		binary.LittleEndian.PutUint32(header, uint32(chunkSize))
		ciphertext := c.send.aead.Seal(wireBuffer[crypto.LengthHeaderSize:crypto.LengthHeaderSize], c.send.nonce[:], chunk, header)
		// increment the 'write state' nonce to follow the security parameters of the AEAD protocol
		incrementNonce(c.send.nonce)
		// Complete any legal short write before advancing the nonce to preserve peer framing.
		if er := writeAll(c.conn, wireBuffer[:crypto.LengthHeaderSize+len(ciphertext)]); er != nil {
			pool.Put(wireBuffer)
			return 0, ErrFailedWrite(er)
		}
		// update number of bytes written
		n += chunkSize
		pool.Put(wireBuffer)
	}
	return
}

// writeLegacy() encrypts and writes data using fixed-size transport V1 records.
func (c *EncryptedConn) writeLegacy(data []byte) (n int, err error) {
	for len(data) != 0 {
		chunkSize := min(len(data), crypto.LegacyMaxDataSize)
		chunk := data[:chunkSize]
		data = data[chunkSize:]
		plainTextBuffer := pool.Get(crypto.LegacyFrameSize)
		cipherTextBuffer := pool.Get(crypto.LegacyEncryptedFrameSize)
		binary.LittleEndian.PutUint32(plainTextBuffer, uint32(chunkSize))
		copy(plainTextBuffer[crypto.LengthHeaderSize:], chunk)
		ciphertext := c.send.aead.Seal(cipherTextBuffer[:0], c.send.nonce[:], plainTextBuffer, nil)
		incrementNonce(c.send.nonce)
		if er := writeAll(c.conn, ciphertext); er != nil {
			pool.Put(plainTextBuffer)
			pool.Put(cipherTextBuffer)
			return n, ErrFailedWrite(er)
		}
		n += chunkSize
		pool.Put(plainTextBuffer)
		pool.Put(cipherTextBuffer)
	}
	return n, nil
}

// Read() checks the connection for cipher data bytes, if found,
// the func decrypts and loads them into the 'data' buffer
func (c *EncryptedConn) Read(data []byte) (n int, err error) {
	// fallback to regular conn in case encrypted conn is not yet set
	if c.receive.aead == nil {
		return c.conn.Read(data)
	}
	// read thread safety
	c.receive.Lock()
	defer c.receive.Unlock()
	// check the unread buffer for leftover overflow chunk bytes from a previous
	// read call if found, then load the leftover bytes into the data buffer
	if bzRead, hadUnread := c.checkUnread(data); hadUnread {
		return bzRead, nil
	}
	if !c.transportV2 {
		return c.readLegacy(data)
	}
	// Read and bound the variable-size record before allocating its payload.
	var header [crypto.LengthHeaderSize]byte
	if _, er := io.ReadFull(c.conn, header[:]); er != nil {
		return 0, er
	}
	chunkLength := binary.LittleEndian.Uint32(header[:])
	if chunkLength == 0 || chunkLength > crypto.MaxDataSize {
		return 0, ErrChunkLargerThanMax()
	}
	ciphertextSize := int(chunkLength) + crypto.Poly1305TagSize
	cipherTextBuffer := pool.Get(ciphertextSize)
	plainTextBuffer := pool.Get(int(chunkLength))
	defer func() { pool.Put(plainTextBuffer); pool.Put(cipherTextBuffer) }()
	if _, er := io.ReadFull(c.conn, cipherTextBuffer); er != nil {
		return 0, er
	}
	// Authenticate the length header so modification fails instead of changing the plaintext interpretation.
	chunk, er := c.receive.aead.Open(plainTextBuffer[:0], c.receive.nonce[:], cipherTextBuffer, header[:])
	if er != nil {
		return 0, ErrConnDecryptFailed(er)
	}
	incrementNonce(c.receive.nonce)
	// load the chunk into the data buffer
	n = copy(data, chunk)
	// if the data buffer is smaller than the chunk, hold them in the 'unread buffer'
	// to be loaded in the following iteration
	c.holdUnread(n, chunk)
	return
}

// readLegacy() reads and decrypts fixed-size transport V1 records.
func (c *EncryptedConn) readLegacy(data []byte) (n int, err error) {
	cipherTextBuffer := pool.Get(crypto.LegacyEncryptedFrameSize)
	plainTextBuffer := pool.Get(crypto.LegacyFrameSize)
	defer func() { pool.Put(plainTextBuffer); pool.Put(cipherTextBuffer) }()
	if _, er := io.ReadFull(c.conn, cipherTextBuffer); er != nil {
		return 0, er
	}
	plainText, er := c.receive.aead.Open(plainTextBuffer[:0], c.receive.nonce[:], cipherTextBuffer, nil)
	if er != nil {
		return 0, ErrConnDecryptFailed(er)
	}
	incrementNonce(c.receive.nonce)
	chunkLength := binary.LittleEndian.Uint32(plainText[:crypto.LengthHeaderSize])
	if chunkLength > crypto.LegacyMaxDataSize {
		return 0, ErrChunkLargerThanMax()
	}
	chunk := plainText[crypto.LengthHeaderSize : crypto.LengthHeaderSize+chunkLength]
	n = copy(data, chunk)
	c.holdUnread(n, chunk)
	return n, nil
}

// TransportV2() reports whether this encrypted connection uses transport V2.
func (c *EncryptedConn) TransportV2() bool { return c != nil && c.transportV2 }

// writeAll() writes a complete transport record despite legal short writes.
func writeAll(conn net.Conn, data []byte) error {
	for len(data) != 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[n:]
	}
	return nil
}

// checkUnread() checks to see if the 'unread buffer' was filled from a partially read chunk
func (c *EncryptedConn) checkUnread(data []byte) (int, bool) {
	if len(c.receive.unread) > 0 {
		// if there is unread
		// load the maximum bytes into the data buffer
		n := copy(data, c.receive.unread)
		c.receive.unread = c.receive.unread[n:]
		return n, true
	}
	return 0, false
}

// holdUnread() holds unread data from a partially read chunk into the 'unread buffer'
func (c *EncryptedConn) holdUnread(bytesRead int, chunk []byte) {
	if bytesRead < len(chunk) {
		c.receive.unread = make([]byte, len(chunk)-bytesRead)
		copy(c.receive.unread, chunk[bytesRead:])
	}
	return
}

// EncryptedConn satisfies the net.conn interface
func (c *EncryptedConn) Close() error {
	var firstErr error
	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		if err := tcpConn.CloseRead(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := tcpConn.CloseWrite(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := c.conn.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if firstErr != nil {
		log.Printf("encrypted conn close error: %v", firstErr)
	}
	return firstErr
}
func (c *EncryptedConn) LocalAddr() net.Addr                { return c.conn.LocalAddr() }
func (c *EncryptedConn) RemoteAddr() net.Addr               { return c.conn.RemoteAddr() }
func (c *EncryptedConn) SetDeadline(t time.Time) error      { return c.conn.SetDeadline(t) }
func (c *EncryptedConn) SetReadDeadline(t time.Time) error  { return c.conn.SetReadDeadline(t) }
func (c *EncryptedConn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

// keySwap() exchanges temporary public keys between peers
func keySwap(conn net.Conn, tempPublicKey []byte, timeout time.Duration) (peerTempPublicKey []byte, err lib.ErrorI) {
	peerTempKey, selfPubKey := new(crypto.ProtoPubKey), &crypto.ProtoPubKey{Pubkey: tempPublicKey}
	if err = parallelSendReceive(conn, selfPubKey, peerTempKey, timeout); err != nil {
		return nil, err
	}
	return peerTempKey.Pubkey, nil
}

// signatureSwap() exchanges the actual peer keys along with a signature between peers
func signatureSwap(conn net.Conn, signature *lib.Signature, timeout time.Duration) (peerSig *lib.Signature, err lib.ErrorI) {
	peerSig = new(lib.Signature)
	err = parallelSendReceive(conn, signature, peerSig, timeout)
	return
}

// peerMetaSwap() exchanges peer metadata between two peers
func peerMetaSwap(conn net.Conn, meta *lib.PeerMeta, timeout time.Duration) (peerMeta *lib.PeerMeta, err lib.ErrorI) {
	peerMeta = new(lib.PeerMeta)
	err = parallelSendReceive(conn, meta, peerMeta, timeout)
	return
}

// parallelSendReceive() executes send and receive functions in parallel
// waits for both functions to complete, and returns if either has an error
func parallelSendReceive(conn net.Conn, a, b proto.Message, timeout time.Duration) lib.ErrorI {
	var g errgroup.Group
	g.Go(func() error { _, err := sendProtoMsg(conn, a, timeout); return err })
	g.Go(func() error { _, err := receiveProtoMsg(conn, b, timeout); return err })
	if er := g.Wait(); er != nil {
		return ErrErrorGroup(er)
	}
	return nil
}

// incrementNonce() increments the AEAD counter
// the counter ensures uniqueness, prevents 1:1 ciphertext for identical plaintext,
// and is seed data for the MAC
func incrementNonce(nonce *[crypto.AEADNonceSize]byte) {
	counter := binary.LittleEndian.Uint64(nonce[4:])
	if counter == math.MaxUint64 {
		counter = 0 // reset - should never happen...
	}
	counter++
	binary.LittleEndian.PutUint64(nonce[4:], counter)
}

func newInternalState(aead cipher.AEAD) aeadState {
	return aeadState{
		Mutex: sync.Mutex{},
		aead:  aead,
		nonce: new([crypto.AEADNonceSize]byte),
	}
}
