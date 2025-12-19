package lib

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/* This file implements dev-ops telemetry for the node in the form of prometheus metrics */

// GUARD RAILS DOCUMENTATION:
// *************************************************************************************************************
// This section describes 1) hard limits and 2) soft limit alert recommendations for health related metrics
//
// Metric Name          | Hard Limit  | Soft Limit | Note
// --------------------------------------------------------------------------------------------------------------------------------------
// NodeStatus           | 0           | n/a        |
// TotalPeers           | 0 peers     | 1 peer     |
// LastHeightTime       | n/a         | 5 min      | Just over 3 rounds at 20s blocks
// ValidatorStatus      | n/a         | not 1      | Monitor unexpected Pause or Unstaking
// BFTRound             | n/a         | 3 rounds   | Soft = Just below the 'LastHeight' time
// BFTElectionTime      | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// BFTElectionVoteTime  | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// BFTProposeTime       | 4 secs      | 3 secs     | Hard = config, Soft = 75% of config timing
// BFTProposeVoteTime   | 4 secs      | 3 secs     | Hard = config, Soft = 75% of config timing
// BFTPrecommitTime     | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// BFTPrecommitVoteTime | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// BFTCommitTime        | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// BFTCommitProcessTime | 2 secs      | 1.5 secs   | Hard = config, Soft = 75% of config timing
// NonSignerPercent     | 33%         | 10%        | Hard = BFT upper bound
// LargestTxSize        | 4KB         | 3KB        | Hard = default mempool config, Soft = 75% of hard
// BlockSize            | 1MB-1652B   | 750KB      | Hard = param - MaxBlockHeader, Soft = 75% of param
// BlockProcessingTime  | 4 secs      | 3 secs     | Hard = MIN(ProposeTimeoutMS, ProposeVoteTimeoutMS)
// BlockVDFIterations   | n/a         | 0          | Soft = unexpected behavior
// RootChainInfoTime    | 2 secs      | 1 sec      | Hard = 10% of block time
// DBPartitionTime      | 10 min      | 5 min      | Hard = arbitrary / high likelihood of interruption
// DBPartitionEntries   | 2,000,000   | 1,500,000  | Hard = Badger default limit (configurable)
// DBPartitionSize      | 128MB       | 75MB       | Hard = Badger set limit (configurable)
// DBCommitTime         | 3 secs      | 2 secs     | Hard = soft of BlockProcessingTime
// DBCommitEntries      | 2,000,000   | 1,500,000  | Hard = Badger default limit (configurable)
// DBCommitSize         | 128MB       | 10MB       | Hard = Badger set limit (configurable)
// MempoolSize          | 10MB        | 2MB        | Hard = default config, Soft = 2 blocks
// MempoolCount         | 5,000       | 3,500      | Hard = default config, Soft = 75% of hard
// DoubleSignerCount    | 1           | n/a        | Hard = any double signer
// DoubleSigner         | 1           | n/a        | Hard = any double sign
// NonSignerCount       | 50          | 20         | Hard = arbitrary, Soft = arbitrary
// NonSigner            | 2           | 1          | Hard = repeat offense, Soft = first occurrence

const metricsPattern = "/metrics"

// Metrics represents a server that exposes Prometheus metrics
type Metrics struct {
	server          *http.Server  // the http prometheus server
	chainID         float64       // the chain id the node is running
	softwareVersion string        // the sofware version the node is running
	config          MetricsConfig // the configuration
	nodeAddress     []byte        // the node's address
	log             LoggerI       // the logger
	startupBlockSet bool          // flag to ensure startup block is only set once

	NodeMetrics             // general telemetry about the node
	BlockMetrics            // block telemetry
	PeerMetrics             // peer telemetry
	P2PMetrics              // p2p performance telemetry
	BFTMetrics              // bft telemetry
	FSMMetrics              // fsm telemetry
	StoreMetrics            // persistence telemetry
	MempoolMetrics          // tx memory pool telemetry
	OracleMetrics           // oracle telemetry
	EthBlockProviderMetrics // ethereum block provider telemetry
}

// NodeMetrics represents general telemetry for the node's health
type NodeMetrics struct {
	NodeStatus       prometheus.Gauge     // is the node alive?
	SyncingStatus    prometheus.Gauge     // is the node syncing?
	GetRootChainInfo prometheus.Histogram // how long does the 'GetRootChainInfo' call take?
	AccountBalance   *prometheus.GaugeVec // what's the balance of this node's account?
	ProposerCount    prometheus.Counter   // how many times did this node propose the block?
	ChainId          prometheus.Gauge     // what chain id is this node running on?
	SoftwareVersion  *prometheus.GaugeVec // what software version is this node running?
	StartupBlock     prometheus.Gauge     // the block height when node first completed syncing (set only once)
}

// BlockMetrics represents telemetry for block health
type BlockMetrics struct {
	BlockProcessingTime prometheus.Histogram // how long does it take for this node to commit a block?
	BlockSize           prometheus.Gauge     // what is the size of the block in bytes?
	BlockNumTxs         prometheus.Gauge     // how many transactions has the node processed?
	LargestTxSize       prometheus.Gauge     // what is the largest tx size in a block?
	BlockVDFIterations  prometheus.Gauge     // how many vdf iterations are included in the block?
	NonSignerPercent    prometheus.Gauge     // what percent of the voting power were non signers
}

// PeerMetrics represents the telemetry for the P2P module
type PeerMetrics struct {
	TotalPeers    prometheus.Gauge // number of peers
	InboundPeers  prometheus.Gauge // number of peers that dialed this node
	OutboundPeers prometheus.Gauge // number of peers that this node dialed
}

// P2PMetrics represents detailed performance telemetry for P2P message sending and receiving
type P2PMetrics struct {
	SendQueueTime       prometheus.Histogram   // time a packet spends waiting in the send queue
	SendWireTime        prometheus.Histogram   // time to write a packet to the wire
	SendTotalTime       prometheus.Histogram   // total time from Send() call to wire write completion
	ReceiveWireTime     prometheus.Histogram   // time to read a packet from the wire TODO Review as this one is not reliable in all scenarios
	ReceiveAssemblyTime prometheus.Histogram   // time to assemble packets into a complete message
	SendQueueDepth      *prometheus.GaugeVec   // current depth of send queue by topic
	InboxQueueDepth     *prometheus.GaugeVec   // current depth of inbox queue by topic
	MessageSize         prometheus.Histogram   // size of messages in bytes
	PacketsPerMessage   prometheus.Histogram   // number of packets per message
	SendQueueTimeout    prometheus.Counter     // count of send queue timeout errors
	SendQueueFull       *prometheus.CounterVec // count of send queue full events by topic
}

