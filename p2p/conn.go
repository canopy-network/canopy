package p2p

import (
	"encoding/binary"
	"io"
	"math/rand/v2"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/lib"
	limiter "github.com/mxk/go-flowrate/flowrate"
	"google.golang.org/protobuf/proto"
)

const (
	maxChunksPerPacket     = 256                                 // maximum number of chunks per packet - *1* means chunking disabled
	maxDataChunkSize       = maxPacketSize - packetHeaderSize    // maximum size of the chunk of bytes in a packet
	maxPacketSize          = maxMessageSize / maxChunksPerPacket // maximum size of the full packet
	packetHeaderSize       = 50                                  // the overhead of the protobuf packet header
	queueSendTimeout       = 10 * time.Second                    // how long a message waits to be queued before throwing an error
	maxMessageSize         = uint32(256 * units.MB)              // the maximum total size of a message once all the packets are added up
	dataFlowRatePerS       = maxMessageSize                      // the maximum number of bytes that may be sent or received per second per MultiConn
	maxChanSize            = 1                                   // maximum number of items in a channel before blocking
	maxInboxQueueSize      = 1000                                // maximum number of items in inbox queue before blocking
	maxStreamSendQueueSize = 1000                                // maximum number of items in a stream send queue before blocking
	keepAlivePeriod        = 10 * time.Second                    // TCP keep-alive probe interval
	heartbeatInterval      = time.Second                         // how often to send heartbeat pings
	heartbeatTimeout       = 2 * time.Second                     // how long to wait for liveness before dropping the peer
	heartbeatTopic         = lib.Topic_HEARTBEAT                 // dedicated heartbeat stream
	heartbeatPing          = "ping"
	heartbeatPong          = "pong"

	// "Peer Reputation Points" are actively maintained for each peer the node is connected to
	// These points allow a node to track peer behavior over its lifetime, allowing it to disconnect from faulty peers
	PollMaxHeightTimeoutS   = 1   // wait time for polling the maximum height of the peers
	SyncTimeoutS            = 5   // wait time to receive an individual block (certificate) from a peer during syncing
	MaxBlockReqPerWindow    = 20  // maximum block (certificate) requests per window per requester
	BlockReqWindowS         = 2   // the 'window of time' before resetting limits for block (certificate) requests
	GoodPeerBookRespRep     = 3   // reputation points for a good peer book response
	GoodBlockRep            = 3   // rep boost for sending us a valid block (certificate)
	UnexpectedBlockRep      = -1  // rep slash for sending us a block we weren't expecting
	InvalidMsgRep           = -3  // slash for an invalid message
	ExceedMaxPBReqRep       = -3  // slash for exceeding the max peer book requests
	UnknownMessageSlash     = -3  // unknown message type is received
	BadStreamSlash          = -3  // unknown stream id is received
	InvalidTxRep            = -3  // rep slash for sending us an invalid transaction
	InvalidBlockRep         = -3  // rep slash for sending an invalid block (certificate) message
	BlockReqExceededRep     = -3  // rep slash for over-requesting blocks (certificates)
	MaxMessageExceededSlash = -10 // slash for sending a 'Message (sum of Packets)' above the allowed maximum size
)

var (
	ReadTimeout  = 40 * time.Second // this is just the default; it gets set by config upon initialization
	WriteTimeout = 80 * time.Second // this is just the default; it gets set by config upon initialization
)

