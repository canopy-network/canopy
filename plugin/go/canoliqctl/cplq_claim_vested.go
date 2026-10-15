package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/contract"
)

// cmdCPLQClaimVested submits MessageCPLQClaimVested, sweeping any newly-unlocked
// CPLQ from the caller's vesting schedules into their liquid balance.
func cmdCPLQClaimVested(args []string, gf globalFlags) error {
	if err := requireArgs(args, 1, commandUsages["cplq-claim-vested"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCPLQClaimVested{FromAddress: from}
	return submitAndReport(gf, signer, "cplq_claim_vested", msg, "cplq-claim-vested",
		fmt.Sprintf("from=%s", signer.Address))
}
