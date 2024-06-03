package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/units"
	"github.com/dgraph-io/badger/v4"
	"github.com/ginchuco/ginchu/consensus"
	"github.com/ginchuco/ginchu/fsm"
	"github.com/ginchuco/ginchu/fsm/types"
	"github.com/ginchuco/ginchu/lib"
	"github.com/ginchuco/ginchu/lib/crypto"
	"github.com/ginchuco/ginchu/store"
	"github.com/julienschmidt/httprouter"
	"github.com/nsf/jsondiff"
	"github.com/rs/cors"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ContentType     = "Content-Type"
	ApplicationJSON = "application/json; charset=utf-8"
	localhost       = "127.0.0.1"
	colon           = ":"

	VersionRouteName        = "version"
	TxRouteName             = "tx"
	HeightRouteName         = "height"
	AccountRouteName        = "account"
	AccountsRouteName       = "accounts"
	PoolRouteName           = "pool"
	PoolsRouteName          = "pools"
	ValidatorRouteName      = "validator"
	ValidatorsRouteName     = "validators"
	ConsValidatorsRouteName = "cons-validators"
	NonSignersRouteName     = "non-signers"
	SupplyRouteName         = "supply"
	ParamRouteName          = "params"
	FeeParamRouteName       = "fee-params"
	GovParamRouteName       = "gov-params"
	ConParamsRouteName      = "con-params"
	ValParamRouteName       = "val-params"
	StateRouteName          = "state"
	StateDiffRouteName      = "state-diff"
	StateDiffGetRouteName   = "state-diff-get"
	CertByHeightRouteName   = "cert-by-height"
	BlocksRouteName         = "blocks"
	BlockByHeightRouteName  = "block-by-height"
	BlockByHashRouteName    = "block-by-hash"
	TxsByHeightRouteName    = "txs-by-height"
	TxsBySenderRouteName    = "txs-by-sender"
	TxsByRecRouteName       = "txs-by-rec"
	TxByHashRouteName       = "tx-by-hash"
	PendingRouteName        = "pending"
	ProposalsRouteName      = "proposals"
	PollRouteName           = "poll"
	// debug
	DebugBlockedRouteName = "blocked"
	DebugHeapRouteName    = "heap"
	DebugCPURouteName     = "cpu"
	DebugRoutineRouteName = "routine"
	// admin
	KeystoreRouteName          = "keystore"
	KeystoreNewKeyRouteName    = "keystore-new-key"
	KeystoreImportRouteName    = "keystore-import"
	KeystoreImportRawRouteName = "keystore-import-raw"
	KeystoreDeleteRouteName    = "keystore-delete"
	KeystoreGetRouteName       = "keystore-get"
	TxSendRouteName            = "tx-send"
	TxStakeRouteName           = "tx-stake"
	TxUnstakeRouteName         = "tx-unstake"
	TxEditStakeRouteName       = "tx-edit-stake"
	TxPauseRouteName           = "tx-pause"
	TxUnpauseRouteName         = "tx-unpause"
	TxChangeParamRouteName     = "tx-change-param"
	TxDAOTransferRouteName     = "tx-dao-transfer"
	ResourceUsageRouteName     = "resource-usage"
	PeerInfoRouteName          = "peer-info"
	ConsensusInfoRouteName     = "consensus-info"
	PeerBookRouteName          = "peer-book"
	ConfigRouteName            = "config"
	LogsRouteName              = "logs"
	AddVoteRouteName           = "add-vote"
	DelVoteRouteName           = "del-vote"
)

const SoftwareVersion = "0.0.0-alpha"

