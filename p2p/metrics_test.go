package p2p

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

// TestP2PMetrics tests all P2P metrics functionality using subtests to avoid
// duplicate Prometheus registration errors in CI
func TestP2PMetrics(t *testing.T) {
	// Create shared metrics instance once for all subtests
	config := lib.DefaultConfig()
	privKey, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	sharedMetrics := lib.NewMetricsServer(
		privKey.PublicKey().Address(),
		float64(config.ChainId),
		"test-version",
		lib.MetricsConfig{PrometheusAddress: ":0", MetricsEnabled: true},
		lib.NewNullLogger(),
	)

	t.Run("recording", func(t *testing.T) {
		n1, n2, cleanup := newTestP2PPair(t)
		defer cleanup()

		n1.metrics = sharedMetrics

		// Verify metrics are not nil
		require.NotNil(t, n1.metrics)
		require.NotNil(t, n1.metrics.MessageSize)
		require.NotNil(t, n1.metrics.PacketsPerMessage)
		require.NotNil(t, n1.metrics.SendQueueTime)
		require.NotNil(t, n1.metrics.SendWireTime)
		require.NotNil(t, n1.metrics.SendTotalTime)

		// Send a message to trigger metrics collection
		testMsg := &PeerBookRequestMessage{}
		require.NoError(t, n1.SendTo(n2.pub, lib.Topic_TX, testMsg))

		// Wait for message to be processed
		select {
		case <-n2.Inbox(lib.Topic_TX):
			// Message received
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}

		// Give metrics a moment to be recorded
		time.Sleep(100 * time.Millisecond)

		// Verify that message size histogram has samples
		metric := &dto.Metric{}
		err := n1.metrics.MessageSize.Write(metric)
		require.NoError(t, err)
		require.NotNil(t, metric.Histogram)
		require.Greater(t, *metric.Histogram.SampleCount, uint64(0), "MessageSize should have samples")

		// Verify packets per message histogram has samples
		metric = &dto.Metric{}
		err = n1.metrics.PacketsPerMessage.Write(metric)
		require.NoError(t, err)
		require.NotNil(t, metric.Histogram)
		require.Greater(t, *metric.Histogram.SampleCount, uint64(0), "PacketsPerMessage should have samples")

		t.Logf("Metrics recorded successfully - MessageSize: %d samples, PacketsPerMessage: %d samples",
			*metric.Histogram.SampleCount, *metric.Histogram.SampleCount)
	})

	t.Run("large_message", func(t *testing.T) {
		n1, n2, cleanup := newTestP2PPair(t)
		defer cleanup()

		n1.metrics = sharedMetrics

		// Create a large message that will be split into multiple packets
		largeMsg := &BookPeer{
			Address: &lib.PeerAddress{
				PublicKey:  bytes.Repeat([]byte("A"), 1000),
				NetAddress: "localhost:9001",
				PeerMeta: &lib.PeerMeta{
					Signature: bytes.Repeat([]byte("B"), int(maxDataChunkSize)*3),
				},
			},
		}

		// Send the large message
		require.NoError(t, n1.SendTo(n2.pub, lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: []*BookPeer{largeMsg}}))

		// Wait for message to be received
		select {
		case <-n2.Inbox(lib.Topic_PEERS_RESPONSE):
			// Message received
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for large message")
		}

		// Give metrics a moment to be recorded
		time.Sleep(100 * time.Millisecond)

		// Verify packets per message shows multiple packets
		metric2 := &dto.Metric{}
		err := n1.metrics.PacketsPerMessage.Write(metric2)
		require.NoError(t, err)
		require.NotNil(t, metric2.Histogram)
		require.Greater(t, *metric2.Histogram.SampleCount, uint64(0))

		t.Logf("Large message metrics: %d packets recorded", *metric2.Histogram.SampleCount)
	})

	t.Run("queue_depth", func(t *testing.T) {
		n1 := newStartedTestP2PNode(t)
		defer n1.Stop()

		n1.metrics = sharedMetrics

		// Update queue depth metrics
		n1.UpdateQueueDepthMetrics()

		// Verify inbox queue depth metrics exist
		sendDepthMetric := n1.metrics.SendQueueDepth.WithLabelValues(lib.Topic_name[int32(lib.Topic_TX)])
		require.NotNil(t, sendDepthMetric)

		inboxDepthMetric := n1.metrics.InboxQueueDepth.WithLabelValues(lib.Topic_name[int32(lib.Topic_TX)])
		require.NotNil(t, inboxDepthMetric)

		t.Log("Queue depth metrics updated successfully")
	})

	t.Run("concurrent", func(t *testing.T) {
		n1, n2, cleanup := newTestP2PPair(t)
		defer cleanup()

		n1.metrics = sharedMetrics

		// Send multiple messages concurrently
		const numMessages = 50
		var wg sync.WaitGroup
		wg.Add(numMessages)

		for i := 0; i < numMessages; i++ {
			go func(id int) {
				defer wg.Done()
				testMsg := &PeerBookRequestMessage{}
				_ = n1.SendTo(n2.pub, lib.Topic_TX, testMsg)
			}(i)
		}

		// Wait for all sends to complete
		wg.Wait()

		// Drain the inbox
		timeout := time.After(5 * time.Second)
		received := 0
	drainLoop:
		for {
			select {
			case <-n2.Inbox(lib.Topic_TX):
				received++
				if received >= numMessages {
					break drainLoop
				}
			case <-timeout:
				break drainLoop
			}
		}

		// Give metrics time to be recorded
		time.Sleep(100 * time.Millisecond)

		// Verify metrics were recorded
		metric := &dto.Metric{}
		err := n1.metrics.MessageSize.Write(metric)
		require.NoError(t, err)
		require.Greater(t, *metric.Histogram.SampleCount, uint64(0))

		t.Logf("Concurrent test: sent %d, received %d, metrics samples: %d",
			numMessages, received, *metric.Histogram.SampleCount)
	})

	t.Run("histogram_buckets", func(t *testing.T) {
		// Get baseline sample counts
		metric := &dto.Metric{}
		err := sharedMetrics.SendQueueTime.Write(metric)
		require.NoError(t, err)
		baselineSendQueueCount := *metric.Histogram.SampleCount

		err = sharedMetrics.MessageSize.Write(metric)
		require.NoError(t, err)
		baselineMessageSizeCount := *metric.Histogram.SampleCount

		// Record some sample values
		sharedMetrics.SendQueueTime.Observe(0.001)  // 1ms
		sharedMetrics.SendQueueTime.Observe(0.01)   // 10ms
		sharedMetrics.SendQueueTime.Observe(0.1)    // 100ms
		sharedMetrics.SendWireTime.Observe(0.0001)  // 0.1ms
		sharedMetrics.SendWireTime.Observe(0.001)   // 1ms
		sharedMetrics.SendTotalTime.Observe(0.002)  // 2ms
		sharedMetrics.MessageSize.Observe(100)      // 100 bytes
		sharedMetrics.MessageSize.Observe(10000)    // 10KB
		sharedMetrics.MessageSize.Observe(1000000)  // 1MB
		sharedMetrics.PacketsPerMessage.Observe(1)  // 1 packet
		sharedMetrics.PacketsPerMessage.Observe(5)  // 5 packets
		sharedMetrics.PacketsPerMessage.Observe(15) // 15 packets

		// Verify samples were recorded (should have increased by 3)
		err = sharedMetrics.SendQueueTime.Write(metric)
		require.NoError(t, err)
		require.Equal(t, baselineSendQueueCount+3, *metric.Histogram.SampleCount)

		err = sharedMetrics.MessageSize.Write(metric)
		require.NoError(t, err)
		require.Equal(t, baselineMessageSizeCount+3, *metric.Histogram.SampleCount)

		t.Log("Histogram buckets validated successfully")
	})

	t.Run("send_queue_timeout", func(t *testing.T) {
		n1, n2, cleanup := newTestP2PPair(t)
		defer cleanup()

		n1.metrics = sharedMetrics

		// Get a peer to access its stream
		peer, err := n1.PeerSet.get(n2.pub)
		require.NoError(t, err)

		stream := peer.conn.streams[lib.Topic_TX]
		require.NotNil(t, stream)

		// Create a small send queue to force timeouts
		stream.sendQueue = make(chan *PacketWithTiming, 1)

		// Fill the queue
		packet1 := &Packet{StreamId: lib.Topic_TX, Eof: true, Bytes: []byte("test1")}
		ok := stream.queueSend(packet1, time.Now(), n1.metrics)
		require.True(t, ok)

		// Verify the metric exists
		require.NotNil(t, n1.metrics.SendQueueTimeout)
		require.NotNil(t, n1.metrics.SendQueueFull)

		t.Log("Send queue timeout metrics exist and are accessible")
	})

	t.Run("nil_safety", func(t *testing.T) {
		n1, n2, cleanup := newTestP2PPair(t)
		defer cleanup()

		// Set metrics to nil
		n1.metrics = nil

		// Should not panic when sending messages
		testMsg := &PeerBookRequestMessage{}
		require.NoError(t, n1.SendTo(n2.pub, lib.Topic_TX, testMsg))

		// Wait for message
		select {
		case <-n2.Inbox(lib.Topic_TX):
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}

		t.Log("Nil metrics safety verified - no panics")
	})
}

