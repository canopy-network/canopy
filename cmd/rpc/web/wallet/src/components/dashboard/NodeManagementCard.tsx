import React, { useCallback, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  Lock,
  Pause,
  Play,
} from "lucide-react";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useAccountData } from "@/hooks/useAccountData";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useValidators } from "@/hooks/useValidators";
import { useDenom } from "@/hooks/useDenom";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";
import { getCanopySymbolByHash } from "@/lib/utils/canopySymbols";

const shortAddr = (address: string) =>
  `${address.substring(0, 8)}…${address.substring(address.length - 4)}`;

type SortColumn = "Account" | "Liquid" | "Staked" | "Status" | null;
type SortDirection = "asc" | "desc";

type ValidatorLike = {
  address: string;
  stakedAmount: number;
  paused?: boolean;
  unstaking?: boolean;
  delegate?: boolean;
  committees?: number[];
};

interface ProcessedAccount {
  address: string;
  nickname: string;
  liquidBalance: number;
  stakedBalance: number;
  status: "Liquid" | "Staked" | "Paused" | "Unstaking";
  validator: ValidatorLike | null;
}

const desktopRowCellClass =
  "px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]";
const actionButtonClass =
  "group rounded-lg border border-[#272729] p-2 transition-all duration-150 hover:border-white/15 hover:bg-[#272729]";

const getAccountStatusBadgeClass = (status: ProcessedAccount["status"]) => {
  return WALLET_BADGE_TONE;
};

const AccountStatusBadge = React.memo<{ label: ProcessedAccount["status"] }>(
  ({ label }) => (
    <span
      className={`${WALLET_BADGE_CLASS} leading-none transition-colors ${getAccountStatusBadgeClass(
        label,
      )}`}
    >
      {label}
    </span>
  ),
);

AccountStatusBadge.displayName = "AccountStatusBadge";

const LatestUpdated = React.memo<{ className?: string }>(({ className }) => (
  <div className={`flex items-center gap-2 lg:gap-4 ${className ?? ""}`}>
    <div className="relative inline-flex items-center gap-1 rounded-full border border-[rgba(53,205,72,0.30)] bg-[rgba(53,205,72,0.12)] px-2 py-0.5">
      <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
      <span className="text-sm leading-none text-[#35cd48]">Live</span>
    </div>
  </div>
));

LatestUpdated.displayName = "LatestUpdated";

const formatTokenAmount = (amount: number, factor: number, symbol: string) => (
  <>
    <span className="text-sm text-foreground tabular-nums">
      {(amount / factor).toLocaleString("en-US", {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })}
    </span>
    <span className="ml-1 text-xs text-muted-foreground/50">{symbol}</span>
  </>
);

const AccountRow = React.memo<{
  account: ProcessedAccount;
  index: number;
  factor: number;
  symbol: string;
  onSend: (address: string) => void;
  onReceive: (address: string) => void;
  onStake: (account: ProcessedAccount) => void;
  onPauseToggle: (account: ProcessedAccount, action: "pause" | "unpause") => void;
}>(({ account, index, factor, symbol, onSend, onReceive, onStake, onPauseToggle }) => {
  const { copyToClipboard } = useCopyToClipboard();

  const thirdAction = useMemo(() => {
    if (account.validator && !account.validator.delegate && !account.validator.unstaking) {
      if (account.validator.paused) {
        return {
          label: "Resume",
          description: "Resume validator activity after a pause.",
          icon: Play,
          onClick: () => onPauseToggle(account, "unpause"),
        };
      }

      if (account.stakedBalance > 0) {
        return {
          label: "Pause",
          description: "Temporarily pause validator activity.",
          icon: Pause,
          onClick: () => onPauseToggle(account, "pause"),
        };
      }
    }

    return {
      label: account.stakedBalance > 0 ? "Edit Stake" : "Stake",
      description:
        account.stakedBalance > 0
          ? "Adjust committees or update the current stake."
          : "Delegate funds from this account.",
      icon: Lock,
      onClick: () => onStake(account),
    };
  }, [account, onPauseToggle, onStake]);

  return (
    <motion.tr
      className="group"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, delay: index * 0.04 }}
    >
      <td
        className={desktopRowCellClass}
        style={{ borderTopLeftRadius: "10px", borderBottomLeftRadius: "10px" }}
      >
        <div className="flex items-center gap-3">
          <img
            src={getCanopySymbolByHash(account.address)}
            alt=""
            className="h-8 w-8 rounded-lg object-contain flex-shrink-0"
          />
          <div>
            <div className="text-sm font-medium text-foreground leading-tight">
              {account.nickname}
            </div>
            <div className="mt-0.5 flex items-center gap-1">
              <span className="text-[11px] text-muted-foreground leading-tight">
                {shortAddr(account.address)}
              </span>
              <button
                onClick={() => copyToClipboard(account.address, "Address")}
                className="rounded p-0.5 text-white/40 transition-colors hover:bg-[#272729] hover:text-white"
                aria-label="Copy address"
              >
                <Copy className="h-2.5 w-2.5" />
              </button>
            </div>
          </div>
        </div>
      </td>
      <td className={desktopRowCellClass}>
        <div>{formatTokenAmount(account.liquidBalance, factor, symbol)}</div>
      </td>
      <td className={desktopRowCellClass}>
        <div>{formatTokenAmount(account.stakedBalance, factor, symbol)}</div>
      </td>
      <td className={desktopRowCellClass}>
        <AccountStatusBadge label={account.status} />
      </td>
      <td
        className={desktopRowCellClass}
        style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
      >
        <div className="flex items-center gap-1.5">
          <ActionTooltip label="Send" description="Transfer funds from this account.">
            <button
              type="button"
              className={actionButtonClass}
              onClick={() => onSend(account.address)}
              aria-label="Send"
            >
              <ArrowUp className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
            </button>
          </ActionTooltip>
          <ActionTooltip
            label="Receive"
            description="Show this address for incoming funds."
          >
            <button
              type="button"
              className={actionButtonClass}
              onClick={() => onReceive(account.address)}
              aria-label="Receive"
            >
              <ArrowDown className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
            </button>
          </ActionTooltip>
          <ActionTooltip
            label={thirdAction.label}
            description={thirdAction.description}
          >
            <button
              type="button"
              className={actionButtonClass}
              onClick={thirdAction.onClick}
              aria-label={thirdAction.label}
            >
              <thirdAction.icon className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
            </button>
          </ActionTooltip>
        </div>
      </td>
    </motion.tr>
  );
});

