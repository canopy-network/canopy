package p2p

import (
	"crypto/cipher"
	"encoding/binary"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	pool "github.com/libp2p/go-buffer-pool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

var (
	// isTest is a bool to turn on and off unsafe mode for testing
	isTest = false
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

// EncryptedConn is made of the underlying tcp connection, send and receive AEAD ciphers, and the peer address
// NOTE: receiving and sending have two distinct AEAD state objects for key / nonce management and simultaneous send/receive
type EncryptedConn struct {
	conn    net.Conn   // underlying connection
	receive aeadState  // encryption state for receiving messages
	send    aeadState  // encryption state for sending messages
	mu      sync.Mutex // mutex to prevent unsafe use of net conn

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
	// create a temporary keypair to establish a shared secret
	tempPrivateKey, _ := crypto.NewEd25519PrivateKey()
	tempPublicKey := tempPrivateKey.PublicKey().Bytes()
	encryptedConn = &EncryptedConn{
		conn: conn,
	}
	// swap temporary keys
	peerTempPublicKey, e := keySwap(encryptedConn, tempPublicKey)
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
	})
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
	// swap peer metadata using the encrypted channel
	peerMeta, err := peerMetaSwap(encryptedConn, meta.Sign(privateKey))
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
		// c.mu.Lock()
		// defer c.mu.Unlock()
		return c.conn.Write(data)
	}
	// thread safety sends
	c.send.Lock()
	defer c.send.Unlock()
	// setup data variables
	chunkSize, chunk := 0, []byte(nil)
	// loop until no more data to send
	for dataSize := len(data); dataSize > 0; dataSize = len(data) {
		// a 'buffer pool' saves resources by re-using buffers, take 2 buffers from the pool
		cipherTextBuffer, plainTextBuffer := pool.Get(crypto.EncryptedFrameSize), pool.Get(crypto.FrameSize)
		// if total data to write can be fit into a single chunk
		if dataSize < crypto.MaxDataSize {
			chunk = data
			data = nil
		} else { // else load the next chunk and save the rest of the data
			chunk = data[:crypto.MaxDataSize]
			data = data[crypto.MaxDataSize:]
		}
		chunkSize = len(chunk)
		// size of data is the first 4 bytes of the message
		binary.LittleEndian.PutUint32(plainTextBuffer, uint32(chunkSize))
		// the actual chunk comes after the first 4 bytes
		copy(plainTextBuffer[crypto.LengthHeaderSize:], chunk)
		// encrypt the plain text into the cipher buffer
		c.send.aead.Seal(cipherTextBuffer[:0], c.send.nonce[:], plainTextBuffer, nil)
		// increment the 'write state' nonce to follow the security parameters of the AEAD protocol
		incrementNonce(c.send.nonce)
		// write the cipher text to the underlying connection
		if _, er := c.conn.Write(cipherTextBuffer); er != nil {
			return 0, ErrFailedWrite(er)
		}
		// update number of bytes written
		n += chunkSize
		// release the buffers back into the pool for reuse
		pool.Put(cipherTextBuffer)
		pool.Put(plainTextBuffer)
	}
	return
}

