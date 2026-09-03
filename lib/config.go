package lib

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/alecthomas/units"
)

/* This file implements logic for 'user controlled' global configurations of each module of the node */

const (
	// GLOBAL CONSTANTS
	UnknownChainId         = uint64(0)            // the default 'unknown' chain id
	CanopyChainId          = uint64(1)            // NOTE: to not break nested-chain recursion, this should not be used except for 'default config/genesis' developer setups
	DAOPoolID              = 2*math.MaxUint16 + 1 // must be above the MaxUint16 * 2 to ensure no 'overlap' with 'chainId + EscrowAddend'
	CanopyMainnetNetworkId = 1                    // the identifier of the 'mainnet' of Canopy
)

const (
	// FILE NAMES in the 'data directory'
	ConfigFilePath    = "config.json"        // the file path for the node configuration
	ValKeyPath        = "validator_key.json" // the file path for the node's private key
	GenesisFilePath   = "genesis.json"       // the file path for the genesis (first block)
	ProposalsFilePath = "proposals.json"     // the file path for governance proposal voting configuration
	PollsFilePath     = "polls.json"         // the file path for governance 'straw' polling voting and tracking
)

// Config is the structure of the user configuration options for a Canopy node
type Config struct {
	MainConfig         // main options spanning over all modules
	LoggerConfig       // logger options
	RPCConfig          // rpc API options
	StateMachineConfig // FSM options
	StoreConfig        // persistence options
	P2PConfig          // peer-to-peer options
	ConsensusConfig    // bft options
	MempoolConfig      // mempool options
	MetricsConfig      // telemetry options
}

// DefaultConfig() returns a Config with developer set options
func DefaultConfig() Config {
	return Config{
		MainConfig:         DefaultMainConfig(),
		RPCConfig:          DefaultRPCConfig(),
		StateMachineConfig: DefaultStateMachineConfig(),
		StoreConfig:        DefaultStoreConfig(),
		P2PConfig:          DefaultP2PConfig(),
		ConsensusConfig:    DefaultConsensusConfig(),
		MempoolConfig:      DefaultMempoolConfig(),
		MetricsConfig:      DefaultMetricsConfig(),
	}
}

// PluginHome() returns the persistent plugin directory under the data dir; the
// single source of truth shared by the node launcher and the auto-updater
func (c Config) PluginHome(plugin string) string {
	return filepath.Join(c.DataDirPath, "plugin", plugin)
}

// MAIN CONFIG BELOW

type MainConfig struct {
	LogLevel            string                 `json:"logLevel"`            // any level includes the levels above it: debug < info < warning < error
	ChainId             uint64                 `json:"chainId"`             // the identifier of this particular chain within a single 'network id'
	SleepUntil          uint64                 `json:"sleepUntil"`          // allows coordinated 'wake-ups' for genesis or chain halt events
	RootChain           []RootChain            `json:"rootChain"`           // a list of the root chain(s) a node could connect to as dictated by the governance parameter 'RootChainId'
	RunVDF              bool                   `json:"runVDF"`              // whether the node should run a Verifiable Delay Function to help secure the network against Long-Range-Attacks
	Headless            bool                   `json:"headless"`            // turn off the web wallet and block explorer 'web' front ends
	AutoUpdate          bool                   `json:"autoUpdate"`          // check for new versions of software each X time
	AutoUpdateRepoOwner string                 `json:"autoUpdateRepoOwner"` // GitHub repo owner for core auto-updates (e.g., "canopy-network")
	AutoUpdateRepoName  string                 `json:"autoUpdateRepoName"`  // GitHub repo name for core auto-updates (e.g., "canopy")
	Plugin              string                 `json:"plugin"`              // the configured plugin to use
	PluginTimeoutMS     int                    `json:"pluginTimeoutMS"`     // plugin request timeout in milliseconds
	PluginAutoUpdate    PluginAutoUpdateConfig `json:"pluginAutoUpdate"`    // plugin auto-update configuration
}

