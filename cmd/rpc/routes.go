package rpc

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const (
	VersionRouteName               = "version"
	TxRouteName                    = "tx"
	HeightRouteName                = "height"
	AccountRouteName               = "account"
	AccountsRouteName              = "accounts"
	PoolRouteName                  = "pool"
	PoolsRouteName                 = "pools"
	ValidatorRouteName             = "validator"
	ValidatorsRouteName            = "validators"
	NonSignersRouteName            = "non-signers"
	SupplyRouteName                = "supply"
	ParamRouteName                 = "params"
	FeeParamRouteName              = "fee-params"
	GovParamRouteName              = "gov-params"
	ConParamsRouteName             = "con-params"
	ValParamRouteName              = "val-params"
	StateRouteName                 = "state"
	StateDiffRouteName             = "state-diff"
	StateDiffGetRouteName          = "state-diff-get"
	CertByHeightRouteName          = "cert-by-height"
	BlocksRouteName                = "blocks"
	BlockByHeightRouteName         = "block-by-height"
	BlockByHashRouteName           = "block-by-hash"
	TxsByHeightRouteName           = "txs-by-height"
	TxsBySenderRouteName           = "txs-by-sender"
	TxsByRecRouteName              = "txs-by-rec"
	TxByHashRouteName              = "tx-by-hash"
	PendingRouteName               = "pending"
	FailedTxRouteName              = "failed-txs"
	ProposalsRouteName             = "proposals"
	PollRouteName                  = "poll"
	CommitteeRouteName             = "committee"
	CommitteeDataRouteName         = "committee-data"
	CommitteesDataRouteName        = "committees-data"
	SubsidizedCommitteesRouteName  = "subsidized-committees"
	RetiredCommitteesRouteName     = "retired-committees"
	OrderRouteName                 = "order"
	OrdersRouteName                = "orders"
	LastProposersRouteName         = "last-proposers"
	IsValidDoubleSignerRouteName   = "valid-double-signer"
	DoubleSignersRouteName         = "double-signers"
	MinimumEvidenceHeightRouteName = "minimum-evidence-height"
	LotteryRouteName               = "lottery"
	RootChainInfoRouteName         = "root-Chain-info"
	ValidatorSetRouteName          = "validator-set"
	CheckpointRouteName            = "checkpoint"
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
	TxSubsidyRouteName         = "tx-subsidy"
	TxCreateOrderRouteName     = "tx-create-order"
	TxEditOrderRouteName       = "tx-edit-order"
	TxDeleteOrderRouteName     = "tx-delete-order"
	TxBuyOrderRouteName        = "tx-buy-order"
	TxStartPollRouteName       = "tx-start-poll"
	TxVotePollRouteName        = "tx-vote-poll"
	ResourceUsageRouteName     = "resource-usage"
	PeerInfoRouteName          = "peer-info"
	ConsensusInfoRouteName     = "consensus-info"
	PeerBookRouteName          = "peer-book"
	ConfigRouteName            = "config"
	LogsRouteName              = "logs"
	AddVoteRouteName           = "add-vote"
	DelVoteRouteName           = "del-vote"
)

type routes map[string]struct {
	Method string
	Path   string
}

