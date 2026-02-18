import React from "react";
import { AnimatePresence, motion } from "framer-motion";
import { X, Copy, ExternalLink, CheckCircle2, AlertTriangle } from "lucide-react";
import { useConfig } from "@/app/providers/ConfigProvider";
import { LucideIcon } from "@/components/ui/LucideIcon";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

export interface TxError {
  code: number;
  module: string;
  msg: string;
}

export interface TxDetail {
  hash: string;
  type: string;
  amount: number;
  fee?: number;
  status: string;
  time: number;
  address?: string;
  error?: TxError;
}

interface TransactionDetailModalProps {
  tx: TxDetail | null;
  open: boolean;
  onClose: () => void;
}

/* --- helpers --------------------------------------------------- */

const toEpochMs = (t: any) => {
  const n = Number(t ?? 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  if (n > 1e16) return Math.floor(n / 1e6);
  if (n > 1e13) return Math.floor(n / 1e3);
  return n;
};

const formatDate = (tsMs: number) =>
  new Date(tsMs).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

const formatTimeAgo = (tsMs: number) => {
  const diff = Math.max(0, Date.now() - (tsMs || 0));
  const m = Math.floor(diff / 60000);
  const h = Math.floor(diff / 3600000);
  const d = Math.floor(diff / 86400000);
  if (m < 60) return `${m} min ago`;
  if (h < 24) return `${h} hour${h > 1 ? "s" : ""} ago`;
  return `${d} day${d > 1 ? "s" : ""} ago`;
};

const getStatusColor = (s: string) => {
  if (s === "Confirmed") return "bg-green-500/20 text-green-400 border-green-500/30";
  if (s === "Pending") return "bg-yellow-500/20 text-yellow-400 border-yellow-500/30";
  if (s === "Open") return "bg-orange-500/20 text-orange-400 border-orange-500/30";
  if (s === "Failed") return "bg-red-500/20 text-red-400 border-red-500/30";
  return "bg-muted/20 text-muted-foreground border-border/30";
};

/* --- sub-components -------------------------------------------- */

const DetailRow = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1 py-3 border-b border-border/30 last:border-0">
    <span className="text-xs text-muted-foreground font-medium uppercase tracking-wider shrink-0">
      {label}
    </span>
    <div className="text-sm text-foreground font-mono break-all text-right">
      {children}
    </div>
  </div>
);

const CopyHash = ({ hash }: { hash: string }) => {
  const { copyToClipboard } = useCopyToClipboard();
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    const ok = await copyToClipboard(hash, "Transaction hash");
    if (ok) {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1.5 group"
      title="Copy full hash"
    >
      <span className="font-mono text-sm text-foreground group-hover:text-primary transition-colors">
        {hash.slice(0, 10)}...{hash.slice(-8)}
      </span>
      {copied ? (
        <CheckCircle2 className="w-3.5 h-3.5 text-green-400 shrink-0" />
      ) : (
        <Copy className="w-3.5 h-3.5 text-muted-foreground group-hover:text-primary shrink-0 transition-colors" />
      )}
    </button>
  );
};

/* --- main modal ------------------------------------------------ */

