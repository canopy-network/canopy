package rpc

import "testing"

func TestListenHost(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{name: "ipv4", address: "0.0.0.0:9001", want: "0.0.0.0"},
		{name: "wildcard", address: ":9001", want: ""},
		{name: "ipv6", address: "[::]:9001", want: "::"},
		{name: "host_without_port", address: "localhost", want: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenHost(tt.address); got != tt.want {
				t.Fatalf("listenHost(%q) = %q, want %q", tt.address, got, tt.want)
			}
		})
	}
}
