package fsm

import (
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHandleMessage(t *testing.T) {
	const amount = uint64(100)
	// pre-create a 'change parameter' proposal to use during testing
	a, err := lib.NewAny(&lib.StringWrapper{Value: NewProtocolVersion(3, 2)})
	require.NoError(t, err)
	msgChangeParam := &MessageChangeParameter{
		ParameterSpace: "cons",
		ParameterKey:   ParamProtocolVersion,
		ParameterValue: a,
		StartHeight:    1,
		EndHeight:      2,
		Signer:         newTestAddressBytes(t),
	}
	// run test cases
	tests := []struct {
		name     string
		detail   string
		preset   func(sm StateMachine) // required state pre-set for message to be accepted
		msg      lib.MessageI
		validate func(sm StateMachine) // 'very basic' validation that the correct message was handled
		error    string
	}{
		{
			name:   "message send",
			detail: "basic 'happy path' handling for message send",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t), 100))
			},
			msg: &MessageSend{
				FromAddress: newTestAddressBytes(t),
				ToAddress:   newTestAddressBytes(t, 1),
				Amount:      amount,
			},
			validate: func(sm StateMachine) {
				// ensure the sender account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the receiver account was added to
				got, e = sm.GetAccountBalance(newTestAddress(t, 1))
				require.NoError(t, e)
				require.Equal(t, amount, got)
			},
		},
		{
			name:   "message stake",
			detail: "basic 'happy path' handling for message stake",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t), 100))
			},
			msg: &MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        amount,
				Committees:    []uint64{lib.CanopyChainId},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t),
				Delegate:      false,
				Compound:      false,
				Signer:        newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the sender account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the validator was created
				val, _ := sm.GetValidator(newTestAddress(t))
				require.NoError(t, e)
				require.True(t, val != nil)
			},
		},
		{
			name:   "message edit-stake",
			detail: "basic 'happy path' handling for message edit-stake",
			preset: func(sm StateMachine) {
				// set account balance
				require.NoError(t, sm.AccountAdd(newTestAddress(t), 1))
				// create validator
				v := &Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyChainId},
					Output:       newTestAddressBytes(t),
				}
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(v.StakedAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(v.StakedAmount))
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set validator committees
				require.NoError(t, sm.AddStakeToCommittees(v.StakedAmount, v.Committees))
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        amount + 1,
				Committees:    []uint64{lib.CanopyChainId},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t),
				Signer:        newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the sender account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the validator stake was updated
				val, e := sm.GetValidator(newTestAddress(t))
				require.NoError(t, e)
				require.Equal(t, amount+1, val.StakedAmount)
			},
		},
		{
			name:   "message unstake",
			detail: "basic 'happy path' handling for message unstake",
			preset: func(sm StateMachine) {
				// create validator
				v := &Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyChainId},
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &MessageUnstake{Address: newTestAddressBytes(t)},
			validate: func(sm StateMachine) {
				// ensure the validator is unstaking
				val, e := sm.GetValidator(newTestAddress(t))
				require.NoError(t, e)
				require.NotZero(t, val.UnstakingHeight)
			},
		},
		{
			name:   "message pause",
			detail: "basic 'happy path' handling for message pause",
			preset: func(sm StateMachine) {
				// create validator
				v := &Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyChainId},
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &MessagePause{Address: newTestAddressBytes(t)},
			validate: func(sm StateMachine) {
				// ensure the validator is paused
				val, e := sm.GetValidator(newTestAddress(t))
				require.NoError(t, e)
				require.NotZero(t, val.MaxPausedHeight)
			},
		},
		{
			name:   "message unpause",
			detail: "basic 'happy path' handling for message unpause",
			preset: func(sm StateMachine) {
				// create validator
				v := &Validator{
					Address:         newTestAddressBytes(t),
					StakedAmount:    amount,
					Committees:      []uint64{lib.CanopyChainId},
					MaxPausedHeight: 1,
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &MessageUnpause{Address: newTestAddressBytes(t)},
			validate: func(sm StateMachine) {
				// ensure the validator is paused
				val, e := sm.GetValidator(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, val.MaxPausedHeight)
			},
		},
		{
			name:   "message change param",
			detail: "basic 'happy path' handling for message change param",
			preset: func(sm StateMachine) {},
			msg:    msgChangeParam,
			validate: func(sm StateMachine) {
				// ensure the validator is paused
				consParams, e := sm.GetParamsCons()
				require.NoError(t, e)
				require.Equal(t, NewProtocolVersion(3, 2), consParams.ProtocolVersion)
			},
		},
		{
			name:   "message dao transfer",
			detail: "basic 'happy path' handling for message dao transfer",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.PoolAdd(lib.DAOPoolID, amount))
			},
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      amount,
				StartHeight: 1,
				EndHeight:   2,
			},
			validate: func(sm StateMachine) {
				// ensure the pool was subtracted from
				got, e := sm.GetPoolBalance(lib.DAOPoolID)
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the receiver account was added to
				got, e = sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Equal(t, amount, got)
			},
		},
		{
			name:   "message subsidy",
			detail: "basic 'happy path' handling for message subsidy",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t), amount))
			},
			msg: &MessageSubsidy{
				Address: newTestAddressBytes(t),
				ChainId: lib.CanopyChainId,
				Amount:  amount,
				Opcode:  "note",
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyChainId)
				require.NoError(t, e)
				require.Equal(t, amount, got)
			},
		},
		{
			name:   "message create order",
			detail: "basic 'happy path' handling for message create order",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t), amount))
				// get the validator params
				params, e := sm.GetParamsVal()
				require.NoError(t, e)
				// update the minimum order size to accomodate the small amount
				params.MinimumOrderSize = amount
				// set the params back in state
				require.NoError(t, sm.SetParamsVal(params))
			},
			msg: &MessageCreateOrder{
				ChainId:              lib.CanopyChainId,
				AmountForSale:        amount,
				RequestedAmount:      1000,
				SellerReceiveAddress: newTestPublicKeyBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyChainId + EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount, got)
				// ensure the order was created
				order, e := sm.GetOrder(0, lib.CanopyChainId)
				require.NoError(t, e)
				require.Equal(t, amount, order.AmountForSale)
			},
		},
		{
			name:   "message edit order",
			detail: "basic 'happy path' handling for message edit order",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t), amount))
				// get the validator params
				params, e := sm.GetParamsVal()
				require.NoError(t, e)
				// update the minimum order size to accomodate the small amount
				params.MinimumOrderSize = amount
				// set the params back in state
				require.NoError(t, sm.SetParamsVal(params))
				// pre-set an order to edit
				// add to the pool
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+EscrowPoolAddend, amount))
				// save the order in state
				_, err = sm.CreateOrder(&lib.SellOrder{
					Committee:            lib.CanopyChainId,
					AmountForSale:        amount,
					RequestedAmount:      1000,
					SellerReceiveAddress: newTestPublicKeyBytes(t),
					SellersSendAddress:   newTestAddressBytes(t),
				}, lib.CanopyChainId)
				require.NoError(t, err)
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        amount * 2,
				RequestedAmount:      2000,
				SellerReceiveAddress: newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyChainId + EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount*2, got)
				// ensure the order was edited
				order, e := sm.GetOrder(0, lib.CanopyChainId)
				require.NoError(t, e)
				require.Equal(t, amount*2, order.AmountForSale)
			},
		},
		{
			name:   "message delete order",
			detail: "basic 'happy path' handling for message delete order",
			preset: func(sm StateMachine) {
				// add to the pool
				require.NoError(t, sm.PoolAdd(lib.CanopyChainId+EscrowPoolAddend, amount))
				// save the order in state
				_, err = sm.CreateOrder(&lib.SellOrder{
					Committee:            lib.CanopyChainId,
					AmountForSale:        amount,
					RequestedAmount:      1000,
					SellerReceiveAddress: newTestPublicKeyBytes(t),
					SellersSendAddress:   newTestAddressBytes(t),
				}, lib.CanopyChainId)
				require.NoError(t, err)
			},
			msg: &MessageDeleteOrder{
				OrderId: 0,
				ChainId: lib.CanopyChainId,
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Equal(t, amount, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyChainId + EscrowPoolAddend)
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the order was deleted
				_, e = sm.GetOrder(0, lib.CanopyChainId)
				require.ErrorContains(t, e, "not found")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// run the preset function
			test.preset(sm)
			// execute the handler
			e := sm.HandleMessage(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", e != nil, e)
			if e != nil {
				require.ErrorContains(t, e, test.error)
				return
			}
			// run the validation
			test.validate(sm)
		})
	}
}

