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
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// uses golang's semver naming convention
	// https://pkg.go.dev/golang.org/x/mod/semver
	SoftwareVersion = "v0.1.18+beta"
	ContentType     = "Content-MessageType"
	ApplicationJSON = "application/json; charset=utf-8"

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

	// handles interactions with the root chain rpc
	rcManager *RCManager

	// handles the indexer blob caching
	indexerBlobCache *indexerBlobCache

	logger lib.LoggerI
}

// NewServer constructs and returns a new Canopy RPC server
func NewServer(controller *controller.Controller, config lib.Config, logger lib.LoggerI) *Server {
	return &Server{
		controller:       controller,
		config:           config,
		logger:           logger,
		rcManager:        NewRCManager(controller, config, logger),
		poll:             make(fsm.Poll),
		pollMux:          &sync.RWMutex{},
		indexerBlobCache: newIndexerBlobCache(100),
	}
}

// Start initializes the Canopy RPC servers
func (s *Server) Start() {
	// Start the Query and Admin RPC servers concurrently
	go s.startRPC(createRouter(s), s.config.RPCPort)
	go s.startRPC(createAdminRouter(s), s.config.AdminPort)

	// Start tasks to update poll results and poll root chain information
	go s.updatePollResults()
	go s.rcManager.Start()
	go s.startEthRPCService()

	// Start heap profiler if enabled (warning: causes GC pauses which may affect RPC latency)
	if s.config.HeapProfilingEnabled {
		go s.startHeapProfiler()
	}

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
		Addr:              colon + port,
		ReadHeaderTimeout: timeout,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		Handler:           cor.Handler(router),
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

// startStaticFileServers starts a file server for the wallet and explorer
func (s *Server) startStaticFileServers() {
	s.logger.Infof("Starting Web Wallet 🔑 http://localhost:%s ⬅️", s.config.WalletPort)
	s.runStaticFileServer(walletFS, walletStaticDir, s.config.WalletPort, s.config)
	s.logger.Infof("Starting Block Explorer 🔍️ http://localhost:%s ⬅️", s.config.ExplorerPort)
	s.runStaticFileServer(explorerFS, explorerStaticDir, s.config.ExplorerPort, s.config)
}

// startHeapProfiler writes periodic heap profiles to the data directory
// Warning: This calls runtime.GC() which causes pauses and may affect RPC latency
func (s *Server) startHeapProfiler() {
	interval := time.Duration(s.config.HeapProfilingIntervalS) * time.Second
	if interval <= 0 {
		interval = 10 * time.Second
	}
	s.logger.Infof("Starting heap profiler with %s interval", interval)
	fileName := "heap1.out"
	for range time.Tick(interval) {
		f, err := os.Create(filepath.Join(s.config.DataDirPath, fileName))
		if err != nil {
			s.logger.Errorf("could not create memory profile: %v", err)
			continue
		}
		runtime.GC() // get up-to-date statistics
		if err = pprof.WriteHeapProfile(f); err != nil {
			s.logger.Errorf("could not write memory profile: %v", err)
		}
		f.Close()
		// Alternate between two files
		if fileName == "heap1.out" {
			fileName = "heap2.out"
		} else {
			fileName = "heap1.out"
		}
	}
}

// submitTx submits transactions to the controller and writes http response
func (s *Server) submitTxs(w http.ResponseWriter, txs []lib.TransactionI) (ok bool) {
	// marshal each transaction to bytes
	var txBytes [][]byte
	for _, tx := range txs {
		bz, err := lib.Marshal(tx)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		txBytes = append(txBytes, bz)
	}
	// send transactions to controller
	if err := s.controller.SendTxMsgs(txBytes); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	// return hashes of all submitted transactions
	var hashes []*string
	for _, bz := range txBytes {
		hash := crypto.HashString(bz)
		hashes = append(hashes, &hash)
	}
	// if only one transaction was submitted, return the hash as a string
	if len(hashes) == 1 {
		write(w, hashes[0], http.StatusOK)
		return
	}
	// if multiple transactions were submitted, return the hashes as an array
	write(w, hashes, http.StatusOK)
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
func (s *Server) getFeeFromState(ptr *txRequest, messageName string, lockorder ...bool) lib.ErrorI {
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
func write(w http.ResponseWriter, payload any, code int) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(code)
	logger := lib.NewDefaultLogger()
	// Marshal and indent the payload
	bz, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		logger.Errorf("[server write()] JSON marshall failed: %v", err)
		return
	}
	if _, err := w.Write(bz); err != nil {
		logger.Errorf("[server write()] write response failed: %v", err)
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
