package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/canopy-network/canopy/fsm"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/p2p"
	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	rpcURL      string
	adminRpcUrl string
	client      http.Client
}

func NewClient(rpcURL, adminRPCUrl string) *Client {
	return &Client{rpcURL: rpcURL, adminRpcUrl: adminRPCUrl, client: http.Client{}}
}

func (c *Client) Version() (version *string, err lib.ErrorI) {
	version = new(string)
	err = c.get(VersionRouteName, "", version)
	return
}

func (c *Client) Height() (p *lib.HeightResult, err lib.ErrorI) {
	p = new(lib.HeightResult)
	err = c.post(HeightRouteName, nil, p)
	return
}

// IndexerBlobs retrieves the indexer blob protobuf and unmarshals it.
func (c *Client) IndexerBlobs(height uint64) (p *fsm.IndexerBlobs, err lib.ErrorI) {
	p = new(fsm.IndexerBlobs)
	req := heightRequest{Height: height}
	bz, err := lib.MarshalJSON(req)
	if err != nil {
		return nil, err
	}
	resp, e := c.client.Post(c.url(IndexerBlobsRouteName, ""), ApplicationJSON, bytes.NewBuffer(bz))
	if e != nil {
		return nil, lib.ErrPostRequest(e)
	}
	defer func() { _ = resp.Body.Close() }()
	body, e := io.ReadAll(resp.Body)
	if e != nil {
		return nil, lib.ErrReadBody(e)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, lib.ErrHttpStatus(resp.Status, resp.StatusCode, body)
	}
	if err := proto.Unmarshal(body, p); err != nil {
		return nil, lib.ErrUnmarshal(err)
	}
	return p, nil
}

func (c *Client) BlockByHeight(height uint64) (p *lib.BlockResult, err lib.ErrorI) {
	p = new(lib.BlockResult)
	err = c.heightRequest(BlockByHeightRouteName, height, p)
	return
}

func (c *Client) BlockByHash(hash string) (p *lib.BlockResult, err lib.ErrorI) {
	p = new(lib.BlockResult)
	err = c.hashRequest(BlockByHashRouteName, hash, p)
	return
}

func (c *Client) Blocks(params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(BlocksRouteName, 0, params, p)
	return
}

func (c *Client) EventsByHeight(height uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(EventsByHeightRouteName, height, params, p)
	return
}

func (c *Client) EventsByAddress(address string, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedAddrRequest(EventsByAddressRouteName, address, params, p)
	return
}

func (c *Client) EventsByChainId(id uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedIdRequest(EventsByChainRouteName, id, params, p)
	return
}

func (c *Client) Pending(params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedAddrRequest(PendingRouteName, "", params, p)
	return
}

func (c *Client) Proposals() (p *fsm.GovProposals, err lib.ErrorI) {
	p = new(fsm.GovProposals)
	err = c.get(ProposalsRouteName, "", p)
	return
}

func (c *Client) Poll() (p *fsm.Poll, err lib.ErrorI) {
	p = new(fsm.Poll)
	err = c.get(PollRouteName, "", p)
	return
}

func (c *Client) AddVote(proposal json.RawMessage, approve bool) (p *voteRequest, err lib.ErrorI) {
	p = new(voteRequest)
	bz, err := lib.MarshalJSON(voteRequest{
		Approve:  approve,
		Proposal: proposal,
	})
	if err != nil {
		return nil, err
	}
	err = c.post(AddVoteRouteName, bz, p, true)
	return
}

func (c *Client) DelVote(hash string) (p *hashRequest, err lib.ErrorI) {
	p = new(hashRequest)
	err = c.hashRequest(DelVoteRouteName, hash, p, true)
	return
}

func (c *Client) CertByHeight(height uint64) (p *lib.QuorumCertificate, err lib.ErrorI) {
	p = new(lib.QuorumCertificate)
	err = c.heightRequest(CertByHeightRouteName, height, p)
	return
}

func (c *Client) TransactionByHash(hash string) (p *lib.TxResult, err lib.ErrorI) {
	p = new(lib.TxResult)
	err = c.hashRequest(TxByHashRouteName, hash, p)
	return
}

