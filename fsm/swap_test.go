package fsm

import (
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHandleCommitteeSwaps(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		preset          []*lib.SellOrder
		orders          *lib.Orders
		alreadyAccepted bool
		noBuyer         bool
		notFound        bool
	}{
		{
			name:   "lock order already accepted",
			detail: "the lock order cannot be claimed as its already reserved",
			preset: []*lib.SellOrder{
				{
					Committee:           lib.CanopyChainId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t),
					SellersSendAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				LockOrders: []*lib.LockOrder{
					{
						OrderId:             0,
						BuyerReceiveAddress: newTestAddressBytes(t, 1),
						BuyerChainDeadline:  100,
					},
				},
			},
			alreadyAccepted: true,
		},
		{
			name:   "reset failed, order not found",
			detail: "can't reset an order that doesn't exist",
			preset: []*lib.SellOrder{
				{
					Committee:           lib.CanopyChainId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t),
					SellersSendAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				ResetOrders: []uint64{1},
			},
			notFound: true,
		},
		{
			name:   "close failed, no buyer",
			detail: "can't close an order that doesn't have a buyer",
			preset: []*lib.SellOrder{
				{
					Committee:          lib.CanopyChainId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				CloseOrders: []uint64{0},
			},
			noBuyer: true,
		},
		{
			name:   "buy, reset, sell",
			detail: "test buy, reset, and sell without error",
			preset: []*lib.SellOrder{
				{
					Committee:          lib.CanopyChainId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t),
				},
				{
					Committee:           lib.CanopyChainId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:  newTestAddressBytes(t),
				},
				{
					Committee:           lib.CanopyChainId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				LockOrders: []*lib.LockOrder{
					{
						OrderId:             0,
						BuyerReceiveAddress: newTestAddressBytes(t, 1),
						BuyerChainDeadline:  100,
					},
				},
				ResetOrders: []uint64{1},
				CloseOrders: []uint64{2},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var escrowPoolBalance uint64
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the sell orders
			for _, preset := range test.preset {
				_, err := sm.CreateOrder(preset, lib.CanopyChainId)
				require.NoError(t, err)
				// simulate the escrow supply
				escrowPoolBalance += preset.AmountForSale
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+types.EscrowPoolAddend, preset.AmountForSale))
			}
			// execute the function call
			sm.HandleCommitteeSwaps(test.orders, lib.CanopyChainId)
			// validate the lock orders
			for _, lockOrder := range test.orders.LockOrders {
				// get the order
				order, e := sm.GetOrder(lockOrder.OrderId, lib.CanopyChainId)
				require.NoError(t, e)
				// if the lock order is already accepted
				if test.alreadyAccepted {
					require.NotEqual(t, lockOrder.BuyerReceiveAddress, order.BuyerReceiveAddress)
				} else {
					// validate the update of the 'buy' fields
					require.Equal(t, lockOrder.BuyerReceiveAddress, order.BuyerReceiveAddress)
					require.Equal(t, lockOrder.BuyerChainDeadline, order.BuyerChainDeadline)
				}
			}
			// validate the reset orders
			for _, resetOrderId := range test.orders.ResetOrders {
				// get the order
				order, e := sm.GetOrder(resetOrderId, lib.CanopyChainId)
				// if order not found to be reset
				if test.notFound {
					require.ErrorContains(t, e, "not found")
				} else {
					require.NoError(t, e)
					// validate the update of the 'buy' fields
					require.Empty(t, order.BuyerReceiveAddress)
					require.Zero(t, order.BuyerChainDeadline)
				}
			}
			var balanceRemovedFromPool uint64
			// validate the close orders
			for _, closeOrder := range test.orders.CloseOrders {
				// define convenience variable for order
				order := test.preset[closeOrder]
				// validate the deletion of the order
				_, e := sm.GetOrder(closeOrder, lib.CanopyChainId)
				// if order no buyer to close
				if test.noBuyer {
					require.NoError(t, e)
				} else {
					require.ErrorContains(t, e, "not found")
					// validate the addition of funds to the buyer
					accountBalance, e := sm.GetAccountBalance(crypto.NewAddress(order.BuyerReceiveAddress))
					require.NoError(t, e)
					require.Equal(t, order.AmountForSale, accountBalance)
					balanceRemovedFromPool += order.AmountForSale
				}
			}
			// validate the removal of funds from the escrow pool
			balance, e := sm.GetPoolBalance(lib.CanopyChainId + types.EscrowPoolAddend)
			require.NoError(t, e)
			require.Equal(t, escrowPoolBalance-balanceRemovedFromPool, balance)
		})
	}
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		expected []*lib.SellOrder
	}{
		{
			name:   "create sell order",
			detail: "create sell order",
			expected: []*lib.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "create sell order for two different committees",
			detail: "create sell order for another committee",
			expected: []*lib.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "id creation order",
			detail: "test the id creation order",
			expected: []*lib.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   1, // only used for validation
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   1, // only used for validation
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			for _, expected := range test.expected {
				// execute the function call
				_, err := sm.CreateOrder(expected, expected.Committee)
				require.NoError(t, err)
				// get the order
				got, err := sm.GetOrder(expected.Id, expected.Committee)
				require.NoError(t, err)
				// compare got vs expected
				require.EqualExportedValues(t, expected, got)
			}
		})
	}
}

