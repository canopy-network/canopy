package fsm

import (
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
)

/* This file contains state machine changes related to 'token swapping' */

// HandleCommitteeSwaps() when the committee submits a 'certificate results transaction', it informs the chain of various actions over sell orders
// - 'buy' is an actor 'claiming / reserving' the sell order
// - 'reset' is a 'claimed' order whose 'buyer' did not send the tokens to the seller before the deadline, thus the order is re-opened for sale
// - 'close' is a 'claimed' order whose 'buyer' sent the tokens to the seller before the deadline, thus the order is 'closed' and the tokens are moved from escrow to the buyer
func (s *StateMachine) HandleCommitteeSwaps(orders *lib.Orders, chainId uint64) {
	if orders != nil {
		// lock orders are a result of the committee witnessing a 'reserve transaction' for the order on the 'buyer chain'
		// think of 'lock orders' like reserving the 'sell order'
		for _, lockOrder := range orders.LockOrders {
			if err := s.LockOrder(lockOrder, chainId); err != nil {
				s.log.Warnf("LockOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
		// reset orders are a result of the committee witnessing 'no-action' from the buyer of the sell order aka NOT sending the
		// corresponding assets before the 'deadline height' of the 'buyer chain'. The buyer address and deadline height are reset and the
		// sell order is listed as 'available' to the rest of the market
		for _, resetOrderId := range orders.ResetOrders {
			if err := s.ResetOrder(resetOrderId, chainId); err != nil {
				s.log.Warnf("ResetOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
		// close orders are a result of the committee witnessing the buyer sending the
		// buy assets before the 'deadline height' of the 'buyer chain'
		for _, closeOrderId := range orders.CloseOrders {
			if err := s.CloseOrder(closeOrderId, chainId); err != nil {
				s.log.Warnf("CloseOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
	}
	// exit
	return
}

// ParseLockOrder() parses a transaction for an embedded lock order messages in the memo field
func (s *StateMachine) ParseLockOrder(tx *lib.Transaction, deadlineBlocks uint64) (bo *lib.LockOrder, ok bool) {
	// create a new reference to a 'lock order' object in order to ensure a non-nil result
	bo = new(lib.LockOrder)
	// attempt to unmarshal the transaction memo into a 'lock order'
	if err := lib.UnmarshalJSON([]byte(tx.Memo), bo); err == nil {
		// sanity check some critical fields of the 'lock order' to ensure the unmarshal was successful
		if len(bo.BuyerSendAddress) != 0 && len(bo.BuyerReceiveAddress) != 0 {
			ok = true
		}
		// set the 'BuyerChainDeadline' in the 'lock order'
		bo.BuyerChainDeadline = s.Height() + deadlineBlocks
	}
	// exit
	return
}

// ProcessRootChainOrderBook() processes the order book from the root-chain and cross-references blocks on this chain to determine
// actions that warrant committee level changes to the root-chain order book like: ResetOrder and CloseOrder
func (s *StateMachine) ProcessRootChainOrderBook(book *lib.OrderBook, b *lib.BlockResult) (closeOrders, resetOrders []uint64) {
	// create a variable to track the 'same-block' transfers
	// map structure is [from+to] -> amount sent
	transferred := make(map[string]uint64)
	// get all the 'Send' transactions from the block
	for _, tx := range b.Transactions {
		// ignore non-send
		if tx.MessageType != types.MessageSendName {
			continue
		}
		// extract the message from the transaction object
		msg, e := lib.FromAny(tx.Transaction.Msg)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		// cast the message to send
		send, ok := msg.(*types.MessageSend)
		if !ok {
			s.log.Error("Non-send message with a send message name (should not happen)")
			continue
		}
		// update the transferred map using the [from + to] as the key and add send.Amount to the value
		transferred[lib.BytesToString(append(tx.Sender, tx.Recipient...))] += send.Amount
	}
	// for each order in the book
	for _, order := range book.Orders {
		// skip any order that is not currently 'locked'
		if len(order.BuyerReceiveAddress) == 0 {
			continue
		}
		// see if the 'locked' order is expired
		if s.height > order.BuyerChainDeadline {
			// add to reset orders
			resetOrders = append(resetOrders, order.Id)
			// go to the next order
			continue
		}
		// extract a key that should correspond to the transferred map
		key := lib.BytesToString(append(order.BuyerSendAddress, order.SellerReceiveAddress...))
		// check if the order was closed this block
		if transferred[key] == order.RequestedAmount {
			// add to the closed order list
			closeOrders = append(closeOrders, order.Id)
			// go to the next order
			continue
		}
		// scan the N-10 through N-15 blocks to ensure no orders are lost
		start, end, n := uint64(0), uint64(0), b.BlockHeader.Height
		// bounds check
		if n >= 15 {
			end = n - 15
		}
		// bounds check
		if n >= 10 {
			start = n - 10
		}
		// don't begin doing this check until after block 15
		if start == 0 || end == 0 {
			continue
		}
		// from blocks N-10 to N-15
		for i := start; i > end; i-- {
			// load the certificate (hopefully from cache)
			qc, err := s.LoadCertificate(i)
			if err != nil {
				s.log.Error(err.Error())
				continue
			}
			// sanity check
			if qc.Results == nil {
				s.log.Error(lib.ErrNilCertResults().Error())
				continue
			}
			// check if the 'close order' command was issued previously
			if qc.Results.Orders != nil && slices.Contains(qc.Results.Orders.CloseOrders, order.Id) {
				// if so, add it to the close orders
				closeOrders = append(closeOrders, order.Id)
				// exit the loop
				break
			}
		}
	}
	// exit
	return
}

// ParseLockOrders() parses the proposal block for memo commands to execute specialized 'lock order' functionality
func (s *StateMachine) ParseLockOrders(b *lib.BlockResult) (lockOrders []*lib.LockOrder) {
	// get the governance parameters from state
	params, err := s.GetParams()
	if err != nil {
		s.log.Error(err.Error())
		return
	}
	// calculate the minimum lock order fee
	minFee := params.Fee.SendFee * params.Validator.LockOrderFeeMultiplier
	// ensure duplicate actions are skipped
	deDupeLockOrders := lib.NewDeDuplicator[uint64]()
	// for each transaction in the block
	for _, tx := range b.Transactions {
		// skip over any that doesn't have the minimum fee or isn't the correct type
		if tx.MessageType != types.MessageSendName && tx.Transaction.Fee < minFee {
			continue
		}
		// parse the transaction for embedded 'lock orders'
		if lockOrder, ok := s.ParseLockOrder(tx.Transaction, params.Validator.BuyDeadlineBlocks); ok {
			// if not found (non-duplicate)
			if found := deDupeLockOrders.Found(lockOrder.OrderId); !found {
				// add to the 'lock orders' list
				lockOrders = append(lockOrders, lockOrder)
			}
		}
	}
	// exit
	return
}

// CreateOrder() adds an order to the order book for a committee in the state db
func (s *StateMachine) CreateOrder(order *lib.SellOrder, chainId uint64) (orderId uint64, err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// add the new sell order to the order book
	orderId = orderBook.AddOrder(order)
	// set the order book back in state
	err = s.SetOrderBook(orderBook)
	// exit
	return
}

// EditOrder() updates an existing order in the order book for a committee in the state db
func (s *StateMachine) EditOrder(order *lib.SellOrder, chainId uint64) (err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// update the order at a specific id
	if err = orderBook.UpdateOrder(int(order.Id), order); err != nil {
		return
	}
	// set the order book back in state
	err = s.SetOrderBook(orderBook)
	// exit
	return
}

// LockOrder() adds a recipient and a deadline height to an existing order and saves it to the state
func (s *StateMachine) LockOrder(lockOrder *lib.LockOrder, chainId uint64) (err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// 'reserve' the order in the book
	if err = orderBook.LockOrder(int(lockOrder.OrderId), lockOrder.BuyerReceiveAddress, lockOrder.BuyerSendAddress, lockOrder.BuyerChainDeadline); err != nil {
		return
	}
	// set the order book back in state
	err = s.SetOrderBook(orderBook)
	// exit
	return
}

// ResetOrder() removes the recipient and deadline height from an existing order and saves it to the state
func (s *StateMachine) ResetOrder(orderId, chainId uint64) (err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// reset the order in the book
	if err = orderBook.ResetOrder(int(orderId)); err != nil {
		return
	}
	// set the order book back in state
	err = s.SetOrderBook(orderBook)
	// exit
	return
}

// CloseOrder() sends the tokens from escrow to the 'buyer address' and deletes the order
func (s *StateMachine) CloseOrder(orderId, chainId uint64) (err lib.ErrorI) {
	// the order is 'closed' and the tokens are moved from escrow to the buyer
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		return
	}
	// ensure the order already was 'claimed / reserved'
	if order.BuyerReceiveAddress == nil {
		return types.ErrInvalidLockOrder()
	}
	// remove the funds from the escrow pool
	if err = s.PoolSub(chainId+types.EscrowPoolAddend, order.AmountForSale); err != nil {
		return
	}
	// send the funds to the recipient address
	if err = s.AccountAdd(crypto.NewAddress(order.BuyerReceiveAddress), order.AmountForSale); err != nil {
		return
	}
	// delete the order
	return s.DeleteOrder(orderId, chainId)
}

// DeleteOrder() deletes an existing order in the order book for a committee in the state db
func (s *StateMachine) DeleteOrder(orderId, chainId uint64) (err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// update the order with a 'nil' value (delete)
	if err = orderBook.UpdateOrder(int(orderId), nil); err != nil {
		return
	}
	// set the order book back in state
	err = s.SetOrderBook(orderBook)
	// exit
	return
}

// GetOrder() sets the order book for a committee in the state db
func (s *StateMachine) GetOrder(orderId uint64, chainId uint64) (order *lib.SellOrder, err lib.ErrorI) {
	// get the order book for a specific chainId from state
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	// get the specific order from the book based on its ID
	return orderBook.GetOrder(int(orderId))
}

// SetOrderBook() sets the order book for a committee in the state db
func (s *StateMachine) SetOrderBook(b *lib.OrderBook) lib.ErrorI {
	// convert the order book into bytes
	orderBookBz, err := lib.Marshal(b)
	if err != nil {
		return err
	}
	// set the order book in the store
	return s.store.Set(types.KeyForOrderBook(b.ChainId), orderBookBz)
}

// SetOrderBooks() sets a series of OrderBooks in the state db
func (s *StateMachine) SetOrderBooks(list *lib.OrderBooks, supply *types.Supply) lib.ErrorI {
	// ensure the order books object reference is not nil
	if list == nil {
		return nil
	}
	// for each book in the order books list
	for _, book := range list.OrderBooks {
		// convert the order book into bytes
		orderBookBz, err := lib.Marshal(book)
		if err != nil {
			return err
		}
		// get the state 'key' for the order book
		key := types.KeyForOrderBook(book.ChainId)
		// write the order book for the committee to state under the 'key'
		if err = s.store.Set(key, orderBookBz); err != nil {
			return err
		}
		// properly mint to the supply pool
		for _, order := range book.Orders {
			// update the 'supply' tracker
			supply.Total += order.AmountForSale
			// calculate the escrow pool id for a specific chainId
			escrowPoolId := book.ChainId + uint64(types.EscrowPoolAddend)
			// add to the 'escrow' pool for the specific id
			if err = s.PoolAdd(escrowPoolId, order.AmountForSale); err != nil {
				return err
			}
		}
	}
	// exit
	return nil
}

// GetOrderBook() retrieves the order book for a committee from the state db
func (s *StateMachine) GetOrderBook(chainId uint64) (b *lib.OrderBook, err lib.ErrorI) {
	// initialize the order book object reference to ensure non nil results
	b = new(lib.OrderBook)
	// update the orders and chainId of the newly created object ref
	b.Orders, b.ChainId = make([]*lib.SellOrder, 0), chainId
	// get order book bytes from the state using the order book key for a specific chainId
	bz, err := s.Get(types.KeyForOrderBook(chainId))
	if err != nil {
		return
	}
	// convert order book bytes into the order book variable
	err = lib.Unmarshal(bz, b)
	// exit
	return
}

// GetOrderBooks() retrieves the lists for all chainIds of open 'sell orders' from the state
func (s *StateMachine) GetOrderBooks() (b *lib.OrderBooks, err lib.ErrorI) {
	// get the order books from the state
	b = new(lib.OrderBooks)
	// create an iterator over the OrderBookPrefix
	it, err := s.Iterator(types.OrderBookPrefix())
	if err != nil {
		return
	}
	// memory cleanup the iterator
	defer it.Close()
	// for each item under the OrderBookPrefix
	for ; it.Valid(); it.Next() {
		// extract the chainId from the key
		id, e := types.IdFromKey(it.Key())
		if e != nil {
			return nil, e
		}
		// get the specific order book for the chainId
		book, e := s.GetOrderBook(id)
		if e != nil {
			return nil, e
		}
		// add the book to the list
		b.OrderBooks = append(b.OrderBooks, book)
	}
	// exit
	return
}
