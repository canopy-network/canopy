package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/julienschmidt/httprouter"
	"github.com/nsf/jsondiff"
)

// Version writes Canopy software's version information
func (s *Server) Version(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	write(w, SoftwareVersion, http.StatusOK)
}

// Transaction submits a transaction
func (s *Server) Transaction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Create a new instance of lib.Transaction to hold the incoming transaction data.
	tx := new(lib.Transaction)
	// Unmarshal the HTTP request body into the transaction instance.
	if ok := unmarshal(w, r, tx); !ok {
		return
	}
	// Submit transaction to RPC server
	s.submitTxs(w, []lib.TransactionI{tx})
}

// Transactions handles multiple transactions in a single request
func (s *Server) Transactions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// create a slice to hold the incoming transactions
	var txs []lib.TransactionI
	// unmarshal the HTTP request body into the transactions slice
	if ok := unmarshal(w, r, &txs); !ok {
		return
	}
	// submit transactions to RPC server
	s.submitTxs(w, txs)
}

// Height responds with the next block version
func (s *Server) Height(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// Create a read-only state for the latest block and write the height
	s.readOnlyState(0, func(state *fsm.StateMachine) lib.ErrorI {
		write(w, &lib.HeightResult{
			Height: state.Height(),
		}, http.StatusOK)
		return nil
	})
}

// Account responds with an account for the specified address
func (s *Server) Account(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetAccount(crypto.NewAddressFromBytes(a))
	})
}

// Accounts responds with accounts based on the page parameters
func (s *Server) Accounts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetAccountsPaginated(p.PageParams)
	})
}

// Pool returns a Pool structure for a specific ID
func (s *Server) Pool(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.GetPool(id)
	})
}

// Pools returns a page of pools
func (s *Server) Pools(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetPoolsPaginated(p.PageParams)
	})
}

// Validator gets the validator at the specified address
func (s *Server) Validator(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetValidator(crypto.NewAddressFromBytes(a))
	})
}

// Validators returns a page of filtered validators
func (s *Server) Validators(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetValidatorsPaginated(p.PageParams, p.ValidatorFilters)
	})
}

// Committee returns a page of committee members ordered from the highest stake to lowest
func (s *Server) Committee(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetCommitteePaginated(p.PageParams, p.ValidatorFilters.Committee)
	})
}

// ValidatorSet retrieves the ValidatorSet that is responsible for the chainId
func (s *Server) ValidatorSet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		members, err := s.GetCommitteeMembers(id)
		if err != nil {
			return nil, err
		}
		return members.ValidatorSet, nil
	})
}

// Checkpoint retrieves the checkpoint block hash for a certain committee and height combination
func (s *Server) Checkpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdIndexer(w, r, func(s lib.StoreI, height, id uint64) (interface{}, lib.ErrorI) {
		return s.GetCheckpoint(id, height)
	})
}

// RootChainInfo retrieves the root chain info for the specified chain
func (s *Server) RootChainInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req := new(heightAndIdRequest)
	// unmarshal request parameters
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	// if height is 0; set to the latest height
	if req.Height == 0 {
		req.Height = s.controller.FSM.Height()
	}
	// load the root chain info directly
	got, err := s.controller.FSM.LoadRootChainInfo(req.ID, req.Height)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, got, http.StatusOK)
}

// CommitteeData retrieves the committee data for the specified chain id
func (s *Server) CommitteeData(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.GetCommitteeData(id)
	})
}

// CommitteesData retrieves all committee data
func (s *Server) CommitteesData(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetCommitteesData() // consider pagination
	})
}

// SubsidizedCommittees returns a list of chainIds that receive a portion of the 'block reward'
func (s *Server) SubsidizedCommittees(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetSubsidizedCommittees() })
}

// RetiredCommittees returns a list of the retired chainIds
func (s *Server) RetiredCommittees(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetRetiredCommittees() })
}

// NonSigners returns all non-quorum-certificate-signers
func (s *Server) NonSigners(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetNonSigners()
	})
}

// Params returns the aggregated ParamSpaces in a single Params object
func (s *Server) Params(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetParams() })
}