func (c *Client) TransactionsByHeight(height uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(TxsByHeightRouteName, height, params, p)
	return
}

func (c *Client) TransactionsBySender(address string, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedAddrRequest(TxsBySenderRouteName, address, params, p)
	return
}

func (c *Client) TransactionsByRecipient(address string, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedAddrRequest(TxsByRecRouteName, address, params, p)
	return
}

func (c *Client) Account(height uint64, address string) (p *fsm.Account, err lib.ErrorI) {
	p = new(fsm.Account)
	err = c.heightAndAddressRequest(AccountRouteName, height, address, p)
	return
}

func (c *Client) Accounts(height uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(AccountsRouteName, height, params, p)
	return
}

func (c *Client) Pool(height uint64, id uint64) (p *fsm.Pool, err lib.ErrorI) {
	p = new(fsm.Pool)
	err = c.heightAndIdRequest(PoolRouteName, height, id, p)
	return
}

func (c *Client) Pools(height uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(PoolsRouteName, height, params, p)
	return
}

func (c *Client) Validator(height uint64, address string) (p *fsm.Validator, err lib.ErrorI) {
	p = new(fsm.Validator)
	err = c.heightAndAddressRequest(ValidatorRouteName, height, address, p)
	return
}

func (c *Client) Validators(height uint64, params lib.PageParams, filter lib.ValidatorFilters) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(ValidatorsRouteName, height, params, p, filter)
	return
}

func (c *Client) Committee(height uint64, id uint64, params lib.PageParams) (p *lib.Page, err lib.ErrorI) {
	p = new(lib.Page)
	err = c.paginatedHeightRequest(CommitteeRouteName, height, params, p, lib.ValidatorFilters{Committee: id})
	return
}

func (c *Client) CommitteeData(height uint64, id uint64) (p *lib.CommitteeData, err lib.ErrorI) {
	p = new(lib.CommitteeData)
	err = c.heightAndIdRequest(CommitteeDataRouteName, height, id, p)
	return
}

func (c *Client) CommitteesData(height uint64) (p *lib.CommitteesData, err lib.ErrorI) {
	p = new(lib.CommitteesData)
	err = c.paginatedHeightRequest(CommitteesDataRouteName, height, lib.PageParams{}, p)
	return
}

func (c *Client) RootChainInfo(height, chainId uint64) (p *lib.RootChainInfo, err lib.ErrorI) {
	p = new(lib.RootChainInfo)
	err = c.heightAndIdRequest(RootChainInfoRouteName, height, chainId, p)
	return
}

func (c *Client) SubsidizedCommittees(height uint64) (p *[]uint64, err lib.ErrorI) {
	p = new([]uint64)
	err = c.heightRequest(SubsidizedCommitteesRouteName, height, p)
	return
}

func (c *Client) RetiredCommittees(height uint64) (p *[]uint64, err lib.ErrorI) {
	p = new([]uint64)
	err = c.heightRequest(RetiredCommitteesRouteName, height, p)
	return
}

func (c *Client) Order(height uint64, orderId string, chainId uint64) (p *lib.SellOrder, err lib.ErrorI) {
	p = new(lib.SellOrder)
	err = c.orderRequest(OrderRouteName, height, orderId, chainId, p)
	return
}

func (c *Client) Orders(height, chainId uint64) (p *lib.OrderBooks, err lib.ErrorI) {
	p = new(lib.OrderBooks)
	err = c.heightAndIdRequest(OrdersRouteName, height, chainId, p)
	return
}

func (c *Client) DexPrice(height, chainId uint64) (p *lib.DexPrice, err lib.ErrorI) {
	p = new(lib.DexPrice)
	err = c.heightAndIdRequest(DexPriceRouteName, height, chainId, p)
	return
}

func (c *Client) DexBatch(height, chainId uint64, withPoints bool) (p *lib.DexBatch, err lib.ErrorI) {
	p = new(lib.DexBatch)
	err = c.heightIdAndPointsRequest(DexBatchRouteName, height, chainId, withPoints, p)
	return
}

