package fsm

import (
	"bytes"
	"encoding/json"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"sort"
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

// BUYER SIDE LOGIC

// ParseLockOrder() parses a transaction for an embedded lock order messages in the memo field
func (s *StateMachine) ParseLockOrder(tx *lib.Transaction, deadlineBlocks uint64) (bo *lib.LockOrder, ok bool) {
	// check for valid json
	if !json.Valid([]byte(tx.Memo)) {
		return nil, false
	}
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

// ParseCloseOrder() parses a transaction for an embedded close order messages in the memo field
func (s *StateMachine) ParseCloseOrder(tx *lib.Transaction) (co *lib.CloseOrder, ok bool) {
	// create a new reference to a 'close order' object in order to ensure a non-nil result
	co = new(lib.CloseOrder)
	// attempt to parse the close order from the memo
	if err := lib.UnmarshalJSON([]byte(tx.Memo), co); err != nil {
		return nil, false
	}
	// exit
	return co, co.CloseOrder // signals if this is a 'close order' or not
}

// ProcessRootChainOrderBook() processes the order book from the root-chain and cross-references blocks on this chain to determine
// actions that warrant committee level changes to the root-chain order book like: ResetOrder and CloseOrder
func (s *StateMachine) ProcessRootChainOrderBook(book *lib.OrderBook, b *lib.BlockResult) (closeOrders, resetOrders [][]byte) {
	// create a variable to track the 'close orders'
	// map structure is [orderId] -> amount sent
	transferred := make(map[string]uint64)
	// get all the 'Send' transactions from the block
	for _, tx := range b.Transactions {
		// ignore non-send, memo-less, or no valid json embedded
		if tx.MessageType != MessageSendName || tx.Transaction.Memo == "" || !json.Valid([]byte(tx.Transaction.Memo)) {
			continue
		}
		// extract the message from the transaction object
		msg, e := lib.FromAny(tx.Transaction.Msg)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		// cast the message to send
		send, ok := msg.(*MessageSend)
		if !ok {
			s.log.Error("Non-send message with a send message name (should not happen)")
			continue
		}
		// try parse close orders
		closeOrder, isCloseOrder := s.ParseCloseOrder(tx.Transaction)
		// if not a 'close order'
		if !isCloseOrder {
			// not a close order
			continue
		}
		// try to get the close order
		order, err := book.GetOrder(closeOrder.OrderId)
		// if an error occurred during the retrieval
		if err != nil {
			s.log.Warnf("An error occurred during the close order retrieval: %s", err.Error())
			continue
		}
		// if the 'sent amount' is the same as the order amount
		if bytes.Equal(send.ToAddress, order.SellerReceiveAddress) && order.RequestedAmount == send.Amount {
			// update the transferred map using the [order_id] as the key and add send.Amount to the value
			transferred[lib.BytesToString(closeOrder.OrderId)] += send.Amount
		}
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
		// check if the order was closed this block
		if sent := transferred[lib.BytesToString(order.Id)]; sent == order.RequestedAmount { // double check on amount for sanity
			// add to the closed order list
			closeOrders = append(closeOrders, order.Id)
			// go to the next order
			continue
		}
		// don't do historical checking before block 16
		if b.BlockHeader.Height < 16 {
			// go to the next order
			continue
		}
		// calculate hte bounds of the loop (N-15 to N-10)
		start, end := b.BlockHeader.Height-15, b.BlockHeader.Height-10
		// from blocks N-15 to N-10
		for i := start; i < end; i++ {
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
			if qc.Results.Orders != nil && lib.ContainsByteSlice(qc.Results.Orders.CloseOrders, order.Id) {
				// ensure no already added to close order
				if !lib.ContainsByteSlice(closeOrders, order.Id) {
					// if so, add it to the close orders
					closeOrders = append(closeOrders, order.Id)
				}
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
	deDupeLockOrders := lib.NewDeDuplicator[string]()
	// for each transaction in the block
	for _, tx := range b.Transactions {
		// skip over any that doesn't have the minimum fee or isn't the correct type
		if tx.MessageType != MessageSendName || tx.Transaction.Memo == "" || tx.Transaction.Fee < minFee {
			continue
		}
		// parse the transaction for embedded 'lock orders'
		if lockOrder, ok := s.ParseLockOrder(tx.Transaction, params.Validator.BuyDeadlineBlocks); ok {
			// if not found (non-duplicate)
			if found := deDupeLockOrders.Found(lib.BytesToString(lockOrder.OrderId)); !found {
				// add to the 'lock orders' list
				lockOrders = append(lockOrders, lockOrder)
			}
		}
	}
	// exit
	return
}

// LockOrder() adds a recipient and a deadline height to an existing order and saves it to the state
func (s *StateMachine) LockOrder(lock *lib.LockOrder, chainId uint64) (err lib.ErrorI) {
	// get the order from state
	order, err := s.GetOrder(lock.OrderId, chainId)
	if err != nil {
		return
	}
	// if the buyer's receive address isn't nil
	if order.BuyerReceiveAddress != nil {
		return lib.ErrOrderLocked()
	}
	// set the buyer's receive, send, and deadline height in the order
	order.BuyerReceiveAddress = lock.BuyerReceiveAddress
	order.BuyerSendAddress = lock.BuyerSendAddress
	order.BuyerChainDeadline = lock.BuyerChainDeadline
	// set the order book back in state
	return s.SetOrder(order, chainId)
}

// ResetOrder() removes the recipient and deadline height from an existing order and saves it to the state
func (s *StateMachine) ResetOrder(orderId []byte, chainId uint64) (err lib.ErrorI) {
	// get the order from state
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		return
	}
	// reset the buyer's receive, send, and deadline height in the order
	order.BuyerReceiveAddress, order.BuyerSendAddress, order.BuyerChainDeadline = nil, nil, 0
	// set the order back in state
	return s.SetOrder(order, chainId)
}

// CloseOrder() sends the tokens from escrow to the 'buyer address' and deletes the order
func (s *StateMachine) CloseOrder(orderId []byte, chainId uint64) (err lib.ErrorI) {
	// the order is 'closed' and the tokens are moved from escrow to the buyer
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		return
	}
	// ensure the order already was 'claimed / reserved'
	if order.BuyerReceiveAddress == nil {
		return ErrInvalidLockOrder()
	}
	// remove the funds from the escrow pool
	if err = s.PoolSub(chainId+EscrowPoolAddend, order.AmountForSale); err != nil {
		return
	}
	// send the funds to the recipient address
	if err = s.AccountAdd(crypto.NewAddress(order.BuyerReceiveAddress), order.AmountForSale); err != nil {
		return
	}
	// delete the order
	return s.DeleteOrder(orderId, chainId)
}

// SetOrder() sets the sell order in state
func (s *StateMachine) SetOrder(order *lib.SellOrder, chainId uint64) (err lib.ErrorI) {
	// convert the order into proto bytes
	protoBytes, err := s.marshalOrder(order)
	if err != nil {
		return
	}
	// set the order book in state
	return s.Set(KeyForOrder(chainId, order.Id), protoBytes)
}

// DeleteOrder() deletes an existing order in the order book for a committee in the state db
func (s *StateMachine) DeleteOrder(orderId []byte, chainId uint64) (err lib.ErrorI) {
	return s.Delete(KeyForOrder(chainId, orderId))
}

// GetOrder() gets the sell order from state
func (s *StateMachine) GetOrder(orderId []byte, chainId uint64) (order *lib.SellOrder, err lib.ErrorI) {
	// get the order proto bytes from the state
	protoBytes, err := s.Get(KeyForOrder(chainId, orderId))
	if err != nil {
		return
	}
	// convert the proto bytes into an order object
	return s.unmarshalOrder(protoBytes)
}

// SetOrderBook() sets the order book for a committee in the state db
func (s *StateMachine) SetOrderBook(b *lib.OrderBook) lib.ErrorI {
	// convert the order book into bytes
	orderBookBz, err := lib.Marshal(b)
	if err != nil {
		return err
	}
	// set the order book in the store
	return s.store.Set(OrderBookPrefix(b.ChainId), orderBookBz)
}

// SetOrderBooks() sets a series of OrderBooks in the state db
func (s *StateMachine) SetOrderBooks(list *lib.OrderBooks, supply *Supply) lib.ErrorI {
	// ensure the order books object reference is not nil
	if list == nil {
		return nil
	}
	// for each book in the order books list
	for _, book := range list.OrderBooks {
		// ensure non nil book
		if book == nil {
			continue
		}
		// for each order in the book
		for _, order := range book.Orders {
			// set the order in state
			if err := s.SetOrder(order, book.ChainId); err != nil {
				return err
			}
			// update the 'supply' tracker
			supply.Total += order.AmountForSale
			// calculate the escrow pool id for a specific chainId
			escrowPoolId := book.ChainId + uint64(EscrowPoolAddend)
			// add to the 'escrow' pool for the specific id
			if err := s.PoolAdd(escrowPoolId, order.AmountForSale); err != nil {
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
	// iterate through the order book prefix
	it, err := s.Iterator(OrderBookPrefix(chainId))
	if err != nil {
		return
	}
	defer it.Close()
	// for each order under this prefix
	for ; it.Valid(); it.Next() {
		// get the order from the iterator value bytes
		order, e := s.unmarshalOrder(it.Value())
		if e != nil {
			// shouldn't happen
			s.log.Error(e.Error())
			// defensive
			continue
		}
		b.Orders = append(b.Orders, order)
	}
	return
}

// GetOrderBooks() retrieves the lists for all chainIds of open 'sell orders' from the state
func (s *StateMachine) GetOrderBooks() (b *lib.OrderBooks, err lib.ErrorI) {
	// get the order books from the state
	b = new(lib.OrderBooks)
	// create an iterator over the OrderBookPrefix
	it, err := s.Iterator(lib.JoinLenPrefix(orderBookPrefix))
	if err != nil {
		return
	}
	// deduplicate committees
	deDupe := lib.NewDeDuplicator[uint64]()
	// memory cleanup the iterator
	defer it.Close()
	// for each item under the OrderBookPrefix
	for ; it.Valid(); it.Next() {
		// extract the chainId from the key
		id, e := IdFromKey(it.Key())
		if e != nil {
			return nil, e
		}
		// skip duplicates
		if deDupe.Found(id) {
			continue
		}
		// get the specific order book for the chainId
		book, e := s.GetOrderBook(id)
		if e != nil {
			return nil, e
		}
		// add the book to the list
		b.OrderBooks = append(b.OrderBooks, book)
	}
	// sort by chain id
	sort.Slice(b.OrderBooks, func(i, j int) bool {
		return b.OrderBooks[i].ChainId < b.OrderBooks[j].ChainId
	})
	// exit
	return
}

// marshalOrder() converts the Validator object to bytes
func (s *StateMachine) marshalOrder(order *lib.SellOrder) ([]byte, lib.ErrorI) {
	// convert the object ref into bytes
	return lib.Marshal(order)
}

// unmarshalOrder() converts bytes into a SellOrder object
func (s *StateMachine) unmarshalOrder(protoBytes []byte) (*lib.SellOrder, lib.ErrorI) {
	if protoBytes == nil {
		return nil, lib.ErrOrderNotFound()
	}
	// create a new SellOrder object reference to ensure a non-nil result
	order := new(lib.SellOrder)
	// populate the object reference with validator bytes
	if err := lib.Unmarshal(protoBytes, order); err != nil {
		return nil, err
	}
	// return the object ref
	return order, nil
}
