package fsm

import (
	"bytes"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

/* All things related to transaction payloads (Messages) */

// MESSAGE HANDLER CODE BELOW

// HandleMessage() routes the MessageI to the correct `handler` based on its `type`
func (s *StateMachine) HandleMessage(msg lib.MessageI) lib.ErrorI {
	// for the message received, route it to the proper handler
	switch x := msg.(type) {
	case *MessageSend:
		return s.HandleMessageSend(x)
	case *MessageStake:
		return s.HandleMessageStake(x)
	case *MessageEditStake:
		return s.HandleMessageEditStake(x)
	case *MessageUnstake:
		return s.HandleMessageUnstake(x)
	case *MessagePause:
		return s.HandleMessagePause(x)
	case *MessageUnpause:
		return s.HandleMessageUnpause(x)
	case *MessageChangeParameter:
		return s.HandleMessageChangeParameter(x)
	case *MessageDAOTransfer:
		return s.HandleMessageDAOTransfer(x)
	case *MessageCertificateResults:
		return s.HandleMessageCertificateResults(x)
	case *MessageSubsidy:
		return s.HandleMessageSubsidy(x)
	case *MessageCreateOrder:
		return s.HandleMessageCreateOrder(x)
	case *MessageEditOrder:
		return s.HandleMessageEditOrder(x)
	case *MessageDeleteOrder:
		return s.HandleMessageDeleteOrder(x)
	case *MessageDexLimitOrder:
		return s.HandleMessageDexLimitOrder(x)
	case *MessageDexLiquidityDeposit:
		return s.HandleMessageDexLiquidityDeposit(x)
	case *MessageDexLiquidityWithdraw:
		return s.HandleMessageDexLiquidityWithdraw(x)
	default:
		return ErrUnknownMessage(x)
	}
}

// HandleMessageSend() is the proper handler for a `Send` message
func (s *StateMachine) HandleMessageSend(msg *MessageSend) lib.ErrorI {
	// subtract from sender
	if err := s.AccountSub(crypto.NewAddressFromBytes(msg.FromAddress), msg.Amount); err != nil {
		return err
	}
	// add to recipient
	return s.AccountAdd(crypto.NewAddressFromBytes(msg.ToAddress), msg.Amount)
}

// HandleMessageStake() is the proper handler for a `Stake` message (Validator does not yet exist in the state)
func (s *StateMachine) HandleMessageStake(msg *MessageStake) lib.ErrorI {
	// convert the message public key bytes into a public key object
	publicKey, e := crypto.NewPublicKeyFromBytes(msg.PublicKey)
	if e != nil {
		return ErrInvalidPublicKey(e)
	}
	// the public key must be a BLS public key to be a validator in order to participate in consensus for efficient signature aggregation
	if !msg.Delegate {
		if _, ok := publicKey.(*crypto.BLS12381PublicKey); !ok {
			return ErrInvalidPublicKey(e)
		}
	}
	// check the net address of the message
	if err := CheckNetAddress(msg.NetAddress, msg.Delegate); err != nil {
		return err
	}
	// extract the address from the BLS public key
	address := publicKey.Address()
	// check if validator exists in state
	exists, err := s.GetValidatorExists(address)
	if err != nil {
		return err
	}
	// fail if validator already exists
	if exists {
		return ErrValidatorExists()
	}
	// subtract the tokens being locked from the signer account
	if err = s.AccountSub(crypto.NewAddress(msg.Signer), msg.Amount); err != nil {
		return err
	}
	// add to the 'total staked' tokens count in the state's supply tracker
	if err = s.AddToStakedSupply(msg.Amount); err != nil {
		return err
	}
	// if the validator is not 'actively participating' it is a delegate
	if msg.Delegate {
		// add to the 'delegate only' staked tokens count in the state's supply tracker
		if err = s.AddToDelegateSupply(msg.Amount); err != nil {
			return err
		}
		// set delegated validator in each committee in state
		if err = s.SetDelegations(address, msg.Amount, msg.Committees); err != nil {
			return err
		}
	} else {
		// set validator in each committee in state
		if err = s.SetCommittees(address, msg.Amount, msg.Committees); err != nil {
			return err
		}
	}
	// set validator in state
	return s.SetValidator(&Validator{
		Address:      address.Bytes(),
		PublicKey:    publicKey.Bytes(),
		NetAddress:   msg.NetAddress,
		StakedAmount: msg.Amount,
		Committees:   msg.Committees,
		Output:       msg.OutputAddress,
		Delegate:     msg.Delegate,
		Compound:     msg.Compound,
	})
}

// HandleMessageEditStake() is the proper handler for a `Edit-Stake` message (Validator already exists in the state)
func (s *StateMachine) HandleMessageEditStake(msg *MessageEditStake) lib.ErrorI {
	// convert the message bytes from the edit-stake message to an object
	address := crypto.NewAddressFromBytes(msg.Address)
	// get the validator from state, if not exists error
	val, err := s.GetValidator(address)
	if err != nil {
		return err
	}
	// check the net address of the message
	if err = CheckNetAddress(msg.NetAddress, val.Delegate); err != nil {
		return err
	}
	// ensure the validator is not currently unstaking
	if val.UnstakingHeight != 0 {
		return ErrValidatorUnstaking()
	}
	// check if output address is being modified and ensure the signer is authorized to do this action
	if !bytes.Equal(val.Output, msg.OutputAddress) && !bytes.Equal(val.Output, msg.Signer) {
		return ErrUnauthorizedTx()
	}
	// calculate the amount to add (if any)
	var amountToAdd uint64
	// handle the various cases depending on the 'new' stake amount
	switch {
	// amount less than stake is allowed to avoid race conditions due to auto-compounding
	// amount LTE stake
	case msg.Amount <= val.StakedAmount:
		amountToAdd = 0
	// amount greater than stake
	case msg.Amount > val.StakedAmount:
		// calculate the amount to add
		amountToAdd = msg.Amount - val.StakedAmount
	}
	// subtract from signer account
	if err = s.AccountSub(crypto.NewAddress(msg.Signer), amountToAdd); err != nil {
		return err
	}
	// update validator the validator's stake
	return s.UpdateValidatorStake(&Validator{
		Address:         val.Address,
		PublicKey:       val.PublicKey,
		NetAddress:      msg.NetAddress,
		StakedAmount:    val.StakedAmount,
		Committees:      val.Committees,
		MaxPausedHeight: val.MaxPausedHeight,
		UnstakingHeight: val.UnstakingHeight,
		Output:          msg.OutputAddress,
		Delegate:        val.Delegate,
		Compound:        msg.Compound,
	}, msg.Committees, amountToAdd)
}

// HandleMessageUnstake() is the proper handler for an `Unstake` message
func (s *StateMachine) HandleMessageUnstake(msg *MessageUnstake) lib.ErrorI {
	// extract an address object from the address bytes in the message
	address := crypto.NewAddressFromBytes(msg.Address)
	// get validator object from state
	validator, err := s.GetValidator(address)
	if err != nil {
		return err
	}
	// check if the validator is already unstaking
	if validator.UnstakingHeight != 0 {
		return ErrValidatorUnstaking()
	}
	// get the governance parameters for 'unstaking blocks'
	p, err := s.GetParamsVal()
	if err != nil {
		return err
	}
	// get unstaking blocks parameter
	var unstakingBlocks uint64
	// if the validator isn't a delegator
	if !validator.Delegate {
		// use UnstakingBlocks for validators
		unstakingBlocks = p.UnstakingBlocks
	} else {
		// use UnstakingBlocks for delegators
		unstakingBlocks = p.DelegateUnstakingBlocks
	}
	// calculate the unstaking height for the validator
	unstakingHeight := s.Height() + unstakingBlocks
	// set the validator as 'unstaking' in the state
	return s.SetValidatorUnstaking(address, validator, unstakingHeight)
}

// HandleMessagePause() is the proper handler for an `Pause` message
func (s *StateMachine) HandleMessagePause(msg *MessagePause) lib.ErrorI {
	// extract the address object from the address bytes from the pause message
	address := crypto.NewAddressFromBytes(msg.Address)
	// get the validator from the state
	validator, err := s.GetValidator(address)
	if err != nil {
		return err
	}
	// ensure the validator is not already paused
	if validator.MaxPausedHeight != 0 {
		return ErrValidatorPaused()
	}
	// ensure the validator is not unstaking
	if validator.UnstakingHeight != 0 {
		return ErrValidatorUnstaking()
	}
	// ensure the validator is not a delegate
	if validator.Delegate {
		return ErrValidatorIsADelegate()
	}
	// get validator the parameters from state
	params, err := s.GetParamsVal()
	if err != nil {
		return err
	}
	// calculate the max paused height by adding MaxPauseBlocks to current height
	maxPausedHeight := s.Height() + params.MaxPauseBlocks
	// set the validator as paused in the state
	return s.SetValidatorPaused(address, validator, maxPausedHeight)
}

// HandleMessageUnpause() is the proper handler for an `Unpause` message
func (s *StateMachine) HandleMessageUnpause(msg *MessageUnpause) lib.ErrorI {
	// extract an address object from the message address bytes
	address := crypto.NewAddressFromBytes(msg.Address)
	// get the validator from the state
	validator, err := s.GetValidator(address)
	if err != nil {
		return err
	}
	// ensure the validator is already paused
	if validator.MaxPausedHeight == 0 {
		return ErrValidatorNotPaused()
	}
	// ensure the validator is not unstaking
	// theoretically should not happen as an unstaking validator should never be paused
	if validator.UnstakingHeight != 0 {
		return ErrValidatorUnstaking()
	}
	// ensure the validator is not a delegate
	// theoretically should not happen as a delegate should never be paused
	if validator.Delegate {
		return ErrValidatorIsADelegate()
	}
	// set the validator as unpaused in the state
	return s.SetValidatorUnpaused(address, validator)
}

// HandleMessageChangeParameter() is the proper handler for an `Change-Parameter` message
func (s *StateMachine) HandleMessageChangeParameter(msg *MessageChangeParameter) lib.ErrorI {
	// requires explicit approval from +2/3 maj of the validator set
	if err := s.ApproveProposal(msg); err != nil {
		return ErrRejectProposal()
	}
	// extract the value from the proto packed 'any'
	protoMsg, err := lib.FromAny(msg.ParameterValue)
	if err != nil {
		return err
	}
	// update the parameter
	return s.UpdateParam(msg.ParameterSpace, msg.ParameterKey, protoMsg)
}

// HandleMessageDAOTransfer() is the proper handler for a `DAO-Transfer` message
func (s *StateMachine) HandleMessageDAOTransfer(msg *MessageDAOTransfer) lib.ErrorI {
	// requires explicit approval from +2/3 maj of the validator set
	if err := s.ApproveProposal(msg); err != nil {
		return ErrRejectProposal()
	}
	// remove from DAO fund
	if err := s.PoolSub(lib.DAOPoolID, msg.Amount); err != nil {
		return err
	}
	// add to account
	return s.AccountAdd(crypto.NewAddressFromBytes(msg.Address), msg.Amount)
}

// HandleMessageCertificateResults() is the proper handler for a `CertificateResults` message
func (s *StateMachine) HandleMessageCertificateResults(msg *MessageCertificateResults) lib.ErrorI {
	// load the root chain id from state
	rootChainId, err := s.GetRootChainId()
	if err != nil {
		return err
	}
	// block any tx message certificate result for root or self chain id, as it is stored in the qc
	if msg.Qc.Header.ChainId == rootChainId || msg.Qc.Header.ChainId == s.Config.ChainId {
		return ErrInvalidCertificateResults()
	}
	s.log.Debugf("Handling certificate results msg with height %d:%d", msg.Qc.Header.Height, msg.Qc.Header.RootHeight)
	// define convenience variables
	chainId := msg.Qc.Header.ChainId
	// get committee for the QC
	committee, err := s.LoadCommittee(chainId, msg.Qc.Header.RootHeight)
	if err != nil {
		return err
	}
	// ensure it's a valid QC
	// max block size is 0 here because there should not be a block attached to this QC
	isPartialQC, err := msg.Qc.Check(committee, 0, &lib.View{NetworkId: uint64(s.NetworkID), ChainId: chainId}, false)
	if err != nil {
		return err
	}
	// if it's not signed by a +2/3rds committee majority
	if isPartialQC {
		return lib.ErrNoMaj23()
	}
	// handle the certificate results
	return s.HandleCertificateResults(msg.Qc, &committee)
}

// HandleMessageSubsidy() is the proper handler for a `Subsidy` message
func (s *StateMachine) HandleMessageSubsidy(msg *MessageSubsidy) lib.ErrorI {
	// get the retired status of the committee
	retired, err := s.CommitteeIsRetired(msg.ChainId)
	if err != nil {
		return err
	}
	// ensure the committee isn't retired
	if retired {
		return ErrNonSubsidizedCommittee()
	}
	// subtract from sender
	if err = s.AccountSub(crypto.NewAddressFromBytes(msg.Address), msg.Amount); err != nil {
		return err
	}
	// add to recipient committee
	return s.PoolAdd(msg.ChainId, msg.Amount)
}

// HandleMessageCreateOrder() is the proper handler for a `CreateOrder` message
func (s *StateMachine) HandleMessageCreateOrder(msg *MessageCreateOrder) (err lib.ErrorI) {
	valParams, err := s.GetParamsVal()
	if err != nil {
		return
	}
	// ensure order isn't below the minimum size
	if msg.AmountForSale < valParams.MinimumOrderSize {
		return ErrMinimumOrderSize()
	}
	// subtract from account balance
	address := crypto.NewAddress(msg.SellersSendAddress)
	if err = s.AccountSub(address, msg.AmountForSale); err != nil {
		return
	}
	// add to committee escrow pool
	if err = s.PoolAdd(msg.ChainId+uint64(EscrowPoolAddend), msg.AmountForSale); err != nil {
		return
	}
	// save the order in state
	return s.SetOrder(&lib.SellOrder{
		Id:                   msg.OrderId,
		Committee:            msg.ChainId,
		Data:                 msg.Data,
		AmountForSale:        msg.AmountForSale,
		RequestedAmount:      msg.RequestedAmount,
		SellerReceiveAddress: msg.SellerReceiveAddress,
		SellersSendAddress:   msg.SellersSendAddress,
	}, msg.ChainId)
}

// HandleMessageEditOrder() is the proper handler for a `EditOrder` message
func (s *StateMachine) HandleMessageEditOrder(msg *MessageEditOrder) (err lib.ErrorI) {
	order, err := s.GetOrder(msg.OrderId, msg.ChainId)
	if err != nil {
		return
	}
	// ensure the order isn't locked
	if order.BuyerReceiveAddress != nil {
		return lib.ErrOrderLocked()
	}
	// get the validator params from state
	valParams, err := s.GetParamsVal()
	if err != nil {
		return
	}
	// ensure order isn't below the minimum size
	if msg.AmountForSale < valParams.MinimumOrderSize {
		return ErrMinimumOrderSize()
	}
	// calculate the difference
	difference, address := int(msg.AmountForSale-order.AmountForSale), crypto.NewAddress(order.SellersSendAddress)
	// if adding to the order
	if difference > 0 {
		amountDifference := uint64(difference)
		if err = s.AccountSub(address, amountDifference); err != nil {
			return
		}
		// add to committee escrow pool
		if err = s.PoolAdd(msg.ChainId+uint64(EscrowPoolAddend), amountDifference); err != nil {
			return
		}
		// if subtracting from the order
	} else if difference < 0 {
		amountDifference := uint64(difference * -1)
		// subtract from the committee escrow pool
		if err = s.PoolSub(msg.ChainId+uint64(EscrowPoolAddend), amountDifference); err != nil {
			return
		}
		if err = s.AccountAdd(address, amountDifference); err != nil {
			return
		}
	}
	// set the new order in state
	return s.SetOrder(&lib.SellOrder{
		Id:                   order.Id,
		Committee:            msg.ChainId,
		Data:                 msg.Data,
		AmountForSale:        msg.AmountForSale,
		RequestedAmount:      msg.RequestedAmount,
		SellerReceiveAddress: msg.SellerReceiveAddress,
		SellersSendAddress:   order.SellersSendAddress,
	}, msg.ChainId)
}

// HandleMessageDeleteOrder() is the proper handler for a `DeleteOrder` message
func (s *StateMachine) HandleMessageDeleteOrder(msg *MessageDeleteOrder) (err lib.ErrorI) {
	order, err := s.GetOrder(msg.OrderId, msg.ChainId)
	if err != nil {
		return
	}
	if order.BuyerReceiveAddress != nil {
		return lib.ErrOrderLocked()
	}
	// subtract from the committee escrow pool
	if err = s.PoolSub(msg.ChainId+uint64(EscrowPoolAddend), order.AmountForSale); err != nil {
		return
	}
	if err = s.AccountAdd(crypto.NewAddress(order.SellersSendAddress), order.AmountForSale); err != nil {
		return
	}
	err = s.DeleteOrder(msg.OrderId, msg.ChainId)
	return
}

// HandleMessageDexLimitOrder() is the proper handler for a `DexLimitOrder` message
func (s *StateMachine) HandleMessageDexLimitOrder(msg *MessageDexLimitOrder) (err lib.ErrorI) {
	// get the next sell batch
	batch, err := s.GetDexBatch(msg.ChainId, false)
	if err != nil {
		return err
	}
	// ensure there's some liquidity in the pool
	if batch.PoolSize == 0 || s.Config.ChainId == msg.ChainId {
		return ErrInvalidLiquidityPool()
	}
	// hard limit orders to 10K per batch to prevent unchecked state growth
	if len(batch.Orders) >= lib.MaxOrdersPerDexBatch {
		return ErrMaxDexBatchSize()
	}
	// move funds from user
	if err = s.AccountSub(crypto.NewAddress(msg.Address), msg.AmountForSale); err != nil {
		return err
	}
	// add funds to holding pool
	if err = s.PoolAdd(msg.ChainId+HoldingPoolAddend, msg.AmountForSale); err != nil {
		return err
	}
	// add the batch to the order
	batch.Orders = append(batch.Orders, &lib.DexLimitOrder{
		AmountForSale:   msg.AmountForSale,
		RequestedAmount: msg.RequestedAmount,
		Address:         msg.Address,
		OrderId:         msg.OrderId,
	})
	// update next sell batch
	return s.SetDexBatch(KeyForNextBatch(msg.ChainId), batch)
}

// HandleMessageDexLiquidityDeposit() is the proper handler for a `DexLiquidityDeposit` message
func (s *StateMachine) HandleMessageDexLiquidityDeposit(msg *MessageDexLiquidityDeposit) (err lib.ErrorI) {
	// get the next sell batch
	batch, err := s.GetDexBatch(msg.ChainId, false)
	if err != nil {
		return err
	}
	// ensure there's some liquidity in the pool
	if batch.PoolSize == 0 || s.Config.ChainId == msg.ChainId {
		return ErrInvalidLiquidityPool()
	}
	// hard limit ops to 5K per batch to prevent unchecked state growth
	if len(batch.Deposits) >= lib.MaxDepositsPerDexBatch {
		return ErrMaxDexBatchSize()
	}
	// move funds from user
	if err = s.AccountSub(crypto.NewAddress(msg.Address), msg.Amount); err != nil {
		return err
	}
	// add funds to holding pool
	if err = s.PoolAdd(msg.ChainId+HoldingPoolAddend, msg.Amount); err != nil {
		return err
	}
	// add the batch to the order
	batch.Deposits = append(batch.Deposits, &lib.DexLiquidityDeposit{
		Address: msg.Address,
		Amount:  msg.Amount,
		OrderId: msg.OrderId,
	})
	// update next sell batch
	return s.SetDexBatch(KeyForNextBatch(msg.ChainId), batch)
}

// HandleMessageDexLiquidityWithdraw() is the proper handler for a `DexLiquidityWithdrawal` message
func (s *StateMachine) HandleMessageDexLiquidityWithdraw(msg *MessageDexLiquidityWithdraw) (err lib.ErrorI) {
	// get the next sell batch
	batch, err := s.GetDexBatch(msg.ChainId, false)
	if err != nil {
		return err
	}
	// ensure there's some liquidity in the pool
	if batch.PoolSize == 0 || s.Config.ChainId == msg.ChainId {
		return ErrInvalidLiquidityPool()
	}
	// hard limit ops to 5K per batch to prevent unchecked state growth
	if len(batch.Withdrawals) >= lib.MaxWithdrawsPerDexBatch {
		return ErrMaxDexBatchSize()
	}
	// get the liquidity pool
	p, err := s.GetPool(msg.ChainId + LiquidityPoolAddend)
	if err != nil {
		return err
	}
	// sanity check has liquidity points
	if _, err = p.GetPointsFor(msg.Address); err != nil {
		return err
	}
	// add the batch to the order
	batch.Withdrawals = append(batch.Withdrawals, &lib.DexLiquidityWithdraw{
		Address: msg.Address,
		Percent: msg.Percent,
		OrderId: msg.OrderId,
	})
	// update next sell batch
	return s.SetDexBatch(KeyForNextBatch(msg.ChainId), batch)
}

// GetFeeForMessageName() returns the associated cost for processing a specific type of message based on the name
func (s *StateMachine) GetFeeForMessageName(name string) (fee uint64, err lib.ErrorI) {
	// retrieve the fee parameters from the state
	feeParams, err := s.GetParamsFee()
	if err != nil {
		return 0, err
	}
	// return the proper fee based on the message name
	switch name {
	case MessageSendName:
		return feeParams.SendFee, nil
	case MessageStakeName:
		return feeParams.StakeFee, nil
	case MessageEditStakeName:
		return feeParams.EditStakeFee, nil
	case MessageUnstakeName:
		return feeParams.UnstakeFee, nil
	case MessagePauseName:
		return feeParams.PauseFee, nil
	case MessageUnpauseName:
		return feeParams.UnpauseFee, nil
	case MessageChangeParameterName:
		return feeParams.ChangeParameterFee, nil
	case MessageDAOTransferName:
		return feeParams.DaoTransferFee, nil
	case MessageCertificateResultsName:
		return feeParams.CertificateResultsFee, nil
	case MessageSubsidyName:
		return feeParams.SubsidyFee, nil
	case MessageCreateOrderName:
		return feeParams.CreateOrderFee, nil
	case MessageEditOrderName:
		return feeParams.EditOrderFee, nil
	case MessageDeleteOrderName:
		return feeParams.DeleteOrderFee, nil
	case MessageDexLimitOrderName:
		return feeParams.DexLimitOrderFee, nil
	case MessageDexLiquidityDepositName:
		return feeParams.DexLiquidityDepositFee, nil
	case MessageDexLiquidityWithdrawName:
		return feeParams.DexLiquidityWithdrawFee, nil
	default:
		return 0, lib.ErrUnknownMessageName(name)
	}
}

// GetAuthorizedSignersFor() returns the addresses that are authorized to sign for this message
func (s *StateMachine) GetAuthorizedSignersFor(msg lib.MessageI) (signers [][]byte, err lib.ErrorI) {
	// based of the message type, route to the proper authorized signers for each message type
	switch x := msg.(type) {
	case *MessageSend:
		return [][]byte{x.FromAddress}, nil
	case *MessageStake:
		address, e := s.pubKeyBytesToAddress(x.PublicKey)
		if e != nil {
			return nil, e
		}
		return [][]byte{address, x.OutputAddress}, nil
	case *MessageEditStake:
		return s.GetAuthorizedSignersForValidator(x.Address)
	case *MessageUnstake:
		return s.GetAuthorizedSignersForValidator(x.Address)
	case *MessagePause:
		return s.GetAuthorizedSignersForValidator(x.Address)
	case *MessageUnpause:
		return s.GetAuthorizedSignersForValidator(x.Address)
	case *MessageChangeParameter:
		return [][]byte{x.Signer}, nil
	case *MessageDAOTransfer:
		return [][]byte{x.Address}, nil
	case *MessageSubsidy:
		return [][]byte{x.Address}, nil
	case *MessageCertificateResults:
		address, e := s.pubKeyBytesToAddress(x.Qc.ProposerKey)
		if e != nil {
			return nil, e
		}
		return [][]byte{address}, nil
	case *MessageCreateOrder:
		return [][]byte{x.SellersSendAddress}, nil
	case *MessageEditOrder:
		order, e := s.GetOrder(x.OrderId, x.ChainId)
		if e != nil {
			return nil, e
		}
		return [][]byte{order.SellersSendAddress}, nil
	case *MessageDeleteOrder:
		order, e := s.GetOrder(x.OrderId, x.ChainId)
		if e != nil {
			return nil, e
		}
		return [][]byte{order.SellersSendAddress}, nil
	case *MessageDexLimitOrder:
		return [][]byte{x.Address}, nil
	case *MessageDexLiquidityDeposit:
		return [][]byte{x.Address}, nil
	case *MessageDexLiquidityWithdraw:
		return [][]byte{x.Address}, nil
	default:
		return nil, ErrUnknownMessage(x)
	}
}
