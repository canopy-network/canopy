import React from "react";
import { motion } from "framer-motion";
import { Copy, LockOpen, Pause, Pen, Play, Scan } from "lucide-react";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useDenom } from "@/hooks/useDenom";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";

interface ValidatorCardProps {
  validator: {
    address: string;
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

type DetailRowProps = {
  label: string;
  value?: string;
  onCopy?: () => void;
  title?: string;
  children?: React.ReactNode;
};

const DetailRow: React.FC<DetailRowProps> = ({
  label,
  value,
  onCopy,
  title,
  children,
}) => (
  <div className="flex items-start gap-2">
    <span className="w-20 shrink-0 pt-0.5 text-[11px] font-medium uppercase tracking-wider text-white/50">
      {label}
    </span>
    <div className="min-w-0 flex-1">
      {children ?? (
        <div className="flex items-center gap-1.5">
          <span
            className="min-w-0 break-all text-xs text-muted-foreground"
            title={title ?? value}
          >
            {value || "—"}
          </span>
          {onCopy && value ? (
            <button
              type="button"
              className="rounded p-0.5 text-muted-foreground/40 transition-colors hover:bg-accent/60 hover:text-primary"
              onClick={onCopy}
              title={`Copy ${label}`}
            >
              <Copy className="h-2.5 w-2.5" />
            </button>
          ) : null}
        </div>
      )}
    </div>
  </div>
);

export const ValidatorCard: React.FC<ValidatorCardProps> = ({
  validator,
  onViewDetails,
}) => {
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
        <DetailRow
          label="Address"
          value={truncateAddress(validator.address)}
          title={validator.address}
          onCopy={() => copyToClipboard(validator.address, "Validator Address")}
        />
      </div>
    </motion.div>
  );
};
