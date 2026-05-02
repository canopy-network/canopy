package canoliq

import (
	"bytes"
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// stake.go implements CLIQ staking — locking liquid CLIQ for governance
// weight, queueing unstakes through an unbond window, and claiming matured
// records back to liquid balance.
//
// Voting weight is read against a snapshot taken at proposal creation height
// (see governance.go::voteWeightFor); to enable that snapshot,
// CLIQStake.staked_at_height records the latest stake increase.

// CheckMessageCLIQStake validates a stake request statelessly.
func (c *Canoliq) CheckMessageCLIQStake(msg *contract.MessageCLIQStake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCLIQUnstake validates an unstake request statelessly.
func (c *Canoliq) CheckMessageCLIQUnstake(msg *contract.MessageCLIQUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.Amount == 0 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// CheckMessageCLIQClaimUnstake validates a claim_unstake request statelessly.
func (c *Canoliq) CheckMessageCLIQClaimUnstake(msg *contract.MessageCLIQClaimUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginCheckResponse {
	if len(msg.FromAddress) != 20 {
		return &contract.PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if fee < params.StakeFee {
		return &contract.PluginCheckResponse{Error: ErrFeeBelowMinimum()}
	}
	return &contract.PluginCheckResponse{
		Recipient:         msg.FromAddress,
		AuthorizedSigners: [][]byte{msg.FromAddress},
	}
}

// DeliverMessageCLIQStake debits the sender's liquid CLIQ balance, credits
// the per-address CLIQStake record, and increments globals.total_staked_cliq.
// Multiple stakes from the same address aggregate; staked_at_height tracks
// the most recent increase so the snapshot guard can reject post-creation
// stake additions.
func (c *Canoliq) DeliverMessageCLIQStake(msg *contract.MessageCLIQStake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	balKey := KeyForCLIQBalance(msg.FromAddress)
	stakeKey := KeyForCLIQStake(msg.FromAddress)
	idxKey := KeyForCLIQStakeIndex()
	gKey := KeyForGlobals()
	cQ, fQ, bQ, sQ, iQ, gQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: bQ, Key: balKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: gQ, Key: gKey},
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
	stake := new(contract.CLIQStake)
	idx := new(contract.CLIQStakeIndex)
	globals := new(contract.CanoliqGlobals)
	var balBz []byte
	stakePresent := false
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
		case bQ:
			balBz = r.Entries[0].Value
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			stakePresent = stake.Address != nil
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	bal := DecodeUint64(balBz)
	if bal < msg.Amount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCLIQ()}
	}
	bal -= msg.Amount
	cnpy.Amount -= fee
	feePool.Amount += fee
	height := c.currentHeight()
	if !stakePresent {
		stake = &contract.CLIQStake{Address: msg.FromAddress, Amount: msg.Amount, StakedAtHeight: height}
		idx.Addresses = appendStakerIfMissing(idx.Addresses, msg.FromAddress)
	} else {
		stake.Amount += msg.Amount
		stake.StakedAtHeight = height
	}
	globals.TotalStakedCliq += msg.Amount
	stakeBz, e := contract.Marshal(stake)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	idxBz, e := contract.Marshal(idx)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
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
		{Key: stakeKey, Value: stakeBz},
		{Key: idxKey, Value: idxBz},
		{Key: gKey, Value: gBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if bal == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: balKey})
	} else {
		sets = append(sets, &contract.PluginSetOp{Key: balKey, Value: EncodeUint64(bal)})
	}
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

// DeliverMessageCLIQUnstake debits the staker's CLIQStake by `amount`, queues
// an UnstakingCLIQ record maturing at h + cliq_unstaking_blocks, and
// decrements globals.total_staked_cliq immediately. The unstaked tokens
// carry zero voting weight from this height onward.
func (c *Canoliq) DeliverMessageCLIQUnstake(msg *contract.MessageCLIQUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	stakeKey := KeyForCLIQStake(msg.FromAddress)
	idxKey := KeyForCLIQStakeIndex()
	gKey := KeyForGlobals()
	cQ, fQ, sQ, iQ, gQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: sQ, Key: stakeKey},
			{QueryId: iQ, Key: idxKey},
			{QueryId: gQ, Key: gKey},
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
	stake := new(contract.CLIQStake)
	idx := new(contract.CLIQStakeIndex)
	globals := new(contract.CanoliqGlobals)
	stakePresent := false
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
		case sQ:
			if e := contract.Unmarshal(r.Entries[0].Value, stake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			stakePresent = stake.Address != nil
		case iQ:
			if e := contract.Unmarshal(r.Entries[0].Value, idx); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		case gQ:
			if e := contract.Unmarshal(r.Entries[0].Value, globals); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
		}
	}
	if !stakePresent || stake.Amount < msg.Amount {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientStakedCLIQ()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	stake.Amount -= msg.Amount
	cnpy.Amount -= fee
	feePool.Amount += fee
	if globals.TotalStakedCliq >= msg.Amount {
		globals.TotalStakedCliq -= msg.Amount
	} else {
		globals.TotalStakedCliq = 0
	}
	id := globals.NextUnstakeId
	globals.NextUnstakeId++
	mature := c.currentHeight() + params.CliqUnstakingBlocks
	unstake := &contract.UnstakingCLIQ{
		Id:           id,
		Address:      msg.FromAddress,
		Amount:       msg.Amount,
		MatureHeight: mature,
	}
	uBz, e := contract.Marshal(unstake)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	gBz, e := contract.Marshal(globals)
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
		{Key: KeyForCLIQUnstaking(msg.FromAddress, id), Value: uBz},
		{Key: gKey, Value: gBz},
		{Key: feePoolKey, Value: feeBz},
	}
	var deletes []*contract.PluginDeleteOp
	if stake.Amount == 0 {
		deletes = append(deletes, &contract.PluginDeleteOp{Key: stakeKey})
		idx.Addresses = removeStaker(idx.Addresses, msg.FromAddress)
		idxBz, e := contract.Marshal(idx)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: idxKey, Value: idxBz})
	} else {
		sBz, e := contract.Marshal(stake)
		if e != nil {
			return &contract.PluginDeliverResponse{Error: e}
		}
		sets = append(sets, &contract.PluginSetOp{Key: stakeKey, Value: sBz})
	}
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