func TestGetFeeForMessage(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		msg    lib.MessageI
	}{
		{
			name:   "msg send",
			detail: "evaluates the function for message send",
			msg:    &MessageSend{},
		},
		{
			name:   "msg stake",
			detail: "evaluates the function for message stake",
			msg:    &MessageStake{},
		},
		{
			name:   "msg edit-stake",
			detail: "evaluates the function for message edit-stake",
			msg:    &MessageEditStake{},
		},
		{
			name:   "msg unstake",
			detail: "evaluates the function for message unstake",
			msg:    &MessageUnstake{},
		},
		{
			name:   "msg pause",
			detail: "evaluates the function for message pause",
			msg:    &MessagePause{},
		},
		{
			name:   "msg unpause",
			detail: "evaluates the function for message unpause",
			msg:    &MessageUnpause{},
		},
		{
			name:   "msg change param",
			detail: "evaluates the function for message change param",
			msg:    &MessageChangeParameter{},
		},
		{
			name:   "msg dao transfer",
			detail: "evaluates the function for message dao transfer",
			msg:    &MessageDAOTransfer{},
		},
		{
			name:   "msg certificate results",
			detail: "evaluates the function for message certificate results",
			msg:    &MessageCertificateResults{},
		},
		{
			name:   "msg subsidy",
			detail: "evaluates the function for message subsidy",
			msg:    &MessageSubsidy{},
		},
		{
			name:   "msg create order",
			detail: "evaluates the function for message create order",
			msg:    &MessageCreateOrder{},
		},
		{
			name:   "msg edit order",
			detail: "evaluates the function for message edit order",
			msg:    &MessageEditOrder{},
		},
		{
			name:   "msg delete order",
			detail: "evaluates the function for message delete order",
			msg:    &MessageDeleteOrder{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// get the fee params
			feeParams, err := sm.GetParamsFee()
			require.NoError(t, err)
			// define expected
			expected := func() uint64 {
				switch test.msg.(type) {
				case *MessageSend:
					return feeParams.SendFee
				case *MessageStake:
					return feeParams.StakeFee
				case *MessageEditStake:
					return feeParams.EditStakeFee
				case *MessageUnstake:
					return feeParams.UnstakeFee
				case *MessagePause:
					return feeParams.PauseFee
				case *MessageUnpause:
					return feeParams.UnpauseFee
				case *MessageChangeParameter:
					return feeParams.ChangeParameterFee
				case *MessageDAOTransfer:
					return feeParams.DaoTransferFee
				case *MessageCertificateResults:
					return feeParams.CertificateResultsFee
				case *MessageSubsidy:
					return feeParams.SubsidyFee
				case *MessageCreateOrder:
					return feeParams.CreateOrderFee
				case *MessageEditOrder:
					return feeParams.EditOrderFee
				case *MessageDeleteOrder:
					return feeParams.DeleteOrderFee
				default:
					panic("unknown msg")
				}
			}()
			// execute function call
			got, err := sm.GetFeeForMessageName(test.msg.Name())
			// validate the expected error
			require.NoError(t, err)
			// compare got vs expected
			require.Equal(t, expected, got)
		})
	}
}

