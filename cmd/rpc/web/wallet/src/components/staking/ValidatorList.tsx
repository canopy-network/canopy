import React, { useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  LockOpen,
  Pause,
  Pen,
  Play,
  Scan,
} from "lucide-react";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useDenom } from "@/hooks/useDenom";
import { getCanopySymbolByHash } from "@/lib/utils/canopySymbols";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";
import { ValidatorCard } from "./ValidatorCard";
import {
  ValidatorDetails,
  ValidatorDetailsModal,
} from "./ValidatorDetailsModal";

interface Validator {
  address: string;
  nickname?: string;
  stakedAmount: number;
  status: "Staked" | "Paused" | "Unstaking" | "Delegate";
  rewards24h: number;
  committees?: number[];
  compound?: boolean;
  isSynced: boolean;
  delegate?: boolean;
  netAddress?: string;
  publicKey?: string;
  output?: string;
}

interface ValidatorListProps {
  validators: Validator[];
}

type SortColumn = "address" | "stakedAmount" | "status" | "rewards24h" | null;
type SortDirection = "asc" | "desc";

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

const desktopHeaderClass =
  "px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4";
const desktopRowCellClass =
  "px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]";
const actionButtonClass =
  "inline-flex items-center justify-center rounded-lg border border-border/60 p-2 text-foreground transition-colors hover:border-white/20 hover:bg-accent";

const formatStakedAmount = (amount: number, factor: number) => {
  if (!amount && amount !== 0) return "0.00";
  return (amount / factor).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
};

const formatRewards = (amount: number, factor: number) => {
  if (!amount && amount !== 0) return "+0.00";
  return `${amount >= 0 ? "+" : ""}${(amount / factor).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`;
};

const truncateAddress = (address: string) =>
  `${address.substring(0, 8)}…${address.substring(address.length - 4)}`;

const statusBadgeClass = (status: Validator["status"]) => {
  return WALLET_BADGE_TONE;
};

const DesktopValidatorRow: React.FC<{
  validator: Validator;
  index: number;
  onViewDetails: () => void;
}> = ({ validator, index, onViewDetails }) => {
  const { copyToClipboard } = useCopyToClipboard();
  const { openAction } = useActionModal();
  const { symbol, factor } = useDenom();

  const handlePauseUnpause = () => {
    const actionId =
      validator.status === "Staked" ? "pauseValidator" : "unpauseValidator";
    openAction(actionId, {
      prefilledData: {
        validatorAddress: validator.address,
        signerAddress: validator.address,
      },
    });
  };

  const handleEditStake = () => {
    openAction("stake", {
      titleOverride: "Edit Stake",
      prefilledData: {
        operator: validator.address,
        selectCommittees: validator.committees || [],
      },
    });
  };

  const handleUnstake = () => {
    openAction("unstake", {
      prefilledData: {
        validatorAddress: validator.address,
      },
    });
  };

  const rewardsColor =
    validator.rewards24h > 0
      ? "text-primary"
      : validator.rewards24h < 0
        ? "text-red-400"
        : "text-foreground";

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
        <div className="flex items-center gap-1.5">
          <img
            src={getCanopySymbolByHash(validator.address)}
            alt=""
            className="h-7 w-7 rounded-lg object-contain flex-shrink-0"
          />
          <span
            className="text-sm font-medium text-foreground"
            title={validator.address}
          >
            {truncateAddress(validator.address)}
          </span>
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground/40 transition-colors hover:bg-accent/60 hover:text-primary"
            onClick={() => copyToClipboard(validator.address, "Validator Address")}
            title="Copy Validator Address"
          >
            <Copy className="h-2.5 w-2.5" />
          </button>
        </div>
      </td>
      <td className={desktopRowCellClass}>
        <span className="text-sm font-medium text-foreground tabular-nums">
          {formatStakedAmount(validator.stakedAmount, factor)} {symbol}
        </span>
      </td>
      <td className={desktopRowCellClass}>
        <span className={`${WALLET_BADGE_CLASS} leading-none ${statusBadgeClass(validator.status)}`}>
          {validator.status}
        </span>
      </td>
      <td className={desktopRowCellClass}>
        <span className={`text-sm font-medium tabular-nums ${rewardsColor}`}>
          {formatRewards(validator.rewards24h, factor)} {symbol}
        </span>
      </td>
      <td
        className={desktopRowCellClass}
        style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
      >
        <div className="flex items-center gap-2">
          {validator.status !== "Unstaking" ? (
            <>
              {!validator.delegate ? (
                <ActionTooltip
                  label={validator.status === "Staked" ? "Pause Validator" : "Resume Validator"}
                  description={validator.status === "Staked" ? "Temporarily pause validator activity." : "Resume validator activity after a pause."}
                >
                  <button
                    type="button"
                    className={actionButtonClass}
                    onClick={handlePauseUnpause}
                    aria-label={validator.status === "Staked" ? "Pause Validator" : "Resume Validator"}
                  >
                    {validator.status === "Paused" ? (
                      <Play className="h-4 w-4" />
                    ) : (
                      <Pause className="h-4 w-4" />
                    )}
                  </button>
                </ActionTooltip>
              ) : null}
              <ActionTooltip
                label="Edit Stake"
                description="Adjust stake settings and committees."
              >
                <button
                  type="button"
                  className={actionButtonClass}
                  onClick={handleEditStake}
                  aria-label="Edit Stake"
                >
                  <Pen className="h-4 w-4" />
                </button>
              </ActionTooltip>
              <ActionTooltip
                label="Unstake Validator"
                description="Begin removing stake from this validator."
              >
                <button
                  type="button"
                  className={actionButtonClass}
                  onClick={handleUnstake}
                  aria-label="Unstake Validator"
                >
                  <LockOpen className="h-4 w-4" />
                </button>
              </ActionTooltip>
            </>
          ) : null}
          <ActionTooltip
            label="Validator Details"
            description="Open validator metadata and network details."
          >
            <button
              type="button"
              className="group rounded-lg border border-[#272729] p-2 transition-all duration-150 hover:border-white/15 hover:bg-[#272729]"
              onClick={onViewDetails}
              aria-label="Validator Details"
            >
              <Scan className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
            </button>
          </ActionTooltip>
        </div>
      </td>
    </motion.tr>
  );
};

