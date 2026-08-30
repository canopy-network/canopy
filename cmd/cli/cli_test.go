package cli

import (
	"reflect"
	"testing"

	"github.com/canopy-network/canopy/lib"
)

func TestParseGlobalFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantDataDir  string
		wantRPC      string
		wantAdmin    string
		wantPosition []string
	}{
		{"defaults", []string{"start"}, lib.DefaultDataDirPath(), "", "", []string{"start"}},
		{"flag before subcmd", []string{"--data-dir", "/custom", "start"}, "/custom", "", "", []string{"start"}},
		{"flag after subcmd", []string{"start", "--data-dir", "/custom"}, "/custom", "", "", []string{"start"}},
		{"equals form", []string{"--data-dir=/custom", "start"}, "/custom", "", "", []string{"start"}},
		{"url overrides", []string{"start", "--rpc-url", "http://x:1", "--admin-url", "http://x:2"}, lib.DefaultDataDirPath(), "http://x:1", "http://x:2", []string{"start"}},
		{"unknown flag tolerated", []string{"start", "--headless"}, lib.DefaultDataDirPath(), "", "", []string{"start"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DataDir, rpcURLFlag, adminURLFlag, externalAddressFlag = "", "", "", ""
			pos, err := ParseGlobalFlags(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if DataDir != tt.wantDataDir || rpcURLFlag != tt.wantRPC || adminURLFlag != tt.wantAdmin {
				t.Fatalf("got dir=%q rpc=%q admin=%q", DataDir, rpcURLFlag, adminURLFlag)
			}
			if !reflect.DeepEqual(pos, tt.wantPosition) {
				t.Fatalf("positional = %v, want %v", pos, tt.wantPosition)
			}
		})
	}
}

func TestGlobalFlagArgs(t *testing.T) {
	DataDir, rpcURLFlag, adminURLFlag, externalAddressFlag = "/custom", "", "", ""
	if got := GlobalFlagArgs(); !reflect.DeepEqual(got, []string{"--data-dir", "/custom"}) {
		t.Fatalf("got %v", got)
	}
	DataDir, rpcURLFlag, adminURLFlag, externalAddressFlag = "/custom", "http://r", "http://a", "tcp://host"
	want := []string{"--data-dir", "/custom", "--rpc-url", "http://r", "--admin-url", "http://a", "--external-address", "tcp://host"}
	if got := GlobalFlagArgs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
