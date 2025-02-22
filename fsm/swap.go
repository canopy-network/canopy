package fsm

import (
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"slices"
)

// HandleCommitteeSwaps() when the committee submits a 'certificate results transaction', it informs the chain of various actions over sell orders
// - 'lock' is an actor 'claiming / reserving' the sell order
// - 'reset' is a 'claimed' order whose 'locker' did not send the tokens to the seller before the deadline, thus the order is re-opened for sale
// - 'close' is a 'claimed' order whose 'locker' sent the tokens to the seller before the deadline, thus the order is 'closed' and the tokens are moved from escrow to the locker
func (s *StateMachine) HandleCommitteeSwaps(orders *lib.Orders, chainId uint64) {
	if orders != nil {
		// lock orders are a result of the committee witnessing a 'reserve transaction' for the order on the 'locker chain'
		// think of 'lock orders' like reserving the 'sell order'
		for _, lockOrder := range orders.LockOrders {
			if err := s.LockOrder(lockOrder, chainId); err != nil {
				s.log.Warnf("LockOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
		// reset orders are a result of the committee witnessing 'no-action' from the locker of the sell order aka NOT sending the
		// corresponding assets before the 'deadline height' of the 'locker chain'. The locker address and deadline height are reset and the
		// sell order is listed as 'available' to the rest of the market
		for _, resetOrderId := range orders.ResetOrders {
			if err := s.ResetOrder(resetOrderId, chainId); err != nil {
				s.log.Warnf("ResetOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
		// close orders are a result of the committee witnessing the locker sending the
		// lock assets before the 'deadline height' of the 'locker chain'
		for _, closeOrderId := range orders.CloseOrders {
			if err := s.CloseOrder(closeOrderId, chainId); err != nil {
				s.log.Warnf("CloseOrder failed (can happen due to asynchronicity): %s", err.Error())
			}
		}
	}
	return
}

// ParseLockOrder() parses a transaction for an embedded lock order messages in the memo field
func (s *StateMachine) ParseLockOrder(tx *lib.Transaction, deadlineBlocks uint64) (bo *lib.LockOrder, ok bool) {
	bo = new(lib.LockOrder)
	if err := lib.UnmarshalJSON([]byte(tx.Memo), bo); err == nil {
		if len(bo.LockerSendAddress) != 0 && len(bo.LockerSendAddress) != 0 {
			ok = true
		}
		// set the locker deadline
		bo.LockerChainDeadline = s.Height() + deadlineBlocks
	}
	return
}

// ProcessRootChainOrderBook() processes the order book from the root-Chain and cross-references
func (s *StateMachine) ProcessRootChainOrderBook(book *lib.OrderBook, b *lib.BlockResult) (closeOrders, resetOrders []uint64) {
	transferred := make(map[string]uint64) // [from+to] -> amount sent
	// get all the 'Send' transactions from the block
	for _, tx := range b.Transactions {
		// ignore non-send
		if tx.MessageType != types.MessageSendName {
			continue
		}
		// parse send
		msg, e := lib.FromAny(tx.Transaction.Msg)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		send, ok := msg.(*types.MessageSend)
		if !ok {
			s.log.Error("Non-send message with a send message name")
			continue
		}
		// add to total transferred
		transferred[lib.BytesToString(append(tx.Sender, tx.Recipient...))] += send.Amount
	}
	// for each order
	for _, order := range book.Orders {
		// skip locker-less orders
		if len(order.LockerReceiveAddress) == 0 {
			continue
		}
		// extract a key for the totalTransferred map
		key := lib.BytesToString(append(order.LockerSendAddress, order.SellerReceiveAddress...))
		// see if expired
		if s.height > order.LockerChainDeadline {
			// add to reset orders
			resetOrders = append(resetOrders, order.Id)
		} else {
			// check if the order was closed this block
			if transferred[key] == order.RequestedAmount {
				closeOrders = append(closeOrders, order.Id)
				continue // go to the next order
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
			for i := start; i > end; i-- {
				// load the certificate (hopefully from cache)
				qc, err := s.LoadCertificate(b.BlockHeader.Height)
				if err != nil {
					s.log.Error(err.Error())
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
	}
	return
}

// ParseLockOrders() parses the proposal block for memo commands to execute specialized 'lock order' functionality
func (s *StateMachine) ParseLockOrders(b *lib.BlockResult) (lockOrders []*lib.LockOrder) {
	params, err := s.GetParams()
	if err != nil {
		s.log.Error(err.Error())
		return
	}
	// calculate the minimum lock order fee
	minFee := params.Fee.SendFee * params.Validator.LockOrderFeeMultiplier
	// for each transaction in the block
	for _, tx := range b.Transactions {
		deDupeLockOrders := make(map[uint64]struct{})
		// skip over any that doesn't have the minimum fee or isn't the correct type
		if tx.MessageType != types.MessageSendName && tx.Transaction.Fee < minFee {
			continue
		}
		// parse the transaction for embedded 'lock orders'
		if lockOrder, ok := s.ParseLockOrder(tx.Transaction, params.Validator.LockDeadlineBlocks); ok {
			if _, found := deDupeLockOrders[lockOrder.OrderId]; !found {
				lockOrders = append(lockOrders, lockOrder)
				deDupeLockOrders[lockOrder.OrderId] = struct{}{}
			}
		}
	}
	return
}

// CreateOrder() adds an order to the order book for a committee in the state db
func (s *StateMachine) CreateOrder(order *lib.SellOrder, chainId uint64) (orderId uint64, err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	orderId = orderBook.AddOrder(order)
	err = s.SetOrderBook(orderBook)
	return
}

// EditOrder() updates an existing order in the order book for a committee in the state db
func (s *StateMachine) EditOrder(order *lib.SellOrder, chainId uint64) (err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	if err = orderBook.UpdateOrder(int(order.Id), order); err != nil {
		return
	}
	err = s.SetOrderBook(orderBook)
	return
}

// LockOrder() adds a recipient and a deadline height to an existing order and saves it to the state
func (s *StateMachine) LockOrder(lockOrder *lib.LockOrder, chainId uint64) (err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	if err = orderBook.LockOrder(int(lockOrder.OrderId), lockOrder.LockerReceiveAddress, lockOrder.LockerSendAddress, lockOrder.LockerChainDeadline); err != nil {
		return
	}
	err = s.SetOrderBook(orderBook)
	return
}

// ResetOrder() removes the recipient and deadline height from an existing order and saves it to the state
func (s *StateMachine) ResetOrder(orderId, chainId uint64) (err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	if err = orderBook.ResetOrder(int(orderId)); err != nil {
		return
	}
	err = s.SetOrderBook(orderBook)
	return
}

// CloseOrder() sends the tokens from escrow to the 'locker address' and deletes the order
func (s *StateMachine) CloseOrder(orderId, chainId uint64) (err lib.ErrorI) {
	// the order is 'closed' and the tokens are moved from escrow to the locker
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		// due to the redundancy 'Look Back' design of the swaps submitting a close order that has no available order is allowed
		// this is considered safe due to the +2/3rd committee signature requirement
		s.log.Warn(err.Error())
		return nil
	}
	// ensure the order already was 'claimed / reserved'
	if order.LockerReceiveAddress == nil {
		return types.ErrInvalidLockOrder()
	}
	// remove the funds from the escrow pool
	if err = s.PoolSub(chainId+types.EscrowPoolAddend, order.AmountForSale); err != nil {
		return err
	}
	// send the funds to the recipient address
	if err = s.AccountAdd(crypto.NewAddress(order.LockerReceiveAddress), order.AmountForSale); err != nil {
		return err
	}
	// delete the order
	return s.DeleteOrder(orderId, chainId)
}

// DeleteOrder() deletes an existing order in the order book for a committee in the state db
func (s *StateMachine) DeleteOrder(orderId, chainId uint64) (err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return
	}
	if err = orderBook.UpdateOrder(int(orderId), nil); err != nil {
		return
	}
	err = s.SetOrderBook(orderBook)
	return
}

// GetOrder() sets the order book for a committee in the state db
func (s *StateMachine) GetOrder(orderId uint64, chainId uint64) (order *lib.SellOrder, err lib.ErrorI) {
	orderBook, err := s.GetOrderBook(chainId)
	if err != nil {
		return nil, err
	}
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
		// write the order book for the committee to state
		key := types.KeyForOrderBook(book.ChainId)
		if err = s.store.Set(key, orderBookBz); err != nil {
			return err
		}
		// properly mint to the supply pool
		for _, order := range book.Orders {
			supply.Total += order.AmountForSale
			if err = s.PoolAdd(book.ChainId+uint64(types.EscrowPoolAddend), order.AmountForSale); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetOrderBook() retrieves the order book for a committee from the state db
func (s *StateMachine) GetOrderBook(chainId uint64) (b *lib.OrderBook, err lib.ErrorI) {
	// initialize the order book variable
	b = new(lib.OrderBook)
	b.Orders, b.ChainId = make([]*lib.SellOrder, 0), chainId
	// get order book bytes from the db
	bz, err := s.Get(types.KeyForOrderBook(chainId))
	if err != nil {
		return
	}
	// convert order book bytes into the order book variable
	err = lib.Unmarshal(bz, b)
	return
}

// GetOrderBooks() retrieves all OrderBooks from the state db
func (s *StateMachine) GetOrderBooks() (b *lib.OrderBooks, err lib.ErrorI) {
	b = new(lib.OrderBooks)
	it, err := s.Iterator(types.OrderBookPrefix())
	if err != nil {
		return
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		id, e := types.IdFromKey(it.Key())
		if e != nil {
			return nil, e
		}
		book, e := s.GetOrderBook(id)
		if e != nil {
			return nil, e
		}
		b.OrderBooks = append(b.OrderBooks, book)
	}
	return
}
