package lib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	// calculate expected
	expected := Config{
		MainConfig:             DefaultMainConfig(),
		RPCConfig:              DefaultRPCConfig(),
		StateMachineConfig:     DefaultStateMachineConfig(),
		StoreConfig:            DefaultStoreConfig(),
		P2PConfig:              DefaultP2PConfig(),
		ConsensusConfig:        DefaultConsensusConfig(),
		MempoolConfig:          DefaultMempoolConfig(),
		MetricsConfig:          DefaultMetricsConfig(),
		EthBlockProviderConfig: DefaultEthBlockProviderConfig(),
		OracleConfig:           DefaultOracleConfig(),
	}
	// execute the function call
	got := DefaultConfig()
	// compare got vs expected
	require.EqualExportedValues(t, expected, got)
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