// FeeParams returns the current state of the governance params in the Fee space
func (s *Server) FeeParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsFee() })
}

// ValParams returns the current state of the governance params in the Validator space
func (s *Server) ValParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsVal() })
}

// ConParams returns the current state of the governance params in the Consensus space
func (s *Server) ConParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsCons() })
}

// GovParams returns the current state of the governance params in the Governance space
func (s *Server) GovParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsGov() })
}

// EcoParameters economic governance parameters
func (s *Server) EcoParameters(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// unmarshal the requested chain id
	post := new(chainRequest)
	if ok := unmarshal(w, r, post); !ok {
		return
	}
	// create a read-only state for the latest block and determine economic parameters
	s.readOnlyState(0, func(state *fsm.StateMachine) lib.ErrorI {
		// get the root id
		rootChainId, err := state.GetRootChainId()
		if err != nil {
			return err
		}
		// get the lottery winner
		s.rcManager.l.Lock()
		delegate, err := s.rcManager.GetLotteryWinner(rootChainId, 0, s.config.ChainId)
		s.rcManager.l.Unlock()
		// if an error occurred
		if err != nil {
			return err
		}
		// ensure non-nil delegate
		if delegate == nil {
			return lib.ErrEmptyLotteryWinner()
		}
		// find proposer cut
		proposerCut := 100 - delegate.Cut
		// remove sub-validator and sub-delegate cuts if requested chain id is non-root id
		if post.ChainId != rootChainId {
			proposerCut -= delegate.Cut // sub-validator
			proposerCut -= delegate.Cut // sub-delegate
		}
		_, daoCut, totalMint, committeeMint, err := state.GetBlockMintStats(post.ChainId)
		if err != nil {
			write(w, err.Error(), http.StatusBadRequest)
			return nil
		}
		write(w, economicParameterResponse{
			DAOCut:           daoCut,
			MintPerBlock:     totalMint,
			MintPerCommittee: committeeMint,
			ProposerCut:      proposerCut,
			DelegateCut:      delegate.Cut,
		}, http.StatusOK)
		return nil
	})
}

// Order gets an order for the specified chain
func (s *Server) Order(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.orderParams(w, r, func(s *fsm.StateMachine, p *orderRequest) (any, lib.ErrorI) {
		orderId, err := lib.StringToBytes(p.OrderId)
		if err != nil {
			return nil, err
		}
		return s.GetOrder(orderId, p.ChainId)
	})
}

// Orders retrieves the order book for a committee with optional filters and pagination
func (s *Server) Orders(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.ordersParams(w, r, func(s *fsm.StateMachine, req *ordersRequest) (any, lib.ErrorI) {
		// convert seller address if provided
		var sellerAddr []byte
		if req.SellersSendAddress != "" {
			var err lib.ErrorI
			sellerAddr, err = lib.StringToBytes(req.SellersSendAddress)
			if err != nil {
				return nil, err
			}
		}
		// use paginated query
		return s.GetOrdersPaginated(sellerAddr, req.Committee, req.PageParams)
	})
}

// DexPrice retrieves the latest dex price for a committee
func (s *Server) DexPrice(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (any, lib.ErrorI) {
		if id == 0 {
			return s.GetDexPrices()
		}
		// return the dex price
		return s.GetDexPrice(id)
	})
}

// DexBatch retrieves the 'locked' dex batch for a committee
func (s *Server) DexBatch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIdAndPointsParams(w, r, func(s *fsm.StateMachine, id uint64, points bool) (any, lib.ErrorI) {
		if id == 0 {
			return s.GetDexBatches(true)
		}
		// return the locked batch
		return s.GetDexBatch(id, true, points) // points augmentation used for liveness safety mirrors
	})
}

// NextDexBatch retrieves the 'up-next' dex batch for a committee
func (s *Server) NextDexBatch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIdAndPointsParams(w, r, func(s *fsm.StateMachine, id uint64, points bool) (any, lib.ErrorI) {
		if id == 0 {
			return s.GetDexBatches(false)
		}
		// return the locked batch
		return s.GetDexBatch(id, false, points)
	})
}