func TestGetAuthorizedSignersFor(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		msg      lib.MessageI
		expected [][]byte
	}{
		{
			name:     "msg send",
			detail:   "retrieves the authorized signers for message send",
			msg:      &MessageSend{FromAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg stake",
			detail:   "retrieves the authorized signers for message stake",
			msg:      &MessageStake{PublicKey: newTestPublicKeyBytes(t), OutputAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t)},
		}, {
			name:     "msg edit-stake",
			detail:   "retrieves the authorized signers for message stake",
			msg:      &MessageEditStake{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg unstake",
			detail:   "retrieves the authorized signers for message unstake",
			msg:      &MessageUnstake{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg pause",
			detail:   "retrieves the authorized signers for message pause",
			msg:      &MessagePause{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg unpause",
			detail:   "retrieves the authorized signers for message unpause",
			msg:      &MessageUnpause{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg change param",
			detail:   "retrieves the authorized signers for message change param",
			msg:      &MessageChangeParameter{Signer: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg dao transfer",
			detail:   "retrieves the authorized signers for message dao transfer",
			msg:      &MessageDAOTransfer{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg subsidy",
			detail:   "retrieves the authorized signers for message subsidy",
			msg:      &MessageSubsidy{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg create order",
			detail:   "retrieves the authorized signers for message create order",
			msg:      &MessageCreateOrder{ChainId: lib.CanopyChainId, SellersSendAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg edit order",
			detail:   "retrieves the authorized signers for message edit order",
			msg:      &MessageEditOrder{ChainId: lib.CanopyChainId},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg delete order",
			detail:   "retrieves the authorized signers for message delete order",
			msg:      &MessageEditOrder{ChainId: lib.CanopyChainId},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:   "msg certificate results",
			detail: "retrieves the authorized signers for message delete order",
			msg: &MessageCertificateResults{
				Qc: &lib.QuorumCertificate{
					Header:      &lib.View{ChainId: lib.CanopyChainId, Height: 1},
					ProposerKey: newTestPublicKeyBytes(t),
				},
			},
			expected: [][]byte{newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set the state machine height at 1 for the 'time machine' call
			sm.height = 1
			// preset a validator
			require.NoError(t, sm.SetValidator(&Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Output:       newTestAddressBytes(t, 1),
			}))
			// preset an order
			_, err := sm.CreateOrder(&lib.SellOrder{
				Committee:          lib.CanopyChainId,
				SellersSendAddress: newTestAddressBytes(t),
			}, lib.CanopyChainId)
			require.NoError(t, err)
			// execute function call
			got, err := sm.GetAuthorizedSignersFor(test.msg)
			// validate the expected error
			require.NoError(t, err)
			// compare got vs expected
			require.Equal(t, test.expected, got)
		})
	}
}

func TestHandleMessageSend(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		presetSender   uint64
		presetReceiver uint64
		msg            *MessageSend
		error          string
	}{
		{
			name:           "insufficient amount",
			detail:         "the sender doesn't have enough tokens",
			presetSender:   1,
			presetReceiver: 0,
			msg: &MessageSend{
				FromAddress: newTestAddressBytes(t),
				ToAddress:   newTestAddressBytes(t, 1),
				Amount:      2,
			},
			error: "insufficient funds",
		},
		{
			name:           "send all",
			detail:         "the sender sends all of its tokens (1) to the recipient",
			presetSender:   1,
			presetReceiver: 0,
			msg: &MessageSend{
				FromAddress: newTestAddressBytes(t),
				ToAddress:   newTestAddressBytes(t, 1),
				Amount:      1,
			},
		},
		{
			name:           "send 1",
			detail:         "the sender sends one of its tokens to the recipient",
			presetSender:   2,
			presetReceiver: 0,
			msg: &MessageSend{
				FromAddress: newTestAddressBytes(t),
				ToAddress:   newTestAddressBytes(t, 1),
				Amount:      1,
			},
		},
		{
			name:           "add one",
			detail:         "the sender sends 1 of its tokens to the recipient, who adds it to their existing balance",
			presetSender:   2,
			presetReceiver: 1,
			msg: &MessageSend{
				FromAddress: newTestAddressBytes(t),
				ToAddress:   newTestAddressBytes(t, 1),
				Amount:      1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create sender addr object
			sender := crypto.NewAddress(test.msg.FromAddress)
			// create recipient addr object
			recipient := crypto.NewAddress(test.msg.ToAddress)
			// preset the accounts with some funds
			require.NoError(t, sm.AccountAdd(sender, test.presetSender))
			require.NoError(t, sm.AccountAdd(recipient, test.presetReceiver))
			// execute the function call
			err := sm.HandleMessageSend(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the send
			got, err := sm.GetAccount(sender)
			require.NoError(t, err)
			require.Equal(t, test.presetSender-test.msg.Amount, got.Amount)
			// validate the receipt
			got, err = sm.GetAccount(recipient)
			require.NoError(t, err)
			// compare got vs expected
			require.Equal(t, test.presetReceiver+test.msg.Amount, got.Amount)
		})
	}
}

func TestHandleMessageStake(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		presetSender    uint64
		presetValidator bool
		msg             *MessageStake
		expected        *Validator
		error           string
	}{
		{
			name:   "invalid public key",
			detail: "the sender public key is invalid",
			msg:    &MessageStake{PublicKey: newTestAddressBytes(t)},
			error:  "public key is invalid",
		},
		{
			name:            "invalid net address",
			detail:          "the validator net address is invalid",
			msg:             &MessageStake{PublicKey: newTestPublicKeyBytes(t)},
			expected:        &Validator{Address: newTestAddressBytes(t)},
			presetValidator: true,
			error:           "net address has invalid length",
		},
		{
			name:            "validator already exists",
			detail:          "the validator already exists in state",
			msg:             &MessageStake{PublicKey: newTestPublicKeyBytes(t), NetAddress: "tcp://example.com"},
			expected:        &Validator{Address: newTestAddressBytes(t), NetAddress: "tcp://example.com"},
			presetValidator: true,
			error:           "validator exists",
		},
		{
			name:         "insufficient amount",
			detail:       "the sender doesn't have enough tokens",
			presetSender: 0,
			msg: &MessageStake{
				PublicKey:  newTestPublicKeyBytes(t),
				NetAddress: "tcp://example.com",
				Amount:     1,
			},
			error: "insufficient funds",
		},
		{
			name:         "stake all funds as committee member",
			detail:       "the sender stakes all funds as committee member",
			presetSender: 1,
			msg: &MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      false,
				Compound:      true,
				Signer:        newTestAddressBytes(t),
			},
			expected: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Delegate:     false,
				Compound:     true,
			},
		},
		{
			name:         "stake partial funds as committee member",
			detail:       "the sender stakes partial funds as committee member",
			presetSender: 2,
			msg: &MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      false,
				Compound:      true,
				Signer:        newTestAddressBytes(t),
			},
			expected: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Delegate:     false,
				Compound:     true,
			},
		},
		{
			name:         "stake all funds as delegate",
			detail:       "the sender stakes all funds as delegate",
			presetSender: 1,
			msg: &MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      true,
				Compound:      true,
				Signer:        newTestAddressBytes(t),
			},
			expected: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Delegate:     true,
				Compound:     true,
			},
		},
		{
			name:         "stake partial funds as delegate",
			detail:       "the sender stakes partial funds as delegate",
			presetSender: 2,
			msg: &MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      true,
				Compound:      true,
				Signer:        newTestAddressBytes(t),
			},
			expected: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Delegate:     true,
				Compound:     true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var sender crypto.AddressI
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create sender pubkey object
			publicKey, err := crypto.NewPublicKeyFromBytes(test.msg.PublicKey)
			if err == nil {
				// create sender addr object
				sender = publicKey.Address()
				// preset the accounts with some funds
				require.NoError(t, sm.AccountAdd(sender, test.presetSender))
			}
			// preset the validator
			if test.presetValidator {
				require.NoError(t, sm.SetValidator(test.expected))
			}
			// execute the function call
			err = sm.HandleMessageStake(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the stake
			got, err := sm.GetAccount(sender)
			require.NoError(t, err)
			require.Equal(t, test.presetSender-test.msg.Amount, got.Amount)
			// validate the creation of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// compare got vs expected
			require.EqualExportedValues(t, test.expected, val)
			// get the supply object
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// validate the addition to the staked pool
			require.Equal(t, test.msg.Amount, supply.Staked)
			// validate the addition to the committees
			for _, id := range val.Committees {
				// get the supply for each committee
				stakedSupply, e := sm.GetCommitteeStakedSupplyForChain(id)
				require.NoError(t, e)
				require.Equal(t, test.msg.Amount, stakedSupply.Amount)
			}
			if val.Delegate {
				// validate the addition to the delegated pool
				require.Equal(t, test.msg.Amount, supply.DelegatedOnly)
				// validate the addition to the delegations only
				for _, id := range val.Committees {
					// get the supply for each committee
					stakedSupply, e := sm.GetDelegateStakedSupplyForChain(id)
					require.NoError(t, e)
					require.Equal(t, test.msg.Amount, stakedSupply.Amount)
					// validate the delegate membership
					page, e := sm.GetDelegatesPaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the list count is correct
					require.Len(t, *list, 1)
					// ensure the expected validator is a member
					require.EqualExportedValues(t, test.expected, (*list)[0])
				}
			} else {
				for _, id := range val.Committees {
					// validate the committee membership
					page, e := sm.GetCommitteePaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the list count is correct
					require.Len(t, *list, 1)
					// ensure the expected validator is a member
					require.EqualExportedValues(t, test.expected, (*list)[0])
				}
			}
		})
	}
}

