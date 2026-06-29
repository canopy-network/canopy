package main

import (
	"fmt"
	"strings"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdCPLQStake submits MessageCPLQStake, locking liquid CPLQ into a stake
// record that confers governance weight on proposals created from this
// height onward. The optional --lock flag commits the stake to a vote-escrow
// tier (T2) for a higher voting multiplier + reward boost.
func cmdCPLQStake(args []string, gf globalFlags) error {
	rest, lockTier, err := parseLockFlag(args)
	if err != nil {
		return err
	}
	if err := requireArgs(rest, 2, commandUsages["cplq-stake"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, rest[0], gf.password)
	if err != nil {
		return err
	}
	amount, err := parseUint(rest[1], "amount")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCPLQStake{FromAddress: from, Amount: amount, LockTier: lockTier}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_stake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cplq-stake submitted: tx_hash=%s from=%s amount=%d lock=%s\n",
		hash, signer.Address, amount, strings.ToLower(strings.TrimPrefix(lockTier.String(), "LOCK_")))
	return nil
}

// parseLockFlag pulls an optional "--lock <tier>" out of args and returns the
// remaining positionals plus the resolved LockTier (default LOCK_NONE).
func parseLockFlag(args []string) (positional []string, tier contract.LockTier, err error) {
	tier = contract.LockTier_LOCK_NONE
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--lock":
			if i+1 >= len(args) {
				return nil, tier, fmt.Errorf("--lock requires a value (none|3m|6m|12m|24m)")
			}
			tier, err = parseLockTier(args[i+1])
			if err != nil {
				return nil, tier, err
			}
			i++
		case strings.HasPrefix(args[i], "--lock="):
			tier, err = parseLockTier(strings.TrimPrefix(args[i], "--lock="))
			if err != nil {
				return nil, tier, err
			}
		default:
			positional = append(positional, args[i])
		}
	}
	return positional, tier, nil
}

// parseLockTier maps a CLI lock token to a LockTier enum value.
func parseLockTier(s string) (contract.LockTier, error) {
	switch strings.ToLower(s) {
	case "", "none":
		return contract.LockTier_LOCK_NONE, nil
	case "3m":
		return contract.LockTier_LOCK_3M, nil
	case "6m":
		return contract.LockTier_LOCK_6M, nil
	case "12m":
		return contract.LockTier_LOCK_12M, nil
	case "24m":
		return contract.LockTier_LOCK_24M, nil
	default:
		return contract.LockTier_LOCK_NONE, fmt.Errorf("invalid --lock %q (want none|3m|6m|12m|24m)", s)
	}
}

// cmdCPLQUnstake submits MessageCPLQUnstake, debiting the staker's record and
// queueing an unbond entry that matures after CplqUnstakingBlocks.
func cmdCPLQUnstake(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["cplq-unstake"]); err != nil {
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

	msg := &contract.MessageCPLQUnstake{FromAddress: from, Amount: amount}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_unstake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cplq-unstake submitted: tx_hash=%s from=%s amount=%d\n", hash, signer.Address, amount)
	return nil
}

// cmdCPLQClaimUnstake submits MessageCPLQClaimUnstake, returning matured
// unstaked CPLQ to the staker's liquid balance.
func cmdCPLQClaimUnstake(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["cplq-claim-unstake"]); err != nil {
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

	msg := &contract.MessageCPLQClaimUnstake{FromAddress: from, UnstakeId: id}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_claim_unstake", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("cplq-claim-unstake submitted: tx_hash=%s from=%s unstake_id=%d\n", hash, signer.Address, id)
	return nil
}