func (c *Client) NextDexBatch(height, chainId uint64, withPoints bool) (p *lib.DexBatch, err lib.ErrorI) {
	p = new(lib.DexBatch)
	err = c.heightIdAndPointsRequest(NextDexBatchRouteName, height, chainId, withPoints, p)
	return
}

func (c *Client) CanopyOrders(height uint64, params lib.PageParams) (orders []*types.WitnessedOrder, err lib.ErrorI) {
	err = c.paginatedHeightRequest(CanopyOrdersRouteName, height, params, &orders)
	return
}

func (c *Client) LastProposers(height uint64) (p *lib.Proposers, err lib.ErrorI) {
	p = new(lib.Proposers)
	err = c.heightRequest(LastProposersRouteName, height, p)
	return
}

func (c *Client) IsValidDoubleSigner(height uint64, address string) (p *bool, err lib.ErrorI) {
	p = new(bool)
	err = c.heightAndAddressRequest(IsValidDoubleSignerRouteName, height, address, p)
	return
}

func (c *Client) Checkpoint(height, id uint64) (p lib.HexBytes, err lib.ErrorI) {
	p = make(lib.HexBytes, 0)
	err = c.heightAndIdRequest(CheckpointRouteName, height, id, &p)
	return
}

func (c *Client) DoubleSigners(height uint64) (p *[]*lib.DoubleSigner, err lib.ErrorI) {
	p = new([]*lib.DoubleSigner)
	err = c.heightRequest(DoubleSignersRouteName, height, p)
	return
}

func (c *Client) ValidatorSet(height uint64, id uint64) (v lib.ValidatorSet, err lib.ErrorI) {
	p := new(lib.ConsensusValidators)
	err = c.heightAndIdRequest(ValidatorSetRouteName, height, id, p)
	if err != nil {
		return lib.ValidatorSet{}, err
	}
	return lib.NewValidatorSet(p)
}

func (c *Client) MinimumEvidenceHeight(height uint64) (p *uint64, err lib.ErrorI) {
	p = new(uint64)
	err = c.heightRequest(MinimumEvidenceHeightRouteName, height, p)
	return
}

func (c *Client) Lottery(height, id uint64) (p *lib.LotteryWinner, err lib.ErrorI) {
	p = new(lib.LotteryWinner)
	err = c.heightAndIdRequest(LotteryRouteName, height, id, p)
	return
}

func (c *Client) Supply(height uint64) (p *fsm.Supply, err lib.ErrorI) {
	p = new(fsm.Supply)
	err = c.heightRequest(SupplyRouteName, height, p)
	return
}

func (c *Client) NonSigners(height uint64) (p *fsm.NonSigners, err lib.ErrorI) {
	p = new(fsm.NonSigners)
	err = c.heightRequest(NonSignersRouteName, height, p)
	return
}

func (c *Client) Params(height uint64) (p *fsm.Params, err lib.ErrorI) {
	p = new(fsm.Params)
	err = c.heightRequest(ParamRouteName, height, p)
	return
}

func (c *Client) FeeParams(height uint64) (p *fsm.FeeParams, err lib.ErrorI) {
	p = new(fsm.FeeParams)
	err = c.heightRequest(FeeParamRouteName, height, p)
	return
}

func (c *Client) GovParams(height uint64) (p *fsm.GovernanceParams, err lib.ErrorI) {
	p = new(fsm.GovernanceParams)
	err = c.heightRequest(GovParamRouteName, height, p)
	return
}

func (c *Client) ConParams(height uint64) (p *fsm.ConsensusParams, err lib.ErrorI) {
	p = new(fsm.ConsensusParams)
	err = c.heightRequest(ConParamsRouteName, height, p)
	return
}

func (c *Client) ValParams(height uint64) (p *fsm.ValidatorParams, err lib.ErrorI) {
	p = new(fsm.ValidatorParams)
	err = c.heightRequest(ValParamRouteName, height, p)
	return
}

