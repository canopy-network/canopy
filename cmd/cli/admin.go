package main

import (
	"encoding/json"
	"fmt"
	"github.com/ginchuco/ginchu/lib"
	"github.com/ginchuco/ginchu/lib/crypto"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"syscall"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "admin only operations for the node",
}

var (
	pwd string
	seq uint64
	fee uint64
	sim bool
)

func init() {
	adminCmd.PersistentFlags().StringVar(&pwd, "password", "", "input a private key password (not recommended)")
	adminCmd.PersistentFlags().Uint64Var(&seq, "sequence", 0, "if multiple transactions in the same block, the sequence must be incremented for each")
	adminCmd.PersistentFlags().BoolVar(&sim, "simulate", false, "simulate won't submit a transaction, rather it will print the json of the transaction that would've been submitted")
	adminCmd.PersistentFlags().Uint64Var(&fee, "fee", 0, "custom fee, by default will use the minimum fee")
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
			writeToConsole(client.KeystoreNewKey(getPassword()))
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
			writeToConsole(client.KeystoreImport(argGetAddr(args[0]), *ptr))
		},
	}

	ksImportRawCmd = &cobra.Command{
		Use:   "ks-import-raw <private-key>",
		Short: "add a new key to the keystore of the node using the raw private key",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreImportRaw(args[0], getPassword()))
		},
	}

	ksDeleteCmd = &cobra.Command{
		Use:   "ks-delete <address>",
		Short: "delete the key associated with the address from the keystore",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreDelete(argGetAddr(args[0])))
		},
	}

	ksGetCmd = &cobra.Command{
		Use:   "ks-get <address>",
		Short: "query the key group associated with the address",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeToConsole(client.KeystoreGet(argGetAddr(args[0]), getPassword()))
		},
	}

	txSendCmd = &cobra.Command{
		Use:   "tx-send <address> <to-address> <amount> --sequence=1 --fee=10000 --simulate=true",
		Short: "send an amount to another address",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxSend(argGetAddr(args[0]), argGetAddr(args[1]), uint64(argToInt(args[2])), getPassword(), !sim, seq, fee))
		},
	}

	txStakeCmd = &cobra.Command{
		Use:   "tx-stake <address> <net-address> <amount> <output> --sequence=1 --fee=10000 --simulate=true",
		Short: "stake a validator",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxStake(argGetAddr(args[0]), args[1], uint64(argToInt(args[2])), argGetAddr(args[3]), getPassword(), !sim, seq, fee))
		},
	}

	txEditStakeCmd = &cobra.Command{
		Use:   "tx-edit-stake <address> <net-address> <amount> <output> --sequence=1 --fee=10000 --simulate=true",
		Short: "edit-stake an active validator. Use the existing value to not edit a field",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxEditStake(argGetAddr(args[0]), args[1], uint64(argToInt(args[2])), argGetAddr(args[3]), getPassword(), !sim, seq, fee))
		},
	}

	txUnstakeCmd = &cobra.Command{
		Use:   "tx-unstake <address> --sequence=1 --fee=10000 --simulate=true",
		Short: "unstake an active validator",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxUnstake(argGetAddr(args[0]), getPassword(), !sim, seq, fee))
		},
	}

	txPauseCmd = &cobra.Command{
		Use:   "tx-pause <address> --sequence=1 --fee=10000 --simulate=true",
		Short: "pause an active validator",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxPause(argGetAddr(args[0]), getPassword(), !sim, seq, fee))
		},
	}

	txUnpauseCmd = &cobra.Command{
		Use:   "tx-unpause <address> --sequence=1 --fee=10000 --simulate=true",
		Short: "unpause a paused validator",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxUnpause(argGetAddr(args[0]), getPassword(), !sim, seq, fee))
		},
	}

	txChangeParamCmd = &cobra.Command{
		Use:   "tx-change-param <address> <param-space> <param-key> <param-value> <proposal-start-block> <proposal-end-block>--sequence=1 --fee=10000 --simulate=true",
		Short: "propose a governance parameter change - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(6),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxChangeParam(argGetAddr(args[0]), args[1], args[2], args[3], uint64(argToInt(args[4])), uint64(argToInt(args[5])), getPassword(), !sim, seq, fee))
		},
	}

	txDAOTransferCmd = &cobra.Command{
		Use:   "tx-dao-transfer <address> <amount> <proposal-start-block> <proposal-end-block>--sequence=1 --fee=10000 --simulate=true",
		Short: "propose a treasury subsidy - use the simulate flag to generate json only",
		Args:  cobra.MinimumNArgs(4),
		Run: func(cmd *cobra.Command, args []string) {
			writeTxResultToConsole(client.TxDaoTransfer(argGetAddr(args[0]), uint64(argToInt(args[1])), uint64(argToInt(args[2])), uint64(argToInt(args[3])), getPassword(), !sim, seq, fee))
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
		writeToConsole(hash, e)
	}
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

func getPassword() string {
	if pwd == "" {
		fmt.Println("Enter password:")
		password, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			l.Fatal(err.Error())
		}
		return string(password)
	}
	return pwd
}