package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/contract"
)

// cmdSpendExecute submits MessageDAOTreasurySpend, attempting to dispatch a
// treasury spend whose proposal has already passed. Above-threshold spends
// require the timelock to elapse and multisig approvals to be in place; this
// command can be invoked repeatedly until both conditions are met.
func cmdSpendExecute(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["spend-execute"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	proposalID, err := parseUint(args[1], "proposal-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageDAOTreasurySpend{FromAddress: from, ProposalId: proposalID}
	return submitAndReport(gf, signer, "dao_treasury_spend", msg, "spend-execute",
		fmt.Sprintf("from=%s proposal_id=%d", signer.Address, proposalID))
}

// cmdMultisigApprove submits MessageMultisigApprove for a queued treasury
// spend. Signer must appear in CanoliqParams.MultisigSigners or the plugin
// rejects the message.
func cmdMultisigApprove(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["multisig-approve"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	spendID, err := parseUint(args[1], "spend-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageMultisigApprove{FromAddress: from, SpendId: spendID}
	return submitAndReport(gf, signer, "multisig_approve", msg, "multisig-approve",
		fmt.Sprintf("signer=%s spend_id=%d", signer.Address, spendID))
}
