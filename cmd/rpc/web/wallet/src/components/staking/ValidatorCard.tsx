import React from "react";
import { motion } from "framer-motion";
import { LockOpen, Pause, Pen, Play, Scan } from "lucide-react";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useDenom } from "@/hooks/useDenom";
import { getCanopySymbolByHash } from "@/lib/utils/canopySymbols";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { CopyableIdentifier } from "@/components/ui/CopyableIdentifier";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";

interface ValidatorCardProps {
  validator: {
    address: string;
    nickname?: string;
    stakedAmount: number;
    status: "Staked" | "Paused" | "Unstaking" | "Delegate";
    rewards24h: number;
    committees?: number[];
    isSynced: boolean;
    delegate?: boolean;
    netAddress?: string;
    publicKey?: string;
    output?: string;
  };
  onViewDetails: () => void;
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

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

const statusBadgeClass = (status: ValidatorCardProps["validator"]["status"]) => {
  return WALLET_BADGE_TONE;
};

const actionButtonClass =
  "inline-flex items-center justify-center rounded-lg border border-border/60 p-2 text-foreground transition-colors hover:border-white/20 hover:bg-accent";

export const ValidatorCard: React.FC<ValidatorCardProps> = ({
  validator,
  onViewDetails,
}) => {
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
    <motion.div
      variants={itemVariants}
      className="rounded-lg border border-[#272729] bg-[#1a1a1a] p-3.5"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="grid flex-1 grid-cols-3 gap-3">
          <div>
            <div className="text-sm font-medium text-foreground">
              {formatStakedAmount(validator.stakedAmount, factor)} {symbol}
            </div>
            <div className="mt-1 text-[11px] uppercase tracking-wider text-white/50">
              Staked
            </div>
          </div>
          <div>
            <div className={`text-sm font-medium ${rewardsColor}`}>
              {formatRewards(validator.rewards24h, factor)} {symbol}
            </div>
            <div className="mt-1 text-[11px] uppercase tracking-wider text-white/50">
              Rewards 24h
            </div>
          </div>
          <div>
            <span className={`${WALLET_BADGE_CLASS} leading-none ${statusBadgeClass(validator.status)}`}>
              {validator.status}
            </span>
          </div>
        </div>

        {validator.status !== "Unstaking" ? (
          <div className="flex items-center gap-2">
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
        ) : (
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
        )}
      </div>

      <div className="mt-3 border-t border-[#272729] pt-3">
        <div className="flex items-center gap-3">
          <img
            src={getCanopySymbolByHash(validator.address)}
            alt=""
            className="h-7 w-7 rounded-lg object-contain flex-shrink-0"
          />
          <div>
            <div className="text-sm font-medium text-foreground leading-tight">
              {validator.nickname || truncateAddress(validator.address)}
            </div>
            <div className="flex items-center gap-1 mt-0.5">
              <CopyableIdentifier value={validator.address} label="Validator Address" className="max-w-[13rem] text-[11px] text-muted-foreground leading-tight">
                {truncateAddress(validator.address)}
              </CopyableIdentifier>
            </div>
          </div>
        </div>
      </div>
    </motion.div>
  );
};
