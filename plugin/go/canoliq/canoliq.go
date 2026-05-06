package canoliq

import (
	"log"
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// Canoliq is the per-FSM-request execution context. Every inbound FSM
// lifecycle message creates a fresh Canoliq with the request's fsmId so
// concurrent requests do not interfere with one another. Height is tracked
// on the long-lived Plugin (not the per-request Canoliq) and surfaced via
// Plugin.height().
type Canoliq struct {
	Config    Config
	FSMConfig *contract.PluginFSMConfig
	plugin    *Plugin
	fsmId     uint64
}

// Genesis runs the canoLiq genesis distribution exactly once. It is idempotent:
// subsequent calls observe genesis_complete=true and short-circuit.
func (c *Canoliq) Genesis(req *contract.PluginGenesisRequest) *contract.PluginGenesisResponse {
	if err := c.runGenesis(req); err != nil {
		return &contract.PluginGenesisResponse{Error: err}
	}
	return &contract.PluginGenesisResponse{}
}

// BeginBlock runs the per-block governance + treasury hooks: self-bootstrap
// genesis if the FSM never sent a PluginGenesisRequest (chain genesis.json
// has no canoliq plugin section), then tally and dispatch any expired
// proposals.
func (c *Canoliq) BeginBlock(req *contract.PluginBeginRequest) *contract.PluginBeginResponse {
	if err := c.bootstrapGenesisIfNeeded(); err != nil {
		return &contract.PluginBeginResponse{Error: err}
	}
	height := req.GetHeight()
	if err := c.processProposals(height); err != nil {
		return &contract.PluginBeginResponse{Error: err}
	}
	return &contract.PluginBeginResponse{}
}

// bootstrapGenesisIfNeeded runs runGenesis exactly once when the plugin
// detects it has not been initialized. The Canopy FSM only sends a
// PluginGenesisRequest if the chain-genesis.json carries a plugin section
// for this plugin id; in localnet/Docker setups that section is typically
// absent, so without this self-bootstrap the plugin runs forever with
// genesis_complete=false and ProcessRewards is a no-op.
//
// runGenesis is idempotent (short-circuits on globals.GenesisComplete), so
// running it from BeginBlock is safe whether or not the FSM also dispatches
// the explicit Genesis call.
func (c *Canoliq) bootstrapGenesisIfNeeded() *contract.PluginError {
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if g.GenesisComplete {
		return nil
	}
	if c.Config.GenesisPath == "" {
		// No genesis source configured. Tests that drive BeginBlock
		// without a genesis file rely on this branch to skip cleanly.
		return nil
	}
	return c.runGenesis(nil)
}

// CheckTx statelessly validates a transaction and returns the authorized
// signer set. The 'send' message is delegated to the existing contract
// package handler so the canoLiq plugin remains a superset of the tutorial.
func (c *Canoliq) CheckTx(request *contract.PluginCheckRequest) *contract.PluginCheckResponse {
	params, err := c.LoadParams()
	if err != nil {
		return &contract.PluginCheckResponse{Error: err}
	}
	msg, err := contract.FromAny(request.Tx.Msg)
	if err != nil {
		return &contract.PluginCheckResponse{Error: err}
	}
	switch x := msg.(type) {
	case *contract.MessageSend:
		return c.checkMessageSend(x, request.Tx.Fee)
	case *contract.MessageCanoliqDeposit:
		return c.CheckMessageCanoliqDeposit(x, request.Tx.Fee, params)
	case *contract.MessageCanoliqRedeem:
		return c.CheckMessageCanoliqRedeem(x, request.Tx.Fee, params)
	case *contract.MessageCanoliqClaimRedemption:
		return c.CheckMessageCanoliqClaimRedemption(x, request.Tx.Fee, params)
	case *contract.MessageCLIQTransfer:
		return c.CheckMessageCLIQTransfer(x, request.Tx.Fee, params)
	case *contract.MessageCLIQClaimVested:
		return c.CheckMessageCLIQClaimVested(x, request.Tx.Fee, params)
	case *contract.MessageCLIQStake:
		return c.CheckMessageCLIQStake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQUnstake:
		return c.CheckMessageCLIQUnstake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQClaimUnstake:
		return c.CheckMessageCLIQClaimUnstake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQProposalCreate:
		return c.CheckMessageCLIQProposalCreate(x, request.Tx.Fee, params)
	case *contract.MessageCLIQVote:
		return c.CheckMessageCLIQVote(x, request.Tx.Fee, params)
	case *contract.MessageBuybackExecute:
		return c.CheckMessageBuybackExecute(x, request.Tx.Fee, params)
	case *contract.MessageDAOTreasurySpend:
		return c.CheckMessageDAOTreasurySpend(x, request.Tx.Fee, params)
	case *contract.MessageMultisigApprove:
		return c.CheckMessageMultisigApprove(x, request.Tx.Fee, params)
	default:
		return &contract.PluginCheckResponse{Error: ErrUnsupportedMessage()}
	}
}

// DeliverTx applies a transaction. Same dispatch shape as CheckTx.
func (c *Canoliq) DeliverTx(request *contract.PluginDeliverRequest) *contract.PluginDeliverResponse {
	params, err := c.LoadParams()
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	msg, err := contract.FromAny(request.Tx.Msg)
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	switch x := msg.(type) {
	case *contract.MessageSend:
		return c.deliverMessageSend(x, request.Tx.Fee)
	case *contract.MessageCanoliqDeposit:
		return c.DeliverMessageCanoliqDeposit(x, request.Tx.Fee, params)
	case *contract.MessageCanoliqRedeem:
		return c.DeliverMessageCanoliqRedeem(x, request.Tx.Fee, params)
	case *contract.MessageCanoliqClaimRedemption:
		return c.DeliverMessageCanoliqClaimRedemption(x, request.Tx.Fee, params)
	case *contract.MessageCLIQTransfer:
		return c.DeliverMessageCLIQTransfer(x, request.Tx.Fee, params)
	case *contract.MessageCLIQClaimVested:
		return c.DeliverMessageCLIQClaimVested(x, request.Tx.Fee, params)
	case *contract.MessageCLIQStake:
		return c.DeliverMessageCLIQStake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQUnstake:
		return c.DeliverMessageCLIQUnstake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQClaimUnstake:
		return c.DeliverMessageCLIQClaimUnstake(x, request.Tx.Fee, params)
	case *contract.MessageCLIQProposalCreate:
		return c.DeliverMessageCLIQProposalCreate(x, request.Tx.Fee, params)
	case *contract.MessageCLIQVote:
		return c.DeliverMessageCLIQVote(x, request.Tx.Fee, params)
	case *contract.MessageBuybackExecute:
		return c.DeliverMessageBuybackExecute(x, request.Tx.Fee, params)
	case *contract.MessageDAOTreasurySpend:
		return c.DeliverMessageDAOTreasurySpend(x, request.Tx.Fee, params)
	case *contract.MessageMultisigApprove:
		return c.DeliverMessageMultisigApprove(x, request.Tx.Fee, params)
	default:
		return &contract.PluginDeliverResponse{Error: ErrUnsupportedMessage()}
	}
}

// EndBlock runs the per-block reward sweep that applies the 12% protocol fee
// and the 40/30/15/15 split to canoLiq's committee reward pool, refreshes
// the read-only snapshot used by the HTTP query layer, and drains any
// pending per-address lazy queries. All of these need an active FSM
// context for state reads — see snapshot.go and lazy_query.go.
func (c *Canoliq) EndBlock(req *contract.PluginEndRequest) *contract.PluginEndResponse {
	if err := c.ProcessRewards(req); err != nil {
		return &contract.PluginEndResponse{Error: err}
	}
	if err := c.refreshSnapshot(req.GetHeight()); err != nil {
		return &contract.PluginEndResponse{Error: err}
	}
	c.drainLazyQueries()
	return &contract.PluginEndResponse{}
}

// CheckMessageCanoliqDeposit validates a deposit statelessly: address shape,
// non-zero amount, and minimum tx fee.
func (c *Canoliq) CheckMessageCanoliqDeposit(msg *contract.MessageCanoliqDeposit, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.DepositFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCanoliqRedeem validates a redeem request statelessly.
func (c *Canoliq) CheckMessageCanoliqRedeem(msg *contract.MessageCanoliqRedeem, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.CcnpyAmount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.RedeemFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCanoliqClaimRedemption validates a claim_redemption request statelessly.
func (c *Canoliq) CheckMessageCanoliqClaimRedemption(msg *contract.MessageCanoliqClaimRedemption, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.ClaimFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCLIQTransfer validates a CLIQ transfer statelessly.
func (c *Canoliq) CheckMessageCLIQTransfer(msg *contract.MessageCLIQTransfer, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 || len(msg.ToAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.CliqTransferFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.ToAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCLIQClaimVested validates a CLIQ claim_vested request.
func (c *Canoliq) CheckMessageCLIQClaimVested(msg *contract.MessageCLIQClaimVested, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.ClaimFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// checkMessageSend forwards to a minimal local validator for plain CNPY
// transfers. Identical to the contract tutorial check, kept here so the
// canoLiq plugin can be a drop-in replacement that still supports send.
func (c *Canoliq) checkMessageSend(msg *contract.MessageSend, _ uint64) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 || len(msg.ToAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.ToAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// deliverMessageSend implements the same CNPY-transfer logic as the contract
// tutorial. It is duplicated rather than imported because contract.Contract's
// plugin handle is unexported and the canoliq plugin uses a different Plugin
// runtime for FSM IO.
func (c *Canoliq) deliverMessageSend(msg *contract.MessageSend, fee uint64) *contract.PluginDeliverResponse {
	fromKey := contract.KeyForAccount(msg.FromAddress)
	toKey := contract.KeyForAccount(msg.ToAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	fromQ, toQ, feeQ := rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: feeQ, Key: feePoolKey},
			{QueryId: fromQ, Key: fromKey},
			{QueryId: toQ, Key: toKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	from, to, feePool := new(contract.Account), new(contract.Account), new(contract.Pool)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case fromQ:
			if e := contract.Unmarshal(r.Entries[0].Value, from); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case toQ:
			if e := contract.Unmarshal(r.Entries[0].Value, to); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case feeQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	deduct := msg.Amount + fee
	if from.Amount < deduct {
		return &contract.PluginDeliverResponse{Error: contract.ErrInsufficientFunds()}
	}
	if string(fromKey) == string(toKey) {
		to = from
	}
	from.Amount -= deduct
	feePool.Amount += fee
	to.Amount += msg.Amount
	fromBz, e := contract.Marshal(from)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	toBz, e := contract.Marshal(to)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: feePoolKey, Value: feeBz},
		{Key: toKey, Value: toBz},
	}
	var deletes []*contract.PluginDeleteOp
	if from.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: fromKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: fromKey, Value: fromBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	log.Printf("canoliq: send delivered: %x → %x amount=%d fee=%d", msg.FromAddress, msg.ToAddress, msg.Amount, fee)
	return &contract.PluginDeliverResponse{}
}