// MultiConn: A rate-limited, multiplexed connection that utilizes a series streams with varying priority for sending and receiving
type MultiConn struct {
	conn          net.Conn                            // underlying connection
	uuid          uint64                              // the unique connection id to prevent race conditions around peer removal logic
	Address       *lib.PeerAddress                    // authenticated peer information
	streams       map[lib.Topic]*Stream               // multiple independent bi-directional communication channels
	quitSending   chan struct{}                       // signal to quit
	quitReceiving chan struct{}                       // signal to quit
	onError       func(error, []byte, string, uint64) // callback to call if peer errors
	error         sync.Once                           // thread safety to ensure MultiConn.onError is only called once
	p2p           *P2P                                // a pointer reference to the P2P module
	close         sync.Once                           // flag to identify if MultiConn is closed
	log           lib.LoggerI                         // logging
	hasError      atomic.Bool                         // flag to identify if MultiConn has encountered an error
	lastPong      atomic.Int64                        // last time we saw a pong (unix nano)
	lastHeard     atomic.Int64                        // last time we read any message from the wire (unix nano)
	lastPingSent  atomic.Int64                        // last time we queued a ping (unix nano)
	lastPingRecv  atomic.Int64                        // last time we received a ping (unix nano)
	lastPongSent  atomic.Int64                        // last time we queued a pong (unix nano)
}

// NewConnection() creates and starts a new instance of a MultiConn
func (p *P2P) NewConnection(conn net.Conn) (*MultiConn, lib.ErrorI) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetWriteBuffer(32 * 1024 * 1024); err != nil {
			p.log.Warnf("Failed to set write buffer: %v", err)
		}
		if err := tcpConn.SetReadBuffer(32 * 1024 * 1024); err != nil {
			p.log.Warnf("Failed to set write buffer: %v", err)
		}
		if err := tcpConn.SetNoDelay(true); err != nil {
			p.log.Warnf("Failed to disable Nagle: %v", err)
		}
		if err := tcpConn.SetKeepAlive(true); err != nil {
			p.log.Warnf("Failed to enable TCP keepalive: %v", err)
		} else if err := tcpConn.SetKeepAlivePeriod(keepAlivePeriod); err != nil {
			p.log.Warnf("Failed to set TCP keepalive period: %v", err)
		}
	}
	// establish an encrypted connection using the handshake
	eConn, err := NewHandshake(conn, p.meta, p.privateKey)
	if err != nil {
		return nil, err
	}
	c := &MultiConn{
		conn:          eConn,
		uuid:          rand.Uint64(),
		Address:       eConn.Address,
		streams:       p.NewStreams(),
		quitSending:   make(chan struct{}, maxChanSize),
		quitReceiving: make(chan struct{}, maxChanSize),
		onError:       p.OnPeerError,
		error:         sync.Once{},
		p2p:           p,
		close:         sync.Once{},
		log:           p.log,
	}
	now := time.Now().UnixNano()
	c.lastPong.Store(now)
	c.lastHeard.Store(now)
	c.lastPingSent.Store(0)
	c.lastPingRecv.Store(0)
	c.lastPongSent.Store(0)
	_ = c.conn.SetReadDeadline(time.Time{})
	_ = c.conn.SetWriteDeadline(time.Time{})
	// start the connection service
	c.Start()
	return c, err
}

// Start() begins send and receive services for a MultiConn
func (c *MultiConn) Start() {
	go c.startSendService()
	go c.startReceiveService()
	go c.startHeartbeat()
}

// Stop() sends exit signals for send and receive loops and closes the connection
func (c *MultiConn) Stop() {
	defer lib.TimeTrack(c.log, time.Now(), time.Second)
	c.close.Do(func() {
		c.p2p.log.Debugf("Stopping peer %s", lib.BytesToString(c.Address.PublicKey))
		c.quitReceiving <- struct{}{}
		c.quitSending <- struct{}{}
		close(c.quitSending)
		close(c.quitReceiving)
		_ = c.conn.Close()

		for _, stream := range c.streams {
			stream.cleanup()
		}
	})
}

