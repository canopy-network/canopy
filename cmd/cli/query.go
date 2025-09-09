package cli

import (
	"strconv"
	"strings"

	"github.com/canopy-network/canopy/lib"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "query the blockchain rpc",
}

var (
	height, startHeight, pageNumber, perPage, committee, unstaking, delegated, paused = uint64(0), uint64(0), 0, 0, uint64(0), "", "", ""
)

func init() {
	queryCmd.PersistentFlags().Uint64Var(&height, "height", 0, "historical height for the query, 0 is latest")
	queryCmd.PersistentFlags().Uint64Var(&height, "start-height", 0, "starting height for queries with a range")
	queryCmd.PersistentFlags().IntVar(&pageNumber, "page-number", 0, "page number on a paginated call")
	queryCmd.PersistentFlags().IntVar(&perPage, "per-page", 0, "number of items per page on a paginated call")
	queryCmd.PersistentFlags().Uint64Var(&committee, "committee", 0, "filter validators by chain id")
	queryCmd.PersistentFlags().StringVar(&unstaking, "unstaking", "", "yes = only unstaking validators, no = only non-unstaking validators")
	queryCmd.PersistentFlags().StringVar(&paused, "paused", "", "yes = only paused validators, no = only unpaused validators")
	queryCmd.PersistentFlags().StringVar(&delegated, "delegated", "", "yes = only delegated validators, no = only non-delegated validators")
	queryCmd.AddCommand(heightCmd)
	queryCmd.AddCommand(accountCmd)
	queryCmd.AddCommand(accountsCmd)
	queryCmd.AddCommand(poolCmd)
	queryCmd.AddCommand(poolsCmd)
	queryCmd.AddCommand(validatorCmd)
	queryCmd.AddCommand(validatorsCmd)
	queryCmd.AddCommand(committeeCmd)
	queryCmd.AddCommand(committeeDataCmd)
	queryCmd.AddCommand(committeesDataCmd)
	queryCmd.AddCommand(subsidizedCommitteeCmd)
	queryCmd.AddCommand(retiredCommitteeCmd)
	queryCmd.AddCommand(orderCmd)
	queryCmd.AddCommand(ordersCmd)
	queryCmd.AddCommand(nonSignersCmd)
	queryCmd.AddCommand(paramsCmd)
	queryCmd.AddCommand(supplyCmd)
	queryCmd.AddCommand(stateCmd)
	queryCmd.AddCommand(stateDiffCmd)
	queryCmd.AddCommand(certCmd)
	queryCmd.AddCommand(blkByHeightCmd)
	queryCmd.AddCommand(blkByHashCmd)
	queryCmd.AddCommand(blocksCmd)
	queryCmd.AddCommand(txsByHeightCmd)
	queryCmd.AddCommand(txsBySenderCmd)
	queryCmd.AddCommand(txsByRecCmd)
	queryCmd.AddCommand(txByHashCmd)
	queryCmd.AddCommand(pendingTxsCmd)
	queryCmd.AddCommand(proposalsCmd)
	queryCmd.AddCommand(pollCmd)
	queryCmd.AddCommand(lastProposersCmd)
	queryCmd.AddCommand(minimumEvidenceHeightCmd)
	queryCmd.AddCommand(isValidDoubleSignerCmd)
	queryCmd.AddCommand(checkpointCmd)
	queryCmd.AddCommand(doubleSignersCmd)
	queryCmd.AddCommand(delegateLotteryCmd)
	queryCmd.AddCommand(rootChainInfoCmd)
	queryCmd.AddCommand(validatorSetCmd)
	queryCmd.AddCommand(dexBatchCmd)
	queryCmd.AddCommand(nextDexBatchCmd)
}