// routePaths contain a mapping from route names to their methods & paths
// TODO Split this into routes & admin routes
var routePaths = routes{
	VersionRouteName:               {Method: http.MethodGet, Path: "/v1/"},
	TxRouteName:                    {Method: http.MethodPost, Path: "/v1/tx"},
	HeightRouteName:                {Method: http.MethodPost, Path: "/v1/query/height"},
	AccountRouteName:               {Method: http.MethodPost, Path: "/v1/query/account"},
	AccountsRouteName:              {Method: http.MethodPost, Path: "/v1/query/accounts"},
	PoolRouteName:                  {Method: http.MethodPost, Path: "/v1/query/pool"},
	PoolsRouteName:                 {Method: http.MethodPost, Path: "/v1/query/pools"},
	ValidatorRouteName:             {Method: http.MethodPost, Path: "/v1/query/validator"},
	ValidatorsRouteName:            {Method: http.MethodPost, Path: "/v1/query/validators"},
	CommitteeRouteName:             {Method: http.MethodPost, Path: "/v1/query/committee"},
	CommitteeDataRouteName:         {Method: http.MethodPost, Path: "/v1/query/committee-data"},
	CommitteesDataRouteName:        {Method: http.MethodPost, Path: "/v1/query/committees-data"},
	SubsidizedCommitteesRouteName:  {Method: http.MethodPost, Path: "/v1/query/subsidized-committees"},
	RetiredCommitteesRouteName:     {Method: http.MethodPost, Path: "/v1/query/retired-committees"},
	NonSignersRouteName:            {Method: http.MethodPost, Path: "/v1/query/non-signers"},
	ParamRouteName:                 {Method: http.MethodPost, Path: "/v1/query/params"},
	SupplyRouteName:                {Method: http.MethodPost, Path: "/v1/query/supply"},
	FeeParamRouteName:              {Method: http.MethodPost, Path: "/v1/query/fee-params"},
	GovParamRouteName:              {Method: http.MethodPost, Path: "/v1/query/gov-params"},
	ConParamsRouteName:             {Method: http.MethodPost, Path: "/v1/query/con-params"},
	ValParamRouteName:              {Method: http.MethodPost, Path: "/v1/query/val-params"},
	StateRouteName:                 {Method: http.MethodGet, Path: "/v1/query/state"},
	StateDiffRouteName:             {Method: http.MethodPost, Path: "/v1/query/state-diff"},
	StateDiffGetRouteName:          {Method: http.MethodGet, Path: "/v1/query/state-diff"},
	CertByHeightRouteName:          {Method: http.MethodPost, Path: "/v1/query/cert-by-height"},
	BlockByHeightRouteName:         {Method: http.MethodPost, Path: "/v1/query/block-by-height"},
	BlocksRouteName:                {Method: http.MethodPost, Path: "/v1/query/blocks"},
	BlockByHashRouteName:           {Method: http.MethodPost, Path: "/v1/query/block-by-hash"},
	TxsByHeightRouteName:           {Method: http.MethodPost, Path: "/v1/query/txs-by-height"},
	TxsBySenderRouteName:           {Method: http.MethodPost, Path: "/v1/query/txs-by-sender"},
	TxsByRecRouteName:              {Method: http.MethodPost, Path: "/v1/query/txs-by-rec"},
	TxByHashRouteName:              {Method: http.MethodPost, Path: "/v1/query/tx-by-hash"},
	OrderRouteName:                 {Method: http.MethodPost, Path: "/v1/query/order"},
	OrdersRouteName:                {Method: http.MethodPost, Path: "/v1/query/orders"},
	LastProposersRouteName:         {Method: http.MethodPost, Path: "/v1/query/last-proposers"},
	IsValidDoubleSignerRouteName:   {Method: http.MethodPost, Path: "/v1/query/valid-double-signer"},
	DoubleSignersRouteName:         {Method: http.MethodPost, Path: "/v1/query/double-signers"},
	MinimumEvidenceHeightRouteName: {Method: http.MethodPost, Path: "/v1/query/minimum-evidence-height"},
	LotteryRouteName:               {Method: http.MethodPost, Path: "/v1/query/lottery"},
	PendingRouteName:               {Method: http.MethodPost, Path: "/v1/query/pending"},
	FailedTxRouteName:              {Method: http.MethodPost, Path: "/v1/query/failed-txs"},
	ProposalsRouteName:             {Method: http.MethodGet, Path: "/v1/gov/proposals"},
	PollRouteName:                  {Method: http.MethodGet, Path: "/v1/gov/poll"},
	RootChainInfoRouteName:         {Method: http.MethodPost, Path: "/v1/query/root-Chain-info"},
	ValidatorSetRouteName:          {Method: http.MethodPost, Path: "/v1/query/validator-set"},
	CheckpointRouteName:            {Method: http.MethodPost, Path: "/v1/query/checkpoint"},
	// debug
	DebugBlockedRouteName: {Method: http.MethodPost, Path: "/debug/blocked"},
	DebugHeapRouteName:    {Method: http.MethodPost, Path: "/debug/heap"},
	DebugCPURouteName:     {Method: http.MethodPost, Path: "/debug/cpu"},
	DebugRoutineRouteName: {Method: http.MethodPost, Path: "/debug/routine"},
	// admin
	KeystoreRouteName:          {Method: http.MethodGet, Path: "/v1/admin/keystore"},
	KeystoreNewKeyRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-new-key"},
	KeystoreImportRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-import"},
	KeystoreImportRawRouteName: {Method: http.MethodPost, Path: "/v1/admin/keystore-import-raw"},
	KeystoreDeleteRouteName:    {Method: http.MethodPost, Path: "/v1/admin/keystore-delete"},
	KeystoreGetRouteName:       {Method: http.MethodPost, Path: "/v1/admin/keystore-get"},
	TxSendRouteName:            {Method: http.MethodPost, Path: "/v1/admin/tx-send"},
	TxStakeRouteName:           {Method: http.MethodPost, Path: "/v1/admin/tx-stake"},
	TxEditStakeRouteName:       {Method: http.MethodPost, Path: "/v1/admin/tx-edit-stake"},
	TxUnstakeRouteName:         {Method: http.MethodPost, Path: "/v1/admin/tx-unstake"},
	TxPauseRouteName:           {Method: http.MethodPost, Path: "/v1/admin/tx-pause"},
	TxUnpauseRouteName:         {Method: http.MethodPost, Path: "/v1/admin/tx-unpause"},
	TxChangeParamRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-change-param"},
	TxDAOTransferRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-dao-transfer"},
	TxCreateOrderRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-create-order"},
	TxEditOrderRouteName:       {Method: http.MethodPost, Path: "/v1/admin/tx-edit-order"},
	TxDeleteOrderRouteName:     {Method: http.MethodPost, Path: "/v1/admin/tx-delete-order"},
	TxBuyOrderRouteName:        {Method: http.MethodPost, Path: "/v1/admin/tx-buy-order"},
	TxSubsidyRouteName:         {Method: http.MethodPost, Path: "/v1/admin/subsidy"},
	TxStartPollRouteName:       {Method: http.MethodPost, Path: "/v1/admin/tx-start-poll"},
	TxVotePollRouteName:        {Method: http.MethodPost, Path: "/v1/admin/tx-vote-poll"},
	ResourceUsageRouteName:     {Method: http.MethodGet, Path: "/v1/admin/resource-usage"},
	PeerInfoRouteName:          {Method: http.MethodGet, Path: "/v1/admin/peer-info"},
	ConsensusInfoRouteName:     {Method: http.MethodGet, Path: "/v1/admin/consensus-info"},
	PeerBookRouteName:          {Method: http.MethodGet, Path: "/v1/admin/peer-book"},
	ConfigRouteName:            {Method: http.MethodGet, Path: "/v1/admin/config"},
	LogsRouteName:              {Method: http.MethodGet, Path: "/v1/admin/log"},
	AddVoteRouteName:           {Method: http.MethodPost, Path: "/v1/gov/add-vote"},
	DelVoteRouteName:           {Method: http.MethodPost, Path: "/v1/gov/del-vote"},
}