// LastProposers returns the last Proposer addresses
func (s *Server) LastProposers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetLastProposers()
	})
}

// MinimumEvidenceHeight returns the minimum height the evidence must be to still be usable
func (s *Server) MinimumEvidenceHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.LoadMinimumEvidenceHeight()
	})
}

// Lottery selects a validator/delegate randomly weighted based on their stake within a committee
func (s *Server) Lottery(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.LotteryWinner(id)
	})
}

// IsValidDoubleSigner returns if the DoubleSigner is already set for a height
func (s *Server) IsValidDoubleSigner(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightAndAddrIndexer(w, r, func(s lib.StoreI, h uint64, a lib.HexBytes) (interface{}, lib.ErrorI) {
		// ensure the last quorum certificate doesn't expose any valid double signers that aren't yet indexed
		qc, err := s.GetQCByHeight(s.Version() - 1)
		if err != nil {
			return nil, err
		}
		if qc.Results != nil && qc.Results.SlashRecipients != nil {
			for _, ds := range qc.Results.SlashRecipients.DoubleSigners {
				// get the public key from the address
				pk, e := crypto.NewPublicKeyFromBytes(ds.Id)
				if e != nil {
					continue
				}
				// if contains height, return not valid signer
				if bytes.Equal(pk.Address().Bytes(), a) && slices.Contains(ds.Heights, h) {
					return false, nil
				}
			}
		}
		// check the indexer
		return s.IsValidDoubleSigner(a, h)
	})
}

// DoubleSigners returns all double signers in the indexer
func (s *Server) DoubleSigners(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, _ uint64, _ lib.PageParams) (interface{}, lib.ErrorI) {
		return s.GetDoubleSigners()
	})
}

// BlockByHeight responds with the block data found at a specific height
func (s *Server) BlockByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetBlockByHeight(h) })
}

// CertByHeight response with the quorum certificate at height h
func (s *Server) CertByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetQCByHeight(h) })
}

// BlockByHash responds with block with hash h
func (s *Server) BlockByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetBlockByHash(h) })
}

// Blocks responds with a page of blocks based on the page parameters
func (s *Server) Blocks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, _ uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetBlocks(p) })
}

// TransactionByHash responds with a transaction with the hash h
func (s *Server) TransactionByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetTxByHash(h) })
}

// TransactionsByHeight response with the transactions at block height h
func (s *Server) TransactionsByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetTxsByHeight(h, true, p) })
}

// EventsByHeight response with the events at block height h
func (s *Server) EventsByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightIndexer(w, r, func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetEventsByBlockHeight(h, true, p)
	})
}

// EventsByAddress response with the events of address a
func (s *Server) EventsByAddress(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetEventsByAddress(a, true, p)
	})
}

// EventsByChain response with the events for a certain chain id
func (s *Server) EventsByChain(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.idIndexer(w, r, func(s lib.StoreI, id uint64, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetEventsByChainId(id, true, p)
	})
}

// Pending responds with a page of unconfirmed mempool transactions
func (s *Server) Pending(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.pageIndexer(w, r, func(_ lib.StoreI, _ crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.controller.GetPendingPage(p)
	})
}

// Supply returns the Supply structure
func (s *Server) Supply(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetSupply()
	})
}

// State exports the blockchain state at the requested height
func (s *Server) State(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request := new(heightsRequest)
	if err := r.ParseForm(); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	request.Height = parseUint64FromString(r.Form.Get("height"))

	s.readOnlyState(request.Height, func(state *fsm.StateMachine) (err lib.ErrorI) {
		exported, err := state.ExportState()
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, exported, http.StatusOK)
		return
	})
}