AccountRow.displayName = "AccountRow";

const AccountMobileCard = React.memo<{
  account: ProcessedAccount;
  index: number;
  factor: number;
  symbol: string;
  onSend: (address: string) => void;
  onReceive: (address: string) => void;
  onStake: (account: ProcessedAccount) => void;
  onPauseToggle: (account: ProcessedAccount, action: "pause" | "unpause") => void;
}>(({ account, index, factor, symbol, onSend, onReceive, onStake, onPauseToggle }) => {
  const { copyToClipboard } = useCopyToClipboard();

  const thirdAction = useMemo(() => {
    if (account.validator && !account.validator.delegate && !account.validator.unstaking) {
      if (account.validator.paused) {
        return {
          label: "Resume",
          description: "Resume validator activity after a pause.",
          icon: Play,
          onClick: () => onPauseToggle(account, "unpause"),
        };
      }

      if (account.stakedBalance > 0) {
        return {
          label: "Pause",
          description: "Temporarily pause validator activity.",
          icon: Pause,
          onClick: () => onPauseToggle(account, "pause"),
        };
      }
    }

    return {
      label: account.stakedBalance > 0 ? "Edit Stake" : "Stake",
      description:
        account.stakedBalance > 0
          ? "Adjust committees or update the current stake."
          : "Delegate funds from this account.",
      icon: Lock,
      onClick: () => onStake(account),
    };
  }, [account, onPauseToggle, onStake]);

  return (
    <motion.div
      className="space-y-3 rounded-lg border border-[#272729] bg-[#1a1a1a] p-3.5"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.18, delay: index * 0.04 }}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <img
            src={getCanopySymbolByHash(account.address)}
            alt=""
            className="h-8 w-8 rounded-lg object-contain flex-shrink-0"
          />
          <div>
            <div className="text-sm font-medium text-foreground leading-tight">
              {account.nickname}
            </div>
            <div className="flex items-center gap-1">
              <span className="text-[11px] text-muted-foreground leading-tight">
                {shortAddr(account.address)}
              </span>
              <button
                onClick={() => copyToClipboard(account.address, "Address")}
                className="rounded p-0.5 text-white/40 transition-colors hover:bg-[#272729] hover:text-white"
                aria-label="Copy address"
              >
                <Copy className="h-2.5 w-2.5" />
              </button>
            </div>
          </div>
        </div>
        <AccountStatusBadge label={account.status} />
      </div>

      <div className="grid grid-cols-2 gap-3 border-t border-[#272729] pt-3">
        <div>
          <div className="mb-1 text-xs font-medium text-white/60">Liquid</div>
          <div className="text-sm text-foreground tabular-nums">
            {(account.liquidBalance / factor).toLocaleString("en-US", {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}{" "}
            {symbol}
          </div>
        </div>
        <div>
          <div className="mb-1 text-xs font-medium text-white/60">Staked</div>
          <div className="text-sm text-foreground tabular-nums">
            {(account.stakedBalance / factor).toLocaleString("en-US", {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}{" "}
            {symbol}
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1.5">
        <ActionTooltip label="Send" description="Transfer funds from this account.">
          <button
            type="button"
            className={actionButtonClass}
            onClick={() => onSend(account.address)}
            aria-label="Send"
          >
            <ArrowUp className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
          </button>
        </ActionTooltip>
        <ActionTooltip
          label="Receive"
          description="Show this address for incoming funds."
        >
          <button
            type="button"
            className={actionButtonClass}
            onClick={() => onReceive(account.address)}
            aria-label="Receive"
          >
            <ArrowDown className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
          </button>
        </ActionTooltip>
        <ActionTooltip label={thirdAction.label} description={thirdAction.description}>
          <button
            type="button"
            className={actionButtonClass}
            onClick={thirdAction.onClick}
            aria-label={thirdAction.label}
          >
            <thirdAction.icon className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
          </button>
        </ActionTooltip>
      </div>
    </motion.div>
  );
});

AccountMobileCard.displayName = "AccountMobileCard";

export const NodeManagementCard = React.memo((): JSX.Element => {
  const { accounts, loading: accountsLoading, selectedAccount, switchAccount } =
    useAccounts();
  const { balances, stakingData, loading: accountDataLoading } = useAccountData();
  const { data: validators = [], isLoading: validatorsLoading, error } =
    useValidators();
  const { openAction } = useActionModal();
  const { symbol, factor } = useDenom();

  const [sortColumn, setSortColumn] = useState<SortColumn>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");

  const isLoading = accountsLoading || accountDataLoading || validatorsLoading;

  const setActiveAccount = useCallback(
    (address: string) => {
      const account = accounts.find((entry) => entry.address === address);
      if (account && selectedAccount?.address !== address) {
        switchAccount(account.id);
      }
      return account ?? null;
    },
    [accounts, selectedAccount?.address, switchAccount],
  );

  const processedAccounts = useMemo((): ProcessedAccount[] => {
    const balanceByAddress = new Map(
      balances.map((entry) => [entry.address, Number(entry.amount || 0)]),
    );
    const stakingByAddress = new Map(
      stakingData.map((entry) => [entry.address, Number(entry.staked || 0)]),
    );
    const validatorByAddress = new Map(
      (validators as ValidatorLike[]).map((validator) => [validator.address, validator]),
    );

    return accounts
      .map((account) => {
        const validator = validatorByAddress.get(account.address) ?? null;
        const stakedBalance = stakingByAddress.get(account.address) ?? 0;
        const liquidBalance = balanceByAddress.get(account.address) ?? 0;

        let status: ProcessedAccount["status"] = "Liquid";
        if (validator?.unstaking) status = "Unstaking";
        else if (validator?.paused) status = "Paused";
        else if (stakedBalance > 0) status = "Staked";

        return {
          address: account.address,
          nickname: account.nickname || shortAddr(account.address),
          liquidBalance,
          stakedBalance,
          status,
          validator,
        };
      })
      .sort((a, b) => a.nickname.localeCompare(b.nickname));
  }, [accounts, balances, stakingData, validators]);

  const handleSort = useCallback((column: Exclude<SortColumn, null>) => {
    setSortColumn((currentColumn) => {
      if (currentColumn === column) {
        setSortDirection((currentDirection) =>
          currentDirection === "desc" ? "asc" : "desc",
        );
        return currentColumn;
      }
      setSortDirection("desc");
      return column;
    });
  }, []);

  const sortedAccounts = useMemo(() => {
    if (!sortColumn) return processedAccounts;

    const sorted = [...processedAccounts];
    sorted.sort((a, b) => {
      let comparison = 0;

      switch (sortColumn) {
        case "Account":
          comparison = a.nickname.localeCompare(b.nickname, undefined, {
            numeric: true,
            sensitivity: "base",
          });
          break;
        case "Liquid":
          comparison = a.liquidBalance - b.liquidBalance;
          break;
        case "Staked":
          comparison = a.stakedBalance - b.stakedBalance;
          break;
        case "Status":
          comparison = a.status.localeCompare(b.status, undefined, {
            numeric: true,
            sensitivity: "base",
          });
          break;
        default:
          comparison = 0;
      }

      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [processedAccounts, sortColumn, sortDirection]);

  const columns = useMemo(
    () => [
      { label: "Account", sortable: true },
      { label: "Liquid", sortable: true },
      { label: "Staked", sortable: true },
      { label: "Status", sortable: true },
      { label: "Actions", sortable: false },
    ],
    [],
  );

  const handleSend = useCallback(
    (address: string) => {
      setActiveAccount(address);
      openAction("send");
    },
    [openAction, setActiveAccount],
  );

  const handleReceive = useCallback(
    (address: string) => {
      setActiveAccount(address);
      openAction("receive");
    },
    [openAction, setActiveAccount],
  );

  const handleStake = useCallback(
    (account: ProcessedAccount) => {
      setActiveAccount(account.address);

      if (account.stakedBalance > 0) {
        openAction("stake", {
          titleOverride: "Edit Stake",
          prefilledData: {
            operator: account.address,
            selectCommittees: account.validator?.committees || [],
          },
        });
        return;
      }

      openAction("stake");
    },
    [openAction, setActiveAccount],
  );

  const handlePauseToggle = useCallback(
    (account: ProcessedAccount, action: "pause" | "unpause") => {
      setActiveAccount(account.address);
      openAction(action === "pause" ? "pauseValidator" : "unpauseValidator", {
        prefilledData: {
          validatorAddress: account.address,
          signerAddress: account.address,
        },
      });
    },
    [openAction, setActiveAccount],
  );

  const cardBase = "canopy-card p-5";
  const cardMotion = {
    initial: { opacity: 0, y: 12 },
    animate: { opacity: 1, y: 0 },
    transition: { duration: 0.35, delay: 0.28 },
  };

  if (isLoading) {
    return (
      <motion.div className={cardBase} {...cardMotion}>
        <LoadingState message="Loading accounts…" size="md" />
      </motion.div>
    );
  }

  if (error) {
    return (
      <motion.div className={cardBase} {...cardMotion}>
        <EmptyState
          icon="AlertCircle"
          title="Error loading accounts"
          description="There was a problem"
          size="md"
        />
      </motion.div>
    );
  }

  return (
    <motion.div className={cardBase} {...cardMotion}>
      <div className="mb-5 flex flex-col items-start justify-between gap-3 leading-none sm:flex-row sm:items-center sm:gap-4">
        <div className="flex items-center gap-2">
          <span className="wallet-card-title">All Accounts</span>
        </div>
        <LatestUpdated className="self-end sm:self-auto" />
      </div>

      <div className="hidden overflow-x-auto md:block">
        {sortedAccounts.length > 0 ? (
          <table
            className="w-full"
            style={{ tableLayout: "auto", borderCollapse: "separate", borderSpacing: "0 4px" }}
          >
            <thead>
              <tr>
                {columns.map(({ label, sortable }) => {
                  const isActive = sortColumn === label;

                  return (
                    <th
                      key={label}
                      className={`px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4 ${
                        sortable ? "cursor-pointer select-none hover:text-white/80" : ""
                      }`}
                      onClick={() =>
                        sortable ? handleSort(label as Exclude<SortColumn, null>) : undefined
                      }
                    >
                      <div className="flex items-center gap-1">
                        {label}
                        {sortable ? (
                          <span className="inline-flex">
                            {isActive ? (
                              sortDirection === "asc" ? (
                                <ChevronUp className="h-3 w-3" />
                              ) : (
                                <ChevronDown className="h-3 w-3" />
                              )
                            ) : (
                              <ChevronsUpDown className="h-3 w-3 opacity-40" />
                            )}
                          </span>
                        ) : null}
                      </div>
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {sortedAccounts.map((account, index) => (
                <AccountRow
                  key={account.address}
                  account={account}
                  index={index}
                  factor={factor}
                  symbol={symbol}
                  onSend={handleSend}
                  onReceive={handleReceive}
                  onStake={handleStake}
                  onPauseToggle={handlePauseToggle}
                />
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState
            icon="Wallet"
            title="No accounts found"
            description="Your accounts will appear here"
            size="sm"
          />
        )}
      </div>

      <div className="space-y-2.5 md:hidden">
        {sortedAccounts.length > 0 ? (
          sortedAccounts.map((account, index) => (
            <AccountMobileCard
              key={account.address}
              account={account}
              index={index}
              factor={factor}
              symbol={symbol}
              onSend={handleSend}
              onReceive={handleReceive}
              onStake={handleStake}
              onPauseToggle={handlePauseToggle}
            />
          ))
        ) : (
          <EmptyState
            icon="Wallet"
            title="No accounts found"
            description="Your accounts will appear here"
            size="sm"
          />
        )}
      </div>
    </motion.div>
  );
});

NodeManagementCard.displayName = "NodeManagementCard";