var (
	heightCmd = &cobra.Command{
		Use:   "height",
		Short: "query the height of the blockchain",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Height())
		},
	}

	accountCmd = &cobra.Command{
		Use:   "account <address> --height=1",
		Short: "query an account on the blockchain",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Account(height, args[0]))
		},
	}

	accountsCmd = &cobra.Command{
		Use:   "accounts --height=1 --per-page=10 --page-number=1",
		Short: "query all accounts on the blockchain",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Accounts(getPaginatedArgs()))
		},
	}

	poolCmd = &cobra.Command{
		Use:   "pool <chain_id> --height=1",
		Short: "query a pool on the blockchain",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Pool(getPoolArgs(args)))
		},
	}

	poolsCmd = &cobra.Command{
		Use:   "pools --height=1 --per-page=10 --page-number=1",
		Short: "query all pools on the blockchain",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Pools(getPaginatedArgs()))
		},
	}

	validatorCmd = &cobra.Command{
		Use:   "validator <address> --height=1",
		Short: "query a validator on the blockchain",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Validator(height, args[0]))
		},
	}

	validatorsCmd = &cobra.Command{
		Use:   "validators --height=1 --per-page=10 --page-number=1 --committee=1 --unstaking=yes --paused=no",
		Short: "query all validators on the blockchain",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Validators(getFilterArgs()))
		},
	}

	committeeCmd = &cobra.Command{
		Use:   "committee <chain_id> --height=1 --per-page=10 --page-number=1",
		Short: "query committee members",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			h, params := getPaginatedArgs()
			writeToConsole(client.Committee(h, uint64(argToInt(args[0])), params))
		},
	}

	committeeDataCmd = &cobra.Command{
		Use:   "committee-data <chain_id> --height=1",
		Short: "query the chain metadata for a committee",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.CommitteeData(height, uint64(argToInt(args[0]))))
		},
	}

	committeesDataCmd = &cobra.Command{
		Use:   "committees-data --height=1",
		Short: "query the chain metadata for all committees",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.CommitteesData(height))
		},
	}

	subsidizedCommitteeCmd = &cobra.Command{
		Use:   "subsidized-committees --height=1",
		Short: "query a list of committees that are subsidized",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.SubsidizedCommittees(height))
		},
	}

	retiredCommitteeCmd = &cobra.Command{
		Use:   "retired-committees --height=1",
		Short: "query a list of retired committees",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.RetiredCommittees(height))
		},
	}

	orderCmd = &cobra.Command{
		Use:   "order <order_id> --height=1 --committee=1",
		Short: "query a specific sell order",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Order(height, args[0], committee))
		},
	}

	ordersCmd = &cobra.Command{
		Use:   "orders --height=1 --committee=1",
		Short: "query all sell orders for a committee",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Orders(height, committee))
		},
	}

	nonSignersCmd = &cobra.Command{
		Use:   "non-signers --height=1",
		Short: "query all bft non signing validators and their non-sign counter",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.NonSigners(height))
		},
	}

	paramsCmd = &cobra.Command{
		Use:   "params --height=1",
		Short: "query all governance params",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Params(height))
		},
	}

	supplyCmd = &cobra.Command{
		Use:   "supply --height=1",
		Short: "query the blockchain token supply",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Supply(height))
		},
	}

	stateCmd = &cobra.Command{
		Use:   "state --height=1",
		Short: "query the blockchain world state",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.State(height))
		},
	}

	stateDiffCmd = &cobra.Command{
		Use:   "state-diff --start-height=1 --height=2",
		Short: "query the blockchain state difference between two heights",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.StateDiff(startHeight, height))
		},
	}

	certCmd = &cobra.Command{
		Use:   "certificate --height=1",
		Short: "query a quorum certificate for a height",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.CertByHeight(height))
		},
	}

	blkByHeightCmd = &cobra.Command{
		Use:   "block --height=1",
		Short: "query a block at a height",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.BlockByHeight(height))
		},
	}

	blkByHashCmd = &cobra.Command{
		Use:   "block-by-hash <hash>",
		Short: "query a block with a hash",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.BlockByHash(args[0]))
		},
	}

	blocksCmd = &cobra.Command{
		Use:   "blocks --per-page=10 --page-number=1",
		Short: "query blocks from the blockchain",
		Run: func(cmd *cobra.Command, args []string) {
			_, p := getPaginatedArgs()
			writeToConsole(client.Blocks(p))
		},
	}

	txsByHeightCmd = &cobra.Command{
		Use:   "txs --height=1 --per-page=10 --page-number=1",
		Short: "query txs at a certain height",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.TransactionsByHeight(getPaginatedArgs()))
		},
	}

	txsBySenderCmd = &cobra.Command{
		Use:   "txs-by-sender <address> --per-page=10 --page-number=1",
		Short: "query txs from a sender address",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, p := getPaginatedArgs()
			writeToConsole(client.TransactionsBySender(args[0], p))
		},
	}

	txsByRecCmd = &cobra.Command{
		Use:   "txs-by-rec <address> --per-page=10 --page-number=1",
		Short: "query txs from a recipient address",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, p := getPaginatedArgs()
			writeToConsole(client.TransactionsByRecipient(args[0], p))
		},
	}

	txByHashCmd = &cobra.Command{
		Use:   "tx <hash>",
		Short: "query a transaction by its hash",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.TransactionByHash(args[0]))
		},
	}

	pendingTxsCmd = &cobra.Command{
		Use:   "pending-txs --per-page=10 --page-number=1",
		Short: "query a transactions in the local mempool but not yet included in a block",
		Run: func(cmd *cobra.Command, args []string) {
			_, p := getPaginatedArgs()
			writeToConsole(client.Pending(p))
		},
	}

	proposalsCmd = &cobra.Command{
		Use:   "proposals",
		Short: "query the nodes votes on governance proposals",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Proposals())
		},
	}

	pollCmd = &cobra.Command{
		Use:   "poll",
		Short: "query the nodes polling results on governance proposals",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Poll())
		},
	}

	lastProposersCmd = &cobra.Command{
		Use:   "last-proposers --height=1",
		Short: "query the last proposers used in leader election",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.LastProposers(height))
		},
	}

	minimumEvidenceHeightCmd = &cobra.Command{
		Use:   "minimum-evidence-height --height=1",
		Short: "query the minimum BFT evidence height used to determine if BFT evidence is valid",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.MinimumEvidenceHeight(height))
		},
	}

	isValidDoubleSignerCmd = &cobra.Command{
		Use:   "valid-double-signer <address> --height=1",
		Short: "query if a double signer is valid at some height",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.IsValidDoubleSigner(height, args[0]))
		},
	}

	checkpointCmd = &cobra.Command{
		Use:   "checkpoint <id> --height=1",
		Short: "query a checkpoint",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Checkpoint(height, uint64(argToInt(args[0]))))
		},
	}

	doubleSignersCmd = &cobra.Command{
		Use:   "double-signers --height=1",
		Short: "query all indexed double signers at some height",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.DoubleSigners(height))
		},
	}

	delegateLotteryCmd = &cobra.Command{
		Use:   "delegate-lottery <chain-id> --height=1",
		Short: "query the winner of the delegate lottery",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Lottery(uint64(argToInt(args[0])), height))
		},
	}

	rootChainInfoCmd = &cobra.Command{
		Use:   "root-chain-info <chain-id> --height=1",
		Short: "query the base chain information needed to complete consensus",
		Long:  "query the base chain information needed to complete consensus: this is a local call so will only work if self is the root-chain",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.RootChainInfo(height, uint64(argToInt(args[0]))))
		},
	}

	validatorSetCmd = &cobra.Command{
		Use:   "validator-set <chain-id> --height=1",
		Short: "query the validator set for a committee at a certain height",
		Long:  "query the validator set for a committee at a certain height",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.ValidatorSet(height, uint64(argToInt(args[0]))))
		},
	}

	dexBatchCmd = &cobra.Command{
		Use:   "dex-batch <chain-id> <with-points> --height=1",
		Short: "query the locked dex batch at a certain height",
		Long:  "query the locked dex batch at a certain height",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.DexBatch(height, uint64(argToInt(args[0])), argToBool(args[1])))
		},
	}

	nextDexBatchCmd = &cobra.Command{
		Use:   "next-dex-batch <chain-id> <with-points> --height=1",
		Short: "query the next dex batch at a certain height",
		Long:  "query the next dex batch at a certain height",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.NextDexBatch(height, uint64(argToInt(args[0])), argToBool(args[1])))
		},
	}
)

