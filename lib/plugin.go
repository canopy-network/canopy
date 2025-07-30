package lib

import (
	"encoding/binary"
	"google.golang.org/protobuf/proto"
	"io"
	"math/rand"
	"net"
	"reflect"
	"slices"
	"sync"
	"time"
)

/* This file contains logic for extensible plugins that enable smart contract abstraction */

// PluginCompatibleFSM: defines the 'expected' interface that plugins utilize to read and write data from the FSM store
type PluginCompatibleFSM interface {
	// StateRead() executes a 'read request' to the state store
	StateRead(request PluginStateReadRequest) (response PluginStateReadResponse, err ErrorI)
	// StateWrite() executes a 'write request' to the state store
	StateWrite(request PluginStateWriteRequest) (response PluginStateWriteResponse, err ErrorI)
}

// Plugin defines the 'VM-less' extension of the Finite State Machine
type Plugin struct {
	config      *PluginConfig                         // the plugin configuration
	conn        net.Conn                              // the underlying unix sock file connection
	pending     map[uint64]chan isPluginToFSM_Payload // the outstanding requests that
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
	// begin the listening service
	go func() {
		if err := p.ListenForInbound(); err != nil {
			p.log.Fatal(err.Error())
		}
	}()
	// exit
	return
}

// Genesis() is the fsm calling the genesis function of the plugin
func (p *Plugin) Genesis(fsm PluginCompatibleFSM, request *PluginGenesisRequest) (*PluginGenesisResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginGenesisResponse), nil
	}
	// send to the plugin and wait for a response (this will set FSM context for the request ID)
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Genesis{Genesis: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*PluginToFSM_Genesis)
	if !ok {
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.Genesis, nil
}

// BeginBlock() is the fsm calling the begin_block function of the plugin
func (p *Plugin) BeginBlock(fsm PluginCompatibleFSM, request *PluginBeginRequest) (*PluginBeginResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginBeginResponse), nil
	}
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Begin{Begin: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*PluginToFSM_Begin)
	if !ok {
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.Begin, nil
}

// CheckTx() is the fsm calling the check_tx function of the plugin
func (p *Plugin) CheckTx(fsm PluginCompatibleFSM, request *PluginCheckRequest) (*PluginCheckResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginCheckResponse), nil
	}
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Check{Check: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*PluginToFSM_Check)
	if !ok {
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.Check, nil
}

// DeliverTx() is the fsm calling the deliver_tx function of the plugin
func (p *Plugin) DeliverTx(fsm PluginCompatibleFSM, request *PluginDeliverRequest) (*PluginDeliverResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginDeliverResponse), nil
	}
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_Deliver{Deliver: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*PluginToFSM_Deliver)
	if !ok {
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.Deliver, nil
}

