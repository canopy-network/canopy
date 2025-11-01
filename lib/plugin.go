package lib

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"reflect"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

/* This file contains logic for extensible plugins that enable smart contract abstraction */

// PluginCompatibleFSM: defines the 'expected' interface that plugins utilize to read and write data from the FSM store
type PluginCompatibleFSM interface {
	// StateRead() executes a 'read request' to the state store
	StateRead(request *PluginStateReadRequest) (response PluginStateReadResponse, err ErrorI)
	// StateWrite() executes a 'write request' to the state store
	StateWrite(request *PluginStateWriteRequest) (response PluginStateWriteResponse, err ErrorI)
}

// Plugin defines the 'VM-less' extension of the Finite State Machine
type Plugin struct {
	config      *PluginConfig                         // the plugin configuration
	conn        net.Conn                              // the underlying unix sock file connection
	pending     map[uint64]chan isPluginToFSM_Payload // the outstanding requests from the FSM
	requestFSMs map[uint64]PluginCompatibleFSM        // maps request IDs to their FSM context for concurrent operations
	l           sync.Mutex                            // thread safety
	log         LoggerI                               // the logger associated with the plugin
}

// NewPlugin() creates and starts a plguin
func NewPlugin(conn net.Conn, log LoggerI) (p *Plugin) {
	// constructs the new plugin
	p = &Plugin{
		conn:        conn,
		pending:     map[uint64]chan isPluginToFSM_Payload{},
		requestFSMs: map[uint64]PluginCompatibleFSM{},
		l:           sync.Mutex{},
		log:         log,
	}
	// debug log plugin creation
	log.Debugf("Creating new plugin with connection: %s", conn.RemoteAddr())
	// begin the listening service
	go p.ListenForInbound()
	// debug log service started
	log.Debugf("Started plugin listening service for connection: %s", conn.RemoteAddr())
	// exit
	return
}

// Genesis() is the fsm calling the genesis function of the plugin
func (p *Plugin) Genesis(fsm PluginCompatibleFSM, request *PluginGenesisRequest) (*PluginGenesisResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginGenesisResponse), nil
	}
	// debug log genesis call start
	p.log.Debugf("Genesis() called with request: %+v", request)
	// debug log config info
	p.log.Debugf("Genesis() using plugin config: %+v", p.config)
	// send to the plugin and wait for a response (this will set FSM context for the request ID)
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Genesis{Genesis: request})
	if err != nil {
		p.log.Debugf("Genesis() error from sendToPluginSync: %v", err)
		return nil, err
	}
	// debug log raw response
	p.log.Debugf("Genesis() received response: %+v", response)
	// get the response
	wrapper, ok := response.(*PluginToFSM_Genesis)
	if !ok {
		p.log.Debugf("Genesis() type assertion failed, got type: %T", response)
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// debug log successful response
	p.log.Debugf("Genesis() returning response: %+v", wrapper.Genesis)
	// return the unwrapped response
	return wrapper.Genesis, nil
}

// BeginBlock() is the fsm calling the begin_block function of the plugin
func (p *Plugin) BeginBlock(fsm PluginCompatibleFSM, request *PluginBeginRequest) (*PluginBeginResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginBeginResponse), nil
	} // debug log begin block call start
	p.log.Debugf("BeginBlock() called with request: %+v", request)

	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Begin{Begin: request})
	if err != nil {
		p.log.Debugf("BeginBlock() error from sendToPluginSync: %v", err)
		return nil, err
	}
	// debug log raw response
	p.log.Debugf("BeginBlock() received response: %+v", response)
	// get the response
	wrapper, ok := response.(*PluginToFSM_Begin)
	if !ok {
		p.log.Debugf("BeginBlock() type assertion failed, got type: %T", response)
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// debug log successful response
	p.log.Debugf("BeginBlock() returning response: %+v", wrapper.Begin)
	// return the unwrapped response
	return wrapper.Begin, nil
}

