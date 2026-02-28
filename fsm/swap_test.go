package fsm

import (
	"bytes"
	"fmt"
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
			name:   "lock order locked",
			detail: "the lock order cannot be claimed as its already reserved",
			preset: []*lib.SellOrder{
				{
					Id:                  newTestOrderId(t, 0),
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
						OrderId:             newTestOrderId(t, 0),
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
				ResetOrders: [][]byte{newTestOrderId(t, 1)},
			},
			notFound: true,
		},
		{
			name:   "close failed, no buyer",
			detail: "can't close an order that doesn't have a buyer",
			preset: []*lib.SellOrder{
				{
					Id:                 newTestOrderId(t, 0),
					Committee:          lib.CanopyChainId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t),
				},
			},
			orders: &lib.Orders{
				CloseOrders: [][]byte{newTestOrderId(t, 0)},
			},
			noBuyer: true,
		},
		{
			name:   "buy, reset, sell",
			detail: "test buy, reset, and sell without error",
			preset: []*lib.SellOrder{
				{
					Id:                 newTestOrderId(t, 0),
					Committee:          lib.CanopyChainId,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t),
				},
				{
					Id:                  newTestOrderId(t, 1),
					Committee:           lib.CanopyChainId,
					AmountForSale:       100,
					RequestedAmount:     100,
					BuyerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:  newTestAddressBytes(t),
				},
				{
					Id:                  newTestOrderId(t, 2),
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
						OrderId:             newTestOrderId(t, 0),
						BuyerReceiveAddress: newTestAddressBytes(t, 1),
						BuyerChainDeadline:  100,
					},
				},
				ResetOrders: [][]byte{newTestOrderId(t, 1)},
				CloseOrders: [][]byte{newTestOrderId(t, 2)},
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
				err := sm.SetOrder(preset, lib.CanopyChainId)
				require.NoError(t, err)
				// simulate the escrow supply
				escrowPoolBalance += preset.AmountForSale
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+EscrowPoolAddend, preset.AmountForSale))
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
				// validate the deletion of the order
				_, e := sm.GetOrder(closeOrder, lib.CanopyChainId)
				// if order no buyer to close
				if test.noBuyer {
					require.NoError(t, e)
				} else {
					require.ErrorContains(t, e, "not found")
					for _, order := range test.preset {
						if bytes.Equal(order.Id, closeOrder) {
							// validate the addition of funds to the buyer
							accountBalance, e := sm.GetAccountBalance(crypto.NewAddress(order.BuyerReceiveAddress))
							require.NoError(t, e)
							require.Equal(t, order.AmountForSale, accountBalance)
							balanceRemovedFromPool += order.AmountForSale
						}
					}
				}
			}
			// validate the removal of funds from the escrow pool
			balance, e := sm.GetPoolBalance(lib.CanopyChainId + EscrowPoolAddend)
			require.NoError(t, e)
			require.Equal(t, escrowPoolBalance-balanceRemovedFromPool, balance)
		})
	}
}