// Send() queues the sending of a message to a specific Stream
func (c *MultiConn) Send(topic lib.Topic, bz []byte) (ok bool) {
	defer lib.TimeTrack(c.log, time.Now(), time.Second)
	startTime := time.Now()
	stream, ok := c.streams[topic]
	if !ok {
		c.log.Errorf("Stream %s does not exist", topic)
		return
	}
	chunks := split(bz, int(maxDataChunkSize))
	var packets []*Packet
	for i, chunk := range chunks {
		packets = append(packets, &Packet{
			StreamId: topic,
			Eof:      i == len(chunks)-1,
			Bytes:    chunk,
		})
	}
	if c.p2p.metrics != nil {
		c.p2p.metrics.MessageSize.Observe(float64(len(bz)))
		c.p2p.metrics.PacketsPerMessage.Observe(float64(len(packets)))
	}
	ok = stream.queueSends(packets, startTime, c.p2p.metrics)
	if !ok {
		c.log.Errorf("Packet(ID:%s) packet failed in queue for: %s", lib.Topic_name[int32(topic)], lib.BytesToTruncatedString(c.Address.PublicKey))
	}
	return
}

// startSendService() starts the main send service
// - converges and writes the send queue from all streams into the underlying tcp connection.
// - manages the keep alive protocol by sending pings and monitoring the receipt of the corresponding pong
func (c *MultiConn) startSendService() {
	defer func() {
		if r := recover(); r != nil {
			c.log.Errorf("panic recovered, err: %s, stack: %s", r, string(debug.Stack()))
		}
	}()
	m := limiter.New(0, 0)
	var pwt *PacketWithTiming
	defer func() { m.Done() }()
	for {
		// select statement ensures the sequential coordination of the concurrent processes
		select {
		case pwt = <-c.streams[heartbeatTopic].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_CONSENSUS].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_BLOCK].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_BLOCK_REQUEST].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_TX].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_PEERS_RESPONSE].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case pwt = <-c.streams[lib.Topic_PEERS_REQUEST].sendQueue:
			c.sendPacketWithTiming(pwt, m)
		case <-c.quitSending: // fires when Stop() is called
			return
		}
	}
}

// startReceiveService() starts the main receive service
// - reads from the underlying tcp connection and 'routes' the messages to the appropriate streams
// - manages keep alive protocol by notifying the 'send service' of pings and pongs
func (c *MultiConn) startReceiveService() {
	defer func() {
		if r := recover(); r != nil {
			c.log.Errorf("panic recovered, err: %s, stack: %s", r, string(debug.Stack()))
		}
	}()
	m := limiter.New(0, 0)
	defer m.Done()
	for {
		select {
		default: // fires unless quit was signaled
			// waits until bytes are received from the conn
			msg, err := c.waitForAndHandleWireBytes(m)
			if err != nil {
				c.Error(err)
				return
			}
			c.lastHeard.Store(time.Now().UnixNano())
			// handle different message types
			switch x := msg.(type) {
			case *Packet: // receive packet is a partial or full 'Message' with a Stream Topic designation and an EOF signal
				if x.StreamId == heartbeatTopic {
					c.handleHeartbeatPacket(x)
					continue
				}
				// load the proper stream
				stream, found := c.streams[x.StreamId]
				if !found {
					c.Error(ErrBadStream(), BadStreamSlash)
					return
				}
				// get the peer info from the peer set
				info, e := c.p2p.GetPeerInfo(c.Address.PublicKey)
				if e != nil {
					c.Error(e)
					return
				}
				// handle the packet within the stream
				if slash, er := stream.handlePacket(info, x, c.p2p.metrics); er != nil {
					c.log.Warnf(er.Error())
					c.Error(er, slash)
					return
				}
			default: // unknown type results in slash and exiting the service
				c.Error(ErrUnknownP2PMsg(x), UnknownMessageSlash)
				return
			}
		case <-c.quitReceiving: // fires when quit is signaled
			return
		}
	}
}

