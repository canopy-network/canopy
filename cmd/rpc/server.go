package rpc

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	pprof2 "runtime/pprof"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/rs/cors"
)

const colon = ":"

var logger lib.LoggerI

// Server represents an RPC server with configuration options.
type Server struct {
	controller *controller.Controller
	config     lib.Config
	logger     lib.LoggerI

	// TODO Consider breaking poll functionality off into separate type
	// poll is a map of PollResults keyed by the hash of the proposal
	poll types.Poll

	// Mutex for Poll handler
	pollMux *sync.RWMutex
}

// NewServer is the constructor function for Server.
func NewServer(controller *controller.Controller, config lib.Config, logger lib.LoggerI) *Server {

	return &Server{
		controller: controller,
		config:     config,
		logger:     logger,
		poll:       make(types.Poll),
		pollMux:    &sync.RWMutex{},
	}
}

func (s *Server) Start() {

	// Create CORS policy
	cor := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS", "POST"},
	})

	// Create a default timeout for HTTP requests
	timeout := time.Duration(s.config.TimeoutS) * time.Second

	router := createRouter(s)
	adminRouter := createAdminRouter(s)

	// Start RPC Server
	go func() {
		s.logger.Infof("Starting RPC server at 0.0.0.0:%s", s.config.RPCPort)
		s.logger.Fatal((&http.Server{
			Addr:    colon + s.config.RPCPort,
			Handler: cor.Handler(http.TimeoutHandler(router, timeout, ErrServerTimeout().Error())),
		}).ListenAndServe().Error())
	}()

	// Start Admin RPC Server
	go func() {
		s.logger.Infof("Starting Admin RPC server at %s:%s", "0.0.0.0", s.config.AdminPort)
		s.logger.Fatal((&http.Server{
			Addr:    colon + s.config.AdminPort,
			Handler: cor.Handler(http.TimeoutHandler(adminRouter, timeout, ErrServerTimeout().Error())),
		}).ListenAndServe().Error())
	}()

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
	if !s.config.Headless {
		s.logger.Infof("Starting Web Wallet 🔑 http://localhost:%s ⬅️", s.config.WalletPort)
		runStaticFileServer(walletFS, walletStaticDir, s.config.WalletPort, s.config)
		s.logger.Infof("Starting Block Explorer 🔍️ http://localhost:%s ⬅️", s.config.ExplorerPort)
		runStaticFileServer(explorerFS, explorerStaticDir, s.config.ExplorerPort, s.config)
	}
}

func (s *Server) submitTx(w http.ResponseWriter, tx any) (ok bool) {
	bz, err := lib.Marshal(tx)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	if err = s.controller.SendTxMsg(bz); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, crypto.HashString(bz), http.StatusOK)
	return true
}
