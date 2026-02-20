package fsm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"sort"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
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
	// create a new reference to a 'lock order' object in order to ensure a non-nil result
	bo = new(lib.LockOrder)
	// attempt to unmarshal the transaction memo into a 'lock order'
	if err := lib.UnmarshalJSON([]byte(tx.Memo), bo); err == nil {
		// sanity check some critical fields of the 'lock order' to ensure the unmarshal was successful
		if len(bo.BuyerSendAddress) != 0 && len(bo.BuyerReceiveAddress) != 0 && bo.ChainId == s.Config.ChainId {
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
	return co, co.ChainId == s.Config.ChainId && co.CloseOrder // signals if this is a 'close order' or not
}

// ProcessRootChainOrderBook() processes the order book from the root-chain and cross-references blocks on this chain to determine
// actions that warrant committee level changes to the root-chain order book like: LockOrder, ResetOrder and CloseOrder
func (s *StateMachine) ProcessRootChainOrderBook(book *lib.OrderBook, proposalBlock *lib.BlockResult) (lockOrders []*lib.LockOrder, closedOrders, resetOrders [][]byte) {
	if book == nil {
		return
	}
	blocks := []*lib.BlockResult{proposalBlock}
	// historical checking logic:
	// don't do historical checking before block 16
	if proposalBlock.BlockHeader.Height >= 16 {
		// calculate the bounds of the loop (N-15 to N-10)
		start, end := proposalBlock.BlockHeader.Height-15, proposalBlock.BlockHeader.Height-10
		// for 5 historical blocks (skips recent to ensure root mempool has time to process already submitted)
		for i := start; i < end; i++ {
			// load the block (hopefully from cache)
			block, err := s.LoadBlock(i)
			if err != nil {
				s.log.Error(err.Error())
				continue
			}
			// add block to the list to check
			blocks = append(blocks, block)
		}
	}
	// parse blocks for lock and close orders
	lockedOrders, closeOrders, coSends := s.ParseBlockForLockAndCloseOrders(blocks...)
	// for each order in the book
	for _, order := range book.Orders {
		// if the order is not locked
		if len(order.BuyerReceiveAddress) == 0 {
			// attempt to get the lock order command from the map
			lockOrder, found := lockedOrders[string(order.Id)]
			if !found {
				continue
			}
			// add to lock orders
			lockOrders = append(lockOrders, lockOrder)
		} else {
			// see if the 'locked' order is expired
			if s.height > order.BuyerChainDeadline {
				// add to reset orders
				resetOrders = append(resetOrders, order.Id)
				// go to the next order
				continue
			}
			// attempt to get the close order command from the map
			closeOrder, found := closeOrders[string(order.Id)]
			if !found {
				continue
			}
			// get the co send to verify the amount
			send := coSends[string(order.Id)]
			// check that sent amount == request amount
			if send.Amount != order.RequestedAmount {
				s.log.Errorf("close order error: sent amount does not equal requested amount, id: %s", lib.BytesToString(closeOrder.OrderId))
				continue
			}
			// add to closed orders
			closedOrders = append(closedOrders, closeOrder.OrderId)
		}
	}
	// exit
	return
}

// ParseCloseOrders() parses the blocks for memo commands to execute specialized 'close order' functionality
func (s *StateMachine) ParseBlockForLockAndCloseOrders(blocks ...*lib.BlockResult) (lockOrders map[string]*lib.LockOrder, closeOrders map[string]*lib.CloseOrder, coSends map[string]*MessageSend) {
	// get the governance parameters from state
	params, err := s.GetParams()
	if err != nil {
		s.log.Error(err.Error())
		return
	}
	// calculate the minimum lock order fee
	minFee := params.Fee.SendFee * params.Validator.LockOrderFeeMultiplier
	// make the maps
	lockOrders = make(map[string]*lib.LockOrder)
	closeOrders = make(map[string]*lib.CloseOrder)
	coSends = make(map[string]*MessageSend)
	// for each block
	for _, b := range blocks {
		// for each transaction in the block
		for _, tx := range b.Transactions {
			// skip over any that doesn't have the minimum fee or isn't the correct type
			if tx.MessageType != MessageSendName || tx.Transaction.Memo == "" || tx.Transaction.Fee < minFee || !json.Valid([]byte(tx.Transaction.Memo)) {
				continue
			}
			// parse the transaction for embedded 'lock orders'
			if lockOrder, ok := s.ParseLockOrder(tx.Transaction, params.Validator.BuyDeadlineBlocks); ok {
				// add to the 'lock orders' list
				lockOrders[string(lockOrder.OrderId)] = lockOrder
				// continue
				continue
			}
			// try parse close orders
			if closeOrder, ok := s.ParseCloseOrder(tx.Transaction); ok {
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
				// add to the 'close orders' list
				closeOrders[string(closeOrder.OrderId)] = closeOrder
				coSends[string(closeOrder.OrderId)] = send
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
	if err = s.SetOrder(order, chainId); err != nil {
		return
	}
	// emit order book lock event
	return s.EventOrderBookLock(order)
}

// ResetOrder() removes the recipient and deadline height from an existing order and saves it to the state
func (s *StateMachine) ResetOrder(orderId []byte, chainId uint64) (err lib.ErrorI) {
	// get the order from state
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		return
	}
	// emit order book reset event before resetting the order (so we have access to order details)
	if err = s.EventOrderBookReset(order); err != nil {
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
	// add swap event
	if err = s.EventOrderBookSwap(order); err != nil {
		return
	}
	// delete the order
	return s.DeleteOrder(orderId, chainId)
}

// SetOrder() sets the sell order in state
func (s *StateMachine) SetOrder(order *lib.SellOrder, chainId uint64) (err lib.ErrorI) {
	// clean up stale buyer index if buyer changed or was removed
	if err = s.cleanupStaleBuyerIndex(order, chainId); err != nil {
		return
	}
	// convert the order into proto bytes
	protoBytes, err := s.marshalOrder(order)
	if err != nil {
		return
	}
	// set the order book in state
	if err = s.Set(KeyForOrder(chainId, order.Id), protoBytes); err != nil {
		return
	}
	// set the secondary index for seller address lookup (value is empty, key is sufficient)
	if err = s.Set(KeyForOrderBySeller(order.SellersSendAddress, chainId, order.Id), []byte{}); err != nil {
		return
	}
	// set the secondary index for buyer address lookup if buyer exists (locked order)
	if len(order.BuyerSendAddress) > 0 {
		if err = s.Set(KeyForOrderByBuyer(order.BuyerSendAddress, chainId, order.Id), []byte{}); err != nil {
			return
		}
	}
	return
}

// cleanupStaleBuyerIndex() removes the old buyer index entry if the buyer changed or was removed
func (s *StateMachine) cleanupStaleBuyerIndex(order *lib.SellOrder, chainId uint64) lib.ErrorI {
	// check if order already exists
	existingOrder, err := s.GetOrder(order.Id, chainId)
	// if order not found, nothing to clean up
	if err != nil && err.Code() == lib.CodeOrderNotFound {
		return nil
	}
	// if other error, return it
	if err != nil {
		return err
	}
	// if existing order has a buyer and it differs from the new order's buyer, clean up the old index
	if existingOrder != nil && len(existingOrder.BuyerSendAddress) > 0 {
		if !bytes.Equal(existingOrder.BuyerSendAddress, order.BuyerSendAddress) {
			return s.Delete(KeyForOrderByBuyer(existingOrder.BuyerSendAddress, chainId, order.Id))
		}
	}
	return nil
}

// DeleteOrder() deletes an existing order in the order book for a committee in the state db
func (s *StateMachine) DeleteOrder(orderId []byte, chainId uint64) (err lib.ErrorI) {
	// get the order first to retrieve addresses for index cleanup
	order, err := s.GetOrder(orderId, chainId)
	if err != nil {
		return
	}
	// delete the primary order entry
	if err = s.Delete(KeyForOrder(chainId, orderId)); err != nil {
		return
	}
	// delete the seller secondary index entry
	if err = s.Delete(KeyForOrderBySeller(order.SellersSendAddress, chainId, orderId)); err != nil {
		return
	}
	// delete the buyer secondary index entry if buyer exists
	if len(order.BuyerSendAddress) > 0 {
		if err = s.Delete(KeyForOrderByBuyer(order.BuyerSendAddress, chainId, orderId)); err != nil {
			return
		}
	}
	return
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
	return s.Set(OrderBookPrefix(b.ChainId), orderBookBz)
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

// GetOrdersBySeller() retrieves all orders for a specific seller address, optionally filtered by chainId
func (s *StateMachine) GetOrdersBySeller(seller []byte, chainId uint64) (b *lib.OrderBooks, err lib.ErrorI) {
	b = new(lib.OrderBooks)
	// determine the prefix to iterate based on whether chainId filter is provided
	var prefix []byte
	if chainId == 0 {
		prefix = OrderBySellerPrefix(seller)
	} else {
		prefix = OrderBySellerAndChainPrefix(seller, chainId)
	}
	// create an iterator over the seller index prefix
	it, err := s.Iterator(prefix)
	if err != nil {
		return
	}
	defer it.Close()
	// map to collect orders by chainId
	ordersByChain := make(map[uint64][]*lib.SellOrder)
	// for each index entry
	for ; it.Valid(); it.Next() {
		// extract chainId and orderId from the index key
		cId, orderId, e := s.parseOrderBySellerKey(it.Key())
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		// get the actual order from the primary store
		order, e := s.GetOrder(orderId, cId)
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		ordersByChain[cId] = append(ordersByChain[cId], order)
	}
	// convert map to OrderBooks structure
	for cId, orders := range ordersByChain {
		b.OrderBooks = append(b.OrderBooks, &lib.OrderBook{
			ChainId: cId,
			Orders:  orders,
		})
	}
	// sort by chain id for consistent output
	sort.Slice(b.OrderBooks, func(i, j int) bool {
		return b.OrderBooks[i].ChainId < b.OrderBooks[j].ChainId
	})
	return
}

// parseOrderBySellerKey() extracts chainId and orderId from an order-by-seller index key
func (s *StateMachine) parseOrderBySellerKey(key []byte) (chainId uint64, orderId []byte, err lib.ErrorI) {
	// key format: orderBySellerPrefix + seller + chainId + orderId (all length-prefixed)
	segments := lib.DecodeLengthPrefixed(key)
	if len(segments) < 4 {
		return 0, nil, ErrInvalidKey(key)
	}
	// segments[0] = prefix, segments[1] = seller, segments[2] = chainId, segments[3] = orderId
	chainId = binary.BigEndian.Uint64(segments[2])
	orderId = segments[3]
	return
}

// parseOrderByBuyerKey() extracts chainId and orderId from an order-by-buyer index key
func (s *StateMachine) parseOrderByBuyerKey(key []byte) (chainId uint64, orderId []byte, err lib.ErrorI) {
	// key format: orderByBuyerPrefix + buyer + chainId + orderId (all length-prefixed)
	segments := lib.DecodeLengthPrefixed(key)
	if len(segments) < 4 {
		return 0, nil, ErrInvalidKey(key)
	}
	// segments[0] = prefix, segments[1] = buyer, segments[2] = chainId, segments[3] = orderId
	chainId = binary.BigEndian.Uint64(segments[2])
	orderId = segments[3]
	return
}

// GetOrdersPaginated() retrieves orders with optional filters and pagination
// Note: seller and buyer filters are mutually exclusive
func (s *StateMachine) GetOrdersPaginated(seller, buyer []byte, chainId uint64, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	// create the page object
	page := lib.NewPage(p, "orders")
	results := &lib.SellOrders{}
	// determine which query path to use based on filters
	if len(seller) > 0 {
		// use seller index for efficient lookup
		return s.getOrdersBySellerPaginated(seller, chainId, page, results)
	}
	if len(buyer) > 0 {
		// use buyer index for efficient lookup
		return s.getOrdersByBuyerPaginated(buyer, chainId, page, results)
	}
	if chainId != 0 {
		// filter by chainId only - iterate orders for that chain
		return s.getOrdersByChainPaginated(chainId, page, results)
	}
	// no filters - iterate all orders across all chains
	return s.getAllOrdersPaginated(page, results)
}

// getOrdersBySellerPaginated() retrieves paginated orders using the seller index
func (s *StateMachine) getOrdersBySellerPaginated(seller []byte, chainId uint64, page *lib.Page, results *lib.SellOrders) (*lib.Page, lib.ErrorI) {
	// determine the prefix based on whether chainId filter is provided
	var prefix []byte
	if chainId == 0 {
		prefix = OrderBySellerPrefix(seller)
	} else {
		prefix = OrderBySellerAndChainPrefix(seller, chainId)
	}
	// use the page Load function with the index prefix
	err := page.Load(prefix, false, results, s, func(k, v []byte) lib.ErrorI {
		// extract chainId and orderId from the index key
		cId, orderId, e := s.parseOrderBySellerKey(k)
		if e != nil {
			return e
		}
		// get the actual order from the primary store
		order, e := s.GetOrder(orderId, cId)
		if e != nil {
			return e
		}
		*results = append(*results, order)
		return nil
	})
	return page, err
}

// getOrdersByBuyerPaginated() retrieves paginated orders using the buyer index
func (s *StateMachine) getOrdersByBuyerPaginated(buyer []byte, chainId uint64, page *lib.Page, results *lib.SellOrders) (*lib.Page, lib.ErrorI) {
	// determine the prefix based on whether chainId filter is provided
	var prefix []byte
	if chainId == 0 {
		prefix = OrderByBuyerPrefix(buyer)
	} else {
		prefix = OrderByBuyerAndChainPrefix(buyer, chainId)
	}
	// use the page Load function with the index prefix
	err := page.Load(prefix, false, results, s, func(k, v []byte) lib.ErrorI {
		// extract chainId and orderId from the index key
		cId, orderId, e := s.parseOrderByBuyerKey(k)
		if e != nil {
			return e
		}
		// get the actual order from the primary store
		order, e := s.GetOrder(orderId, cId)
		if e != nil {
			return e
		}
		*results = append(*results, order)
		return nil
	})
	return page, err
}

// getOrdersByChainPaginated() retrieves paginated orders for a specific chain
func (s *StateMachine) getOrdersByChainPaginated(chainId uint64, page *lib.Page, results *lib.SellOrders) (*lib.Page, lib.ErrorI) {
	// use the page Load function with the order book prefix
	err := page.Load(OrderBookPrefix(chainId), false, results, s, func(k, v []byte) lib.ErrorI {
		order, e := s.unmarshalOrder(v)
		if e != nil {
			return e
		}
		*results = append(*results, order)
		return nil
	})
	return page, err
}

// getAllOrdersPaginated() retrieves paginated orders across all chains
func (s *StateMachine) getAllOrdersPaginated(page *lib.Page, results *lib.SellOrders) (*lib.Page, lib.ErrorI) {
	// use the page Load function with the base order book prefix
	err := page.Load(lib.JoinLenPrefix(orderBookPrefix), false, results, s, func(k, v []byte) lib.ErrorI {
		order, e := s.unmarshalOrder(v)
		if e != nil {
			return e
		}
		*results = append(*results, order)
		return nil
	})
	return page, err
}

// GetTotalEscrowed() checks all order books for escrowed funds for a specific address
func (s *StateMachine) GetTotalEscrowed(address crypto.AddressI) (total uint64, err lib.ErrorI) {
	orderBooks, err := s.GetOrderBooks()
	if err != nil {
		return
	}
	// for each order book
	for _, book := range orderBooks.OrderBooks {
		// for each order
		for _, order := range book.Orders {
			if address == nil || bytes.Equal(order.SellersSendAddress, address.Bytes()) {
				total += order.AmountForSale
			}
		}
	}
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
