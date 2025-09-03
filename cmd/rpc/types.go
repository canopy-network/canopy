package rpc

import (
	"encoding/json"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

// =====================================================
// Query Request Types
// =====================================================
type heightRequest struct {
	Height uint64 `json:"height"`
}

type chainRequest struct {
	ChainId uint64 `json:"chainId"`
}

type orderRequest struct {
	ChainId uint64 `json:"chainId"`
	OrderId string `json:"orderId"`
	heightRequest
}

type heightsRequest struct {
	heightRequest
	StartHeight uint64 `json:"startHeight"`
}

type idRequest struct {
	ID uint64 `json:"id"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

type nicknameRequest struct {
	Nickname string `json:"nickname"`
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

type heightAndIdRequest struct {
	heightRequest
	idRequest
}

type heightIdAndPointsRequest struct {
	heightAndIdRequest
	Points bool `json:"points"`
}

type keystoreRequest struct {
	addressRequest
	passwordRequest
	nicknameRequest
	PrivateKey lib.HexBytes `json:"privateKey"`
	crypto.EncryptedPrivateKey
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

type resourceUsageResponse struct {
	Process ProcessResourceUsage `json:"process"`
	System  SystemResourceUsage  `json:"system"`
}

func (h *heightRequest) GetHeight() uint64 {
	return h.Height
}

type queryWithHeight interface {
	GetHeight() uint64
}

type hashRequest struct {
	Hash string `json:"hash"`
}

type addressRequest struct {
	Address lib.HexBytes `json:"address"`
}

type committeesRequest struct {
	Committees string
}

type economicParameterResponse struct {
	MintPerBlock     uint64 `json:"MintPerBlock"`
	MintPerCommittee uint64 `json:"MintPerCommittee"`
	DAOCut           uint64 `json:"DAOCut"`
	ProposerCut      uint64 `json:"ProposerCut"`
	DelegateCut      uint64 `json:"DelegateCut"`
}

// =====================================================
// Transaction Request Types
// =====================================================
type txSend struct {
	Fee      uint64 `json:"fee"`
	Amount   uint64 `json:"amount"`
	Output   string `json:"output"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
}

type txAddress struct {
	Fee uint64 `json:"fee"`
	fromFields
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	signerFields
}

type txStake struct {
	Fee             uint64 `json:"fee"`
	Amount          uint64 `json:"amount"`
	Output          string `json:"output"`
	Delegate        bool   `json:"delegate"`
	EarlyWithdrawal bool   `json:"earlyWithdrawal"`
	NetAddress      string `json:"netAddress"`
	Submit          bool   `json:"submit"`
	Password        string `json:"password"`
	fromFields
	signerFields
	txChangeParamRequest
	committeesRequest
}

type txChangeParam struct {
	Fee      uint64 `json:"fee"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
	txChangeParamRequest
}

type txDaoTransfer struct {
	Fee      uint64 `json:"fee"`
	Amount   uint64 `json:"amount"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
	txChangeParamRequest
}

type txSubsidy struct {
	Fee      uint64 `json:"fee"`
	Amount   uint64 `json:"amount"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	OpCode   string `json:"opCode"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txCreateOrder struct {
	Fee            uint64       `json:"fee"`
	Amount         uint64       `json:"amount"`
	Password       string       `json:"password"`
	Data           lib.HexBytes `json:"data"`
	Submit         bool         `json:"submit"`
	ReceiveAmount  uint64       `json:"receiveAmount"`
	ReceiveAddress lib.HexBytes `json:"receiveAddress"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txEditOrder struct {
	Fee            uint64       `json:"fee"`
	Amount         uint64       `json:"amount"`
	Password       string       `json:"password"`
	Submit         bool         `json:"submit"`
	ReceiveAmount  uint64       `json:"receiveAmount"`
	ReceiveAddress lib.HexBytes `json:"receiveAddress"`
	OrderId        string       `json:"orderId"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txDeleteOrder struct {
	Fee      uint64 `json:"fee"`
	OrderId  string `json:"orderId"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txDexLimitOrder struct {
	Fee           uint64 `json:"fee"`
	Amount        uint64 `json:"amount"`
	ReceiveAmount uint64 `json:"receiveAmount"`
	Submit        bool   `json:"submit"`
	Password      string `json:"password"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txDexLiquidityDeposit struct {
	Fee      uint64 `json:"fee"`
	Amount   uint64 `json:"amount"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txDexLiquidityWithdraw struct {
	Fee      uint64 `json:"fee"`
	Percent  int    `json:"percent"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txLockOrder struct {
	Fee            uint64       `json:"fee"`
	OrderId        string       `json:"orderId"`
	ReceiveAddress lib.HexBytes `json:"receiveAddress"`
	Submit         bool         `json:"submit"`
	Password       string       `json:"password"`
	fromFields
	txChangeParamRequest
	committeesRequest
}

type txCloseOrder struct {
	Fee      uint64 `json:"fee"`
	OrderId  string `json:"orderId"`
	Submit   bool   `json:"submit"`
	Password string `json:"password"`
	fromFields
}

type txStartPoll struct {
	Fee      uint64          `json:"fee"`
	PollJSON json.RawMessage `json:"pollJSON"`
	Submit   bool            `json:"submit"`
	Password string          `json:"password"`
	fromFields
}

type txVotePoll struct {
	Fee         uint64          `json:"fee"`
	PollJSON    json.RawMessage `json:"pollJSON"`
	PollApprove bool            `json:"pollApprove"`
	Submit      bool            `json:"submit"`
	Password    string          `json:"password"`
	fromFields
}

type txChangeParamRequest struct {
	ParamSpace string `json:"paramSpace"`
	ParamKey   string `json:"paramKey"`
	ParamValue string `json:"paramValue"`
	StartBlock uint64 `json:"startBlock"`
	EndBlock   uint64 `json:"endBlock"`
}

// fromFields contains the address and/or nickname for the from fields
type fromFields struct {
	Address  lib.HexBytes `json:"address"`
	Nickname string       `json:"nickname"`
}

// signerFields contains the signer address and/or nickname for the signer fields
type signerFields struct {
	Signer         lib.HexBytes `json:"signer"`
	SignerNickname string       `json:"signerNickname"`
}

// txRequest is used server side to unmarshall all transaction requests
type txRequest struct {
	Amount          uint64          `json:"amount"`
	PubKey          string          `json:"pubKey"`
	NetAddress      string          `json:"netAddress"`
	Output          string          `json:"output"`
	OpCode          lib.HexBytes    `json:"opCode"`
	Data            lib.HexBytes    `json:"data"`
	Fee             uint64          `json:"fee"`
	Delegate        bool            `json:"delegate"`
	EarlyWithdrawal bool            `json:"earlyWithdrawal"`
	Submit          bool            `json:"submit"`
	ReceiveAmount   uint64          `json:"receiveAmount"`
	ReceiveAddress  lib.HexBytes    `json:"receiveAddress"`
	Percent         int             `json:"percent"`
	OrderId         string          `json:"orderId"`
	Memo            string          `json:"memo"`
	PollJSON        json.RawMessage `json:"pollJSON"`
	PollApprove     bool            `json:"pollApprove"`
	Signer          lib.HexBytes    `json:"signer"`
	SignerNickname  string          `json:"signerNickname"`
	addressRequest
	nicknameRequest
	passwordRequest
	txChangeParamRequest
	committeesRequest
}
