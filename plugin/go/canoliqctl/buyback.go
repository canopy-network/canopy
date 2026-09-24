package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/contract"
)

// cmdBuybackExecute submits MessageBuybackExecute, idempotently running a
// buyback proposal that has already passed.
func cmdBuybackExecute(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["buyback-execute"]); err != nil {
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

	msg := &contract.MessageBuybackExecute{FromAddress: from, ProposalId: proposalID}
	return submitAndReport(gf, signer, "buyback_execute", msg, "buyback-execute",
		fmt.Sprintf("from=%s proposal_id=%d", signer.Address, proposalID))
}
