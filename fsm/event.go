package fsm

import (
	"github.com/canopy-network/canopy/lib"
)

// EventReward() adds a validator rewarded event
func (s *StateMachine) EventReward(address []byte, amount, chainId uint64) lib.ErrorI {
	return s.addEvent(lib.EventTypeReward, &lib.EventReward{Amount: amount}, address, chainId)
}

// EventSlash() adds a validator slashed event
func (s *StateMachine) EventSlash(address []byte, amount uint64) lib.ErrorI {
	return s.addEvent(lib.EventTypeSlash, &lib.EventSlash{Amount: amount}, address)
}

// EventAutoPause() adds a validator automatically paused event
func (s *StateMachine) EventAutoPause(address []byte) lib.ErrorI {
	return s.addEvent(lib.EventTypeAutoPause, &lib.EventAutoPause{}, address)
}

// EventAutoBeginUnstaking() adds a validator automatically begin the unstaking process event
func (s *StateMachine) EventAutoBeginUnstaking(address []byte) lib.ErrorI {
	return s.addEvent(lib.EventTypeAutoBeginUnstaking, &lib.EventAutoBeginUnstaking{}, address)
}

// EventFinishUnstaking() adds a validator completing the unstaking process event
func (s *StateMachine) EventFinishUnstaking(address []byte) lib.ErrorI {
	return s.addEvent(lib.EventTypeFinishUnstaking, &lib.EventFinishUnstaking{}, address)
}

// EventOrderBookSwap() adds an order book token swap event to the indexer
func (s *StateMachine) EventOrderBookSwap(order *lib.SellOrder) lib.ErrorI {
	return s.addEvent(lib.EventTypeOrderBookSwap, &lib.EventOrderBookSwap{
		SoldAmount:           order.AmountForSale,
		BoughtAmount:         order.RequestedAmount,
		Data:                 order.Data,
		SellerReceiveAddress: order.SellerReceiveAddress,
		BuyerSendAddress:     order.BuyerSendAddress,
		SellersSendAddress:   order.SellersSendAddress,
		OrderId:              order.Id,
	}, order.BuyerReceiveAddress, order.Committee)
}

// EventDexSwap() adds an AMM token swap event to the indexer
func (s *StateMachine) EventDexSwap(address, orderId []byte, soldAmount, boughtAmount, chainId uint64, localOrigin, success bool) lib.ErrorI {
	return s.addEvent(lib.EventTypeDexSwap, &lib.EventDexSwap{
		SoldAmount:   soldAmount,
		BoughtAmount: boughtAmount,
		LocalOrigin:  localOrigin,
		Success:      success,
		OrderId:      orderId,
	}, address, chainId)
}

// EventDexLiquidityDeposit() adds an AMM liquidity deposit event to the indexer
func (s *StateMachine) EventDexLiquidityDeposit(address, orderId []byte, amount, pointsAdded, chainId uint64, localOrigin bool) lib.ErrorI {
	return s.addEvent(lib.EventTypeDexLiquidityDeposit, &lib.EventDexLiquidityDeposit{
		Amount:      amount,
		LocalOrigin: localOrigin,
		OrderId:     orderId,
		Points:      pointsAdded,
	}, address, chainId)
}

// EventDexLiquidityWithdraw() adds a liquidity withdraw event to the indexer
func (s *StateMachine) EventDexLiquidityWithdraw(address, orderId []byte, localAmount, remoteAmount, pointsBurned, chainId uint64) lib.ErrorI {
	return s.addEvent(lib.EventTypeDexLiquidityWithdraw, &lib.EventDexLiquidityWithdrawal{
		LocalAmount:  localAmount,
		RemoteAmount: remoteAmount,
		OrderId:      orderId,
		PointsBurned: pointsBurned,
	}, address, chainId)
}

// addEvent() is a helper function that creates an event with common fields set and adds it to the tracker
func (s *StateMachine) addEvent(eventType lib.EventType, msg interface{}, address []byte, chainId ...uint64) lib.ErrorI {
	e := &lib.Event{
		EventType: string(eventType),
		Height:    s.Height(),
		Reference: s.events.GetReference(),
		Address:   address,
	}

	// Set the oneof message field based on event type
	switch eventType {
	case lib.EventTypeReward:
		e.Msg = &lib.Event_Reward{Reward: msg.(*lib.EventReward)}
	case lib.EventTypeSlash:
		e.Msg = &lib.Event_Slash{Slash: msg.(*lib.EventSlash)}
	case lib.EventTypeAutoPause:
		e.Msg = &lib.Event_AutoPause{AutoPause: msg.(*lib.EventAutoPause)}
	case lib.EventTypeAutoBeginUnstaking:
		e.Msg = &lib.Event_AutoBeginUnstaking{AutoBeginUnstaking: msg.(*lib.EventAutoBeginUnstaking)}
	case lib.EventTypeFinishUnstaking:
		e.Msg = &lib.Event_FinishUnstaking{FinishUnstaking: msg.(*lib.EventFinishUnstaking)}
	case lib.EventTypeDexSwap:
		e.Msg = &lib.Event_DexSwap{DexSwap: msg.(*lib.EventDexSwap)}
	case lib.EventTypeDexLiquidityDeposit:
		e.Msg = &lib.Event_DexLiquidityDeposit{DexLiquidityDeposit: msg.(*lib.EventDexLiquidityDeposit)}
	case lib.EventTypeDexLiquidityWithdraw:
		e.Msg = &lib.Event_DexLiquidityWithdrawal{DexLiquidityWithdrawal: msg.(*lib.EventDexLiquidityWithdrawal)}
	case lib.EventTypeOrderBookSwap:
		e.Msg = &lib.Event_OrderBookSwap{OrderBookSwap: msg.(*lib.EventOrderBookSwap)}
	}

	// optionally set chainId if provided
	if len(chainId) > 0 {
		e.ChainId = chainId[0]
	}

	// add the event to the tracker
	return s.events.Add(e)
}