// CheckTx() is the fsm calling the check_tx function of the plugin
func (p *Plugin) CheckTx(fsm PluginCompatibleFSM, request *PluginCheckRequest) (*PluginCheckResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginCheckResponse), nil
	}
	// debug log check tx call start
	p.log.Debugf("CheckTx() called with request: %+v", request)
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Check{Check: request})
	if err != nil {
		p.log.Debugf("CheckTx() error from sendToPluginSync: %v", err)
		return nil, err
	}
	// debug log raw response
	p.log.Debugf("CheckTx() received response: %+v", response)
	// get the response
	wrapper, ok := response.(*PluginToFSM_Check)
	if !ok {
		p.log.Debugf("CheckTx() type assertion failed, got type: %T", response)
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// debug log successful response
	p.log.Debugf("CheckTx() returning response: %+v", wrapper.Check)
	// return the unwrapped response
	return wrapper.Check, nil
}

// DeliverTx() is the fsm calling the deliver_tx function of the plugin
func (p *Plugin) DeliverTx(fsm PluginCompatibleFSM, request *PluginDeliverRequest) (*PluginDeliverResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginDeliverResponse), nil
	}
	// debug log deliver tx call start
	p.log.Debugf("DeliverTx() called with request: %+v", request)
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Deliver{Deliver: request})
	if err != nil {
		p.log.Debugf("DeliverTx() error from sendToPluginSync: %v", err)
		return nil, err
	}
	// debug log raw response
	p.log.Debugf("DeliverTx() received response: %+v", response)
	// get the response
	wrapper, ok := response.(*PluginToFSM_Deliver)
	if !ok {
		p.log.Debugf("DeliverTx() type assertion failed, got type: %T", response)
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// debug log successful response
	p.log.Debugf("DeliverTx() returning response: %+v", wrapper.Deliver)
	// return the unwrapped response
	return wrapper.Deliver, nil
}

// EndBlock() is the fsm calling the end_block function of the plugin
func (p *Plugin) EndBlock(fsm PluginCompatibleFSM, request *PluginEndRequest) (*PluginEndResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginEndResponse), nil
	}
	// debug log end block call start
	p.log.Debugf("EndBlock() called with request: %+v", request)
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_End{End: request})
	if err != nil {
		p.log.Debugf("EndBlock() error from sendToPluginSync: %v", err)
		return nil, err
	}
	// debug log raw response
	p.log.Debugf("EndBlock() received response: %+v", response)
	// get the response
	wrapper, ok := response.(*PluginToFSM_End)
	if !ok {
		p.log.Debugf("EndBlock() type assertion failed, got type: %T", response)
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// debug log successful response
	p.log.Debugf("EndBlock() returning response: %+v", wrapper.End)
	// return the unwrapped response
	return wrapper.End, nil
}

// SupportsTransaction() indicates if the transaction type is supported 'or not'
func (p *Plugin) SupportsTransaction(name string) bool {
	// defensive nil check
	if p == nil || p.config == nil {
		return false
	}
	// debug log transaction support check
	p.log.Debugf("SupportsTransaction() called for transaction: %s", name)
	// debug log supported transactions
	p.log.Debugf("SupportsTransaction() checking against supported transactions: %+v", p.config.SupportedTransactions)
	// return if the plugin supports the transaction
	supported := slices.Contains(p.config.SupportedTransactions, name)
	// debug log result
	p.log.Debugf("SupportsTransaction() result for %s: %t", name, supported)
	return supported
}

// ListenForInbound() routes inbound requests from the plugin
func (p *Plugin) ListenForInbound() {
	// debug log listener start
	p.log.Debug("ListenForInbound() started, waiting for messages from plugin")
	for {
		// block until a message is received
		msg := new(PluginToFSM)
		// debug log before receiving message
		p.log.Debug("ListenForInbound() waiting for next message")
		if err := p.receiveProtoMsg(msg); err != nil {
			p.log.Debugf("ListenForInbound() error receiving message: %v", err)
			log.Fatal(err.Error())
		}
		// debug log received message
		p.log.Debugf("ListenForInbound() received message ID %d, payload type: %T", msg.Id, msg.Payload)
		go func() {
			if err := func() ErrorI {
				// route the message
				switch payload := msg.Payload.(type) {
				// response to a request made by the FSM
				case *PluginToFSM_Genesis, *PluginToFSM_Begin, *PluginToFSM_Check, *PluginToFSM_Deliver, *PluginToFSM_End:
					p.log.Debugf("ListenForInbound() routing FSM response message ID %d", msg.Id)
					return p.handlePluginResponse(msg)
				// inbound requests from the plugin
				case *PluginToFSM_Config:
					p.log.Debugf("ListenForInbound() routing config message ID %d", msg.Id)
					return p.handleConfigMessage(msg)
				case *PluginToFSM_StateRead:
					p.log.Debugf("ListenForInbound() routing state read request ID %d", msg.Id)
					return p.handleStateReadRequest(msg)
				case *PluginToFSM_StateWrite:
					p.log.Debugf("ListenForInbound() routing state write request ID %d", msg.Id)
					return p.handleStateWriteRequest(msg)
				default:
					p.log.Debugf("ListenForInbound() unknown message type: %T for message ID %d", payload, msg.Id)
					return ErrInvalidPluginToFSMMessage(reflect.TypeOf(payload))
				}
			}(); err != nil {
				p.log.Debugf("ListenForInbound() error handling message ID %d: %v", msg.Id, err)
				log.Fatal(err.Error())
			}
		}()
	}
}

