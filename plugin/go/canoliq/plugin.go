package canoliq

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/canopy-network/go-plugin/contract"
)

// Plugin is the canoLiq counterpart to contract.Plugin. It owns the FSM
// connection and routes inbound FSM lifecycle events to a per-request
// *Canoliq context, mirroring the structure of contract/plugin.go.
type Plugin struct {
	fsmConfig       *contract.PluginFSMConfig
	pluginConfig    *contract.PluginConfig
	conn            net.Conn
	pending         map[uint64]chan isFSMResponse
	requestContract map[uint64]*Canoliq
	l               sync.Mutex
	config          Config
	// height is the most recent block height observed via PluginBeginRequest
	// or PluginEndRequest. DeliverTx requests do not carry a height, so we
	// fall back to this latest known value.
	height uint64
	// fakeStore is a unit-test hook. When non-nil, StateRead/StateWrite
	// answer from the in-memory store instead of round-tripping to the
	// FSM over the unix socket. Always nil in production builds.
	fakeStore fakeStoreHook
	// rpc is the optional HTTP query server. Nil when Config.RpcAddress is
	// empty. main.go calls Shutdown on it during graceful exit.
	rpc *RPCServer
	// snapshot holds the most recently published view of canoliq-owned
	// state. Refreshed inside EndBlock by Canoliq.refreshSnapshot; read
	// lock-free by HTTP query handlers via Plugin.Snapshot().
	snapshot snapshotPointer
}

// RPC returns the active HTTP query server, or nil if disabled.
func (p *Plugin) RPC() *RPCServer { return p.rpc }

// fakeStoreHook is the test-only contract for in-memory state. The concrete
// implementation lives in fakeplugin_test.go (build-tagged out of release
// binaries by being in a *_test.go file).
type fakeStoreHook interface {
	read(*contract.PluginStateReadRequest) *contract.PluginStateReadResponse
	write(*contract.PluginStateWriteRequest) *contract.PluginStateWriteResponse
}

// CurrentHeight returns the latest block height observed by the plugin.
func (p *Plugin) CurrentHeight() uint64 {
	p.l.Lock()
	defer p.l.Unlock()
	return p.height
}

func (p *Plugin) setHeight(h uint64) {
	p.l.Lock()
	defer p.l.Unlock()
	if h > p.height {
		p.height = h
	}
}

// isFSMResponse is the (untyped) payload returned by the FSM. The contract
// package's isFSMToPlugin_Payload oneof interface is unexported, so we use
// any here and rely on type assertions to unwrap concrete variants.
type isFSMResponse = any

const socketPath = "plugin.sock"

// StartPlugin establishes the unix-socket connection to the FSM, performs
// the handshake, starts the inbound dispatch loop, and (if configured)
// brings up the read-only HTTP query server. It returns the long-lived
// *Plugin so the caller can drive graceful shutdown of the HTTP listener.
func StartPlugin(c Config) *Plugin {
	sockPath := filepath.Join(c.DataDirPath, socketPath)
	var conn net.Conn
	for range time.Tick(time.Second) {
		var err error
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
		log.Printf("canoliq: failed to connect to plugin socket: %v\n", err)
	}
	p := &Plugin{
		pluginConfig:    CanoliqConfig,
		conn:            conn,
		pending:         map[uint64]chan isFSMResponse{},
		requestContract: map[uint64]*Canoliq{},
		l:               sync.Mutex{},
		config:          c,
	}
	go p.ListenForInbound()
	if err := p.Handshake(); err != nil {
		log.Fatal(err.Error())
	}
	if c.RpcAddress != "" {
		srv, err := StartRPCServer(p, c.RpcAddress)
		if err != nil {
			log.Fatalf("canoliq: rpc start failed: %v", err)
		}
		p.rpc = srv
	}
	return p
}