var (
	app    *consensus.Consensus
	db     *badger.DB
	conf   lib.Config
	logger lib.LoggerI

	router = routes{
		VersionRouteName:        {Method: http.MethodGet, Path: "/v1/", HandlerFunc: Version},
		TxRouteName:             {Method: http.MethodPost, Path: "/v1/tx", HandlerFunc: Transaction},
		HeightRouteName:         {Method: http.MethodPost, Path: "/v1/query/height", HandlerFunc: Height},
		AccountRouteName:        {Method: http.MethodPost, Path: "/v1/query/account", HandlerFunc: Account},
		AccountsRouteName:       {Method: http.MethodPost, Path: "/v1/query/accounts", HandlerFunc: Accounts},
		PoolRouteName:           {Method: http.MethodPost, Path: "/v1/query/pool", HandlerFunc: Pool},
		PoolsRouteName:          {Method: http.MethodPost, Path: "/v1/query/pools", HandlerFunc: Pools},
		ValidatorRouteName:      {Method: http.MethodPost, Path: "/v1/query/validator", HandlerFunc: Validator},
		ValidatorsRouteName:     {Method: http.MethodPost, Path: "/v1/query/validators", HandlerFunc: Validators},
		ConsValidatorsRouteName: {Method: http.MethodPost, Path: "/v1/query/cons-validators", HandlerFunc: ConsValidators},
		NonSignersRouteName:     {Method: http.MethodPost, Path: "/v1/query/non-signers", HandlerFunc: NonSigners},
		ParamRouteName:          {Method: http.MethodPost, Path: "/v1/query/params", HandlerFunc: Params},
		SupplyRouteName:         {Method: http.MethodPost, Path: "/v1/query/supply", HandlerFunc: Supply},
		FeeParamRouteName:       {Method: http.MethodPost, Path: "/v1/query/fee-params", HandlerFunc: FeeParams},
		GovParamRouteName:       {Method: http.MethodPost, Path: "/v1/query/gov-params", HandlerFunc: GovParams},
		ConParamsRouteName:      {Method: http.MethodPost, Path: "/v1/query/con-params", HandlerFunc: ConParams},
		ValParamRouteName:       {Method: http.MethodPost, Path: "/v1/query/val-params", HandlerFunc: ValParams},
		StateRouteName:          {Method: http.MethodPost, Path: "/v1/query/state", HandlerFunc: State},
		StateDiffRouteName:      {Method: http.MethodPost, Path: "/v1/query/state-diff", HandlerFunc: StateDiff},
		StateDiffGetRouteName:   {Method: http.MethodGet, Path: "/v1/query/state-diff", HandlerFunc: StateDiff},
		CertByHeightRouteName:   {Method: http.MethodPost, Path: "/v1/query/cert-by-height", HandlerFunc: CertByHeight},
		BlockByHeightRouteName:  {Method: http.MethodPost, Path: "/v1/query/block-by-height", HandlerFunc: BlockByHeight},
		BlocksRouteName:         {Method: http.MethodPost, Path: "/v1/query/blocks", HandlerFunc: Blocks},
		BlockByHashRouteName:    {Method: http.MethodPost, Path: "/v1/query/block-by-hash", HandlerFunc: BlockByHash},
		TxsByHeightRouteName:    {Method: http.MethodPost, Path: "/v1/query/txs-by-height", HandlerFunc: TransactionsByHeight},
		TxsBySenderRouteName:    {Method: http.MethodPost, Path: "/v1/query/txs-by-sender", HandlerFunc: TransactionsBySender},
		TxsByRecRouteName:       {Method: http.MethodPost, Path: "/v1/query/txs-by-rec", HandlerFunc: TransactionsByRecipient},
		TxByHashRouteName:       {Method: http.MethodPost, Path: "/v1/query/tx-by-hash", HandlerFunc: TransactionByHash},
		PendingRouteName:        {Method: http.MethodPost, Path: "/v1/query/pending", HandlerFunc: Pending},
		ProposalsRouteName:      {Method: http.MethodGet, Path: "/v1/gov/proposals", HandlerFunc: Proposals},
		PollRouteName:           {Method: http.MethodGet, Path: "/v1/gov/poll", HandlerFunc: Poll},
		// debug
		DebugBlockedRouteName: {Method: http.MethodPost, Path: "/debug/blocked", HandlerFunc: debugHandler(DebugBlockedRouteName)},
		DebugHeapRouteName:    {Method: http.MethodPost, Path: "/debug/heap", HandlerFunc: debugHandler(DebugHeapRouteName)},
		DebugCPURouteName:     {Method: http.MethodPost, Path: "/debug/cpu", HandlerFunc: debugHandler(DebugHeapRouteName)},
		DebugRoutineRouteName: {Method: http.MethodPost, Path: "/debug/routine", HandlerFunc: debugHandler(DebugRoutineRouteName)},
		// admin
		KeystoreRouteName:          {Method: http.MethodGet, Path: "/v1/admin/keystore", HandlerFunc: Keystore, AdminOnly: true},
		KeystoreNewKeyRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-new-key", HandlerFunc: KeystoreNewKey, AdminOnly: true},
		KeystoreImportRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-import", HandlerFunc: KeystoreImport, AdminOnly: true},
		KeystoreImportRawRouteName: {Method: http.MethodPost, Path: "/v1/admin/keystore-import-raw", HandlerFunc: KeystoreImportRaw, AdminOnly: true},
		KeystoreDeleteRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-delete", HandlerFunc: KeystoreDelete, AdminOnly: true},
		KeystoreGetRouteName:       {Method: http.MethodPost, Path: "/v1/admin/keystore-get", HandlerFunc: KeystoreGetKeyGroup, AdminOnly: true},
		TxSendRouteName:            {Method: http.MethodPost, Path: "/v1/admin/tx-send", HandlerFunc: TransactionSend, AdminOnly: true},
		TxStakeRouteName:           {Method: http.MethodPost, Path: "/v1/admin/tx-stake", HandlerFunc: TransactionStake, AdminOnly: true},
		TxEditStakeRouteName:       {Method: http.MethodPost, Path: "/v1/admin/tx-edit-stake", HandlerFunc: TransactionEditStake, AdminOnly: true},
		TxUnstakeRouteName:         {Method: http.MethodPost, Path: "/v1/admin/tx-unstake", HandlerFunc: TransactionUnstake, AdminOnly: true},
		TxPauseRouteName:           {Method: http.MethodPost, Path: "/v1/admin/tx-pause", HandlerFunc: TransactionPause, AdminOnly: true},
		TxUnpauseRouteName:         {Method: http.MethodPost, Path: "/v1/admin/tx-unpause", HandlerFunc: TransactionUnpause, AdminOnly: true},
		TxChangeParamRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-change-param", HandlerFunc: TransactionChangeParam, AdminOnly: true},
		TxDAOTransferRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-dao-transfer", HandlerFunc: TransactionDAOTransfer, AdminOnly: true},
		ResourceUsageRouteName:     {Method: http.MethodGet, Path: "/v1/admin/resource-usage", HandlerFunc: ResourceUsage, AdminOnly: true},
		PeerInfoRouteName:          {Method: http.MethodGet, Path: "/v1/admin/peer-info", HandlerFunc: PeerInfo, AdminOnly: true},
		ConsensusInfoRouteName:     {Method: http.MethodGet, Path: "/v1/admin/consensus-info", HandlerFunc: ConsensusInfo, AdminOnly: true},
		PeerBookRouteName:          {Method: http.MethodGet, Path: "/v1/admin/peer-book", HandlerFunc: PeerBook, AdminOnly: true},
		ConfigRouteName:            {Method: http.MethodGet, Path: "/v1/admin/config", HandlerFunc: Config, AdminOnly: true},
		LogsRouteName:              {Method: http.MethodGet, Path: "/v1/admin/log", HandlerFunc: logsHandler(), AdminOnly: true},
		AddVoteRouteName:           {Method: http.MethodPost, Path: "/v1/gov/add-vote", HandlerFunc: AddVote, AdminOnly: true},
		DelVoteRouteName:           {Method: http.MethodPost, Path: "/v1/gov/del-vote", HandlerFunc: DelVote, AdminOnly: true},
	}
)

