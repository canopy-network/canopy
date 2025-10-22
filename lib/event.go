package lib

import (
	"encoding/json"
)

type EventType string

const (
	EventStageBeginBlock = "begin_block"
	EventStageEndBlock   = "end_block"

	EventTypeReward               EventType = "reward"
	EventTypeSlash                EventType = "slash"
	EventTypeAutoPause            EventType = "automatic-pause"
	EventTypeAutoBeginUnstaking   EventType = "automatic-begin-unstaking"
	EventTypeFinishUnstaking      EventType = "automatic-finish-unstaking"
	EventTypeDexSwap              EventType = "dex-swap"
	EventTypeDexLiquidityDeposit  EventType = "dex-liquidity-deposit"
	EventTypeDexLiquidityWithdraw EventType = "dex-liquidity-withdraw"
	EventTypeOrderBookSwap        EventType = "order-book-swap"
)

type EventsTracker struct {
	Reference string // the 'begin_block' / tx_hash / 'end_block' -> reference for events
	Events    Events // the actual events
}

// Add() adds an event to the tracker
func (t *EventsTracker) Add(event *Event) (e ErrorI) {
	if t == nil {
		return ErrEmptyEventsTracker()
	}
	t.Events = append(t.Events, event)
	return
}

// Refer() sets a reference string for the event tracker
func (t *EventsTracker) Refer(s string) {
	if t == nil {
		return
	}
	t.Reference = s
}

// GetReference() is an accessor for the reference string
func (t *EventsTracker) GetReference() string {
	if t == nil {
		return ""
	}
	return t.Reference
}

// Reset() resets the event tracker and returns the captured events
func (t *EventsTracker) Reset() (e Events) {
	if t == nil {
		return
	}
	// save
	e = t.Events
	// reset
	t.Events, t.Reference = nil, ""
	// exit
	return
}

type Events []*Event

func (e *Events) Len() int      { return len(*e) }
func (e *Events) New() Pageable { return &Events{} }

// eventJSON represents the JSON structure for Event marshalling/unmarshalling
type eventJSON struct {
	EventType   string          `json:"eventType"`
	Msg         json.RawMessage `json:"msg,omitempty"`
	Height      uint64          `json:"height"`
	Reference   string          `json:"reference"`
	ChainId     uint64          `json:"chainId"`
	BlockHeight uint64          `json:"blockHeight,omitempty"`
	BlockHash   HexBytes        `json:"blockHash,omitempty"`
	Address     HexBytes        `json:"address,omitempty"`
}

