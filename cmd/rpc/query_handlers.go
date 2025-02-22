package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/nsf/jsondiff"
)

const (
	SoftwareVersion = "0.0.0-alpha"
	ContentType     = "Content-MessageType"
	ApplicationJSON = "application/json; charset=utf-8"
	localhost       = "localhost"
)

func (s *Server) Version(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	write(w, SoftwareVersion, http.StatusOK)
}

func (s *Server) Transaction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tx := new(lib.Transaction)
	if ok := unmarshal(w, r, tx); !ok {
		return
	}
	s.submitTx(w, tx)
}

func (s *Server) Height(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	state, ok := getStateMachineWithHeight(0, w)
	if !ok {
		return
	}

	// investigate state.Discard use here
	write(w, state.Height(), http.StatusOK)
}

func (s *Server) BlockByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetBlockByHeight(h) })
}

func (s *Server) CertByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetQCByHeight(h) })
}

func (s *Server) BlockByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetBlockByHash(h) })
}

func (s *Server) Blocks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, _ uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetBlocks(p) })
}

func (s *Server) TransactionByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetTxByHash(h) })
}

func (s *Server) TransactionsByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetTxsByHeight(h, true, p) })
}

func (s *Server) Pending(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pageIndexer(w, r, func(_ lib.StoreI, _ crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.controller.GetPendingPage(p)
	})
}

func (s *Server) Account(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetAccount(crypto.NewAddressFromBytes(a))
	})
}

func (s *Server) Accounts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetAccountsPaginated(p.PageParams)
	})
}

func (s *Server) Pool(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.GetPool(id)
	})
}

func (s *Server) Pools(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetPoolsPaginated(p.PageParams)
	})
}

func (s *Server) Validator(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetValidator(crypto.NewAddressFromBytes(a))
	})
}

func (s *Server) Validators(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetValidatorsPaginated(p.PageParams, p.ValidatorFilters)
	})
}

func (s *Server) Committee(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetCommitteePaginated(p.PageParams, p.ValidatorFilters.Committee)
	})
}

func (s *Server) ValidatorSet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		members, err := s.GetCommitteeMembers(id)
		if err != nil {
			return nil, err
		}
		return members.ValidatorSet, nil
	})
}

func (s *Server) Checkpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdIndexer(w, r, func(s lib.StoreI, height, id uint64) (interface{}, lib.ErrorI) {
		return s.GetCheckpoint(id, height)
	})
}

func (s *Server) RootChainInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		// get the previous state machine height
		lastSM, err := s.TimeMachine(s.Height() - 1)
		if err != nil {
			return nil, err
		}
		// get the committee
		validatorSet, err := s.GetCommitteeMembers(id)
		if err != nil {
			return nil, err
		}
		// get the previous committee
		// allow an error here to have size 0 validator sets
		lastValidatorSet, _ := lastSM.GetCommitteeMembers(id)
		// get the last proposers
		lastProposers, err := s.GetLastProposers()
		if err != nil {
			return nil, err
		}
		// get the minimum evidence height
		minimumEvidenceHeight, err := s.LoadMinimumEvidenceHeight()
		if err != nil {
			return nil, err
		}
		// get the committee data
		committeeData, err := s.GetCommitteeData(id)
		if err != nil {
			return nil, err
		}
		// get the delegate lottery winner
		lotteryWinner, err := s.LotteryWinner(id)
		if err != nil {
			return nil, err
		}
		// get the order book
		orders, err := s.GetOrderBook(id)
		if err != nil {
			return nil, err
		}
		return &lib.RootChainInfo{
			Height:                 s.Height(),
			ValidatorSet:           validatorSet,
			LastValidatorSet:       lastValidatorSet,
			LastProposers:          lastProposers,
			MinimumEvidenceHeight:  minimumEvidenceHeight,
			LastChainHeightUpdated: committeeData.LastChainHeightUpdated,
			LotteryWinner:          lotteryWinner,
			Orders:                 orders,
		}, nil
	})
}

func (s *Server) CommitteeData(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.GetCommitteeData(id)
	})
}

func (s *Server) CommitteesData(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetCommitteesData() // consider pagination
	})
}

func (s *Server) SubsidizedCommittees(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetSubsidizedCommittees() })
}

func (s *Server) RetiredCommittees(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetRetiredCommittees() })
}

func (s *Server) Order(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	orderParams(w, r, func(s *fsm.StateMachine, p *orderRequest) (any, lib.ErrorI) {
		return s.GetOrder(p.OrderId, p.ChainId)
	})
}

func (s *Server) Orders(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (any, lib.ErrorI) {
		if id == 0 {
			return s.GetOrderBooks()
		}
		b, err := s.GetOrderBook(id)
		if err != nil {
			return nil, err
		}
		return &lib.OrderBooks{OrderBooks: []*lib.OrderBook{b}}, nil
	})
}

func (s *Server) LastProposers(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetLastProposers()
	})
}

func (s *Server) MinimumEvidenceHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.LoadMinimumEvidenceHeight()
	})
}

func (s *Server) Lottery(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndIdParams(w, r, func(s *fsm.StateMachine, id uint64) (interface{}, lib.ErrorI) {
		return s.LotteryWinner(id)
	})
}

func (s *Server) IsValidDoubleSigner(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndAddrIndexer(w, r, func(s lib.StoreI, h uint64, a lib.HexBytes) (interface{}, lib.ErrorI) {
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

func (s *Server) DoubleSigners(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, _ uint64, _ lib.PageParams) (interface{}, lib.ErrorI) {
		return s.GetDoubleSigners()
	})
}

func (s *Server) Params(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetParams() })
}

func (s *Server) FeeParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsFee() })
}

func (s *Server) ValParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsVal() })
}

func (s *Server) ConParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsCons() })
}

func (s *Server) GovParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsGov() })
}

func (s *Server) NonSigners(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetNonSigners()
	})
}

func (s *Server) Supply(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetSupply()
	})
}

func (s *Server) State(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request := new(heightsRequest)
	if err := r.ParseForm(); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	request.Height = parseUint64FromString(r.Form.Get("height"))
	sm, ok := getStateMachineWithHeight(request.Height, w)
	if !ok {
		return
	}
	defer sm.Discard()
	state, err := sm.ExportState()
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, state, http.StatusOK)
}

func (s *Server) StateDiff(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	sm1, sm2, opts, ok := getDoubleStateMachineFromHeightParams(w, r, p)
	if !ok {
		return
	}
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
		logger.Error(err.Error())
	}
}

func (s *Server) TransactionsBySender(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsBySender(a, true, p)
	})
}

func (s *Server) TransactionsByRecipient(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsByRecipient(a, true, p)
	})
}

func (s *Server) FailedTxs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(_ lib.StoreI, address crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.controller.GetFailedTxsPage(address.String(), p)
	})
}

func (s *Server) Proposals(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	bz, err := os.ReadFile(filepath.Join(s.config.DataDirPath, lib.ProposalsFilePath))
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentType, ApplicationJSON)
	if _, err = w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}

func (s *Server) Poll(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	s.pollMux.Lock()
	bz, e := lib.MarshalJSONIndent(s.poll)
	s.pollMux.Unlock()
	if e != nil {
		write(w, e, http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}