func TestHandleMessageEditStake(t *testing.T) {
	// predefine a function that calculates the differences between two uint64 slices
	difference := func(a, b []uint64) []uint64 {
		y := make(map[uint64]struct{}, len(b))
		for _, x := range b {
			y[x] = struct{}{}
		}
		var diff []uint64
		for _, x := range a {
			if _, found := y[x]; !found {
				diff = append(diff, x)
			}
		}
		return diff
	}
	tests := []struct {
		name              string
		detail            string
		presetSender      uint64
		presetValidator   *Validator
		msg               *MessageEditStake
		expectedValidator *Validator
		expectedSupply    *Supply
		error             string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to edit it",
			msg:    &MessageEditStake{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		},
		{
			name:   "net address",
			detail: "the validator's net address has an invalid length",
			presetValidator: &Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &MessageEditStake{Address: newTestAddressBytes(t)},
			error: "net address has invalid length",
		},
		{
			name:   "unstaking",
			detail: "the validator is unstaking and cannot be edited",
			presetValidator: &Validator{
				Address:         newTestAddressBytes(t),
				NetAddress:      "tcp://example.com",
				UnstakingHeight: 1,
			},
			msg:   &MessageEditStake{Address: newTestAddressBytes(t), NetAddress: "tcp://example.com"},
			error: "unstaking",
		},
		{
			name:   "unauthorized output change",
			detail: "the sender is unable to change the output address",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        2,
				Committees:    []uint64{0, 1},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Compound:      true,
			},
			error: "unauthorized tx",
		},
		{
			name:   "insufficient funds",
			detail: "the sender doesn't have enough funds to complete the edit stake",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        2,
				Committees:    []uint64{0, 1},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t),
				Compound:      true,
			},
			error: "insufficient funds",
		},
		{
			name:   "edit stake, same balance, same committees",
			detail: "the validator is updated but the balance and committees remains the same",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "tcp://example2.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Compound:      false,
				Signer:        newTestAddressBytes(t),
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example2.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Compound:     false,
			},
			expectedSupply: &Supply{
				Total:  1,
				Staked: 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: 1,
					},
					{
						Id:     1,
						Amount: 1,
					},
				},
			},
		},
		{
			name:   "edit stake, same balance, same delegations",
			detail: "the validator is updated but the balance and delegations remains the same",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Delegate:     true,
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				OutputAddress: newTestAddressBytes(t),
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total:         1,
				Staked:        1,
				DelegatedOnly: 1,
				CommitteeStaked: []*Pool{
					{
						Id:     0,
						Amount: 1,
					},
					{
						Id:     1,
						Amount: 1,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     0,
						Amount: 1,
					},
					{
						Id:     1,
						Amount: 1,
					},
				},
			},
		},
		{
			name:   "edit stake, same balance, different committees",
			detail: "the validator is updated with different committees but the balance remains the same",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{1, 2, 3},
				NetAddress:    "tcp://example.com",
				OutputAddress: newTestAddressBytes(t),
				Compound:      true,
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "tcp://example.com",
				StakedAmount: 1,
				Committees:   []uint64{1, 2, 3},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			expectedSupply: &Supply{
				Total:  1,
				Staked: 1,
				CommitteeStaked: []*Pool{
					{
						Id:     1,
						Amount: 1,
					},
					{
						Id:     2,
						Amount: 1,
					},
					{
						Id:     3,
						Amount: 1,
					},
				},
			},
		},
		{
			name:   "edit stake, same balance, different delegations",
			detail: "the validator is updated with different delegations but the balance remains the same",
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			msg: &MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     1,
				Committees: []uint64{1, 2, 3},
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{1, 2, 3},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total:         1,
				Staked:        1,
				DelegatedOnly: 1,
				CommitteeStaked: []*Pool{
					{
						Id:     1,
						Amount: 1,
					},
					{
						Id:     2,
						Amount: 1,
					},
					{
						Id:     3,
						Amount: 1,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     1,
						Amount: 1,
					},
					{
						Id:     2,
						Amount: 1,
					},
					{
						Id:     3,
						Amount: 1,
					},
				},
			},
		},
		{
			name:         "edit stake, different balance, different committees",
			detail:       "the validator is updated with different committees and balance",
			presetSender: 2,
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				NetAddress:   "tcp://example.com",
				Committees:   []uint64{0, 1},
			},
			msg: &MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     2,
				NetAddress: "tcp://example.com",
				Committees: []uint64{1, 2, 3},
				Signer:     newTestAddressBytes(t),
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 2,
				NetAddress:   "tcp://example.com",
				Committees:   []uint64{1, 2, 3},
			},
			expectedSupply: &Supply{
				Total:  3,
				Staked: 2,
				CommitteeStaked: []*Pool{
					{
						Id:     1,
						Amount: 2,
					},
					{
						Id:     2,
						Amount: 2,
					},
					{
						Id:     3,
						Amount: 2,
					},
				},
			},
		},
		{
			name:         "edit stake, different balance, different delegations",
			detail:       "the validator is updated with different delegations and balance",
			presetSender: 2,
			presetValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			msg: &MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     2,
				Committees: []uint64{1, 2, 3},
				Signer:     newTestAddressBytes(t),
			},
			expectedValidator: &Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 2,
				Committees:   []uint64{1, 2, 3},
				Delegate:     true,
			},
			expectedSupply: &Supply{
				Total:         3,
				Staked:        2,
				DelegatedOnly: 2,
				CommitteeStaked: []*Pool{
					{
						Id:     1,
						Amount: 2,
					},
					{
						Id:     2,
						Amount: 2,
					},
					{
						Id:     3,
						Amount: 2,
					},
				},
				CommitteeDelegatedOnly: []*Pool{
					{
						Id:     1,
						Amount: 2,
					},
					{
						Id:     2,
						Amount: 2,
					},
					{
						Id:     3,
						Amount: 2,
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var sender crypto.AddressI
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create sender address object
			sender = crypto.NewAddress(test.msg.Address)
			// preset the accounts with some funds
			require.NoError(t, sm.AccountAdd(sender, test.presetSender))
			// preset the validator
			if test.presetValidator != nil {
				supply := &Supply{}
				require.NoError(t, sm.SetValidators([]*Validator{test.presetValidator}, supply))
				supply.Total = test.presetSender + test.presetValidator.StakedAmount
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			err := sm.HandleMessageEditStake(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the account
			got, err := sm.GetAccount(sender)
			require.NoError(t, err)
			require.Equal(t, test.presetSender-(test.msg.Amount-test.presetValidator.StakedAmount), got.Amount)
			// validate the update of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// compare got vs expected
			require.EqualExportedValues(t, test.expectedValidator, val)
			// get the supply object
			supply, err := sm.GetSupply()
			require.NoError(t, err)
			// validate the update to the supply
			require.EqualExportedValues(t, test.expectedSupply, supply)
			// calculate differences between before and after committees
			nonMembershipCommittees := difference(test.presetValidator.Committees, val.Committees)
			if val.Delegate {
				for _, id := range val.Committees {
					// validate the delegate membership
					page, e := sm.GetDelegatesPaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the list count is correct
					require.Len(t, *list, 1)
					// ensure the expected validator is a member
					require.EqualExportedValues(t, test.expectedValidator, (*list)[0])
				}
				for _, id := range nonMembershipCommittees {
					// validate the delegate non-membership
					page, e := sm.GetDelegatesPaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the non membership
					require.Len(t, *list, 0)
				}
			} else {
				for _, id := range val.Committees {
					// validate the committee membership
					page, e := sm.GetCommitteePaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the list count is correct
					require.Len(t, *list, 1)
					// ensure the expected validator is a member
					require.EqualExportedValues(t, test.expectedValidator, (*list)[0])
				}
				for _, id := range nonMembershipCommittees {
					// validate the committee non-membership
					page, e := sm.GetCommitteePaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*ValidatorPage)
					// ensure the non membership
					require.Len(t, *list, 0)
				}
			}
		})
	}
}

