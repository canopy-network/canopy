package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdClaim submits MessageCanoliqClaimRedemption, withdrawing matured CNPY
// from a queued redemption back to the user's account.
func cmdClaim(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["claim"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	id, err := parseUint(args[1], "redemption-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCanoliqClaimRedemption{
		FromAddress:  from,
		RedemptionId: id,
	}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "canoliq_claim_redemption", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("claim submitted: tx_hash=%s from=%s redemption_id=%d\n", hash, signer.Address, id)
	return nil
}