// HandleConfigMessage() handles an inbound configuration message
func (p *Plugin) handleConfigMessage(msg *PluginToFSM) ErrorI {
	m, ok := msg.Payload.(*PluginToFSM_Config)
	// debug log type assertion result
	p.log.Debugf("handleConfigMessage() type assertion success: %t", ok)
	// debug log config message handling start
	p.log.Debugf("handleConfigMessage() processing message ID %d", msg.Id)
	// validate the config
	if !ok || m.Config == nil || m.Config.Name == "" || m.Config.Id == 0 || m.Config.Version == 0 {
		if !ok {
			p.log.Debug("handleConfigMessage() type assertion failed")
		} else if m.Config == nil {
			p.log.Debug("handleConfigMessage() config is nil")
		} else {
			p.log.Debugf("handleConfigMessage() invalid config fields: name='%s', id=%d, version=%d", m.Config.Name, m.Config.Id, m.Config.Version)
		}
		return ErrInvalidPluginConfig()
	}
	// debug log received config
	p.log.Debugf("handleConfigMessage() received valid config: %+v", m.Config)
	// set config
	p.config = m.Config
	// debug log config set
	p.log.Debug("handleConfigMessage() plugin config updated successfully")
	// ack the config - send FSMToPlugin config response
	response := &FSMToPlugin{
		Id:      msg.Id,
		Payload: &FSMToPlugin_Config{Config: &PluginFSMConfig{Config: m.Config}},
	}
	// debug log sending response
	p.log.Debugf("handleConfigMessage() sending config acknowledgment for message ID %d", msg.Id)
	err := p.sendProtoMsg(response)
	if err != nil {
		p.log.Debugf("handleConfigMessage() error sending config response: %v", err)
	} else {
		p.log.Debug("handleConfigMessage() config acknowledgment sent successfully")
	}
	return err
}

// HandleStateReadRequest() handles an inbound state read request from a specific FSM context
func (p *Plugin) handleStateReadRequest(msg *PluginToFSM) ErrorI {
	// debug log state read request start
	p.log.Debugf("handleStateReadRequest() processing message ID %d", msg.Id)
	// get the FSM context for this request ID
	p.l.Lock()
	fsm := p.requestFSMs[msg.Id]
	// debug log FSM lookup
	p.log.Debugf("handleStateReadRequest() FSM lookup for ID %d: found=%t, total_fsms=%d", msg.Id, fsm != nil, len(p.requestFSMs))
	p.l.Unlock()
	// debug log request details
	request := msg.GetStateRead()
	p.log.Debugf("handleStateReadRequest() state read request: %+v", request)
	// check if FSM context exists
	if fsm == nil {
		p.log.Debugf("handleStateReadRequest() no FSM context found for request ID %d", msg.Id)
		return ErrInvalidPluginRespId()
	}
	// forward request to the appropriate FSM
	p.log.Debug("handleStateReadRequest() forwarding request to FSM")
	response, err := fsm.StateRead(request)
	if err != nil {
		p.log.Debugf("handleStateReadRequest() FSM StateRead error: %v", err)
		response.Error = NewPluginError(err)
	} else {
		p.log.Debugf("handleStateReadRequest() FSM StateRead success: %+v", response)
	}
	// send response back to FSM
	p.log.Debugf("handleStateReadRequest() sending response back for message ID %d", msg.Id)
	sendErr := p.sendProtoMsg(&FSMToPlugin{
		Id: msg.Id,
		Payload: &FSMToPlugin_StateRead{
			StateRead: &response,
		},
	})
	if sendErr != nil {
		p.log.Debugf("handleStateReadRequest() error sending response: %v", sendErr)
	} else {
		p.log.Debug("handleStateReadRequest() response sent successfully")
	}
	return sendErr
}

