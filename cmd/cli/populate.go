package cli

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	PrivateKeyHex  = "6c275055a4f6ae6bccf1e6552e172c7b8cc538a7b8d2dd645125df9e25c9ed2d"
	PrivateKeyHex2 = "0add1e53f45d439b98182216a9215db238aaff12d1b057b3130b0c5e8e9b0b36"
	PrivateKeyHex3 = "3e9eb7a46defde2fa42dc74d85d0ee8cebaf327efa24b972e97b54af38d6e602"
	Addr2          = "02cd4e5eb53ea665702042a6ed6d31d616054dc5"
	Addr3          = "6f94783856d5ce46d24dd5946215086211d70776"

	RecipientAddress = Addr2
	DelegateAddress  = Addr3

	BaseRPCURL                 = "http://node-1:50002"
	BaseAdminRPCURL            = "http://node-1:50003"
	BaseAdminRPCURL2           = "http://node-2:40003"
	SendTxURL                  = BaseAdminRPCURL + "/v1/admin/tx-send"
	StakeTxURL                 = BaseAdminRPCURL + "/v1/admin/tx-stake"
	EditStakeTxURL             = BaseAdminRPCURL + "/v1/admin/tx-edit-stake"
	PauseTxURL                 = BaseAdminRPCURL + "/v1/admin/tx-pause"
	UnpauseTxURL               = BaseAdminRPCURL + "/v1/admin/tx-unpause"
	UnstakeTxURL               = BaseAdminRPCURL + "/v1/admin/tx-unstake"
	ChangeParamTxURL           = BaseAdminRPCURL + "/v1/admin/tx-change-param"
	DAOTransferTxURL           = BaseAdminRPCURL + "/v1/admin/tx-dao-transfer"
	SubsidyTxURL               = BaseAdminRPCURL + "/v1/admin/tx-subsidy"
	CreateOrderTxURL           = BaseAdminRPCURL + "/v1/admin/tx-create-order"
	EditOrderTxURL             = BaseAdminRPCURL + "/v1/admin/tx-edit-order"
	DeleteOrderTxURL           = BaseAdminRPCURL + "/v1/admin/tx-delete-order"
	TxDexLimitOrderTxURL       = BaseAdminRPCURL + "/v1/admin/tx-dex-limit-order"
	TxDexLiquidityDepositTxURL = BaseAdminRPCURL + "/v1/admin/tx-dex-liquidity-deposit"
	TxDexLiqudityWithdrawTxURL = BaseAdminRPCURL + "/v1/admin/tx-dex-liquidity-withdraw"
	TxLockOrderTxURL           = BaseAdminRPCURL2 + "/v1/admin/tx-lock-order"
	TxCloseOrderTxURL          = BaseAdminRPCURL2 + "/v1/admin/tx-close-order"
	TxStartPollTxURL           = BaseAdminRPCURL + "/v1/admin/tx-start-poll"
	TxVotePollTxURL            = BaseAdminRPCURL + "/v1/admin/tx-vote-poll"

	QueryHeightURL = BaseRPCURL + "/v1/query/height"

	Pwd = "test"

	Amount = 1000
)