// BFTMetrics represents the telemetry for the BFT module
type BFTMetrics struct {
	Height            prometheus.Gauge     // what's the height of this chain?
	Round             prometheus.Gauge     // what's the current BFT round
	Phase             prometheus.Gauge     // what's the current BFT phase
	ElectionTime      prometheus.Histogram // how long did the election phase take?
	ElectionVoteTime  prometheus.Histogram // how long did the election vote phase take?
	ProposeTime       prometheus.Histogram // how long did the propose phase take?
	ProposeVoteTime   prometheus.Histogram // how long did the propose vote phase take?
	PrecommitTime     prometheus.Histogram // how long did the precommit phase take?
	PrecommitVoteTime prometheus.Histogram // how long did the precommit vote phase take?
	CommitTime        prometheus.Histogram // how long did the commit phase take?
	CommitProcessTime prometheus.Histogram // how long did the commit process phase take?
	RootHeight        prometheus.Gauge     // what's the height of the root-chain?
	RootChainId       prometheus.Gauge     // what's the chain id of the root-chain?
}

// FSMMetrics represents the telemetry of the FSM module for the node's address
type FSMMetrics struct {
	ValidatorStatus            *prometheus.GaugeVec // what's the status of this validator?
	ValidatorType              *prometheus.GaugeVec // what's the type of this validator?
	ValidatorCompounding       *prometheus.GaugeVec // is this validator compounding?
	ValidatorStakeAmount       *prometheus.GaugeVec // what's the stake amount of this validator
	ValidatorBlockProducer     *prometheus.GaugeVec // was this validator a block producer? // TODO duplicate of canopy_proposer_count
	ValidatorNonSigner         *prometheus.GaugeVec // was this validator a non signer?
	ValidatorNonSignerCount    *prometheus.GaugeVec // was any validator a non signer?
	ValidatorDoubleSigner      *prometheus.GaugeVec // was this validator a double signer?
	ValidatorDoubleSignerCount *prometheus.GaugeVec // was any validator a double signer?
	ValidatorCount             *prometheus.GaugeVec // how many validators are there?
}

// StoreMetrics represents the telemetry of the 'store' package
type StoreMetrics struct {
	DBPartitionTime      prometheus.Histogram // how long does the db partition take?
	DBFlushPartitionTime prometheus.Histogram // how long does the db partition flush take?
	DBPartitionEntries   prometheus.Gauge     // how many entries in the partition batch?
	DBPartitionSize      prometheus.Gauge     // how big is the partition batch?
	DBCommitTime         prometheus.Histogram // how long does the db commit take?
	DBCommitEntries      prometheus.Gauge     // how many entries in the commit batch?
	DBCommitSize         prometheus.Gauge     // how big is the commit batch?
}

// MempoolMetrics represents the telemetry of the memory pool of pending transactions
type MempoolMetrics struct {
	MempoolSize    prometheus.Gauge // how many bytes are in the mempool?
	MempoolTxCount prometheus.Gauge // how many transactions are in the mempool?
}

// OracleMetrics represents the telemetry for the Oracle module
type OracleMetrics struct {
	// Block processing metrics
	OracleBlockProcessingTime prometheus.Histogram // how long does it take to process blocks?
	OrderValidationTime       prometheus.Histogram // how long does it take to validate orders?
	// Order counting metrics
	OrdersWitnessed prometheus.Counter // total orders witnessed
	OrdersValidated prometheus.Counter // total orders validated successfully
	OrdersSubmitted prometheus.Counter // total orders submitted for consensus
	OrdersRejected  prometheus.Counter // total orders rejected during validation
	// Order store metrics
	TotalOrdersStored prometheus.Gauge // total orders currently stored in order store
	LockOrdersStored  prometheus.Gauge // total lock orders currently stored
	CloseOrdersStored prometheus.Gauge // total close orders currently stored
	// State management metrics
	SafeHeight                prometheus.Gauge // current safe block height
	SourceChainHeight         prometheus.Gauge // current source chain height
	LockOrderSubmissionsSize  prometheus.Gauge // size of lock order submissions map
	CloseOrderSubmissionsSize prometheus.Gauge // size of close order submissions map
	// Error and reorg metrics
	ChainReorgs           prometheus.Counter // total chain reorganizations detected
	OrdersPruned          prometheus.Counter // total orders pruned during cleanup
	BlockProcessingErrors prometheus.Counter // total block processing errors
	// Performance metrics
	OrderBookUpdateTime prometheus.Histogram // how long does it take to update order book?
	RootChainSyncTime   prometheus.Histogram // how long does it take to sync with root chain?

	// Block height metrics
	LastProcessedHeight        prometheus.Gauge     // last source chain block height processed
	ConfirmationLag            prometheus.Gauge     // gap between source chain height and safe height
	OrdersAwaitingConfirmation prometheus.Gauge     // orders witnessed but not yet at safe height
	ReorgRollbackDepth         prometheus.Histogram // how many blocks reorgs roll back

	// Order lifecycle metrics
	OrdersNotInOrderbook prometheus.Counter // orders witnessed but not found in order book
	OrdersDuplicate      prometheus.Counter // duplicate orders (already in store)
	OrdersArchived       prometheus.Counter // orders successfully archived
	LockOrdersCommitted  prometheus.Counter // lock orders committed via certificate
	CloseOrdersCommitted prometheus.Counter // close orders committed via certificate

	// Validation failure metrics
	ValidationFailures *prometheus.CounterVec // validation failures by reason

	// Submission tracking metrics
	OrdersHeldAwaitingSafe  prometheus.Counter // orders not submitted due to safe height
	OrdersHeldProposeDelay  prometheus.Counter // orders held by ProposeDelayBlocks
	OrdersHeldResubmitDelay prometheus.Counter // orders held by resubmit cooldown
	LockOrderResubmissions  prometheus.Counter // lock orders resubmitted
	CloseOrderResubmissions prometheus.Counter // close orders resubmitted

	// Store operation metrics
	StoreWriteErrors  prometheus.Counter // order store write failures
	StoreReadErrors   prometheus.Counter // order store read failures
	StoreRemoveErrors prometheus.Counter // order store remove failures
}

