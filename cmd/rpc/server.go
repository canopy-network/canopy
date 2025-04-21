package rpc

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	pprof2 "runtime/pprof"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

const (
	colon = ":"

	SoftwareVersion = "0.0.0-alpha"
	ContentType     = "Content-MessageType"
	ApplicationJSON = "application/json; charset=utf-8"
	localhost       = "localhost"

	walletStaticDir   = "web/wallet/out"
	explorerStaticDir = "web/explorer/out"
)

// Server represents a Canopy RPC server with configuration options.
type Server struct {
	// Canopy node controller
	controller *controller.Controller

	// Canopy node configuration
	config lib.Config

	// poll is a map of PollResults keyed by the hash of the proposal
	poll fsm.Poll

	// Mutex for Poll handler
	pollMux *sync.RWMutex

	// RemoteCallbacks to the root chain rpc
	remoteCallbacks *lib.RemoteCallbacks

	logger lib.LoggerI
}

// NewServer constructs and returns a new Canopy RPC server
func NewServer(controller *controller.Controller, config lib.Config, logger lib.LoggerI) *Server {
	return &Server{
		controller: controller,
		config:     config,
		logger:     logger,
		poll:       make(fsm.Poll),
		pollMux:    &sync.RWMutex{},
	}
}

// Start initializes the Canopy RPC servers
func (s *Server) Start() {
	// Start the Query and Admin RPC servers concurrently
	go s.startRPC(createRouter(s), s.config.RPCPort)
	go s.startRPC(createAdminRouter(s), s.config.AdminPort)

	// Start tasks to update poll results and poll root chain information
	go s.updatePollResults()
	go s.pollRootChainInfo()

	go func() { // TODO remove DEBUG ONLY
		fileName := "heap1.out"
		for range time.Tick(time.Second * 10) {
			f, err := os.Create(filepath.Join(s.config.DataDirPath, fileName))
			if err != nil {
				s.logger.Fatalf("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err = pprof2.WriteHeapProfile(f); err != nil {
				s.logger.Fatalf("could not write memory profile: ", err)
			}
			f.Close()
			fileName = "heap2.out"
		}
	}()

	if s.config.Headless {
		return
	}

	// Start in-process HTTP servers for the wallet and explorer
	s.startStaticFileServers()
}

// startRPC starts an RPC server with the provided router and port
func (s *Server) startRPC(router *httprouter.Router, port string) {

	// Create CORS policy
	cor := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS", "POST"},
	})

	// Create a default timeout for HTTP requests
	timeout := time.Duration(s.config.TimeoutS) * time.Second

	// Start RPC server
	s.logger.Infof("Starting RPC server at 0.0.0.0:%s", port)
	s.logger.Fatal((&http.Server{
		Addr:    colon + port,
		Handler: cor.Handler(http.TimeoutHandler(router, timeout, lib.ErrServerTimeout().Error())),
	}).ListenAndServe().Error())
}

// updatePollResults() updates the poll results based on the current token power
func (s *Server) updatePollResults() {
	for {
		p := new(fsm.ActivePolls)
		if err := func() (err error) {
			if err = p.NewFromFile(s.config.DataDirPath); err != nil {
				return
			}

			s.readOnlyState(0, func(sm *fsm.StateMachine) lib.ErrorI {
				// cleanup old polls
				p.Cleanup(sm.Height())
				if err := p.SaveToFile(s.config.DataDirPath); err != nil {
					return err
				}

				// convert the poll to a result
				result, err := sm.PollsToResults(p)
				if err != nil || len(result) == 0 {
					return err
				}

				// make results available to RPC clients
				s.pollMux.Lock()
				s.poll = result
				s.pollMux.Unlock()
				return nil
			})
			return nil

		}(); err != nil {
			s.logger.Error(err.Error())
		}
		time.Sleep(time.Second * 3)
	}
}

