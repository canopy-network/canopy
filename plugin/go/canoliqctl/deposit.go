package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdDeposit submits MessageCanoliqDeposit, which mints cCNPY 1:1 (or at the
// current pool ratio) against the deposited CNPY.
func cmdDeposit(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["deposit"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	amount, err := parseUint(args[1], "amount")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCanoliqDeposit{
		FromAddress: from,
		Amount:      amount,
	}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "canoliq_deposit", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("deposit submitted: tx_hash=%s from=%s amount=%d\n", hash, signer.Address, amount)
	return nil
}