func (c *Client) State(height uint64) (p *fsm.GenesisState, err lib.ErrorI) {
	var param string
	if height != 0 {
		param = fmt.Sprintf("?height=%d", height)
	}
	p = new(fsm.GenesisState)
	err = c.get(StateRouteName, param, p)
	return
}

func (c *Client) StateDiff(height, startHeight uint64) (diff string, err lib.ErrorI) {
	bz, err := lib.MarshalJSON(heightsRequest{heightRequest: heightRequest{height}, StartHeight: startHeight})
	if err != nil {
		return
	}
	resp, e := c.client.Post(c.url(StateDiffRouteName, "", false), ApplicationJSON, bytes.NewBuffer(bz))
	if e != nil {
		return "", lib.ErrPostRequest(e)
	}
	bz, e = io.ReadAll(resp.Body)
	if e != nil {
		return "", lib.ErrReadBody(e)
	}
	diff = string(bz)
	return
}

func (c *Client) TransactionJSON(tx json.RawMessage) (hash *string, err lib.ErrorI) {
	hash = new(string)
	err = c.post(TxRouteName, tx, hash)
	return
}

func (c *Client) Transaction(tx lib.TransactionI) (hash *string, err lib.ErrorI) {
	bz, err := lib.MarshalJSON(tx)
	if err != nil {
		return nil, err
	}
	hash = new(string)
	err = c.post(TxRouteName, bz, hash)
	return
}

func (c *Client) Transactions(txs []lib.TransactionI) (hash *string, err lib.ErrorI) {
	bz, err := lib.MarshalJSON(txs)
	if err != nil {
		return nil, err
	}
	hash = new(string)
	err = c.post(TxsRouteName, bz, hash)
	return
}

func (c *Client) Keystore() (keystore *crypto.Keystore, err lib.ErrorI) {
	keystore = new(crypto.Keystore)
	err = c.get(KeystoreRouteName, "", keystore, true)
	return
}
func (c *Client) KeystoreNewKey(password, nickname string) (address crypto.AddressI, err lib.ErrorI) {
	address = new(crypto.Address)
	err = c.keystoreRequest(KeystoreNewKeyRouteName, keystoreRequest{
		passwordRequest: passwordRequest{password},
		nicknameRequest: nicknameRequest{nickname},
	}, address)
	return
}

func (c *Client) KeystoreImport(address, nickname string, epk crypto.EncryptedPrivateKey) (returned crypto.AddressI, err lib.ErrorI) {
	bz, err := lib.NewHexBytesFromString(address)
	if err != nil {
		return nil, err
	}
	returned = new(crypto.Address)
	err = c.keystoreRequest(KeystoreImportRouteName, keystoreRequest{
		addressRequest:      addressRequest{bz},
		nicknameRequest:     nicknameRequest{nickname},
		EncryptedPrivateKey: epk,
	}, returned)
	return
}

func (c *Client) KeystoreImportRaw(privateKey, password, nickname string) (returned crypto.AddressI, err lib.ErrorI) {
	bz, err := lib.NewHexBytesFromString(privateKey)
	if err != nil {
		return nil, err
	}
	returned = new(crypto.Address)
	err = c.keystoreRequest(KeystoreImportRawRouteName, keystoreRequest{
		PrivateKey:      bz,
		passwordRequest: passwordRequest{password},
		nicknameRequest: nicknameRequest{nickname},
	}, returned)
	return
}

type AddrOrNickname struct {
	Address  string
	Nickname string
}

func (c *Client) KeystoreDelete(addrOrNickname AddrOrNickname) (returned crypto.AddressI, err lib.ErrorI) {
	returned = new(crypto.Address)

	if addrOrNickname.Address != "" {
		var bz lib.HexBytes
		bz, err = lib.NewHexBytesFromString(addrOrNickname.Address)
		if err != nil {
			return
		}
		err = c.keystoreRequest(KeystoreDeleteRouteName, keystoreRequest{
			addressRequest: addressRequest{bz},
		}, returned)
		return
	}

	if addrOrNickname.Nickname != "" {
		err = c.keystoreRequest(KeystoreDeleteRouteName, keystoreRequest{
			nicknameRequest: nicknameRequest{addrOrNickname.Nickname},
		}, returned)
		return
	}

	return
}