// PluginAutoUpdateConfig holds configuration for plugin auto-updates
type PluginAutoUpdateConfig struct {
	Enabled   bool   `json:"enabled"`   // whether plugin auto-update is enabled
	RepoOwner string `json:"repoOwner"` // GitHub repository owner (e.g., "canopy-network")
	RepoName  string `json:"repoName"`  // GitHub repository name (e.g., "canopy")
}

// DefaultMainConfig() sets log level to 'info'
func DefaultMainConfig() MainConfig {
	return MainConfig{
		LogLevel: "info", // everything but debug is the default
		RootChain: []RootChain{
			{
				ChainId: CanopyChainId,            // RootChainId is chain id 1
				Url:     "http://localhost:50002", // RooChainURL points to self
			},
		},
		RunVDF:          true,          // run the VDF by default
		ChainId:         CanopyChainId, // default chain url is 1
		Headless:        false,         // serve the web wallet and block explorer by default
		AutoUpdate:      true,          // set it as default while in inmature state
		PluginTimeoutMS: 1000,          // 1 second default plugin timeout
	}
}

// GetLogLevel() parses the log string in the config file into a LogLevel Enum
func (m *MainConfig) GetLogLevel() int32 {
	switch {
	case strings.Contains(strings.ToLower(m.LogLevel), "deb"):
		return DebugLevel
	case strings.Contains(strings.ToLower(m.LogLevel), "inf"):
		return InfoLevel
	case strings.Contains(strings.ToLower(m.LogLevel), "war"):
		return WarnLevel
	case strings.Contains(strings.ToLower(m.LogLevel), "err"):
		return ErrorLevel
	default:
		return DebugLevel
	}
}

// RPC CONFIG BELOW

type RPCConfig struct {
	WalletPort                 string `json:"walletPort"`                 // the port where the web wallet is hosted
	ExplorerPort               string `json:"explorerPort"`               // the port where the block explorer is hosted
	RPCPort                    string `json:"rpcPort"`                    // the port where the rpc server is hosted
	AdminPort                  string `json:"adminPort"`                  // the port where the admin rpc server is hosted
	ProfilingPort              string `json:"profilingPort"`              // the port where the pprof profiling server is hosted
	RPCUrl                     string `json:"rpcURL"`                     // the url where the rpc server is hosted
	AdminRPCUrl                string `json:"adminRPCUrl"`                // the url where the admin rpc server is hosted
	TimeoutS                   int    `json:"timeoutS"`                   // the rpc request timeout in seconds
	IndexerBlobCacheEntries    int    `json:"indexerBlobCacheEntries"`    // number of cached indexer blobs to keep in memory
	MaxRCSubscribers           int    `json:"maxRCSubscribers"`           // max total root-chain subscribers
	MaxRCSubscribersPerChain   int    `json:"maxRCSubscribersPerChain"`   // max root-chain subscribers per chain id
	RCSubscriberReadLimitBytes int64  `json:"rcSubscriberReadLimitBytes"` // max bytes allowed in a single ws message from a subscriber
	RCSubscriberWriteTimeoutMS int    `json:"rcSubscriberWriteTimeoutMS"` // ws write timeout for publishing root-chain info
	RCSubscriberPongWaitS      int    `json:"rcSubscriberPongWaitS"`      // time to wait for pong responses
	RCSubscriberPingPeriodS    int    `json:"rcSubscriberPingPeriodS"`    // how often to ping subscribers
}

// RootChain defines a rpc url to a possible 'root chain' which is used if the governance parameter RootChainId == ChainId
type RootChain struct {
	ChainId uint64 `json:"chainId"` // used if the governance parameter RootChainId == ChainId
	Url     string `json:"url"`     // the url to the 'root chain' rpc
}