export const ValidatorList: React.FC<ValidatorListProps> = ({ validators }) => {
  const [sortColumn, setSortColumn] = useState<SortColumn>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc");
  const [activeValidator, setActiveValidator] = useState<ValidatorDetails | null>(
    null,
  );

  const columns = useMemo(
    () => [
      { label: "Validator", key: "address" as const, sortable: true },
      { label: "Staked", key: "stakedAmount" as const, sortable: true },
      { label: "Status", key: "status" as const, sortable: true },
      { label: "Rewards Δ24h", key: "rewards24h" as const, sortable: true },
      { label: "Actions", key: null, sortable: false },
    ],
    [],
  );

  const handleSort = (column: Exclude<SortColumn, null>) => {
    setSortColumn((currentColumn) => {
      if (currentColumn === column) {
        setSortDirection((currentDirection) =>
          currentDirection === "desc" ? "asc" : "desc",
        );
        return currentColumn;
      }

      setSortDirection(column === "address" || column === "status" ? "asc" : "desc");
      return column;
    });
  };

  const sortedValidators = useMemo(() => {
    if (!sortColumn) return validators;

    const statusOrder: Record<Validator["status"], number> = {
      Staked: 0,
      Delegate: 1,
      Paused: 2,
      Unstaking: 3,
    };

    const sorted = [...validators];
    sorted.sort((a, b) => {
      let comparison = 0;

      switch (sortColumn) {
        case "address":
          comparison = a.address.localeCompare(b.address);
          break;
        case "stakedAmount":
          comparison = a.stakedAmount - b.stakedAmount;
          break;
        case "status":
          comparison = statusOrder[a.status] - statusOrder[b.status];
          break;
        case "rewards24h":
          comparison = a.rewards24h - b.rewards24h;
          break;
        default:
          comparison = 0;
      }

      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [sortColumn, sortDirection, validators]);

  if (validators.length === 0) {
    return (
      <motion.div
        variants={itemVariants}
        className="rounded-xl border border-border/60 bg-card p-12"
      >
        <div className="text-center text-muted-foreground">No validators found</div>
      </motion.div>
    );
  }

  return (
    <>
      <div className="hidden overflow-x-auto md:block">
        <table
          className="w-full"
          style={{ tableLayout: "auto", borderCollapse: "separate", borderSpacing: "0 4px" }}
        >
          <thead>
            <tr>
              {columns.map(({ label, key, sortable }) => {
                const isActive = sortColumn === key;

                return (
                  <th
                    key={label}
                    className={`${desktopHeaderClass} ${sortable ? "cursor-pointer select-none hover:text-white/80" : ""}`}
                    onClick={() => (sortable && key ? handleSort(key) : undefined)}
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
            {sortedValidators.map((validator, index) => (
              <DesktopValidatorRow
                key={validator.address}
                validator={validator}
                index={index}
                onViewDetails={() => setActiveValidator(validator)}
              />
            ))}
          </tbody>
        </table>
      </div>

      <div className="space-y-2.5 md:hidden">
        {sortedValidators.map((validator) => (
          <ValidatorCard
            key={validator.address}
            validator={validator}
            onViewDetails={() => setActiveValidator(validator)}
          />
        ))}
      </div>

      <ValidatorDetailsModal
        open={!!activeValidator}
        validator={activeValidator}
        onClose={() => setActiveValidator(null)}
      />
    </>
  );
};