func TestEditOrder(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		preset   *lib.SellOrder
		expected *lib.SellOrder
		error    string
	}{
		{
			name:   "order not found",
			detail: "order not preset so no order id is found",
			expected: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        101,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			error: "not found",
		},
		{
			name:   "update amount",
			detail: "update the amount for sale without error",
			preset: &lib.SellOrder{
				Committee:            lib.CanopyChainId,
				AmountForSale:        100,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			expected: &lib.SellOrder{
				Committee:            lib.CanopyChainId,
				AmountForSale:        101,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				_, err := sm.CreateOrder(test.preset, test.preset.Committee)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.EditOrder(test.expected, lib.CanopyChainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the order
			got, err := sm.GetOrder(test.expected.Id, test.expected.Committee)
			require.NoError(t, err)
			// compare got vs expected
			require.EqualExportedValues(t, test.expected, got)
		})
	}
}

func TestLockOrder(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *lib.SellOrder
		order  *lib.LockOrder
		error  string
	}{
		{
			name:   "lock order not found",
			detail: "the lock order cannot be found",
			order: &lib.LockOrder{

				OrderId:             0,
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "not found",
		},
		{
			name:   "lock order already accepted",
			detail: "the lock order cannot be claimed as its already reserved",
			preset: &lib.SellOrder{
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: &lib.LockOrder{

				OrderId:             0,
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "order already accepted",
		},
		{
			name:   "lock order",
			detail: "successful lock order without error",
			preset: &lib.SellOrder{
				Committee:          lib.CanopyChainId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSendAddress: newTestAddressBytes(t),
			},
			order: &lib.LockOrder{
				OrderId:             0,
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				_, err := sm.CreateOrder(test.preset, lib.CanopyChainId)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.LockOrder(test.order, lib.CanopyChainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the order
			order, e := sm.GetOrder(test.order.OrderId, lib.CanopyChainId)
			require.NoError(t, e)
			// validate the update of the 'buy' fields
			require.Equal(t, test.order.BuyerReceiveAddress, order.BuyerReceiveAddress)
			require.Equal(t, test.order.BuyerChainDeadline, order.BuyerChainDeadline)
		})
	}
}

func TestResetOrder(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *lib.SellOrder
		order  uint64
		error  string
	}{
		{
			name:   "reset order not found",
			detail: "the buy reset cannot be found",
			order:  0,
			error:  "not found",
		},
		{
			name:   "reset order",
			detail: "successful reset order without error",
			preset: &lib.SellOrder{
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				_, err := sm.CreateOrder(test.preset, lib.CanopyChainId)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.ResetOrder(test.order, lib.CanopyChainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the order
			order, e := sm.GetOrder(test.order, lib.CanopyChainId)
			require.NoError(t, e)
			// validate the update of the 'buy' fields
			require.Empty(t, order.BuyerReceiveAddress)
			require.Zero(t, order.BuyerChainDeadline)
		})
	}
}

func TestCloseOrder(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *lib.SellOrder
		order  uint64
		error  string
	}{
		{
			name:   "close order not already accepted",
			detail: "there's no existing buyer for the close order",
			preset: &lib.SellOrder{
				Committee:          lib.CanopyChainId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSendAddress: newTestAddressBytes(t),
			},
			order: 0,
			error: "lock order invalid",
		},
		{
			name:   "close order",
			detail: "successful reset order without error",
			preset: &lib.SellOrder{
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				_, err := sm.CreateOrder(test.preset, lib.CanopyChainId)
				require.NoError(t, err)
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+types.EscrowPoolAddend, test.preset.AmountForSale))
			}
			// execute the function call
			err := sm.CloseOrder(test.order, lib.CanopyChainId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// define convenience variable for order
			order := test.preset
			// validate the deletion of the order
			_, e := sm.GetOrder(test.order, lib.CanopyChainId)
			require.ErrorContains(t, e, "not found")
			// validate the addition of funds to the buyer
			accountBalance, e := sm.GetAccountBalance(crypto.NewAddress(order.BuyerReceiveAddress))
			require.NoError(t, e)
			require.Equal(t, order.AmountForSale, accountBalance)
			// validate the removal of funds from the escrow pool
			balance, e := sm.GetPoolBalance(lib.CanopyChainId + types.EscrowPoolAddend)
			require.NoError(t, e)
			require.Zero(t, balance)
		})
	}
}

func TestDeleteOrder(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		preset   []*lib.SellOrder
		toDelete []*lib.SellOrder
		error    string
	}{
		{
			name:   "order not found",
			detail: "order not found because it wasn't preset",
			preset: []*lib.SellOrder{},
			toDelete: []*lib.SellOrder{
				{
					Id:        0,
					Committee: 0,
				},
			},
			error: "not found",
		},
		{
			name:   "delete sell order",
			detail: "delete sell order",
			preset: []*lib.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "delete sell order for two different committees",
			detail: "delete sell order for another committee",
			preset: []*lib.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			for _, expected := range test.preset {
				// execute the function call
				_, err := sm.CreateOrder(expected, expected.Committee)
				require.NoError(t, err)
				// get the order
				got, err := sm.GetOrder(expected.Id, expected.Committee)
				require.NoError(t, err)
				// compare got vs expected
				require.EqualExportedValues(t, expected, got)
			}
			for _, del := range test.toDelete {
				// delete the order
				err := sm.DeleteOrder(del.Id, del.Committee)
				// validate the expected error
				require.Equal(t, test.error != "", err != nil, err)
				if err != nil {
					require.ErrorContains(t, err, test.error)
					return
				}
				// get the order
				_, err = sm.GetOrder(del.Id, del.Committee)
				require.ErrorContains(t, err, "not found")
			}
		})
	}
}

