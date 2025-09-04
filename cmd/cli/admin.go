package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/canopy-network/canopy/cmd/rpc"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "admin only operations for the node",
}

var (
	pwd             string
	nick            string
	data            string
	fee             uint64
	delegate        bool
	earlyWithdrawal bool
	sim             bool
)

func init() {
	rootCmd.PersistentFlags().StringVar(&pwd, "password", "", "input a private key password (not recommended)")
	rootCmd.PersistentFlags().StringVar(&nick, "nickname", "", "input nickname for key")
	adminCmd.PersistentFlags().BoolVar(&sim, "simulate", false, "simulate won't submit a transaction, rather it will print the json of the transaction that would've been submitted")
	adminCmd.PersistentFlags().Uint64Var(&fee, "fee", 0, "custom fee, by default will use the minimum fee")
	txStakeCmd.PersistentFlags().BoolVar(&delegate, "delegate", false, "delegate tokens to committee(s) only without actual validator operation")
	txEditStakeCmd.PersistentFlags().BoolVar(&delegate, "delegate", false, "delegate tokens to committee(s) only without actual validator operation")
	txStakeCmd.PersistentFlags().BoolVar(&earlyWithdrawal, "early-withdrawal", false, "immediately withdrawal any rewards (with penalty) directly to output address instead of auto-compounding directly to stake")
	txEditStakeCmd.PersistentFlags().BoolVar(&earlyWithdrawal, "early-withdrawal", false, "immediately withdrawal any rewards (with penalty) directly to output address instead of auto-compounding directly to stake")
	txCreateOrderCmd.PersistentFlags().StringVar(&data, "data", "", "data for create order")
	adminCmd.AddCommand(ksCmd)
	adminCmd.AddCommand(ksNewKeyCmd)
	adminCmd.AddCommand(ksImportCmd)
	adminCmd.AddCommand(ksImportRawCmd)
	adminCmd.AddCommand(ksDeleteCmd)
	adminCmd.AddCommand(ksGetCmd)
	adminCmd.AddCommand(txSendCmd)
	adminCmd.AddCommand(txStakeCmd)
	adminCmd.AddCommand(txEditStakeCmd)
	adminCmd.AddCommand(txUnstakeCmd)
	adminCmd.AddCommand(txPauseCmd)
	adminCmd.AddCommand(txUnpauseCmd)
	adminCmd.AddCommand(txChangeParamCmd)
	adminCmd.AddCommand(txDAOTransferCmd)
	adminCmd.AddCommand(txSubsidyCmd)
	adminCmd.AddCommand(txCreateOrderCmd)
	adminCmd.AddCommand(txEditOrderCmd)
	adminCmd.AddCommand(txDeleteOrderCmd)
	adminCmd.AddCommand(txDexLimitOrderCmd)
	adminCmd.AddCommand(txDexLiquidityDepositCmd)
	adminCmd.AddCommand(txDexLiquidityWithdrawCmd)
	adminCmd.AddCommand(txLockOrderCmd)
	adminCmd.AddCommand(txStartPollCmd)
	adminCmd.AddCommand(approveTxVotePoll)
	adminCmd.AddCommand(rejectTxVotePoll)
	adminCmd.AddCommand(resourceUsageCmd)
	adminCmd.AddCommand(peerInfoCmd)
	adminCmd.AddCommand(peerBookCmd)
	adminCmd.AddCommand(consensusInfoCmd)
	adminCmd.AddCommand(configCmd)
	adminCmd.AddCommand(logsCmd)
	adminCmd.AddCommand(approveProposalCmd)
	adminCmd.AddCommand(rejectProposalCmd)
	adminCmd.AddCommand(deleteVoteCmd)
}