// DefaultRPCConfig() sets rpc url to localhost and sets wallet, explorer, rpc, and admin ports from [50000-50003]
func DefaultRPCConfig() RPCConfig {
	return RPCConfig{
		WalletPort:                 "50000",                    // find the wallet on localhost:50000
		ExplorerPort:               "50001",                    // find the explorer on localhost:50001
		RPCPort:                    "50002",                    // the rpc is served on localhost:50002
		AdminPort:                  "50003",                    // the admin rpc is served on localhost:50003
		ProfilingPort:              "6060",                     // the pprof profiling server is served on localhost:6060
		RPCUrl:                     "http://localhost:50002",   // use a local rpc by default
		AdminRPCUrl:                "http://localhost:50003",   // use a local admin rpc by default
		TimeoutS:                   3,                          // the rpc timeout is 3 seconds
		IndexerBlobCacheEntries:    64,                         // cache the most recent indexer blobs
		MaxRCSubscribers:           512,                        // limit total root-chain subscribers
		MaxRCSubscribersPerChain:   128,                        // limit subscribers per chain id
		RCSubscriberReadLimitBytes: int64(64 * units.Kilobyte), // cap inbound ws message sizes
		RCSubscriberWriteTimeoutMS: 10000,                      // 10s write deadline for publishes
		RCSubscriberPongWaitS:      60,                         // 60s pong wait
		RCSubscriberPingPeriodS:    50,                         // 50s ping interval
	}
}

// STATE MACHINE CONFIG BELOW

// defaults for on-chain minting schedule
const (
	// the number of tokens in micro denomination that are initially (before halvenings) minted per block
	DefaultInitialTokensPerBlock = uint64(80 * 1000000) // 80 CNPY
	// the number of blocks between each halvening (block reward is cut in half) event
	DefaultBlocksPerHalvening = uint64(3150000) // ~ 2 years - 20 second blocks
)

// StateMachineConfig houses FSM level options
type StateMachineConfig struct {
	InitialTokensPerBlock uint64   `json:"initialTokensPerBlock"`         // initial micro tokens minted per block (before halvenings)
	BlocksPerHalvening    uint64   `json:"blocksPerHalvening"`            // number of blocks between block reward halvings
	FaucetAddress         string   `json:"faucetAddress"`                 // if set: "send" txs from this address will auto-mint on insufficient funds (dev/test only)
	RestrictedAddresses   []string `json:"restrictedAddresses,omitempty"` // additional locally restricted addresses used when voting on proposals
}

// DefaultStateMachineConfig returns FSM defaults
func DefaultStateMachineConfig() StateMachineConfig {
	return StateMachineConfig{
		InitialTokensPerBlock: DefaultInitialTokensPerBlock,
		BlocksPerHalvening:    DefaultBlocksPerHalvening,
		FaucetAddress:         "",
	}
}

// CONSENSUS CONFIG BELOW

// ConsensusConfig defines the consensus phase timeouts for bft synchronicity
// NOTES:
// - BlockTime = ElectionTimeout + ElectionVoteTimeout + ProposeTimeout + ProposeVoteTimeout + PrecommitTimeout + PrecommitVoteTimeout + CommitTimeout + CommitProcess
// - async faults may lead to extended block time
// - social consensus dictates BlockTime for the protocol - being oo fast or too slow can lead to Non-Signing and Consensus failures
type ConsensusConfig struct {
	NewHeightTimeoutMs      int `json:"newHeightTimeoutMS"`      // how long (in milliseconds) the replica sleeps before moving to the ELECTION phase
	ElectionTimeoutMS       int `json:"electionTimeoutMS"`       // minus VRF creation time (if Candidate), is how long (in milliseconds) the replica sleeps before moving to ELECTION-VOTE phase
	ElectionVoteTimeoutMS   int `json:"electionVoteTimeoutMS"`   // minus QC validation + vote time, is how long (in milliseconds) the replica sleeps before moving to PROPOSE phase
	ProposeTimeoutMS        int `json:"proposeTimeoutMS"`        // minus Proposal creation time (if Leader), is how long (in milliseconds) the replica sleeps before moving to PROPOSE-VOTE phase
	ProposeVoteTimeoutMS    int `json:"proposeVoteTimeoutMS"`    // minus QC validation + vote time, is how long (in milliseconds) the replica sleeps before moving to PRECOMMIT phase
	PrecommitTimeoutMS      int `json:"precommitTimeoutMS"`      // minus Proposal-QC aggregation time (if Leader), how long (in milliseconds) the replica sleeps before moving to the PRECOMMIT-VOTE phase
	PrecommitVoteTimeoutMS  int `json:"precommitVoteTimeoutMS"`  // minus QC validation + vote time, is how long (in milliseconds) the replica sleeps before moving to COMMIT phase
	CommitTimeoutMS         int `json:"commitTimeoutMS"`         // minus Precommit-QC aggregation time (if Leader), how long (in milliseconds) the replica sleeps before moving to the COMMIT-PROCESS phase
	RoundInterruptTimeoutMS int `json:"roundInterruptTimeoutMS"` // minus gossiping current Round time, how long (in milliseconds) the replica sleeps before moving to PACEMAKER phase
}

