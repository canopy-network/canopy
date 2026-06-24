package main

import (
	"fmt"
	"strings"

	"github.com/canopy-network/go-plugin/canoliqctl/internal"
	"github.com/canopy-network/go-plugin/contract"
)

// cmdVote submits MessageCPLQVote against an active proposal. Vote weight
// equals the staker's CPLQStake balance evaluated at proposal.creation_height,
// so post-creation stake increases carry zero weight.
//
// proposal-create is intentionally not implemented yet: its payload is a
// google.protobuf.Any wrapping one of three sub-types (param_change | buyback |
// treasury_spend), each with its own argument surface. Wiring those is a
// follow-up — for Phase 1.5 verification we only need votes on proposals
// authored elsewhere (in-process tests, hand-crafted JSON, or a future
// proposal-create subcommand).
func cmdVote(args []string, gf globalFlags) error {
	if err := requireArgs(args, 3, commandUsages["vote"]); err != nil {
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
	choice, err := parseVoteChoice(args[2])
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageCPLQVote{
		FromAddress: from,
		ProposalId:  proposalID,
		Choice:      choice,
	}
	hash, err := internal.SubmitPluginTx(gf.rpcURL, signer, "cplq_vote", msg, txParams(gf))
	if err != nil {
		return err
	}
	fmt.Printf("vote submitted: tx_hash=%s from=%s proposal_id=%d choice=%s\n",
		hash, signer.Address, proposalID, choice)
	return nil
}

func parseVoteChoice(s string) (contract.VoteChoice, error) {
	switch strings.ToLower(s) {
	case "yes", "y":
		return contract.VoteChoice_VOTE_YES, nil
	case "no", "n":
		return contract.VoteChoice_VOTE_NO, nil
	case "abstain", "a":
		return contract.VoteChoice_VOTE_ABSTAIN, nil
	default:
		return contract.VoteChoice_VOTE_UNKNOWN, fmt.Errorf("invalid choice %q (want yes|no|abstain)", s)
	}
}