var (
	ksCmd = &cobra.Command{
		Use:   "ks",
		Short: "query the keystore of the node",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Keystore())
		},
	}

	ksNewKeyCmd = &cobra.Command{
		Use:   "ks-new-key",
		Short: "add a new key to the keystore of the node",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreNewKey(getPassword(), getNickname()))
		},
	}

	ksImportCmd = &cobra.Command{
		Use:   "ks-import <address> <encrypted-pk-json>",
		Short: "add a new key to the keystore of the node using the encrypted private key",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ptr := new(crypto.EncryptedPrivateKey)
			if err := lib.UnmarshalJSON([]byte(args[1]), ptr); err != nil {
				l.Fatal(err.Error())
			}
			writeToConsole(client.KeystoreImport(argGetAddr(args[0]), getNickname(), *ptr))
		},
	}

	ksImportRawCmd = &cobra.Command{
		Use:   "ks-import-raw <private-key>",
		Short: "add a new key to the keystore of the node using the raw private key",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreImportRaw(args[0], getPassword(), getNickname()))
		},
	}

	ksDeleteCmd = &cobra.Command{
		Use:   "ks-delete <address or nickname>",
		Short: "delete the key associated with the address or nickname from the keystore",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreDelete(argGetAddrOrNickname(args[0])))
		},
	}

	ksGetCmd = &cobra.Command{
		Use:   "ks-get <address or nickname>",
		Short: "query the key group associated with the address or nickname",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreGet(argGetAddrOrNickname(args[0]), getPassword()))
		},
	}

	txSendCmd = &cobra.Command{
		Use:     "tx-send <address or nickname> <to-address> <amount> --fee=10000 --simulate=true",
		Short:   "send an amount to another address",
		Example: "tx-send dfd3c8dff19da7682f7fe5fde062c813b55c9eee eed6c9dff19da7682f7fe5fde062c813b42c7abc 10000",
		Args:    cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxSend(argGetAddrOrNickname(args[0]), argGetAddr(args[1]), uint64(argToInt(args[2])), getPassword(), !sim, fee))
		},
	}

	txStakeCmd = &cobra.Command{
		Use:     "tx-stake <address or nickname> <net-address> <amount> <committees> <output> <signer address or nickname> --delegated --early-withdrawal --fee=10000 --simulate=true",
		Short:   "stake a validator",
		Long:    "tx-stake <address that signs blocks and operates the validators> <url where the node hosted> <the amount to be staked> <comma separated list of chainIds> <address for rewards> <signer address>--delegated --early-withdrawal  --fee=10000 --simulate=true",
		Example: "tx-stake dfd3c8dff19da7682f7fe5fde062c813b55c9eee https://canopy-rocks.net:9000 100000000 0,21,22 abc3c8dff19da7682f7fe5fde062c813b55c9abc dfd3c8dff19da7682f7fe5fde062c813b55c9eee",
		Args:    cobra.MinimumNArgs(6),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxStake(argGetAddrOrNickname(args[0]), args[1], uint64(argToInt(args[2])), argToCommittees(args[3]), argGetAddr(args[4]), argGetAddrOrNickname(args[5]), delegate, earlyWithdrawal, getPassword(), !sim, fee))
		},
	}

	txEditStakeCmd = &cobra.Command{
		Use:     "tx-edit-stake <address or nickname> <net-address> <amount> <committees> <output> <signer address or nickname> --delegated --early-withdrawal --fee=10000 --simulate=true",
		Short:   "edit-stake an active validator. Use the existing value to not edit a field",
		Long:    "tx-edit-stake <address that signs blocks and operates the validators> <url where the node hosted> <the amount to be staked> <comma separated list of chainIds> <address for rewards> <address for rewards> <signer address> --delegated --early-withdrawal  --fee=10000 --simulate=true",
		Example: "tx-edit-stake dfd3c8dff19da7682f7fe5fde062c813b55c9eee https://canopy-rocks.net:9001 100000001 0,21,22 abc3c8dff19da7682f7fe5fde062c813b55c9abc dfd3c8dff19da7682f7fe5fde062c813b55c9eee",
		Args:    cobra.MinimumNArgs(6),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxEditStake(argGetAddrOrNickname(args[0]), args[1], uint64(argToInt(args[2])), argToCommittees(args[3]), argGetAddr(args[4]), argGetAddrOrNickname(args[5]), delegate, earlyWithdrawal, getPassword(), !sim, fee))
		},
	}

	txUnstakeCmd = &cobra.Command{
		Use:     "tx-unstake <address or nickname> <signer address or nickname>  --fee=10000 --simulate=true",
		Short:   "begin unstaking an active validator; may take some time to fully unstake",
		Example: "tx-unstake dfd3c8dff19da7682f7fe5fde062c813b55c9eee dfd3c8dff19da7682f7fe5fde062c813b55c9eee",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxUnstake(argGetAddrOrNickname(args[0]), argGetAddrOrNickname(args[1]), getPassword(), !sim, fee))
		},
	}

	txPauseCmd = &cobra.Command{
		Use:     "tx-pause <address or nickname> <signer address or nickname>  --fee=10000 --simulate=true",
		Short:   "pause an active validator",
		Example: "tx-pause dfd3c8dff19da7682f7fe5fde062c813b55c9eee dfd3c8dff19da7682f7fe5fde062c813b55c9eee",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxPause(argGetAddrOrNickname(args[0]), argGetAddrOrNickname(args[1]), getPassword(), !sim, fee))
		},
	}

	txUnpauseCmd = &cobra.Command{
		Use:     "tx-unpause <address or nickname> <signer address or nickname> --fee=10000 --simulate=true",
		Short:   "unpause a paused validator",
		Example: "tx-unpause dfd3c8dff19da7682f7fe5fde062c813b55c9eee dfd3c8dff19da7682f7fe5fde062c813b55c9eee",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxUnpause(argGetAddrOrNickname(args[0]), argGetAddrOrNickname(args[1]), getPassword(), !sim, fee))
		},
	}

	txChangeParamCmd = &cobra.Command{
		Use:   "tx-change-param <address or nickname> <param-space> <param-key> <param-value> <proposal-start-block> <proposal-end-block> --fee=10000 --simulate=true",
		Short: "propose a governance parameter change - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(6),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxChangeParam(argGetAddrOrNickname(args[0]), args[1], args[2], args[3], uint64(argToInt(args[4])), uint64(argToInt(args[5])), getPassword(), !sim, fee))
		},
	}

	txDAOTransferCmd = &cobra.Command{
		Use:   "tx-dao-transfer <address or nickname> <amount> <proposal-start-block> <proposal-end-block> --fee=10000 --simulate=true",
		Short: "propose a treasury subsidy - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDaoTransfer(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), uint64(argToInt(args[3])), getPassword(), !sim, fee))
		},
	}

	txSubsidyCmd = &cobra.Command{
		Use:   "tx-subsidy <address or nickname> <amount> <chain-id> <opcode> --fee=10000 --simulate=true",
		Short: "subsidize the reward pool of a committee - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxSubsidy(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), args[3], getPassword(), !sim, fee))
		},
	}

	txCreateOrderCmd = &cobra.Command{
		Use:   "tx-create-order <address or nickname> <sell-amount> <receive-amount> <chain-id> <receive-address> --fee=10000 --data=<hex-data> --simulate=true",
		Short: "create a sell order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(5),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxCreateOrder(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), uint64(argToInt(args[3])), args[4], getPassword(), argStringToBytes(data), !sim, fee))
		},
	}

	txEditOrderCmd = &cobra.Command{
		Use:   "tx-edit-order <address or nickname> <sell-amount> <receive-amount> <order-id> <chain-id> <receive-address> --fee=10000 --simulate=true",
		Short: "edit an existing sell order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(6),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxEditOrder(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), args[3], uint64(argToInt(args[4])), args[5], getPassword(), !sim, fee))
		},
	}

	txDeleteOrderCmd = &cobra.Command{
		Use:   "tx-delete-order <address or nickname> <order-id> <chain-id> --fee=10000 --simulate=true",
		Short: "delete an existing sell order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDeleteOrder(argGetAddrOrNickname(args[0]), args[1], uint64(argToInt(args[2])), getPassword(), !sim, fee))
		},
	}

	txDexLimitOrderCmd = &cobra.Command{
		Use:   "tx-dex-limit-order <address or nickname> <amount> <receive-amount> <chain-id> --fee=10000 --simulate=true",
		Short: "create a new dex limit order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDexLimitOrder(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), uint64(argToInt(args[3])), getPassword(), !sim, fee))
		},
	}

	txDexLiquidityDepositCmd = &cobra.Command{
		Use:   "tx-dex-liquidity-deposit <address or nickname> <amount> <chain-id> --fee=10000 --simulate=true",
		Short: "executes a dex liquidity deposit - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDexLiquidityDeposit(argGetAddrOrNickname(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), getPassword(), !sim, fee))
		},
	}

	txDexLiquidityWithdrawCmd = &cobra.Command{
		Use:   "tx-dex-liquidity-withdraw <address or nickname> <percent> <chain-id> --fee=10000 --simulate=true",
		Short: "executes a dex liquidity withdraw - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDexLiquidityWithdraw(argGetAddrOrNickname(args[0]), argToInt(args[1]), uint64(argToInt(args[2])), getPassword(), !sim, fee))
		},
	}

	txLockOrderCmd = &cobra.Command{
		Use:   "tx-lock-order <address or nickname> <canopy-receive-address> <order-id> --fee=10000 --simulate=true",
		Short: "lock an existing sell order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxLockOrder(argGetAddrOrNickname(args[0]), argGetAddr(args[1]), args[2], getPassword(), !sim, fee))
		},
	}

	txCloseOrderCmd = &cobra.Command{
		Use:   "tx-close-order <address or nickname> <canopy-receive-address> <order-id> --fee=10000 --simulate=true",
		Short: "closes an existing locked sell order - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxCloseOrder(argGetAddrOrNickname(args[0]), args[1], getPassword(), !sim, fee))
		},
	}

	txStartPollCmd = &cobra.Command{
		Use:   "tx-start-poll <address or nickname> <poll-json> --fee=50000 --simulate=true",
		Short: "start a straw poll on canopy - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxStartPoll(argGetAddrOrNickname(args[0]), []byte(args[1]), getPassword(), !sim, fee))
		},
	}

	approveTxVotePoll = &cobra.Command{
		Use:   "approve-tx-vote-poll <address or nickname> <poll-json> --fee=10000 --simulate=true",
		Short: "approve vote on a straw poll for canopy - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxVotePoll(argGetAddrOrNickname(args[0]), []byte(args[1]), true, getPassword(), !sim, fee))
		},
	}

	rejectTxVotePoll = &cobra.Command{
		Use:   "approve-tx-vote-poll <address or nickname> <poll-json> --fee=10000 --simulate=true",
		Short: "reject vote on a straw poll for canopy - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxVotePoll(argGetAddrOrNickname(args[0]), []byte(args[1]), false, getPassword(), !sim, fee))
		},
	}

	resourceUsageCmd = &cobra.Command{
		Use:   "resource-usage",
		Short: "get node resource usage",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.ResourceUsage())
		},
	}

	peerInfoCmd = &cobra.Command{
		Use:   "peer-info",
		Short: "get node peers",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.PeerInfo())
		},
	}

	peerBookCmd = &cobra.Command{
		Use:   "peer-book",
		Short: "get node peer book",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.PeerBook())
		},
	}

	consensusInfoCmd = &cobra.Command{
		Use:   "consensus-info",
		Short: "get node consensus info",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.PeerInfo())
		},
	}

	configCmd = &cobra.Command{
		Use:   "config",
		Short: "get node configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Config())
		},
	}

	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "get node logs",
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.Logs())
		},
	}

	approveProposalCmd = &cobra.Command{
		Use:   "proposal-approve <proposal-json>",
		Short: "add vote approval for a governance proposal. If a validator this is how the node will poll and vote",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.AddVote([]byte(args[0]), true))
		},
	}

	rejectProposalCmd = &cobra.Command{
		Use:   "proposal-reject <proposal-json>",
		Short: "add vote rejection for a governance proposal. If a validator this is how the node will poll and vote",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.AddVote([]byte(args[0]), false))
		},
	}

	deleteVoteCmd = &cobra.Command{
		Use:   "proposal-delete-vote <proposal-hash>",
		Short: "delete a vote for a governance proposal",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.DelVote(args[0]))
		},
	}
)