// HandleStateWriteRequest() handles an inbound state write request from a specific FSM context
func (p *Plugin) handleStateWriteRequest(msg *PluginToFSM) ErrorI {
	// debug log state write request start
	p.log.Debugf("handleStateWriteRequest() processing message ID %d", msg.Id)
	// get the FSM context for this request ID
	p.l.Lock()
	fsm := p.requestFSMs[msg.Id]
	// debug log FSM lookup
	p.log.Debugf("handleStateWriteRequest() FSM lookup for ID %d: found=%t, total_fsms=%d", msg.Id, fsm != nil, len(p.requestFSMs))
	p.l.Unlock()
	// debug log request details
	request := msg.GetStateWrite()
	p.log.Debugf("handleStateWriteRequest() state write request: %+v", request)
	// check if FSM context exists
	if fsm == nil {
		p.log.Debugf("handleStateWriteRequest() no FSM context found for request ID %d", msg.Id)
		return ErrInvalidPluginRespId()
	}
	// forward request to the appropriate FSM
	p.log.Debug("handleStateWriteRequest() forwarding request to FSM")
	response, err := fsm.StateWrite(request)
	if err != nil {
		p.log.Debugf("handleStateWriteRequest() FSM StateWrite error: %v", err)
		response.Error = NewPluginError(err)
	} else {
		p.log.Debugf("handleStateWriteRequest() FSM StateWrite success: %+v", response)
	}
	// send response back to FSM
	p.log.Debugf("handleStateWriteRequest() sending response back for message ID %d", msg.Id)
	sendErr := p.sendProtoMsg(&FSMToPlugin{
		Id: msg.Id,
		Payload: &FSMToPlugin_StateWrite{
			StateWrite: &response,
		},
	})
	if sendErr != nil {
		p.log.Debugf("handleStateWriteRequest() error sending response: %v", sendErr)
	} else {
		p.log.Debug("handleStateWriteRequest() response sent successfully")
	}
	return sendErr
}

// HandlePluginResponse() routes the inbound response appropriately
func (p *Plugin) handlePluginResponse(msg *PluginToFSM) ErrorI {
	// debug log plugin response handling start
	p.log.Debugf("handlePluginResponse() processing response for message ID %d, payload type: %T", msg.Id, msg.Payload)
	// thread safety
	p.l.Lock()
	defer p.l.Unlock()
	// debug log current pending requests
	p.log.Debugf("handlePluginResponse() current pending requests: %d, request FSMs: %d", len(p.pending), len(p.requestFSMs))
	// get the requester channel
	ch, ok := p.pending[msg.Id]
	if !ok {
		p.log.Debugf("handlePluginResponse() no pending request found for ID %d", msg.Id)
		return ErrInvalidPluginRespId()
	}
	// debug log successful channel lookup
	p.log.Debugf("handlePluginResponse() found pending channel for ID %d", msg.Id)
	// remove the message from the pending list and FSM context
	delete(p.pending, msg.Id)
	delete(p.requestFSMs, msg.Id)
	// debug log cleanup complete
	p.log.Debugf("handlePluginResponse() cleaned up request ID %d, remaining pending: %d", msg.Id, len(p.pending))
	// forward the message to the requester
	go func() {
		p.log.Debugf("handlePluginResponse() forwarding response payload to waiting channel for ID %d", msg.Id)
		ch <- msg.Payload
	}()
	// exit without error
	return nil
}