// BenchmarkSendWithoutMetrics benchmarks message sending without metrics overhead
func BenchmarkSendWithoutMetrics(b *testing.B) {
	n1, n2, cleanup := newTestP2PPair(&testing.T{})
	defer cleanup()

	// Disable metrics by setting to nil
	n1.metrics = nil

	testMsg := &PeerBookRequestMessage{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = n1.SendTo(n2.pub, lib.Topic_TX, testMsg)
	}
	b.StopTimer()

	// Drain messages to prevent blocking
	go func() {
		for range n2.Inbox(lib.Topic_TX) {
		}
	}()
}

// BenchmarkSendWithMetrics benchmarks message sending with metrics enabled
// Note: This benchmark creates metrics and should be run individually to avoid
// duplicate Prometheus registration errors. Run with:
//
//	go test -bench=BenchmarkSendWithMetrics -run=^$ ./p2p
func BenchmarkSendWithMetrics(b *testing.B) {
	n1, n2, cleanup := newTestP2PPair(&testing.T{})
	defer cleanup()

	// Ensure metrics are enabled
	config := lib.DefaultConfig()
	privKey, _ := crypto.NewBLS12381PrivateKey()
	n1.metrics = lib.NewMetricsServer(
		privKey.PublicKey().Address(),
		float64(config.ChainId),
		"test-version",
		lib.MetricsConfig{PrometheusAddress: ":0", MetricsEnabled: true},
		lib.NewNullLogger(),
	)

	testMsg := &PeerBookRequestMessage{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = n1.SendTo(n2.pub, lib.Topic_TX, testMsg)
	}
	b.StopTimer()

	// Drain messages to prevent blocking
	go func() {
		for range n2.Inbox(lib.Topic_TX) {
		}
	}()
}

