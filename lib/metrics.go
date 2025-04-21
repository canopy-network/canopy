package lib

import (
	"bytes"
	"context"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"
)

/* This file implements dev-ops telemetry for the node in the form of prometheus metrics */

const metricsPattern = "/metrics"

// Metrics represents a server that exposes Prometheus metrics
type Metrics struct {
	server      *http.Server  // the http prometheus server
	config      MetricsConfig // the configuration
	nodeAddress []byte        // the node's address
	log         LoggerI       // the logger

	NodeMetrics       // general telemetry about the node
	PeerMetrics       // peer telemetry
	BFTMetrics        // bft telemetry
	GovernanceMetrics // governance telemetry
	FSMMetrics        // fsm telemetry
	IndexerMetrics    // indexer telemetry
}

// NodeMetrics represents general telemetry for the node's health
type NodeMetrics struct {
	NodeStatus          prometheus.Gauge     // is the node alive?
	BlockProcessingTime prometheus.Histogram // how long does it take for this node to process a block?
}

// PeerMetrics represents the telemetry for the P2P module
type PeerMetrics struct {
	TotalPeers    prometheus.Gauge // number of peers
	InboundPeers  prometheus.Gauge // number of peers that dialed this node
	OutboundPeers prometheus.Gauge // number of peers that this node dialed
}

// BFTMetrics represents the telemetry for the BFT module
type BFTMetrics struct {
	Height        prometheus.Gauge   // what's the height of this chain?
	RootHeight    prometheus.Gauge   // what's the height of the root-chain?
	SyncingStatus prometheus.Gauge   // is the node syncing?
	ProposerCount prometheus.Counter // how many times did this node propose the block?
}

type GovernanceMetrics struct {
	ActiveProposals prometheus.Gauge // the number of governance proposals active in the moment
}

// FSMMetrics represents the telemetry of the FSM module for the node's address
type FSMMetrics struct {
	ValidatorStatus      *prometheus.GaugeVec // what's the status of this validator?
	ValidatorType        *prometheus.GaugeVec // what's the type of this validator?
	ValidatorCompounding *prometheus.GaugeVec // is this validator compounding?
	ValidatorStakeAmount *prometheus.GaugeVec // what's the stake amount of this validator
	AccountBalance       *prometheus.GaugeVec // what's the balance of this node's account?
}

// IndexerMetrics represents the telemetry of the indexer for the node's address
type IndexerMetrics struct {
	TransactionCount prometheus.Counter     // how many transactions has the node processed?
	TxsReceived      *prometheus.CounterVec // how many transactions has the node's account received?
	TxsSent          *prometheus.CounterVec // how many transactions has the node's account sent?
}

// NewMetricsServer() creates a new telemetry server
func NewMetricsServer(nodeAddress crypto.AddressI, config MetricsConfig) *Metrics {
	mux := http.NewServeMux()
	mux.Handle(metricsPattern, promhttp.Handler())
	return &Metrics{
		server:      &http.Server{Addr: config.PrometheusAddress, Handler: mux},
		config:      config,
		nodeAddress: nodeAddress.Bytes(),
		log:         NewDefaultLogger(),
		NodeMetrics: NodeMetrics{
			NodeStatus: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_node_status",
				Help: "The node is alive and processing blocks",
			}),
			BlockProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "canopy_block_processing_time",
				Help: "Time to process a block in seconds",
			}),
		},
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
		BFTMetrics: BFTMetrics{
			Height: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_height",
				Help: "Current BFT height",
			}),
			RootHeight: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_bft_root_height",
				Help: "Current BFT root height",
			}),
			SyncingStatus: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_node_syncing_status",
				Help: "Node syncing status (1 for syncing, 0 for synced)",
			}),
			ProposerCount: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_blocks_validated_by_proposer",
				Help: "Total number of blocks validated when node is proposer",
			}),
		},
		GovernanceMetrics: GovernanceMetrics{
			ActiveProposals: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "canopy_governance_active_proposals",
				Help: "Number of active governance proposals",
			}),
		},
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
				Name: "canopy_stake_amount",
				Help: "Validator stake in uCNPY",
			}, []string{"address"}),
			AccountBalance: promauto.NewGaugeVec(prometheus.GaugeOpts{
				Name: "canopy_account_balance",
				Help: "Account balance in uCNPY",
			}, []string{"address"}),
		},
		IndexerMetrics: IndexerMetrics{
			TransactionCount: promauto.NewCounter(prometheus.CounterOpts{
				Name: "canopy_transaction_count",
				Help: "Total number of transactions processed",
			}),
			TxsReceived: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_transaction_received",
				Help: "Number of received transactions",
			}, []string{"address"}),
			TxsSent: promauto.NewCounterVec(prometheus.CounterOpts{
				Name: "canopy_transaction_sent",
				Help: "Number of sent transactions",
			}, []string{"address"}),
		},
	}
}

// Start() starts the telemetry server
func (m *Metrics) Start() {
	// exit if empty
	if m == nil {
		return
	}
	// if the metrics server is enabled
	if m.config.Enabled {
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
	if m.config.Enabled {
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
		m.SyncingStatus.Set(1)
	} else {
		m.SyncingStatus.Set(0)
	}
}

// UpdateGovernanceMetrics() is a setter for the peer metrics
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

// UpdateGovernanceMetrics() is a setter for the BFT metrics
func (m *Metrics) UpdateBFTMetrics(height, rootHeight uint64) {
	// exit if empty
	if m == nil {
		return
	}
	// set the height of this chain
	m.Height.Set(float64(height))
	// set the height of the root chain
	m.RootHeight.Set(float64(rootHeight))
}

// UpdateGovernanceMetrics() is a setter for the governance metrics
func (m *Metrics) UpdateGovernanceMetrics(activeProposals int) {
	// exit if empty
	if m == nil {
		return
	}
	// update the number of active proposals
	m.ActiveProposals.Set(float64(activeProposals))
}

// UpdateValidator() updates the validator metrics for prometheus
func (m *Metrics) UpdateValidator(address string, stakeAmount uint64, unstaking, paused, delegate, compounding bool) {
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
	// update the status metric
	switch {
	case unstaking:
		// if the val is unstaking
		m.ValidatorStatus.WithLabelValues(address).Set(1)
	case paused:
		// if the val is paused
		m.ValidatorStatus.WithLabelValues(address).Set(2)
	default:
		// if the val is active
		m.ValidatorStatus.WithLabelValues(address).Set(0)
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

// UpdateIndexer() updates the indexer metrics
func (m *Metrics) UpdateIndexer(address string, received, sent uint64) {
	// exit if empty
	if m == nil {
		return
	}
	// updates the number of transactions received
	m.TxsReceived.WithLabelValues(address).Add(float64(received))
	// updates the number of transactions sent
	m.TxsSent.WithLabelValues(address).Add(float64(sent))
}

// UpdateBlockMetrics() updates the metrics about the last block
func (m *Metrics) UpdateBlockMetrics(proposerAddress []byte, txCount int, duration time.Duration) {
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
	m.TransactionCount.Add(float64(txCount))
	// update the block processing time in seconds
	m.BlockProcessingTime.Observe(duration.Seconds())
}