func TestMessageUnstake(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
		msg    *MessageUnstake
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to unstake it",
			msg:    &MessageUnstake{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator already unstaking",
			detail: "validator is already unstaking so this operation is invalid",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &MessageUnstake{Address: newTestAddressBytes(t)},
			error: "validator is unstaking",
		},
		{
			name:   "validator not delegate",
			detail: "validator is not a delegate",
			preset: &Validator{
				Address: newTestAddressBytes(t),
			},
			msg: &MessageUnstake{Address: newTestAddressBytes(t)},
		},
		{
			name:   "validator a delegate",
			detail: "validator is a delegate",
			preset: &Validator{
				Address:  newTestAddressBytes(t),
				Delegate: true,
			},
			msg: &MessageUnstake{Address: newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create the sender address object
			sender := crypto.NewAddress(test.msg.Address)
			// preset the validator
			if test.preset != nil {
				supply := &Supply{}
				require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			err := sm.HandleMessageUnstake(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the unstaking of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// calculate the finish unstaking height
			var unstakingBlocks uint64
			if val.Delegate {
				unstakingBlocks = valParams.DelegateUnstakingBlocks
			} else {
				unstakingBlocks = valParams.UnstakingBlocks
			}
			finishUnstakingHeight := unstakingBlocks + sm.Height()
			// compare got vs expected
			require.Equal(t, finishUnstakingHeight, val.UnstakingHeight)
			// ensure unstaking key exists
			list, e := sm.GetAddressList(UnstakingPrefix(finishUnstakingHeight))
			require.NoError(t, e)
			require.Contains(t, list.Addresses, sender.Bytes())
		})
	}
}

func TestMessagePause(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
		msg    *MessagePause
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to pause it",
			msg:    &MessagePause{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator already paused",
			detail: "validator is already paused so this operation is invalid",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				MaxPausedHeight: 1,
			},
			msg:   &MessagePause{Address: newTestAddressBytes(t)},
			error: "validator paused",
		},
		{
			name:   "validator unstaking",
			detail: "validator is unstaking so this operation is invalid",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &MessagePause{Address: newTestAddressBytes(t)},
			error: "validator is unstaking",
		},
		{
			name:   "validator is a delegate",
			detail: "validator is a delegate",
			preset: &Validator{
				Address:  newTestAddressBytes(t),
				Delegate: true,
			},
			msg:   &MessagePause{Address: newTestAddressBytes(t)},
			error: "validator is a delegate",
		},
		{
			name:   "validator is not a delegate",
			detail: "validator is not a delegate",
			preset: &Validator{
				Address: newTestAddressBytes(t),
			},
			msg: &MessagePause{Address: newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create the sender address object
			sender := crypto.NewAddress(test.msg.Address)
			// preset the validator
			if test.preset != nil {
				supply := &Supply{}
				require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
				require.NoError(t, sm.SetSupply(supply))
			}
			// execute the function call
			err := sm.HandleMessagePause(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the pausing of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// calculate the finish unstaking height
			maxPauseBlocks := valParams.MaxPauseBlocks + sm.Height()
			// compare got vs expected
			require.Equal(t, maxPauseBlocks, val.MaxPausedHeight)
			// ensure paused key exists
			list, e := sm.GetAddressList(PausedPrefix(maxPauseBlocks))
			require.NoError(t, e)
			require.Contains(t, list.Addresses, sender.Bytes())
		})
	}
}

func TestMessageUnpause(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *Validator
		msg    *MessageUnpause
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to unpause it",
			msg:    &MessageUnpause{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator not paused",
			detail: "validator is not paused so this operation is invalid",
			preset: &Validator{
				Address: newTestAddressBytes(t),
			},
			msg:   &MessageUnpause{Address: newTestAddressBytes(t)},
			error: "validator not paused",
		},
		{
			name:   "validator is paused",
			detail: "validator is paused",
			preset: &Validator{
				Address:         newTestAddressBytes(t),
				MaxPausedHeight: 1,
			},
			msg: &MessageUnpause{Address: newTestAddressBytes(t)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// create the sender address object
			sender := crypto.NewAddress(test.msg.Address)
			// preset the validator
			if test.preset != nil {
				supply := &Supply{}
				require.NoError(t, sm.SetValidators([]*Validator{test.preset}, supply))
				require.NoError(t, sm.SetSupply(supply))
				// preset the validator as paused
				require.NoError(t, sm.SetValidatorPaused(sender, test.preset, test.preset.MaxPausedHeight))
			}
			// execute the function call
			err := sm.HandleMessageUnpause(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// validate the unstaking of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// compare got vs expected
			require.EqualValues(t, 0, val.MaxPausedHeight)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// check for the paused key
			list, e := sm.GetAddressList(PausedPrefix(valParams.MaxPauseBlocks + sm.Height()))
			require.NoError(t, e)
			require.NotContains(t, list.Addresses, sender.Bytes())
		})
	}
}