func Populate() {
	WaitBlocks(1)
	fmt.Println("DANGEROUS: DELETE AUTO-PROPOSAL ACCEPTANCE BEFORE MERGING")
	// This data population program executes all 13 transaction types in various scenarios over ~ 10 blocks
	// - Acc: Send
	// - Val: Stake
	// - Val: EditStake
	// - Val: Pause
	// - Val: Unpause
	// - Val: Unstake
	// - Gov: Subsidy
	// - Gov: DAO Transfer
	// - Gov: Change param
	// - Gov: Start Poll
	// - Gov: Vote Poll
	// - Swp: Create Order
	// - Swp: Edit Order
	// - Swp: Delete Order
	// - Dex: Limit Order
	// - Dex: Liquidity Deposit
	// - Dex: Liquidity Withdrawal

	// STEP 1:
	// - Three private keys are loaded.
	// - Corresponding public addresses (`address`, `address2`, `address3`) are extracted.
	// - These addresses are used as senders for different transactions.
	// NOTE: these are the default keys in the keystore. This program will not work if they aren't in the keystore.

	pk, _ := crypto.NewPrivateKeyFromString(PrivateKeyHex)
	pk2, _ := crypto.NewPrivateKeyFromString(PrivateKeyHex2)
	pk3, _ := crypto.NewPrivateKeyFromString(PrivateKeyHex3)

	address := pk.PublicKey().Address()
	address2 := pk2.PublicKey().Address()
	address3 := pk3.PublicKey().Address()

	// STEP 2:
	// 1. DoSendTx(address) – Sends tokens from `address`.
	// 2. DoChangeParamTx(address) – Changes some blockchain parameter.
	// 3. DoDAOTransfer(address) – Transfers funds via DAO mechanism.
	// 4. DoSubsidyTx(address) – Executes a subsidy transaction.
	// 5. DoStartPollTx(address) – Starts a voting poll.
	// 6. DoVotePollTx(address) – Votes on the poll.
	// 7. DoStakeDelegateTx(address3) – Delegates stake from `address3`.
	// 8. DoDexLimitOrder(address) – Places a limit order on a DEX.
	// 9. DoDexLiquidityDeposit(address) – Deposits liquidity into DEX.
	// 10. DoCreateOrderTx(address) – Creates two orders; orderId will be deleted, orderId2 will be locked/closed.
	// 11. DoStakeValidatorTx(address2) – Stakes tokens as a validator from address2.
	// 12. Wait 2 blocks

	DoSendTx(address)
	DoChangeParamTx(address)
	DoDAOTransfer(address)
	DoSubsidyTx(address)
	DoStartPollTx(address)
	DoVotePollTx(address)
	DoStakeDelegateTx(address3)
	DoDexLimitOrder(address)
	DoDexLiquidityDeposit(address)
	orderId := DoCreateOrderTx(address)
	orderId2 := DoCreateOrderTx(address)
	DoStakeValidatorTx(address2)
	WaitBlocks()

	// STEP 3:
	// 1. DoEditOrderTx(address, orderId) – Edits the first order.
	// 2. DoDeleteOrderTx(address, orderId) – Deletes the first order.
	// 3. DoEditStakeValidatorTx(address2) – Modifies validator stake.
	// 4. Wait 2 blocks
	DoEditOrderTx(address, orderId)
	DoDeleteOrderTx(address, orderId)
	DoEditStakeValidatorTx(address2)
	WaitBlocks()

	// STEP 4:
	// 1. DoPauseTx(address2) – Pauses validator or protocol actions.
	// 2. WaitBlocks() – Waits 2 blocks.
	// 3. DoUnpauseTx(address2) – Unpauses validator or protocol actions.
	// 4. WaitBlocks() – Waits 2 blocks.
	// 5. DoUnstakeTx(address2) – Unstakes validator tokens.
	DoPauseTx(address2)
	WaitBlocks()
	DoUnpauseTx(address2)
	WaitBlocks()
	DoUnstakeTx(address2)

	// STEP 5: Execute a dex liquidity withdrawal from the deposit earlier
	DoDexLiqudityWithdrawal(address)

	// STEP 6:
	// 1. DoLockOrderTx(orderId2) – Locks the second order.
	// 2. WaitBlocks() – Waits 2 blocks.
	// 3. DoCloseOrderTx(orderId2) – Closes the second order.
	DoLockOrderTx(orderId2)
	WaitBlocks()
	DoCloseOrderTx(orderId2)
}

func DoSendTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing transfer of %d CNPY to %s\n", Amount, RecipientAddress)
	// send the tx
	tx := &txSend{
		Fee:      20_000,
		Amount:   Amount, // in uCNPY
		Output:   RecipientAddress,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
	}
	postTx(tx, SendTxURL)
}

func DoStakeValidatorTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing stake of %d CNPY for %s\n", Amount, RecipientAddress)
	// send the tx
	tx := &txStake{
		Fee:             10_000,
		Amount:          Amount, // in uCNPY
		Output:          RecipientAddress,
		Delegate:        false,
		EarlyWithdrawal: true,
		NetAddress:      "tcp://fake.com",
		Submit:          true,
		Password:        Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "3,4",
		},
	}
	postTx(tx, StakeTxURL)
}

func DoEditStakeValidatorTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing edit-stake of %d CNPY for %s\n", Amount, RecipientAddress)
	// send the tx
	tx := &txStake{
		Fee:             10_000,
		Amount:          2 * Amount, // in uCNPY
		Output:          RecipientAddress,
		Delegate:        false,
		EarlyWithdrawal: false,
		NetAddress:      "tcp://fake2.com",
		Submit:          true,
		Password:        Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "1",
		},
	}
	postTx(tx, EditStakeTxURL)
}

func DoPauseTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing pause for %s\n", RecipientAddress)
	// send the tx
	tx := &txAddress{
		Fee:      10_000,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
	}
	postTx(tx, PauseTxURL)
}

func DoUnpauseTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing unpause for %s\n", RecipientAddress)
	// send the tx
	tx := &txAddress{
		Fee:      10_000,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
	}
	postTx(tx, UnpauseTxURL)
}

func DoUnstakeTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing unstake for %s\n", RecipientAddress)
	// send the tx
	tx := &txAddress{
		Fee:      10_000,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
	}
	postTx(tx, UnstakeTxURL)
}

func DoStakeDelegateTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing delegate stake of %d CNPY for %s\n", Amount, DelegateAddress)
	// send the tx
	tx := &txStake{
		Fee:             10_000,
		Amount:          Amount, // in uCNPY
		Output:          RecipientAddress,
		Delegate:        true,
		EarlyWithdrawal: false,
		Submit:          true,
		Password:        Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		signerFields: signerFields{
			Signer: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "3,4",
		},
	}
	postTx(tx, StakeTxURL)
}

func DoChangeParamTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing change param tx for %s\n", "sendFee")
	// send the tx
	tx := &txChangeParam{
		Fee:      10_000,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		txChangeParamRequest: txChangeParamRequest{
			ParamSpace: "fee",
			ParamKey:   "sendFee",
			ParamValue: "20000",
			StartBlock: 1,
			EndBlock:   100,
		},
	}
	postTx(tx, ChangeParamTxURL)
}

func DoDAOTransfer(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing a dao transfer tx\n")
	// send the tx
	tx := &txDaoTransfer{
		Fee:      10_000,
		Amount:   Amount,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		txChangeParamRequest: txChangeParamRequest{
			StartBlock: 1,
			EndBlock:   1000,
		},
	}
	postTx(tx, DAOTransferTxURL)
}

func DoSubsidyTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing subsidty tx of %d CNPY for %s\n", Amount, DelegateAddress)
	// send the tx
	tx := &txSubsidy{
		Fee:      10_000,
		Amount:   Amount, // in uCNPY
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		txChangeParamRequest: txChangeParamRequest{},
		committeesRequest: committeesRequest{
			Committees: "111",
		},
	}
	postTx(tx, SubsidyTxURL)
}

func DoCreateOrderTx(address crypto.AddressI) string {
	// log the tx
	fmt.Printf("Executing create order tx of %d CNPY for %s\n", Amount, address.String())
	// send the tx
	tx := &txCreateOrder{
		Fee:            10_000,
		Amount:         Amount, // in uCNPY
		Password:       Pwd,
		Data:           crypto.Hash([]byte("fake")),
		Submit:         true,
		ReceiveAmount:  Amount + 1,
		ReceiveAddress: address.Bytes(),
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "2",
		},
	}
	return postTx(tx, CreateOrderTxURL)[:40]
}

func DoEditOrderTx(address crypto.AddressI, orderId string) {
	// log the tx
	fmt.Printf("Executing edit order tx of %d CNPY for %s\n", Amount, address.String())
	// send the tx
	tx := &txEditOrder{
		Fee:            10_000,
		Amount:         Amount, // in uCNPY
		Password:       Pwd,
		Submit:         true,
		ReceiveAmount:  Amount + 1,
		ReceiveAddress: address.Bytes(),
		OrderId:        orderId,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "2",
		},
	}
	postTx(tx, EditOrderTxURL)
}

func DoDeleteOrderTx(address crypto.AddressI, orderId string) {
	// log the tx
	fmt.Printf("Executing delete order tx of %d CNPY for %s\n", Amount, address.String())
	// send the tx
	tx := &txDeleteOrder{
		Fee:      10_000,
		Password: Pwd,
		Submit:   true,
		OrderId:  orderId,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		committeesRequest: committeesRequest{
			Committees: "2",
		},
	}
	postTx(tx, DeleteOrderTxURL)
}

func DoLockOrderTx(orderId string) {
	recipient, _ := hex.DecodeString(Addr2)
	// log the tx
	fmt.Printf("Executing lock order tx for %s\n", Addr2)
	// send the tx
	tx := &txLockOrder{
		Fee:            20_000,
		OrderId:        orderId,
		ReceiveAddress: recipient,
		Submit:         true,
		Password:       Pwd,
		fromFields: fromFields{
			Address: recipient,
		},
	}
	postTx(tx, TxLockOrderTxURL)
}

func DoCloseOrderTx(orderId string) {
	recipient, _ := hex.DecodeString(Addr2)
	// log the tx
	fmt.Printf("Executing close order tx for %s\n", Addr2)
	// send the tx
	tx := &txCloseOrder{
		Fee:      20_000,
		OrderId:  orderId,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: recipient,
		},
	}
	postTx(tx, TxCloseOrderTxURL)
}