func TestSetOrder(t *testing.T) {
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
					Id:                   newTestOrderId(t, 0),
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
					Id:                   newTestOrderId(t, 0),
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   newTestOrderId(t, 0),
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
					Id:                   newTestOrderId(t, 0),
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   newTestOrderId(t, 0),
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   newTestOrderId(t, 1), // only used for validation
					Committee:            lib.CanopyChainId + 1,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   newTestOrderId(t, 1), // only used for validation
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
				err := sm.SetOrder(expected, expected.Committee)
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
				err := sm.SetOrder(test.preset, test.preset.Committee)
				require.NoError(t, err)
			}
			// execute the function call
			err := sm.SetOrder(test.expected, lib.CanopyChainId)
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

				OrderId:             newTestOrderId(t, 0),
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "not found",
		},
		{
			name:   "lock order locked",
			detail: "the lock order cannot be claimed as its already reserved",
			preset: &lib.SellOrder{
				Id:                  newTestOrderId(t, 0),
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: &lib.LockOrder{

				OrderId:             newTestOrderId(t, 0),
				BuyerReceiveAddress: newTestAddressBytes(t, 1),
				BuyerChainDeadline:  100,
			},
			error: "order locked",
		},
		{
			name:   "lock order",
			detail: "successful lock order without error",
			preset: &lib.SellOrder{
				Id:                 newTestOrderId(t, 0),
				Committee:          lib.CanopyChainId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSendAddress: newTestAddressBytes(t),
			},
			order: &lib.LockOrder{
				OrderId:             newTestOrderId(t, 0),
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
				err := sm.SetOrder(test.preset, lib.CanopyChainId)
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
		order  []byte
		error  string
	}{
		{
			name:   "reset order not found",
			detail: "the buy reset cannot be found",
			order:  newTestOrderId(t, 0),
			error:  "not found",
		},
		{
			name:   "reset order",
			detail: "successful reset order without error",
			preset: &lib.SellOrder{
				Id:                  newTestOrderId(t, 0),
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: newTestOrderId(t, 0),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				err := sm.SetOrder(test.preset, lib.CanopyChainId)
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
		order  []byte
		error  string
	}{
		{
			name:   "close order not already accepted",
			detail: "there's no existing buyer for the close order",
			preset: &lib.SellOrder{
				Id:                 newTestOrderId(t, 0),
				Committee:          lib.CanopyChainId,
				AmountForSale:      100,
				RequestedAmount:    100,
				SellersSendAddress: newTestAddressBytes(t),
			},
			order: newTestOrderId(t, 0),
			error: "lock order invalid",
		},
		{
			name:   "close order",
			detail: "successful reset order without error",
			preset: &lib.SellOrder{
				Id:                  newTestOrderId(t, 0),
				Committee:           lib.CanopyChainId,
				AmountForSale:       100,
				RequestedAmount:     100,
				BuyerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:  newTestAddressBytes(t),
			},
			order: newTestOrderId(t, 0),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// preset the order
			if test.preset != nil {
				err := sm.SetOrder(test.preset, lib.CanopyChainId)
				require.NoError(t, err)
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+EscrowPoolAddend, test.preset.AmountForSale))
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
			balance, e := sm.GetPoolBalance(lib.CanopyChainId + EscrowPoolAddend)
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
			name:   "delete sell order",
			detail: "delete sell order",
			preset: []*lib.SellOrder{
				{
					Id:                   newTestOrderId(t, 0),
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
					Id:                   newTestOrderId(t, 0),
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t, 1),
					SellersSendAddress:   newTestAddressBytes(t),
				},
				{
					Id:                   newTestOrderId(t, 0),
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
				err := sm.SetOrder(expected, expected.Committee)
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
							Id:                   newTestOrderId(t, 0),
							Committee:            0,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   newTestOrderId(t, 1),
							Committee:            0,
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
							Id:                   newTestOrderId(t, 2),
							Committee:            1,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t, 1),
							SellersSendAddress:   newTestAddressBytes(t),
						},
						{
							Id:                   newTestOrderId(t, 3),
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
			supply := &Supply{}
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
				balance, e := sm.GetPoolBalance(id + EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount, balance)
			}
		})
	}
}

