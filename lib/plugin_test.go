package lib

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// byteWriterConn forwards Write one byte at a time so concurrent writers
// interleave unless the caller serializes them.
type byteWriterConn struct {
	net.Conn
}

func (c byteWriterConn) Write(b []byte) (int, error) {
	total := 0
	for len(b) > 0 {
		n, err := c.Conn.Write(b[:1])
		total += n
		if n > 0 {
			b = b[n:]
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func TestSendLengthPrefixedSerializesWrites(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	p := &Plugin{conn: byteWriterConn{client}, log: NewNullLogger()}
	const n = 50
	const payloadLen = 64
	payloads := make([][]byte, n)
	for i := 0; i < n; i++ {
		payloads[i] = make([]byte, payloadLen)
		for j := 0; j < payloadLen; j++ {
			payloads[i][j] = byte(i + 1)
		}
	}

	errCh := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			errCh <- p.sendLengthPrefixed(payloads[i])
		}(i)
	}

	got := make(map[byte]int, n)
	for i := 0; i < n; i++ {
		lenBuf := make([]byte, 4)
		_, err := io.ReadFull(server, lenBuf)
		require.NoError(t, err)
		length := binary.BigEndian.Uint32(lenBuf)
		require.Equal(t, uint32(payloadLen), length)
		body := make([]byte, length)
		_, err = io.ReadFull(server, body)
		require.NoError(t, err)
		first := body[0]
		for _, b := range body {
			require.Equal(t, first, b)
		}
		got[first]++
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	require.Len(t, got, n)
}