// EthBlockProviderMetrics represents the telemetry for the Ethereum block provider
type EthBlockProviderMetrics struct {
	// Block and transaction processing metrics
	BlockFetchTime         prometheus.Histogram // how long does it take to fetch Ethereum blocks?
	TransactionProcessTime prometheus.Histogram // how long does it take to process Ethereum transactions?
	ReceiptFetchTime       prometheus.Histogram // how long does it take to fetch transaction receipts?
	// Token cache metrics
	TokenCacheHits   prometheus.Counter // total ERC20 token cache hits
	TokenCacheMisses prometheus.Counter // total ERC20 token cache misses
	// Connection and error metrics
	ConnectionErrors prometheus.Counter // total Ethereum connection errors
	// Processing counters
	BlocksProcessed       prometheus.Counter // total Ethereum blocks processed
	TransactionsProcessed prometheus.Counter // total Ethereum transactions processed
	TransactionRetries    prometheus.Counter // total Ethereum transaction processing retries

	// Connection & Sync Status Metrics (High Priority)
	RPCConnectionAttempts prometheus.Counter     // total RPC connection attempts
	RPCConnectionErrors   *prometheus.CounterVec // RPC connection errors by error type
	WSConnectionAttempts  prometheus.Counter     // total WebSocket connection attempts
	WSSubscriptionErrors  prometheus.Counter     // WebSocket subscription failures
	ConnectionState       prometheus.Gauge       // current connection state (0=disconnected, 1=connecting, 2=rpc_connected, 3=fully_connected)
	SyncStatus            prometheus.Gauge       // sync status (0=unsynced, 1=syncing, 2=synced)
	BlockHeightLag        prometheus.Gauge       // blocks behind chain head

	// Block Processing Metrics (High Priority)
	BlockFetchErrors        *prometheus.CounterVec // block fetch errors by error type
	BlockProcessingTimeouts prometheus.Counter     // blocks that timed out during processing
	ProcessBlocksBatchSize  prometheus.Histogram   // blocks processed per batch
	ReorgDetected           prometheus.Counter     // chain reorganizations detected

	// Transaction Processing Metrics (Medium Priority)
	TransactionsTotal           prometheus.Counter     // total transactions encountered
	TransactionParseErrors      *prometheus.CounterVec // TX parsing errors by error type
	TransactionRetryByAttempt   *prometheus.CounterVec // retry attempts by attempt number
	TransactionExhaustedRetries prometheus.Counter     // transactions that exhausted all retries
	TransactionSuccessStatus    *prometheus.CounterVec // TX success/failed/unknown breakdown
	ReceiptFetchErrors          prometheus.Counter     // receipt fetch failures

	// Order Detection Metrics (Medium Priority)
	ERC20TransferDetected prometheus.Counter     // ERC20 transfers detected
	LockOrderDetected     prometheus.Counter     // lock orders successfully parsed
	CloseOrderDetected    prometheus.Counter     // close orders successfully parsed
	OrderValidationErrors *prometheus.CounterVec // order validation errors by order_type and error_type

	// Token Cache Error Metrics (Medium Priority)
	TokenInfoFetchErrors      *prometheus.CounterVec // token info fetch errors by field
	TokenContractCallTimeouts prometheus.Counter     // token contract call timeouts
}