// startHeartbeat periodically sends ping packets and drops the peer if no inbound traffic is seen in time.
//
// Using "any inbound message" for liveness avoids disconnecting a busy connection just because a pong
// was delayed/lost, while still reacting quickly to a stalled link.
func (c *MultiConn) startHeartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if time.Since(time.Unix(0, c.lastHeard.Load())) > heartbeatTimeout {
				if c.p2p.metrics != nil {
					c.p2p.metrics.HeartbeatTimeout.Inc()
				}
				lastHeardAge := time.Since(time.Unix(0, c.lastHeard.Load()))
				lastPongAge := time.Since(time.Unix(0, c.lastPong.Load()))
				lastPingSentAge := time.Duration(0)
				if ts := c.lastPingSent.Load(); ts != 0 {
					lastPingSentAge = time.Since(time.Unix(0, ts))
				}
				lastPingRecvAge := time.Duration(0)
				if ts := c.lastPingRecv.Load(); ts != 0 {
					lastPingRecvAge = time.Since(time.Unix(0, ts))
				}
				lastPongSentAge := time.Duration(0)
				if ts := c.lastPongSent.Load(); ts != 0 {
					lastPongSentAge = time.Since(time.Unix(0, ts))
				}

				c.log.Warnf(
					"Heartbeat timeout: closing peer %s lastHeardAge=%s lastPongAge=%s lastPingSentAge=%s lastPingRecvAge=%s lastPongSentAge=%s",
					lib.BytesToTruncatedString(c.Address.PublicKey),
					lastHeardAge, lastPongAge, lastPingSentAge, lastPingRecvAge, lastPongSentAge,
				)
				c.Error(ErrPongTimeout())
				return
			}
			c.sendHeartbeat()
		case <-c.quitSending:
			return
		case <-c.quitReceiving:
			return
		}
	}
}

func (c *MultiConn) sendHeartbeat() {
	stream, ok := c.streams[heartbeatTopic]
	if !ok {
		return
	}
	sendStart := time.Now()
	if ok := stream.queueSend(&Packet{
		StreamId: heartbeatTopic,
		Eof:      true,
		Bytes:    []byte(heartbeatPing),
	}, sendStart, c.p2p.metrics); ok {
		c.lastPingSent.Store(sendStart.UnixNano())
		if c.p2p.metrics != nil {
			c.p2p.metrics.HeartbeatPingSent.Inc()
		}
	}
}

func (c *MultiConn) handleHeartbeatPacket(packet *Packet) {
	switch string(packet.Bytes) {
	case heartbeatPing:
		c.lastPingRecv.Store(time.Now().UnixNano())
		if c.p2p.metrics != nil {
			c.p2p.metrics.HeartbeatPingRecv.Inc()
		}
		// respond
		stream, ok := c.streams[heartbeatTopic]
		if !ok {
			return
		}
		sendStart := time.Now()
		if ok := stream.queueSend(&Packet{
			StreamId: heartbeatTopic,
			Eof:      true,
			Bytes:    []byte(heartbeatPong),
		}, sendStart, c.p2p.metrics); ok {
			c.lastPongSent.Store(sendStart.UnixNano())
			if c.p2p.metrics != nil {
				c.p2p.metrics.HeartbeatPongSent.Inc()
			}
		}
	case heartbeatPong:
		now := time.Now()
		c.lastPong.Store(now.UnixNano())
		if c.p2p.metrics != nil {
			c.p2p.metrics.HeartbeatPongRecv.Inc()
			if ts := c.lastPingSent.Load(); ts != 0 {
				rtt := now.Sub(time.Unix(0, ts)).Seconds()
				if rtt >= 0 {
					c.p2p.metrics.HeartbeatRTT.Observe(rtt)
				}
			}
		}
	default:
		c.log.Warnf("Unknown heartbeat payload from %s", lib.BytesToTruncatedString(c.Address.PublicKey))
	}
}

// Error() when an error occurs on the MultiConn execute a callback. Optionally pass a reputation delta to slash the peer
func (c *MultiConn) Error(err error, reputationDelta ...int32) {
	if len(reputationDelta) == 1 {
		c.p2p.ChangeReputation(c.Address.PublicKey, reputationDelta[0])
	}
	// call onError() for the peer
	c.error.Do(func() {
		// mark the connection as failed under lock, but avoid holding the lock during teardown
		unlock := lockWithTrace("p2p", &c.p2p.mux, c.p2p.log)
		c.hasError.Store(true)
		// run the callback after releasing the lock to prevent blocking other readers
		c.onError(err, c.Address.PublicKey, c.conn.RemoteAddr().String(), c.uuid)
		unlock()
		// stop the multi-conn
		c.Stop()
	})
}