// updateRootChainHeight queries and updates the root chain height
func (s *Server) updateRootChainHeight(state *fsm.StateMachine, rootChainHeight *uint64) (err lib.ErrorI) {
	// get the consensus params from the app
	consParams, err := state.GetParamsCons()
	if err != nil {
		return
	}
	// get the remote callbacks for the root chain id
	s.remoteCallbacks, err = s.RemoteCallbacks(consParams.RootChainId)
	if err != nil {
		s.logger.Errorf("callbacks failed with err: %s")
		return err
	}
	// query the base chain height
	height, err := s.remoteCallbacks.Height()
	if err != nil {
		s.logger.Errorf("GetRootChainHeight failed with err")
		return err
	}
	// check if a new height was received
	if *height <= *rootChainHeight {
		return
	}
	// update the root chain height
	*rootChainHeight = *height
	// if a new height received
	s.logger.Infof("New RootChain height %d detected!", *rootChainHeight)
	// execute the requests to get the base chain information
	for retry := lib.NewRetry(s.config.RootChainPollMS, 10); retry.WaitAndDoRetry(); {
		s.logger.Infof("Retrieved root height info for %d!", *rootChainHeight)
		// retrieve the root-Chain info
		rootChainInfo, e := s.remoteCallbacks.RootChainInfo(*rootChainHeight, s.config.ChainId)
		if e == nil && rootChainInfo != nil && rootChainInfo.ValidatorSet.NumValidators != 0 {
			// update the controller with new root-Chain info
			s.controller.UpdateRootChainInfo(rootChainInfo)
			s.logger.Info("Updated RootChain information")
			break
		}
		s.logger.Errorf("GetRootChainInfo failed with err %s", e.Error())
	}
	return
}

// pollRootChainInfo() retrieves information from the root-Chain required for consensus
func (s *Server) pollRootChainInfo() {
	// Track the root chain height
	rootChainHeight := uint64(0)
	// execute the loop every conf.RootChainPollMS duration
	ticker := time.NewTicker(time.Duration(s.config.RootChainPollMS) * time.Millisecond)
	for range ticker.C {
		if err := func() (err error) {
			// Create a read-only state machine context
			err = s.readOnlyState(0, func(state *fsm.StateMachine) (err lib.ErrorI) {
				// Update the root chain height
				return s.updateRootChainHeight(state, &rootChainHeight)
			})
			return
		}(); err != nil {
			s.logger.Warnf(err.Error())
		}
	}
}

// RemoteCallbacks() enables the retrieval of remote RPC API calls for a certain root chain id
func (s *Server) RemoteCallbacks(rootChainId uint64) (*lib.RemoteCallbacks, lib.ErrorI) {
	// get the url for the root chain as set by the state
	var rootChainUrl string
	// for each item in the root chain config
	for _, chain := range s.config.RootChain {
		// if the chain id matches
		if chain.ChainId == rootChainId {
			// use that root chain url
			rootChainUrl = chain.Url
		}
	}
	// check if root chain url isn't empty
	if rootChainUrl == "" {
		s.logger.Errorf("Config.JSON missing RootChainID=%d failed with", rootChainId)
		return nil, lib.ErrEmptyChainId()
	}
	// create a rpc client
	rpcClient := NewClient(rootChainUrl, "")
	// set the remote callbacks
	return &lib.RemoteCallbacks{
		Height:              rpcClient.Height,
		RootChainInfo:       rpcClient.RootChainInfo,
		ValidatorSet:        rpcClient.ValidatorSet,
		IsValidDoubleSigner: rpcClient.IsValidDoubleSigner,
		Lottery:             rpcClient.Lottery,
		Orders:              rpcClient.Orders,
		Order:               rpcClient.Order,
		Checkpoint:          rpcClient.Checkpoint,
		Transaction:         rpcClient.Transaction,
	}, nil
}

// startStaticFileServers starts a file server for the wallet and explorer
func (s *Server) startStaticFileServers() {
	s.logger.Infof("Starting Web Wallet 🔑 http://localhost:%s ⬅️", s.config.WalletPort)
	s.runStaticFileServer(walletFS, walletStaticDir, s.config.WalletPort, s.config)
	s.logger.Infof("Starting Block Explorer 🔍️ http://localhost:%s ⬅️", s.config.ExplorerPort)
	s.runStaticFileServer(explorerFS, explorerStaticDir, s.config.ExplorerPort, s.config)
}

