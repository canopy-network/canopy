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
		name   string
		detail string
		preset []*types.SellOrder
		orders *lib.Orders
		error  string
	}{
		{
			name:   "buy order already accepted",
			detail: "the buy order cannot be claimed as its already reserved",
			preset: []*types.SellOrder{
				{
					Committee:           lib.CanopyCommitteeId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t),
					SellersSellAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				BuyOrders: []*lib.BuyOrder{
					{
						OrderId:             0,
						BuyerReceiveAddress: newTestAddressBytes(t, 1),
						BuyerChainDeadline:  100,
					},
				},
			},
			error: "order already accepted",
		},
		{
			name:   "reset failed, order not found",
			detail: "can't reset an order that doesn't exist",
			preset: []*types.SellOrder{
				{
					Committee:           lib.CanopyCommitteeId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t),
					SellersSellAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				ResetOrders: []uint64{1},
			},
			error: "not found",
		},
		{
			name:   "close failed, no buyer",
			detail: "can't close an order that doesn't have a buyer",
			preset: []*types.SellOrder{
				{
					Committee:          lib.CanopyCommitteeId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSellAddress: newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				CloseOrders: []uint64{0},
			},
			error: "buy order invalid",
		},
		{
			name:   "buy, reset, sell",
			detail: "test buy, reset, and sell without error",
			preset: []*types.SellOrder{
				{
					Committee:          lib.CanopyCommitteeId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSellAddress: newTestAddressBytes(t),
				},
				{
					Committee:           lib.CanopyCommitteeId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:  newTestAddressBytes(t),
				},
				{
					Committee:           lib.CanopyCommitteeId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:  newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				BuyOrders: []*lib.BuyOrder{
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
				_, err := sm.CreateOrder(preset, lib.CanopyCommitteeId)
				require.NoError(t, err)
				// simulate the escrow supply
				escrowPoolBalance += preset.AmountForSale
				require.NoError(t, sm.PoolAdd(lib.CanopyCommitteeId+types.EscrowPoolAddend, preset.AmountForSale))
			}
			// execute the function call
			err := sm.HandleCommitteeSwaps(test.orders, lib.CanopyCommitteeId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the buy orders
			for _, buyOrder := range test.orders.BuyOrders {
				// get the order
				order, e := sm.GetOrder(buyOrder.OrderId, lib.CanopyCommitteeId)
				require.NoError(t, e)
				// validate the update of the 'buy' fields
				require.Equal(t, buyOrder.BuyerReceiveAddress, order.BuyerReceiveAddress)
				require.Equal(t, buyOrder.BuyerChainDeadline, order.BuyerChainDeadline)
			}
			// validate the reset orders
			for _, resetOrderId := range test.orders.ResetOrders {
				// get the order
				order, e := sm.GetOrder(resetOrderId, lib.CanopyCommitteeId)
				require.NoError(t, e)
				// validate the update of the 'buy' fields
				require.Empty(t, order.BuyerReceiveAddress)
				require.Zero(t, order.BuyerChainDeadline)
			}
			var balanceRemovedFromPool uint64
			// validate the close orders
			for _, closeOrder := range test.orders.CloseOrders {
				// define convenience variable for order
				order := test.preset[closeOrder]
				// validate the deletion of the order
				_, e := sm.GetOrder(closeOrder, lib.CanopyCommitteeId)
				require.ErrorContains(t, e, "not found")
				// validate the addition of funds to the buyer
				accountBalance, e := sm.GetAccountBalance(crypto.NewAddress(order.BuyerReceiveAddress))
				require.NoError(t, e)
				require.Equal(t, order.AmountForSale, accountBalance)
				balanceRemovedFromPool += order.AmountForSale
			}
			// validate the removal of funds from the escrow pool
			balance, e := sm.GetPoolBalance(lib.CanopyCommitteeId + types.EscrowPoolAddend)
			require.NoError(t, e)
			require.Equal(t, escrowPoolBalance-balanceRemovedFromPool, balance)
		})
	}
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		expected []*types.SellOrder
	}{
		{
			name:   "create sell order",
			detail: "create sell order",
			expected: []*types.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "create sell order for two different committees",
			detail: "create sell order for another committee",
			expected: []*types.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "id creation order",
			detail: "test the id creation order",
			expected: []*types.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   1, // only used for validation
					Committee:            lib.CanopyCommitteeId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   1, // only used for validation
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
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
		preset   *types.SellOrder
		expected *types.SellOrder
		error    string
	}{
		{
			name:   "order not found",
			detail: "order not preset so no order id is found",
			expected: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        101,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			error: "not found",
		},
		{
			name:   "update amount",
			detail: "update the amount for sale without error",
			preset: &types.SellOrder{
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        100,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			expected: &types.SellOrder{
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        101,
				RequestedAmount:      100,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
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
			err := sm.EditOrder(test.expected, lib.CanopyCommitteeId)
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

func TestBuyOrder(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *types.SellOrder
		order  *lib.BuyOrder
		error  string
	}{
		{
			name:   "buy order not found",
			detail: "the buy order cannot be found",
			order: &lib.BuyOrder{

				OrderId:             0,
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "not found",
		},
		{
			name:   "buy order already accepted",
			detail: "the buy order cannot be claimed as its already reserved",
			preset: &types.SellOrder{
				Committee:           lib.CanopyCommitteeId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:  newTestAddressBytes(t),
			},
			order: &lib.BuyOrder{

				OrderId:             0,
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "order already accepted",
		},
		{
			name:   "buy order",
			detail: "successful buy order without error",
			preset: &types.SellOrder{
				Committee:          lib.CanopyCommitteeId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSellAddress: newTestAddressBytes(t),
			},
			order: &lib.BuyOrder{
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
				_, err := sm.CreateOrder(test.preset, lib.CanopyCommitteeId)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.BuyOrder(test.order, lib.CanopyCommitteeId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the order
			order, e := sm.GetOrder(test.order.OrderId, lib.CanopyCommitteeId)
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
		preset *types.SellOrder
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
			preset: &types.SellOrder{
				Committee:           lib.CanopyCommitteeId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:  newTestAddressBytes(t),
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
				_, err := sm.CreateOrder(test.preset, lib.CanopyCommitteeId)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.ResetOrder(test.order, lib.CanopyCommitteeId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the order
			order, e := sm.GetOrder(test.order, lib.CanopyCommitteeId)
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
		preset *types.SellOrder
		order  uint64
		error  string
	}{
		{
			name:   "close order not already accepted",
			detail: "there's no existing buyer for the close order",
			preset: &types.SellOrder{
				Committee:          lib.CanopyCommitteeId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSellAddress: newTestAddressBytes(t),
			},
			order: 0,
			error: "buy order invalid",
		},
		{
			name:   "close order",
			detail: "successful reset order without error",
			preset: &types.SellOrder{
				Committee:           lib.CanopyCommitteeId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:  newTestAddressBytes(t),
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
				_, err := sm.CreateOrder(test.preset, lib.CanopyCommitteeId)
				require.NoError(t, err)
				require.NoError(t, sm.PoolAdd(lib.CanopyCommitteeId+types.EscrowPoolAddend, test.preset.AmountForSale))
			}
			// execute the function call
			err := sm.CloseOrder(test.order, lib.CanopyCommitteeId)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// define convenience variable for order
			order := test.preset
			// validate the deletion of the order
			_, e := sm.GetOrder(test.order, lib.CanopyCommitteeId)
			require.ErrorContains(t, e, "not found")
			// validate the addition of funds to the buyer
			accountBalance, e := sm.GetAccountBalance(crypto.NewAddress(order.BuyerReceiveAddress))
			require.NoError(t, e)
			require.Equal(t, order.AmountForSale, accountBalance)
			// validate the removal of funds from the escrow pool
			balance, e := sm.GetPoolBalance(lib.CanopyCommitteeId + types.EscrowPoolAddend)
			require.NoError(t, e)
			require.Zero(t, balance)
		})
	}
}

func TestDeleteOrder(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		preset   []*types.SellOrder
		toDelete []*types.SellOrder
		error    string
	}{
		{
			name:   "order not found",
			detail: "order not found because it wasn't preset",
			preset: []*types.SellOrder{},
			toDelete: []*types.SellOrder{
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
			preset: []*types.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
			},
		},
		{
			name:   "delete sell order for two different committees",
			detail: "delete sell order for another committee",
			preset: []*types.SellOrder{
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   0,
					Committee:            lib.CanopyCommitteeId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSellAddress:   newTestAddressBytes(t),
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
		expected                 *types.OrderBooks
		expectedTotalAmount      uint64
		expectedCommitteeAmounts map[uint64]uint64
	}{
		{
			name:   "various",
			detail: "various set to ensure get returns proper order books and supply",
			expected: &types.OrderBooks{OrderBooks: []*types.OrderBook{
				{
					CommitteeId: 0,
					Orders: []*types.SellOrder{
						{
							Id:                   1,
							Committee:            2,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSellAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   0,
							Committee:            1,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSellAddress:   newTestAddressBytes(t),
						},
					},
				},
				{
					CommitteeId: 1,
					Orders: []*types.SellOrder{
						{
							Id:                   1,
							Committee:            2,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSellAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   0,
							Committee:            1,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSellAddress:   newTestAddressBytes(t),
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