// waitForAndHandleWireBytes() a rate limited handler of inbound bytes from the wire.
// Blocks until bytes are received converts bytes into a proto.Message using an Envelope
func (c *MultiConn) waitForAndHandleWireBytes(m *limiter.Monitor) (proto.Message, lib.ErrorI) {
	receiveStart := time.Now()
	// initialize the wrapper object
	msg := new(Envelope)
	// restrict the instantaneous data flow to rate bytes per second
	// Limit() request maxPacketSize bytes from the limiter and the limiter
	// will block the execution until at or below the desired rate of flow
	//m.Limit(int(maxPacketSize), int64(dataFlowRatePerS), true)
	// read the proto message from the wire
	_, err := receiveProtoMsg(c.conn, msg)
	if err != nil {
		return nil, err
	}
	if c.p2p.metrics != nil {
		c.p2p.metrics.ReceiveWireTime.Observe(time.Since(receiveStart).Seconds())
	}
	// update the limiter
	//m.Update(lenM)
	// unmarshal the payload from proto.any
	return lib.FromAny(msg.Payload)
}

// sendPacketWithTiming() a rate limited writer with metrics tracking
func (c *MultiConn) sendPacketWithTiming(pwt *PacketWithTiming, m *limiter.Monitor) {
	if pwt == nil || pwt.packet == nil {
		return
	}
	dequeueTime := time.Now()
	if c.p2p.metrics != nil {
		queueDuration := dequeueTime.Sub(pwt.queueStart).Seconds()
		c.p2p.metrics.SendQueueTime.Observe(queueDuration)
	}
	wireStart := time.Now()
	c.sendWireBytes(pwt.packet, m)
	if c.p2p.metrics != nil {
		wireDuration := time.Since(wireStart).Seconds()
		totalDuration := time.Since(pwt.sendStart).Seconds()
		c.p2p.metrics.SendWireTime.Observe(wireDuration)
		c.p2p.metrics.SendTotalTime.Observe(totalDuration)
	}
}

// sendPacket() a rate limited writer of outbound bytes to the wire
// wraps a proto.Message into a universal Envelope, then converts to bytes and
// sends them across the wire without violating the data flow rate limits
// message may be a Packet, a Ping or a Pong
func (c *MultiConn) sendPacket(packet *Packet, m *limiter.Monitor) {
	if packet != nil {
		//c.log.Debugf("Send Packet to %s (ID:%s, L:%d, E:%t), hash: %s",
		//	lib.BytesToTruncatedString(c.Address.PublicKey),
		//	lib.Topic_name[int32(packet.StreamId)],
		//	len(packet.Bytes),
		//	packet.Eof,
		//	crypto.ShortHashString(packet.Bytes),
		//)
	}
	// send packet as message over the wire
	c.sendWireBytes(packet, m)
}

// sendWireBytes() a rate limited writer of outbound bytes to the wire
// wraps a proto.Message into a universal Envelope, then converts to bytes and
// sends them across the wire without violating the data flow rate limits
// message may be a Packet, a Ping or a Pong
func (c *MultiConn) sendWireBytes(message proto.Message, m *limiter.Monitor) {
	defer lib.TimeTrack(c.log, time.Now(), time.Second)
	// convert the proto.Message into a proto.Any
	a, err := lib.NewAny(message)
	if err != nil {
		c.Error(err)
	}
	// restrict the instantaneous data flow to rate bytes per second
	// Limit() request maxPacketSize bytes from the limiter and the limiter
	// will block the execution until at or below the desired rate of flow
	m.Limit(int(maxPacketSize), int64(dataFlowRatePerS), true)
	//defer lib.TimeTrack(c.log, time.Now())
	// send the proto message wrapped in an Envelope over the wire
	lenM, err := sendProtoMsg(c.conn, &Envelope{Payload: a})
	if err != nil {
		c.Error(err)
	}
	// update the rate limiter with how many bytes were written
	m.Update(lenM)
}