// NewMetricsServer() creates a new telemetry server
func NewMetricsServer(nodeAddress crypto.AddressI, chainID float64, softwareVersion string, config MetricsConfig, logger LoggerI) *Metrics {
	mux := http.NewServeMux()
	mux.Handle(metricsPattern, promhttp.Handler())
	return &Metrics{
		server:          &http.Server{Addr: config.PrometheusAddress, Handler: mux},
		config:          config,
		nodeAddress:     nodeAddress.Bytes(),
		chainID:         float64(chainID),
		softwareVersion: softwareVersion,
		log:             logger,
		// NODE
		NodeMetrics: NodeMetrics{
			NodeStatus: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_node_status",
				Help: "The node is alive and processing blocks",
			}),
			GetRootChainInfo: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_root_chain_info_time",
				Help: "The time it takes to process a 'GetRootChainInfo' call",
			}),
			SyncingStatus: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_syncing_status",
				Help: "Node syncing status (0 for syncing, 1 for synced)",
			}),
			ProposerCount: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_proposer_count",
				Help: "Total blocks produced by this node",
			}),
			AccountBalance: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_account_balance",
				Help: "Account balance in uCNPY of the node's address",
			}, []string{"address"}),
			ChainId: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_chain_id",
				Help: "The chain ID this node is running on",
			}),
			SoftwareVersion: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_software_version",
				Help: "The software version this node is running",
			}, []string{"version"}),
			StartupBlock: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_startup_block",
				Help: "The block height when node first completed syncing after startup (set only once per run)",
			}),
		},
		// BLOCK
		BlockMetrics: BlockMetrics{
			BlockProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_block_processing_time",
				Help: "The time it takes to process a received canopy block in seconds",
			}),
			BlockSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_block_size",
				Help: "The size of the last block in bytes",
			}),
			BlockNumTxs: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_block_num_txs",
				Help: "The number of transactions in the last canopy block",
			}),
			LargestTxSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_block_largest_txn",
				Help: "The largest transactions in the last canopy block in bytes",
			}),
			BlockVDFIterations: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_block_vdf_iterations",
				Help: "The number of vdf iterations in the last canopy block",
			}),
			NonSignerPercent: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_block_non_signer_percentage",
				Help: "The percent (%) of voting power that did not sign the last block",
			}),
		},
		// PEER
		PeerMetrics: PeerMetrics{
			TotalPeers: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_peer_total",
				Help: "Total number of peers",
			}),
			InboundPeers: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_peer_inbound",
				Help: "Number of inbound peers",
			}),
			OutboundPeers: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_peer_outbound",
				Help: "Number of outbound peers",
			}),
		},
		// P2P Performance
		P2PMetrics: P2PMetrics{
			SendQueueTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_send_queue_time_seconds",
				Help:    "Time a packet spends waiting in the send queue before being sent",
				Buckets: prometheus.DefBuckets,
			}),
			SendWireTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_send_wire_time_seconds",
				Help:    "Time to write a packet to the wire (network)",
				Buckets: prometheus.DefBuckets,
			}),
			SendTotalTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_send_total_time_seconds",
				Help:    "Total time from Send() call to wire write completion",
				Buckets: prometheus.DefBuckets,
			}),
			ReceiveWireTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_receive_wire_time_seconds",
				Help:    "Time to read a packet from the wire (network)",
				Buckets: prometheus.DefBuckets,
			}),
			ReceiveAssemblyTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_receive_assembly_time_seconds",
				Help:    "Time to assemble packets into a complete message",
				Buckets: prometheus.DefBuckets,
			}),
			SendQueueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_p2p_send_queue_depth",
				Help: "Current depth of send queue by topic",
			}, []string{"topic"}),
			InboxQueueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_p2p_inbox_queue_depth",
				Help: "Current depth of inbox queue by topic",
			}, []string{"topic"}),
			MessageSize: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_message_size_bytes",
				Help:    "Size of messages in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~100MB
			}),
			PacketsPerMessage: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_p2p_packets_per_message",
				Help:    "Number of packets per message",
				Buckets: prometheus.LinearBuckets(1, 1, 20), // 1 to 20 packets
			}),
			SendQueueTimeout: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_p2p_send_queue_timeout_total",
				Help: "Total count of send queue timeout errors",
			}),
			SendQueueFull: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_p2p_send_queue_full_total",
				Help: "Total count of send queue full events by topic",
			}, []string{"topic"}),
		},
		// BFT
		BFTMetrics: BFTMetrics{
			Height: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_height",
				Help: "Current height of consensus",
			}),
			Round: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_round",
				Help: "Current round of consensus",
			}),
			Phase: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_phase",
				Help: "Current phase of consensus",
			}),
			ElectionTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_election_time",
				Help: "Execution time of the ELECTION bft phase",
			}),
			ElectionVoteTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_election_vote_time",
				Help: "Execution time of the ELECTION_VOTE bft phase",
			}),
			ProposeTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_propose_time",
				Help: "Execution time of the PROPOSE bft phase",
			}),
			ProposeVoteTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_propose_vote_time",
				Help: "Execution time of the PROPOSE_VOTE bft phase",
			}),
			PrecommitTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_precommit_time",
				Help: "Execution time of the PRECOMMIT bft phase",
			}),
			PrecommitVoteTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_precommit_vote_time",
				Help: "Execution time of the PRECOMMIT_VOTE bft phase",
			}),
			CommitTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_commit_time",
				Help: "Execution time of the COMMIT bft phase",
			}),
			CommitProcessTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_bft_commit_process_time",
				Help: "Execution time of the COMMIT_PROCESS bft phase",
			}),
			RootHeight: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_root_height",
				Help: "Current height of the `root_chain` the quorum is operating on",
			}),
			RootChainId: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_root_chain_id",
				Help: "The chain ID of the root chain this node is operating on",
			}),
		},
		// FSM
		FSMMetrics: FSMMetrics{
			ValidatorStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_status",
				Help: "Validator status (0: Unstaked, 1: Staked, 2: Unstaking, 3: Paused)",
			}, []string{"address"}),
			ValidatorType: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_type",
				Help: "Validator type (0: Delegate, 1: Validator)",
			}, []string{"address"}),
			ValidatorCompounding: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_compounding",
				Help: "Validator compounding status (1: true, 0: false)",
			}, []string{"address"}),
			ValidatorStakeAmount: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_stake_amount",
				Help: "Validator stake in uCNPY",
			}, []string{"address"}),
			ValidatorBlockProducer: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_block_producer",
				Help: "Validator was block producer (1: true, 0: false)",
			}, []string{"address"}),
			ValidatorNonSigner: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_non_signer",
				Help: "Validator was block non signer (1: true, 0: false)",
			}, []string{"address"}),
			ValidatorNonSignerCount: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_non_signer_count",
				Help: "Count of non signers within the non-sign-window",
			}, []string{"type"}),
			ValidatorDoubleSigner: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_double_signer",
				Help: "Validator was double signer (1: true, 0: false)",
			}, []string{"address"}),
			ValidatorDoubleSignerCount: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_double_signer_count",
				Help: "Count of double signers for the last block",
			}, []string{"type"}),
			ValidatorCount: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_validator_count",
				Help: "Count of validators",
			}, []string{"type"}),
		},
		// STORE
		StoreMetrics: StoreMetrics{
			DBPartitionTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_store_partition_time",
				Help: "Execution time of the database partition",
			}),
			DBFlushPartitionTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_store_flush_partition_time",
				Help: "Execution time of the database partition flush",
			}),
			DBPartitionEntries: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_store_partition_entries",
				Help: "Number of entries in the partition batch",
			}),
			DBPartitionSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_store_partition_size",
				Help: "Number of bytes in the partition batch",
			}),
			DBCommitTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_store_commit_time",
				Help: "Execution time of the flushing of the commit batch",
			}),
			DBCommitEntries: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_store_commit_entries",
				Help: "Number of entries in the commit batch",
			}),
			DBCommitSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_store_commit_size",
				Help: "Number of bytes in the commit batch",
			}),
		},
		// MEMPOOL
		MempoolMetrics: MempoolMetrics{
			MempoolSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_mempool_size",
				Help: "Count of bytes in the transaction memory pool",
			}),
			MempoolTxCount: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_mempool_tx_count",
				Help: "Count of transactions in the transaction memory pool",
			}),
		},
		// ORACLE
		OracleMetrics: OracleMetrics{
			OracleBlockProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_oracle_block_processing_time",
				Help: "Time to process blocks in the oracle in seconds",
			}),
			OrderValidationTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_oracle_order_validation_time",
				Help: "Time to validate orders in the oracle in seconds",
			}),
			OrdersWitnessed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_witnessed_total",
				Help: "Total number of orders witnessed from Ethereum",
			}),
			OrdersValidated: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_validated_total",
				Help: "Total number of orders validated successfully",
			}),
			OrdersSubmitted: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_submitted_total",
				Help: "Total number of orders submitted for consensus",
			}),
			OrdersRejected: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_rejected_total",
				Help: "Total number of orders rejected during validation",
			}),
			TotalOrdersStored: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_total_orders_stored",
				Help: "Total number of orders currently stored in order store",
			}),
			LockOrdersStored: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_lock_orders_stored",
				Help: "Total number of lock orders currently stored",
			}),
			CloseOrdersStored: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_close_orders_stored",
				Help: "Total number of close orders currently stored",
			}),
			SafeHeight: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_safe_height",
				Help: "Current safe block height in the oracle",
			}),
			SourceChainHeight: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_source_chain_height",
				Help: "Current source chain height in the oracle",
			}),
			LockOrderSubmissionsSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_lock_order_submissions_size",
				Help: "Size of the lock order submissions map",
			}),
			CloseOrderSubmissionsSize: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_close_order_submissions_size",
				Help: "Size of the close order submissions map",
			}),
			ChainReorgs: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_chain_reorgs_total",
				Help: "Total number of chain reorganizations detected",
			}),
			OrdersPruned: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_pruned_total",
				Help: "Total number of orders pruned during cleanup",
			}),
			BlockProcessingErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_block_processing_errors_total",
				Help: "Total number of block processing errors",
			}),
			OrderBookUpdateTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_oracle_order_book_update_time",
				Help: "Time to update order book in the oracle in seconds",
			}),
			RootChainSyncTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_oracle_root_chain_sync_time",
				Help: "Time to sync with root chain in the oracle in seconds",
			}),
			// Block height metrics
			LastProcessedHeight: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_last_processed_height",
				Help: "Last source chain block height processed by the oracle",
			}),
			ConfirmationLag: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_confirmation_lag",
				Help: "Gap between source chain height and safe height (blocks awaiting confirmation)",
			}),
			OrdersAwaitingConfirmation: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_oracle_orders_awaiting_confirmation",
				Help: "Number of orders witnessed but not yet at safe height",
			}),
			ReorgRollbackDepth: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_oracle_reorg_rollback_depth",
				Help:    "Number of blocks rolled back during chain reorganizations",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			}),
			// Order lifecycle metrics
			OrdersNotInOrderbook: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_not_in_orderbook_total",
				Help: "Total orders witnessed but not found in order book",
			}),
			OrdersDuplicate: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_duplicate_total",
				Help: "Total duplicate orders encountered (already in store)",
			}),
			OrdersArchived: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_archived_total",
				Help: "Total orders successfully archived",
			}),
			LockOrdersCommitted: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_lock_orders_committed_total",
				Help: "Total lock orders committed via certificate",
			}),
			CloseOrdersCommitted: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_close_orders_committed_total",
				Help: "Total close orders committed via certificate",
			}),
			// Validation failure metrics
			ValidationFailures: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_oracle_validation_failures_total",
				Help: "Total validation failures by reason",
			}, []string{"reason"}),
			// Submission tracking metrics
			OrdersHeldAwaitingSafe: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_held_awaiting_safe_total",
				Help: "Total orders not submitted due to safe height requirement",
			}),
			OrdersHeldProposeDelay: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_held_propose_delay_total",
				Help: "Total orders held by ProposeDelayBlocks configuration",
			}),
			OrdersHeldResubmitDelay: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_orders_held_resubmit_delay_total",
				Help: "Total orders held by resubmit cooldown",
			}),
			LockOrderResubmissions: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_lock_order_resubmissions_total",
				Help: "Total lock orders resubmitted",
			}),
			CloseOrderResubmissions: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_close_order_resubmissions_total",
				Help: "Total close orders resubmitted",
			}),
			// Store operation metrics
			StoreWriteErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_store_write_errors_total",
				Help: "Total order store write failures",
			}),
			StoreReadErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_store_read_errors_total",
				Help: "Total order store read failures",
			}),
			StoreRemoveErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_oracle_store_remove_errors_total",
				Help: "Total order store remove failures",
			}),
		},
		// ETH
		EthBlockProviderMetrics: EthBlockProviderMetrics{
			BlockFetchTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_eth_block_fetch_time",
				Help: "Time to fetch Ethereum blocks in seconds",
			}),
			TransactionProcessTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_eth_transaction_process_time",
				Help: "Time to process Ethereum transactions in seconds",
			}),
			ReceiptFetchTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_eth_receipt_fetch_time",
				Help: "Time to fetch Ethereum transaction receipts in seconds",
			}),
			TokenCacheHits: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_token_cache_hits_total",
				Help: "Total number of ERC20 token cache hits",
			}),
			TokenCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_token_cache_misses_total",
				Help: "Total number of ERC20 token cache misses",
			}),
			ConnectionErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_connection_errors_total",
				Help: "Total number of Ethereum connection errors",
			}),
			BlocksProcessed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_blocks_processed_total",
				Help: "Total number of Ethereum blocks processed",
			}),
			TransactionsProcessed: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_transactions_processed_total",
				Help: "Total number of Ethereum transactions processed",
			}),
			TransactionRetries: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_transaction_retries_total",
				Help: "Total number of Ethereum transaction processing retries",
			}),
			// Connection & Sync Status Metrics
			RPCConnectionAttempts: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_rpc_connection_attempts_total",
				Help: "Total RPC connection attempts",
			}),
			RPCConnectionErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_rpc_connection_errors_total",
				Help: "RPC connection errors by error type",
			}, []string{"error_type"}),
			WSConnectionAttempts: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_ws_connection_attempts_total",
				Help: "Total WebSocket connection attempts",
			}),
			WSSubscriptionErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_ws_subscription_errors_total",
				Help: "WebSocket subscription failures",
			}),
			ConnectionState: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_eth_connection_state",
				Help: "Current connection state (0=disconnected, 1=connecting, 2=rpc_connected, 3=fully_connected)",
			}),
			SyncStatus: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_eth_sync_status",
				Help: "Sync status (0=unsynced, 1=syncing, 2=synced)",
			}),
			BlockHeightLag: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_eth_block_height_lag",
				Help: "Number of blocks behind chain head",
			}),
			// Block Processing Metrics
			BlockFetchErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_block_fetch_errors_total",
				Help: "Block fetch errors by error type",
			}, []string{"error_type"}),
			BlockProcessingTimeouts: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_block_processing_timeouts_total",
				Help: "Blocks that timed out during processing",
			}),
			ProcessBlocksBatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
				Name:    "canopy_eth_process_blocks_batch_size",
				Help:    "Number of blocks processed per batch",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			}),
			ReorgDetected: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_reorg_detected_total",
				Help: "Chain reorganizations detected",
			}),
			// Transaction Processing Metrics
			TransactionsTotal: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_transactions_total",
				Help: "Total transactions encountered in blocks",
			}),
			TransactionParseErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_transaction_parse_errors_total",
				Help: "Transaction parsing errors by error type",
			}, []string{"error_type"}),
			TransactionRetryByAttempt: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_transaction_retry_by_attempt_total",
				Help: "Transaction retry attempts by attempt number",
			}, []string{"attempt"}),
			TransactionExhaustedRetries: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_transaction_exhausted_retries_total",
				Help: "Transactions that exhausted all retry attempts",
			}),
			TransactionSuccessStatus: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_transaction_success_status_total",
				Help: "Transaction success/failed/unknown breakdown",
			}, []string{"status"}),
			ReceiptFetchErrors: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_receipt_fetch_errors_total",
				Help: "Receipt fetch failures",
			}),
			// Order Detection Metrics
			ERC20TransferDetected: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_erc20_transfer_detected_total",
				Help: "ERC20 transfers detected",
			}),
			LockOrderDetected: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_lock_order_detected_total",
				Help: "Lock orders successfully parsed",
			}),
			CloseOrderDetected: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_close_order_detected_total",
				Help: "Close orders successfully parsed",
			}),
			OrderValidationErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_order_validation_errors_total",
				Help: "Order validation errors by order type and error type",
			}, []string{"order_type", "error_type"}),
			// Token Cache Error Metrics
			TokenInfoFetchErrors: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_eth_token_info_fetch_errors_total",
				Help: "Token info fetch errors by field",
			}, []string{"field"}),
			TokenContractCallTimeouts: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_eth_token_contract_call_timeouts_total",
				Help: "Token contract call timeouts",
			}),
		},
	}
}