// Read() checks the connection for cipher data bytes, if found,
// the func decrypts and loads them into the 'data' buffer
func (c *EncryptedConn) Read(data []byte) (n int, err error) {
	// fallback to regular conn in case encrypted conn is not yet set
	if c.receive.aead == nil {
		// flag to allow unit test to pass (without this a deadlock would happen because of using same instances of conn in read and write)
		// if !isTest {
		// 	c.mu.Lock()
		// 	defer c.mu.Unlock()
		// }
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
	// a 'buffer pool' saves resources by re-using buffers, take 2 buffers from the pool
	cipherTextBuffer, plainTextBuffer := pool.Get(crypto.EncryptedFrameSize), pool.Get(crypto.FrameSize)
	defer func() { pool.Put(plainTextBuffer); pool.Put(cipherTextBuffer) }()
	// load up bytes from the connection into the cipher text buffer
	if _, er := io.ReadFull(c.conn, cipherTextBuffer); er != nil {
		return 0, er
	}
	// decrypt and load the cipher text buffer into the plain text buffer
	if _, er := c.receive.aead.Open(plainTextBuffer[:0], c.receive.nonce[:], cipherTextBuffer, nil); er != nil {
		return n, ErrConnDecryptFailed(er)
	}
	// increment the 'read state' nonce to follow the security parameters of the AEAD protocol
	incrementNonce(c.receive.nonce)
	// read the first 4 bytes to get the length of the chunk
	chunkLength := binary.LittleEndian.Uint32(plainTextBuffer) // read the length header
	// ensure it follows the max chunk sizing protocol
	if chunkLength > crypto.MaxDataSize {
		return 0, ErrChunkLargerThanMax()
	}
	// remove the first 4 bytes to get the chunk payload
	chunk := plainTextBuffer[crypto.LengthHeaderSize : crypto.LengthHeaderSize+chunkLength]
	// load the chunk into the data buffer
	n = copy(data, chunk)
	// if the data buffer is smaller than the chunk, hold them in the 'unread buffer'
	// to be loaded in the following iteration
	c.holdUnread(n, chunk)
	return
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
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.Close()
}
func (c *EncryptedConn) LocalAddr() net.Addr {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.LocalAddr()
}
func (c *EncryptedConn) RemoteAddr() net.Addr {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.RemoteAddr()
}
func (c *EncryptedConn) SetDeadline(t time.Time) error {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.SetDeadline(t)
}
func (c *EncryptedConn) SetReadDeadline(t time.Time) error {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.SetReadDeadline(t)
}
func (c *EncryptedConn) SetWriteDeadline(t time.Time) error {
	// c.mu.Lock()
	// defer c.mu.Unlock()
	return c.conn.SetWriteDeadline(t)
}

// keySwap() exchanges temporary public keys between peers
func keySwap(conn net.Conn, tempPublicKey []byte) (peerTempPublicKey []byte, err lib.ErrorI) {
	peerTempKey, selfPubKey := new(crypto.ProtoPubKey), &crypto.ProtoPubKey{Pubkey: tempPublicKey}
	if err = parallelSendReceive(conn, selfPubKey, peerTempKey); err != nil {
		return nil, err
	}
	return peerTempKey.Pubkey, nil
}

// signatureSwap() exchanges the actual peer keys along with a signature between peers
func signatureSwap(conn net.Conn, signature *lib.Signature) (peerSig *lib.Signature, err lib.ErrorI) {
	peerSig = new(lib.Signature)
	err = parallelSendReceive(conn, signature, peerSig)
	return
}

// peerMetaSwap() exchanges peer metadata between two peers
func peerMetaSwap(conn net.Conn, meta *lib.PeerMeta) (peerMeta *lib.PeerMeta, err lib.ErrorI) {
	peerMeta = new(lib.PeerMeta)
	err = parallelSendReceive(conn, meta, peerMeta)
	return
}

// parallelSendReceive() executes send and receive functions in parallel
// waits for both functions to complete, and returns if either has an error
func parallelSendReceive(conn net.Conn, a, b proto.Message) lib.ErrorI {
	var g errgroup.Group
	g.Go(func() error { return sendProtoMsg(conn, a) })
	g.Go(func() error { return receiveProtoMsg(conn, b) })
	if er := g.Wait(); er != nil {
		return ErrErrorGroup(er)
	}
	return nil
}

// receiveProtoMsg() encodes and sends a proto message to a net.Conn
// NOTE: this function is only used before establishing the encrypted conn
func sendProtoMsg(conn net.Conn, ptr proto.Message) lib.ErrorI {
	bz, err := lib.Marshal(ptr)
	if err != nil {
		return err
	}
	if e := conn.SetWriteDeadline(time.Now().Add(time.Second)); e != nil {
		return ErrFailedWrite(e)
	}
	if _, er := conn.Write(bz); er != nil {
		return ErrFailedWrite(er)
	}
	return nil
}

// receiveProtoMsg() receives and decodes a proto message from a net.Conn
// NOTE: this function is only used before establishing the encrypted conn
func receiveProtoMsg(conn net.Conn, ptr proto.Message) lib.ErrorI {
	buffer := make([]byte, units.KB)
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		return ErrFailedRead(err)
	}
	n, err := conn.Read(buffer)
	if err != nil {
		return ErrFailedRead(err)
	}
	if e := lib.Unmarshal(buffer[:n], ptr); e != nil {
		return e
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
