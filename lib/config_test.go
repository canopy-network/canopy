package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	// calculate expected
	expected := Config{
		MainConfig:         DefaultMainConfig(),
		RPCConfig:          DefaultRPCConfig(),
		StateMachineConfig: DefaultStateMachineConfig(),
		StoreConfig:        DefaultStoreConfig(),
		P2PConfig:          DefaultP2PConfig(),
		ConsensusConfig:    DefaultConsensusConfig(),
		MempoolConfig:      DefaultMempoolConfig(),
		MetricsConfig:      DefaultMetricsConfig(),
	}
	// execute the function call
	got := DefaultConfig()
	// compare got vs expected, LSSCompactionInterval is randomized so it needs to be ignored
	diff := cmp.Diff(expected, got, cmpopts.IgnoreFields(Config{}, "LSSCompactionInterval"))
	require.Empty(t, diff, "config mismatch: %s", diff)
}

func TestFileConfig(t *testing.T) {
	filePath := "./test_config"
	// define a variable to test upon
	config := DefaultConfig()
	// write to file
	require.NoError(t, config.WriteToFile(filePath))
	defer os.RemoveAll(filePath)
	// read from file
	got, err := NewConfigFromFile(filePath)
	require.NoError(t, err)
	// compare got vs expected
	require.Equal(t, config, got)
}

func TestRestrictedAddressesOmitEmpty(t *testing.T) {
	bz, err := json.Marshal(DefaultConfig())
	require.NoError(t, err)
	require.NotContains(t, string(bz), "restrictedAddresses")
	c := DefaultConfig()
	c.RestrictedAddresses = []string{"01"}
	bz, err = json.Marshal(c)
	require.NoError(t, err)
	require.True(t, bytes.Contains(bz, []byte(`"restrictedAddresses":["01"]`)))
}
