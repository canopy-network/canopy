package lib

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddOrder(t *testing.T) {
	tests := []struct {
		name           string
		initialOrders  []*SellOrder
		newOrder       *SellOrder
		expectedID     uint64
		expectedOrders []*SellOrder
	}{
		{
			name:          "Add to empty OrderBook",
			initialOrders: []*SellOrder{},
			newOrder:      &SellOrder{},
			expectedID:    0,
			expectedOrders: []*SellOrder{
				{Id: 0},
			},
		},
		{
			name: "Add to OrderBook with empty slot",
			initialOrders: []*SellOrder{
				{Id: 0},
				nil,
				{Id: 2},
			},
			newOrder:   &SellOrder{},
			expectedID: 1,
			expectedOrders: []*SellOrder{
				{Id: 0},
				{Id: 1},
				{Id: 2},
			},
		},
		{
			name: "Add to OrderBook with no empty slots",
			initialOrders: []*SellOrder{
				{Id: 0},
				{Id: 1},
			},
			newOrder:   &SellOrder{},
			expectedID: 2,
			expectedOrders: []*SellOrder{
				{Id: 0},
				{Id: 1},
				{Id: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// initialize the OrderBook
			orderBook := &OrderBook{Orders: test.initialOrders}
			// add the order
			require.Equal(t, test.expectedID, orderBook.AddOrder(test.newOrder))
			require.Equal(t, test.expectedOrders, orderBook.Orders)
		})
	}
}

func TestLockOrder(t *testing.T) {
	tests := []struct {
		name                     string
		initialOrders            []*SellOrder
		orderId                  int
		buyersReceiveAddress     []byte
		buyersSendAddress        []byte
		buyerChainDeadlineHeight uint64
		error                    string
		expectedOrder            *SellOrder
	}{
		{
			name: "Order already claimed",
			initialOrders: []*SellOrder{
				{
					Id:                  0,
					BuyerReceiveAddress: []byte("existing_receive"),
				},
			},
			orderId:                  0,
			buyersReceiveAddress:     []byte("buyer_receive"),
			buyersSendAddress:        []byte("buyer_send"),
			buyerChainDeadlineHeight: 200,
			error:                    "order already accepted",
			expectedOrder: &SellOrder{
				Id:                  0,
				BuyerReceiveAddress: []byte("existing_receive"),
			},
		},
		{
			name:                     "Order not found",
			initialOrders:            []*SellOrder{{Id: 0}},
			orderId:                  99,
			buyersReceiveAddress:     []byte("buyer_receive"),
			buyersSendAddress:        []byte("buyer_send"),
			buyerChainDeadlineHeight: 300,
			error:                    "not found",
			expectedOrder:            nil,
		},
		{
			name:                     "Successful order claim",
			initialOrders:            []*SellOrder{{Id: 0, BuyerReceiveAddress: nil}},
			orderId:                  0,
			buyersReceiveAddress:     []byte("buyer_receive"),
			buyersSendAddress:        []byte("buyer_send"),
			buyerChainDeadlineHeight: 100,
			expectedOrder: &SellOrder{
				Id:                  0,
				BuyerReceiveAddress: []byte("buyer_receive"),
				BuyerSendAddress:    []byte("buyer_send"),
				BuyerChainDeadline:  100,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// init the OrderBook
			orderBook := &OrderBook{Orders: test.initialOrders}
			// execute the function call
			err := orderBook.LockOrder(
				test.orderId,
				test.buyersReceiveAddress,
				test.buyersSendAddress,
				test.buyerChainDeadlineHeight,
			)
			// check the error
			if test.error != "" {
				require.ErrorContains(t, err, test.error)
			}
			// check the state of the order
			if test.expectedOrder != nil {
				require.Equal(t, test.expectedOrder, orderBook.Orders[test.orderId])
			}
		})
	}
}

func TestResetOrder(t *testing.T) {
	tests := []struct {
		name          string
		initialOrders []*SellOrder
		orderId       int
		error         string
		expectedOrder *SellOrder
	}{
		{
			name: "Order not found",
			initialOrders: []*SellOrder{
				{
					Id:                  1,
					BuyerReceiveAddress: []byte("buyer_receive"),
					BuyerChainDeadline:  200,
				},
			},
			orderId:       99, // Non-existent order
			error:         "not found",
			expectedOrder: nil,
		},
		{
			name: "Unclaimed order reset",
			initialOrders: []*SellOrder{
				{
					Id:                  0,
					BuyerReceiveAddress: nil,
					BuyerChainDeadline:  0,
				},
			},
			orderId: 0,
			expectedOrder: &SellOrder{
				Id:                  0,
				BuyerReceiveAddress: nil,
				BuyerChainDeadline:  0,
			},
		},
		{
			name: "Successful order reset",
			initialOrders: []*SellOrder{
				{
					Id:                  0,
					BuyerReceiveAddress: []byte("buyer_receive"),
					BuyerChainDeadline:  100,
				},
			},
			orderId: 0,
			expectedOrder: &SellOrder{
				Id:                  0,
				BuyerReceiveAddress: nil,
				BuyerChainDeadline:  0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// init the OrderBook
			orderBook := &OrderBook{
				Orders: test.initialOrders,
			}
			// execute the function call
			err := orderBook.ResetOrder(test.orderId)
			if test.error != "" {
				require.ErrorContains(t, err, test.error)
			}
			// verify the state of the order (if it exists)
			if test.expectedOrder != nil {
				require.Equal(t, test.expectedOrder, orderBook.Orders[test.orderId])
			}
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	tests := []struct {
		name           string
		initialOrders  []*SellOrder
		orderId        int
		newOrder       *SellOrder
		error          ErrorI
		expectedOrders []*SellOrder
	}{
		{
			name: "Update an existing order",
			initialOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				{Id: 1, AmountForSale: 200},
			},
			orderId: 1,
			newOrder: &SellOrder{
				Id:            1,
				AmountForSale: 300,
			},
			expectedOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				{Id: 1, AmountForSale: 300},
			},
		},
		{
			name: "Delete an order from the middle",
			initialOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				{Id: 1, AmountForSale: 200},
				{Id: 2, AmountForSale: 300},
			},
			orderId:  1,
			newOrder: nil,
			expectedOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				nil,
				{Id: 2, AmountForSale: 300},
			},
		},
		{
			name: "Delete the last order and shrink slice",
			initialOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				{Id: 1, AmountForSale: 200},
			},
			orderId:  1,
			newOrder: nil,
			expectedOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
			},
		},
		{
			name: "Order not found",
			initialOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
			},
			orderId:  2,
			newOrder: &SellOrder{Id: 2, AmountForSale: 400},
			error:    ErrOrderNotFound(2),
			expectedOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
			},
		},
		{
			name: "Shrink slice with trailing nils",
			initialOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
				nil,
				nil,
			},
			orderId:  2,
			newOrder: nil,
			expectedOrders: []*SellOrder{
				{Id: 0, AmountForSale: 100},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// init the OrderBook
			orderBook := &OrderBook{Orders: test.initialOrders}
			// execute the function call
			require.Equal(t, test.error, orderBook.UpdateOrder(test.orderId, test.newOrder))
			// verify the state of the order book
			require.Equal(t, test.expectedOrders, orderBook.Orders)
		})
	}
}