// sendToPluginSync() sends to the plugin and waits for a response, tracking FSM context
func (p *Plugin) sendToPluginSync(fsm PluginCompatibleFSM, request isFSMToPlugin_Payload) (isPluginToFSM_Payload, ErrorI) {
	// debug log sync send start
	p.log.Debugf("sendToPluginSync() starting sync send with request type: %T", request)
	// send to the plugin
	ch, requestId, err := p.sendToPluginAsync(fsm, request)
	if err != nil {
		p.log.Debugf("sendToPluginSync() error from sendToPluginAsync: %v", err)
		return nil, err
	}
	// debug log async send success
	p.log.Debugf("sendToPluginSync() async send successful, waiting for response with ID %d", requestId)
	// wait for the response
	response, err := p.waitForResponse(ch, requestId)
	if err != nil {
		p.log.Debugf("sendToPluginSync() error waiting for response ID %d: %v", requestId, err)
	} else {
		p.log.Debugf("sendToPluginSync() received response for ID %d: %T", requestId, response)
	}
	// clean up FSM context after operation completes
	p.l.Lock()
	delete(p.requestFSMs, requestId)
	p.l.Unlock()
	// debug log cleanup
	p.log.Debugf("sendToPluginSync() cleaned up FSM context for request ID %d", requestId)
	return response, err
}

// sendToPluginAsync() sends to the plugin but doesn't wait for a response, tracking FSM context
func (p *Plugin) sendToPluginAsync(fsm PluginCompatibleFSM, request isFSMToPlugin_Payload) (ch chan isPluginToFSM_Payload, requestId uint64, err ErrorI) {
	// generate the request UUID
	requestId = rand.Uint64()
	// debug log request ID generation
	p.log.Debugf("sendToPluginAsync() generated request ID %d for request type: %T", requestId, request)
	// make a channel to receive the response
	ch = make(chan isPluginToFSM_Payload, 1)
	// add to the pending list and FSM context map
	p.l.Lock()
	p.pending[requestId] = ch
	p.requestFSMs[requestId] = fsm // Track FSM for this request
	// debug log tracking info
	p.log.Debugf("sendToPluginAsync() tracking request ID %d, total pending: %d, total FSMs: %d", requestId, len(p.pending), len(p.requestFSMs))
	p.l.Unlock()
	// send the payload with the request ID
	p.log.Debugf("sendToPluginAsync() sending message with ID %d to plugin", requestId)
	err = p.sendProtoMsg(&FSMToPlugin{Id: requestId, Payload: request})
	if err != nil {
		p.log.Debugf("sendToPluginAsync() error sending message ID %d: %v", requestId, err)
		// clean up on error
		p.l.Lock()
		delete(p.pending, requestId)
		delete(p.requestFSMs, requestId)
		p.l.Unlock()
	} else {
		p.log.Debugf("sendToPluginAsync() message ID %d sent successfully", requestId)
	}
	// exit
	return
}

// waitForResponse() waits for a response from the plugin given a specific pending channel and request ID
func (p *Plugin) waitForResponse(ch chan isPluginToFSM_Payload, requestId uint64) (isPluginToFSM_Payload, ErrorI) {
	// debug log wait start
	p.log.Debugf("waitForResponse() waiting for response to request ID %d", requestId)
	select {
	// received response
	case response := <-ch:
		p.log.Debugf("waitForResponse() received response for request ID %d: %T", requestId, response)
		return response, nil
	// timeout
	case <-time.After(time.Second):
		p.log.Debugf("waitForResponse() timeout waiting for response to request ID %d", requestId)
		// safely remove the request and FSM context
		p.l.Lock()
		delete(p.pending, requestId)
		delete(p.requestFSMs, requestId)
		// debug log cleanup after timeout
		p.log.Debugf("waitForResponse() cleaned up timed out request ID %d, remaining pending: %d", requestId, len(p.pending))
		p.l.Unlock()
		// exit with timeout error
		return nil, ErrPluginTimeout()
	}
}

// sendProtoMsg() encodes and sends a length-prefixed proto message to a net.Conn
func (p *Plugin) sendProtoMsg(ptr proto.Message) ErrorI {
	// debug log proto message send start
	p.log.Debugf("sendProtoMsg() sending proto message type: %T", ptr)
	// marshal into proto bytes
	bz, err := Marshal(ptr)
	if err != nil {
		p.log.Debugf("sendProtoMsg() marshal error: %v", err)
		return err
	}
	// debug log marshal success
	p.log.Debugf("sendProtoMsg() marshaled message, size: %d bytes", len(bz))
	// send the bytes prefixed by length
	sendErr := p.sendLengthPrefixed(bz)
	if sendErr != nil {
		p.log.Debugf("sendProtoMsg() send error: %v", sendErr)
	} else {
		p.log.Debug("sendProtoMsg() message sent successfully")
	}
	return sendErr
}

