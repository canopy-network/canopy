package lib

import (
	"bytes"
	"context"
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

	NodeMetrics    // general telemetry about the node
	BlockMetrics   // block telemetry
	PeerMetrics    // peer telemetry
	BFTMetrics     // bft telemetry
	FSMMetrics     // fsm telemetry
	StoreMetrics   // persistence telemetry
	MempoolMetrics // tx memory pool telemetry
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