// submitTx submits a transaction to the controller and writes http response
func (s *Server) submitTx(w http.ResponseWriter, tx any) (ok bool) {

	// Marshal the transaction
	bz, err := lib.Marshal(tx)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}

	// Send transaction to controller
	if err = s.controller.SendTxMsg(bz); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}

	// Write transaction to http response
	write(w, crypto.HashString(bz), http.StatusOK)
	return true
}

// setupStateMachine creates and returns a read-only state machine
func (s *Server) getStateMachineWithHeight(height uint64, w http.ResponseWriter) (*fsm.StateMachine, bool) {

	// Investigate  memory use of state. State.Discard needs to be called
	state, err := s.controller.FSM.TimeMachine(height)
	if err != nil {
		write(w, lib.ErrTimeMachine(err), http.StatusInternalServerError)
		return nil, false
	}
	return state, true
}

// getFeeFromState populates txRequest with the fee for the transaction type specified in messageName
func (s *Server) getFeeFromState(w http.ResponseWriter, ptr *txRequest, messageName string, lockorder ...bool) lib.ErrorI {
	return s.readOnlyState(0, func(state *fsm.StateMachine) lib.ErrorI {
		// Get fee for transaction
		minimumFee, err := state.GetFeeForMessageName(messageName)
		if err != nil {
			return err
		}
		// Apply the fee multiplier for buy orders
		isLockOrder := len(lockorder) == 1 && lockorder[0]
		if isLockOrder {
			// Get governance params
			params, e := state.GetParamsVal()
			if e != nil {
				return e
			}
			// Apply the fee multiplier
			minimumFee *= params.LockOrderFeeMultiplier
		}
		// Apply a minimum fee in the case of 0 fees
		if ptr.Fee == 0 {
			ptr.Fee = minimumFee
		}
		// Error if fee below minimum
		if ptr.Fee < minimumFee {
			return fsm.ErrTxFeeBelowStateLimit()
		}
		return nil
	})
}

// readOnlyStateFromHeightParams is a helper function to safely wrap TimeMachine access
func (s *Server) readOnlyStateFromHeightParams(w http.ResponseWriter, r *http.Request, ptr queryWithHeight, callback func(s *fsm.StateMachine) lib.ErrorI) (err lib.ErrorI) {

	// Unmarshal request parameters
	if ok := unmarshal(w, r, ptr); !ok {
		return
	}

	return s.readOnlyState(ptr.GetHeight(), callback)
}

// readOnlyState is a helper function to safely wrap TimeMachine access
func (s *Server) readOnlyState(height uint64, callback func(s *fsm.StateMachine) lib.ErrorI) lib.ErrorI {
	// Create a new TimeMachine at specified height
	state, err := s.controller.FSM.TimeMachine(height)
	if err != nil {
		return lib.ErrTimeMachine(err)
	}

	// Discard state, ensuring proper cleanup is performed
	defer state.Discard()

	// Execute the provided callback function with the read-only state
	err = callback(state)
	if err != nil {
		return err
	}
	return nil
}

// logsHandler writes the Canopy logfile
func logsHandler(s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		// Construct the full file path to the Canopy log file
		filePath := filepath.Join(s.config.DataDirPath, lib.LogDirectory, lib.LogFileName)

		// Read the entire contents of the log file and split by newlines
		f, _ := os.ReadFile(filePath)
		split := bytes.Split(f, []byte("\n"))

		// Prepare a slice to hold the reversed lines
		var flipped []byte

		// Iterate over the lines in reverse order
		for i := len(split) - 1; i >= 0; i-- {
			// Append each line to the `flipped` slice followed by a newline character
			flipped = append(append(flipped, split[i]...), []byte("\n")...)
		}

		// Write the reversed lines to the HTTP response
		if _, err := w.Write(flipped); err != nil {
			s.logger.Error(err.Error())
		}
	}
}

