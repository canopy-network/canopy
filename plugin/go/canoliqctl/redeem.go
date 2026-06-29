package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdRedeem submits MessageCanoliqRedeem, burning cCNPY and queueing a CNPY
// redemption that matures after the validator unstaking window.
func cmdRedeem(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["redeem"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	amount, err := parseUint(args[1], "ccnpy-amount")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCanoliqRedeem{
		FromAddress: from,
		CcnpyAmount: amount,
	}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "canoliq_redeem", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("redeem submitted: tx_hash=%s from=%s ccnpy=%d\n", hash, signer.Address, amount)
	return nil
}
