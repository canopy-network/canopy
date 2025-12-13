package fsm

import (
	"encoding/json"
	"fmt"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/proto"
)

const (
	// Names for each Transaction Message (payload) type
	MessageSendName                 = "send"
	MessageStakeName                = "stake"
	MessageUnstakeName              = "unstake"
	MessageEditStakeName            = "editStake"
	MessagePauseName                = "pause"
	MessageUnpauseName              = "unpause"
	MessageChangeParameterName      = "changeParameter"
	MessageDAOTransferName          = "daoTransfer"
	MessageCertificateResultsName   = "certificateResults"
	MessageSubsidyName              = "subsidy"
	MessageCreateOrderName          = "createOrder"
	MessageEditOrderName            = "editOrder"
	MessageDeleteOrderName          = "deleteOrder"
	MessageDexLimitOrderName        = "dexLimitOrder"
	MessageDexLiquidityDepositName  = "dexLiquidityDeposit"
	MessageDexLiquidityWithdrawName = "dexLiquidityWithdraw"
)

func init() {
	// Register all messages types for conversion of bytes to the correct MessageI implementation
	lib.RegisteredMessages[MessageSendName] = new(MessageSend)
	lib.RegisteredMessages[MessageStakeName] = new(MessageStake)
	lib.RegisteredMessages[MessageEditStakeName] = new(MessageEditStake)
	lib.RegisteredMessages[MessageUnstakeName] = new(MessageUnstake)
	lib.RegisteredMessages[MessagePauseName] = new(MessagePause)
	lib.RegisteredMessages[MessageUnpauseName] = new(MessageUnpause)
	lib.RegisteredMessages[MessageChangeParameterName] = new(MessageChangeParameter)
	lib.RegisteredMessages[MessageDAOTransferName] = new(MessageDAOTransfer)
	lib.RegisteredMessages[MessageCertificateResultsName] = new(MessageCertificateResults)
	lib.RegisteredMessages[MessageSubsidyName] = new(MessageSubsidy)
	lib.RegisteredMessages[MessageCreateOrderName] = new(MessageCreateOrder)
	lib.RegisteredMessages[MessageEditOrderName] = new(MessageEditOrder)
	lib.RegisteredMessages[MessageDeleteOrderName] = new(MessageDeleteOrder)
	lib.RegisteredMessages[MessageDexLimitOrderName] = new(MessageDexLimitOrder)
	lib.RegisteredMessages[MessageDexLiquidityDepositName] = new(MessageDexLiquidityDeposit)
	lib.RegisteredMessages[MessageDexLiquidityWithdrawName] = new(MessageDexLiquidityWithdraw)
}

var _ lib.MessageI = &MessageSend{} // interface enforcement

// Check() validates the Message structure
func (x *MessageSend) Check() lib.ErrorI {
	if err := checkAddress(x.FromAddress); err != nil {
		return err
	}
	if x.ToAddress == nil {
		return ErrRecipientAddressEmpty()
	}
	if len(x.ToAddress) != crypto.AddressSize {
		return ErrRecipientAddressSize()
	}
	return checkAmount(x.Amount)
}

func (x *MessageSend) Name() string      { return MessageSendName }
func (x *MessageSend) New() lib.MessageI { return new(MessageSend) }
func (x *MessageSend) Recipient() []byte { return crypto.NewAddressFromBytes(x.ToAddress).Bytes() } // who is the receiver of the message

// MarshalJSON() is the json.Marshaller implementation for MessageSend
func (x MessageSend) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageSend{
		FromAddress: x.FromAddress,
		ToAddress:   x.ToAddress,
		Amount:      x.Amount,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageSend
func (x *MessageSend) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageSend
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageSend{
		FromAddress: j.FromAddress,
		ToAddress:   j.ToAddress,
		Amount:      j.Amount,
	}
	return
}

type jsonMessageSend struct {
	FromAddress lib.HexBytes `json:"fromAddress"`
	ToAddress   lib.HexBytes `json:"toAddress"`
	Amount      uint64       `json:"amount"`
}

var _ lib.MessageI = &MessageStake{} // interface enforcement