// DefaultConsensusConfig() configures the block time
func DefaultConsensusConfig() ConsensusConfig {
	return ConsensusConfig{
		NewHeightTimeoutMs:     4500, // 4.5 seconds
		ElectionTimeoutMS:      1500, // 1.5 seconds
		ElectionVoteTimeoutMS:  1500, // 1.5 seconds
		ProposeTimeoutMS:       2500, // 2.5 seconds
		ProposeVoteTimeoutMS:   4000, // 4 seconds
		PrecommitTimeoutMS:     2000, // 2 seconds
		PrecommitVoteTimeoutMS: 2000, // 2 seconds
		CommitTimeoutMS:        2000, // 2 seconds
	}
}

// BlockTimeMS() returns the expected block time in milliseconds
func (c *ConsensusConfig) BlockTimeMS() int {
	return c.NewHeightTimeoutMs +
		c.ElectionTimeoutMS +
		c.ElectionVoteTimeoutMS +
		c.ProposeTimeoutMS +
		c.ProposeVoteTimeoutMS +
		c.PrecommitTimeoutMS +
		c.PrecommitVoteTimeoutMS +
		c.CommitTimeoutMS
}

// P2P CONFIG BELOW

// P2PConfig defines peering compatibility and limits as well as actions on specific peering IPs / IDs
type P2PConfig struct {
	NetworkID           uint64            `json:"networkID"`           // the ID for the peering network
	ListenAddress       string            `json:"listenAddress"`       // listen for incoming connection
	ExternalAddress     string            `json:"externalAddress"`     // advertise for external dialing
	MaxInbound          int               `json:"maxInbound"`          // max inbound peers
	MaxOutbound         int               `json:"maxOutbound"`         // max outbound peers
	TrustedPeerIDs      []string          `json:"trustedPeerIDs"`      // trusted public keys
	DialPeers           []string          `json:"dialPeers"`           // peers to consistently dial until expo-backoff fails (format pubkey@ip:port)
	BannedPeerIDs       []string          `json:"bannedPeersIDs"`      // banned public keys
	BannedIPs           []string          `json:"bannedIPs"`           // banned IPs
	MinimumPeersToStart int               `json:"minimumPeersToStart"` // the minimum connections required to start consensus
	ValidatorTCPProxy   map[uint64]string `json:"validator_tcp_proxy"` // tcp proxy config mapping listen port to target address
	GossipThreshold     uint              `json:"gossipThreshold"`     // number of must connects needed to switch to full gossip
}

func DefaultP2PConfig() P2PConfig {
	return P2PConfig{
		NetworkID:           CanopyMainnetNetworkId,
		ListenAddress:       "0.0.0.0:9001",      // default TCP address is 9001 for chain 1 (9002 for chain 2 etc.)
		ExternalAddress:     "",                  // should be populated by the user
		MaxInbound:          21,                  // inbounds should be close to 3x greater than outbounds
		MaxOutbound:         7,                   // to ensure 'new joiners' have slots to take
		MinimumPeersToStart: 0,                   // requires no peers to start consensus by default (suitable for 1 node network)
		ValidatorTCPProxy:   map[uint64]string{}, // initialize the map
	}
}

