package contract

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

/* This file contains boilerplate logic to interact with the Canopy FSM via socket file */

// Plugin defines the 'VM-less' extension of the Finite State Machine
type Plugin struct {
	fsmConfig       *PluginFSMConfig                      // the FSM configuration
	pluginConfig    *PluginConfig                         // the plugin configuration
	conn            net.Conn                              // the underlying unix sock file connection
	pending         map[uint64]chan isFSMToPlugin_Payload // the outstanding requests from the contract
	requestContract map[uint64]*Contract                  // maps request IDs to their contract context for concurrent operations
	l               sync.Mutex                            // thread safety
	config          Config                                // general app config
}

// socketPath is the name of the plugin socket exposed by the base SDK
const socketPath = "plugin.sock"

// StartPlugin() creates and starts a plguin
func StartPlugin(c Config) {
	var conn net.Conn
	// connect to the socket
	sockPath := filepath.Join(c.DataDirPath, socketPath)
	// connect to the existing Unix socket
	for range time.Tick(time.Second) {
		var err error
		// connect to the unix socket
		conn, err = net.Dial("unix", sockPath)
		// exit if no error
		if err == nil {
			break
		}
		// log the connection error
		log.Printf("Failed to connect to plugin socket%v\n", err)
	}
	// constructs the new plugin
	p := &Plugin{
		pluginConfig:    ContractConfig,
		conn:            conn,
		pending:         map[uint64]chan isFSMToPlugin_Payload{},
		requestContract: map[uint64]*Contract{},
		l:               sync.Mutex{},
		config:          c,
	}
	// begin the listening service
	go p.ListenForInbound()
	// execute the handshake
	if err := p.Handshake(); err != nil {
		log.Fatal(err.Error())
	}
	// exit
	return
}