export const TransactionDetailModal: React.FC<TransactionDetailModalProps> = ({
  tx,
  open,
  onClose,
}) => {
  const { manifest, chain } = useConfig();

  const getIcon = (type: string) =>
    manifest?.ui?.tx?.typeIconMap?.[type] ?? "Circle";
  const getTxMap = (type: string) =>
    manifest?.ui?.tx?.typeMap?.[type] ?? type;
  const getFundWay = (type: string) =>
    manifest?.ui?.tx?.fundsWay?.[type] ?? "neutral";

  const symbol = chain?.denom?.symbol ?? "CNPY";
  const decimals = Number(chain?.denom?.decimals ?? 6);
  const toDisplay = (n: number) => n / Math.pow(10, decimals);

  const explorerBase = chain?.explorer ?? "";

  React.useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  return (
    <AnimatePresence>
      {open && tx && (
        <>
          <motion.div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />

          <motion.div
            className="fixed inset-0 z-50 flex items-center justify-center p-4"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          >
            <motion.div
              className="w-full max-w-lg bg-card rounded-2xl border border-border shadow-2xl overflow-hidden"
              initial={{ scale: 0.95, y: 16 }}
              animate={{ scale: 1, y: 0 }}
              exit={{ scale: 0.95, y: 16, opacity: 0 }}
              transition={{ type: "spring", stiffness: 320, damping: 28 }}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between px-6 py-4 border-b border-border/50">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                    <LucideIcon
                      name={getIcon(tx.type)}
                      className="w-4 h-4 text-primary"
                    />
                  </div>
                  <div>
                    <h2 className="text-foreground font-semibold text-sm leading-tight">
                      {getTxMap(tx.type)}
                    </h2>
                    <p className="text-muted-foreground text-xs">
                      Transaction detail
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`px-2.5 py-0.5 rounded-full text-xs font-medium border ${getStatusColor(tx.status)}`}
                  >
                    {tx.status}
                  </span>
                  <button
                    onClick={onClose}
                    className="w-8 h-8 rounded-lg flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </div>

              <div className="px-6 py-2 max-h-[65vh] overflow-y-auto">
                <section className="py-4 border-b border-border/30">
                  <p className="text-xs text-muted-foreground uppercase tracking-wider font-medium mb-2">
                    Transaction Hash
                  </p>
                  <div className="flex items-center justify-between gap-2 bg-background/60 rounded-lg px-3 py-2.5">
                    <CopyHash hash={tx.hash} />
                    {explorerBase && (
                      <a
                        href={explorerBase + tx.hash}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-muted-foreground hover:text-primary transition-colors shrink-0"
                        title="View on Explorer"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </a>
                    )}
                  </div>
                </section>

                <section className="py-2">
                  {tx.time > 0 && (
                    <DetailRow label="Time">
                      <div className="text-right">
                        <div>{formatDate(toEpochMs(tx.time))}</div>
                        <div className="text-xs text-muted-foreground font-sans mt-0.5">
                          {formatTimeAgo(toEpochMs(tx.time))}
                        </div>
                      </div>
                    </DetailRow>
                  )}

                  {tx.amount != null && (
                    <DetailRow label="Amount">
                      <span
                        className={
                          getFundWay(tx.type) === "in"
                            ? "text-green-400"
                            : getFundWay(tx.type) === "out"
                              ? "text-red-400"
                              : "text-foreground"
                        }
                      >
                        {getFundWay(tx.type) === "out"
                          ? "-"
                          : getFundWay(tx.type) === "in"
                            ? "+"
                            : ""}
                        {toDisplay(Number(tx.amount)).toLocaleString("en-US", {
                          minimumFractionDigits: 2,
                          maximumFractionDigits: 6,
                        })}{" "}
                        {symbol}
                      </span>
                    </DetailRow>
                  )}

                  {tx.fee != null && tx.fee > 0 && (
                    <DetailRow label="Network Fee">
                      <span className="text-muted-foreground">
                        {toDisplay(Number(tx.fee)).toLocaleString("en-US", {
                          minimumFractionDigits: 2,
                          maximumFractionDigits: 6,
                        })}{" "}
                        {symbol}
                      </span>
                    </DetailRow>
                  )}

                  {tx.address && (
                    <DetailRow label="Account">
                      <span className="font-mono text-xs">
                        {tx.address.slice(0, 10)}...{tx.address.slice(-8)}
                      </span>
                    </DetailRow>
                  )}

                  <DetailRow label="Type">
                    <div className="flex items-center gap-1.5 justify-end font-sans">
                      <LucideIcon
                        name={getIcon(tx.type)}
                        className="w-3.5 h-3.5 text-muted-foreground"
                      />
                      <span>{getTxMap(tx.type)}</span>
                    </div>
                  </DetailRow>
                </section>

                {tx.error && (
                  <section className="py-4">
                    <div className="flex items-center gap-2 mb-2">
                      <AlertTriangle className="w-3.5 h-3.5 text-red-400 shrink-0" />
                      <p className="text-xs text-red-400 uppercase tracking-wider font-medium">
                        Transaction Error
                      </p>
                    </div>
                    <div className="bg-red-500/10 border border-red-500/20 rounded-lg px-4 py-3 space-y-2">
                      <div className="flex items-center justify-between gap-4">
                        <span className="text-xs text-muted-foreground font-medium uppercase tracking-wider shrink-0">
                          Code
                        </span>
                        <span className="text-sm text-red-400 font-mono">{tx.error.code}</span>
                      </div>
                      <div className="flex items-center justify-between gap-4">
                        <span className="text-xs text-muted-foreground font-medium uppercase tracking-wider shrink-0">
                          Module
                        </span>
                        <span className="text-sm text-foreground font-mono">{tx.error.module}</span>
                      </div>
                      <div className="flex flex-col gap-1 pt-1 border-t border-red-500/20">
                        <span className="text-xs text-muted-foreground font-medium uppercase tracking-wider">
                          Message
                        </span>
                        <span className="text-sm text-red-300 break-words">{tx.error.msg}</span>
                      </div>
                    </div>
                  </section>
                )}
              </div>

              {explorerBase && (
                <div className="px-6 py-4 border-t border-border/50 flex justify-end">
                  <a
                    href={explorerBase + tx.hash}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:text-primary/80 transition-colors"
                  >
                    View on Explorer
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                </div>
              )}
            </motion.div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};

export default TransactionDetailModal;