// receiveProtoMsg() receives and decodes a length-prefixed proto message from a net.Conn
func (p *Plugin) receiveProtoMsg(ptr proto.Message) ErrorI {
	// debug log proto message receive start
	p.log.Debug("receiveProtoMsg() waiting to receive proto message")
	// read the message from the wire
	msg, err := p.receiveLengthPrefixed()
	if err != nil {
		p.log.Debugf("receiveProtoMsg() receive error: %v", err)
		return err
	}
	// debug log successful receive
	p.log.Debugf("receiveProtoMsg() received message, size: %d bytes", len(msg))
	// unmarshal into proto
	if err = Unmarshal(msg, ptr); err != nil {
		p.log.Debugf("receiveProtoMsg() unmarshal error: %v", err)
		return err
	}
	// debug log successful unmarshal
	p.log.Debugf("receiveProtoMsg() successfully unmarshaled into type: %T", ptr)
	return nil
}

// sendLengthPrefixed() sends a message that is prefix by length through a tcp connection
func (p *Plugin) sendLengthPrefixed(bz []byte) ErrorI {
	// debug log length prefixed send start
	p.log.Debugf("sendLengthPrefixed() sending %d bytes", len(bz))
	// create the length prefix (4 bytes, big endian)
	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(bz)))
	// debug log length prefix
	p.log.Debugf("sendLengthPrefixed() length prefix: %d (0x%08x)", uint32(len(bz)), uint32(len(bz)))
	// write the message (length prefixed)
	totalBytes := append(lengthPrefix, bz...)
	p.log.Debugf("sendLengthPrefixed() writing total %d bytes (4 prefix + %d data)", len(totalBytes), len(bz))
	if _, er := p.conn.Write(totalBytes); er != nil {
		p.log.Debugf("sendLengthPrefixed() write error: %v", er)
		return ErrFailedPluginWrite(er)
	}
	// debug log successful write
	p.log.Debugf("sendLengthPrefixed() successfully wrote %d bytes to connection", len(totalBytes))
	return nil
}

// receiveLengthPrefixed() reads a length prefixed message from a tcp connection
func (p *Plugin) receiveLengthPrefixed() ([]byte, ErrorI) {
	// debug log receive start
	p.log.Debug("receiveLengthPrefixed() reading length prefix")
	// read the 4-byte length prefix
	lengthBuffer := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lengthBuffer); err != nil {
		p.log.Debugf("receiveLengthPrefixed() error reading length prefix: %v", err)
		return nil, ErrFailedPluginRead(err)
	}
	// determine the length of the message
	messageLength := binary.BigEndian.Uint32(lengthBuffer)
	// debug log message length
	p.log.Debugf("receiveLengthPrefixed() message length: %d (0x%08x)", messageLength, messageLength)
	// validate message length
	if messageLength > 1024*1024 { // 1MB safety limit
		p.log.Debugf("receiveLengthPrefixed() message length too large: %d bytes", messageLength)
		return nil, ErrFailedPluginRead(fmt.Errorf("message too large: %d bytes", messageLength))
	}
	// read the actual message bytes
	msg := make([]byte, messageLength)
	p.log.Debugf("receiveLengthPrefixed() reading message data (%d bytes)", messageLength)
	if _, err := io.ReadFull(p.conn, msg); err != nil {
		p.log.Debugf("receiveLengthPrefixed() error reading message data: %v", err)
		return nil, ErrFailedPluginRead(err)
	}
	// debug log successful receive
	p.log.Debugf("receiveLengthPrefixed() successfully read %d bytes", len(msg))
	// exit with no error
	return msg, nil
}

// E() converts a plugin error to the ErrorI interface
func (x *PluginError) E() ErrorI {
	if x == nil {
		return nil
	}
	return &Error{
		ECode:   ErrorCode(x.Code),
		EModule: ErrorModule(x.Module),
		Msg:     x.Msg,
	}
}

// NewPluginError() creates a plugin error from an ErrorI
func NewPluginError(err ErrorI) *PluginError {
	return &PluginError{
		Code:   uint64(err.Code()),
		Module: string(err.Module()),
		Msg:    err.Message(),
	}
}