// Check() validates the Message structure
func (x *MessageStake) Check() lib.ErrorI {
	if err := checkOutputAddress(x.OutputAddress); err != nil {
		return err
	}
	if err := checkPubKey(x.PublicKey, x.Delegate); err != nil {
		return err
	}
	if err := checkCommittees(x.Committees); err != nil {
		return err
	}
	if err := ensureEmpty(x.Signer); err != nil {
		return err
	}
	return checkAmount(x.Amount)
}

func (x *MessageStake) Name() string      { return MessageStakeName }
func (x *MessageStake) New() lib.MessageI { return new(MessageStake) }
func (x *MessageStake) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageStake
func (x MessageStake) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageStake{
		PublicKey:     x.PublicKey,
		Amount:        x.Amount,
		Committees:    x.Committees,
		NetAddress:    x.NetAddress,
		OutputAddress: x.OutputAddress,
		Delegate:      x.Delegate,
		Compound:      x.Compound,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageStake
func (x *MessageStake) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageStake
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageStake{
		PublicKey:     j.PublicKey,
		Amount:        j.Amount,
		Committees:    j.Committees,
		NetAddress:    j.NetAddress,
		OutputAddress: j.OutputAddress,
		Delegate:      j.Delegate,
		Compound:      j.Compound,
	}
	return
}

type jsonMessageStake struct {
	PublicKey     lib.HexBytes `json:"publickey"`
	Amount        uint64       `json:"amount"`
	Committees    []uint64     `json:"committees"`
	NetAddress    string       `json:"netAddress"`
	OutputAddress lib.HexBytes `json:"outputAddress"`
	Delegate      bool         `json:"delegate"`
	Compound      bool         `json:"compound"`
}

var _ lib.MessageI = &MessageEditStake{} // interface enforcement

// Check() validates the Message structure
func (x *MessageEditStake) Check() lib.ErrorI {
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if err := checkOutputAddress(x.OutputAddress); err != nil {
		return err
	}
	if err := checkCommittees(x.Committees); err != nil {
		return err
	}
	if err := ensureEmpty(x.Signer); err != nil {
		return err
	}
	return checkAmount(x.Amount)
}

func (x *MessageEditStake) Name() string      { return MessageEditStakeName }
func (x *MessageEditStake) New() lib.MessageI { return new(MessageEditStake) }
func (x *MessageEditStake) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageEditStake
func (x MessageEditStake) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageEditStake{
		Address:       x.Address,
		Amount:        x.Amount,
		Committees:    x.Committees,
		NetAddress:    x.NetAddress,
		OutputAddress: x.OutputAddress,
		Compound:      x.Compound,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditStake
func (x *MessageEditStake) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageEditStake
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageEditStake{
		Address:       j.Address,
		Amount:        j.Amount,
		Committees:    j.Committees,
		NetAddress:    j.NetAddress,
		OutputAddress: j.OutputAddress,
		Compound:      j.Compound,
	}
	return
}

type jsonMessageEditStake struct {
	Address       lib.HexBytes `json:"address"`
	Amount        uint64       `json:"amount"`
	Committees    []uint64     `json:"committees"`
	NetAddress    string       `json:"netAddress"`
	OutputAddress lib.HexBytes `json:"outputAddress"`
	Compound      bool         `json:"compound"`
}

var _ lib.MessageI = &MessageUnstake{} // interface enforcement

// Check() validates the Message structure
func (x *MessageUnstake) Check() lib.ErrorI { return checkAddress(x.Address) }
func (x *MessageUnstake) Name() string      { return MessageUnstakeName }
func (x *MessageUnstake) New() lib.MessageI { return new(MessageUnstake) }
func (x *MessageUnstake) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageUnstake
func (x MessageUnstake) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonHexAddressMsg{Address: x.Address})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageUnstake
func (x *MessageUnstake) UnmarshalJSON(b []byte) (err error) {
	var j jsonHexAddressMsg
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageUnstake{Address: j.Address}
	return
}

type jsonHexAddressMsg struct {
	Address lib.HexBytes `json:"address"`
}

var _ lib.MessageI = &MessagePause{} // interface enforcement