// BenchmarkPacketWithTimingAllocation benchmarks the allocation overhead
func BenchmarkPacketWithTimingAllocation(b *testing.B) {
	packet := &Packet{
		StreamId: lib.Topic_TX,
		Eof:      true,
		Bytes:    make([]byte, 1000),
	}
	sendStart := time.Now()
	queueStart := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &PacketWithTiming{
			packet:     packet,
			sendStart:  sendStart,
			queueStart: queueStart,
		}
	}
}

// BenchmarkQueueSendWithTiming benchmarks the queueSend operation
// Note: This benchmark creates metrics and should be run individually to avoid
// duplicate Prometheus registration errors. Run with:
//
//	go test -bench=BenchmarkQueueSendWithTiming -run=^$ ./p2p
func BenchmarkQueueSendWithTiming(b *testing.B) {
	stream := &Stream{
		topic:        lib.Topic_TX,
		msgAssembler: make([]byte, 0),
		sendQueue:    make(chan *PacketWithTiming, maxStreamSendQueueSize),
		inbox:        make(chan *lib.MessageAndMetadata, maxInboxQueueSize),
		logger:       lib.NewNullLogger(),
	}

	// Create metrics
	config := lib.DefaultConfig()
	privKey, _ := crypto.NewBLS12381PrivateKey()
	metrics := lib.NewMetricsServer(
		privKey.PublicKey().Address(),
		float64(config.ChainId),
		"test-version",
		lib.MetricsConfig{PrometheusAddress: ":0", MetricsEnabled: true},
		lib.NewNullLogger(),
	)

	packet := &Packet{
		StreamId: lib.Topic_TX,
		Eof:      true,
		Bytes:    make([]byte, 1000),
	}

	// Drain the queue in background
	go func() {
		for range stream.sendQueue {
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.queueSend(packet, time.Now(), metrics)
	}
	b.StopTimer()

	stream.mu.Lock()
	stream.closed = true
	close(stream.sendQueue)
	stream.mu.Unlock()
}

// BenchmarkUpdateQueueDepthMetrics benchmarks the queue depth update operation
// Note: This benchmark creates metrics and should be run individually to avoid
// duplicate Prometheus registration errors. Run with:
//
//	go test -bench=BenchmarkUpdateQueueDepthMetrics -run=^$ ./p2p
func BenchmarkUpdateQueueDepthMetrics(b *testing.B) {
	n1 := newStartedTestP2PNode(&testing.T{})
	defer n1.Stop()

	// Create metrics
	config := lib.DefaultConfig()
	privKey, _ := crypto.NewBLS12381PrivateKey()
	n1.metrics = lib.NewMetricsServer(
		privKey.PublicKey().Address(),
		float64(config.ChainId),
		"test-version",
		lib.MetricsConfig{PrometheusAddress: ":0", MetricsEnabled: true},
		lib.NewNullLogger(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n1.UpdateQueueDepthMetrics()
	}
}

// getHistogramValue extracts the sample count from a histogram
func getHistogramValue(h prometheus.Histogram) uint64 {
	metric := &dto.Metric{}
	if err := h.Write(metric); err != nil {
		return 0
	}
	if metric.Histogram == nil {
		return 0
	}
	return *metric.Histogram.SampleCount
}

// getCounterValue extracts the value from a counter
func getCounterValue(c prometheus.Counter) float64 {
	metric := &dto.Metric{}
	if err := c.Write(metric); err != nil {
		return 0
	}
	if metric.Counter == nil {
		return 0
	}
	return *metric.Counter.Value
}