// Start() starts the telemetry server
func (m *Metrics) Start() {
	// exit if empty
	if m == nil {
		return
	}
	// set the chain ID and software version metrics (one-time on startup)
	m.ChainId.Set(m.chainID)
	m.SoftwareVersion.WithLabelValues(m.softwareVersion).Set(1)
	// if the metrics server is enabled
	if m.config.MetricsEnabled {
		go func() {
			m.log.Infof("Starting metrics server on %s", m.config.PrometheusAddress)
			// run the server
			if err := m.server.ListenAndServe(); err != nil {
				if err != http.ErrServerClosed {
					m.log.Errorf("Metrics server failed with err: %s", err.Error())
				}
			}
		}()
	}
}

// Stop() gracefully stops the telemetry server
func (m *Metrics) Stop() {
	// exit if empty
	if m == nil {
		return
	}
	// if the metrics server isn't enabled
	if m.config.MetricsEnabled {
		// shutdown the server
		if err := m.server.Shutdown(context.Background()); err != nil {
			m.log.Error(err.Error())
		}
	}
}

// UpdateNodeMetrics updates the node syncing status
func (m *Metrics) UpdateNodeMetrics(isSyncing bool) {
	// exit if empty
	if m == nil {
		return
	}
	// set node is active
	m.NodeStatus.Set(1)
	// update syncing status
	if isSyncing {
		m.SyncingStatus.Set(0)
	} else {
		m.SyncingStatus.Set(1)
	}
}

