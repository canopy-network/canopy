package canoliq

import (
	"bytes"

	"github.com/canopy-network/go-plugin/contract"
)

// treasury.go implements canoLiq DAO treasury spends. WP §9 / Tokenomics §7
// require multisig + timelock for spending above a threshold. The spend
// lifecycle is:
//   1. Proposal passes (governance.go::dispatchPassed) → queueTreasurySpend
//      writes a TreasurySpend with executable_height adjusted by timelock.
//   2. (above-threshold only) MessageMultisigApprove records per-signer
//      approvals up to multisig_threshold.
//   3. MessageDAOTreasurySpend actually moves the funds when the timelock +
//      multisig conditions are satisfied. Execution is always explicit — there
//      is no BeginBlock auto-execute path (BeginBlock only queues).

// queueTreasurySpend writes a TreasurySpend record on proposal pass. Above
// treasury_threshold spends carry a timelock and a multisig requirement.
func (c *Canoliq) queueTreasurySpend(prop *contract.Proposal, payload *contract.ProposalTreasurySpend, params *contract.CanoliqParams, height uint64) *contract.PluginError {
	if payload.Recipient == nil || len(payload.Recipient) != 20 {
		return ErrInvalidAddress()
	}
	if payload.Amount == 0 {
		return ErrInvalidAmount()
	}
	if payload.Denomination != contract.SpendDenomination_SPEND_CNPY && payload.Denomination != contract.SpendDenomination_SPEND_CPLQ {
		return ErrInvalidProposalPayload()
	}
	// Multisig is still gated purely on amount vs the treasury threshold.
	requires := payload.Amount > params.TreasuryThreshold
	// Timelock comes from the proposal's recorded tier (so a small spend's 48h
	// and a large spend's 7d coexist); a tier timelock of 0 means immediate.
	// Legacy proposals (nil tier) keep the old rule: timelock only when the
	// spend requires multisig.
	executable := height
	if prop.Tier != nil {
		executable = height + prop.Tier.TimelockBlocks
	} else if requires {
		executable = height + params.TimelockBlocks
	}
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	id := g.NextSpendId + 1
	g.NextSpendId = id
	spend := &contract.TreasurySpend{
		Id:               id,
		ProposalId:       prop.Id,
		ExecutableHeight: executable,
		Payload:          payload,
		RequiresMultisig: requires,
		Executed:         false,
	}
	bz, e := contract.Marshal(spend)
	if e != nil {
		return e
	}
	gBz, e := contract.Marshal(g)
	if e != nil {
		return e
	}
	idx, err := c.loadSpendIndex()
	if err != nil {
		return err
	}
	idx = appendSpendID(idx, id)
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{
			{Key: KeyForTreasurySpend(id), Value: bz},
			{Key: KeyForGlobals(), Value: gBz},
			{Key: KeyForSpendIndex(), Value: idxBz},
		},
	}); err != nil {
		return err
	}
	return nil
}