// DeliverMessageCLIQClaimUnstake matures an UnstakingCLIQ record by returning
// the CLIQ to the staker's liquid balance once mature_height has passed.
func (c *Canoliq) DeliverMessageCLIQClaimUnstake(msg *contract.MessageCLIQClaimUnstake, fee uint64, params *contract.CanoliqParams) *contract.PluginDeliverResponse {
	_ = params
	cnpyKey := contract.KeyForAccount(msg.FromAddress)
	feePoolKey := contract.KeyForFeePool(c.Config.ChainId)
	uKey := KeyForCLIQUnstaking(msg.FromAddress, msg.UnstakeId)
	balKey := KeyForCLIQBalance(msg.FromAddress)
	cQ, fQ, uQ, bQ := rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{
			{QueryId: cQ, Key: cnpyKey},
			{QueryId: fQ, Key: feePoolKey},
			{QueryId: uQ, Key: uKey},
			{QueryId: bQ, Key: balKey},
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
	unstake := new(contract.UnstakingCLIQ)
	var balBz []byte
	uPresent := false
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
		case uQ:
			if e := contract.Unmarshal(r.Entries[0].Value, unstake); e != nil {
				return &contract.PluginDeliverResponse{Error: e}
			}
			uPresent = unstake.Address != nil
		case bQ:
			balBz = r.Entries[0].Value
		}
	}
	if !uPresent {
		return &contract.PluginDeliverResponse{Error: ErrUnstakeNotFound()}
	}
	if c.currentHeight() < unstake.MatureHeight {
		return &contract.PluginDeliverResponse{Error: ErrUnstakeNotMature()}
	}
	if cnpy.Amount < fee {
		return &contract.PluginDeliverResponse{Error: ErrInsufficientCNPY()}
	}
	cnpy.Amount -= fee
	feePool.Amount += fee
	bal := DecodeUint64(balBz) + unstake.Amount
	cnpyBz, e := contract.Marshal(cnpy)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	feeBz, e := contract.Marshal(feePool)
	if e != nil {
		return &contract.PluginDeliverResponse{Error: e}
	}
	sets := []*contract.PluginSetOp{
		{Key: balKey, Value: EncodeUint64(bal)},
		{Key: feePoolKey, Value: feeBz},
	}
	deletes := []*contract.PluginDeleteOp{{Key: uKey}}
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

// loadCLIQStake reads a per-address CLIQStake record. Returns (nil, nil)
// when the staker has no record, distinguishable from an empty stake.
func (c *Canoliq) loadCLIQStake(addr []byte) (*contract.CLIQStake, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForCLIQStake(addr)}},
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
	stake := new(contract.CLIQStake)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, stake); e != nil {
		return nil, e
	}
	return stake, nil
}

// loadStakeIndex reads the singleton stake index. Returns an empty index when
// no stakers exist yet.
func (c *Canoliq) loadStakeIndex() (*contract.CLIQStakeIndex, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForCLIQStakeIndex()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	idx := new(contract.CLIQStakeIndex)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, idx); e != nil {
			return nil, e
		}
	}
	return idx, nil
}

func appendStakerIfMissing(addrs [][]byte, addr []byte) [][]byte {
	for _, a := range addrs {
		if bytes.Equal(a, addr) {
			return addrs
		}
	}
	return append(addrs, addr)
}

func removeStaker(addrs [][]byte, addr []byte) [][]byte {
	out := addrs[:0]
	for _, a := range addrs {
		if bytes.Equal(a, addr) {
			continue
		}
		out = append(out, a)
	}
	return out
}
