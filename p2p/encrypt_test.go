package p2p

import (
	"net"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestEncryptedConn(t *testing.T) {
	msg1, msg2 := []byte("foo"), []byte("bar")
	p1, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	p2, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	c1, c2 := net.Pipe()
	defer func() { c1.Close(); c2.Close() }()
	var e1, e2 *EncryptedConn
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		e1, err = NewHandshake(c1, &lib.PeerMeta{ChainId: 1}, p1)
		wg.Done()
		require.NoError(t, err)
	}()
	e2, err = NewHandshake(c2, &lib.PeerMeta{ChainId: 1}, p2)
	require.NoError(t, err)
	wg.Wait()
	require.Equal(t, e1.Address.PublicKey, p2.PublicKey().Bytes())
	require.Equal(t, e2.Address.PublicKey, p1.PublicKey().Bytes())
	go func() {
		_, err = e1.Write(msg1)
		require.NoError(t, err)
	}()
	buff := make([]byte, 3)
	_, err = e2.Read(buff)
	require.NoError(t, err)
	require.Equal(t, msg1, buff)
	go func() {
		_, err = e2.Write(msg2)
		require.NoError(t, err)
	}()
	_, err = e1.Read(buff)
	require.NoError(t, err)
	require.Equal(t, msg2, buff)
}