func StartRPC(a *consensus.Consensus, c lib.Config, l lib.LoggerI) {
	cor := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "OPTIONS", "POST"},
	})
	s, timeout := a.FSM.Store().(lib.StoreI), time.Duration(c.TimeoutS)*time.Second
	app, conf, db, logger = a, c, s.DB(), l
	l.Infof("Starting RPC server at 0.0.0.0:%s", c.RPCPort)
	go func() {
		l.Fatal((&http.Server{
			Addr:    colon + c.RPCPort,
			Handler: cor.Handler(http.TimeoutHandler(router.New(), timeout, ErrServerTimeout().Error())),
		}).ListenAndServe().Error())
	}()
	l.Infof("Starting Admin RPC server at %s:%s", localhost, c.AdminPort)
	go func() {
		l.Fatal((&http.Server{
			Addr:    localhost + colon + c.AdminPort,
			Handler: cor.Handler(http.TimeoutHandler(router.NewAdmin(), timeout, ErrServerTimeout().Error())),
		}).ListenAndServe().Error())
	}()
	go pollValidators(time.Minute)
	go resetSeqCacher(time.Second * 5)
}

func Version(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	write(w, SoftwareVersion, http.StatusOK)
}

func Height(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	state, ok := getStateMachineWithHeight(0, w)
	if !ok {
		return
	}
	write(w, state.Height(), http.StatusOK)
}

func BlockByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetBlockByHeight(h) })
}

func CertByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, _ lib.PageParams) (any, lib.ErrorI) { return s.GetQCByHeight(h) })
}

func BlockByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetBlockByHash(h) })
}

func Blocks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, _ uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetBlocks(p) })
}

func TransactionByHash(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	hashIndexer(w, r, func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI) { return s.GetTxByHash(h) })
}

func TransactionsByHeight(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightIndexer(w, r, func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI) { return s.GetTxsByHeight(h, true, p) })
}

func Pending(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(_ lib.StoreI, _ crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return app.GetPendingPage(p)
	})
}

func Proposals(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	bz, err := os.ReadFile(filepath.Join(conf.DataDirPath, lib.ProposalsFilePath))
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentType, ApplicationJSON)
	if _, err = w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}

var (
	pollMux = sync.Mutex{}
	poll    = make(types.Poll)
)

func Poll(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	pollMux.Lock()
	bz, e := lib.MarshalJSONIndent(poll)
	pollMux.Unlock()
	if e != nil {
		write(w, e, http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}

func pollValidators(frequency time.Duration) {
	s, e := store.NewStoreWithDB(db, logger)
	if e != nil {
		panic(e)
	}
	defer s.Discard()
	for {
		state, err := fsm.New(conf, s, logger)
		if err != nil {
			logger.Error(err.Error())
			time.Sleep(frequency)
			continue
		}
		vals, err := state.GetConsensusValidators()
		if err != nil {
			logger.Error(err.Error())
			time.Sleep(frequency)
			continue
		}
		pollMux.Lock()
		poll = types.PollValidators(vals, router[ProposalsRouteName].Path, logger)
		pollMux.Unlock()
		time.Sleep(frequency)
	}
}

func AddVote(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	proposals := make(types.Proposals)
	if err := proposals.NewFromFile(conf.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	j := new(voteRequest)
	if !unmarshal(w, r, j) {
		return
	}
	prop, err := types.NewProposalFromBytes(j.Proposal)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	if err = proposals.Add(prop, j.Approve); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	if err = proposals.SaveToFile(conf.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, j, http.StatusOK)
}

func DelVote(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	proposals := make(types.Proposals)
	if err := proposals.NewFromFile(conf.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	j := new(voteRequest)
	if !unmarshal(w, r, j) {
		return
	}
	prop, err := types.NewProposalFromBytes(j.Proposal)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	proposals.Del(prop)
	if err = proposals.SaveToFile(conf.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, j, http.StatusOK)
}

func TransactionsBySender(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsBySender(a, true, p)
	})
}

func TransactionsByRecipient(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	addrIndexer(w, r, func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI) {
		return s.GetTxsByRecipient(a, true, p)
	})
}

func Account(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetAccount(crypto.NewAddressFromBytes(a))
	})
}

func Accounts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetAccountsPaginated(p.PageParams)
	})
}

func Pool(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndNameParams(w, r, func(s *fsm.StateMachine, n string) (interface{}, lib.ErrorI) {
		return s.GetPool(types.PoolID(types.PoolID_value[n]))
	})
}

func Pools(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetPoolsPaginated(p.PageParams)
	})
}

func NonSigners(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetNonSigners()
	})
}

func Supply(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) {
		return s.GetSupply()
	})
}

func Validator(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightAndAddressParams(w, r, func(s *fsm.StateMachine, a lib.HexBytes) (interface{}, lib.ErrorI) {
		return s.GetValidator(crypto.NewAddressFromBytes(a))
	})
}

func Validators(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetValidatorsPaginated(p.PageParams, p.ValidatorFilters)
	})
}

func ConsValidators(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightPaginated(w, r, func(s *fsm.StateMachine, p *paginatedHeightRequest) (interface{}, lib.ErrorI) {
		return s.GetConsValidatorsPaginated(p.PageParams)
	})
}

func Params(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (interface{}, lib.ErrorI) { return s.GetParams() })
}

func FeeParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsFee() })
}

func ValParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsVal() })
}

func ConParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsCons() })
}

func GovParams(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.GetParamsGov() })
}

func State(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	heightParams(w, r, func(s *fsm.StateMachine) (any, lib.ErrorI) { return s.ExportState() })
}

func StateDiff(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

func Transaction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tx := new(lib.Transaction)
	if ok := unmarshal(w, r, tx); !ok {
		return
	}
	submitTx(w, tx)
}

func Keystore(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystore, err := crypto.NewKeystoreFromFile(conf.DataDirPath)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, keystore, http.StatusOK)
}