// Handshake() sends the contract configuration to the FSM and awaits a reply
func (p *Plugin) Handshake() *PluginError {
	// log the handshake
	log.Println("Handshaking with FSM")
	// send to the plugin sync
	response, err := p.sendToPluginSync(&Contract{}, &PluginToFSM_Config{Config: p.pluginConfig})
	if err != nil {
		return err
	}
	// get the response
	wrapper, ok := response.(*FSMToPlugin_Config)
	if !ok {
		return ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	// set pluginConfig
	p.fsmConfig = wrapper.Config
	// handshake conmplete
	return nil
}

// Genesis() is the fsm calling the genesis function of the plugin
func (p *Plugin) StateRead(c *Contract, request *PluginStateReadRequest) (*PluginStateReadResponse, *PluginError) {
	// send to the plugin and wait for a response (this will set FSM context for the request ID)
	response, err := p.sendToPluginSync(c, &PluginToFSM_StateRead{StateRead: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*FSMToPlugin_StateRead)
	if !ok {
		return nil, ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.StateRead, nil
}

func (p *Plugin) StateWrite(c *Contract, request *PluginStateWriteRequest) (*PluginStateWriteResponse, *PluginError) {
	// send to the plugin and wait for a response (this will set FSM context for the request ID)
	response, err := p.sendToPluginSync(c, &PluginToFSM_StateWrite{StateWrite: request})
	if err != nil {
		return nil, err
	}
	// get the response
	wrapper, ok := response.(*FSMToPlugin_StateWrite)
	if !ok {
		return nil, ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	// return the unwrapped response
	return wrapper.StateWrite, nil
}

// ListenForInbound() routes inbound requests from the plugin
func (p *Plugin) ListenForInbound() {
	for {
		// block until a message is received
		msg := new(FSMToPlugin)
		if err := p.receiveProtoMsg(msg); err != nil {
			log.Fatal(err.Error())
		}
		go func() {
			if err := func() *PluginError {
				// create a new instance of a contract
				response, c := isPluginToFSM_Payload(nil), &Contract{Config: p.config, FSMConfig: p.fsmConfig, plugin: p, fsmId: msg.Id}
				// route the message
				switch payload := msg.Payload.(type) {
				// response to a request made by the Contract
				case *FSMToPlugin_Config, *FSMToPlugin_StateRead, *FSMToPlugin_StateWrite:
					log.Println("Received FSM response")
					return p.handleFSMResponse(msg)
				// inbound requests from the FSM
				case *FSMToPlugin_Genesis:
					log.Println("Received genesis request from FSM")
					response = &PluginToFSM_Genesis{c.Genesis(msg.GetGenesis())}
				case *FSMToPlugin_Begin:
					log.Println("Received begin request from FSM")
					response = &PluginToFSM_Begin{c.BeginBlock(msg.GetBegin())}
				case *FSMToPlugin_Check:
					log.Println("Received check request from FSM")
					response = &PluginToFSM_Check{c.CheckTx(msg.GetCheck())}
				case *FSMToPlugin_Deliver:
					log.Println("Received deliver request from FSM")
					response = &PluginToFSM_Deliver{c.DeliverTx(msg.GetDeliver())}
				case *FSMToPlugin_End:
					log.Println("PABLITO WAS HERE")
					log.Println("EL AYUWOKI WAS HERE")
					log.Println("Received end request from FSM")
					response = &PluginToFSM_End{c.EndBlock(msg.GetEnd())}
				default:
					return ErrInvalidFSMToPluginMMessage(reflect.TypeOf(payload))
				}
				return p.sendProtoMsg(&PluginToFSM{
					Id:      msg.Id,
					Payload: response,
				})
			}(); err != nil {
				log.Fatal(err.Error())
			}
		}()
	}
}

// HandlePluginResponse() routes the inbound response appropriately
func (p *Plugin) handleFSMResponse(msg *FSMToPlugin) *PluginError {
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
	delete(p.requestContract, msg.Id)
	// forward the message to the requester
	go func() { ch <- msg.Payload }()
	// exit without error
	return nil
}

// sendToPluginSync() sends to the plugin and waits for a response, tracking FSM context
func (p *Plugin) sendToPluginSync(c *Contract, request isPluginToFSM_Payload) (isFSMToPlugin_Payload, *PluginError) {
	// send to the plugin
	ch, requestId, err := p.sendToPluginAsync(c, request)
	if err != nil {
		return nil, err
	}
	// wait for the response
	response, err := p.waitForResponse(ch, requestId)
	// clean up FSM context after operation completes
	p.l.Lock()
	delete(p.requestContract, requestId)
	p.l.Unlock()
	return response, err
}

// sendToPluginAsync() sends to the plugin but doesn't wait for a response, tracking FSM context
func (p *Plugin) sendToPluginAsync(c *Contract, request isPluginToFSM_Payload) (ch chan isFSMToPlugin_Payload, requestId uint64, err *PluginError) {
	// generate the request UUID
	requestId = c.fsmId
	// make a channel to receive the response
	ch = make(chan isFSMToPlugin_Payload, 1)
	// add to the pending list and FSM context map
	p.l.Lock()
	p.pending[requestId] = ch
	p.requestContract[requestId] = c // Track contract for this request
	p.l.Unlock()
	// send the payload with the request ID
	err = p.sendProtoMsg(&PluginToFSM{Id: requestId, Payload: request})
	// exit
	return
}

// waitForResponse() waits for a response from the plugin given a specific pending channel and request ID
func (p *Plugin) waitForResponse(ch chan isFSMToPlugin_Payload, requestId uint64) (isFSMToPlugin_Payload, *PluginError) {
	select {
	// received response
	case response := <-ch:
		return response, nil
	// timeout
	case <-time.After(10 * time.Second):
		// safely remove the request and FSM context
		p.l.Lock()
		delete(p.pending, requestId)
		delete(p.requestContract, requestId)
		p.l.Unlock()
		// exit with timeout error
		return nil, ErrPluginTimeout()
	}
}

// sendProtoMsg() encodes and sends a length-prefixed proto message to a net.Conn
func (p *Plugin) sendProtoMsg(ptr proto.Message) *PluginError {
	// marshal into proto bytes
	bz, err := Marshal(ptr)
	if err != nil {
		return err
	}
	// send the bytes prefixed by length
	return p.sendLengthPrefixed(bz)
}

// receiveProtoMsg() receives and decodes a length-prefixed proto message from a net.Conn
func (p *Plugin) receiveProtoMsg(ptr proto.Message) *PluginError {
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
func (p *Plugin) sendLengthPrefixed(bz []byte) *PluginError {
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
func (p *Plugin) receiveLengthPrefixed() ([]byte, *PluginError) {
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

// CODEC IMPLEMENTATION BELOW

// marshaller() defines a protobuf marshaller
var marshaller = proto.MarshalOptions{Deterministic: true}

// Marshal() serializes a proto.Message into a byte slice
func Marshal(message any) ([]byte, *PluginError) {
	// convert the message into proto bytes using the proto marshaller
	protoBytes, err := marshaller.Marshal(message.(proto.Message))
	// if an error occurred during the conversion process
	if err != nil {
		// exit with a wrapped error
		return nil, ErrMarshal(err)
	}
	// exit
	return protoBytes, nil
}

// Unmarshal() deserializes a byte slice into a proto.Message
func Unmarshal(protoBytes []byte, ptr any) *PluginError {
	// if protoBytes are empty or ptr is nil
	if protoBytes == nil || ptr == nil {
		// return with no error
		return nil
	}
	// populate the ptr with the proto bytes
	if err := proto.Unmarshal(protoBytes, ptr.(proto.Message)); err != nil {
		// exit with wrapped error
		return ErrUnmarshal(err)
	}
	// exit
	return nil
}

// FromAny() converts an anypb.Any type back into a proto.Message
func FromAny(any *anypb.Any) (proto.Message, *PluginError) {
	// convert the proto any into a proto message
	msg, err := anypb.UnmarshalNew(any, proto.UnmarshalOptions{})
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, ErrFromAny(err)
	}
	// exit with the proto message
	return msg, nil
}

// JoinLenPrefix() appends the items together separated by a single byte to represent the length of the segment
func JoinLenPrefix(toAppend ...[]byte) []byte {
	// calculate total length first
	totalLen := 0
	for _, item := range toAppend {
		// if the item isn't empty, calculate the size
		if item != nil {
			// 1 byte for length + item length
			totalLen += 1 + len(item)
		}
	}
	// make the proper size buffer
	res := make([]byte, 0, totalLen)
	// iterate through each 'segment' and append it
	for _, item := range toAppend {
		// if item is empty, skip
		if item == nil {
			continue
		}
		// length
		res = append(res, byte(len(item)))
		// item
		res = append(res, item...)
	}
	// return the result
	return res
}

// CONFIG IMPLEMENTATION BELOW

type Config struct {
	ChainId     uint64 `json:"chainId"`
	DataDirPath string `json:"dataDirPath"`
}

// DefaultConfig() returns the default configuration
func DefaultConfig() Config {
	// return the default configuration
	return Config{
		ChainId:     1,
		DataDirPath: filepath.Join("/tmp/plugin/"),
	}
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
	// define the default pluginConfig to fill in any blanks in the file
	c := DefaultConfig()
	// populate the default pluginConfig with the file bytes
	if err = json.Unmarshal(fileBytes, &c); err != nil {
		// exit with error
		return Config{}, err
	}
	// exit
	return c, nil
}