// EndBlock() is the fsm calling the end_block function of the plugin
func (p *Plugin) EndBlock(fsm PluginCompatibleFSM, request *PluginEndRequest) (*PluginEndResponse, ErrorI) {
	// defensive nil check
	if p == nil || p.config == nil {
		return new(PluginEndResponse), nil
	}
	// send to the plugin and wait for a response
	response, err := p.sendToPluginSync(fsm, &FSMToPlugin_End{End: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*PluginToFSM_End)
	if !ok {
		return nil, ErrUnexpectedPluginToFSM(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.End, nil
}

// SupportsTransaction() indicates if the transaction type is supported 'or not'
func (p *Plugin) SupportsTransaction(name string) bool {
	// defensive nil check
	if p == nil || p.config == nil {
		return false
	}
	// return if the plugin supports the transaction
	return slices.Contains(p.config.SupportedTransactions, name)
}

// ListenForInbound() routes inbound requests from the plugin
func (p *Plugin) ListenForInbound() ErrorI {
	for {
		// block until a message is received
		msg := new(PluginToFSM)
		if err := p.receiveProtoMsg(msg); err != nil {
			return err
		}
		// route the message
		switch payload := msg.Payload.(type) {
		// response to a request made by the FSM
		case *PluginToFSM_Genesis, *PluginToFSM_Begin, *PluginToFSM_Deliver, *PluginToFSM_End:
			return p.handlePluginResponse(msg)
		// inbound requests from the plugin
		case *PluginToFSM_Config:
			return p.handleConfigMessage(payload.Config)
		case *PluginToFSM_StateRead:
			return p.handleStateReadRequest(msg)
		case *PluginToFSM_StateWrite:
			return p.handleStateWriteRequest(msg)
		default:
			return ErrInvalidPluginToFSMMessage(reflect.TypeOf(payload))
		}
	}
}

// HandleConfigMessage() handles an inbound configuration message
func (p *Plugin) handleConfigMessage(config *PluginConfig) ErrorI {
	// validate the config
	if config == nil || config.Name == "" || config.Id == 0 || config.Version == 0 {
		return ErrInvalidPluginConfig()
	}
	// set config
	p.config = config
	// ack the config - send FSMToPlugin config response
	response := &FSMToPlugin{
		Payload: &FSMToPlugin_Config{Config: &PluginFSMConfig{}},
	}
	return p.sendProtoMsg(response)
}

// HandleStateReadRequest() handles an inbound state read request from a specific FSM context
func (p *Plugin) handleStateReadRequest(msg *PluginToFSM) ErrorI {
	// get the FSM context for this request ID
	p.l.Lock()
	fsm := p.requestFSMs[msg.Id]
	p.l.Unlock()
	// forward request to the appropriate FSM
	response, err := fsm.StateRead(*msg.GetStateRead())
	if err != nil {
		response.Error = NewPluginError(err)
	}
	// send response back to FSM
	return p.sendProtoMsg(&FSMToPlugin{
		Id: msg.Id,
		Payload: &FSMToPlugin_StateRead{
			StateRead: &response,
		},
	})
}

// HandleStateWriteRequest() handles an inbound state write request from a specific FSM context
func (p *Plugin) handleStateWriteRequest(msg *PluginToFSM) ErrorI {
	// get the FSM context for this request ID
	p.l.Lock()
	fsm := p.requestFSMs[msg.Id]
	p.l.Unlock()
	// forward request to the appropriate FSM
	response, err := fsm.StateWrite(*msg.GetStateWrite())
	if err != nil {
		response.Error = NewPluginError(err)
	}
	// send response back to FSM
	return p.sendProtoMsg(&FSMToPlugin{
		Id: msg.Id,
		Payload: &FSMToPlugin_StateWrite{
			StateWrite: &response,
		},
	})
}

// HandlePluginResponse() routes the inbound response appropriately
func (p *Plugin) handlePluginResponse(msg *PluginToFSM) ErrorI {
	// thread safety
	p.l.Lock()
	defer p.l.Unlock()
	// get the requester channel
	ch, ok := p.pending[msg.Id]
	if !ok {
		return ErrInvalidPluginRespId()
	}
	// remove the message from the pending list and FSM context
	delete(p.pending, msg.Id)
	delete(p.requestFSMs, msg.Id)
	// forward the message to the requester
	go func() { ch <- msg.Payload }()
	// exit without error
	return nil
}

// sendToPluginSync() sends to the plugin and waits for a response, tracking FSM context
func (p *Plugin) sendToPluginSync(fsm PluginCompatibleFSM, request isFSMToPlugin_Payload) (isPluginToFSM_Payload, ErrorI) {
	// send to the plugin
	ch, requestId, err := p.sendToPluginAsync(fsm, request)
	if err != nil {
		return nil, err
	}
	// wait for the response
	response, err := p.waitForResponse(ch, requestId)
	// clean up FSM context after operation completes
	p.l.Lock()
	delete(p.requestFSMs, requestId)
	p.l.Unlock()
	return response, err
}

// sendToPluginAsync() sends to the plugin but doesn't wait for a response, tracking FSM context
func (p *Plugin) sendToPluginAsync(fsm PluginCompatibleFSM, request isFSMToPlugin_Payload) (ch chan isPluginToFSM_Payload, requestId uint64, err ErrorI) {
	// generate the request UUID
	requestId = rand.Uint64()
	// make a channel to receive the response
	ch = make(chan isPluginToFSM_Payload, 1)
	// add to the pending list and FSM context map
	p.l.Lock()
	p.pending[requestId] = ch
	p.requestFSMs[requestId] = fsm // Track FSM for this request
	p.l.Unlock()
	// send the payload with the request ID
	err = p.sendProtoMsg(&FSMToPlugin{Id: requestId, Payload: request})
	// exit
	return
}

// waitForResponse() waits for a response from the plugin given a specific pending channel and request ID
func (p *Plugin) waitForResponse(ch chan isPluginToFSM_Payload, requestId uint64) (isPluginToFSM_Payload, ErrorI) {
	select {
	// received response
	case response := <-ch:
		return response, nil
	// timeout
	case <-time.After(time.Second):
		// safely remove the request and FSM context
		p.l.Lock()
		delete(p.pending, requestId)
		delete(p.requestFSMs, requestId)
		p.l.Unlock()
		// exit with timeout error
		return nil, ErrPluginTimeout()
	}
}

// sendProtoMsg() encodes and sends a length-prefixed proto message to a net.Conn
func (p *Plugin) sendProtoMsg(ptr proto.Message) ErrorI {
	// marshal into proto bytes
	bz, err := Marshal(ptr)
	if err != nil {
		return err
	}
	// send the bytes prefixed by length
	return p.sendLengthPrefixed(bz)
}

// receiveProtoMsg() receives and decodes a length-prefixed proto message from a net.Conn
func (p *Plugin) receiveProtoMsg(ptr proto.Message) ErrorI {
	// read the message from the wire
	msg, err := p.receiveLengthPrefixed()
	if err != nil {
		return err
	}
	// unmarshal into proto
	if err = Unmarshal(msg, ptr); err != nil {
		return err
	}
	return nil
}

// sendLengthPrefixed() sends a message that is prefix by length through a tcp connection
func (p *Plugin) sendLengthPrefixed(bz []byte) ErrorI {
	// create the length prefix (4 bytes, big endian)
	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(bz)))
	// write the message (length prefixed)
	if _, er := p.conn.Write(append(lengthPrefix, bz...)); er != nil {
		return ErrFailedPluginWrite(er)
	}
	return nil
}

// receiveLengthPrefixed() reads a length prefixed message from a tcp connection
func (p *Plugin) receiveLengthPrefixed() ([]byte, ErrorI) {
	// read the 4-byte length prefix
	lengthBuffer := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lengthBuffer); err != nil {
		return nil, ErrFailedPluginRead(err)
	}
	// determine the length of the message
	messageLength := binary.BigEndian.Uint32(lengthBuffer)
	// read the actual message bytes
	msg := make([]byte, messageLength)
	if _, err := io.ReadFull(p.conn, msg); err != nil {
		return nil, ErrFailedPluginRead(err)
	}
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