func KeystoreNewKey(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		pk, err := crypto.NewBLSPrivateKey()
		if err != nil {
			return nil, err
		}
		address, err := k.ImportRaw(pk.Bytes(), ptr.Password)
		if err != nil {
			return nil, err
		}
		return address, k.SaveToFile(conf.DataDirPath)
	})
}

func KeystoreImport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		if err := k.Import(ptr.Address, &ptr.EncryptedPrivateKey); err != nil {
			return nil, err
		}
		return ptr.Address, k.SaveToFile(conf.DataDirPath)
	})
}

func KeystoreImportRaw(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		fmt.Println(ptr.PrivateKey.String())
		address, err := k.ImportRaw(ptr.PrivateKey, ptr.Password)
		if err != nil {
			return nil, err
		}
		return address, k.SaveToFile(conf.DataDirPath)
	})
}

func KeystoreDelete(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		k.DeleteKey(ptr.Address)
		return ptr.Address, k.SaveToFile(conf.DataDirPath)
	})
}

func KeystoreGetKeyGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		return k.GetKeyGroup(ptr.Address, ptr.Password)
	})
}

type txRequest struct {
	Amount     uint64 `json:"amount"`
	NetAddress string `json:"netAddress"`
	Output     string `json:"output"`
	Sequence   uint64 `json:"sequence"`
	Fee        uint64 `json:"fee"`
	Submit     bool   `json:"submit"`
	addressRequest
	passwordRequest
	txChangeParamRequest
}

type txChangeParamRequest struct {
	ParamSpace string `json:"paramSpace"`
	ParamKey   string `json:"paramKey"`
	ParamValue string `json:"paramValue"`
	StartBlock uint64 `json:"startBlock"`
	EndBlock   uint64 `json:"endBlock"`
}

func TransactionSend(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		toAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		return types.NewSendTransaction(p, toAddress, ptr.Amount, ptr.Sequence, ptr.Fee)
	})
}

func TransactionStake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		outputAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		return types.NewStakeTx(p, outputAddress, ptr.NetAddress, ptr.Amount, ptr.Sequence, ptr.Fee)
	})
}

func TransactionEditStake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		outputAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		return types.NewEditStakeTx(p, outputAddress, ptr.NetAddress, ptr.Amount, ptr.Sequence, ptr.Fee)
	})
}

func TransactionUnstake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		return types.NewUnstakeTx(p, ptr.Sequence, ptr.Fee)
	})
}

func TransactionPause(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		return types.NewPauseTx(p, ptr.Sequence, ptr.Fee)
	})
}

func TransactionUnpause(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		return types.NewUnpauseTx(p, ptr.Sequence, ptr.Fee)
	})
}

func TransactionChangeParam(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		ptr.ParamSpace = types.FormatParamSpace(ptr.ParamSpace)
		isString, err := types.IsStringParam(ptr.ParamSpace, ptr.ParamKey)
		if err != nil {
			return nil, err
		}
		if isString {
			return types.NewChangeParamTxString(p, ptr.ParamSpace, ptr.ParamKey, ptr.ParamValue, ptr.StartBlock, ptr.EndBlock, ptr.Sequence, ptr.Fee)
		} else {
			paramValue, err := strconv.ParseUint(ptr.ParamValue, 10, 64)
			if err != nil {
				return nil, err
			}
			return types.NewChangeParamTxUint64(p, ptr.ParamSpace, ptr.ParamKey, paramValue, ptr.StartBlock, ptr.EndBlock, ptr.Sequence, ptr.Fee)
		}
	})
}

func TransactionDAOTransfer(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		return types.NewDAOTransferTx(p, ptr.Amount, ptr.StartBlock, ptr.EndBlock, ptr.Sequence, ptr.Fee)
	})
}

func ConsensusInfo(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	summary, err := app.JSONSummary()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	if _, e := w.Write(summary); e != nil {
		logger.Error(e.Error())
	}
}

func PeerInfo(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	peers, numInbound, numOutbound := app.P2P.GetAllInfos()
	write(w, &peerInfoResponse{
		ID:          app.P2P.ID(),
		NumPeers:    numInbound + numOutbound,
		NumInbound:  numInbound,
		NumOutbound: numOutbound,
		Peers:       peers,
	}, http.StatusOK)
}

func PeerBook(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	write(w, app.P2P.GetBookPeers(), http.StatusOK)
}

func Config(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	write(w, conf, http.StatusOK)
}