func TestHandleMessageChangeParameter(t *testing.T) {
	uint64Any, _ := lib.NewAny(&lib.UInt64Wrapper{Value: 100})
	stringAny, _ := lib.NewAny(&lib.StringWrapper{Value: "2/2"})
	tests := []struct {
		name           string
		detail         string
		proposalConfig GovProposalVoteConfig
		height         uint64
		msg            *MessageChangeParameter
		error          string
	}{
		{
			name:           "before start height",
			detail:         "the start height is greater than state machine height",
			height:         1,
			proposalConfig: AcceptAllProposals,
			msg: &MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   ParamUnstakingBlocks,
				ParameterValue: uint64Any,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
			error: "proposal rejected",
		},
		{
			name:           "after end height",
			detail:         "the end height is less than state machine height",
			height:         4,
			proposalConfig: AcceptAllProposals,
			msg: &MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   ParamUnstakingBlocks,
				ParameterValue: uint64Any,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
			error: "proposal rejected",
		},
		{
			name:           "reject all config",
			detail:         "configuration is set to reject all",
			proposalConfig: RejectAllProposals,
			height:         2,
			msg: &MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   ParamUnstakingBlocks,
				ParameterValue: uint64Any,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
			error: "proposal rejected",
		},
		{
			name:           "change unstaking blocks",
			detail:         "successfully change unstaking blocks with the message",
			proposalConfig: AcceptAllProposals,
			height:         2,
			msg: &MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   ParamUnstakingBlocks,
				ParameterValue: uint64Any,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
		},
		{
			name:           "change protocol version",
			detail:         "successfully the protocol version with the message",
			proposalConfig: AcceptAllProposals,
			height:         2,
			msg: &MessageChangeParameter{
				ParameterSpace: "cons",
				ParameterKey:   ParamProtocolVersion,
				ParameterValue: stringAny,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set state machine height
			sm.height = test.height
			// set state machine proposal configuration
			sm.proposeVoteConfig = test.proposalConfig
			// extract the value from the object
			var (
				uint64Value *lib.UInt64Wrapper
				stringValue *lib.StringWrapper
			)
			// extract the value from any
			value, err := lib.FromAny(test.msg.ParameterValue)
			require.NoError(t, err)
			if i, isUint64 := value.(*lib.UInt64Wrapper); isUint64 {
				uint64Value = i
			} else if s, isString := value.(*lib.StringWrapper); isString {
				stringValue = s
			}
			// execute the function call
			err = sm.HandleMessageChangeParameter(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get params object from state
			got, err := sm.GetParams()
			require.NoError(t, err)
			// validate the update
			switch test.msg.ParameterKey {
			case ParamUnstakingBlocks: // validator
				require.Equal(t, uint64Value.Value, got.Validator.UnstakingBlocks)
			case ParamProtocolVersion: // consensus
				require.Equal(t, stringValue.Value, got.Consensus.ProtocolVersion)
			}
		})
	}
}

func TestHandleMessageDAOTransfer(t *testing.T) {
	tests := []struct {
		name           string
		detail         string
		daoPreset      uint64
		proposalConfig GovProposalVoteConfig
		height         uint64
		msg            *MessageDAOTransfer
		error          string
	}{
		{
			name:           "before start height",
			detail:         "the start height is greater than state machine height",
			height:         1,
			daoPreset:      1,
			proposalConfig: AcceptAllProposals,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      1,
				StartHeight: 2,
				EndHeight:   3,
			},
			error: "proposal rejected",
		},
		{
			name:           "after end height",
			detail:         "the end height is less than state machine height",
			height:         4,
			proposalConfig: AcceptAllProposals,
			daoPreset:      1,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      1,
				StartHeight: 2,
				EndHeight:   3,
			},
			error: "proposal rejected",
		},
		{
			name:           "reject all config",
			detail:         "configuration is set to reject all",
			proposalConfig: RejectAllProposals,
			height:         2,
			daoPreset:      1,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      1,
				StartHeight: 2,
				EndHeight:   3,
			},
			error: "proposal rejected",
		},
		{
			name:           "insufficient funds",
			detail:         "dao doesn't have the funds",
			proposalConfig: AcceptAllProposals,
			height:         2,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      1,
				StartHeight: 2,
				EndHeight:   3,
			},
			error: "insufficient funds",
		},
		{
			name:           "successful transfer",
			detail:         "a successful dao transfer was completed with the message",
			proposalConfig: AcceptAllProposals,
			daoPreset:      1,
			height:         2,
			msg: &MessageDAOTransfer{
				Address:     newTestAddressBytes(t),
				Amount:      1,
				StartHeight: 2,
				EndHeight:   3,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// set state machine height
			sm.height = test.height
			// set state machine proposal configuration
			sm.proposeVoteConfig = test.proposalConfig
			// preset the dao amount
			require.NoError(t, sm.PoolAdd(lib.DAOPoolID, test.daoPreset))
			// execute the function call
			err := sm.HandleMessageDAOTransfer(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the dao pool
			got, err := sm.GetPoolBalance(lib.DAOPoolID)
			require.NoError(t, err)
			// validate the transfer
			require.Equal(t, test.daoPreset-test.msg.Amount, got)
			// get the recipient account
			got, err = sm.GetAccountBalance(crypto.NewAddress(test.msg.Address))
			require.Equal(t, test.msg.Amount, got)
		})
	}
}

