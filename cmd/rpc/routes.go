package rpc

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Canopy RPC Paths
const (
	VersionRoutePath               = "/v1/"
	TxRoutePath                    = "/v1/tx"
	HeightRoutePath                = "/v1/query/height"
	AccountRoutePath               = "/v1/query/account"
	AccountsRoutePath              = "/v1/query/accounts"
	PoolRoutePath                  = "/v1/query/pool"
	PoolsRoutePath                 = "/v1/query/pools"
	ValidatorRoutePath             = "/v1/query/validator"
	ValidatorsRoutePath            = "/v1/query/validators"
	CommitteeRoutePath             = "/v1/query/committee"
	CommitteeDataRoutePath         = "/v1/query/committee-data"
	CommitteesDataRoutePath        = "/v1/query/committees-data"
	SubsidizedCommitteesRoutePath  = "/v1/query/subsidized-committees"
	RetiredCommitteesRoutePath     = "/v1/query/retired-committees"
	NonSignersRoutePath            = "/v1/query/non-signers"
	ParamRoutePath                 = "/v1/query/params"
	SupplyRoutePath                = "/v1/query/supply"
	FeeParamRoutePath              = "/v1/query/fee-params"
	GovParamRoutePath              = "/v1/query/gov-params"
	ConParamsRoutePath             = "/v1/query/con-params"
	ValParamRoutePath              = "/v1/query/val-params"
	StateRoutePath                 = "/v1/query/state"
	StateDiffRoutePath             = "/v1/query/state-diff"
	StateDiffGetRoutePath          = "/v1/query/state-diff"
	CertByHeightRoutePath          = "/v1/query/cert-by-height"
	BlockByHeightRoutePath         = "/v1/query/block-by-height"
	BlocksRoutePath                = "/v1/query/blocks"
	BlockByHashRoutePath           = "/v1/query/block-by-hash"
	TxsByHeightRoutePath           = "/v1/query/txs-by-height"
	TxsBySenderRoutePath           = "/v1/query/txs-by-sender"
	TxsByRecRoutePath              = "/v1/query/txs-by-rec"
	TxByHashRoutePath              = "/v1/query/tx-by-hash"
	OrderRoutePath                 = "/v1/query/order"
	OrdersRoutePath                = "/v1/query/orders"
	LastProposersRoutePath         = "/v1/query/last-proposers"
	IsValidDoubleSignerRoutePath   = "/v1/query/valid-double-signer"
	DoubleSignersRoutePath         = "/v1/query/double-signers"
	MinimumEvidenceHeightRoutePath = "/v1/query/minimum-evidence-height"
	LotteryRoutePath               = "/v1/query/lottery"
	PendingRoutePath               = "/v1/query/pending"
	FailedTxRoutePath              = "/v1/query/failed-txs"
	ProposalsRoutePath             = "/v1/gov/proposals"
	PollRoutePath                  = "/v1/gov/poll"
	RootChainInfoRoutePath         = "/v1/query/root-Chain-info"
	ValidatorSetRoutePath          = "/v1/query/validator-set"
	CheckpointRoutePath            = "/v1/query/checkpoint"
	// debug
	DebugBlockedRoutePath = "/debug/blocked"
	DebugHeapRoutePath    = "/debug/heap"
	DebugCPURoutePath     = "/debug/cpu"
	DebugRoutineRoutePath = "/debug/routine"
	// admin
	KeystoreRoutePath          = "/v1/admin/keystore"
	KeystoreNewKeyRoutePath    = "/v1/admin/keystore-new-key"
	KeystoreImportRoutePath    = "/v1/admin/keystore-import"
	KeystoreImportRawRoutePath = "/v1/admin/keystore-import-raw"
	KeystoreDeleteRoutePath    = "/v1/admin/keystore-delete"
	KeystoreGetRoutePath       = "/v1/admin/keystore-get"
	TxSendRoutePath            = "/v1/admin/tx-send"
	TxStakeRoutePath           = "/v1/admin/tx-stake"
	TxEditStakeRoutePath       = "/v1/admin/tx-edit-stake"
	TxUnstakeRoutePath         = "/v1/admin/tx-unstake"
	TxPauseRoutePath           = "/v1/admin/tx-pause"
	TxUnpauseRoutePath         = "/v1/admin/tx-unpause"
	TxChangeParamRoutePath     = "/v1/admin/tx-change-param"
	TxDAOTransferRoutePath     = "/v1/admin/tx-dao-transfer"
	TxCreateOrderRoutePath     = "/v1/admin/tx-create-order"
	TxEditOrderRoutePath       = "/v1/admin/tx-edit-order"
	TxDeleteOrderRoutePath     = "/v1/admin/tx-delete-order"
	TxLockOrderRoutePath       = "/v1/admin/tx-buy-order"
	TxSubsidyRoutePath         = "/v1/admin/subsidy"
	TxStartPollRoutePath       = "/v1/admin/tx-start-poll"
	TxVotePollRoutePath        = "/v1/admin/tx-vote-poll"
	ResourceUsageRoutePath     = "/v1/admin/resource-usage"
	PeerInfoRoutePath          = "/v1/admin/peer-info"
	ConsensusInfoRoutePath     = "/v1/admin/consensus-info"
	PeerBookRoutePath          = "/v1/admin/peer-book"
	ConfigRoutePath            = "/v1/admin/config"
	LogsRoutePath              = "/v1/admin/log"
	AddVoteRoutePath           = "/v1/gov/add-vote"
	DelVoteRoutePath           = "/v1/gov/del-vote"
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
	TxLockOrderRouteName       = "tx-lock-order"
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

// routes contains the method and path for a canopy command
type routes map[string]struct {
	Method string
	Path   string
}

// routePaths contain a mapping from route names to their methods & paths
// TODO Split this into routes & admin routes
var routePaths = routes{
	VersionRouteName:               {Method: http.MethodGet, Path: VersionRoutePath},
	TxRouteName:                    {Method: http.MethodPost, Path: TxRoutePath},
	HeightRouteName:                {Method: http.MethodPost, Path: HeightRoutePath},
	AccountRouteName:               {Method: http.MethodPost, Path: AccountRoutePath},
	AccountsRouteName:              {Method: http.MethodPost, Path: AccountsRoutePath},
	PoolRouteName:                  {Method: http.MethodPost, Path: PoolRoutePath},
	PoolsRouteName:                 {Method: http.MethodPost, Path: PoolsRoutePath},
	ValidatorRouteName:             {Method: http.MethodPost, Path: ValidatorRoutePath},
	ValidatorsRouteName:            {Method: http.MethodPost, Path: ValidatorsRoutePath},
	CommitteeRouteName:             {Method: http.MethodPost, Path: CommitteeRoutePath},
	CommitteeDataRouteName:         {Method: http.MethodPost, Path: CommitteeDataRoutePath},
	CommitteesDataRouteName:        {Method: http.MethodPost, Path: CommitteesDataRoutePath},
	SubsidizedCommitteesRouteName:  {Method: http.MethodPost, Path: SubsidizedCommitteesRoutePath},
	RetiredCommitteesRouteName:     {Method: http.MethodPost, Path: RetiredCommitteesRoutePath},
	NonSignersRouteName:            {Method: http.MethodPost, Path: NonSignersRoutePath},
	ParamRouteName:                 {Method: http.MethodPost, Path: ParamRoutePath},
	SupplyRouteName:                {Method: http.MethodPost, Path: SupplyRoutePath},
	FeeParamRouteName:              {Method: http.MethodPost, Path: FeeParamRoutePath},
	GovParamRouteName:              {Method: http.MethodPost, Path: GovParamRoutePath},
	ConParamsRouteName:             {Method: http.MethodPost, Path: ConParamsRoutePath},
	ValParamRouteName:              {Method: http.MethodPost, Path: ValParamRoutePath},
	StateRouteName:                 {Method: http.MethodGet, Path: StateRoutePath},
	StateDiffRouteName:             {Method: http.MethodPost, Path: StateDiffRoutePath},
	StateDiffGetRouteName:          {Method: http.MethodGet, Path: StateDiffGetRoutePath},
	CertByHeightRouteName:          {Method: http.MethodPost, Path: CertByHeightRoutePath},
	BlockByHeightRouteName:         {Method: http.MethodPost, Path: BlockByHeightRoutePath},
	BlocksRouteName:                {Method: http.MethodPost, Path: BlocksRoutePath},
	BlockByHashRouteName:           {Method: http.MethodPost, Path: BlockByHashRoutePath},
	TxsByHeightRouteName:           {Method: http.MethodPost, Path: TxsByHeightRoutePath},
	TxsBySenderRouteName:           {Method: http.MethodPost, Path: TxsBySenderRoutePath},
	TxsByRecRouteName:              {Method: http.MethodPost, Path: TxsByRecRoutePath},
	TxByHashRouteName:              {Method: http.MethodPost, Path: TxByHashRoutePath},
	OrderRouteName:                 {Method: http.MethodPost, Path: OrderRoutePath},
	OrdersRouteName:                {Method: http.MethodPost, Path: OrdersRoutePath},
	LastProposersRouteName:         {Method: http.MethodPost, Path: LastProposersRoutePath},
	IsValidDoubleSignerRouteName:   {Method: http.MethodPost, Path: IsValidDoubleSignerRoutePath},
	DoubleSignersRouteName:         {Method: http.MethodPost, Path: DoubleSignersRoutePath},
	MinimumEvidenceHeightRouteName: {Method: http.MethodPost, Path: MinimumEvidenceHeightRoutePath},
	LotteryRouteName:               {Method: http.MethodPost, Path: LotteryRoutePath},
	PendingRouteName:               {Method: http.MethodPost, Path: PendingRoutePath},
	FailedTxRouteName:              {Method: http.MethodPost, Path: FailedTxRoutePath},
	ProposalsRouteName:             {Method: http.MethodGet, Path: ProposalsRoutePath},
	PollRouteName:                  {Method: http.MethodGet, Path: PollRoutePath},
	RootChainInfoRouteName:         {Method: http.MethodPost, Path: RootChainInfoRoutePath},
	ValidatorSetRouteName:          {Method: http.MethodPost, Path: ValidatorSetRoutePath},
	CheckpointRouteName:            {Method: http.MethodPost, Path: CheckpointRoutePath},
	// debug
	DebugBlockedRouteName: {Method: http.MethodPost, Path: DebugBlockedRoutePath},
	DebugHeapRouteName:    {Method: http.MethodPost, Path: DebugHeapRoutePath},
	DebugCPURouteName:     {Method: http.MethodPost, Path: DebugCPURoutePath},
	DebugRoutineRouteName: {Method: http.MethodPost, Path: DebugRoutineRoutePath},
	// admin
	KeystoreRouteName:          {Method: http.MethodGet, Path: KeystoreRoutePath},
	KeystoreNewKeyRouteName:    {Method: http.MethodPost, Path: KeystoreNewKeyRoutePath},
	KeystoreImportRouteName:    {Method: http.MethodPost, Path: KeystoreImportRoutePath},
	KeystoreImportRawRouteName: {Method: http.MethodPost, Path: KeystoreImportRawRoutePath},
	KeystoreDeleteRouteName:    {Method: http.MethodPost, Path: KeystoreDeleteRoutePath},
	KeystoreGetRouteName:       {Method: http.MethodPost, Path: KeystoreGetRoutePath},
	TxSendRouteName:            {Method: http.MethodPost, Path: TxSendRoutePath},
	TxStakeRouteName:           {Method: http.MethodPost, Path: TxStakeRoutePath},
	TxEditStakeRouteName:       {Method: http.MethodPost, Path: TxEditOrderRoutePath},
	TxUnstakeRouteName:         {Method: http.MethodPost, Path: TxUnstakeRoutePath},
	TxPauseRouteName:           {Method: http.MethodPost, Path: TxPauseRoutePath},
	TxUnpauseRouteName:         {Method: http.MethodPost, Path: TxUnpauseRoutePath},
	TxChangeParamRouteName:     {Method: http.MethodPost, Path: TxChangeParamRoutePath},
	TxDAOTransferRouteName:     {Method: http.MethodPost, Path: TxDAOTransferRoutePath},
	TxCreateOrderRouteName:     {Method: http.MethodPost, Path: TxCreateOrderRoutePath},
	TxEditOrderRouteName:       {Method: http.MethodPost, Path: TxEditStakeRoutePath},
	TxDeleteOrderRouteName:     {Method: http.MethodPost, Path: TxDeleteOrderRoutePath},
	TxLockOrderRouteName:       {Method: http.MethodPost, Path: TxLockOrderRoutePath},
	TxSubsidyRouteName:         {Method: http.MethodPost, Path: TxSubsidyRoutePath},
	TxStartPollRouteName:       {Method: http.MethodPost, Path: TxStartPollRoutePath},
	TxVotePollRouteName:        {Method: http.MethodPost, Path: TxVotePollRoutePath},
	ResourceUsageRouteName:     {Method: http.MethodGet, Path: ResourceUsageRoutePath},
	PeerInfoRouteName:          {Method: http.MethodGet, Path: PeerInfoRoutePath},
	ConsensusInfoRouteName:     {Method: http.MethodGet, Path: ConsensusInfoRoutePath},
	PeerBookRouteName:          {Method: http.MethodGet, Path: PeerBookRoutePath},
	ConfigRouteName:            {Method: http.MethodGet, Path: ConfigRoutePath},
	LogsRouteName:              {Method: http.MethodGet, Path: LogsRoutePath},
	AddVoteRouteName:           {Method: http.MethodPost, Path: AddVoteRoutePath},
	DelVoteRouteName:           {Method: http.MethodPost, Path: DelVoteRoutePath},
}

type httpRouteHandlers map[string]httprouter.Handle

// createRouter creates a route handler map for query routes
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

// createRouter creates a route handler map for admin routes
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
		TxLockOrderRouteName:       s.TransactionLockOrder,
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
