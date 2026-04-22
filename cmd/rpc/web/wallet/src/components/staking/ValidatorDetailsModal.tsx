import React, { useEffect } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { createPortal } from "react-dom";
import { Copy, Globe, Scan, X } from "lucide-react";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useDenom } from "@/hooks/useDenom";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";

export interface ValidatorDetails {
  address: string;
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

const truncateMiddle = (value: string, start = 12, end = 8) => {
  if (value.length <= start + end + 1) return value;
  return `${value.slice(0, start)}…${value.slice(-end)}`;
};

const formatAmount = (amount: number, factor: number) =>
  (amount / factor).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });

const formatRewards = (amount: number, factor: number) =>
  `${amount >= 0 ? "+" : ""}${formatAmount(amount, factor)}`;

const statusBadgeClass = (status: ValidatorDetails["status"]) => {
  return WALLET_BADGE_TONE;
};

const DetailBlock: React.FC<{
  label: string;
  value?: string;
  onCopy?: () => void;
  icon?: React.ReactNode;
  children?: React.ReactNode;
}> = ({ label, value, onCopy, icon, children }) => (
  <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
    <div className="mb-2 text-[11px] font-medium uppercase tracking-wider text-white/50">
      {label}
    </div>
    {children ?? (
      <div className="flex items-start gap-2">
        {icon}
        <span className="min-w-0 break-all text-sm text-foreground">
          {value || "—"}
        </span>
        {onCopy && value ? (
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground/40 transition-colors hover:bg-accent/60 hover:text-primary"
            onClick={onCopy}
            title={`Copy ${label}`}
          >
            <Copy className="h-3 w-3" />
          </button>
        ) : null}
      </div>
    )}
  </div>
);