func getPoolArgs(args []string) (h uint64, id uint64) {
	h = height
	id = uint64(argToInt(args[0]))
	return
}

func getPaginatedArgs() (h uint64, params lib.PageParams) {
	h = height
	params = lib.PageParams{
		PageNumber: pageNumber,
		PerPage:    perPage,
	}
	return
}

func getFilterArgs() (h uint64, params lib.PageParams, filters lib.ValidatorFilters) {
	h, params = getPaginatedArgs()
	switch {
	case strings.Contains(strings.ToLower(unstaking), "y"):
		filters.Unstaking = lib.FilterOption_MustBe
	case strings.Contains(strings.ToLower(unstaking), "n"):
		filters.Unstaking = lib.FilterOption_Exclude
	}
	switch {
	case strings.Contains(strings.ToLower(paused), "y"):
		filters.Paused = lib.FilterOption_MustBe
	case strings.Contains(strings.ToLower(paused), "n"):
		filters.Paused = lib.FilterOption_Exclude
	}
	switch {
	case strings.Contains(strings.ToLower(delegated), "y"):
		filters.Delegate = lib.FilterOption_MustBe
	case strings.Contains(strings.ToLower(delegated), "n"):
		filters.Delegate = lib.FilterOption_Exclude
	}
	filters.Committee = committee
	return
}

func argToInt(arg string) int {
	i, err := strconv.Atoi(arg)
	if err != nil {
		l.Fatal(err.Error())
	}
	return i
}

func argToBool(arg string) bool {
	value, err := strconv.ParseBool(arg)
	if err != nil {
		l.Fatal(err.Error())
	}
	return value
}