func FDCount(pid int32) (int, error) {
	cmd := []string{"-a", "-n", "-P", "-p", strconv.Itoa(int(pid))}
	out, err := Exec("lsof", cmd...)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(out), "\n")
	var ret []string
	for _, l := range lines[1:] {
		if len(l) == 0 {
			continue
		}
		ret = append(ret, l)
	}
	return len(ret), nil
}

func Exec(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return buf.Bytes(), err
	}

	if err := cmd.Wait(); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}

func ResourceUsage(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	pm, err := mem.VirtualMemory() // os memory
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	c, err := cpu.Times(false) // os cpu
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	cp, err := cpu.Percent(0, false) // os cpu percent
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	d, err := disk.Usage("/") // os disk
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	name, err := p.Name()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	ioCounters, err := net.IOCounters(false)
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	status, err := p.Status()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	fds, err := FDCount(p.Pid)
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	numThreads, err := p.NumThreads()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	memPercent, err := p.MemoryPercent()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	utc, err := p.CreateTime()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, resourceUsageResponse{
		Process: ProcessResourceUsage{
			Name:          name,
			Status:        status[0],
			CreateTime:    time.Unix(utc, 0).Format(time.RFC822),
			FDCount:       uint64(fds),
			ThreadCount:   uint64(numThreads),
			MemoryPercent: float64(memPercent),
			CPUPercent:    cpuPercent,
		},
		System: SystemResourceUsage{
			TotalRAM:        pm.Total,
			AvailableRAM:    pm.Available,
			UsedRAM:         pm.Used,
			UsedRAMPercent:  pm.UsedPercent,
			FreeRAM:         pm.Free,
			UsedCPUPercent:  cp[0],
			UserCPU:         c[0].User,
			SystemCPU:       c[0].System,
			IdleCPU:         c[0].Idle,
			TotalDisk:       d.Total,
			UsedDisk:        d.Used,
			UsedDiskPercent: d.UsedPercent,
			FreeDisk:        d.Free,
			ReceivedBytesIO: ioCounters[0].BytesRecv,
			WrittenBytesIO:  ioCounters[0].BytesSent,
		},
	}, http.StatusOK)
}

var (
	seqCache  = map[string]uint64{}
	seqCacheL = sync.Mutex{}
)

func resetSeqCacher(duration time.Duration) {
	height := uint64(0)
	for range time.Tick(duration) {
		func() {
			s, err := store.NewStoreWithDB(db, logger)
			if err != nil {
				logger.Error(err.Error())
				return
			}
			defer s.Discard()
			if s.Version() == height {
				return
			}
			state, err := fsm.New(conf, s, logger)
			if err != nil {
				logger.Error(err.Error())
				return
			}
			seqCacheL.Lock()
			for a, seq := range seqCache {
				addr, _ := crypto.NewAddressFromString(a)
				acc, e := state.GetAccount(addr)
				if e == nil && acc.Sequence >= seq {
					delete(seqCache, a)
				}
			}
			seqCacheL.Unlock()
			height = s.Version()
		}()
	}
}

func setSequence(state *fsm.StateMachine, address crypto.AddressI, ptr *txRequest) {
	account, err := state.GetAccount(address)
	if err == nil {
		if ptr.Sequence == 0 {
			addrString := lib.BytesToString(account.Address)
			seqCacheL.Lock()
			cachedSeq := seqCache[addrString]
			if cachedSeq > account.Sequence {
				ptr.Sequence = cachedSeq + 1
			} else {
				ptr.Sequence = account.Sequence + 1
			}
			if ptr.Submit {
				seqCache[addrString] = ptr.Sequence
			}
			seqCacheL.Unlock()
		}
	}
}

func decSequence(address crypto.AddressI) {
	addr := lib.BytesToString(address.Bytes())
	seqCacheL.Lock()
	if _, ok := seqCache[addr]; ok {
		seqCache[addr]--
	}
	seqCacheL.Unlock()
}

func txHandler(w http.ResponseWriter, r *http.Request, callback func(privateKey crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error)) {
	ptr := new(txRequest)
	if ok := unmarshal(w, r, ptr); !ok {
		return
	}
	keystore, ok := newKeystore(w)
	if !ok {
		return
	}
	privateKey, err := keystore.GetKey(ptr.Address, ptr.Password)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	state, ok := getStateMachineWithHeight(0, w)
	if !ok {
		return
	}
	if ptr.Fee == 0 {
		feeParams, e := state.GetParamsFee()
		if e != nil {
			write(w, e, http.StatusBadRequest)
			return
		}
		ptr.Fee = feeParams.MessageSendFee
	}
	addr := privateKey.PublicKey().Address()
	setSequence(state, addr, ptr)
	p, err := callback(privateKey, ptr)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	if ptr.Submit {
		if !submitTx(w, p) {
			decSequence(addr)
		}
	} else {
		bz, e := lib.MarshalJSONIndent(p)
		if e != nil {
			write(w, e, http.StatusBadRequest)
			return
		}
		if _, err = w.Write(bz); err != nil {
			logger.Error(err.Error())
			return
		}
	}
}