func (c *Client) KeystoreGet(addrOrNickname AddrOrNickname, password string) (returned *crypto.KeyGroup, err lib.ErrorI) {
	returned = new(crypto.KeyGroup)

	if addrOrNickname.Address != "" {
		var bz lib.HexBytes
		bz, err = lib.NewHexBytesFromString(addrOrNickname.Address)
		if err != nil {
			return
		}
		err = c.keystoreRequest(KeystoreGetRouteName, keystoreRequest{
			addressRequest:  addressRequest{bz},
			passwordRequest: passwordRequest{password},
		}, returned)
		return
	}

	if addrOrNickname.Nickname != "" {
		err = c.keystoreRequest(KeystoreGetRouteName, keystoreRequest{
			nicknameRequest: nicknameRequest{addrOrNickname.Nickname},
			passwordRequest: passwordRequest{password},
		}, returned)
		return
	}

	return
}

func getFrom(address, nickname string) (from fromFields, err lib.ErrorI) {
	if address != "" {
		from.Address, err = lib.NewHexBytesFromString(address)
		if err != nil {
			return
		}
	}

	from.Nickname = nickname
	return from, nil
}

func getSigner(signer AddrOrNickname) (s signerFields, err lib.ErrorI) {
	if signer.Address != "" {
		s.Signer, err = lib.NewHexBytesFromString(signer.Address)
		if err != nil {
			return
		}
	}

	s.SignerNickname = signer.Nickname
	return
}

func (c *Client) TxSend(from AddrOrNickname, rec string, amt uint64, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txSend{
		Fee:      optFee,
		Amount:   amt,
		Output:   rec,
		Submit:   submit,
		Password: pwd,
	}

	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxSendRouteName, txReq, submit)
}

func (c *Client) TxStake(addrOrNick AddrOrNickname, netAddr string, amt uint64, committees, output string, signer AddrOrNickname, delegate, earlyWithdrawal bool, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	return c.txStake(addrOrNick, netAddr, amt, committees, output, delegate, earlyWithdrawal, signer, pwd, submit, false, optFee)
}

func (c *Client) TxEditStake(addrOrNick AddrOrNickname, netAddr string, amt uint64, committees, output string, signer AddrOrNickname, delegate, earlyWithdrawal bool, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	return c.txStake(addrOrNick, netAddr, amt, committees, output, delegate, earlyWithdrawal, signer, pwd, submit, true, optFee)
}

func (c *Client) TxUnstake(addrOrNick, signer AddrOrNickname, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	return c.txAddress(TxUnstakeRouteName, addrOrNick, signer, pwd, submit, optFee)
}

func (c *Client) TxPause(addrOrNick, signer AddrOrNickname, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	return c.txAddress(TxPauseRouteName, addrOrNick, signer, pwd, submit, optFee)
}

func (c *Client) TxUnpause(addrOrNick, signer AddrOrNickname, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	return c.txAddress(TxUnpauseRouteName, addrOrNick, signer, pwd, submit, optFee)
}

func (c *Client) TxChangeParam(from AddrOrNickname, pSpace, pKey, pValue string, startBlk, endBlk uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txChangeParam{
		Fee:      optFee,
		Submit:   submit,
		Password: pwd,
		txChangeParamRequest: txChangeParamRequest{
			ParamSpace: pSpace,
			ParamKey:   pKey,
			ParamValue: pValue,
			StartBlock: startBlk,
			EndBlock:   endBlk,
		},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}
	return c.transactionRequest(TxChangeParamRouteName, txReq, submit)
}

func (c *Client) TxDaoTransfer(from AddrOrNickname, amt, startBlk, endBlk uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txDaoTransfer{
		Fee:      optFee,
		Submit:   submit,
		Password: pwd,
		Amount:   amt,
		txChangeParamRequest: txChangeParamRequest{
			StartBlock: startBlk,
			EndBlock:   endBlk,
		},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}
	return c.transactionRequest(TxDAOTransferRouteName, txReq, submit)
}