// StateDiff returns the different between the state at two block heights
func (s *Server) StateDiff(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	sm1, sm2, opts, ok := s.getDoubleStateMachineFromHeightParams(w, r, p)
	if !ok {
		return
	}
	defer sm1.Discard()
	defer sm2.Discard()
	state1, e := sm1.ExportState()
	if e != nil {
		write(w, e.Error(), http.StatusInternalServerError)
		return
	}
	state2, e := sm2.ExportState()
	if e != nil {
		write(w, e.Error(), http.StatusInternalServerError)
		return
	}
	j1, _ := json.Marshal(state1)
	j2, _ := json.Marshal(state2)
	_, differ := jsondiff.Compare(j1, j2, opts)
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		differ = "<pre>" + differ + "</pre>"
	}
	if _, err := w.Write([]byte(differ)); err != nil {
		s.logger.Error(err.Error())
	}
}

// TransactionsBySender returns transactions for the specified sender address
func (s *Server) TransactionsBySender(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsBySender(a, true, p)
	})
}

// TransactionsByRecipient returns transactions for the specified recipient address
func (s *Server) TransactionsByRecipient(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsByRecipient(a, true, p)
	})
}

// FailedTxs returns a list of failed mempool transactions for the specified address
func (s *Server) FailedTxs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Invoke helper with the HTTP request, response writer and an inline callback
	s.addrIndexer(w, r, func(_ lib.StoreI, address crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.controller.GetFailedTxsPage(address.String(), p)
	})
}

// Proposals returns the proposals present
func (s *Server) Proposals(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	bz, err := os.ReadFile(filepath.Join(s.config.DataDirPath, lib.ProposalsFilePath))
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentType, ApplicationJSON)
	if _, err = w.Write(bz); err != nil {
		s.logger.Error(err.Error())
	}
}

// Poll returns poll results
func (s *Server) Poll(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	s.pollMux.Lock()
	bz, e := lib.MarshalJSONIndent(s.poll)
	s.pollMux.Unlock()
	if e != nil {
		write(w, e, http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bz); err != nil {
		s.logger.Error(err.Error())
	}
}

// IndexerBlobs returns the current and previous indexer blobs as protobuf bytes
func (s *Server) IndexerBlobs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req := new(indexerBlobsRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	_, bz, err := s.IndexerBlobsCached(req.Height, req.Delta)
	if err != nil {
		status := http.StatusBadRequest
		if err.Code() == lib.CodeMarshal {
			status = http.StatusInternalServerError
		}
		write(w, err, status)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set(ContentType, "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(bz); err != nil {
		s.logger.Error(err.Error())
	}
}

// IndexerBlobsCached() is a helper function for the indexer blobs implementation
func (s *Server) IndexerBlobsCached(height uint64, delta bool) (*fsm.IndexerBlobs, []byte, lib.ErrorI) {
	currentHeight := s.controller.FSM.Height()
	if height == 0 || height > currentHeight {
		height = currentHeight
	}

	if entry, ok := s.indexerBlobCache.get(height); ok && entry != nil && entry.blobs != nil && entry.protoBytes != nil {
		if !delta {
			return entry.blobs, entry.protoBytes, nil
		}
		blobDelta, err := fsm.DeltaIndexerBlobs(entry.blobs)
		if err != nil {
			return nil, nil, err
		}
		deltaBytes, err := lib.Marshal(blobDelta)
		if err != nil {
			return nil, nil, err
		}
		return blobDelta, deltaBytes, nil
	}

	current, err := s.controller.FSM.IndexerBlob(height)
	if err != nil {
		return nil, nil, err
	}

	var previous *fsm.IndexerBlob
	// IndexerBlob(height) is only valid for height >= 2 (it pairs state@height with block height-1).
	// Therefore "previous" exists only when (height-1) >= 2, i.e. height >= 3.
	if height > 2 {
		if cachedPrev, ok := s.indexerBlobCache.getCurrent(height - 1); ok {
			previous = cachedPrev
		} else {
			prev, prevErr := s.controller.FSM.IndexerBlob(height - 1)
			if prevErr != nil {
				return nil, nil, prevErr
			}
			previous = prev
		}
	}

	blobs := &fsm.IndexerBlobs{
		Current:  current,
		Previous: previous,
	}
	protoBytes, err := lib.Marshal(blobs)
	if err != nil {
		return nil, nil, err
	}

	s.indexerBlobCache.put(height, &indexerBlobCacheEntry{
		height:     height,
		blobs:      blobs,
		protoBytes: protoBytes,
	})

	if !delta {
		return blobs, protoBytes, nil
	}
	blobDelta, err := fsm.DeltaIndexerBlobs(blobs)
	if err != nil {
		return nil, nil, err
	}
	deltaBytes, err := lib.Marshal(blobDelta)
	if err != nil {
		return nil, nil, err
	}
	return blobDelta, deltaBytes, nil
}

// orderParams is a helper function to abstract common workflows around a callback requiring a state machine and order request
func (s *Server) orderParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, request *orderRequest) (any, lib.ErrorI)) {
	req := new(orderRequest)

	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state, req)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// ordersParams is a helper function to abstract common workflows around a callback requiring a state machine and orders request