func writeTxResultToConsole(hash *string, tx json.RawMessage, e lib.ErrorI) {
	if sim {
		writeToConsole(tx, e)
	} else {
		var hashString string
		if hash != nil {
			hashString = *hash
		}
		writeToConsole(hashString, e)
	}
}

func argStringToBytes(data string) []byte {
	bz := common.FromHex(data)
	return bz
}

func argGetAddr(arg string) string {
	bz, err := lib.StringToBytes(arg)
	if err != nil {
		l.Fatalf("%s isn't a proper hex string: %s", arg, err.Error())
	}
	if len(bz) != crypto.AddressSize {
		l.Fatalf("%s is not a 20 byte address", arg)
	}
	return arg
}

func argGetAddrOrNickname(arg string) rpc.AddrOrNickname {
	bz, err := lib.StringToBytes(arg)
	if err != nil {
		return rpc.AddrOrNickname{
			Nickname: arg,
		}
	}
	if len(bz) != crypto.AddressSize {
		return rpc.AddrOrNickname{
			Nickname: arg,
		}
	}
	return rpc.AddrOrNickname{
		Address: arg,
	}
}

func argToCommittees(arg string) string {
	if _, err := rpc.StringToCommittees(arg); err != nil {
		l.Fatal(err.Error())
	}
	return arg
}

func getNickname() string {
	if nick == "" {
		fmt.Println("Enter nickname:")
		_, err := fmt.Scanln(&nick)
		if err != nil {
			fmt.Println("Error reading input:", err)
			return ""
		}
	}
	return nick
}

func getPassword() string {
	if pwd == "" {
		fmt.Println("Enter password:")
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			l.Fatal(err.Error())
		}
		if password == nil {
			fmt.Println("Password cannot be empty")
			return getPassword()
		}
		return string(password)
	}
	return pwd
}