// STORE CONFIG BELOW

// StoreConfig is user configurations for the key value database
type StoreConfig struct {
	DataDirPath               string `json:"dataDirPath"`               // path of the designated folder where the application stores its data
	DBName                    string `json:"dbName"`                    // name of the database
	IndexByAccount            bool   `json:"indexByAccount"`            // index transactions by account
	InMemory                  bool   `json:"inMemory"`                  // non-disk database, only for testing
	StateChangeJournalEnabled bool   `json:"stateChangeJournalEnabled"` // persist state-change keys for indexer blob deltas
	// recommended range: 500-2000 for optimal performance. Values below 500 increase disk I/O
	// by several orders of magnitude, reducing performance and accelerating disk degradation during
	// sync. Lower values also increase the risk of data loss due to a pebble issue where batches are
	// returned before commit completion when compaction runs concurrently with commits.
	LSSCompactionInterval uint64 `json:"lssCompactionInterval"` // interval for compacting latest store data
	BackupDirectory       string `json:"backupDirectory"`       // directory where backups of the database are stored
	BackupInterval        uint64 `json:"backupInterval"`        // interval in blocks for creating backups of the database (0 to disable automatic backups)
	CompressionProfile    string `json:"compressionProfile"`    // the pebbledb compression profile to use.
}

// DefaultDataDirPath() is $USERHOME/.canopy
func DefaultDataDirPath() string {
	// get the user home
	home, err := os.UserHomeDir()
	// home, err := os.Getwd()
	// if unable to get the user home
	if err != nil {
		// fatal error
		panic(err)
	}
	// exit with full default data directory path
	// return filepath.Join(home, "canopy_2")
	return filepath.Join(home, ".canopy")
}

// DefaultStoreConfig() returns the developer recommended store configuration
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		DataDirPath:               DefaultDataDirPath(),                      // use the default data dir path
		DBName:                    "canopy",                                  // 'canopy' database name
		IndexByAccount:            true,                                      // index transactions by account
		InMemory:                  false,                                     // persist to disk, not memory
		StateChangeJournalEnabled: false,                                     // state-change journaling is disabled by default
		LSSCompactionInterval:     uint64(rand.Int32N(101) + 500),            // clean every 500-600 blocks (random)
		BackupDirectory:           path.Join(DefaultDataDirPath(), "backup"), // backup directory name
		BackupInterval:            0,                                         // backups disabled by default
		CompressionProfile:        "zstd",
	}
}

// MEMPOOL CONFIG BELOW

// MempoolConfig is the user configuration of the unconfirmed transaction pool
type MempoolConfig struct {
	MaxTotalBytes              uint64 `json:"maxTotalBytes"`              // maximum collective bytes in the pool
	MaxTransactionCount        uint32 `json:"maxTransactionCount"`        // max number of Transactions
	IndividualMaxTxSize        uint32 `json:"individualMaxTxSize"`        // max bytes of a single Transaction
	DropPercentage             int    `json:"dropPercentage"`             // percentage that is dropped from the bottom of the queue if limits are reached
	LazyMempoolCheckFrequencyS int    `json:"lazyMempoolCheckFrequencyS"` // how often the mempool is checked for new transactions besides the mandatory (after Commit) (0) for none
}

// DefaultMempoolConfig() returns the developer created Mempool options
func DefaultMempoolConfig() MempoolConfig {
	return MempoolConfig{
		MaxTotalBytes:              uint64(10 * units.MB),      // 10 MB max size
		MaxTransactionCount:        5000,                       // 5000 max transactions
		IndividualMaxTxSize:        uint32(4 * units.Kilobyte), // 4 KB max individual tx size
		DropPercentage:             35,                         // drop 35% if limits are reached
		LazyMempoolCheckFrequencyS: 2,                          // check every 2 seconds
	}
}

