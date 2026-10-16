package contract

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
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

	p := &Plugin{conn: byteWriterConn{client}}
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
			if err := p.sendLengthPrefixed(payloads[i]); err != nil {
				errCh <- err
				return
			}
			errCh <- nil
		}(i)
	}

	got := make(map[byte]int, n)
	for i := 0; i < n; i++ {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(server, lenBuf); err != nil {
			t.Fatal(err)
		}
		length := binary.BigEndian.Uint32(lenBuf)
		if length != uint32(payloadLen) {
			t.Fatalf("garbled length prefix: got %d", length)
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(server, body); err != nil {
			t.Fatal(err)
		}
		first := body[0]
		for _, b := range body {
			if b != first {
				t.Fatalf("interleaved payload bytes")
			}
		}
		got[first]++
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(got) != n {
		t.Fatalf("expected %d distinct frames, got %d", n, len(got))
	}
}