func TestHandleMessageCertificateResults(t *testing.T) {
	// pre-define a quorum certificate to insert into the message change certificate results
	certificateResults := &lib.CertificateResult{
		RewardRecipients: &lib.RewardRecipients{
			PaymentPercents: []*lib.PaymentPercents{{Address: newTestAddressBytes(t), Percent: 100, ChainId: 1}},
		},
		SlashRecipients: &lib.SlashRecipients{
			DoubleSigners: []*lib.DoubleSigner{
				{Id: newTestPublicKeyBytes(t), Heights: []uint64{1}},
			},
		},
		Orders: &lib.Orders{
			LockOrders: []*lib.LockOrder{{
				OrderId:             0,
				BuyerSendAddress:    newTestAddressBytes(t),
				BuyerReceiveAddress: newTestAddressBytes(t),
				BuyerChainDeadline:  100,
			}},
			ResetOrders: []uint64{1},
			CloseOrders: []uint64{2},
		},
		Checkpoint: &lib.Checkpoint{Height: 1, BlockHash: crypto.Hash([]byte("block_hash"))},
	}
	tests := []struct {
		name                   string
		detail                 string
		nonSubsidizedCommittee bool
		noCommitteeMembers     bool
		msg                    *MessageCertificateResults
		error                  string
	}{
		{
			name:                   "canopy committee",
			detail:                 "the canopy committee tries to send a certificate results transaction",
			nonSubsidizedCommittee: true,
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					RootHeight: 3,
					ChainId:    lib.CanopyChainId,
				},
			}},
			error: "invalid certificate results",
		},
		{
			name:                   "non subsidized committee",
			detail:                 "the committee is not subsidized",
			nonSubsidizedCommittee: true,
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					RootHeight: 3,
					ChainId:    lib.CanopyChainId + 1,
				},
			}},
			error: "non subsidized committee",
		},
		{
			name:               "no committee members exist for that id",
			detail:             "there are no committee members for that ID",
			noCommitteeMembers: true,
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					RootHeight: 3,
					ChainId:    lib.CanopyChainId + 1,
				},
			}},
			error: "there are no validators in the set",
		},
		{
			name:               "no committee members exist for that id",
			detail:             "there are no committee members for that ID",
			noCommitteeMembers: true,
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					RootHeight: 3,
					ChainId:    lib.CanopyChainId + 1,
				},
			}},
			error: "there are no validators in the set",
		},
		{
			name:   "empty quorum certificate",
			detail: "the QC is empty",
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					RootHeight: 2,
					ChainId:    lib.CanopyChainId + 1,
				},
			}},
			error: "empty quorum certificate",
		},
		{
			name:   "valid qc",
			detail: "the qc sent is valid",
			msg: &MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:     1,
					NetworkId:  1,
					RootHeight: 3,
					ChainId:    lib.CanopyChainId + 1,
				},
				Results:     certificateResults,
				ResultsHash: certificateResults.Hash(),
				BlockHash:   crypto.Hash([]byte("some_block")),
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// check if the pool is subsidized
			if !test.nonSubsidizedCommittee {
				// subsidize the committee
				require.NoError(t, sm.PoolAdd(test.msg.Qc.Header.ChainId, 1))
			}
			// check if there exists a committee
			if !test.noCommitteeMembers {
				// track the supply
				supply := &Supply{}
				// for 4 validators
				for i := 0; i < 4; i++ {
					// set the validator
					require.NoError(t, sm.SetValidators([]*Validator{{
						Address:      newTestAddressBytes(t, i),
						PublicKey:    newTestPublicKeyBytes(t, i),
						StakedAmount: 100,
						Committees:   []uint64{lib.CanopyChainId + 1},
					}}, supply))
				}
				// set the supply in state
				require.NoError(t, sm.SetSupply(supply))
				// create an aggregate signature
				// get the committee members
				committee, err := sm.GetCommitteeMembers(lib.CanopyChainId + 1)
				require.NoError(t, err)
				// create a copy of the multikey
				mk := committee.MultiKey.Copy()
				// only sign with 3/4 to test the non-signer reduction
				for i := 0; i < 3; i++ {
					privateKey := newTestKeyGroup(t, i).PrivateKey
					// search for the proper index for the signer
					for j, pubKey := range mk.PublicKeys() {
						// if found, add the signer
						if privateKey.PublicKey().Equals(pubKey) {
							// sign the qc
							require.NoError(t, mk.AddSigner(privateKey.Sign(test.msg.Qc.SignBytes()), j))
						}
					}
				}
				// aggregate the signature
				aggSig, e := mk.AggregateSignatures()
				require.NoError(t, e)
				// attach the signature to the message
				test.msg.Qc.Signature = &lib.AggregateSignature{
					Signature: aggSig,
					Bitmap:    mk.Bitmap(),
				}
			}
			// commit to lock in the validator set for the previous height
			// this is required because LoadCommittee() doesn't read from the current
			// database 'txn' rather the underlying db (which does not yet have the validator set)
			_, err := sm.Store().(lib.StoreI).Commit()
			require.NoError(t, err)
			// increment height as height 2 ignores byzantine evidence
			sm.height++
			// preset some sell orders to test with
			for i := 0; i < 3; i++ {
				var buyerAddress []byte
				// set order #1, #2 with a buyer for 'reset' and 'close' functionality
				if i != 0 {
					buyerAddress = newTestAddressBytes(t)
				}
				// upsert each order in state
				_, err = sm.CreateOrder(&lib.SellOrder{
					Committee:           lib.CanopyChainId + 1,
					BuyerReceiveAddress: buyerAddress,
					BuyerChainDeadline:  0,
					SellersSendAddress:  newTestAddressBytes(t),
				}, lib.CanopyChainId+1)
				// ensure no error
				require.NoError(t, err)
			}
			// execute function call
			err = sm.HandleMessageCertificateResults(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// 1) validate the 'lock order'
			func() {
				order, e := sm.GetOrder(0, lib.CanopyChainId+1)
				require.NoError(t, e)
				// convenience variable for lock order
				lockOrder := test.msg.Qc.Results.Orders.LockOrders[0]
				// validate the receipt address was set
				require.Equal(t, lockOrder.BuyerReceiveAddress, order.BuyerReceiveAddress)
				// validate the deadline was set
				require.Equal(t, lockOrder.BuyerChainDeadline, order.BuyerChainDeadline)
			}()
			// 2) validate the 'reset order'
			func() {
				order, e := sm.GetOrder(1, lib.CanopyChainId+1)
				require.NoError(t, e)
				// validate the receipt address was reset
				require.Len(t, order.BuyerReceiveAddress, 0)
				// validate the deadline was reset
				require.Zero(t, order.BuyerChainDeadline)
			}()

			// 3) validate the 'close order'
			func() {
				_, err = sm.GetOrder(2, lib.CanopyChainId+1)
				require.ErrorContains(t, err, "order with id 2 not found")
			}()

			// 4) validate the 'checkpoint' service
			func() {
				// define convenience variable for checkpoint
				expected := test.msg.Qc.Results.Checkpoint
				// get the checkpoint
				got, e := sm.store.(lib.StoreI).GetCheckpoint(lib.CanopyChainId+1, expected.Height)
				require.NoError(t, e)
				// check got vs expected
				require.Equal(t, lib.HexBytes(expected.BlockHash), got)
			}()

			// 5) validate the 'committee data'
			func() {
				committeeData, e := sm.GetCommitteeData(lib.CanopyChainId + 1)
				require.NoError(t, e)
				// validate the committee height was properly set
				require.Equal(t, test.msg.Qc.Header.RootHeight, committeeData.LastRootHeightUpdated)
				// validate the chain height was properly set
				require.Equal(t, test.msg.Qc.Header.Height, committeeData.LastChainHeightUpdated)
				// validate the number of samples was properly set
				require.EqualValues(t, 1, committeeData.NumberOfSamples)
				// validate the payment percent was set
				require.Len(t, committeeData.PaymentPercents, 1)
				// convenience variable for payment percent validation
				expected := test.msg.Qc.Results.RewardRecipients.PaymentPercents[0]
				// validate the payment percent WITH the non-signer reduction applied
				require.Equal(t, expected.Percent, committeeData.PaymentPercents[0].Percent)
			}()
		})
	}
}