// Check() validates the Message structure
func (x *MessagePause) Check() lib.ErrorI { return checkAddress(x.Address) }
func (x *MessagePause) Name() string      { return MessagePauseName }
func (x *MessagePause) New() lib.MessageI { return new(MessagePause) }
func (x *MessagePause) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessagePause
func (x MessagePause) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonHexAddressMsg{Address: x.Address})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessagePause
func (x *MessagePause) UnmarshalJSON(b []byte) (err error) {
	var j jsonHexAddressMsg
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessagePause{Address: j.Address}
	return
}

var _ lib.MessageI = &MessageUnpause{} // interface enforcement

// Check() validates the Message structure
func (x *MessageUnpause) Check() lib.ErrorI { return checkAddress(x.Address) }
func (x *MessageUnpause) Name() string      { return MessageUnpauseName }
func (x *MessageUnpause) New() lib.MessageI { return new(MessageUnpause) }
func (x *MessageUnpause) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageUnpause
func (x MessageUnpause) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonHexAddressMsg{Address: x.Address})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageUnpause
func (x *MessageUnpause) UnmarshalJSON(b []byte) (err error) {
	var j jsonHexAddressMsg
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageUnpause{Address: j.Address}
	return
}

var _ lib.MessageI = &MessageChangeParameter{} // interface enforcement

// Check() validates the Message structure
func (x *MessageChangeParameter) Check() lib.ErrorI {
	if err := checkAddress(x.Signer); err != nil {
		return err
	}
	if x.ParameterKey == "" {
		return ErrParamKeyEmpty()
	}
	if x.ParameterValue == nil {
		return ErrParamValueEmpty()
	}
	if err := checkStartEndHeight(x); err != nil {
		return err
	}
	if x.ProposalHash != "" {
		return ErrInvalidProposalHash()
	}
	return nil
}