func keystoreHandler(w http.ResponseWriter, r *http.Request, callback func(keystore *crypto.Keystore, ptr *keystoreRequest) (any, error)) {
	keystore, ok := newKeystore(w)
	if !ok {
		return
	}
	ptr := new(keystoreRequest)
	if ok = unmarshal(w, r, ptr); !ok {
		return
	}
	p, err := callback(keystore, ptr)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndAddressParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, lib.HexBytes) (any, lib.ErrorI)) {
	req := new(heightAndAddressRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req.Address)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndNameParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, string) (any, lib.ErrorI)) {
	req := new(heightAndNameRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req.Name)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine) (any, lib.ErrorI)) {
	req := new(heightRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightPaginated(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, p *paginatedHeightRequest) (any, lib.ErrorI)) {
	req := new(paginatedHeightRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedHeightRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	if req.Height == 0 {
		req.Height = s.Version() - 1
	}
	p, err := callback(s, req.Height, req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func hashIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI)) {
	req := new(hashRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	bz, err := lib.StringToBytes(req.Hash)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	p, err := callback(s, bz)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func addrIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	p, err := callback(s, crypto.NewAddressFromBytes(req.Address), req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

type hashRequest struct {
	Hash string `json:"hash"`
}

type addressRequest struct {
	Address lib.HexBytes `json:"address"`
}

type heightRequest struct {
	Height uint64 `json:"height"`
}

type heightsRequest struct {
	heightRequest
	StartHeight uint64 `json:"startHeight"`
}

type nameRequest struct {
	Name string `json:"name"`
}
type passwordRequest struct {
	Password string `json:"password"`
}
type voteRequest struct {
	Approve  bool            `json:"approve"`
	Proposal json.RawMessage `json:"proposal"`
}

type paginatedAddressRequest struct {
	addressRequest
	lib.PageParams
}

type paginatedHeightRequest struct {
	heightRequest
	lib.PageParams
	lib.ValidatorFilters
}

type heightAndAddressRequest struct {
	heightRequest
	addressRequest
}

type heightAndNameRequest struct {
	heightRequest
	nameRequest
}

type keystoreRequest struct {
	addressRequest
	passwordRequest
	PrivateKey lib.HexBytes `json:"privateKey"`
	crypto.EncryptedPrivateKey
}

type resourceUsageResponse struct {
	Process ProcessResourceUsage `json:"process"`
	System  SystemResourceUsage  `json:"system"`
}

type peerInfoResponse struct {
	ID          *lib.PeerAddress `json:"id"`
	NumPeers    int              `json:"numPeers"`
	NumInbound  int              `json:"numInbound"`
	NumOutbound int              `json:"numOutbound"`
	Peers       []*lib.PeerInfo  `json:"peers"`
}

type ProcessResourceUsage struct {
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	CreateTime    string  `json:"createTime"`
	FDCount       uint64  `json:"fdCount"`
	ThreadCount   uint64  `json:"threadCount"`
	MemoryPercent float64 `json:"usedMemoryPercent"`
	CPUPercent    float64 `json:"usedCPUPercent"`
}

type SystemResourceUsage struct {
	// ram
	TotalRAM       uint64  `json:"totalRAM"`
	AvailableRAM   uint64  `json:"availableRAM"`
	UsedRAM        uint64  `json:"usedRAM"`
	UsedRAMPercent float64 `json:"usedRAMPercent"`
	FreeRAM        uint64  `json:"freeRAM"`
	// CPU
	UsedCPUPercent float64 `json:"usedCPUPercent"`
	UserCPU        float64 `json:"userCPU"`
	SystemCPU      float64 `json:"systemCPU"`
	IdleCPU        float64 `json:"idleCPU"`
	// disk
	TotalDisk       uint64  `json:"totalDisk"`
	UsedDisk        uint64  `json:"usedDisk"`
	UsedDiskPercent float64 `json:"usedDiskPercent"`
	FreeDisk        uint64  `json:"freeDisk"`
	// io
	ReceivedBytesIO uint64 `json:"ReceivedBytesIO"`
	WrittenBytesIO  uint64 `json:"WrittenBytesIO"`
}

func (h *heightRequest) GetHeight() uint64 {
	return h.Height
}

type queryWithHeight interface {
	GetHeight() uint64
}

func newKeystore(w http.ResponseWriter) (k *crypto.Keystore, ok bool) {
	k, err := crypto.NewKeystoreFromFile(conf.DataDirPath)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	ok = true
	return
}

func submitTx(w http.ResponseWriter, tx any) (ok bool) {
	bz, err := lib.Marshal(tx)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	if err = app.NewTx(bz); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, crypto.HashString(bz), http.StatusOK)
	return true
}

func getStateMachineFromHeightParams(w http.ResponseWriter, r *http.Request, ptr queryWithHeight) (sm *fsm.StateMachine, ok bool) {
	if ok = unmarshal(w, r, ptr); !ok {
		return
	}
	return getStateMachineWithHeight(ptr.GetHeight(), w)
}

func parseUint64FromString(s string) uint64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return uint64(i)
}

func getDoubleStateMachineFromHeightParams(w http.ResponseWriter, r *http.Request, p httprouter.Params) (sm1, sm2 *fsm.StateMachine, o *jsondiff.Options, ok bool) {
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
	sm1, ok = getStateMachineWithHeight(request.Height, w)
	if !ok {
		return
	}
	if request.StartHeight == 0 {
		request.StartHeight = sm1.Height() - 1
	}
	sm2, ok = getStateMachineWithHeight(request.StartHeight, w)
	o = &opts
	return
}

func getStateMachineWithHeight(height uint64, w http.ResponseWriter) (sm *fsm.StateMachine, ok bool) {
	s, ok := setupStore(w)
	if !ok {
		return
	}
	return setupStateMachine(height, s, w)
}

func setupStore(w http.ResponseWriter) (lib.StoreI, bool) {
	s, err := store.NewStoreWithDB(db, logger)
	if err != nil {
		write(w, ErrNewStore(err), http.StatusInternalServerError)
		return nil, false
	}
	return s, true
}

// TODO likely a memory leak here from un-discarded stores
func setupStateMachine(height uint64, s lib.StoreI, w http.ResponseWriter) (*fsm.StateMachine, bool) {
	state, err := fsm.New(conf, s, logger)
	if err != nil {
		write(w, ErrNewFSM(err), http.StatusInternalServerError)
		return nil, false
	}
	if height != 0 {
		state, err = state.TimeMachine(height)
		if err != nil {
			write(w, ErrTimeMachine(err), http.StatusInternalServerError)
		}
	}
	return state, true
}

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

func write(w http.ResponseWriter, payload interface{}, code int) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(code)
	bz, _ := json.MarshalIndent(payload, "", "  ")
	if _, err := w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}

type routes map[string]struct {
	Method      string
	Path        string
	HandlerFunc httprouter.Handle
	AdminOnly   bool
}

func (r routes) New() (router *httprouter.Router) {
	router = httprouter.New()
	for _, route := range r {
		if !route.AdminOnly {
			router.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}
	return
}

func (r routes) NewAdmin() (router *httprouter.Router) {
	router = httprouter.New()
	for _, route := range r {
		if route.AdminOnly {
			router.Handle(route.Method, route.Path, route.HandlerFunc)
		}
	}
	return
}

func debugHandler(routeName string) httprouter.Handle {
	f := func(w http.ResponseWriter, r *http.Request) {}
	switch routeName {
	case DebugHeapRouteName, DebugRoutineRouteName, DebugBlockedRouteName:
		f = func(w http.ResponseWriter, r *http.Request) {
			pprof.Handler(routeName).ServeHTTP(w, r)
		}
	case DebugCPURouteName:
		f = pprof.Profile
	}
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		f(w, r)
	}
}

func logsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		filePath := filepath.Join(conf.DataDirPath, lib.LogDirectory, lib.LogFileName)
		f, _ := os.ReadFile(filePath)
		split := bytes.Split(f, []byte("\n"))
		var flipped []byte
		for i := len(split) - 1; i >= 0; i-- {
			flipped = append(append(flipped, split[i]...), []byte("\n")...)
		}
		if _, err := w.Write(flipped); err != nil {
			logger.Error(err.Error())
		}
	}
}