// PacketWithTiming wraps a Packet with timing information for metrics
type PacketWithTiming struct {
	packet     *Packet
	sendStart  time.Time // when Send() was called
	queueStart time.Time // when packet was queued
}

// Stream: an independent, bidirectional communication channel that is scoped to a single topic.
// In a multiplexed connection there is typically more than one stream per connection
type Stream struct {
	topic        lib.Topic                    // the subject and priority of the stream
	sendQueue    chan *PacketWithTiming       // a queue of unsent messages with timing
	msgAssembler []byte                       // collects and adds incoming packets until the entire message is received (EOF signal)
	inbox        chan *lib.MessageAndMetadata // the channel where fully received messages are held for other parts of the app to read
	mu           sync.Mutex                   // mutex to prevent race conditions when sending packets (all packets of the same message should be one right after the other)
	closed       bool                         // flag to identify if stream is closed
	logger       lib.LoggerI
}

// queueSends() schedules the packets to be sent ensuring coordination with the mutex
func (s *Stream) queueSends(packets []*Packet, sendStart time.Time, metrics *lib.Metrics) bool {
	defer lib.TimeTrack(s.logger, time.Now(), time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, packet := range packets {
		ok := s.queueSend(packet, sendStart, metrics)
		if !ok {
			return false
		}
	}
	return true
}

// queueSend() schedules the packet to be sent
func (s *Stream) queueSend(p *Packet, sendStart time.Time, metrics *lib.Metrics) bool {
	if s.closed {
		return false
	}
	queueStart := time.Now()
	pwt := &PacketWithTiming{
		packet:     p,
		sendStart:  sendStart,
		queueStart: queueStart,
	}
	select {
	case s.sendQueue <- pwt: // enqueue to the back of the line
		return true
	case <-time.After(queueSendTimeout): // may timeout if queue remains full
		if metrics != nil {
			metrics.SendQueueTimeout.Inc()
			metrics.SendQueueFull.WithLabelValues(lib.Topic_name[int32(p.StreamId)]).Inc()
		}
		return false
	}
}

// handlePacket() merge the new packet with the previously received ones until the entire message is complete (EOF signal)
func (s *Stream) handlePacket(peerInfo *lib.PeerInfo, packet *Packet, metrics *lib.Metrics) (int32, lib.ErrorI) {
	assemblyStart := time.Now()
	msgAssemblerLen, packetLen := len(s.msgAssembler), len(packet.Bytes)
	//s.logger.Debugf("Received Packet from %s (ID:%s, L:%d, E:%t), hash: %s",
	//	lib.BytesToTruncatedString(peerInfo.Address.PublicKey),
	//	lib.Topic_name[int32(packet.StreamId)],
	//	len(packet.Bytes),
	//	packet.Eof,
	//	crypto.ShortHashString(packet.Bytes),
	//)
	//defer s.logger.Debugf("Done receiving: %s", crypto.ShortHashString(packet.Bytes))
	// if the addition of this new packet pushes the total message size above max
	if int(maxMessageSize) < msgAssemblerLen+packetLen {
		s.msgAssembler = s.msgAssembler[:0]
		return MaxMessageExceededSlash, ErrMaxMessageSize()
	}
	// combine this packet with the previously received ones
	s.msgAssembler = append(s.msgAssembler, packet.Bytes...)
	// if the packet is signalling message end
	if packet.Eof {
		// create a buffer to retain the message assembler bytes
		msg := make([]byte, len(s.msgAssembler))
		// copy the message assembler bytes into a buffer
		copy(msg, s.msgAssembler)
		// wrap with metadata
		m := &lib.MessageAndMetadata{
			Message: msg,
			Sender:  peerInfo,
		}
		if metrics != nil {
			metrics.ReceiveAssemblyTime.Observe(time.Since(assemblyStart).Seconds())
		}
		//s.logger.Debugf("Inbox %s queue: %d", lib.Topic_name[int32(packet.StreamId)], len(s.inbox))
		// add to inbox for other parts of the app to read
		select {
		case s.inbox <- m:
			if len(s.inbox) > maxInboxQueueSize/4 {
				s.logger.Errorf("OVERSIZE INBOX: %d", len(s.inbox))
			}
		default:
			s.logger.Errorf("CRITICAL: Inbox %s queue full in receive service", lib.Topic_name[int32(packet.StreamId)])
			s.logger.Error("Dropping all messages")
			// drain inbox
			func() {
				for {
					select {
					case <-s.inbox:
						// drop
					default:
						// channel is empty now
						return
					}
				}
			}()
		}
		// reset receiving buffer
		s.msgAssembler = s.msgAssembler[:0]
	}
	return 0, nil
}

// HELPERS BELOW

// sendProtoMsg() encodes and sends a length-prefixed proto message to a net.Conn
// returns length of message bytes
func sendProtoMsg(conn net.Conn, ptr proto.Message, timeout ...time.Duration) (int, lib.ErrorI) {
	// marshal into proto bytes
	bz, err := lib.Marshal(ptr)
	if err != nil {
		return 0, err
	}
	// send the bytes prefixed by length
	return len(bz), sendLengthPrefixed(conn, bz, timeout...)
}

// receiveProtoMsg() receives and decodes a length-prefixed proto message from a net.Conn
// returns length of message bytes
func receiveProtoMsg(conn net.Conn, ptr proto.Message, timeout ...time.Duration) (int, lib.ErrorI) {
	// read the message from the wire
	msg, err := receiveLengthPrefixed(conn, timeout...)
	if err != nil {
		return 0, err
	}
	// unmarshal into proto
	if err = lib.Unmarshal(msg, ptr); err != nil {
		return 0, err
	}
	return len(msg), nil
}

// sendLengthPrefixed() sends a message that is prefix by length through a tcp connection
func sendLengthPrefixed(conn net.Conn, bz []byte, timeout ...time.Duration) lib.ErrorI {
	defer lib.TimeTrack(l, time.Now(), time.Second)
	// set the write timeout
	writeTimeout := WriteTimeout
	if len(timeout) == 1 {
		writeTimeout = timeout[0]
	}
	// create the length prefix (2 bytes, big endian)
	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(bz)))
	// set the write deadline
	if e := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); e != nil {
		return ErrFailedWrite(e)
	}
	// write the message (length prefixed)
	if _, er := conn.Write(append(lengthPrefix, bz...)); er != nil {
		return ErrFailedWrite(er)
	}
	// disable deadline
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