func (s *Server) ordersParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, request *ordersRequest) (any, lib.ErrorI)) {
	req := new(ordersRequest)

	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state, req)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// heightParams is a helper function to abstract common workflows around a callback requiring a state machine
func (s *Server) heightParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine) (any, lib.ErrorI)) {
	req := new(heightRequest)
	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// heightPaginated is a helper function to abstract common workflows around a callback requiring a state machine and page parameters
func (s *Server) heightPaginated(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, p *paginatedHeightRequest) (any, lib.ErrorI)) {
	req := new(paginatedHeightRequest)
	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state, req)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// heightAndAddressParams is a helper function to execute a callback with a state machine and address as parameters
func (s *Server) heightAndAddressParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, lib.HexBytes) (any, lib.ErrorI)) {
	req := new(heightAndAddressRequest)
	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		if req.Address == nil {
			write(w, fsm.ErrAddressEmpty(), http.StatusBadRequest)
			return
		}
		p, err := callback(state, req.Address)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// heightAndIdParams is a helper function to execute a callback with a state machine and ID as parameters
func (s *Server) heightAndIdParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, uint64) (any, lib.ErrorI)) {
	req := new(heightAndIdRequest)
	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state, req.ID)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// heightIdAndPointsParams is a helper function to execute a callback with a state machine, ID and points as parameters
func (s *Server) heightIdAndPointsParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, uint64, bool) (any, lib.ErrorI)) {
	req := new(heightIdAndPointsRequest)
	s.readOnlyStateFromHeightParams(w, r, req, func(state *fsm.StateMachine) (err lib.ErrorI) {
		p, err := callback(state, req.ID, req.Points)
		if err != nil {
			write(w, err, http.StatusBadRequest)
			return
		}
		write(w, p, http.StatusOK)
		return
	})
}

// getDoubleStateMachineFromHeightParams is a helper function to get two read-only state machines at the specified heights
func (s *Server) getDoubleStateMachineFromHeightParams(w http.ResponseWriter, r *http.Request, p httprouter.Params) (sm1, sm2 *fsm.StateMachine, o *jsondiff.Options, ok bool) {
	request, opts := new(heightsRequest), jsondiff.Options{}
	switch r.Method {
	case http.MethodGet:
		opts = jsondiff.DefaultHTMLOptions()
		opts.ChangedSeparator = " <- "
		if err := r.ParseForm(); err != nil {
			ok = false
			write(w, err, http.StatusBadRequest)
			return
		}
		request.Height = parseUint64FromString(r.Form.Get("height"))
		request.StartHeight = parseUint64FromString(r.Form.Get("startHeight"))
	case http.MethodPost:
		opts = jsondiff.DefaultConsoleOptions()
		if ok = unmarshal(w, r, request); !ok {
			return
		}
	}
	sm1, ok = s.getStateMachineWithHeight(request.Height, w)
	if !ok {
		return
	}
	if request.StartHeight == 0 {
		request.StartHeight = sm1.Height() - 1
	}
	sm2, ok = s.getStateMachineWithHeight(request.StartHeight, w)
	o = &opts
	return
}