// CheckMessageDAOTreasurySpend validates a spend trigger statelessly.
func (c *Canoliq) CheckMessageDAOTreasurySpend(msg *contract.MessageDAOTreasurySpend, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
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

// CheckMessageMultisigApprove validates a multisig approval statelessly.
func (c *Canoliq) CheckMessageMultisigApprove(msg *contract.MessageMultisigApprove, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.MultisigApproveFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// DeliverMessageDAOTreasurySpend triggers execution of a queued spend by
// proposal id. Resolves the underlying TreasurySpend record, asserts
// timelock + multisig coverage, then moves CNPY (or CPLQ) from the
// canoLiq DAO treasury bucket to the recipient.
func (c *Canoliq) DeliverMessageDAOTreasurySpend(msg *contract.MessageDAOTreasurySpend, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	cQ, fQ := qid(), qid()
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
	spend, err := c.findSpendForProposal(msg.ProposalId)
	if err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	if spend == nil {
		return &contract.PluginDeliverResponse{Error: ErrSpendNotFound()}
	}
	if spend.Executed {
		return &contract.PluginDeliverResponse{Error: ErrSpendAlreadyExecuted()}
	}
	if c.currentHeight() < spend.ExecutableHeight {
		return &contract.PluginDeliverResponse{Error: ErrSpendNotReady()}
	}
	if spend.RequiresMultisig {
		approvals, err := c.countMultisigApprovals(spend.Id, params)
		if err != nil {
			return &contract.PluginDeliverResponse{Error: err}
		}
		if approvals < params.MultisigThreshold {
			return &contract.PluginDeliverResponse{Error: ErrSpendNotReady()}
		}
	}
	if err := c.applySpend(spend); err != nil {
		return &contract.PluginDeliverResponse{Error: err}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// DeliverMessageMultisigApprove records a per-signer approval against a
// pending TreasurySpend. Only signers in params.multisig_signers may approve.
func (c *Canoliq) DeliverMessageMultisigApprove(msg *contract.MessageMultisigApprove, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	if !isMultisigSigner(msg.FromAddress, params.MultisigSigners) {
		return &contract.PluginDeliverResponse{Error: ErrNotMultisigSigner()}
	}
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	approvalKey := KeyForMultisigApproval(msg.SpendId, msg.FromAddress)
	spendKey := KeyForTreasurySpend(msg.SpendId)
	cQ, fQ, aQ, sQ := qid(), qid(), qid(), qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: aQ, Key: approvalKey},
			{QueryId: sQ, Key: spendKey},
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
	spend := new(contract.TreasurySpend)
	approvalPresent, spendPresent := false, false
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
		case aQ:
			approvalPresent = len(r.Entries[0].Value) > 0
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, spend); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			spendPresent = spend.Id != 0
		}
	}
	if !spendPresent {
		return &contract.PluginDeliverResponse{Error: ErrSpendNotFound()}
	}
	if spend.Executed {
		return &contract.PluginDeliverResponse{Error: ErrSpendAlreadyExecuted()}
	}
	if approvalPresent {
		return &contract.PluginDeliverResponse{Error: ErrAlreadyApproved()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	approval := &contract.MultisigApproval{SpendId: msg.SpendId, Signer: msg.FromAddress, Height: c.currentHeight()}
	aBz, e := contract.Marshal(approval)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: approvalKey, Value: aBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if cnpy.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: cnpyKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: cnpyKey, Value: cnpyBz})
	}
	if _, e := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets, Deletes: deletes}); e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	return &contract.PluginDeliverResponse{}
}

// applySpend moves the configured amount from the appropriate treasury bucket
// to the recipient and marks the spend executed. Idempotent at the caller —
// the executed flag prevents double-spend.
func (c *Canoliq) applySpend(spend *contract.TreasurySpend) *contract.PluginError {
	switch spend.Payload.Denomination {
	case contract.SpendDenomination_SPEND_CNPY:
		treasury := c.readScalar(KeyForTreasuryCNPY())
		if treasury < spend.Payload.Amount {
			return ErrInsufficientTreasuryCNPY()
		}
		treasury -= spend.Payload.Amount
		recipKey := contract.KeyForAccount(spend.Payload.Recipient)
		recip, err := c.loadAccount(recipKey)
		if err != nil {
			return err
		}
		recip.Amount += spend.Payload.Amount
		recipBz, e := contract.Marshal(recip)
		if e != nil {
			return e
		}
		spend.Executed = true
		spendBz, e := contract.Marshal(spend)
		if e != nil {
			return e
		}
		idx, err := c.loadSpendIndex()
		if err != nil {
			return err
		}
		idx = removeSpendID(idx, spend.Id)
		idxBz, e := contract.Marshal(idx)
		if e != nil {
			return e
		}
		// T5: track cumulative CNPY treasury burn for the runway metric.
		g, err := c.LoadGlobals()
		if err != nil {
			return err
		}
		g.TreasurySpentTotal += spend.Payload.Amount
		gBz, e := contract.Marshal(g)
		if e != nil {
			return e
		}
		_, err = c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
			Sets: []*contract.PluginSetOp{
				{Key: KeyForTreasuryCNPY(), Value: EncodeUint64(treasury)},
				{Key: recipKey, Value: recipBz},
				{Key: KeyForTreasurySpend(spend.Id), Value: spendBz},
				{Key: KeyForSpendIndex(), Value: idxBz},
				{Key: KeyForGlobals(), Value: gBz},
			},
		})
		return err
	case contract.SpendDenomination_SPEND_CPLQ:
		treasury := c.readScalar(KeyForTreasuryCPLQ())
		if treasury < spend.Payload.Amount {
			return ErrInsufficientTreasuryCPLQ()
		}
		treasury -= spend.Payload.Amount
		recipBalKey := KeyForCPLQBalance(spend.Payload.Recipient)
		recipBal := DecodeUint64(c.readBytes(recipBalKey)) + spend.Payload.Amount
		spend.Executed = true
		spendBz, e := contract.Marshal(spend)
		if e != nil {
			return e
		}
		idx, err := c.loadSpendIndex()
		if err != nil {
			return err
		}
		idx = removeSpendID(idx, spend.Id)
		idxBz, e := contract.Marshal(idx)
		if e != nil {
			return e
		}
		// L4: CPLQ held in the DAO treasury is not circulating; paying it out to
		// a recipient's liquid balance returns it to circulation, so bump
		// CplqCirculatingSupply by the spend amount (CplqTotalSupply is
		// unchanged — no mint/burn). (The deeper buyback↔circulating
		// reconciliation remains a Phase-3 item.)
		g, err := c.LoadGlobals()
		if err != nil {
			return err
		}
		g.CplqCirculatingSupply += spend.Payload.Amount
		gBz, e := contract.Marshal(g)
		if e != nil {
			return e
		}
		_, err = c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
			Sets: []*contract.PluginSetOp{
				{Key: KeyForTreasuryCPLQ(), Value: EncodeUint64(treasury)},
				{Key: recipBalKey, Value: EncodeUint64(recipBal)},
				{Key: KeyForTreasurySpend(spend.Id), Value: spendBz},
				{Key: KeyForSpendIndex(), Value: idxBz},
				{Key: KeyForGlobals(), Value: gBz},
			},
		})
		return err
	default:
		return ErrInvalidProposalPayload()
	}
}