func (c *Client) TxSubsidy(from AddrOrNickname, amt, chainId uint64, opCode string,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txSubsidy{
		Fee:               optFee,
		Submit:            submit,
		Password:          pwd,
		Amount:            amt,
		OpCode:            opCode,
		committeesRequest: committeesRequest{fmt.Sprintf("%d", chainId)},
	}

	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}
	return c.transactionRequest(TxSubsidyRouteName, txReq, submit)
}

func (c *Client) TxCreateOrder(from AddrOrNickname, sellAmount, receiveAmount, chainId uint64, receiveAddress string,
	pwd string, data lib.HexBytes, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	receiveAddr, err := lib.NewHexBytesFromString(receiveAddress)
	if err != nil {
		return nil, nil, err
	}
	txReq := txCreateOrder{
		Fee:                  optFee,
		Amount:               sellAmount,
		Submit:               submit,
		ReceiveAmount:        receiveAmount,
		ReceiveAddress:       receiveAddr,
		Data:                 data,
		Password:             pwd,
		txChangeParamRequest: txChangeParamRequest{},
		committeesRequest:    committeesRequest{fmt.Sprintf("%d", chainId)},
	}

	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}
	return c.transactionRequest(TxCreateOrderRouteName, txReq, submit)
}

func (c *Client) TxEditOrder(from AddrOrNickname, sellAmount, receiveAmount uint64, orderId string, chainId uint64, receiveAddress string,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	receiveAddr, err := lib.NewHexBytesFromString(receiveAddress)
	if err != nil {
		return nil, nil, err
	}
	txReq := txEditOrder{
		Fee:                  optFee,
		Amount:               sellAmount,
		ReceiveAmount:        receiveAmount,
		ReceiveAddress:       receiveAddr,
		OrderId:              orderId,
		Submit:               submit,
		Password:             pwd,
		txChangeParamRequest: txChangeParamRequest{},
		committeesRequest:    committeesRequest{fmt.Sprintf("%d", chainId)},
	}

	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxEditOrderRouteName, txReq, submit)
}

func (c *Client) TxDeleteOrder(from AddrOrNickname, orderId string, chainId uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txDeleteOrder{
		Fee:               optFee,
		OrderId:           orderId,
		Submit:            submit,
		Password:          pwd,
		committeesRequest: committeesRequest{fmt.Sprintf("%d", chainId)},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxDeleteOrderRouteName, txReq, submit)
}

func (c *Client) TxDexLimitOrder(from AddrOrNickname, amount, receiveAmount, chainId uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txDexLimitOrder{
		Fee:               optFee,
		Amount:            amount,
		ReceiveAmount:     receiveAmount,
		Submit:            submit,
		Password:          pwd,
		committeesRequest: committeesRequest{fmt.Sprintf("%d", chainId)},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxDexLimitOrderRouteName, txReq, submit)
}

func (c *Client) TxDexLiquidityDeposit(from AddrOrNickname, amount, chainId uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txDexLiquidityDeposit{
		Fee:               optFee,
		Amount:            amount,
		Submit:            submit,
		Password:          pwd,
		committeesRequest: committeesRequest{fmt.Sprintf("%d", chainId)},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxDexLiquidityDepositRouteName, txReq, submit)
}

func (c *Client) TxDexLiquidityWithdraw(from AddrOrNickname, percent int, chainId uint64,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txDexLiquidityWithdraw{
		Fee:               optFee,
		Percent:           percent,
		Submit:            submit,
		Password:          pwd,
		committeesRequest: committeesRequest{fmt.Sprintf("%d", chainId)},
	}
	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxDexLiquidityWithdrawRouteName, txReq, submit)
}

func (c *Client) TxLockOrder(from AddrOrNickname, receiveAddress string, orderId string,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	receiveHex, err := lib.NewHexBytesFromString(receiveAddress)
	if err != nil {
		return nil, nil, err
	}
	txReq := txLockOrder{
		Fee:            optFee,
		OrderId:        orderId,
		ReceiveAddress: receiveHex,
		Submit:         submit,
		Password:       pwd,
	}
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxLockOrderRouteName, txReq, submit)
}

