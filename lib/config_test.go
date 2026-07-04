package lib

import (
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
		// oracle + block-provider defaults are populated by DefaultConfig(); the golden
		// expectation must include them or the embedded structs stay zero-valued and mismatch
		EthBlockProviderConfig: DefaultEthBlockProviderConfig(),
		SolBlockProviderConfig: DefaultSolBlockProviderConfig(),
		OracleConfig:           DefaultOracleConfig(),
	}
	// execute the function call
	got := DefaultConfig()
	// compare got vs expected, LSSCompactionInterval is randomized so it needs to be ignored
	diff := cmp.Diff(expected, got, cmpopts.IgnoreFields(Config{}, "LSSCompactionInterval"))
	require.Empty(t, diff, "config mismatch: %s", diff)
}

func TestOracleConfigValidate(t *testing.T) {
	// the shipped default must satisfy the invariant
	require.NoError(t, DefaultOracleConfig().Validate())

	tests := []struct {
		name    string
		cfg     OracleConfig
		wantErr bool
	}{
		{
			name:    "disabled oracle skips validation even when invariant violated",
			cfg:     OracleConfig{OracleEnabled: false, ProposeDelayBlocks: 1, SafeBlockConfirmations: 5},
			wantErr: false,
		},
		{
			name:    "propose delay greater than confirmations is valid",
			cfg:     OracleConfig{OracleEnabled: true, ProposeDelayBlocks: 7, SafeBlockConfirmations: 5},
			wantErr: false,
		},
		{
			name:    "propose delay equal to confirmations gives zero margin - rejected",
			cfg:     OracleConfig{OracleEnabled: true, ProposeDelayBlocks: 5, SafeBlockConfirmations: 5},
			wantErr: true,
		},
		{
			name:    "propose delay less than confirmations gives negative margin - rejected",
			cfg:     OracleConfig{OracleEnabled: true, ProposeDelayBlocks: 3, SafeBlockConfirmations: 5},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
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