// loadAccount reads an Account record. Empty when absent so callers can
// always read-modify-write.
func (c *Canoliq) loadAccount(key []byte) (*contract.Account, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: key}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	acc := new(contract.Account)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, acc); e != nil {
			return nil, e
		}
	}
	return acc, nil
}

// readBytes reads a raw byte value at `key`, or nil if absent.
func (c *Canoliq) readBytes(key []byte) []byte {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: key}},
	})
	if err != nil || resp == nil || resp.Error != nil {
		return nil
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil
	}
	return resp.Results[0].Entries[0].Value
}

// findSpendForProposal locates the queued TreasurySpend whose proposal_id
// matches. Returns (nil, nil) when not found.
func (c *Canoliq) findSpendForProposal(proposalID uint64) (*contract.TreasurySpend, *contract.PluginError) {
	idx, err := c.loadSpendIndex()
	if err != nil {
		return nil, err
	}
	for _, id := range idx.Ids {
		spend, err := c.loadSpend(id)
		if err != nil {
			return nil, err
		}
		if spend != nil && spend.ProposalId == proposalID {
			return spend, nil
		}
	}
	return nil, nil
}

func (c *Canoliq) loadSpend(id uint64) (*contract.TreasurySpend, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForTreasurySpend(id)}},
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
	spend := new(contract.TreasurySpend)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, spend); e != nil {
		return nil, e
	}
	return spend, nil
}

func (c *Canoliq) loadSpendIndex() (*contract.ProposalIndex, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForSpendIndex()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	idx := new(contract.ProposalIndex)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
			return nil, e
		}
	}
	return idx, nil
}

// countMultisigApprovals tallies recorded approvals against the configured
// signer set. Stale approvals from removed signers do not count.
func (c *Canoliq) countMultisigApprovals(spendID uint64, params *contract.CanoliqParams) (uint64, *contract.PluginError) {
	count := uint64(0)
	for _, signer := range params.MultisigSigners {
		raw := c.readBytes(KeyForMultisigApproval(spendID, signer))
		if len(raw) == 0 {
			continue
		}
		approval := new(contract.MultisigApproval)
		if e := contract.Unmarshal(raw, approval); e != nil {
			return 0, e
		}
		if bytes.Equal(approval.Signer, signer) {
			count++
		}
	}
	return count, nil
}

func isMultisigSigner(addr []byte, signers [][]byte) bool {
	for _, s := range signers {
		if bytes.Equal(s, addr) {
			return true
		}
	}
	return false
}

func appendSpendID(idx *contract.ProposalIndex, id uint64) *contract.ProposalIndex {
	for _, existing := range idx.Ids {
		if existing == id {
			return idx
		}
	}
	idx.Ids = append(idx.Ids, id)
	return idx
}

func removeSpendID(idx *contract.ProposalIndex, id uint64) *contract.ProposalIndex {
	out := idx.Ids[:0]
	for _, existing := range idx.Ids {
		if existing == id {
			continue
		}
		out = append(out, existing)
	}
	idx.Ids = out
	return idx
}