// MarshalJSON implements custom JSON marshalling for Event, converting []byte fields to HexBytes
func (e *Event) MarshalJSON() ([]byte, error) {
	if e == nil {
		return json.Marshal(nil)
	}

	// Marshal the Msg field separately
	var msgBytes []byte
	var err error
	if e.Msg != nil {
		switch msg := e.Msg.(type) {
		case *Event_Reward:
			msgBytes, err = json.Marshal(msg.Reward)
		case *Event_Slash:
			msgBytes, err = json.Marshal(msg.Slash)
		case *Event_DexLiquidityDeposit:
			msgBytes, err = json.Marshal(msg.DexLiquidityDeposit)
		case *Event_DexLiquidityWithdrawal:
			msgBytes, err = json.Marshal(msg.DexLiquidityWithdrawal)
		case *Event_DexSwap:
			msgBytes, err = json.Marshal(msg.DexSwap)
		case *Event_OrderBookSwap:
			msgBytes, err = json.Marshal(msg.OrderBookSwap)
		case *Event_AutoPause:
			msgBytes, err = json.Marshal(msg.AutoPause)
		case *Event_AutoBeginUnstaking:
			msgBytes, err = json.Marshal(msg.AutoBeginUnstaking)
		case *Event_FinishUnstaking:
			msgBytes, err = json.Marshal(msg.FinishUnstaking)
		}
		if err != nil {
			return nil, err
		}
	}

	temp := eventJSON{
		EventType:   e.EventType,
		Msg:         msgBytes,
		Height:      e.Height,
		Reference:   e.Reference,
		ChainId:     e.ChainId,
		BlockHeight: e.BlockHeight,
		BlockHash:   e.BlockHash,
		Address:     e.Address,
	}

	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshalling for Event, converting HexBytes to []byte fields
func (e *Event) UnmarshalJSON(data []byte) error {
	var temp eventJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set basic fields
	e.EventType = temp.EventType
	e.Height = temp.Height
	e.Reference = temp.Reference
	e.ChainId = temp.ChainId
	e.BlockHeight = temp.BlockHeight
	e.BlockHash = temp.BlockHash
	e.Address = temp.Address

	// Handle the Msg field based on EventType
	if len(temp.Msg) > 0 {
		switch temp.EventType {
		case string(EventTypeReward):
			var reward EventReward
			if err := json.Unmarshal(temp.Msg, &reward); err != nil {
				return err
			}
			e.Msg = &Event_Reward{Reward: &reward}
		case string(EventTypeSlash):
			var slash EventSlash
			if err := json.Unmarshal(temp.Msg, &slash); err != nil {
				return err
			}
			e.Msg = &Event_Slash{Slash: &slash}
		case string(EventTypeAutoPause):
			var autoPause EventAutoPause
			if err := json.Unmarshal(temp.Msg, &autoPause); err != nil {
				return err
			}
			e.Msg = &Event_AutoPause{AutoPause: &autoPause}
		case string(EventTypeAutoBeginUnstaking):
			var autoBeginUnstaking EventAutoBeginUnstaking
			if err := json.Unmarshal(temp.Msg, &autoBeginUnstaking); err != nil {
				return err
			}
			e.Msg = &Event_AutoBeginUnstaking{AutoBeginUnstaking: &autoBeginUnstaking}
		case string(EventTypeFinishUnstaking):
			var finishUnstaking EventFinishUnstaking
			if err := json.Unmarshal(temp.Msg, &finishUnstaking); err != nil {
				return err
			}
			e.Msg = &Event_FinishUnstaking{FinishUnstaking: &finishUnstaking}
		case string(EventTypeDexSwap):
			var dexSwap EventDexSwap
			if err := json.Unmarshal(temp.Msg, &dexSwap); err != nil {
				return err
			}
			e.Msg = &Event_DexSwap{DexSwap: &dexSwap}
		case string(EventTypeDexLiquidityDeposit):
			var dexLiquidityDeposit EventDexLiquidityDeposit
			if err := json.Unmarshal(temp.Msg, &dexLiquidityDeposit); err != nil {
				return err
			}
			e.Msg = &Event_DexLiquidityDeposit{DexLiquidityDeposit: &dexLiquidityDeposit}
		case string(EventTypeDexLiquidityWithdraw):
			var dexLiquidityWithdraw EventDexLiquidityWithdrawal
			if err := json.Unmarshal(temp.Msg, &dexLiquidityWithdraw); err != nil {
				return err
			}
			e.Msg = &Event_DexLiquidityWithdrawal{DexLiquidityWithdrawal: &dexLiquidityWithdraw}
		case string(EventTypeOrderBookSwap):
			var orderBookSwap EventOrderBookSwap
			if err := json.Unmarshal(temp.Msg, &orderBookSwap); err != nil {
				return err
			}
			e.Msg = &Event_OrderBookSwap{OrderBookSwap: &orderBookSwap}
		}
	}

	return nil
}

// eventOrderBookSwapJSON represents the JSON structure for EventOrderBookSwap marshalling/unmarshalling
type eventOrderBookSwapJSON struct {
	SoldAmount           uint64   `json:"soldAmount,omitempty"`
	BoughtAmount         uint64   `json:"boughtAmount,omitempty"`
	Data                 HexBytes `json:"data,omitempty"`
	SellerReceiveAddress HexBytes `json:"sellerReceiveAddress,omitempty"`
	BuyerSendAddress     HexBytes `json:"buyerSendAddress,omitempty"`
	SellersSendAddress   HexBytes `json:"sellersSendAddress,omitempty"`
	OrderId              HexBytes `json:"orderId,omitempty"`
}

// MarshalJSON implements custom JSON marshalling for EventOrderBookSwap, converting []byte fields to HexBytes
func (e *EventOrderBookSwap) MarshalJSON() ([]byte, error) {
	if e == nil {
		return json.Marshal(nil)
	}

	temp := eventOrderBookSwapJSON{
		SoldAmount:           e.SoldAmount,
		BoughtAmount:         e.BoughtAmount,
		Data:                 e.Data,
		SellerReceiveAddress: e.SellerReceiveAddress,
		BuyerSendAddress:     e.BuyerSendAddress,
		SellersSendAddress:   e.SellersSendAddress,
		OrderId:              e.OrderId,
	}

	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshalling for EventOrderBookSwap, converting HexBytes to []byte fields
func (e *EventOrderBookSwap) UnmarshalJSON(data []byte) error {
	var temp eventOrderBookSwapJSON

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the fields
	e.SoldAmount = temp.SoldAmount
	e.BoughtAmount = temp.BoughtAmount
	e.Data = temp.Data
	e.SellerReceiveAddress = temp.SellerReceiveAddress
	e.BuyerSendAddress = temp.BuyerSendAddress
	e.SellersSendAddress = temp.SellersSendAddress
	e.OrderId = temp.OrderId

	return nil
}

// eventDexSwap represents the JSON structure for EventDexSwap marshalling/unmarshalling
type eventDexSwap struct {
	SoldAmount   uint64   `json:"soldAmount"`
	BoughtAmount uint64   `json:"boughtAmount"`
	LocalOrigin  bool     `json:"localOrigin"`
	Success      bool     `json:"success"`
	OrderId      HexBytes `json:"orderId"`
}

// MarshalJSON implements custom JSON marshalling for EventDexSwap, converting []byte fields to HexBytes
func (e EventDexSwap) MarshalJSON() ([]byte, error) {
	temp := eventDexSwap{
		SoldAmount:   e.SoldAmount,
		BoughtAmount: e.BoughtAmount,
		LocalOrigin:  e.LocalOrigin,
		Success:      e.Success,
		OrderId:      e.OrderId,
	}
	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshalling for EventDexSwap, converting HexBytes fields to []byte
func (e *EventDexSwap) UnmarshalJSON(b []byte) error {
	temp := eventDexSwap{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	*e = EventDexSwap{
		SoldAmount:   temp.SoldAmount,
		BoughtAmount: temp.BoughtAmount,
		LocalOrigin:  temp.LocalOrigin,
		Success:      temp.Success,
		OrderId:      temp.OrderId,
	}
	return nil
}

// eventLiquidityDeposit represents the JSON structure for EventLiquidityDeposit marshalling/unmarshalling
type eventDexLiquidityDeposit struct {
	Amount      uint64   `json:"amount"`
	LocalOrigin bool     `json:"localOrigin"`
	OrderId     HexBytes `json:"orderId"`
}

// MarshalJSON() implements custom JSON marshalling for EventLiquidityDeposit, converting []byte fields to HexBytes
func (e EventDexLiquidityDeposit) MarshalJSON() ([]byte, error) {
	temp := eventDexLiquidityDeposit{
		Amount:      e.Amount,
		LocalOrigin: e.LocalOrigin,
		OrderId:     e.OrderId,
	}
	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshalling for EventDexLiquidityDeposit, converting HexBytes fields to []byte
func (e *EventDexLiquidityDeposit) UnmarshalJSON(b []byte) error {
	temp := eventDexLiquidityDeposit{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	*e = EventDexLiquidityDeposit{
		Amount:      temp.Amount,
		LocalOrigin: temp.LocalOrigin,
		OrderId:     temp.OrderId,
	}
	return nil
}

// eventLiquidityDeposit represents the JSON structure for EventLiquidityDeposit marshalling/unmarshalling
type eventDexLiquidityWithdrawal struct {
	LocalAmount  uint64   `json:"localAmount"`
	RemoteAmount uint64   `json:"remoteAmount"`
	OrderId      HexBytes `json:"orderId"`
}

// MarshalJSON() implements custom JSON marshalling for EventDexLiquidityWithdrawal, converting []byte fields to HexBytes
func (e EventDexLiquidityWithdrawal) MarshalJSON() ([]byte, error) {
	temp := eventDexLiquidityWithdrawal{
		LocalAmount:  e.LocalAmount,
		RemoteAmount: e.RemoteAmount,
		OrderId:      e.OrderId,
	}
	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshalling for EventDexLiquidityWithdrawal, converting HexBytes fields to []byte
func (e *EventDexLiquidityWithdrawal) UnmarshalJSON(b []byte) error {
	temp := eventDexLiquidityWithdrawal{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	*e = EventDexLiquidityWithdrawal{
		LocalAmount:  temp.LocalAmount,
		RemoteAmount: temp.RemoteAmount,
		OrderId:      temp.OrderId,
	}
	return nil
}