func DoDexLimitOrder(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing dex limit order tx for %s\n", address)
	// send the tx
	tx := &txDexLimitOrder{
		Fee:           10_000,
		Amount:        25,
		ReceiveAmount: 19,
		Submit:        true,
		Password:      Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		committeesRequest: committeesRequest{"2"},
	}
	postTx(tx, TxDexLimitOrderTxURL)
}

func DoDexLiquidityDeposit(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing dex liquidity deposit tx for %s\n", address)
	// send the tx
	tx := &txDexLiquidityDeposit{
		Fee:      10_000,
		Amount:   100,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		committeesRequest: committeesRequest{"2"},
	}
	postTx(tx, TxDexLiquidityDepositTxURL)
}

func DoDexLiqudityWithdrawal(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing dex liquidity withdrawal tx for %s\n", address)
	// send the tx
	tx := &txDexLiquidityWithdraw{
		Fee:      10_000,
		Percent:  50,
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
		txChangeParamRequest: txChangeParamRequest{},
		committeesRequest:    committeesRequest{"2"},
	}
	postTx(tx, TxDexLiqudityWithdrawTxURL)
}

func DoStartPollTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing start poll tx for %s\n", address)
	// send the tx
	tx := &txStartPoll{
		Fee:      20_000,
		PollJSON: []byte(`{"proposal": "canopy network is the best","endBlock": 100,"URL": "https://discord.com/link-to-thread"}`),
		Submit:   true,
		Password: Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
	}
	postTx(tx, TxStartPollTxURL)
}

func DoVotePollTx(address crypto.AddressI) {
	// log the tx
	fmt.Printf("Executing vote poll tx for %s\n", address)
	// send the tx
	tx := &txVotePoll{
		Fee:         20_000,
		PollJSON:    []byte(`{"proposal": "canopy network is the best","endBlock": 100,"URL": "tcp://discord.com/link-to-thread"}`),
		PollApprove: true,
		Submit:      true,
		Password:    Pwd,
		fromFields: fromFields{
			Address: address.Bytes(),
		},
	}
	postTx(tx, TxVotePollTxURL)
}

func WaitBlocks(numBlocks ...int) {
	waitBlocks, lastHeight := 2, 0
	type HeightResp struct {
		Height int `json:"height"`
	}
	if len(numBlocks) == 1 {
		waitBlocks = numBlocks[0]
	}
	for range time.Tick(time.Second) {
		// get height
		got, err := post(QueryHeightURL, nil)
		if err != nil {
			panic(err)
		}
		// unmarshal response
		resp := &HeightResp{}
		if err = json.Unmarshal(got, resp); err != nil {
			panic(err)
		}
		if lastHeight != 0 && resp.Height >= lastHeight+waitBlocks {
			return
		}
		if lastHeight == 0 {
			lastHeight = resp.Height
		}
	}
}

var count int

func postTx(obj any, url string) string {
	count++
	// marshal the tx
	bz, e := json.MarshalIndent(obj, "", "  ")
	if e != nil {
		log.Fatal("Error marshalling:", e)
	}
	// send the txn
	hash, e := post(url, bz)
	if e != nil {
		log.Fatal("Error POSTing transfer:", e)
	}
	fmt.Printf("Tx #%d, Hash:%s\n", count, string(hash))
	return strings.Trim(string(hash), "\"")
}

var httpClient = &http.Client{}

func post(url string, bz []byte) ([]byte, error) {
	// generate the request
	request, e := http.NewRequest("POST", url, bytes.NewBuffer(bz))
	if e != nil {
		return nil, fmt.Errorf("Error creating request to %s:%s\n", url, e.Error())
	}
	// execute the request
	resp, e := httpClient.Do(request)
	if e != nil {
		return nil, fmt.Errorf("Error executing request to %s:%s\n", url, e.Error())
	}
	defer resp.Body.Close()
	// check the status code
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Non 200 status code from %s:%d\n", url, resp.StatusCode)
	}
	// read the request bytes
	respBz, e := io.ReadAll(resp.Body)
	if e != nil {
		return nil, fmt.Errorf("Error reading response from %s:%s\n", url, e.Error())
	}
	// return
	return respBz, nil
}

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

type paginatedIdRequest struct {
	idRequest
	lib.PageParams
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

type eventTypeRequest struct {
	EventType string `json:"eventType"`
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
	Percent         uint64          `json:"percent"`
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