// MetricsConfig represents the configuration for the metrics server
type MetricsConfig struct {
	MetricsEnabled         bool   `json:"metricsEnabled"`         // if the metrics are enabled
	PrometheusAddress      string `json:"prometheusAddress"`      // the address of the server
	HeapProfilingEnabled   bool   `json:"heapProfilingEnabled"`   // enable periodic heap profiling (warning: causes GC pauses)
	HeapProfilingIntervalS int    `json:"heapProfilingIntervalS"` // interval in seconds between heap profile snapshots
}

// DefaultMetricsConfig() returns the default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		MetricsEnabled:         true,           // enabled by default
		PrometheusAddress:      "0.0.0.0:9090", // the default prometheus address
		HeapProfilingEnabled:   false,          // disabled by default (causes GC pauses)
		HeapProfilingIntervalS: 10,             // 10 second interval when enabled
	}
}

// WriteToFile() saves the Config object to a JSON file
func (c Config) WriteToFile(filepath string) error {
	// convert the config to indented 'pretty' json bytes
	jsonBytes, err := json.MarshalIndent(c, "", "  ")
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return err
	}
	// write the config.json file to the data directory
	return os.WriteFile(filepath, jsonBytes, 0600)
}

// NewConfigFromFile() populates a Config object from a JSON file
func NewConfigFromFile(filepath string) (Config, error) {
	// read the file into bytes using
	fileBytes, err := os.ReadFile(filepath)
	// if an error occurred
	if err != nil {
		// exit with error
		return Config{}, err
	}
	// define the default config to fill in any blanks in the file
	c := DefaultConfig()
	// populate the default config with the file bytes
	if err = json.Unmarshal(fileBytes, &c); err != nil {
		// exit with error
		return Config{}, err
	}
	// exit
	return c, nil
}