// logHandler serves as a middleware that logs incoming RPC calls for debugging purposes.
type logHandler struct {
	path string
	h    httprouter.Handle
}

// Handle
func (h logHandler) Handle(resp http.ResponseWriter, req *http.Request, p httprouter.Params) {
	// Uncomment the line below to enable endpoint path logging for debugging.
	// logger.Debug(h.path)

	// Call the actual handler function with the response, request, and parameters.
	h.h(resp, req, p)
}

//go:embed all:web/explorer/out
var explorerFS embed.FS

//go:embed all:web/wallet/out
var walletFS embed.FS

// runStaticFileServer creates a web server serving static files
func (s *Server) runStaticFileServer(fileSys fs.FS, dir, port string, conf lib.Config) {
	// Attempt to get a sub-filesystem rooted at the specified directory
	distFS, err := fs.Sub(fileSys, dir)
	if err != nil {
		s.logger.Error(fmt.Sprintf("an error occurred running the static file server for %s: %s", dir, err.Error()))
		return
	}

	// Create a new ServeMux to handle incoming HTTP requests
	mux := http.NewServeMux()

	// Define a handler function for the root path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// serve `index.html` with dynamic config injection
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {

			// Construct the file path for `index.html`
			filePath := path.Join(dir, "index.html")

			// Open the file and defer closing until the function exits
			data, e := fileSys.Open(filePath)
			if e != nil {
				http.NotFound(w, r)
				return
			}
			defer data.Close()

			// Read the content of `index.html` into a byte slice
			htmlBytes, e := fs.ReadFile(fileSys, filePath)
			if e != nil {
				http.NotFound(w, r)
				return
			}

			// Inject the configuration into the HTML file content
			injectedHTML := injectConfig(string(htmlBytes), conf)

			// Set the response header as HTML and write the injected content to the response
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(injectedHTML))
			return
		}

		// For all other requests, serve the files directly from the file system
		http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
	})

	// Start the HTTP server in a new goroutine and listen on the specified port
	go func() {
		// Log a fatal error if the server fails to start
		s.logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux).Error())
	}()
}

// injectConfig() injects the config.json into the HTML file
func injectConfig(html string, config lib.Config) string {
	script := fmt.Sprintf(`<script>
		window.__CONFIG__ = {
            rpcURL: "%s",
            adminRPCURL: "%s",
            chainId: %d
        };
	</script>`, config.RPCUrl, config.AdminRPCUrl, config.ChainId)

	// inject the script just before </head>
	return strings.Replace(html, "</head>", script+"</head>", 1)
}

// unmarshal reads request body and unmarshals it into ptr
func unmarshal(w http.ResponseWriter, r *http.Request, ptr interface{}) bool {
	bz, err := io.ReadAll(io.LimitReader(r.Body, int64(units.MB)))
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return false
	}
	defer func() { _ = r.Body.Close() }()
	if err = json.Unmarshal(bz, ptr); err != nil {
		write(w, err, http.StatusBadRequest)
		return false
	}
	return true
}

// write marshaled payload to w
func write(w http.ResponseWriter, payload interface{}, code int) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(code)

	// Marshal and indent the payload
	bz, _ := json.MarshalIndent(payload, "", "  ")
	if _, err := w.Write(bz); err != nil {
		l := lib.LoggerI(nil) // TODO temporary fix
		l.Error(err.Error())
	}
}

// StringToCommittees converts a comma separated string of committees to uint64
func StringToCommittees(s string) (committees []uint64, error error) {
	// Do not convert a single int - a single int is an option for subsidy txn
	i, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		return []uint64{i}, nil
	}

	// Remove all spaces and split on comma
	commaSeparatedArr := strings.Split(strings.ReplaceAll(s, " ", ""), ",")
	if len(commaSeparatedArr) == 0 {
		return nil, lib.ErrStringToCommittee(s)
	}

	// Convert each element to uint64
	for _, c := range commaSeparatedArr {
		ui, e := strconv.ParseUint(c, 10, 64)
		if e != nil {
			return nil, e
		}
		committees = append(committees, ui)
	}
	return
}