func TestMessageSubsidy(t *testing.T) {
	tests := []struct {
		name          string
		detail        string
		presetAccount uint64
		presetPool    uint64
		msg           *MessageSubsidy
		error         string
	}{
		{
			name:          "insufficient funds",
			detail:        "the account does not have enough funds to complete the transfer",
			presetAccount: 1,
			msg: &MessageSubsidy{
				Address: newTestAddressBytes(t),
				ChainId: lib.CanopyChainId,
				Amount:  2,
			},
			error: "insufficient funds",
		},
		{
			name:          "successful transfer",
			detail:        "the transfer is successful",
			presetAccount: 1,
			msg: &MessageSubsidy{
				Address: newTestAddressBytes(t),
				ChainId: lib.CanopyChainId,
				Amount:  1,
			},
		},
		{
			name:          "successful transfer pre-balance",
			detail:        "the transfer is successful with pool having a non-zero balance to start",
			presetAccount: 1,
			presetPool:    2,
			msg: &MessageSubsidy{
				Address: newTestAddressBytes(t),
				ChainId: lib.CanopyChainId,
				Amount:  1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// define an address variable for convenience
			address := crypto.NewAddress(test.msg.Address)
			// preset the account with tokens
			require.NoError(t, sm.AccountAdd(address, test.presetAccount))
			// preset the pool with tokens
			require.NoError(t, sm.PoolAdd(test.msg.ChainId, test.presetPool))
			// execute the function
			err := sm.HandleMessageSubsidy(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the account balance
			got, err := sm.GetAccountBalance(address)
			require.NoError(t, err)
			// validate the subtraction from the account
			require.Equal(t, test.presetAccount-test.msg.Amount, got)
			// get the pool balance
			got, err = sm.GetPoolBalance(test.msg.ChainId)
			require.NoError(t, err)
			// validate the addition to the pool
			require.Equal(t, test.presetPool+test.msg.Amount, got)
		})
	}
}

func TestMessageCreateOrder(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		presetAccount    uint64
		minimumOrderSize uint64
		msg              *MessageCreateOrder
		error            string
	}{
		{
			name:             "below minimum",
			detail:           "the order does not satisfy the minimum order size",
			presetAccount:    1,
			minimumOrderSize: 2,
			msg: &MessageCreateOrder{
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			error: "minimum order size",
		},
		{
			name:             "insufficient funds",
			detail:           "the account does not have sufficient funds to cover the sell order",
			minimumOrderSize: 1,
			msg: &MessageCreateOrder{
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			error: "insufficient funds",
		},
		{
			name:             "valid sell order",
			detail:           "the message creates a sell order in state",
			presetAccount:    1,
			minimumOrderSize: 1,
			msg: &MessageCreateOrder{
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// define an address variable for convenience
			address := crypto.NewAddress(test.msg.SellersSendAddress)
			// preset the minimum order size
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// set minimum order size
			valParams.MinimumOrderSize = test.minimumOrderSize
			// set back in state
			require.NoError(t, sm.SetParamsVal(valParams))
			// preset the account with tokens
			require.NoError(t, sm.AccountAdd(address, test.presetAccount))
			// execute the function
			err = sm.HandleMessageCreateOrder(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the account balance
			got, err := sm.GetAccountBalance(address)
			require.NoError(t, err)
			// validate the subtraction from the account
			require.Equal(t, test.presetAccount-test.msg.AmountForSale, got)
			// get the pool balance
			got, err = sm.GetPoolBalance(test.msg.ChainId + EscrowPoolAddend)
			require.NoError(t, err)
			// validate the addition to the pool
			require.Equal(t, test.msg.AmountForSale, got)
			// get the order in state
			order, err := sm.GetOrder(0, test.msg.ChainId)
			require.NoError(t, err)
			// validate the creation of the order
			require.EqualExportedValues(t, &lib.SellOrder{
				Committee:            test.msg.ChainId,
				AmountForSale:        test.msg.AmountForSale,
				RequestedAmount:      test.msg.RequestedAmount,
				SellerReceiveAddress: test.msg.SellerReceiveAddress,
				SellersSendAddress:   test.msg.SellersSendAddress,
			}, order)
		})
	}
}

func TestHandleMessageEditOrder(t *testing.T) {
	tests := []struct {
		name             string
		detail           string
		presetAccount    uint64
		minimumOrderSize uint64
		preset           *lib.SellOrder
		msg              *MessageEditOrder
		expected         *lib.SellOrder
		error            string
	}{
		{
			name:   "no order found",
			detail: "there exists no order",
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "not found",
		},
		{
			name:   "order locked",
			detail: "a buyer has already accepted the order, thus it cannot be edited",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				BuyerReceiveAddress:  newTestAddressBytes(t, 1), // signals a buyer
				BuyerChainDeadline:   100,                       // signals a buyer
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "order locked",
		},
		{
			name:             "minimum order size",
			detail:           "the edited order does not satisfy the minimum order size",
			minimumOrderSize: 2,
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "minimum order size",
		}, {
			name:   "insufficient funds",
			detail: "the account does not have the balance to cover the edit",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "insufficient funds",
		},
		{
			name:   "edit receive address",
			detail: "the order simply updates the receive address but the amount stays the same",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSendAddress:   newTestAddressBytes(t),
			},
		},
		{
			name:             "increase sell amount",
			detail:           "the order has a increased the sell amount",
			presetAccount:    1,
			minimumOrderSize: 0,
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        2,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSendAddress:   newTestAddressBytes(t),
			},
		},
		{
			name:   "decrease sell amount",
			detail: "the order has a decreased the sell amount",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageEditOrder{
				OrderId:              0,
				ChainId:              lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSendAddress:   newTestAddressBytes(t),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var address crypto.AddressI
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// define an address variable for convenience
			if test.preset != nil {
				address = crypto.NewAddress(test.preset.SellersSendAddress)
				// preset the minimum order size
				valParams, err := sm.GetParamsVal()
				require.NoError(t, err)
				// set minimum order size
				valParams.MinimumOrderSize = test.minimumOrderSize
				// set back in state
				require.NoError(t, sm.SetParamsVal(valParams))
				// preset the account with tokens
				require.NoError(t, sm.AccountAdd(address, test.presetAccount))
				// get the proper order book
				orderBook, err := sm.GetOrderBook(lib.CanopyChainId)
				require.NoError(t, err)
				// preset the sell order
				_ = orderBook.AddOrder(test.preset)
				// set it back in state
				require.NoError(t, sm.SetOrderBook(orderBook))
				// preset the pool with the amount to sell
				require.NoError(t, sm.PoolAdd(test.preset.Committee+EscrowPoolAddend, test.preset.AmountForSale))
			}
			// execute the function
			err := sm.HandleMessageEditOrder(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the account balance
			got, err := sm.GetAccountBalance(address)
			require.NoError(t, err)
			// validate the subtraction/addition to/from the account
			require.Equal(t, test.presetAccount-(test.msg.AmountForSale-test.preset.AmountForSale), got)
			// get the pool balance
			got, err = sm.GetPoolBalance(test.msg.ChainId + EscrowPoolAddend)
			require.NoError(t, err)
			// validate the subtraction/addition to/from the pool
			require.Equal(t, test.preset.AmountForSale-(test.preset.AmountForSale-test.msg.AmountForSale), got)
			// get the order in state
			order, err := sm.GetOrder(0, test.msg.ChainId)
			require.NoError(t, err)
			// validate the creation of the order
			require.EqualExportedValues(t, test.expected, order)
		})
	}
}

func TestHandleMessageDelete(t *testing.T) {
	tests := []struct {
		name          string
		detail        string
		presetAccount uint64
		preset        *lib.SellOrder
		msg           *MessageDeleteOrder
		error         string
	}{
		{
			name:   "no order found",
			detail: "there exists no order",
			msg: &MessageDeleteOrder{
				OrderId: 0,
				ChainId: lib.CanopyChainId,
			},
			error: "not found",
		},
		{
			name:   "order locked",
			detail: "a buyer has already accepted the order, thus it cannot be edited",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				BuyerReceiveAddress:  newTestAddressBytes(t, 1), // signals a buyer
				BuyerChainDeadline:   100,                       // signals a buyer
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageDeleteOrder{
				OrderId: 0,
				ChainId: lib.CanopyChainId,
			},
			error: "order locked",
		},
		{
			name:   "successful delete",
			detail: "the order delete was successful",
			preset: &lib.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyChainId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSendAddress:   newTestAddressBytes(t),
			},
			msg: &MessageDeleteOrder{
				OrderId: 0,
				ChainId: lib.CanopyChainId,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var address crypto.AddressI
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// define an address variable for convenience
			if test.preset != nil {
				address = crypto.NewAddress(test.preset.SellersSendAddress)
				// preset the account with tokens
				require.NoError(t, sm.AccountAdd(address, test.presetAccount))
				// get the proper order book
				orderBook, err := sm.GetOrderBook(lib.CanopyChainId)
				require.NoError(t, err)
				// preset the sell order
				_ = orderBook.AddOrder(test.preset)
				// set it back in state
				require.NoError(t, sm.SetOrderBook(orderBook))
				// preset the pool with the amount to sell
				require.NoError(t, sm.PoolAdd(test.preset.Committee+EscrowPoolAddend, test.preset.AmountForSale))
			}
			// execute the function
			err := sm.HandleMessageDeleteOrder(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// get the account balance
			got, err := sm.GetAccountBalance(address)
			require.NoError(t, err)
			// validate the addition to the account
			require.Equal(t, test.presetAccount+test.preset.AmountForSale, got)
			// get the pool balance
			got, err = sm.GetPoolBalance(test.msg.ChainId + EscrowPoolAddend)
			require.NoError(t, err)
			// validate the subtraction from the pool
			require.Zero(t, got)
			// validate the delete
			_, err = sm.GetOrder(0, test.msg.ChainId)
			require.ErrorContains(t, err, "not found")
		})
	}
}
