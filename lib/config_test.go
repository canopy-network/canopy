package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

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

// TestConfigDefaults() verifies the consensus defaults used when optional configuration is absent.
func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()
	require.Equal(t, uint64(80_000_000), config.InitialTokensPerBlock)
	require.Equal(t, uint64(3_150_000), config.BlocksPerHalvening)
	require.Equal(t, 500, config.NewHeightTimeoutMs)
	require.Equal(t, 500*time.Millisecond, config.ConsensusConfig.RoundRobinBlockInterval())
	require.Equal(t, 175*time.Millisecond, config.ConsensusConfig.BlockConsensusLeadTime())
	require.Equal(t, time.Second, config.ConsensusConfig.MaximumFutureBlockTime())
	require.Equal(t, uint64(64_000_000), config.MaxTotalBytes)
	require.Equal(t, uint32(250_000), config.MaxTransactionCount)

	config.RoundRobinBlockIntervalMS = 750
	config.BlockConsensusLeadTimeMS = 225
	config.MaximumFutureBlockTimeMS = 1500
	require.Equal(t, 750*time.Millisecond, config.ConsensusConfig.RoundRobinBlockInterval())
	require.Equal(t, 225*time.Millisecond, config.ConsensusConfig.BlockConsensusLeadTime())
	require.Equal(t, 1500*time.Millisecond, config.ConsensusConfig.MaximumFutureBlockTime())
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

// TestFileConfigWithoutProposerModeDefaultsToVRF() verifies old config files retain VRF proposer selection.
func TestFileConfigWithoutProposerModeDefaultsToVRF(t *testing.T) {
	filePath := t.TempDir() + "/config.json"
	require.NoError(t, os.WriteFile(filePath, []byte(`{"chainId":1}`), 0o600))

	config, err := NewConfigFromFile(filePath)
	require.NoError(t, err)
	require.Equal(t, ProposerModeVRF, config.ProposerMode)
	require.Equal(t, 500*time.Millisecond, config.ConsensusConfig.RoundRobinBlockInterval())
	require.Equal(t, 175*time.Millisecond, config.ConsensusConfig.BlockConsensusLeadTime())
	require.Equal(t, time.Second, config.ConsensusConfig.MaximumFutureBlockTime())
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
