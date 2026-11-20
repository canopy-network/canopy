package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/contract"
)

// cmdCanoliqTransfer submits MessageCanoliqTransfer, moving liquid cCNPY
// between accounts.
func cmdCanoliqTransfer(args []string, gf globalFlags) error {
	if err := requireArgs(args, 3, commandUsages["canoliq-transfer"]); err != nil {
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

	msg := &contract.MessageCanoliqTransfer{
		FromAddress: from,
		ToAddress:   to,
		Amount:      amount,
	}
	return submitAndReport(gf, signer, "canoliq_transfer", msg, "canoliq-transfer",
		fmt.Sprintf("from=%s to=%s amount=%d", signer.Address, args[1], amount))
}
