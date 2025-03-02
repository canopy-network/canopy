package controller

import (
	"bytes"
	"slices"

	"github.com/canopy-network/canopy/bft"
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
)

/* This file implements the 'Certificate Result' logic which ensures the */

// NewCertificateResults() creates a structure to hold the results of the certificate produced by a quorum in consensus
func (c *Controller) NewCertificateResults(block *lib.Block, blockResult *lib.BlockResult, evidence *bft.ByzantineEvidence) (results *lib.CertificateResult) {
	// calculate reward recipients, creating a 'certificate results' object reference in the process
	results = c.CalculateRewardRecipients(block.BlockHeader.ProposerAddress, c.RootChainHeight())
	// handle swaps
	c.HandleSwaps(blockResult, results, c.RootChainHeight())
	// set slash recipients
	c.CalculateSlashRecipients(results, evidence)
	// set checkpoint
	c.CalculateCheckpoint(blockResult, results)
	// handle retired status
	c.HandleRetired(results)
	// exit
	return
}

// SendCertificateResultsTx() originates and auto-sends a CertificateResultsTx after successfully leading a Consensus height
func (c *Controller) SendCertificateResultsTx(qc *lib.QuorumCertificate) {
	// get the root chain id from the state
	rootChainId, err := c.FSM.GetRootChainId()
	// if the chain is its own root, don't send a transaction
	if c.Config.ChainId == rootChainId {
		// exit
		return
	}
	// save the block data from the quorum certificate
	blk := qc.Block
	// set the block data back in the certificate object after this function completes
	defer func() { qc.Block = blk }()
	// omit the block when sending the transaction as it's not relevant to the root chain
	qc.Block = nil
	// create a new certificate results transaction
	tx, err := types.NewCertificateResultsTx(c.PrivateKey, qc, rootChainId, c.Config.NetworkID, 0, c.RootChainHeight(), "")
	// if an error occurred during the tx creation
	if err != nil {
		// log the error
		c.log.Errorf("Creating auto-certificate-results-txn failed with err: %s", err.Error())
		// exit
		return
	}
	// handle the transaction on the root-Chain
	hash, err := c.RootChainInfo.RemoteCallbacks.Transaction(tx)
	// if an error occurred during the tx submission
	if err != nil {
		// log the error
		c.log.Errorf("Submitting auto-certificate-results-txn failed with err: %s", err.Error())
		// exit
		return
	}
	// log the submission of the transaction
	c.log.Infof("Successfully submitted the certificate-results-txn with hash %s", *hash)
}

// CalculateRewardRecipients() calculates the block reward recipients of the proposal
func (c *Controller) CalculateRewardRecipients(proposerAddress []byte, rootChainHeight uint64) (results *lib.CertificateResult) {
	// set block reward recipients
	results = &lib.CertificateResult{
		RewardRecipients: &lib.RewardRecipients{},
		SlashRecipients:  new(lib.SlashRecipients),
	}
	// get the root chain id from the governance params in the state machine
	rootChainId, err := c.FSM.GetRootChainId()
	// if an error occurred
	if err != nil {
		// log the error
		c.log.Warnf("An error occurred getting the root chain id from state: %s", err.Error())
		// exit with a non-nil result
		return results
	}
	// create variables to hold potential 'reward recipients' (delegate, nested-validator, nested-delegate)
	var delegate, nValidator, nDelegate *lib.LotteryWinner
	// start the proposer with a 100% allocation
	proposer := &lib.LotteryWinner{Winner: proposerAddress, Cut: 100}
	// get the delegate and their cut from the state machine
	delegate, err = c.GetRootChainLotteryWinner(rootChainHeight)
	// if an error occurred
	if err != nil {
		// log the error and continue
		c.log.Warnf("An error occurred choosing a root chain delegate lottery winner: %s", err.Error())
	}
	// calculate if this chain is its own root chain
	isOwnRoot := rootChainId == c.Config.ChainId
	// add the delegate as a reward recipient, subtracting share away from the proposer
	c.AddRewardRecipient(proposer, delegate, results, isOwnRoot, rootChainId)
	// if this chain isn't its own root chain, add the nested participants to the 'reward recipients
	if !isOwnRoot {
		// get the nested-validator lottery winner for the 'self chain' (if self is not root)
		nValidator, err = c.FSM.LotteryWinner(c.Config.ChainId, true)
		// if an error occurred
		if err != nil {
			// log the error and continue
			c.log.Warnf("An error occurred choosing a nested-validator lottery winner: %s", err.Error())
		}
		// add the nested validator as a reward recipient, subtracting share away from the proposer
		c.AddRewardRecipient(proposer, nValidator, results, isOwnRoot, rootChainId)
		// get the nested-delegate lottery winner for the 'self chain' (if self is not root)
		nDelegate, err = c.FSM.LotteryWinner(c.Config.ChainId)
		// if an error occurred
		if err != nil {
			// log the error and continue
			c.log.Warnf("An error occurred choosing a nested-delegate lottery winner: %s", err.Error())
		}
		// add the nested delegate as a reward recipient, subtracting share away from the proposer
		c.AddRewardRecipient(proposer, nDelegate, results, isOwnRoot, rootChainId)
	}
	// finally add the proposer at the end after ensuring their proper percent
	c.AddPaymentPercent(proposer, results, isOwnRoot, rootChainId)
	// exit
	return
}

