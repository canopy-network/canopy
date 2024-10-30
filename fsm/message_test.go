package fsm

import (
	"github.com/ginchuco/canopy/fsm/types"
	"github.com/ginchuco/canopy/lib"
	"github.com/ginchuco/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHandleMessage(t *testing.T) {
	const amount = uint64(100)
	// pre-create a 'change parameter' proposal to use during testing
	a, err := lib.NewAny(&lib.StringWrapper{Value: types.NewProtocolVersion(3, 2)})
	require.NoError(t, err)
	msgChangeParam := &types.MessageChangeParameter{
		ParameterSpace: "cons",
		ParameterKey:   types.ParamProtocolVersion,
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
			msg: &types.MessageSend{
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
			msg: &types.MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        amount,
				Committees:    []uint64{lib.CanopyCommitteeId},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the sender account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the validator was created
				exists, e := sm.GetValidatorExists(newTestAddress(t))
				require.NoError(t, e)
				require.True(t, exists)
			},
		},
		{
			name:   "message edit-stake",
			detail: "basic 'happy path' handling for message edit-stake",
			preset: func(sm StateMachine) {
				// set account balance
				require.NoError(t, sm.AccountAdd(newTestAddress(t), 1))
				// create validator
				v := &types.Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				}
				// add the validator stake to total supply
				require.NoError(t, sm.AddToTotalSupply(v.StakedAmount))
				// add the validator stake to supply
				require.NoError(t, sm.AddToStakedSupply(v.StakedAmount))
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
				// set validator committees
				require.NoError(t, sm.SetCommittees(crypto.NewAddress(v.Address), v.StakedAmount, v.Committees))
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        amount + 1,
				Committees:    []uint64{lib.CanopyCommitteeId},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t),
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
				v := &types.Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &types.MessageUnstake{Address: newTestAddressBytes(t)},
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
				v := &types.Validator{
					Address:      newTestAddressBytes(t),
					StakedAmount: amount,
					Committees:   []uint64{lib.CanopyCommitteeId},
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &types.MessagePause{Address: newTestAddressBytes(t)},
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
				v := &types.Validator{
					Address:         newTestAddressBytes(t),
					StakedAmount:    amount,
					Committees:      []uint64{lib.CanopyCommitteeId},
					MaxPausedHeight: 1,
				}
				// set the validator in state
				require.NoError(t, sm.SetValidator(v))
			},
			msg: &types.MessageUnpause{Address: newTestAddressBytes(t)},
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
				require.Equal(t, types.NewProtocolVersion(3, 2), consParams.ProtocolVersion)
			},
		},
		{
			name:   "message dao transfer",
			detail: "basic 'happy path' handling for message dao transfer",
			preset: func(sm StateMachine) {
				require.NoError(t, sm.PoolAdd(lib.DAOPoolID, amount))
			},
			msg: &types.MessageDAOTransfer{
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
			msg: &types.MessageSubsidy{
				Address:     newTestAddressBytes(t),
				CommitteeId: lib.CanopyCommitteeId,
				Amount:      amount,
				Opcode:      "note",
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyCommitteeId)
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
				params.ValidatorMinimumOrderSize = amount
				// set the params back in state
				require.NoError(t, sm.SetParamsVal(params))
			},
			msg: &types.MessageCreateOrder{
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        amount,
				RequestedAmount:      1000,
				SellerReceiveAddress: newTestPublicKeyBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyCommitteeId + types.EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount, got)
				// ensure the order was created
				order, e := sm.GetOrder(0, lib.CanopyCommitteeId)
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
				params.ValidatorMinimumOrderSize = amount
				// set the params back in state
				require.NoError(t, sm.SetParamsVal(params))
				// pre-set an order to edit
				// add to the pool
				require.NoError(t, sm.PoolAdd(lib.CanopyCommitteeId+types.EscrowPoolAddend, amount))
				// save the order in state
				_, err = sm.CreateOrder(&types.SellOrder{
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        amount,
					RequestedAmount:      1000,
					SellerReceiveAddress: newTestPublicKeyBytes(t),
					SellersSellAddress:   newTestAddressBytes(t),
				}, lib.CanopyCommitteeId)
				require.NoError(t, err)
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
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
				got, e = sm.GetPoolBalance(lib.CanopyCommitteeId + types.EscrowPoolAddend)
				require.NoError(t, e)
				require.Equal(t, amount*2, got)
				// ensure the order was edited
				order, e := sm.GetOrder(0, lib.CanopyCommitteeId)
				require.NoError(t, e)
				require.Equal(t, amount*2, order.AmountForSale)
			},
		},
		{
			name:   "message delete order",
			detail: "basic 'happy path' handling for message delete order",
			preset: func(sm StateMachine) {
				// add to the pool
				require.NoError(t, sm.PoolAdd(lib.CanopyCommitteeId+types.EscrowPoolAddend, amount))
				// save the order in state
				_, err = sm.CreateOrder(&types.SellOrder{
					Committee:            lib.CanopyCommitteeId,
					AmountForSale:        amount,
					RequestedAmount:      1000,
					SellerReceiveAddress: newTestPublicKeyBytes(t),
					SellersSellAddress:   newTestAddressBytes(t),
				}, lib.CanopyCommitteeId)
				require.NoError(t, err)
			},
			msg: &types.MessageDeleteOrder{
				OrderId:     0,
				CommitteeId: lib.CanopyCommitteeId,
			},
			validate: func(sm StateMachine) {
				// ensure the account was subtracted from
				got, e := sm.GetAccountBalance(newTestAddress(t))
				require.NoError(t, e)
				require.Equal(t, amount, got)
				// ensure the pool was added to
				got, e = sm.GetPoolBalance(lib.CanopyCommitteeId + types.EscrowPoolAddend)
				require.NoError(t, e)
				require.Zero(t, got)
				// ensure the order was deleted
				_, e = sm.GetOrder(0, lib.CanopyCommitteeId)
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
			msg:    &types.MessageSend{},
		},
		{
			name:   "msg stake",
			detail: "evaluates the function for message stake",
			msg:    &types.MessageStake{},
		},
		{
			name:   "msg edit-stake",
			detail: "evaluates the function for message edit-stake",
			msg:    &types.MessageEditStake{},
		},
		{
			name:   "msg unstake",
			detail: "evaluates the function for message unstake",
			msg:    &types.MessageUnstake{},
		},
		{
			name:   "msg pause",
			detail: "evaluates the function for message pause",
			msg:    &types.MessagePause{},
		},
		{
			name:   "msg unpause",
			detail: "evaluates the function for message unpause",
			msg:    &types.MessageUnpause{},
		},
		{
			name:   "msg change param",
			detail: "evaluates the function for message change param",
			msg:    &types.MessageChangeParameter{},
		},
		{
			name:   "msg dao transfer",
			detail: "evaluates the function for message dao transfer",
			msg:    &types.MessageDAOTransfer{},
		},
		{
			name:   "msg certificate results",
			detail: "evaluates the function for message certificate results",
			msg:    &types.MessageCertificateResults{},
		},
		{
			name:   "msg subsidy",
			detail: "evaluates the function for message subsidy",
			msg:    &types.MessageSubsidy{},
		},
		{
			name:   "msg create order",
			detail: "evaluates the function for message create order",
			msg:    &types.MessageCreateOrder{},
		},
		{
			name:   "msg edit order",
			detail: "evaluates the function for message edit order",
			msg:    &types.MessageEditOrder{},
		},
		{
			name:   "msg delete order",
			detail: "evaluates the function for message delete order",
			msg:    &types.MessageDeleteOrder{},
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
				case *types.MessageSend:
					return feeParams.MessageSendFee
				case *types.MessageStake:
					return feeParams.MessageStakeFee
				case *types.MessageEditStake:
					return feeParams.MessageEditStakeFee
				case *types.MessageUnstake:
					return feeParams.MessageUnstakeFee
				case *types.MessagePause:
					return feeParams.MessagePauseFee
				case *types.MessageUnpause:
					return feeParams.MessageUnpauseFee
				case *types.MessageChangeParameter:
					return feeParams.MessageChangeParameterFee
				case *types.MessageDAOTransfer:
					return feeParams.MessageDaoTransferFee
				case *types.MessageCertificateResults:
					return feeParams.MessageCertificateResultsFee
				case *types.MessageSubsidy:
					return feeParams.MessageSubsidyFee
				case *types.MessageCreateOrder:
					return feeParams.MessageCreateOrderFee
				case *types.MessageEditOrder:
					return feeParams.MessageEditOrderFee
				case *types.MessageDeleteOrder:
					return feeParams.MessageDeleteOrderFee
				default:
					panic("unknown msg")
				}
			}()
			// execute function call
			got, err := sm.GetFeeForMessage(test.msg)
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
			msg:      &types.MessageSend{FromAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg stake",
			detail:   "retrieves the authorized signers for message stake",
			msg:      &types.MessageStake{PublicKey: newTestPublicKeyBytes(t), OutputAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t)},
		}, {
			name:     "msg edit-stake",
			detail:   "retrieves the authorized signers for message stake",
			msg:      &types.MessageEditStake{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg unstake",
			detail:   "retrieves the authorized signers for message unstake",
			msg:      &types.MessageUnstake{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg pause",
			detail:   "retrieves the authorized signers for message pause",
			msg:      &types.MessagePause{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg unpause",
			detail:   "retrieves the authorized signers for message unpause",
			msg:      &types.MessageUnpause{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t), newTestAddressBytes(t, 1)},
		}, {
			name:     "msg change param",
			detail:   "retrieves the authorized signers for message change param",
			msg:      &types.MessageChangeParameter{Signer: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg dao transfer",
			detail:   "retrieves the authorized signers for message dao transfer",
			msg:      &types.MessageDAOTransfer{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg subsidy",
			detail:   "retrieves the authorized signers for message subsidy",
			msg:      &types.MessageSubsidy{Address: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg create order",
			detail:   "retrieves the authorized signers for message create order",
			msg:      &types.MessageCreateOrder{SellersSellAddress: newTestAddressBytes(t)},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg edit order",
			detail:   "retrieves the authorized signers for message edit order",
			msg:      &types.MessageEditOrder{},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:     "msg delete order",
			detail:   "retrieves the authorized signers for message delete order",
			msg:      &types.MessageEditOrder{},
			expected: [][]byte{newTestAddressBytes(t)},
		}, {
			name:   "msg certificate results",
			detail: "retrieves the authorized signers for message delete order",
			msg: &types.MessageCertificateResults{
				Qc: &lib.QuorumCertificate{
					Header: &lib.View{CommitteeId: lib.CanopyCommitteeId, Height: 1},
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
			require.NoError(t, sm.SetValidator(&types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				StakedAmount: 100,
				Output:       newTestAddressBytes(t, 1),
			}))
			// preset a committee member
			require.NoError(t, sm.SetCommitteeMember(newTestAddress(t), lib.CanopyCommitteeId, 100))
			// preset an order
			_, err := sm.CreateOrder(&types.SellOrder{
				Committee:          lib.CanopyCommitteeId,
				SellersSellAddress: newTestAddressBytes(t),
			}, lib.CanopyCommitteeId)
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
		msg            *types.MessageSend
		error          string
	}{
		{
			name:           "insufficient amount",
			detail:         "the sender doesn't have enough tokens",
			presetSender:   1,
			presetReceiver: 0,
			msg: &types.MessageSend{
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
			msg: &types.MessageSend{
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
			msg: &types.MessageSend{
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
			msg: &types.MessageSend{
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
		msg             *types.MessageStake
		expected        *types.Validator
		error           string
	}{
		{
			name:   "invalid public key",
			detail: "the sender public key is invalid",
			msg:    &types.MessageStake{PublicKey: newTestAddressBytes(t)},
			error:  "public key is invalid",
		}, {
			name:            "validator already exists",
			detail:          "the validator already exists in state",
			msg:             &types.MessageStake{PublicKey: newTestPublicKeyBytes(t)},
			expected:        &types.Validator{Address: newTestAddressBytes(t)},
			presetValidator: true,
			error:           "validator exists",
		},
		{
			name:         "insufficient amount",
			detail:       "the sender doesn't have enough tokens",
			presetSender: 0,
			msg: &types.MessageStake{
				PublicKey: newTestPublicKeyBytes(t),
				Amount:    1,
			},
			error: "insufficient funds",
		},
		{
			name:         "stake all funds as committee member",
			detail:       "the sender stakes all funds as committee member",
			presetSender: 1,
			msg: &types.MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      false,
				Compound:      true,
			},
			expected: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
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
			msg: &types.MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      false,
				Compound:      true,
			},
			expected: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
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
			msg: &types.MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      true,
				Compound:      true,
			},
			expected: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
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
			msg: &types.MessageStake{
				PublicKey:     newTestPublicKeyBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Delegate:      true,
				Compound:      true,
			},
			expected: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
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
				stakedSupply, e := sm.GetCommitteeStakedSupply(id)
				require.NoError(t, e)
				require.Equal(t, test.msg.Amount, stakedSupply.Amount)
			}
			if val.Delegate {
				// validate the addition to the delegated pool
				require.Equal(t, test.msg.Amount, supply.Delegated)
				// validate the addition to the delegations only
				for _, id := range val.Committees {
					// get the supply for each committee
					stakedSupply, e := sm.GetDelegateStakedSupply(id)
					require.NoError(t, e)
					require.Equal(t, test.msg.Amount, stakedSupply.Amount)
					// validate the delegate membership
					page, e := sm.GetDelegatesPaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*types.ValidatorPage)
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
					list := (page.Results).(*types.ValidatorPage)
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
		presetValidator   *types.Validator
		msg               *types.MessageEditStake
		expectedValidator *types.Validator
		expectedSupply    *types.Supply
		error             string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to edit it",
			msg:    &types.MessageEditStake{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		},
		{
			name:   "unstaking",
			detail: "the validator is unstaking and cannot be edited",
			presetValidator: &types.Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &types.MessageEditStake{Address: newTestAddressBytes(t)},
			error: "unstaking",
		},
		{
			name:   "unauthorized output change",
			detail: "the sender is unable to change the output address",
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        2,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Compound:      true,
			},
			error: "unauthorized tx",
		},
		{
			name:   "invalid amount",
			detail: "the sender attempts to lower the stake by edit-stake",
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 2,
			},
			msg: &types.MessageEditStake{
				Address: newTestAddressBytes(t),
				Amount:  1,
			},
			error: "amount is invalid",
		},
		{
			name:   "insufficient funds",
			detail: "the sender doesn't have enough funds to complete the edit stake",
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        2,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t),
				Compound:      true,
			},
			error: "insufficient funds",
		},
		{
			name:   "edit stake, same balance, same committees",
			detail: "the validator is updated but the balance and committees remains the same",
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				NetAddress:    "http://example2.com",
				OutputAddress: newTestAddressBytes(t, 1),
				Compound:      false,
				Signer:        newTestAddressBytes(t),
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example2.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t, 1),
				Compound:     false,
			},
			expectedSupply: &types.Supply{
				Total:  1,
				Staked: 1,
				CommitteesWithDelegations: []*types.Pool{
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
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Delegate:     true,
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{0, 1},
				OutputAddress: newTestAddressBytes(t),
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Delegate:     true,
			},
			expectedSupply: &types.Supply{
				Total:     1,
				Staked:    1,
				Delegated: 1,
				CommitteesWithDelegations: []*types.Pool{
					{
						Id:     0,
						Amount: 1,
					},
					{
						Id:     1,
						Amount: 1,
					},
				},
				DelegationsOnly: []*types.Pool{
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
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			msg: &types.MessageEditStake{
				Address:       newTestAddressBytes(t),
				Amount:        1,
				Committees:    []uint64{1, 2, 3},
				NetAddress:    "http://example.com",
				OutputAddress: newTestAddressBytes(t),
				Compound:      true,
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				PublicKey:    newTestPublicKeyBytes(t),
				NetAddress:   "http://example.com",
				StakedAmount: 1,
				Committees:   []uint64{1, 2, 3},
				Output:       newTestAddressBytes(t),
				Compound:     true,
			},
			expectedSupply: &types.Supply{
				Total:  1,
				Staked: 1,
				CommitteesWithDelegations: []*types.Pool{
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
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			msg: &types.MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     1,
				Committees: []uint64{1, 2, 3},
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{1, 2, 3},
				Delegate:     true,
			},
			expectedSupply: &types.Supply{
				Total:     1,
				Staked:    1,
				Delegated: 1,
				CommitteesWithDelegations: []*types.Pool{
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
				DelegationsOnly: []*types.Pool{
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
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
			},
			msg: &types.MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     2,
				Committees: []uint64{1, 2, 3},
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 2,
				Committees:   []uint64{1, 2, 3},
			},
			expectedSupply: &types.Supply{
				Total:  3,
				Staked: 2,
				CommitteesWithDelegations: []*types.Pool{
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
			presetValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 1,
				Committees:   []uint64{0, 1},
				Delegate:     true,
			},
			msg: &types.MessageEditStake{
				Address:    newTestAddressBytes(t),
				Amount:     2,
				Committees: []uint64{1, 2, 3},
			},
			expectedValidator: &types.Validator{
				Address:      newTestAddressBytes(t),
				StakedAmount: 2,
				Committees:   []uint64{1, 2, 3},
				Delegate:     true,
			},
			expectedSupply: &types.Supply{
				Total:     3,
				Staked:    2,
				Delegated: 2,
				CommitteesWithDelegations: []*types.Pool{
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
				DelegationsOnly: []*types.Pool{
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
				supply := &types.Supply{}
				require.NoError(t, sm.SetValidators([]*types.Validator{test.presetValidator}, supply))
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
					list := (page.Results).(*types.ValidatorPage)
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
					list := (page.Results).(*types.ValidatorPage)
					// ensure the non membership
					require.Len(t, *list, 0)
				}
			} else {
				for _, id := range val.Committees {
					// validate the committee membership
					page, e := sm.GetCommitteePaginated(lib.PageParams{}, id)
					require.NoError(t, e)
					// extract the list from the page
					list := (page.Results).(*types.ValidatorPage)
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
					list := (page.Results).(*types.ValidatorPage)
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
		preset *types.Validator
		msg    *types.MessageUnstake
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to unstake it",
			msg:    &types.MessageUnstake{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator already unstaking",
			detail: "validator is already unstaking so this operation is invalid",
			preset: &types.Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &types.MessageUnstake{Address: newTestAddressBytes(t)},
			error: "validator is unstaking",
		},
		{
			name:   "validator not delegate",
			detail: "validator is not a delegate",
			preset: &types.Validator{
				Address: newTestAddressBytes(t),
			},
			msg: &types.MessageUnstake{Address: newTestAddressBytes(t)},
		},
		{
			name:   "validator a delegate",
			detail: "validator is a delegate",
			preset: &types.Validator{
				Address:  newTestAddressBytes(t),
				Delegate: true,
			},
			msg: &types.MessageUnstake{Address: newTestAddressBytes(t)},
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
				supply := &types.Supply{}
				require.NoError(t, sm.SetValidators([]*types.Validator{test.preset}, supply))
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
				unstakingBlocks = valParams.ValidatorDelegateUnstakingBlocks
			} else {
				unstakingBlocks = valParams.ValidatorUnstakingBlocks
			}
			finishUnstakingHeight := unstakingBlocks + sm.Height()
			// compare got vs expected
			require.Equal(t, finishUnstakingHeight, val.UnstakingHeight)
			// check for the unstaking key
			bz, err := sm.Get(types.KeyForUnstaking(finishUnstakingHeight, sender))
			require.NoError(t, err)
			require.Len(t, bz, 1)
		})
	}
}

func TestMessagePause(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *types.Validator
		msg    *types.MessagePause
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to pause it",
			msg:    &types.MessagePause{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator already paused",
			detail: "validator is already paused so this operation is invalid",
			preset: &types.Validator{
				Address:         newTestAddressBytes(t),
				MaxPausedHeight: 1,
			},
			msg:   &types.MessagePause{Address: newTestAddressBytes(t)},
			error: "validator paused",
		},
		{
			name:   "validator unstaking",
			detail: "validator is unstaking so this operation is invalid",
			preset: &types.Validator{
				Address:         newTestAddressBytes(t),
				UnstakingHeight: 1,
			},
			msg:   &types.MessagePause{Address: newTestAddressBytes(t)},
			error: "validator is unstaking",
		},
		{
			name:   "validator is a delegate",
			detail: "validator is a delegate",
			preset: &types.Validator{
				Address:  newTestAddressBytes(t),
				Delegate: true,
			},
			msg:   &types.MessagePause{Address: newTestAddressBytes(t)},
			error: "validator is a delegate",
		},
		{
			name:   "validator is not a delegate",
			detail: "validator is not a delegate",
			preset: &types.Validator{
				Address: newTestAddressBytes(t),
			},
			msg: &types.MessagePause{Address: newTestAddressBytes(t)},
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
				supply := &types.Supply{}
				require.NoError(t, sm.SetValidators([]*types.Validator{test.preset}, supply))
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
			// validate the unstaking of the validator object
			val, err := sm.GetValidator(sender)
			require.NoError(t, err)
			// get validator params
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// calculate the finish unstaking height
			maxPauseBlocks := valParams.ValidatorMaxPauseBlocks + sm.Height()
			// compare got vs expected
			require.Equal(t, maxPauseBlocks, val.MaxPausedHeight)
			// check for the paused key
			bz, err := sm.Get(types.KeyForPaused(maxPauseBlocks, sender))
			require.NoError(t, err)
			require.Len(t, bz, 1)
		})
	}
}

func TestMessageUnpause(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		preset *types.Validator
		msg    *types.MessageUnpause
		error  string
	}{
		{
			name:   "validator doesn't exist",
			detail: "validator does not exist to unpause it",
			msg:    &types.MessageUnpause{Address: newTestAddressBytes(t)},
			error:  "validator does not exist",
		}, {
			name:   "validator not paused",
			detail: "validator is not paused so this operation is invalid",
			preset: &types.Validator{
				Address: newTestAddressBytes(t),
			},
			msg:   &types.MessageUnpause{Address: newTestAddressBytes(t)},
			error: "validator not paused",
		},
		{
			name:   "validator unstaking",
			detail: "validator is unstaking so this operation is invalid",
			preset: &types.Validator{
				Address:         newTestAddressBytes(t),
				MaxPausedHeight: 1,
				UnstakingHeight: 1,
			},
			msg:   &types.MessageUnpause{Address: newTestAddressBytes(t)},
			error: "validator is unstaking",
		},
		{
			name:   "validator is paused",
			detail: "validator is paused",
			preset: &types.Validator{
				Address:         newTestAddressBytes(t),
				MaxPausedHeight: 1,
			},
			msg: &types.MessageUnpause{Address: newTestAddressBytes(t)},
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
				supply := &types.Supply{}
				require.NoError(t, sm.SetValidators([]*types.Validator{test.preset}, supply))
				require.NoError(t, sm.SetSupply(supply))
				// preset the validator as paused
				require.NoError(t, sm.Set(types.KeyForPaused(test.preset.MaxPausedHeight, sender), []byte{0x0}))
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
			bz, err := sm.Get(types.KeyForPaused(valParams.ValidatorMaxPauseBlocks+sm.Height(), sender))
			require.NoError(t, err)
			require.Nil(t, bz)
		})
	}
}

func TestHandleMessageChangeParameter(t *testing.T) {
	uint64Any, _ := lib.NewAny(&lib.UInt64Wrapper{Value: 100})
	stringAny, _ := lib.NewAny(&lib.StringWrapper{Value: "1/2"})
	tests := []struct {
		name           string
		detail         string
		proposalConfig types.GovProposalVoteConfig
		height         uint64
		msg            *types.MessageChangeParameter
		error          string
	}{
		{
			name:           "before start height",
			detail:         "the start height is greater than state machine height",
			height:         1,
			proposalConfig: types.AcceptAllProposals,
			msg: &types.MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   types.ParamValidatorUnstakingBlocks,
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
			proposalConfig: types.AcceptAllProposals,
			msg: &types.MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   types.ParamValidatorUnstakingBlocks,
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
			proposalConfig: types.RejectAllProposals,
			height:         2,
			msg: &types.MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   types.ParamValidatorUnstakingBlocks,
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
			proposalConfig: types.AcceptAllProposals,
			height:         2,
			msg: &types.MessageChangeParameter{
				ParameterSpace: "val",
				ParameterKey:   types.ParamValidatorUnstakingBlocks,
				ParameterValue: uint64Any,
				StartHeight:    2,
				EndHeight:      3,
				Signer:         newTestAddressBytes(t),
			},
		},
		{
			name:           "change protocol version",
			detail:         "successfully the protocol version with the message",
			proposalConfig: types.AcceptAllProposals,
			height:         2,
			msg: &types.MessageChangeParameter{
				ParameterSpace: "cons",
				ParameterKey:   types.ParamProtocolVersion,
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
			case types.ParamValidatorUnstakingBlocks: // validator
				require.Equal(t, uint64Value.Value, got.Validator.ValidatorUnstakingBlocks)
			case types.ParamProtocolVersion: // consensus
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
		proposalConfig types.GovProposalVoteConfig
		height         uint64
		msg            *types.MessageDAOTransfer
		error          string
	}{
		{
			name:           "before start height",
			detail:         "the start height is greater than state machine height",
			height:         1,
			daoPreset:      1,
			proposalConfig: types.AcceptAllProposals,
			msg: &types.MessageDAOTransfer{
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
			proposalConfig: types.AcceptAllProposals,
			daoPreset:      1,
			msg: &types.MessageDAOTransfer{
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
			proposalConfig: types.RejectAllProposals,
			height:         2,
			daoPreset:      1,
			msg: &types.MessageDAOTransfer{
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
			proposalConfig: types.AcceptAllProposals,
			height:         2,
			msg: &types.MessageDAOTransfer{
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
			proposalConfig: types.AcceptAllProposals,
			daoPreset:      1,
			height:         2,
			msg: &types.MessageDAOTransfer{
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
			PaymentPercents: []*lib.PaymentPercents{{Address: newTestAddressBytes(t), Percent: 100}},
		},
		SlashRecipients: &lib.SlashRecipients{
			DoubleSigners: []*lib.DoubleSigner{
				{PubKey: newTestPublicKeyBytes(t), Heights: []uint64{1}},
			},
			BadProposers: [][]byte{newTestPublicKeyBytes(t)},
		},
		Orders: &lib.Orders{
			BuyOrders: []*lib.BuyOrder{{
				OrderId:             0,
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
		msg                    *types.MessageCertificateResults
		error                  string
	}{
		{
			name:                   "non subsidized committee",
			detail:                 "the committee is not subsidized",
			nonSubsidizedCommittee: true,
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 3,
					CommitteeId:     lib.CanopyCommitteeId,
				},
			}},
			error: "non subsidized committee",
		},
		{
			name:   "older committee height",
			detail: "a committee height that is LTE state machine height - 2",
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 0,
					CommitteeId:     lib.CanopyCommitteeId,
				},
			}},
			error: "invalid certificate committee height",
		},
		{
			name:               "no committee members exist for that id",
			detail:             "there are no committee members for that ID",
			noCommitteeMembers: true,
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 3,
					CommitteeId:     lib.CanopyCommitteeId,
				},
			}},
			error: "there are no validators in the set",
		},
		{
			name:               "no committee members exist for that id",
			detail:             "there are no committee members for that ID",
			noCommitteeMembers: true,
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 3,
					CommitteeId:     lib.CanopyCommitteeId,
				},
			}},
			error: "there are no validators in the set",
		},
		{
			name:   "empty quorum certificate",
			detail: "the QC is empty",
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 3,
					CommitteeId:     lib.CanopyCommitteeId,
				},
			}},
			error: "empty quorum certificate",
		},
		{
			name:   "valid qc",
			detail: "the qc sent is valid",
			msg: &types.MessageCertificateResults{Qc: &lib.QuorumCertificate{
				Header: &lib.View{
					Height:          1,
					CommitteeHeight: 3,
					CommitteeId:     lib.CanopyCommitteeId,
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
			// increment height as height 2 ignores byzantine evidence
			sm.height++
			// check if the pool is subsidized
			if !test.nonSubsidizedCommittee {
				// subsidize the committee
				require.NoError(t, sm.PoolAdd(test.msg.Qc.Header.CommitteeId, 1))
			}
			// check if there exists a committee
			if !test.noCommitteeMembers {
				// track the supply
				supply := &types.Supply{}
				// for 4 validators
				for i := 0; i < 4; i++ {
					// set the validator
					require.NoError(t, sm.SetValidators([]*types.Validator{{
						Address:      newTestAddressBytes(t, i),
						PublicKey:    newTestPublicKeyBytes(t, i),
						StakedAmount: 100,
						Committees:   []uint64{lib.CanopyCommitteeId},
					}}, supply))
					// set the committee member
					require.NoError(t, sm.SetCommitteeMember(newTestAddress(t, i), lib.CanopyCommitteeId, 100))
				}
				// set the supply in state
				require.NoError(t, sm.SetSupply(supply))
				// create an aggregate signature
				// get the committee members
				committee, err := sm.GetCommitteeMembers(lib.CanopyCommitteeId, true)
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
			// preset some sell orders to test with
			for i := 0; i < 3; i++ {
				var buyerAddress []byte
				// set order #1, #2 with a buyer for 'reset' and 'close' functionality
				if i != 0 {
					buyerAddress = newTestAddressBytes(t)
				}
				// upsert each order in state
				_, err := sm.CreateOrder(&types.SellOrder{
					Committee:           lib.CanopyCommitteeId,
					BuyerReceiveAddress: buyerAddress,
					BuyerChainDeadline:  0,
					SellersSellAddress:  newTestAddressBytes(t),
				}, lib.CanopyCommitteeId)
				// ensure no error
				require.NoError(t, err)
			}
			// execute function call
			err := sm.HandleMessageCertificateResults(test.msg)
			// validate the expected error
			require.Equal(t, test.error != "", err != nil, err)
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// 1) validate the 'buy order'
			func() {
				order, e := sm.GetOrder(0, lib.CanopyCommitteeId)
				require.NoError(t, e)
				// convenience variable for buy order
				buyOrder := test.msg.Qc.Results.Orders.BuyOrders[0]
				// validate the receipt address was set
				require.Equal(t, buyOrder.BuyerReceiveAddress, order.BuyerReceiveAddress)
				// validate the deadline was set
				require.Equal(t, buyOrder.BuyerChainDeadline, order.BuyerChainDeadline)
			}()
			// 2) validate the 'reset order'
			func() {
				order, e := sm.GetOrder(1, lib.CanopyCommitteeId)
				require.NoError(t, e)
				// validate the receipt address was reset
				require.Len(t, order.BuyerReceiveAddress, 0)
				// validate the deadline was reset
				require.Zero(t, order.BuyerChainDeadline)
			}()

			// 3) validate the 'close order'
			func() {
				_, err = sm.GetOrder(2, lib.CanopyCommitteeId)
				require.ErrorContains(t, err, "order with id 2 not found")
			}()

			// 4) validate the 'checkpoint' service
			func() {
				// define convenience variable for checkpoint
				expected := test.msg.Qc.Results.Checkpoint
				// get the checkpoint
				got, e := sm.store.(lib.StoreI).GetCheckpoint(lib.CanopyCommitteeId, expected.Height)
				require.NoError(t, e)
				// check got vs expected
				require.Equal(t, expected.BlockHash, got)
			}()

			// 5) validate the 'committee data'
			func() {
				committeeData, e := sm.GetCommitteeData(lib.CanopyCommitteeId)
				require.NoError(t, e)
				// validate the committee height was properly set
				require.Equal(t, test.msg.Qc.Header.CommitteeHeight, committeeData.CommitteeHeight)
				// validate the chain height was properly set
				require.Equal(t, test.msg.Qc.Header.Height, committeeData.ChainHeight)
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
		msg           *types.MessageSubsidy
		error         string
	}{
		{
			name:          "insufficient funds",
			detail:        "the account does not have enough funds to complete the transfer",
			presetAccount: 1,
			msg: &types.MessageSubsidy{
				Address:     newTestAddressBytes(t),
				CommitteeId: lib.CanopyCommitteeId,
				Amount:      2,
			},
			error: "insufficient funds",
		},
		{
			name:          "successful transfer",
			detail:        "the transfer is successful",
			presetAccount: 1,
			msg: &types.MessageSubsidy{
				Address:     newTestAddressBytes(t),
				CommitteeId: lib.CanopyCommitteeId,
				Amount:      1,
			},
		},
		{
			name:          "successful transfer pre-balance",
			detail:        "the transfer is successful with pool having a non-zero balance to start",
			presetAccount: 1,
			presetPool:    2,
			msg: &types.MessageSubsidy{
				Address:     newTestAddressBytes(t),
				CommitteeId: lib.CanopyCommitteeId,
				Amount:      1,
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
			require.NoError(t, sm.PoolAdd(test.msg.CommitteeId, test.presetPool))
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
			got, err = sm.GetPoolBalance(test.msg.CommitteeId)
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
		msg              *types.MessageCreateOrder
		error            string
	}{
		{
			name:             "below minimum",
			detail:           "the order does not satisfy the minimum order size",
			presetAccount:    1,
			minimumOrderSize: 2,
			msg: &types.MessageCreateOrder{
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			error: "minimum order size",
		},
		{
			name:             "insufficient funds",
			detail:           "the account does not have sufficient funds to cover the sell order",
			minimumOrderSize: 1,
			msg: &types.MessageCreateOrder{
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			error: "insufficient funds",
		},
		{
			name:             "valid sell order",
			detail:           "the message creates a sell order in state",
			presetAccount:    1,
			minimumOrderSize: 1,
			msg: &types.MessageCreateOrder{
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      1,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// define an address variable for convenience
			address := crypto.NewAddress(test.msg.SellersSellAddress)
			// preset the minimum order size
			valParams, err := sm.GetParamsVal()
			require.NoError(t, err)
			// set minimum order size
			valParams.ValidatorMinimumOrderSize = test.minimumOrderSize
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
			got, err = sm.GetPoolBalance(test.msg.CommitteeId + types.EscrowPoolAddend)
			require.NoError(t, err)
			// validate the addition to the pool
			require.Equal(t, test.msg.AmountForSale, got)
			// get the order in state
			order, err := sm.GetOrder(0, test.msg.CommitteeId)
			require.NoError(t, err)
			// validate the creation of the order
			require.EqualExportedValues(t, &types.SellOrder{
				Committee:            test.msg.CommitteeId,
				AmountForSale:        test.msg.AmountForSale,
				RequestedAmount:      test.msg.RequestedAmount,
				SellerReceiveAddress: test.msg.SellerReceiveAddress,
				SellersSellAddress:   test.msg.SellersSellAddress,
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
		preset           *types.SellOrder
		msg              *types.MessageEditOrder
		expected         *types.SellOrder
		error            string
	}{
		{
			name:   "no order found",
			detail: "there exists no order",
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "not found",
		},
		{
			name:   "order already accepted",
			detail: "a buyer has already accepted the order, thus it cannot be edited",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				BuyerReceiveAddress:  newTestAddressBytes(t, 1), // signals a buyer
				BuyerChainDeadline:   100,                       // signals a buyer
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "order already accepted",
		},
		{
			name:             "minimum order size",
			detail:           "the edited order does not satisfy the minimum order size",
			minimumOrderSize: 2,
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "minimum order size",
		}, {
			name:   "insufficient funds",
			detail: "the account does not have the balance to cover the edit",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			error: "insufficient funds",
		},
		{
			name:   "edit receive address",
			detail: "the order simply updates the receive address but the amount stays the same",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSellAddress:   newTestAddressBytes(t),
			},
		},
		{
			name:             "increase sell amount",
			detail:           "the order has a increased the sell amount",
			presetAccount:    1,
			minimumOrderSize: 0,
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        2,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSellAddress:   newTestAddressBytes(t),
			},
		},
		{
			name:   "decrease sell amount",
			detail: "the order has a decreased the sell amount",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageEditOrder{
				OrderId:              0,
				CommitteeId:          lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
			},
			expected: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				SellerReceiveAddress: newTestAddressBytes(t, 2),
				SellersSellAddress:   newTestAddressBytes(t),
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
				address = crypto.NewAddress(test.preset.SellersSellAddress)
				// preset the minimum order size
				valParams, err := sm.GetParamsVal()
				require.NoError(t, err)
				// set minimum order size
				valParams.ValidatorMinimumOrderSize = test.minimumOrderSize
				// set back in state
				require.NoError(t, sm.SetParamsVal(valParams))
				// preset the account with tokens
				require.NoError(t, sm.AccountAdd(address, test.presetAccount))
				// get the proper order book
				orderBook, err := sm.GetOrderBook(lib.CanopyCommitteeId)
				require.NoError(t, err)
				// preset the sell order
				_ = orderBook.AddOrder(test.preset)
				// set it back in state
				require.NoError(t, sm.SetOrderBook(orderBook))
				// preset the pool with the amount to sell
				require.NoError(t, sm.PoolAdd(test.preset.Committee+types.EscrowPoolAddend, test.preset.AmountForSale))
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
			got, err = sm.GetPoolBalance(test.msg.CommitteeId + types.EscrowPoolAddend)
			require.NoError(t, err)
			// validate the subtraction/addition to/from the pool
			require.Equal(t, test.preset.AmountForSale-(test.preset.AmountForSale-test.msg.AmountForSale), got)
			// get the order in state
			order, err := sm.GetOrder(0, test.msg.CommitteeId)
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
		preset        *types.SellOrder
		msg           *types.MessageDeleteOrder
		error         string
	}{
		{
			name:   "no order found",
			detail: "there exists no order",
			msg: &types.MessageDeleteOrder{
				OrderId:     0,
				CommitteeId: lib.CanopyCommitteeId,
			},
			error: "not found",
		},
		{
			name:   "order already accepted",
			detail: "a buyer has already accepted the order, thus it cannot be edited",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        1,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				BuyerReceiveAddress:  newTestAddressBytes(t, 1), // signals a buyer
				BuyerChainDeadline:   100,                       // signals a buyer
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageDeleteOrder{
				OrderId:     0,
				CommitteeId: lib.CanopyCommitteeId,
			},
			error: "order already accepted",
		},
		{
			name:   "successful delete",
			detail: "the order delete was successful",
			preset: &types.SellOrder{
				Id:                   0,
				Committee:            lib.CanopyCommitteeId,
				AmountForSale:        2,
				RequestedAmount:      0,
				SellerReceiveAddress: newTestAddressBytes(t),
				SellersSellAddress:   newTestAddressBytes(t),
			},
			msg: &types.MessageDeleteOrder{
				OrderId:     0,
				CommitteeId: lib.CanopyCommitteeId,
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
				address = crypto.NewAddress(test.preset.SellersSellAddress)
				// preset the account with tokens
				require.NoError(t, sm.AccountAdd(address, test.presetAccount))
				// get the proper order book
				orderBook, err := sm.GetOrderBook(lib.CanopyCommitteeId)
				require.NoError(t, err)
				// preset the sell order
				_ = orderBook.AddOrder(test.preset)
				// set it back in state
				require.NoError(t, sm.SetOrderBook(orderBook))
				// preset the pool with the amount to sell
				require.NoError(t, sm.PoolAdd(test.preset.Committee+types.EscrowPoolAddend, test.preset.AmountForSale))
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
			got, err = sm.GetPoolBalance(test.msg.CommitteeId + types.EscrowPoolAddend)
			require.NoError(t, err)
			// validate the subtraction from the pool
			require.Zero(t, got)
			// validate the delete
			_, err = sm.GetOrder(0, test.msg.CommitteeId)
			require.ErrorContains(t, err, "not found")
		})
	}
}