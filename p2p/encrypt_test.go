package p2p

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

// TestTransportV2Activation() verifies the configured hard-cutover activation behavior.
func TestTransportV2Activation(t *testing.T) {
	activation := time.Unix(2_000, 0)
	require.False(t, transportV2ActiveAtUnix(activation, 0))
	require.False(t, transportV2ActiveAtUnix(activation.Add(-time.Second), activation.Unix()))
	require.True(t, transportV2ActiveAtUnix(activation, activation.Unix()))
}

func TestEncryptedConn(t *testing.T) {
	msg1, msg2 := []byte("foo"), []byte("bar")
	e1, e2 := encryptedPipe(t)
	defer func() { _ = e1.Close(); _ = e2.Close() }()
	done := make(chan error, 1)
	go func() {
		_, err := e1.Write(msg1)
		done <- err
	}()
	buff := make([]byte, 3)
	_, err := e2.Read(buff)
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Equal(t, msg1, buff)
	go func() {
		_, err := e2.Write(msg2)
		done <- err
	}()
	_, err = e1.Read(buff)
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Equal(t, msg2, buff)
}

// TestEncryptedConnVariableLengthRecords() verifies transport V2 handles variable-length encrypted records.
func TestEncryptedConnVariableLengthRecords(t *testing.T) {
	e1, e2 := encryptedPipe(t, true)
	defer func() { _ = e1.Close(); _ = e2.Close() }()

	payload := bytes.Repeat([]byte{0x5a}, crypto.MaxDataSize*3+17)
	done := make(chan error, 1)
	go func() {
		_, err := e1.Write(payload)
		done <- err
	}()

	got := make([]byte, len(payload))
	_, err := io.ReadFull(e2, got)
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Equal(t, payload, got)
}

// encryptedPipe() creates a connected encrypted transport pair for tests.
func encryptedPipe(t *testing.T, forceTransportV2 ...bool) (*EncryptedConn, *EncryptedConn) {
	t.Helper()
	p1, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	p2, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)
	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })

	result := make(chan *EncryptedConn, 1)
	errs := make(chan error, 1)
	transportV2 := false
	if len(forceTransportV2) != 0 {
		transportV2 = forceTransportV2[0]
	}
	go func() {
		conn, errI := newHandshake(c1, &lib.PeerMeta{ChainId: 1}, p1, transportV2)
		if errI != nil {
			errs <- errI
			return
		}
		result <- conn
	}()
	e2, errI := newHandshake(c2, &lib.PeerMeta{ChainId: 1}, p2, transportV2)
	require.NoError(t, errI)
	select {
	case err = <-errs:
		require.NoError(t, err)
		return nil, nil
	case e1 := <-result:
		return e1, e2
	}
}