// AddRewardRecipient() adds a reward recipient to the list of reward recipients in the certificate result
func (c *Controller) AddRewardRecipient(proposer, toAdd *lib.LotteryWinner, results *lib.CertificateResult, isOwnRoot bool, rootChainId uint64) {
	// skip any nil recipient
	if toAdd == nil || len(toAdd.Winner) == 0 {
		// exit
		return
	}
	// ensure the new recipient's cut doesn't underflow the proposers cut
	if proposer.Cut < toAdd.Cut {
		c.log.Warnf("Not enough proposer cut for winner")
		// exit
		return
	}
	// calculate new proposer cut
	proposer.Cut -= toAdd.Cut
	// add the payment percent for the recipient
	c.AddPaymentPercent(toAdd, results, isOwnRoot, rootChainId)
}

// AddPaymentPercent() adds a payment percent to the certificate result for both the root chain and nested chain ids (if different)
func (c *Controller) AddPaymentPercent(toAdd *lib.LotteryWinner, results *lib.CertificateResult, isOwnRoot bool, rootChainId uint64) {
	// add the payment percent for the participant for the 'root chain id'
	c.addPaymentPercent(toAdd, results, rootChainId)
	// if this chain is not its own root (nested chain)
	if !isOwnRoot {
		// add the payment percent for the participant for the 'nested chain id'
		c.addPaymentPercent(toAdd, results, c.Config.ChainId)
	}
}

// addPaymentPercent() is a helper function to add a payment percent to the certificate result
func (c *Controller) addPaymentPercent(toAdd *lib.LotteryWinner, results *lib.CertificateResult, chainId uint64) {
	// check if the winner's address is already in the reward recipients list for the root chain
	// if found, update their reward percentage - else add them as a new recipient
	if !slices.ContainsFunc(results.RewardRecipients.PaymentPercents, func(pp *lib.PaymentPercents) (has bool) {
		// if the address and chain id matches
		if bytes.Equal(pp.Address, toAdd.Winner) && pp.ChainId == chainId {
			// mark as found
			has = true
			// increase their reward percentage by 'cut'
			pp.Percent += toAdd.Cut
		}
		return
	}) {
		// if the winner is not found in the list, add them as a new recipient
		results.RewardRecipients.PaymentPercents = append(results.RewardRecipients.PaymentPercents,
			&lib.PaymentPercents{Address: toAdd.Winner, Percent: toAdd.Cut, ChainId: chainId})
	}
}

// HandleSwaps() handles the 'buy' side of the sell orders
func (c *Controller) HandleSwaps(blockResult *lib.BlockResult, results *lib.CertificateResult, rootChainHeight uint64) {
	// parse the last block for 'lock orders'
	lockOrders := c.FSM.ParseLockOrders(blockResult)
	// get orders from the root-Chain
	orders, err := c.LoadRootChainOrderBook(rootChainHeight)
	// if an error occurred while loading the orders
	if err != nil {
		// exit without handling
		return
	}
	// process the root chain order book against the state
	closeOrders, resetOrders := c.FSM.ProcessRootChainOrderBook(orders, blockResult)
	// add the orders to the certificate result - truncating the 'lock orders' for defensive spam protection
	results.Orders = &lib.Orders{
		LockOrders:  lib.TruncateSlice(lockOrders, 1000),
		ResetOrders: resetOrders,
		CloseOrders: closeOrders,
	}
}

// CalculateSlashRecipients() calculates the addresses who receive slashes on the root-Chain
func (c *Controller) CalculateSlashRecipients(results *lib.CertificateResult, be *bft.ByzantineEvidence) {
	// define an error variable to be able to populate the Double signers directly
	var err lib.ErrorI
	// use the bft object to fill in the Byzantine Evidence
	results.SlashRecipients.DoubleSigners, err = c.Consensus.ProcessDSE(be.DSE.Evidence...)
	// if an error occurred
	if err != nil {
		// log the warning
		c.log.Warn(err.Error())
		// exit
		return
	}
	// if any slash recipients added
	if numSlashRecipients := len(results.SlashRecipients.DoubleSigners); numSlashRecipients != 0 {
		// log the addition
		c.log.Infof("Added %d slash recipients due to byzantine evidence", numSlashRecipients)
	}
}

// define how often the chain checkpoints with its root
const CheckpointFrequency = 100

// CalculateCheckpoint() calculates the checkpoint for the checkpoint as a service functionality
func (c *Controller) CalculateCheckpoint(blockResult *lib.BlockResult, results *lib.CertificateResult) {
	// each checkpoint frequency
	if blockResult.BlockHeader.Height%CheckpointFrequency == 0 {
		// log the addition to the certificate results
		c.log.Info("Checkpoint set in certificate results")
		// update the checkpoint in the certificate results
		results.Checkpoint = &lib.Checkpoint{
			Height:    blockResult.BlockHeader.Height,
			BlockHash: blockResult.BlockHeader.Hash,
		}
	}
}

// HandleRetired() checks if the committee is retiring and sets in the results accordingly
func (c *Controller) HandleRetired(results *lib.CertificateResult) {
	// get the governance params from the 'nested chain' FSM
	cons, err := c.FSM.GetParamsCons()
	// if an error occurred
	if err != nil {
		// log the error
		c.log.Error(err.Error())
		// exit
		return
	}
	// set the 'retired' field based on the retired consensus param not being 0
	results.Retired = cons.Retired != 0
}
