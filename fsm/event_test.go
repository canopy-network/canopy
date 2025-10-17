package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestEventReward(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		amount   uint64
		chainId  uint64
		expected string
		error    string
	}{
		{
			name:     "valid reward event",
			detail:   "successfully adds a validator reward event",
			address:  newTestAddressBytes(t),
			amount:   100,
			chainId:  1,
			expected: string(lib.EventTypeReward),
		},
		{
			name:     "zero amount reward",
			detail:   "adds a reward event with zero amount",
			address:  newTestAddressBytes(t),
			amount:   0,
			chainId:  1,
			expected: string(lib.EventTypeReward),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventReward(test.address, test.amount, test.chainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
			require.Equal(t, test.chainId, events[0].ChainId)
		})
	}
}

func TestEventSlash(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		amount   uint64
		expected string
		error    string
	}{
		{
			name:     "valid slash event",
			detail:   "successfully adds a validator slash event",
			address:  newTestAddressBytes(t),
			amount:   50,
			expected: string(lib.EventTypeSlash),
		},
		{
			name:     "zero amount slash",
			detail:   "adds a slash event with zero amount",
			address:  newTestAddressBytes(t),
			amount:   0,
			expected: string(lib.EventTypeSlash),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventSlash(test.address, test.amount)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
		})
	}
}

func TestEventAutoPause(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		expected string
		error    string
	}{
		{
			name:     "valid auto pause event",
			detail:   "successfully adds a validator auto pause event",
			address:  newTestAddressBytes(t),
			expected: string(lib.EventTypeAutoPause),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventAutoPause(test.address)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
		})
	}
}

func TestEventAutoBeginUnstaking(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		expected string
		error    string
	}{
		{
			name:     "valid auto begin unstaking event",
			detail:   "successfully adds a validator auto begin unstaking event",
			address:  newTestAddressBytes(t),
			expected: string(lib.EventTypeAutoBeginUnstaking),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventAutoBeginUnstaking(test.address)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
		})
	}
}

func TestEventFinishUnstaking(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		expected string
		error    string
	}{
		{
			name:     "valid finish unstaking event",
			detail:   "successfully adds a validator finish unstaking event",
			address:  newTestAddressBytes(t),
			expected: string(lib.EventTypeFinishUnstaking),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventFinishUnstaking(test.address)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
		})
	}
}

func TestEventOrderBookSwap(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		order    *lib.SellOrder
		expected string
		error    string
	}{
		{
			name:   "valid order book swap event",
			detail: "successfully adds an order book swap event",
			order: &lib.SellOrder{
				Id:                   []byte("order123"),
				AmountForSale:        100,
				RequestedAmount:      200,
				Data:                 []byte("swap_data"),
				SellerReceiveAddress: newTestAddressBytes(t),
				BuyerSendAddress:     newTestAddressBytes(t, 2),
				SellersSendAddress:   newTestAddressBytes(t, 3),
				BuyerReceiveAddress:  newTestAddressBytes(t, 4),
				Committee:            1,
			},
			expected: string(lib.EventTypeOrderBookSwap),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventOrderBookSwap(test.order)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.order.BuyerReceiveAddress, events[0].Address)
			require.Equal(t, test.order.Committee, events[0].ChainId)
		})
	}
}

func TestEventDexSwap(t *testing.T) {
	tests := []struct {
		name         string
		detail       string
		address      []byte
		soldAmount   uint64
		boughtAmount uint64
		chainId      uint64
		inbound      bool
		success      bool
		expected     string
		error        string
	}{
		{
			name:         "valid successful dex swap event",
			detail:       "successfully adds a successful dex swap event",
			address:      newTestAddressBytes(t),
			soldAmount:   100,
			boughtAmount: 95,
			chainId:      1,
			inbound:      true,
			success:      true,
			expected:     string(lib.EventTypeDexSwap),
		},
		{
			name:         "valid failed dex swap event",
			detail:       "successfully adds a failed dex swap event",
			address:      newTestAddressBytes(t),
			soldAmount:   100,
			boughtAmount: 0,
			chainId:      1,
			inbound:      false,
			success:      false,
			expected:     string(lib.EventTypeDexSwap),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventDexSwap(test.address, test.soldAmount, test.boughtAmount, test.chainId, test.inbound, test.success)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
			require.Equal(t, test.chainId, events[0].ChainId)
		})
	}
}

func TestEventDexLiquidityDeposit(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		address  []byte
		amount   uint64
		chainId  uint64
		inbound  bool
		expected string
		error    string
	}{
		{
			name:     "valid liquidity deposit event",
			detail:   "successfully adds a liquidity deposit event",
			address:  newTestAddressBytes(t),
			amount:   1000,
			chainId:  1,
			inbound:  true,
			expected: string(lib.EventTypeDexLiquidityDeposit),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventDexLiquidityDeposit(test.address, test.amount, test.chainId, test.inbound)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
			require.Equal(t, test.chainId, events[0].ChainId)
		})
	}
}

func TestEventDexLiquidityWithdraw(t *testing.T) {
	tests := []struct {
		name         string
		detail       string
		address      []byte
		localAmount  uint64
		remoteAmount uint64
		chainId      uint64
		expected     string
		error        string
	}{
		{
			name:         "valid liquidity withdraw event",
			detail:       "successfully adds a liquidity withdraw event",
			address:      newTestAddressBytes(t),
			localAmount:  500,
			remoteAmount: 250,
			chainId:      1,
			expected:     string(lib.EventTypeDexLiquidityWithdraw),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.EventDexLiquidityWithdraw(test.address, test.localAmount, test.remoteAmount, test.chainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, test.expected, events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
			require.Equal(t, test.chainId, events[0].ChainId)
		})
	}
}

func TestAddEvent(t *testing.T) {
	tests := []struct {
		name      string
		detail    string
		eventType lib.EventType
		msg       proto.Message
		address   []byte
		chainId   []uint64
		error     string
	}{
		{
			name:      "valid event with chain id",
			detail:    "successfully adds an event with chain id",
			eventType: lib.EventTypeReward,
			msg:       &lib.EventReward{Amount: 100},
			address:   newTestAddressBytes(t),
			chainId:   []uint64{1},
		},
		{
			name:      "valid event without chain id",
			detail:    "successfully adds an event without chain id",
			eventType: lib.EventTypeSlash,
			msg:       &lib.EventSlash{Amount: 50},
			address:   newTestAddressBytes(t),
			chainId:   nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// execute the function call
			err := sm.addEvent(test.eventType, test.msg, test.address, test.chainId...)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// verify event was added
			events := sm.events.Events
			require.Len(t, events, 1)
			require.Equal(t, string(test.eventType), events[0].EventType)
			require.Equal(t, test.address, events[0].Address)
			require.Equal(t, sm.Height(), events[0].Height)
			if len(test.chainId) > 0 {
				require.Equal(t, test.chainId[0], events[0].ChainId)
			} else {
				require.Equal(t, uint64(0), events[0].ChainId)
			}
		})
	}
}