func (c *Client) TxCloseOrder(from AddrOrNickname, orderId string, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txCloseOrder{
		Fee:      optFee,
		OrderId:  orderId,
		Submit:   submit,
		Password: pwd,
	}

	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxCloseOrderRouteName, txReq, submit)
}

func (c *Client) TxStartPoll(from AddrOrNickname, pollJSON json.RawMessage,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txStartPoll{
		Fee:      optFee,
		PollJSON: pollJSON,
		Submit:   submit,
		Password: pwd,
	}

	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxStartPollRouteName, txReq, submit)
}

func (c *Client) TxVotePoll(from AddrOrNickname, pollJSON json.RawMessage, pollApprove bool,
	pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txVotePoll{
		Fee:         optFee,
		PollJSON:    pollJSON,
		PollApprove: pollApprove,
		Submit:      submit,
		Password:    pwd,
	}

	var err lib.ErrorI
	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(TxVotePollRouteName, txReq, submit)
}

func (c *Client) ResourceUsage() (returned *resourceUsageResponse, err lib.ErrorI) {
	returned = new(resourceUsageResponse)
	err = c.get(ResourceUsageRouteName, "", returned, true)
	return
}

func (c *Client) PeerInfo() (returned *peerInfoResponse, err lib.ErrorI) {
	returned = new(peerInfoResponse)
	err = c.get(PeerInfoRouteName, "", returned, true)
	return
}

func (c *Client) ConsensusInfo() (returned *controller.ConsensusSummary, err lib.ErrorI) {
	returned = new(controller.ConsensusSummary)
	err = c.get(ConsensusInfoRouteName, "", returned, true)
	return
}

func (c *Client) PeerBook() (returned *[]*p2p.BookPeer, err lib.ErrorI) {
	returned = new([]*p2p.BookPeer)
	err = c.get(PeerBookRouteName, "", returned, true)
	return
}

func (c *Client) Config() (returned *lib.Config, err lib.ErrorI) {
	returned = new(lib.Config)
	err = c.get(ConfigRouteName, "", returned, true)
	return
}

func (c *Client) Logs() (logs string, err lib.ErrorI) {
	resp, e := c.client.Get(c.url(LogsRouteName, "", true))
	if e != nil {
		return "", lib.ErrGetRequest(err)
	}
	bz, e := io.ReadAll(resp.Body)
	if e != nil {
		return "", lib.ErrGetRequest(e)
	}
	return string(bz), nil
}

func (c *Client) txAddress(route string, from, signer AddrOrNickname, pwd string, submit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txAddress{
		Fee:      optFee,
		Submit:   submit,
		Password: pwd,
	}

	var err lib.ErrorI
	txReq.signerFields, err = getSigner(signer)
	if err != nil {
		return nil, nil, err
	}

	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(route, txReq, submit)
}

func (c *Client) txStake(from AddrOrNickname, netAddr string, amt uint64, committees, output string, delegate, earlyWithdrawal bool, signer AddrOrNickname, pwd string, submit, edit bool, optFee uint64) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	txReq := txStake{
		Fee:                  optFee,
		Amount:               amt,
		NetAddress:           netAddr,
		Output:               output,
		Delegate:             delegate,
		EarlyWithdrawal:      earlyWithdrawal,
		Submit:               submit,
		Password:             pwd,
		txChangeParamRequest: txChangeParamRequest{},
		committeesRequest:    committeesRequest{Committees: committees},
	}
	route := TxStakeRouteName
	if edit {
		route = TxEditStakeRouteName
	}

	var err lib.ErrorI
	txReq.signerFields, err = getSigner(signer)
	if err != nil {
		return nil, nil, err
	}

	txReq.fromFields, err = getFrom(from.Address, from.Nickname)
	if err != nil {
		return nil, nil, err
	}

	return c.transactionRequest(route, txReq, submit)
}