func TestGetSetOrderBooks(t *testing.T) {
	tests := []struct {
		name                     string
		detail                   string
		expected                 *lib.OrderBooks
		expectedTotalAmount      uint64
		expectedCommitteeAmounts map[uint64]uint64
	}{
		{
			name:   "various",
			detail: "various set to ensure get returns proper order books and supply",
			expected: &lib.OrderBooks{OrderBooks: []*lib.OrderBook{
				{
					ChainId: 0,
					Orders: []*lib.SellOrder{
						{
							Id:                   1,
							Committee:            2,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   0,
							Committee:            1,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
					},
				},
				{
					ChainId: 1,
					Orders: []*lib.SellOrder{
						{
							Id:                   1,
							Committee:            2,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   0,
							Committee:            1,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
					},
				},
			}},
			expectedTotalAmount: 400,
			expectedCommitteeAmounts: map[uint64]uint64{
				0: 200, 1: 200,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			supply := &types.Supply{}
			// set the order books
			require.NoError(t, sm.SetOrderBooks(test.expected, supply))
			// get the order books
			got, err := sm.GetOrderBooks()
			require.NoError(t, err)
			// check got vs expected
			require.EqualExportedValues(t, test.expected, got)
			// check supply set
			require.Equal(t, test.expectedTotalAmount, supply.Total)
			// validate committee amounts
			for id, amount := range test.expectedCommitteeAmounts {
				balance, e := sm.GetPoolBalance(id + types.EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount, balance)
			}
		})
	}
}
