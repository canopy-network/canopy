package main

import (
	"fmt"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdCLIQStake submits MessageCLIQStake, locking liquid CLIQ into a stake
// record that confers governance weight on proposals created from this
// height onward.
func cmdCLIQStake(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["cliq-stake"]); err != nil {
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

	msg := &contract.MessageCLIQStake{FromAddress: from, Amount: amount}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cliq_stake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cliq-stake submitted: tx_hash=%s from=%s amount=%d\n", hash, signer.Address, amount)
	return nil
}

// cmdCLIQUnstake submits MessageCLIQUnstake, debiting the staker's record and
// queueing an unbond entry that matures after CliqUnstakingBlocks.
func cmdCLIQUnstake(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["cliq-unstake"]); err != nil {
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

	msg := &contract.MessageCLIQUnstake{FromAddress: from, Amount: amount}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cliq_unstake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cliq-unstake submitted: tx_hash=%s from=%s amount=%d\n", hash, signer.Address, amount)
	return nil
}

// cmdCLIQClaimUnstake submits MessageCLIQClaimUnstake, returning matured
// unstaked CLIQ to the staker's liquid balance.
func cmdCLIQClaimUnstake(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["cliq-claim-unstake"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	id, err := parseUint(args[1], "unstake-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCLIQClaimUnstake{FromAddress: from, UnstakeId: id}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cliq_claim_unstake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cliq-claim-unstake submitted: tx_hash=%s from=%s unstake_id=%d\n", hash, signer.Address, id)
	return nil
}