// UpdatePeerMetrics() is a setter for the peer metrics
func (m *Metrics) UpdatePeerMetrics(total, inbound, outbound int) {
	// exit if empty
	if m == nil {
		return
	}
	// set total number of peers
	m.TotalPeers.Set(float64(total))
	// set total number of peers that dialed this node
	m.InboundPeers.Set(float64(inbound))
	// set total number of peers that this node dialed
	m.OutboundPeers.Set(float64(outbound))
}

// UpdateBFTMetrics() is a setter for the BFT metrics
func (m *Metrics) UpdateBFTMetrics(height, rootHeight, rootChainId, round uint64, phase Phase, phaseStartTime time.Time) {
	// exit if empty
	if m == nil {
		return
	}
	// set the height of this chain
	m.Height.Set(float64(height))
	// set the height of the root chain
	m.RootHeight.Set(float64(rootHeight))
	// set the chain id of the root chain
	m.RootChainId.Set(float64(rootChainId))
	// set the round
	m.Round.Set(float64(round))
	// set the phase
	m.Phase.Set(float64(phase))
	// set the phase duration
	switch phase {
	case Phase_ELECTION:
		m.ElectionTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_ELECTION_VOTE:
		m.ElectionVoteTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_PROPOSE:
		m.ProposeTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_PROPOSE_VOTE:
		m.ProposeVoteTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_PRECOMMIT:
		m.PrecommitTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_PRECOMMIT_VOTE:
		m.PrecommitVoteTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_COMMIT:
		m.CommitTime.Observe(time.Since(phaseStartTime).Seconds())
	case Phase_COMMIT_PROCESS:
		m.CommitProcessTime.Observe(time.Since(phaseStartTime).Seconds())
	}
}

// UpdateValidator() updates the validator metrics for prometheus
func (m *Metrics) UpdateValidator(address string, stakeAmount uint64, unstaking, paused, delegate, compounding, isProducer bool,
	nonSigners map[string]uint64, doubleSigners []crypto.AddressI) {
	// exit if empty
	if m == nil {
		return
	}
	// update the auto-compounding metric
	if compounding {
		m.ValidatorCompounding.WithLabelValues(address).Set(float64(1))
	} else {
		m.ValidatorCompounding.WithLabelValues(address).Set(float64(0))
	}
	// update the validator stake amount
	m.ValidatorStakeAmount.WithLabelValues(address).Set(float64(stakeAmount))
	// update the delegate metric
	if delegate {
		m.ValidatorType.WithLabelValues(address).Set(float64(0))
	} else {
		m.ValidatorType.WithLabelValues(address).Set(float64(1))
	}
	// update block producer
	if isProducer {
		m.ValidatorBlockProducer.WithLabelValues(address).Set(float64(1))
	} else {
		m.ValidatorBlockProducer.WithLabelValues(address).Set(float64(0))
	}
	var isNonSigner bool
	// update non signer
	for nonSignerAddress := range nonSigners {
		if address == nonSignerAddress {
			isNonSigner = true
		}
	}
	m.ValidatorNonSignerCount.WithLabelValues("any").Set(float64(len(nonSigners)))
	if isNonSigner {
		m.ValidatorNonSigner.WithLabelValues(address).Set(float64(1))
	} else {
		m.ValidatorNonSigner.WithLabelValues(address).Set(float64(0))
	}
	var isDoubleSigner bool
	// update double signer
	for _, doubleSigner := range doubleSigners {
		if doubleSigner.String() == address {
			isDoubleSigner = true
		}
	}
	m.ValidatorDoubleSignerCount.WithLabelValues("any").Set(float64(len(doubleSigners)))
	if isDoubleSigner {
		m.ValidatorDoubleSigner.WithLabelValues(address).Set(float64(1))
	} else {
		m.ValidatorDoubleSigner.WithLabelValues(address).Set(float64(0))
	}
	// update the status metric
	switch {
	case unstaking:
		// if the val is unstaking
		m.ValidatorStatus.WithLabelValues(address).Set(2)
	case paused:
		// if the val is paused
		m.ValidatorStatus.WithLabelValues(address).Set(3)
	case stakeAmount == 0:
		// if the val is unstaked
		m.ValidatorStatus.WithLabelValues(address).Set(0)
	default:
		// if the val is active
		m.ValidatorStatus.WithLabelValues(address).Set(1)
	}
}

// UpdateAccount() updates the account balance of the node
func (m *Metrics) UpdateAccount(address string, balance uint64) {
	// exit if empty
	if m == nil {
		return
	}
	// update the account balance
	m.AccountBalance.WithLabelValues(address).Set(float64(balance))
}

// UpdateStoreMetrics() updates the store telemetry
func (m *Metrics) UpdateStoreMetrics(size, entries int64, startTime time.Time, startFlushTime time.Time) {
	// exit if empty
	if m == nil {
		return
	}
	// update the partition metrics
	if !startTime.IsZero() {
		// updates the size in bytes
		m.DBPartitionSize.Set(float64(size))
		// updates the number of entries
		m.DBPartitionEntries.Set(float64(entries))
		// update the processing time in seconds
		m.DBFlushPartitionTime.Observe(time.Since(startFlushTime).Seconds())
		// update the processing time in seconds
		m.DBPartitionTime.Observe(time.Since(startTime).Seconds())
	} else {
		// updates the size in bytes
		m.DBCommitSize.Set(float64(size))
		// updates the number of entries
		m.DBCommitEntries.Set(float64(entries))
		// update the processing time in seconds
		m.DBCommitTime.Observe(time.Since(startFlushTime).Seconds())
	}
}

// UpdateBlockMetrics() updates the metrics about the last block
func (m *Metrics) UpdateBlockMetrics(proposerAddress []byte, blockSize, txCount, vdfIterations uint64, duration time.Duration) {
	// exit if empty
	if m == nil {
		return
	}
	// if this node was the proposer
	if bytes.Equal(proposerAddress, m.nodeAddress) {
		// update the proposal count
		m.ProposerCount.Inc()
	}
	// update the number of transactions
	m.BlockNumTxs.Set(float64(txCount))
	// update the block processing time in seconds
	m.BlockProcessingTime.Observe(duration.Seconds())
	// update block size
	m.BlockSize.Set(float64(blockSize))
	// update the block vdf iterations
	m.BlockVDFIterations.Set(float64(vdfIterations))
}

// UpdateMempoolMetrics() updates mempool telemetry
func (m *Metrics) UpdateMempoolMetrics(txCount, size int) {
	// exit if empty
	if m == nil {
		return
	}
	// update the transaction count metric
	m.MempoolTxCount.Set(float64(txCount))
	// update the mempool size metric
	m.MempoolSize.Set(float64(size))
}