type httpRouteHandlers map[string]httprouter.Handle

func createRouter(s *Server) *httprouter.Router {
	var r = httpRouteHandlers{
		VersionRouteName:               s.Version,
		TxRouteName:                    s.Transaction,
		HeightRouteName:                s.Height,
		AccountRouteName:               s.Account,
		AccountsRouteName:              s.Accounts,
		PoolRouteName:                  s.Pool,
		PoolsRouteName:                 s.Pools,
		ValidatorRouteName:             s.Validator,
		ValidatorsRouteName:            s.Validators,
		CommitteeRouteName:             s.Committee,
		CommitteeDataRouteName:         s.CommitteeData,
		CommitteesDataRouteName:        s.CommitteesData,
		SubsidizedCommitteesRouteName:  s.SubsidizedCommittees,
		RetiredCommitteesRouteName:     s.RetiredCommittees,
		NonSignersRouteName:            s.NonSigners,
		ParamRouteName:                 s.Params,
		SupplyRouteName:                s.Supply,
		FeeParamRouteName:              s.FeeParams,
		GovParamRouteName:              s.GovParams,
		ConParamsRouteName:             s.ConParams,
		ValParamRouteName:              s.ValParams,
		StateRouteName:                 s.State,
		StateDiffRouteName:             s.StateDiff,
		StateDiffGetRouteName:          s.StateDiff,
		CertByHeightRouteName:          s.CertByHeight,
		BlockByHeightRouteName:         s.BlockByHeight,
		BlocksRouteName:                s.Blocks,
		BlockByHashRouteName:           s.BlockByHash,
		TxsByHeightRouteName:           s.TransactionsByHeight,
		TxsBySenderRouteName:           s.TransactionsBySender,
		TxsByRecRouteName:              s.TransactionsByRecipient,
		TxByHashRouteName:              s.TransactionByHash,
		OrderRouteName:                 s.Order,
		OrdersRouteName:                s.Orders,
		LastProposersRouteName:         s.LastProposers,
		IsValidDoubleSignerRouteName:   s.IsValidDoubleSigner,
		DoubleSignersRouteName:         s.DoubleSigners,
		MinimumEvidenceHeightRouteName: s.MinimumEvidenceHeight,
		LotteryRouteName:               s.Lottery,
		PendingRouteName:               s.Pending,
		FailedTxRouteName:              s.FailedTxs,
		ProposalsRouteName:             s.Proposals,
		PollRouteName:                  s.Poll,
		RootChainInfoRouteName:         s.RootChainInfo,
		ValidatorSetRouteName:          s.ValidatorSet,
		CheckpointRouteName:            s.Checkpoint,
	}

	router := httprouter.New()

	for name, handler := range r {
		path := routePaths[name]
		router.Handle(path.Method, path.Path, logHandler{path.Path, handler}.Handle)
	}

	return router
}