var l = lib.NewDefaultLogger()

// receiveLengthPrefixed() reads a length prefixed message from a tcp connection
func receiveLengthPrefixed(conn net.Conn, timeout ...time.Duration) ([]byte, lib.ErrorI) {
	// set the read timeout
	readTimeout := ReadTimeout
	if len(timeout) == 1 {
		readTimeout = timeout[0]
	}
	// set the read conn deadline
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, ErrFailedRead(err)
	}
	// read the 4-byte length prefix
	lengthBuffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuffer); err != nil {
		return nil, ErrFailedRead(err)
	}
	// determine the length of the message
	messageLength := binary.BigEndian.Uint32(lengthBuffer)
	// ensure the message size isn't larger than the allowed max packet size
	if messageLength > maxPacketSize {
		return nil, ErrMaxMessageSize()
	}
	// read the actual message bytes
	msg := make([]byte, messageLength)
	if _, err := io.ReadFull(conn, msg); err != nil {
		return nil, ErrFailedRead(err)
	}
	// disable deadline
	_ = conn.SetReadDeadline(time.Time{})
	// exit with no error
	return msg, nil
}

// split returns bytes split to size up to the lim param
func split(buf []byte, lim int) [][]byte {
	if len(buf) == 0 {
		return [][]byte{buf}
	}
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}
	return chunks
}