// UpdateNonSignerPercent() updates the percent of the non-signers for a block
func (m *Metrics) UpdateNonSignerPercent(as *AggregateSignature, set ValidatorSet) {
	// exit if empty
	if m == nil {
		return
	}
	_, nonSignerPercent, err := as.GetNonSigners(set.ValidatorSet)
	if err != nil {
		m.log.Error(err.Error())
		return
	}
	// update the metric
	m.NonSignerPercent.Set(float64(nonSignerPercent))
}

// UpdateLargestTxSize() updates the largest size tx included in a block
func (m *Metrics) UpdateLargestTxSize(size uint64) {
	// exit if empty
	if m == nil {
		return
	}
	// update the metric
	m.LargestTxSize.Set(float64(size))
}

// UpdateGetRootChainInfo() updates the time it took to execute a fsm.GetRootChainInfo() call
func (m *Metrics) UpdateGetRootChainInfo(startTime time.Time) {
	// exit if empty
	if m == nil {
		return
	}
	// update the metric
	m.GetRootChainInfo.Observe(time.Since(startTime).Seconds())
}

// SetStartupBlock() sets the block height when the node first completed syncing after startup
func (m *Metrics) SetStartupBlock(blockHeight uint64) {
	// exit if empty
	if m == nil {
		return
	}
	// only set the startup block metric once per node run
	if !m.startupBlockSet {
		m.StartupBlock.Set(float64(blockHeight))
		m.startupBlockSet = true
	}
}

func (m *Metrics) UpdateValidatorCount(count int) {
	// exit if empty
	if m == nil {
		return
	}
	// update the metric
	m.ValidatorCount.WithLabelValues("total").Set(float64(count))
}

// UpdateOracleBlockMetrics() updates oracle block processing metrics
func (m *Metrics) UpdateOracleBlockMetrics(processingTime time.Duration) {
	// exit if empty
	if m == nil {
		return
	}
	// update the block processing time
	m.OracleBlockProcessingTime.Observe(processingTime.Seconds())
}

// UpdateOracleOrderMetrics() updates oracle order processing metrics
func (m *Metrics) UpdateOracleOrderMetrics(witnessed, validated, submitted, rejected int, validationTime time.Duration) {
	// exit if empty
	if m == nil {
		return
	}
	// update counters
	m.OrdersWitnessed.Add(float64(witnessed))
	m.OrdersValidated.Add(float64(validated))
	m.OrdersSubmitted.Add(float64(submitted))
	m.OrdersRejected.Add(float64(rejected))
	// update timing metrics
	if validationTime > 0 {
		m.OrderValidationTime.Observe(validationTime.Seconds())
	}
}

// UpdateOracleStateMetrics() updates oracle state management metrics
func (m *Metrics) UpdateOracleStateMetrics(safeHeight, sourceHeight uint64, lockOrderSubmissionsSize, closeOrderSubmissionsSize int) {
	// exit if empty
	if m == nil {
		return
	}
	// update state metrics
	m.SafeHeight.Set(float64(safeHeight))
	m.SourceChainHeight.Set(float64(sourceHeight))
	m.LockOrderSubmissionsSize.Set(float64(lockOrderSubmissionsSize))
	m.CloseOrderSubmissionsSize.Set(float64(closeOrderSubmissionsSize))
}

// UpdateOracleStoreMetrics() updates oracle order store metrics
func (m *Metrics) UpdateOracleStoreMetrics(lockOrders, closeOrders int) {
	// exit if empty
	if m == nil {
		return
	}
	// update store count metrics
	m.TotalOrdersStored.Set(float64(lockOrders + closeOrders))
	m.LockOrdersStored.Set(float64(lockOrders))
	m.CloseOrdersStored.Set(float64(closeOrders))
}

// UpdateOracleErrorMetrics() updates oracle error and reorg metrics
func (m *Metrics) UpdateOracleErrorMetrics(reorgs, pruned, blockErrors int) {
	// exit if empty
	if m == nil {
		return
	}
	// update error counters
	m.ChainReorgs.Add(float64(reorgs))
	m.OrdersPruned.Add(float64(pruned))
	m.BlockProcessingErrors.Add(float64(blockErrors))
}

// UpdateEthBlockProviderMetrics() updates Ethereum block provider metrics
func (m *Metrics) UpdateEthBlockProviderMetrics(blockFetchTime, transactionProcessTime, receiptFetchTime time.Duration,
	cacheHits, cacheMisses, connectionErrors, blocksProcessed, transactionsProcessed, retries int) {
	// exit if empty
	if m == nil {
		return
	}
	// update timing metrics
	if blockFetchTime > 0 {
		m.BlockFetchTime.Observe(blockFetchTime.Seconds())
	}
	if transactionProcessTime > 0 {
		m.TransactionProcessTime.Observe(transactionProcessTime.Seconds())
	}
	if receiptFetchTime > 0 {
		m.ReceiptFetchTime.Observe(receiptFetchTime.Seconds())
	}
	// update counters
	m.TokenCacheHits.Add(float64(cacheHits))
	m.TokenCacheMisses.Add(float64(cacheMisses))
	m.ConnectionErrors.Add(float64(connectionErrors))
	m.BlocksProcessed.Add(float64(blocksProcessed))
	m.TransactionsProcessed.Add(float64(transactionsProcessed))
	m.TransactionRetries.Add(float64(retries))
}

// UpdateOracleHeightMetrics() updates oracle block height tracking metrics
func (m *Metrics) UpdateOracleHeightMetrics(lastHeight, safeHeight, sourceHeight uint64, awaitingConfirmation int) {
	if m == nil {
		return
	}
	m.LastProcessedHeight.Set(float64(lastHeight))
	m.ConfirmationLag.Set(float64(sourceHeight - safeHeight))
	m.OrdersAwaitingConfirmation.Set(float64(awaitingConfirmation))
}

// RecordOracleReorgDepth() records the depth of a chain reorganization rollback
func (m *Metrics) RecordOracleReorgDepth(depth uint64) {
	if m == nil {
		return
	}
	m.ReorgRollbackDepth.Observe(float64(depth))
}

// IncrementValidationFailure() increments the validation failure counter for a specific reason
func (m *Metrics) IncrementValidationFailure(reason string) {
	if m == nil {
		return
	}
	m.ValidationFailures.WithLabelValues(reason).Inc()
}

// UpdateOracleLifecycleMetrics() updates order lifecycle metrics
func (m *Metrics) UpdateOracleLifecycleMetrics(notInOrderbook, duplicate, archived, lockCommitted, closeCommitted int) {
	if m == nil {
		return
	}
	m.OrdersNotInOrderbook.Add(float64(notInOrderbook))
	m.OrdersDuplicate.Add(float64(duplicate))
	m.OrdersArchived.Add(float64(archived))
	m.LockOrdersCommitted.Add(float64(lockCommitted))
	m.CloseOrdersCommitted.Add(float64(closeCommitted))
}