export const ValidatorDetailsModal: React.FC<{
  open: boolean;
  validator: ValidatorDetails | null;
  onClose: () => void;
}> = ({ open, validator, onClose }) => {
  const { copyToClipboard } = useCopyToClipboard();
  const { symbol, factor } = useDenom();

  useEffect(() => {
    if (!open) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };

    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", onKeyDown);

    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [open, onClose]);

  if (typeof document === "undefined") return null;

  return createPortal(
    <AnimatePresence>
      {open && validator ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 flex items-start justify-center bg-[#0f0f0f]/80 p-2 pt-[calc(env(safe-area-inset-top)+60px)] backdrop-blur-md sm:items-center sm:p-4"
          style={{ zIndex: 9999 }}
          onClick={onClose}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.96, y: 12 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 12 }}
            transition={{ duration: 0.2, ease: "easeInOut" }}
            className="relative flex w-full max-w-[min(100vw-1rem,56rem)] flex-col overflow-hidden rounded-lg border border-[#272729] bg-[#171717] shadow-[0_24px_72px_rgba(0,0,0,0.55)] sm:rounded-xl"
            onClick={(event) => event.stopPropagation()}
          >
            <button
              type="button"
              onClick={onClose}
              className="absolute right-3 top-3 z-10 rounded-lg border border-[#272729] bg-[#0f0f0f] p-1 text-white/60 transition-colors hover:bg-[#272729] hover:text-white sm:right-4 sm:top-4"
              title="Close"
            >
              <X className="h-4 w-4 sm:h-5 sm:w-5" />
            </button>

            <div className="shrink-0 px-4 pb-3 pt-5 sm:px-6 sm:pb-4 sm:pt-6">
              <div className="mb-2 flex items-start gap-3 pr-8">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-[#272729] bg-[#0f0f0f]">
                  <Scan className="h-5 w-5 text-[#35cd48]" />
                </div>
                <div>
                  <h2 className="text-xl font-semibold text-foreground sm:text-2xl">
                    Validator Details
                  </h2>
                  <p className="mt-1 text-sm text-muted-foreground">
                    View validator metadata and staking information.
                  </p>
                </div>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto px-4 pb-4 sm:px-6 sm:pb-6">
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-white/50">
                    Staked
                  </div>
                  <div className="mt-2 text-lg font-semibold text-foreground">
                    {formatAmount(validator.stakedAmount, factor)} {symbol}
                  </div>
                </div>
                <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-white/50">
                    Rewards 24h
                  </div>
                  <div
                    className={`mt-2 text-lg font-semibold ${
                      validator.rewards24h >= 0 ? "text-primary" : "text-red-400"
                    }`}
                  >
                    {formatRewards(validator.rewards24h, factor)} {symbol}
                  </div>
                </div>
                <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
                  <div className="text-[11px] font-medium uppercase tracking-wider text-white/50">
                    Status
                  </div>
                  <div className="mt-2">
                    <span className={`${WALLET_BADGE_CLASS} leading-none ${statusBadgeClass(validator.status)}`}>
                      {validator.status}
                    </span>
                  </div>
                </div>
              </div>

              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <DetailBlock
                  label="Address"
                  value={validator.address}
                  onCopy={() => copyToClipboard(validator.address, "Validator Address")}
                />
                <DetailBlock
                  label="Public Key"
                  onCopy={
                    validator.publicKey
                      ? () => copyToClipboard(validator.publicKey!, "Public Key")
                      : undefined
                  }
                >
                  <div className="flex items-start gap-2">
                    <span
                      className="min-w-0 break-all text-sm text-foreground"
                      title={validator.publicKey}
                    >
                      {validator.publicKey
                        ? truncateMiddle(validator.publicKey)
                        : "—"}
                    </span>
                    {validator.publicKey ? (
                      <button
                        type="button"
                        className="rounded p-0.5 text-muted-foreground/40 transition-colors hover:bg-accent/60 hover:text-primary"
                        onClick={() => copyToClipboard(validator.publicKey!, "Public Key")}
                        title="Copy Public Key"
                      >
                        <Copy className="h-3 w-3" />
                      </button>
                    ) : null}
                  </div>
                </DetailBlock>
                <DetailBlock
                  label="Net Address"
                  value={validator.netAddress}
                  icon={
                    validator.netAddress ? (
                      <Globe className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground/60" />
                    ) : undefined
                  }
                  onCopy={
                    validator.netAddress
                      ? () => copyToClipboard(validator.netAddress!, "Network Address")
                      : undefined
                  }
                />
                <DetailBlock label="Committees">
                  {(validator.committees?.length ?? 0) > 0 ? (
                    <div className="flex flex-wrap gap-1.5">
                      {validator.committees!.map((committee) => (
                        <span
                          key={committee}
                          className="rounded border border-primary/20 bg-primary/12 px-2 py-0.5 text-xs font-medium text-primary"
                        >
                          {committee}
                        </span>
                      ))}
                    </div>
                  ) : (
                    <span className="text-sm text-muted-foreground">—</span>
                  )}
                </DetailBlock>
                <DetailBlock
                  label="Role"
                  value={validator.delegate ? "Delegate" : "Validator"}
                />
                <DetailBlock label="Auto-Compounding">
                  <span
                    className={`inline-flex items-center rounded-md border px-2 py-1 text-xs font-medium ${
                      validator.compound === true
                        ? "border-[#35cd48]/35 bg-[#35cd48]/12 text-[#35cd48]"
                        : validator.compound === false
                          ? "border-[#272729] bg-[#171717] text-white/70"
                          : "border-[#272729] bg-[#171717] text-muted-foreground"
                    }`}
                  >
                    {validator.compound === true
                      ? "Enabled"
                      : validator.compound === false
                        ? "Disabled"
                        : "Unknown"}
                  </span>
                </DetailBlock>
                {validator.output ? (
                  <DetailBlock
                    label="Output"
                    value={validator.output}
                    onCopy={() => copyToClipboard(validator.output!, "Output")}
                  />
                ) : null}
              </div>
            </div>
          </motion.div>
        </motion.div>
      ) : null}
    </AnimatePresence>,
    document.body,
  );
};
