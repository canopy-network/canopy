package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// buyback.go executes a passed ProposalBuyback. WP §6 specifies "market
// buyback and burn or direct distribution governed by DAO" — this Phase 2
// implementation is an internal accounting swap at a proposal-set price:
// CNPY drains from canoliq/buyback/pool into treasury/canoliq, CLIQ moves
// out of treasury/cliq, and the disposition (BURN or DISTRIBUTE_STAKERS) is
// applied. Real on-chain market routes are deferred to Phase 3.

// CheckMessageBuybackExecute validates a buyback trigger statelessly.
func (c *Canoliq) CheckMessageBuybackExecute(msg *contract.MessageBuybackExecute, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
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

// DeliverMessageBuybackExecute applies a passed ProposalBuyback. The order
// is keyed by proposal_id; re-execution is a no-op once executed=true.
func (c *Canoliq) DeliverMessageBuybackExecute(msg *contract.MessageBuybackExecute, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	cQ, fQ := rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
		},
	})
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if resp.Error != nil {
		return &contract.PluginDeliverResponse{Error: resp.Error}
	}
	cnpy := new(contract.Account)
	feePool := new(contract.Pool)
	for _, r := range resp.Results {
		if len(r.Entries) == 0 {
			continue
		}
		switch r.QueryId {
		case cQ:
			if e := contract.Unmarshal(r.Entries[0].Value, cnpy); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case fQ:
			if e := contract.Unmarshal(r.Entries[0].Value, feePool); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	order, err := c.loadBuybackOrder(msg.ProposalId)
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if order == nil {
		return &contract.PluginDeliverResponse{Error: ErrBuybackOrderNotFound()}
	}
	if order.Executed {
		return &contract.PluginDeliverResponse{Error: ErrProposalAlreadyExecuted()}
	}
	if order.Payload == nil {
		return &contract.PluginDeliverResponse{Error: ErrInvalidProposalPayload()}
	}
	payload := order.Payload
	available := c.readScalar(KeyForBuybackPool())
	cnpyDraw := payload.CnpyAmount
	if cnpyDraw > available {
		cnpyDraw = available
	}
	if cnpyDraw == 0 {
		return &contract.PluginDeliverResponse{Error: ErrInvalidAmount()}
	}
	if payload.PriceMicroCnpyPerCliq == 0 {
		return &contract.PluginDeliverResponse{Error: ErrInvalidProposalPayload()}
	}
	cliqAcquired := mulDiv(cnpyDraw, 1_000_000, payload.PriceMicroCnpyPerCliq)
	if cliqAcquired == 0 {
		return &contract.PluginDeliverResponse{Error: ErrPoolMath("buyback acquired zero")}
	}
	treasuryCLIQ := c.readScalar(KeyForTreasuryCLIQ())
	if treasuryCLIQ < cliqAcquired {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientTreasuryCLIQ()}
	}
	treasuryCLIQ -= cliqAcquired
	treasuryCNPY := c.readScalar(KeyForTreasuryCNPY()) + cnpyDraw
	available -= cnpyDraw
	cnpy.Amount -= fee
	feePool.Amount += fee
	sets := []*contract.PluginSetOp{
		{Key: KeyForBuybackPool(), Value: EncodeUint64(available)},
		{Key: KeyForTreasuryCLIQ(), Value: EncodeUint64(treasuryCLIQ)},
		{Key: KeyForTreasuryCNPY(), Value: EncodeUint64(treasuryCNPY)},
	}
	switch payload.Mode {
	case contract.BuybackMode_BUYBACK_BURN:
		globals, err := c.LoadGlobals()
		if err != nil {
			return &contract.PluginDeliverResponse{Error: err}
		}
		if globals.CliqTotalSupply >= cliqAcquired {
			globals.CliqTotalSupply -= cliqAcquired
		} else {
			globals.CliqTotalSupply = 0
		}
		if globals.CliqCirculatingSupply >= cliqAcquired {
			globals.CliqCirculatingSupply -= cliqAcquired
		} else {
			globals.CliqCirculatingSupply = 0
		}
		gBz, e := contract.Marshal(globals)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: KeyForGlobals(), Value: gBz})
	case contract.BuybackMode_BUYBACK_DISTRIBUTE_STAKERS:
		distributeSets, err := c.distributeBuybackToStakers(cliqAcquired)
		if err != nil {
			return &contract.PluginDeliverResponse{Error: err}
		}
		sets = append(sets, distributeSets...)
	default:
		return &contract.PluginDeliverResponse{Error: ErrInvalidProposalPayload()}
	}
	order.CnpyDrawn = cnpyDraw
	order.CliqAcquired = cliqAcquired
	order.Executed = true
	order.ExecutedAtHeight = c.currentHeight()
	oBz, e := contract.Marshal(order)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets = append(sets, &contract.PluginSetOp{Key: KeyForBuybackOrder(msg.ProposalId), Value: oBz})
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets = append(sets, &contract.PluginSetOp{Key: feePoolKey, Value: feeBz})
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	_ = params
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// distributeBuybackToStakers builds the set ops crediting acquired CLIQ to
// each active staker pro-rata to their CLIQStake. Rounding remainder is
// credited to the largest staker so the total exactly matches cliq_acquired.
func (c *Canoliq) distributeBuybackToStakers(cliqAcquired uint64) ([]*contract.PluginSetOp, *contract.PluginError) {
	idx, err := c.loadStakeIndex()
	if err != nil {
		return nil, err
	}
	if len(idx.Addresses) == 0 {
		// No stakers to distribute to; treat as a buyback void (return CLIQ to
		// treasury) by emitting no sets — caller already deducted treasury_cliq,
		// so we re-credit it.
		return []*contract.PluginSetOp{{Key: KeyForTreasuryCLIQ(), Value: EncodeUint64(c.readScalar(KeyForTreasuryCLIQ()) + cliqAcquired)}}, nil
	}
	stakes := make([]*contract.CLIQStake, 0, len(idx.Addresses))
	totalStake := uint64(0)
	for _, addr := range idx.Addresses {
		stake, err := c.loadCLIQStake(addr)
		if err != nil {
			return nil, err
		}
		if stake == nil || stake.Amount == 0 {
			continue
		}
		stakes = append(stakes, stake)
		totalStake += stake.Amount
	}
	if totalStake == 0 {
		return []*contract.PluginSetOp{{Key: KeyForTreasuryCLIQ(), Value: EncodeUint64(c.readScalar(KeyForTreasuryCLIQ()) + cliqAcquired)}}, nil
	}
	sets := make([]*contract.PluginSetOp, 0, len(stakes))
	allocated := uint64(0)
	largestIdx := 0
	for i, s := range stakes {
		if s.Amount > stakes[largestIdx].Amount {
			largestIdx = i
		}
	}
	credits := make([]uint64, len(stakes))
	for i, s := range stakes {
		credits[i] = mulDiv(cliqAcquired, s.Amount, totalStake)
		allocated += credits[i]
	}
	if allocated < cliqAcquired {
		credits[largestIdx] += cliqAcquired - allocated
	}
	for i, s := range stakes {
		current := DecodeUint64(c.readBytes(KeyForCLIQBalance(s.Address)))
		current += credits[i]
		sets = append(sets, &contract.PluginSetOp{Key: KeyForCLIQBalance(s.Address), Value: EncodeUint64(current)})
	}
	return sets, nil
}

// loadBuybackOrder reads the post-pass receipt for a proposal id.
func (c *Canoliq) loadBuybackOrder(proposalID uint64) (*contract.BuybackOrder, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForBuybackOrder(proposalID)}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	order := new(contract.BuybackOrder)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, order); e != nil {
		return nil, e
	}
	return order, nil
}
