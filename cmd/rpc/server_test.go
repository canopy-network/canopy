package rpc

import (
	"errors"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

type timeoutWriteError struct{}

func (timeoutWriteError) Error() string   { return "write tcp 127.0.0.1:50002: i/o timeout" }
func (timeoutWriteError) Timeout() bool   { return true }
func (timeoutWriteError) Temporary() bool { return false }

func TestIsClientWriteError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "net timeout",
			err:  timeoutWriteError{},
			want: true,
		},
		{
			name: "broken pipe",
			err:  syscall.EPIPE,
			want: true,
		},
		{
			name: "connection reset",
			err:  syscall.ECONNRESET,
			want: true,
		},
		{
			name: "string timeout",
			err:  errors.New("write tcp 172.29.0.9:50002->172.29.0.1:38824: i/o timeout"),
			want: true,
		},
		{
			name: "unexpected write failure",
			err:  errors.New("short write"),
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, isClientWriteError(test.err))
		})
	}
}