func (c *Client) transactionRequest(routeName string, txRequest any, submit bool) (hash *string, tx json.RawMessage, e lib.ErrorI) {
	bz, e := lib.MarshalJSON(txRequest)
	if e != nil {
		return
	}
	if submit {
		hash = new(string)
		e = c.post(routeName, bz, hash, true)
	} else {
		tx = json.RawMessage{}
		e = c.post(routeName, bz, &tx, true)
	}
	return
}

func (c *Client) keystoreRequest(routeName string, keystoreRequest keystoreRequest, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(keystoreRequest)
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr, true)
	return
}

func (c *Client) paginatedHeightRequest(routeName string, height uint64, p lib.PageParams, ptr any, filters ...lib.ValidatorFilters) (err lib.ErrorI) {
	var vf lib.ValidatorFilters
	if filters != nil {
		vf = filters[0]
	}
	bz, err := lib.MarshalJSON(paginatedHeightRequest{
		heightRequest:    heightRequest{height},
		PageParams:       p,
		ValidatorFilters: vf,
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) paginatedIdRequest(routeName string, id uint64, p lib.PageParams, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(paginatedIdRequest{
		idRequest: idRequest{
			ID: id,
		},
		PageParams: p,
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) paginatedAddrRequest(routeName string, address string, p lib.PageParams, ptr any) (err lib.ErrorI) {
	addr, err := lib.StringToBytes(address)
	if err != nil {
		return err
	}
	bz, err := lib.MarshalJSON(paginatedAddressRequest{
		addressRequest: addressRequest{addr},
		PageParams:     p,
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) heightRequest(routeName string, height uint64, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(heightRequest{Height: height})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) orderRequest(routeName string, height uint64, orderId string, chainId uint64, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(orderRequest{
		ChainId: chainId,
		OrderId: orderId,
		heightRequest: heightRequest{
			Height: height,
		},
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) hashRequest(routeName string, hash string, ptr any, admin ...bool) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(hashRequest{Hash: hash})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr, admin...)
	return
}

func (c *Client) heightAndAddressRequest(routeName string, height uint64, address string, ptr any) (err lib.ErrorI) {
	addr, err := lib.StringToBytes(address)
	if err != nil {
		return err
	}
	bz, err := lib.MarshalJSON(heightAndAddressRequest{
		heightRequest:  heightRequest{height},
		addressRequest: addressRequest{addr},
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) heightAndIdRequest(routeName string, height, id uint64, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(heightAndIdRequest{
		heightRequest: heightRequest{height},
		idRequest:     idRequest{id},
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) heightIdAndPointsRequest(routeName string, height, id uint64, points bool, ptr any) (err lib.ErrorI) {
	bz, err := lib.MarshalJSON(heightIdAndPointsRequest{
		heightAndIdRequest: heightAndIdRequest{
			heightRequest: heightRequest{height},
			idRequest:     idRequest{id},
		},
		Points: points,
	})
	if err != nil {
		return
	}
	err = c.post(routeName, bz, ptr)
	return
}

func (c *Client) url(routeName, param string, admin ...bool) string {
	// if an admin call
	if admin != nil && admin[0] {
		return c.adminRpcUrl + routePaths[routeName].Path + param
	}
	// if non admin call
	return c.rpcURL + routePaths[routeName].Path + param
}

func (c *Client) post(routeName string, json []byte, ptr any, admin ...bool) lib.ErrorI {
	resp, err := c.client.Post(c.url(routeName, "", admin...), ApplicationJSON, bytes.NewBuffer(json))
	if err != nil {
		return lib.ErrPostRequest(err)
	}
	return c.unmarshal(resp, ptr)
}

func (c *Client) get(routeName, param string, ptr any, admin ...bool) lib.ErrorI {
	resp, err := c.client.Get(c.url(routeName, param, admin...))
	if err != nil {
		return lib.ErrGetRequest(err)
	}
	return c.unmarshal(resp, ptr)
}

func (c *Client) unmarshal(resp *http.Response, ptr any) lib.ErrorI {
	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		return lib.ErrReadBody(err)
	}
	if resp.StatusCode != http.StatusOK {
		return lib.ErrHttpStatus(resp.Status, resp.StatusCode, bz)
	}
	return lib.UnmarshalJSON(bz, ptr)
}