func TestProcessRootChainOrderBookLockBindsBuyerSender(t *testing.T) {
	sm := newTestStateMachine(t)
	orderID := newTestOrderId(t, 99)
	sender := newTestAddressBytes(t, 1)
	spoofedBuyer := newTestAddressBytes(t, 2)
	buyerReceive := newTestAddressBytes(t, 3)

	lockMemo, err := lib.MarshalJSON(&lib.LockOrder{
		OrderId:             orderID,
		ChainId:             sm.Config.ChainId,
		BuyerSendAddress:    spoofedBuyer,
		BuyerReceiveAddress: buyerReceive,
	})
	require.NoError(t, err)

	book := &lib.OrderBook{
		ChainId: sm.Config.ChainId,
		Orders: []*lib.SellOrder{{
			Id:                 orderID,
			Committee:          sm.Config.ChainId,
			AmountForSale:      100,
			RequestedAmount:    50,
			SellersSendAddress: newTestAddressBytes(t, 7),
		}},
	}
	proposal := &lib.BlockResult{
		BlockHeader: &lib.BlockHeader{Height: sm.Height()},
		Transactions: []*lib.TxResult{
			newTestSendTxResult(t, sender, sender, 1, 1_000_000, string(lockMemo), sm.Config.ChainId),
		},
	}

	lockOrders, closedOrders, resetOrders := sm.ProcessRootChainOrderBook(book, proposal)
	require.Len(t, lockOrders, 1)
	require.Empty(t, closedOrders)
	require.Empty(t, resetOrders)
	require.Equal(t, sender, lockOrders[0].BuyerSendAddress)
	require.NotEqual(t, spoofedBuyer, lockOrders[0].BuyerSendAddress)
}

func TestProcessRootChainOrderBookCloseRequiresSenderAndRecipientBinding(t *testing.T) {
	sm := newTestStateMachine(t)
	orderID := newTestOrderId(t, 100)
	lockedBuyer := newTestAddressBytes(t, 1)
	validRecipient := newTestAddressBytes(t, 2)

	closeMemo, err := lib.MarshalJSON(&lib.CloseOrder{
		OrderId:    orderID,
		ChainId:    sm.Config.ChainId,
		CloseOrder: true,
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		sendFrom     []byte
		sendTo       []byte
		expectClosed bool
	}{
		{
			name:         "valid sender and recipient",
			sendFrom:     lockedBuyer,
			sendTo:       validRecipient,
			expectClosed: true,
		},
		{
			name:         "invalid sender",
			sendFrom:     newTestAddressBytes(t, 3),
			sendTo:       validRecipient,
			expectClosed: false,
		},
		{
			name:         "invalid recipient",
			sendFrom:     lockedBuyer,
			sendTo:       newTestAddressBytes(t, 4),
			expectClosed: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			book := &lib.OrderBook{
				ChainId: sm.Config.ChainId,
				Orders: []*lib.SellOrder{{
					Id:                   orderID,
					Committee:            sm.Config.ChainId,
					AmountForSale:        100,
					RequestedAmount:      50,
					SellerReceiveAddress: validRecipient,
					SellersSendAddress:   newTestAddressBytes(t, 6),
					BuyerSendAddress:     lockedBuyer,
					BuyerReceiveAddress:  newTestAddressBytes(t, 5),
					BuyerChainDeadline:   sm.Height() + 100,
				}},
			}
			proposal := &lib.BlockResult{
				BlockHeader: &lib.BlockHeader{Height: sm.Height()},
				Transactions: []*lib.TxResult{
					newTestSendTxResult(t, test.sendFrom, test.sendTo, 50, 1_000_000, string(closeMemo), sm.Config.ChainId),
				},
			}

			_, closedOrders, resetOrders := sm.ProcessRootChainOrderBook(book, proposal)
			require.Empty(t, resetOrders)
			if test.expectClosed {
				require.Len(t, closedOrders, 1)
				require.Equal(t, orderID, closedOrders[0])
			} else {
				require.Empty(t, closedOrders)
			}
		})
	}
}

func newTestSendTxResult(t *testing.T, from, to []byte, amount, fee uint64, memo string, chainID uint64) *lib.TxResult {
	anyMsg, err := lib.NewAny(&MessageSend{
		FromAddress: from,
		ToAddress:   to,
		Amount:      amount,
	})
	require.NoError(t, err)
	return &lib.TxResult{
		Sender:      from,
		Recipient:   to,
		MessageType: MessageSendName,
		Transaction: &lib.Transaction{
			MessageType: MessageSendName,
			Msg:         anyMsg,
			Fee:         fee,
			Memo:        memo,
			ChainId:     chainID,
		},
	}
}

func newTestOrderId(_ *testing.T, variant int) []byte {
	return []byte(fmt.Sprintf("%d", variant))
}