// MarshalJSON() is the json.Marshaller implementation for MessageChangeParameter
func (x MessageChangeParameter) MarshalJSON() ([]byte, error) {
	a, err := lib.FromAny(x.ParameterValue)
	if err != nil {
		return nil, err
	}
	var parameterValue any
	switch p := a.(type) {
	case *lib.StringWrapper:
		parameterValue = p.Value
	case *lib.UInt64Wrapper:
		parameterValue = p.Value
	default:
		return nil, fmt.Errorf("unknown parameter type %T", p)
	}
	return json.Marshal(jsonMessageChangeParameter{
		ParameterSpace: x.ParameterSpace,
		ParameterKey:   x.ParameterKey,
		ParameterValue: parameterValue,
		StartHeight:    x.StartHeight,
		EndHeight:      x.EndHeight,
		Signer:         x.Signer,
		ProposalHash:   x.ProposalHash,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageChangeParameter
func (x *MessageChangeParameter) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageChangeParameter
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	var parameterValue proto.Message
	switch p := j.ParameterValue.(type) {
	case string:
		parameterValue = &lib.StringWrapper{Value: p}
	case uint64:
		parameterValue = &lib.UInt64Wrapper{Value: p}
	case float64:
		parameterValue = &lib.UInt64Wrapper{Value: uint64(p)}
	default:
		return fmt.Errorf("unknown parameter type %T", p)
	}
	a, err := lib.NewAny(parameterValue)
	if err != nil {
		return
	}
	*x = MessageChangeParameter{
		ParameterSpace: j.ParameterSpace,
		ParameterKey:   j.ParameterKey,
		ParameterValue: a,
		StartHeight:    j.StartHeight,
		EndHeight:      j.EndHeight,
		Signer:         j.Signer,
		ProposalHash:   j.ProposalHash,
	}
	return
}

type jsonMessageChangeParameter struct {
	ParameterSpace string       `json:"parameterSpace"`
	ParameterKey   string       `json:"parameterKey"`
	ParameterValue any          `json:"parameterValue"`
	StartHeight    uint64       `json:"startHeight"`
	EndHeight      uint64       `json:"endHeight"`
	Signer         lib.HexBytes `json:"signer"`
	ProposalHash   string       `json:"proposalHash,omitempty"`
}

func (x *MessageChangeParameter) Name() string      { return MessageChangeParameterName }
func (x *MessageChangeParameter) New() lib.MessageI { return new(MessageChangeParameter) }
func (x *MessageChangeParameter) Recipient() []byte { return nil }

var _ lib.MessageI = &MessageDAOTransfer{} // interface enforcement

// Check() validates the Message structure
func (x *MessageDAOTransfer) Check() lib.ErrorI {
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if err := checkStartEndHeight(x); err != nil {
		return err
	}
	if x.ProposalHash != "" {
		return ErrInvalidProposalHash()
	}
	return checkAmount(x.Amount)
}

func (x *MessageDAOTransfer) Name() string      { return MessageDAOTransferName }
func (x *MessageDAOTransfer) New() lib.MessageI { return new(MessageDAOTransfer) }
func (x *MessageDAOTransfer) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageDAOTransfer
func (x MessageDAOTransfer) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageDaoTransfer{
		Address:      x.Address,
		Amount:       x.Amount,
		StartHeight:  x.StartHeight,
		EndHeight:    x.EndHeight,
		ProposalHash: x.ProposalHash,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageDAOTransfer
func (x *MessageDAOTransfer) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageDaoTransfer
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageDAOTransfer{
		Address:      j.Address,
		Amount:       j.Amount,
		StartHeight:  j.StartHeight,
		EndHeight:    j.EndHeight,
		ProposalHash: j.ProposalHash,
	}
	return
}

type jsonMessageDaoTransfer struct {
	Address      lib.HexBytes `json:"address"`
	Amount       uint64       `json:"amount"`
	StartHeight  uint64       `json:"startHeight"`
	EndHeight    uint64       `json:"endHeight"`
	ProposalHash string       `json:"proposalHash,omitempty"`
}

var _ lib.MessageI = &MessageCertificateResults{} // interface enforcement

// Check() validates the Message structure
func (x *MessageCertificateResults) Check() lib.ErrorI {
	if x == nil {
		return ErrEmptyCertificateResults()
	}
	if err := x.Qc.CheckBasic(); err != nil {
		return err
	}
	results := x.Qc.Results
	if results == nil {
		return ErrEmptyCertificateResults()
	}
	if x.Qc.Block != nil {
		return lib.ErrNilBlock()
	}
	if err := checkChainId(x.Qc.Header.ChainId); err != nil {
		return err
	}
	if err := results.RewardRecipients.CheckBasic(); err != nil {
		return err
	}
	if results.RewardRecipients.NumberOfSamples != 0 {
		return ErrInvalidNumOfSamples()
	}
	if results.Checkpoint != nil {
		if len(results.Checkpoint.BlockHash) > 100 {
			return lib.ErrInvalidBlockHash()
		}
	}
	return checkOrders(results.Orders)
}

func (x *MessageCertificateResults) Name() string      { return MessageCertificateResultsName }
func (x *MessageCertificateResults) New() lib.MessageI { return new(MessageCertificateResults) }
func (x *MessageCertificateResults) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageProposal
func (x MessageCertificateResults) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageCertificateResults{
		Qc: x.Qc,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageProposal
func (x *MessageCertificateResults) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageCertificateResults
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageCertificateResults{
		Qc: j.Qc,
	}
	return
}

type jsonMessageCertificateResults struct {
	Qc *lib.QuorumCertificate `json:"qc"`
}

var _ lib.MessageI = &MessageSubsidy{} // interface enforcement

// Check() validates the Message structure
func (x *MessageSubsidy) Check() lib.ErrorI {
	if x == nil {
		return ErrInvalidSubisdy()
	}
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if len(x.Opcode) > 100 {
		return ErrInvalidOpcode()
	}
	return nil
}

func (x *MessageSubsidy) Name() string      { return MessageSubsidyName }
func (x *MessageSubsidy) New() lib.MessageI { return new(MessageSubsidy) }
func (x *MessageSubsidy) Recipient() []byte { return nil }

// MarshalJSON() is the json.Marshaller implementation for MessageSubsidy
func (x MessageSubsidy) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageSubsidy{
		Address: x.Address,
		ChainId: x.ChainId,
		Amount:  x.Amount,
		Opcode:  x.Opcode,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageSubsidy
func (x *MessageSubsidy) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageSubsidy
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageSubsidy{
		Address: x.Address,
		ChainId: j.ChainId,
		Amount:  j.Amount,
		Opcode:  j.Opcode,
	}
	return
}

type jsonMessageSubsidy struct {
	Address lib.HexBytes `json:"address"`
	ChainId uint64       `json:"chainID"`
	Amount  uint64       `json:"amount"`
	Opcode  lib.HexBytes `json:"opcode"`
}

var _ lib.MessageI = &MessageCreateOrder{} // interface enforcement

func (x *MessageCreateOrder) New() lib.MessageI { return new(MessageCreateOrder) }
func (x *MessageCreateOrder) Name() string      { return MessageCreateOrderName }
func (x *MessageCreateOrder) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageCreateOrder) Check() lib.ErrorI {
	if err := checkChainId(x.ChainId); err != nil {
		return err
	}
	if len(x.Data) > 100 {
		return ErrInvalidOpcode()
	}
	if x.AmountForSale == 0 || x.RequestedAmount == 0 {
		return ErrInvalidAmount()
	}
	if err := ensureEmpty(x.OrderId); err != nil {
		return err
	}
	return checkExternalAddress(x.SellerReceiveAddress)
}

// MarshalJSON() is the json.Marshaller implementation for MessageCreateOrder
func (x *MessageCreateOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageCreateOrder{
		ChainId:              x.ChainId,
		AmountForSale:        x.AmountForSale,
		Data:                 x.Data,
		RequestedAmount:      x.RequestedAmount,
		SellerReceiveAddress: x.SellerReceiveAddress,
		SellersSellAddress:   x.SellersSendAddress,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageCreateOrder
func (x *MessageCreateOrder) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageCreateOrder
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageCreateOrder{
		ChainId:              j.ChainId,
		Data:                 j.Data,
		AmountForSale:        j.AmountForSale,
		RequestedAmount:      j.RequestedAmount,
		SellerReceiveAddress: j.SellerReceiveAddress,
		SellersSendAddress:   j.SellersSellAddress,
	}
	return
}

type jsonMessageCreateOrder struct {
	ChainId              uint64       `json:"chainId"`
	AmountForSale        uint64       `json:"amountForSale"`
	Data                 lib.HexBytes `json:"data"`
	RequestedAmount      uint64       `json:"requestedAmount"`
	SellerReceiveAddress lib.HexBytes `json:"sellerReceiveAddress"`
	SellersSellAddress   lib.HexBytes `json:"sellersSendAddress"`
}

var _ lib.MessageI = &MessageEditOrder{} // interface enforcement

func (x *MessageEditOrder) New() lib.MessageI { return new(MessageEditOrder) }
func (x *MessageEditOrder) Name() string      { return MessageEditOrderName }
func (x *MessageEditOrder) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageEditOrder) Check() lib.ErrorI {
	if err := checkChainId(x.ChainId); err != nil {
		return err
	}
	if len(x.Data) > 100 {
		return ErrInvalidOpcode()
	}
	if x.AmountForSale == 0 || x.RequestedAmount == 0 {
		return ErrInvalidAmount()
	}
	return checkExternalAddress(x.SellerReceiveAddress)
}

// MarshalJSON() is the json.Marshaller implementation for MessageEditOrder
func (x *MessageEditOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageEditOrder{
		OrderId:              x.OrderId,
		ChainId:              x.ChainId,
		Data:                 x.Data,
		AmountForSale:        x.AmountForSale,
		RequestedAmount:      x.RequestedAmount,
		SellerReceiveAddress: x.SellerReceiveAddress,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditOrder
func (x *MessageEditOrder) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageEditOrder
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageEditOrder{
		OrderId:              j.OrderId,
		ChainId:              j.ChainId,
		Data:                 j.Data,
		AmountForSale:        j.AmountForSale,
		RequestedAmount:      j.RequestedAmount,
		SellerReceiveAddress: j.SellerReceiveAddress,
	}
	return
}

type jsonMessageEditOrder struct {
	OrderId              lib.HexBytes `json:"orderID"`
	ChainId              uint64       `json:"chainID"`
	Data                 lib.HexBytes `json:"data"`
	AmountForSale        uint64       `json:"amountForSale"`
	RequestedAmount      uint64       `json:"requestedAmount"`
	SellerReceiveAddress lib.HexBytes `json:"sellerReceiveAddress"`
}

var _ lib.MessageI = &MessageDeleteOrder{} // interface enforcement

func (x *MessageDeleteOrder) New() lib.MessageI { return new(MessageDeleteOrder) }
func (x *MessageDeleteOrder) Name() string      { return MessageDeleteOrderName }
func (x *MessageDeleteOrder) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageDeleteOrder) Check() lib.ErrorI { return checkChainId(x.ChainId) }

// MarshalJSON() is the json.Marshaller implementation for MessageEditOrder
func (x *MessageDeleteOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageDeleteOrder{
		OrderId: x.OrderId,
		ChainId: x.ChainId,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditOrder
func (x *MessageDeleteOrder) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageDeleteOrder
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageDeleteOrder{
		OrderId: j.OrderId,
		ChainId: j.ChainId,
	}
	return
}

type jsonMessageDeleteOrder struct {
	OrderId lib.HexBytes `json:"orderID"`
	ChainId uint64       `json:"chainID"`
}

var _ lib.MessageI = &MessageDexLimitOrder{} // interface enforcement

func (x *MessageDexLimitOrder) New() lib.MessageI { return new(MessageDexLimitOrder) }
func (x *MessageDexLimitOrder) Name() string      { return MessageDexLimitOrderName }
func (x *MessageDexLimitOrder) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageDexLimitOrder) Check() lib.ErrorI {
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if err := checkAmount(x.AmountForSale); err != nil {
		return err
	}
	if err := checkAmount(x.RequestedAmount); err != nil {
		return err
	}
	return checkChainId(x.ChainId)
}

// MarshalJSON() is the json.Marshaller implementation for MessageEditOrder
func (x *MessageDexLimitOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageDexLimitOrder{
		ChainId:            x.ChainId,
		AmountForSale:      x.AmountForSale,
		RequestedAmount:    x.RequestedAmount,
		SellersSendAddress: x.Address,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditOrder
func (x *MessageDexLimitOrder) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageDexLimitOrder
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageDexLimitOrder{
		ChainId:         j.ChainId,
		AmountForSale:   j.AmountForSale,
		RequestedAmount: j.RequestedAmount,
		Address:         j.SellersSendAddress,
	}
	return
}

type jsonMessageDexLimitOrder struct {
	ChainId            uint64       `json:"chainID"`
	AmountForSale      uint64       `json:"amountForSale"`
	RequestedAmount    uint64       `json:"requestedAmount"`
	SellersSendAddress lib.HexBytes `json:"sellerReceiveAddress"`
}

var _ lib.MessageI = &MessageDexLiquidityDeposit{} // interface enforcement

func (x *MessageDexLiquidityDeposit) New() lib.MessageI { return new(MessageDexLiquidityDeposit) }
func (x *MessageDexLiquidityDeposit) Name() string      { return MessageDexLiquidityDepositName }
func (x *MessageDexLiquidityDeposit) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageDexLiquidityDeposit) Check() lib.ErrorI {
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if err := checkAmount(x.Amount); err != nil {
		return err
	}
	return checkChainId(x.ChainId)
}

// MarshalJSON() is the json.Marshaller implementation for MessageEditOrder
func (x *MessageDexLiquidityDeposit) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageDexLiquidityDeposit{
		ChainId: x.ChainId,
		Amount:  x.Amount,
		Address: x.Address,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditOrder
func (x *MessageDexLiquidityDeposit) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageDexLiquidityDeposit
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageDexLiquidityDeposit{
		ChainId: j.ChainId,
		Amount:  j.Amount,
		Address: j.Address,
	}
	return
}

type jsonMessageDexLiquidityDeposit struct {
	ChainId uint64       `json:"chainID"`
	Amount  uint64       `json:"amount"`
	Address lib.HexBytes `json:"address"`
}

var _ lib.MessageI = &MessageDexLiquidityWithdraw{} // interface enforcement

func (x *MessageDexLiquidityWithdraw) New() lib.MessageI { return new(MessageDexLiquidityWithdraw) }
func (x *MessageDexLiquidityWithdraw) Name() string      { return MessageDexLiquidityWithdrawName }
func (x *MessageDexLiquidityWithdraw) Recipient() []byte { return nil }

// Check() validates the Message structure
func (x *MessageDexLiquidityWithdraw) Check() lib.ErrorI {
	if err := checkAddress(x.Address); err != nil {
		return err
	}
	if err := checkPercent(x.Percent); err != nil {
		return err
	}
	return checkChainId(x.ChainId)
}

// MarshalJSON() is the json.Marshaller implementation for MessageEditOrder
func (x *MessageDexLiquidityWithdraw) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonMessageDexLiquidityWithdraw{
		ChainId: x.ChainId,
		Percent: x.Percent,
		Address: x.Address,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for MessageEditOrder
func (x *MessageDexLiquidityWithdraw) UnmarshalJSON(b []byte) (err error) {
	var j jsonMessageDexLiquidityWithdraw
	if err = json.Unmarshal(b, &j); err != nil {
		return
	}
	*x = MessageDexLiquidityWithdraw{
		ChainId: j.ChainId,
		Percent: j.Percent,
		Address: j.Address,
	}
	return
}

type jsonMessageDexLiquidityWithdraw struct {
	ChainId uint64       `json:"chainID"`
	Percent uint64       `json:"percent"`
	Address lib.HexBytes `json:"address"`
}

func ensureEmpty(b []byte) lib.ErrorI {
	if len(b) != 0 {
		return ErrNotEmpty()
	}
	return nil
}

// checkAmount() validates the amount sent in the Message
func checkAmount(amount uint64) lib.ErrorI {
	if amount == 0 {
		return ErrInvalidAmount()
	}
	return nil
}

// checkPercent() validates the percent sent in the Message
func checkPercent(percent uint64) lib.ErrorI {
	if percent == 0 {
		return lib.ErrInvalidPercentAllocation()
	}
	if percent > 100 {
		return lib.ErrInvalidPercentAllocation()
	}
	return nil
}

// checkAddress() validates the address in the Message
func checkAddress(address []byte) lib.ErrorI {
	if address == nil {
		return ErrAddressEmpty()
	}
	if len(address) != crypto.AddressSize {
		return ErrAddressSize()
	}
	return nil
}

// checkExternalAddress() validates an address from an external blockchain
func checkExternalAddress(address []byte) lib.ErrorI {
	addressLen := len(address)
	if addressLen == 0 || addressLen > 255 {
		return ErrAddressSize()
	}
	return nil
}

// CheckNetAddress() validates the p2p address in the Message
func CheckNetAddress(netAddress string, isDelegate bool) lib.ErrorI {
	netAddressLen := len(netAddress)
	if isDelegate {
		if netAddressLen != 0 {
			return ErrInvalidNetAddressLen()
		}
		return nil
	}
	if netAddressLen < 1 || netAddressLen > 255 {
		return ErrInvalidNetAddressLen()
	}
	// ensure the net address is a valid
	if !lib.ValidNetURLInput(netAddress) {
		return lib.ErrInvalidStateNetAddress(netAddress)
	}
	return nil
}

// checkOutputAddress() validates the rewards address in the Message
func checkOutputAddress(output []byte) lib.ErrorI {
	if output == nil {
		return ErrOutputAddressEmpty()
	}
	if len(output) != crypto.AddressSize {
		return ErrOutputAddressSize()
	}
	return nil
}

// checkPubKey() validates the public key in the Message
func checkPubKey(publicKey []byte, delegate bool) lib.ErrorI {
	if publicKey == nil {
		return ErrPublicKeyEmpty()
	}
	// if actively participating in consensus
	if !delegate {
		// ensure the public key is a BLS key
		if len(publicKey) != crypto.BLS12381PubKeySize {
			return ErrPublicKeySize()
		}
	}
	return nil
}

// checkCommittees() validates the committees list in the message
func checkCommittees(committees []uint64) lib.ErrorI {
	numCommittees := len(committees)
	if numCommittees > 1000 || numCommittees == 0 {
		return ErrInvalidNumCommittees()
	}
	seen := make(map[uint64]struct{}, numCommittees)
	for _, committee := range committees {
		if _, dup := seen[committee]; dup {
			return ErrInvalidNumCommittees()
		}
		seen[committee] = struct{}{}
		if err := checkChainId(committee); err != nil {
			return err
		}
	}
	return nil
}

func checkChainId(i uint64) lib.ErrorI {
	for _, reserved := range ReservedIDs {
		if i == reserved {
			return ErrInvalidChainId()
		}
	}
	// ensure the chain id doesn't exceed max
	if i > MaxChainId {
		return ErrInvalidChainId()
	}
	return nil
}

// checkStartEndHeight() validates the start/end height of the message
func checkStartEndHeight(proposal GovProposal) lib.ErrorI {
	blockRange := proposal.GetEndHeight() - proposal.GetStartHeight()
	if blockRange > 10000 || blockRange <= 0 {
		return ErrInvalidBlockRange()
	}
	return nil
}

// checkOrders() validates the (swap) orders within the transaction
func checkOrders(orders *lib.Orders) lib.ErrorI {
	if orders != nil {
		// ensure no duplicate lock orders
		deDupe := lib.NewDeDuplicator[string]()
		for _, lockOrder := range orders.LockOrders {
			if lockOrder == nil {
				return ErrInvalidLockOrder()
			}
			if found := deDupe.Found(lib.BytesToString(lockOrder.OrderId)); found {
				return ErrDuplicateLockOrder()
			}
			if err := checkAddress(lockOrder.BuyerReceiveAddress); err != nil {
				return err
			}
			if lockOrder.BuyerChainDeadline == 0 {
				return ErrInvalidBuyerDeadline()
			}
		}
		// ensure no duplicate reset orders
		deDupe = lib.NewDeDuplicator[string]()
		for _, resetOrder := range orders.ResetOrders {
			if found := deDupe.Found(lib.BytesToString(resetOrder)); found {
				return ErrInvalidCloseOrder()
			}
		}
		// ensure no duplicate close orders
		deDupe = lib.NewDeDuplicator[string]()
		for _, closeOrder := range orders.CloseOrders {
			if found := deDupe.Found(lib.BytesToString(closeOrder)); found {
				return ErrInvalidCloseOrder()
			}
		}
	}
	return nil
}

// messageFromTxJSON() extracts a lib.MessageI from a transaction json
func messageFromTxJSON(txJSONBytes []byte) (message lib.MessageI, tx *lib.Transaction, err lib.ErrorI) {
	// create a new transaction object reference to ensure a non-nil transaction
	tx = new(lib.Transaction)
	// populate the object ref with the bytes of the transaction
	if err = lib.UnmarshalJSON(txJSONBytes, tx); err != nil {
		// exit with error
		return
	}
	// perform basic validations against the tx object
	if err = tx.CheckBasic(); err != nil {
		// exit with error
		return
	}
	// extract the message from a protobuf any
	p, err := lib.FromAny(tx.Msg)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return
	}
	// cast the proto message to a Message interface that may be interpreted
	message, castOk := p.(lib.MessageI)
	// if cast fails, throw an error
	if !castOk {
		// exit with invalid cast
		return nil, nil, ErrInvalidTxMessage()
	}
	// do stateless checks on the message
	if err = message.Check(); err != nil {
		// exit with error
		return
	}
	// exit
	return
}

// TxHashFromJSON converts the json transaction into a proto tx hash
func TxHashFromJSON(transactionJSON json.RawMessage) (txHash string, err lib.ErrorI) {
	// extract the message from the transaction
	_, tx, err := messageFromTxJSON(transactionJSON)
	// if an error occurred during the extraction
	if err != nil {
		// exit with error
		return
	}
	// convert into proto bytes
	protoBytes, err := lib.Marshal(tx)
	// if an error occurred during the encoding
	if err != nil {
		// exit with error
		return
	}
	// exit with hash
	return crypto.HashString(protoBytes), nil
}