// heightIndexer is a helper function to abstract common workflows around a callback requiring height and page params
func (s *Server) heightIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI)) {
	// Initialize a new paginatedHeightRequest
	req := new(paginatedHeightRequest)
	// Attempt to unmarshal the request into the req object
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	// Set up the store for the request context
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	// Ensure that the store is discarded safely after processing
	defer st.Discard()
	if req.Height == 0 {
		req.Height = st.Version() - 1
	}
	// Execute callback with store, height, and pagination parameters
	p, err := callback(st, req.Height, req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	// Write the successful result to the response
	write(w, p, http.StatusOK)
}

// idIndexer is a helper function to abstract common workflows around a callback requiring id and page params
func (s *Server) idIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, id uint64, p lib.PageParams) (any, lib.ErrorI)) {
	// Initialize a new paginatedHeightRequest
	req := new(paginatedIdRequest)
	// Attempt to unmarshal the request into the req object
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	// Set up the store for the request context
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	// Ensure that the store is discarded safely after processing
	defer st.Discard()
	// Execute callback with store, height, and pagination parameters
	p, err := callback(st, req.ID, req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	// Write the successful result to the response
	write(w, p, http.StatusOK)
}

// heightAndAddrIndexer is a helper function to abstract common workflows around a callback requiring height and address parameters
func (s *Server) heightAndAddrIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h uint64, address lib.HexBytes) (any, lib.ErrorI)) {
	req := new(heightAndAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	defer st.Discard()
	if req.Height == 0 {
		req.Height = st.Version() - 1
	}
	p, err := callback(st, req.Height, req.Address)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

// heightAndIdIndexer is a helper function to abstract common workflows around a callback requiring height and ID parameters
func (s *Server) heightAndIdIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h, id uint64) (any, lib.ErrorI)) {
	req := new(heightAndIdRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	defer st.Discard()
	if req.Height == 0 {
		req.Height = st.Version() - 1
	}
	p, err := callback(st, req.Height, req.ID)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

// hashIndexer is a helper function to abstract common workflows around a callback requiring a hash parameter
func (s *Server) hashIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI)) {
	req := new(hashRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	defer st.Discard()
	bz, err := lib.StringToBytes(req.Hash)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	p, err := callback(st, bz)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

// addrIndexer is a helper function to abstract common workflows around a callback requiring an address and page parameterse
func (s *Server) addrIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	defer st.Discard()
	if req.Address == nil {
		write(w, fsm.ErrAddressEmpty(), http.StatusBadRequest)
		return
	}
	p, err := callback(st, crypto.NewAddressFromBytes(req.Address), req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

// pageIndexer is a helper function to abstract common workflows around a callback requiring an address and page parameterse
// TODO very similar to above
func (s *Server) pageIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	st, ok := s.setupStore(w)
	if !ok {
		return
	}
	defer st.Discard()
	p, err := callback(st, crypto.NewAddressFromBytes(req.Address), req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

// setupStore creates a new store from the state machine's database. This store must be closed safely with Discard()
func (s *Server) setupStore(w http.ResponseWriter) (lib.StoreI, bool) {
	db := s.controller.FSM.Store().(lib.StoreI).DB()
	st, err := store.NewStoreWithDB(s.config, db, nil, s.logger)
	if err != nil {
		write(w, lib.ErrNewStore(err), http.StatusInternalServerError)
		return nil, false
	}
	return st, true
}

// withStore() executes a read only store function
func (s *Server) withStore(fn func(st *store.Store) (any, error)) (any, error) {
	st, err := store.NewStoreWithDB(s.config, s.controller.FSM.Store().(lib.StoreI).DB(), nil, s.logger)
	if err != nil {
		return nil, err
	}
	defer st.Discard()
	return fn(st)
}

// debugHandler is the http handler for debugging endpoints
func debugHandler(routeName string) httprouter.Handle {
	var f http.HandlerFunc
	switch routeName {
	case DebugHeapRouteName, DebugGoroutineRouteName, DebugBlockedRouteName:
		f = func(w http.ResponseWriter, r *http.Request) {
			pprof.Handler(routeName).ServeHTTP(w, r)
		}
	case DebugCPURouteName:
		f = pprof.Profile
	default:
		f = func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}
	}
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		f(w, r)
	}
}

// parseUint64FromString parses a string into a uint64
func parseUint64FromString(s string) uint64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return uint64(i)
}
