package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdCPLQTransfer submits MessageCPLQTransfer, moving liquid (already-vested)
// CPLQ between accounts.
func cmdCPLQTransfer(args []string, gf globalFlags) error {
	if err := requireArgs(args, 3, commandUsages["cplq-transfer"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	to, err := addrFromHex(args[1])
	if err != nil {
		return err
	}
	amount, err := parseUint(args[2], "amount")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCPLQTransfer{
		FromAddress: from,
		ToAddress:   to,
		Amount:      amount,
	}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_transfer", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cplq-transfer submitted: tx_hash=%s from=%s to=%s amount=%d\n", hash, signer.Address, args[1], amount)
	return nil
}