// Handshake sends CanoliqConfig to the FSM and stores its response.
func (p *Plugin) Handshake() *contract.PluginError {
	log.Println("canoliq: handshaking with FSM")
	response, err := p.sendToPluginSync(&Canoliq{}, &contract.PluginToFSM_Config{Config: p.pluginConfig})
	if err != nil {
		return err
	}
	wrapper, ok := response.(*contract.FSMToPlugin_Config)
	if !ok {
		return contract.ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	p.fsmConfig = wrapper.Config
	return nil
}

// StateRead issues a state read request to the FSM and blocks until response.
func (p *Plugin) StateRead(c *Canoliq, request *contract.PluginStateReadRequest) (*contract.PluginStateReadResponse, *contract.PluginError) {
	if p.fakeStore != nil {
		return p.fakeStore.read(request), nil
	}
	response, err := p.sendToPluginSync(c, &contract.PluginToFSM_StateRead{StateRead: request})
	if err != nil {
		return nil, err
	}
	wrapper, ok := response.(*contract.FSMToPlugin_StateRead)
	if !ok {
		return nil, contract.ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	return wrapper.StateRead, nil
}

// StateWrite issues a state write request to the FSM and blocks until response.
func (p *Plugin) StateWrite(c *Canoliq, request *contract.PluginStateWriteRequest) (*contract.PluginStateWriteResponse, *contract.PluginError) {
	if p.fakeStore != nil {
		return p.fakeStore.write(request), nil
	}
	response, err := p.sendToPluginSync(c, &contract.PluginToFSM_StateWrite{StateWrite: request})
	if err != nil {
		return nil, err
	}
	wrapper, ok := response.(*contract.FSMToPlugin_StateWrite)
	if !ok {
		return nil, contract.ErrUnexpectedFSMToPlugin(reflect.TypeOf(response))
	}
	return wrapper.StateWrite, nil
}

// ListenForInbound receives FSM messages and dispatches each to a fresh
// per-request *Canoliq, just like contract.Plugin.ListenForInbound.
func (p *Plugin) ListenForInbound() {
	for {
		msg := new(contract.FSMToPlugin)
		if err := p.receiveProtoMsg(msg); err != nil {
			log.Fatal(err.Error())
		}
		go func() {
			if err := func() *contract.PluginError {
				c := &Canoliq{Config: p.config, FSMConfig: p.fsmConfig, plugin: p, fsmId: msg.Id}
				var response isPluginRequest
				switch payload := msg.Payload.(type) {
				case *contract.FSMToPlugin_Config, *contract.FSMToPlugin_StateRead, *contract.FSMToPlugin_StateWrite:
					return p.handleFSMResponse(msg)
				case *contract.FSMToPlugin_Genesis:
					log.Println("canoliq: genesis request from FSM")
					response = &contract.PluginToFSM_Genesis{Genesis: c.Genesis(msg.GetGenesis())}
				case *contract.FSMToPlugin_Begin:
					p.setHeight(msg.GetBegin().GetHeight())
					response = &contract.PluginToFSM_Begin{Begin: c.BeginBlock(msg.GetBegin())}
				case *contract.FSMToPlugin_Check:
					response = &contract.PluginToFSM_Check{Check: c.CheckTx(msg.GetCheck())}
				case *contract.FSMToPlugin_Deliver:
					response = &contract.PluginToFSM_Deliver{Deliver: c.DeliverTx(msg.GetDeliver())}
				case *contract.FSMToPlugin_End:
					p.setHeight(msg.GetEnd().GetHeight())
					response = &contract.PluginToFSM_End{End: c.EndBlock(msg.GetEnd())}
				default:
					return contract.ErrInvalidFSMToPluginMMessage(reflect.TypeOf(payload))
				}
				return p.sendPluginToFSM(msg.Id, response)
			}(); err != nil {
				log.Fatal(err.Error())
			}
		}()
	}
}

// isPluginRequest is the local alias for the PluginToFSM oneof payload set.
type isPluginRequest = any

// handleFSMResponse routes FSM responses back to the goroutine that issued
// the original request via a pending channel, keyed by FSM request id.
func (p *Plugin) handleFSMResponse(msg *contract.FSMToPlugin) *contract.PluginError {
	p.l.Lock()
	defer p.l.Unlock()
	ch, ok := p.pending[msg.Id]
	if !ok {
		return contract.ErrInvalidPluginRespId()
	}
	delete(p.pending, msg.Id)
	delete(p.requestContract, msg.Id)
	go func() { ch <- msg.Payload }()
	return nil
}

func (p *Plugin) sendToPluginSync(c *Canoliq, request any) (isFSMResponse, *contract.PluginError) {
	ch, requestId, err := p.sendToPluginAsync(c, request)
	if err != nil {
		return nil, err
	}
	response, err := p.waitForResponse(ch, requestId)
	p.l.Lock()
	delete(p.requestContract, requestId)
	p.l.Unlock()
	return response, err
}

func (p *Plugin) sendToPluginAsync(c *Canoliq, request any) (chan isFSMResponse, uint64, *contract.PluginError) {
	requestId := c.fsmId
	ch := make(chan isFSMResponse, 1)
	p.l.Lock()
	p.pending[requestId] = ch
	p.requestContract[requestId] = c
	p.l.Unlock()
	err := p.sendPluginToFSM(requestId, request)
	return ch, requestId, err
}

// sendPluginToFSM marshals a PluginToFSM oneof payload via the contract
// package's helper to bypass the unexported oneof interface boundary.
func (p *Plugin) sendPluginToFSM(id uint64, payload any) *contract.PluginError {
	msg, err := contract.NewPluginToFSM(id, payload)
	if err != nil {
		return err
	}
	return p.sendProtoMsg(msg)
}

func (p *Plugin) waitForResponse(ch chan isFSMResponse, requestId uint64) (isFSMResponse, *contract.PluginError) {
	select {
	case response := <-ch:
		return response, nil
	case <-time.After(10 * time.Second):
		p.l.Lock()
		delete(p.pending, requestId)
		delete(p.requestContract, requestId)
		p.l.Unlock()
		return nil, contract.ErrPluginTimeout()
	}
}

func (p *Plugin) sendProtoMsg(ptr interface{}) *contract.PluginError {
	bz, err := contract.Marshal(ptr)
	if err != nil {
		return err
	}
	return p.sendLengthPrefixed(bz)
}

func (p *Plugin) receiveProtoMsg(ptr interface{}) *contract.PluginError {
	msg, err := p.receiveLengthPrefixed()
	if err != nil {
		return err
	}
	return contract.Unmarshal(msg, ptr)
}

func (p *Plugin) sendLengthPrefixed(bz []byte) *contract.PluginError {
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, uint32(len(bz)))
	if _, er := p.conn.Write(append(prefix, bz...)); er != nil {
		return contract.ErrFailedPluginWrite(er)
	}
	return nil
}

func (p *Plugin) receiveLengthPrefixed() ([]byte, *contract.PluginError) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lenBuf); err != nil {
		return nil, contract.ErrFailedPluginRead(err)
	}
	msgLen := binary.BigEndian.Uint32(lenBuf)
	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(p.conn, msg); err != nil {
		return nil, contract.ErrFailedPluginRead(err)
	}
	return msg, nil
}