// UpdateOracleSubmissionMetrics() updates submission tracking metrics
func (m *Metrics) UpdateOracleSubmissionMetrics(heldSafe, heldPropose, heldResubmit, lockResub, closeResub int) {
	if m == nil {
		return
	}
	m.OrdersHeldAwaitingSafe.Add(float64(heldSafe))
	m.OrdersHeldProposeDelay.Add(float64(heldPropose))
	m.OrdersHeldResubmitDelay.Add(float64(heldResubmit))
	m.LockOrderResubmissions.Add(float64(lockResub))
	m.CloseOrderResubmissions.Add(float64(closeResub))
}

// UpdateOracleStoreErrorMetrics() updates store operation error metrics
func (m *Metrics) UpdateOracleStoreErrorMetrics(writeErrors, readErrors, removeErrors int) {
	if m == nil {
		return
	}
	m.StoreWriteErrors.Add(float64(writeErrors))
	m.StoreReadErrors.Add(float64(readErrors))
	m.StoreRemoveErrors.Add(float64(removeErrors))
}

// ========== Eth Block Provider Metrics Helper Functions ==========

// SetEthConnectionState sets the current connection state
// States: 0=disconnected, 1=connecting, 2=rpc_connected, 3=fully_connected
func (m *Metrics) SetEthConnectionState(state int) {
	if m == nil {
		return
	}
	m.ConnectionState.Set(float64(state))
}

// SetEthSyncStatus sets the current sync status
// States: 0=unsynced, 1=syncing, 2=synced
func (m *Metrics) SetEthSyncStatus(status int) {
	if m == nil {
		return
	}
	m.SyncStatus.Set(float64(status))
}

// SetEthBlockHeightLag sets the number of blocks behind chain head
func (m *Metrics) SetEthBlockHeightLag(lag uint64) {
	if m == nil {
		return
	}
	m.BlockHeightLag.Set(float64(lag))
}

// IncrementEthRPCConnectionAttempt increments the RPC connection attempt counter
func (m *Metrics) IncrementEthRPCConnectionAttempt() {
	if m == nil {
		return
	}
	m.RPCConnectionAttempts.Inc()
}

// IncrementEthRPCConnectionError increments the RPC connection error counter for a specific error type
func (m *Metrics) IncrementEthRPCConnectionError(errorType string) {
	if m == nil {
		return
	}
	m.RPCConnectionErrors.WithLabelValues(errorType).Inc()
}

// IncrementEthWSConnectionAttempt increments the WebSocket connection attempt counter
func (m *Metrics) IncrementEthWSConnectionAttempt() {
	if m == nil {
		return
	}
	m.WSConnectionAttempts.Inc()
}

// IncrementEthWSSubscriptionError increments the WebSocket subscription error counter
func (m *Metrics) IncrementEthWSSubscriptionError() {
	if m == nil {
		return
	}
	m.WSSubscriptionErrors.Inc()
}

// IncrementEthBlockFetchError increments the block fetch error counter for a specific error type
func (m *Metrics) IncrementEthBlockFetchError(errorType string) {
	if m == nil {
		return
	}
	m.BlockFetchErrors.WithLabelValues(errorType).Inc()
}

// IncrementEthBlockProcessingTimeout increments the block processing timeout counter
func (m *Metrics) IncrementEthBlockProcessingTimeout() {
	if m == nil {
		return
	}
	m.BlockProcessingTimeouts.Inc()
}

// RecordEthProcessBlocksBatchSize records the number of blocks processed in a batch
func (m *Metrics) RecordEthProcessBlocksBatchSize(batchSize int) {
	if m == nil {
		return
	}
	m.ProcessBlocksBatchSize.Observe(float64(batchSize))
}

// IncrementEthReorgDetected increments the chain reorganization detected counter
func (m *Metrics) IncrementEthReorgDetected() {
	if m == nil {
		return
	}
	m.ReorgDetected.Inc()
}

// IncrementEthTransactionsTotal increments the total transactions counter
func (m *Metrics) IncrementEthTransactionsTotal(count int) {
	if m == nil {
		return
	}
	m.TransactionsTotal.Add(float64(count))
}

// IncrementEthTransactionParseError increments the transaction parse error counter for a specific error type
func (m *Metrics) IncrementEthTransactionParseError(errorType string) {
	if m == nil {
		return
	}
	m.TransactionParseErrors.WithLabelValues(errorType).Inc()
}

// IncrementEthTransactionRetryByAttempt increments the transaction retry counter for a specific attempt number
func (m *Metrics) IncrementEthTransactionRetryByAttempt(attempt int) {
	if m == nil {
		return
	}
	m.TransactionRetryByAttempt.WithLabelValues(fmt.Sprintf("%d", attempt)).Inc()
}

// IncrementEthTransactionExhaustedRetries increments the exhausted retries counter
func (m *Metrics) IncrementEthTransactionExhaustedRetries() {
	if m == nil {
		return
	}
	m.TransactionExhaustedRetries.Inc()
}

// IncrementEthTransactionSuccessStatus increments the transaction success status counter
// Status values: "success", "failed", "unknown"
func (m *Metrics) IncrementEthTransactionSuccessStatus(status string) {
	if m == nil {
		return
	}
	m.TransactionSuccessStatus.WithLabelValues(status).Inc()
}

// IncrementEthReceiptFetchError increments the receipt fetch error counter
func (m *Metrics) IncrementEthReceiptFetchError() {
	if m == nil {
		return
	}
	m.ReceiptFetchErrors.Inc()
}

// IncrementEthERC20TransferDetected increments the ERC20 transfer detected counter
func (m *Metrics) IncrementEthERC20TransferDetected() {
	if m == nil {
		return
	}
	m.ERC20TransferDetected.Inc()
}

// IncrementEthLockOrderDetected increments the lock order detected counter
func (m *Metrics) IncrementEthLockOrderDetected() {
	if m == nil {
		return
	}
	m.LockOrderDetected.Inc()
}

// IncrementEthCloseOrderDetected increments the close order detected counter
func (m *Metrics) IncrementEthCloseOrderDetected() {
	if m == nil {
		return
	}
	m.CloseOrderDetected.Inc()
}

// IncrementEthOrderValidationError increments the order validation error counter
func (m *Metrics) IncrementEthOrderValidationError(orderType, errorType string) {
	if m == nil {
		return
	}
	m.OrderValidationErrors.WithLabelValues(orderType, errorType).Inc()
}

// IncrementEthTokenInfoFetchError increments the token info fetch error counter for a specific field
func (m *Metrics) IncrementEthTokenInfoFetchError(field string) {
	if m == nil {
		return
	}
	m.TokenInfoFetchErrors.WithLabelValues(field).Inc()
}

// IncrementEthTokenContractCallTimeout increments the token contract call timeout counter
func (m *Metrics) IncrementEthTokenContractCallTimeout() {
	if m == nil {
		return
	}
	m.TokenContractCallTimeouts.Inc()
}
