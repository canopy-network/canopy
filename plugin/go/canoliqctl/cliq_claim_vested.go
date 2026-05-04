package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdCLIQClaimVested submits MessageCLIQClaimVested, sweeping any newly-unlocked
// CLIQ from the caller's vesting schedules into their liquid balance.
func cmdCLIQClaimVested(args []string, gf globalFlags) error {
	if err := requireArgs(args, 1, commandUsages["cliq-claim-vested"]); err != nil {
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

	msg := &contract.MessageCLIQClaimVested{FromAddress: from}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cliq_claim_vested", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cliq-claim-vested submitted: tx_hash=%s from=%s\n", hash, signer.Address)
	return nil
}