// RestrictedAddresses were populated from OFAC's 2026-08-28 SDN.XML: https://sanctionslistservice.ofac.treas.gov/api/PublicationPreview/exports/SDN.XML
var RestrictedAddresses = map[string]struct{}{
	"0330070fd38ec3bb94f58fa55d40368271e9e54a": {},
	"038989cbb1710c72b9920dc4fa529158f463e72c": {},
	"04dba1194ee10112fe6c3207c0687def0e78bacf": {},
	"08723392ed15743cc38513c4925f5e6be5c17243": {},
	"08b2efdcdb8822efe5ad0eae55517cf5dc544251": {},
	"0931ca4d13bb4ba75d9b7132ab690265d749a5e7": {},
	"098b716b8aaf21512996dc57eb0615e2383e2f96": {},
	"0ee5067b06776a89ccc7dc8ee369984ad7db5e06": {},
	"12de548f79a50d2bd05481c8515c1ef5183666a9": {},
	"14779cec0b117d5194c750c55ea1f42086631964": {},
	"175d44451403edf28469df03a9280c1197adb92c": {},
	"1967d8af5bd86a497fb3dd7899a020e47560daaf": {},
	"1999ef52700c34de7ec2b68a28aafb37db0c5ade": {},
	"19aa5fe80d33a56d56c78e82ea5e50e5d80b4dff": {},
	"19f8f2b0915daa12a3f5c9cf01df9e24d53794f7": {},
	"1b8579cf6ab12ea6b74ac5fa41f3829a3cb61e6e": {},
	"1cab8177ace78b1b6b1c393371f4f2dcae40cbeb": {},
	"1d19b52b54e7ef5ea1a4b40b616165e798eac9f8": {},
	"1da5821544e25c636c1417ba96ade4cf6d2f9b5a": {},
	"21b8d56bda776bbe68655a16895afd96f5534fed": {},
	"252a8bd2319d8a555b872990601221b3a2053bce": {},
	"2711d73d559f62f4f855ee21f38378f528e07985": {},
	"2c7dcd774b33e10367f7d6385479e04f97d179dc": {},
	"2f389ce8bd8ff92de3402ffce4691d17fc4f6535": {},
	"308ed4b7b49797e1a98d3818bff6fe5385410370": {},
	"32da24ca413f3e7b53145d4737e172c3bdf81e3e": {},
	"35fb6f6db4fb05e6a4ce86f2c93691425626d4b1": {},
	"38735f03b30fbc022ddd06abed01f0ca823c6a94": {},
	"39d908dac893cbcb53cc86e0ecc369aa4def1a29": {},
	"3ad9db589d201a710ed237c829c7860ba86510fc": {},
	"3cbded43efdaf0fc77b9c55f6fc9988fcc9b757d": {},
	"3cffd56b47b7b41c56258d9c7731abadc360e073": {},
	"3e37627deaa754090fbfbb8bd226c1ce66d255e9": {},
	"4060cbf80734193f521a3cc6fd4e985df2825279": {},
	"43fa21d92141ba9db43052492e0deee5aa5f0a93": {},
	"48549a34ae37b12f6a30566245176994e17c6b4a": {},
	"4f428c11dc82388fa5136d636e613ad923eb700b": {},
	"4f47bc496083c727c5fbe3ce9cdf2b0f6496270c": {},
	"502371699497d08d5339c870851898d6d72521dd": {},
	"530a64c0ce595026a4a556b703644228179e2d57": {},
	"532b77b33a040587e9fd1800088225f99b8b0e8a": {},
	"53b6936513e738f44fb50d2b9476730c0ab3bfc1": {},
	"5512d943ed1f7c8a43f3435c85f7ab68b30121b0": {},
	"56de1527136f76a809e5b14ded6103eecd072ba7": {},
	"57ec89a0c056163a0314e413320f9b3abe761259": {},
	"5a14e72060c11313e38738009254a90968f58f51": {},
	"5a7a51bfb49f190e5a6060a5bc6052ac14a3b59f": {},
	"5d5b5dafecbf31bdb08bfd3edad4f2694372d0ef": {},
	"5f48c2a71b2cc96e3f0ccae4e39318ff0dc375b2": {},
	"67d40ee1a85bf4a4bb7ffae16de985e8427b6b45": {},
	"6b0736fed0634e15e19cc57fba19cd179c13abca": {},
	"6b69e2a7545c166417a80c61a77562052bffa9c5": {},
	"6be0ae71e6c41f2f9d0d1a3b8d0f75e6f6a0b46e": {},
	"6f1ca141a28907f78ebaa64fb83a9088b02a8352": {},
	"6fac4d18c912343bf86fa7049364dd4e424ab9c0": {},
	"72a5843cc08275c8171e582972aa4fda8c397b2a": {},
	"747afb5c7a7fc34b547cd0fdebf9b91759c5a52b": {},
	"76ea76ca4eb727f18956ab93445a94c5280412b9": {},
	"797d7ae72ebddcdea2a346c1834e04d1f8df102b": {},
	"7ced75026204ac29c34bea98905d4c949f27361e": {},
	"7db418b5d567a4e0e8c59ad71be1fce48f3e6107": {},
	"7f03679b56d8772530efa516b58bb83d4829e881": {},
	"7f19720a857f834887fc9a7bc0a0fbe7fc7f8102": {},
	"7f367cc41522ce07553e823bf3be79a889debe1b": {},
	"7ff9cfad3877f21d41da833e2f775db0569ee3d9": {},
	"83e5bc4ffa856bb84bb88581f5dd62a433a25e0d": {},
	"8576acc5c05d6ce88f4e49bf65bdf0c62f91353c": {},
	"8694ed130432be2cd3efff2e4d9dc52351dc7423": {},
	"8ac5381fcd9e7395d14e02986c344aada84b4bc6": {},
	"8d79c73daae8630c88de372ba8f57592fa987607": {},
	"8dce2aac0de82bdcaf6b4373b79f94331b8e4995": {},
	"901bb9583b24d97e995513c6778dc6888ab6870e": {},
	"931546d9e66836abf687d2bc64b30407bac8c568": {},
	"95584c303fcd48af5c6b9873015f2ad0ca84eae3": {},
	"961c5be54a2ffc17cf4cb021d863c42dacd47fc1": {},
	"9697749a9e8d6c119d8eeb0d6268a1b99c40684c": {},
	"97b1043abd9e6fc31681635166d430a458d14f9c": {},
	"983a81ca6fb1e441266d2fbcb7d8e530ac2e05a2": {},
	"9be599d7867f5e1a2d7ec6db9710df2b98a15573": {},
	"9c2bc757b66f24d60f016b6237f8cdd414a879fa": {},
	"9dd7fa4b4950154f7e75bdd8a77266b99b94ec08": {},
	"9f4cda013e354b8fc285bf4b9a60460cee7f7ea9": {},
	"a0e1c89ef1a489c9c7de96311ed5ce5d32c20e4b": {},
	"a40cfbfc8534ffc84e20a7d8bbc3729b26a35f6f": {},
	"a7e5d5a720f06526557c513402f2e6b5fa20b008": {},
	"ac4cc4b68ea24bbfaac8fd127b67ed445accce22": {},
	"b338962b92cd818d6aef0a32a9ecd01212a71f33": {},
	"b5a69da691670f62510793f79a9b36c7db1a7b7c": {},
	"b637f84b66876ebf609c2a4208905f9ddac9d075": {},
	"b6f5ec1a0a9cd1526536d3f0426c429529471f40": {},
	"bb69e01921b17cd22080968bcc96ba6115da6062": {},
	"bd3276f265b83b5e828c05f46cde9d10a1521a24": {},
	"c103b7dc095c904b92081eef0c1640081ec01c10": {},
	"c2a3829f459b3edd87791c74cd45402ba0a20be3": {},
	"c455f7fd3e0e12afd51fba5c106909934d8a0e4a": {},
	"cb74874f1e06fcf80a306e06e5379a44b488ba2d": {},
	"d04e33461fea8302c5e1e13895b60cee8aefda7f": {},
	"d0975b32cea532eadddfc9c60481976e39db3472": {},
	"d5ed34b52ac4ab84d8fa8a231a3218bbf01ed510": {},
	"d81414abc631c6cadae1c6198b0c2b15a9b4fde5": {},
	"d8500c631dc32fa18645b7436344a99e4825e10e": {},
	"d882cfc20f52f2599d84b8e8d58c7fb62cfe344b": {},
	"db2720ebad55399117ddb4c4a4afd9a4ccada8fe": {},
	"dcbeffbecce100cce9e4b153c4e15cb885643193": {},
	"e05f529f5284d75624eba386cb716928c3b54a2a": {},
	"e1d865c3d669dcc8c57c8d023140cb204e672ee4": {},
	"e1e4c5e5ed8f03ae61b581e2def126025f2b9401": {},
	"e3d35f68383732649669aa990832e017340dbca5": {},
	"e7aa314c77f4233c18c6cc84384a9247c0cf367b": {},
	"e950dc316b836e4eefb8308bf32bf7c72a1358ff": {},
	"eb507efa9ee692a4c774ad1de9f3cb26fc459da3": {},
	"ed6e0a7e4ac94d976eebfb82ccf777a3c6bad921": {},
	"ef85a6fafa5942a964dc618e94e230881d29ce2a": {},
	"efe301d259f525ca1ba74a7977b80d5b060b3cca": {},
	"f1c4c44d2dcbcfa704349e3b57628dbd8404e597": {},
	"f2235d55b2950a0b1317469d72d07ae65b2e27cb": {},
	"f3701f445b6bdafedbca97d1e477357839e4120d": {},
	"f4377eda661e04b6dda78969796ed31658d602d4": {},
	"f45ecc3a59c7911181c659ce9115854c6175be91": {},
	"f7b31119c2682c88d88d455dbb9d5932c65cf1be": {},
	"fac583c0cf07ea434052c49115a4682172ab6b4f": {},
	"fb3eff152ea55d1bfa04dbdd509a80fd7b72cdeb": {},
	"fda1ec4a6178d4916b001a065422d31ebe5f62ff": {},
	"fec8a60023265364d066a1212fde3930f6ae8da7": {},
}