func createAdminRouter(s *Server) *httprouter.Router {
	var r = httpRouteHandlers{
		KeystoreRouteName:          s.Keystore,
		KeystoreNewKeyRouteName:    s.KeystoreNewKey,
		KeystoreImportRouteName:    s.KeystoreImport,
		KeystoreImportRawRouteName: s.KeystoreImportRaw,
		KeystoreDeleteRouteName:    s.KeystoreDelete,
		KeystoreGetRouteName:       s.KeystoreGetKeyGroup,
		TxSendRouteName:            s.TransactionSend,
		TxStakeRouteName:           s.TransactionStake,
		TxEditStakeRouteName:       s.TransactionEditStake,
		TxUnstakeRouteName:         s.TransactionUnstake,
		TxPauseRouteName:           s.TransactionPause,
		TxUnpauseRouteName:         s.TransactionUnpause,
		TxChangeParamRouteName:     s.TransactionChangeParam,
		TxDAOTransferRouteName:     s.TransactionDAOTransfer,
		TxCreateOrderRouteName:     s.TransactionCreateOrder,
		TxEditOrderRouteName:       s.TransactionEditOrder,
		TxDeleteOrderRouteName:     s.TransactionDeleteOrder,
		TxBuyOrderRouteName:        s.TransactionBuyOrder,
		TxSubsidyRouteName:         s.TransactionSubsidy,
		TxStartPollRouteName:       s.TransactionStartPoll,
		TxVotePollRouteName:        s.TransactionVotePoll,
		ResourceUsageRouteName:     s.ResourceUsage,
		PeerInfoRouteName:          s.PeerInfo,
		ConsensusInfoRouteName:     s.ConsensusInfo,
		PeerBookRouteName:          s.PeerBook,
		ConfigRouteName:            s.Config,
		LogsRouteName:              logsHandler(s),
		AddVoteRouteName:           s.AddVote,
		DelVoteRouteName:           s.DelVote,
		// debug
		DebugBlockedRouteName: debugHandler(DebugBlockedRouteName),
		DebugHeapRouteName:    debugHandler(DebugHeapRouteName),
		DebugCPURouteName:     debugHandler(DebugHeapRouteName),
		DebugRoutineRouteName: debugHandler(DebugRoutineRouteName),
	}

	router := httprouter.New()

	for name, handler := range r {
		path := routePaths[name]
		router.Handle(path.Method, path.Path, logHandler{path.Path, handler}.Handle)
	}

	return router
}
