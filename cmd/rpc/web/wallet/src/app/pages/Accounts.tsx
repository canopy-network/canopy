import React, { useCallback, useMemo, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  Droplets,
  Percent,
  Search,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceHistory } from "@/hooks/useBalanceHistory";
import { useStakedBalanceHistory } from "@/hooks/useStakedBalanceHistory";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useConfig } from "@/app/providers/ConfigProvider";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import AnimatedNumber from "@/components/ui/AnimatedNumber";
import { ActionTooltip } from "@/components/ui/ActionTooltip";
import { PageHeader } from "@/components/layouts/PageHeader";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";
import { getCanopySymbol } from "@/lib/utils/canopySymbols";

const desktopRowCellClass =
  "px-2 sm:px-3 lg:px-4 py-2 text-xs sm:text-sm text-white whitespace-nowrap align-middle transition-colors group-hover:bg-[#272729] bg-[#1a1a1a]";

const LatestUpdated = ({ className = "" }: { className?: string }) => (
  <div className={`flex items-center gap-2 lg:gap-4 ${className}`}>
    <div className="relative inline-flex items-center gap-1.5 rounded-full border border-[rgba(53,205,72,0.30)] bg-[rgba(53,205,72,0.12)] px-4 py-1">
      <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
      <span className="text-sm font-medium text-[#35cd48]">Live</span>
    </div>
  </div>
);

export const Accounts = () => {
  const {
    accounts,
    loading: accountsLoading,
    selectedAccount,
    switchAccount,
  } = useAccounts();
  const {
    totalBalance,
    totalLiquid,
    totalStaked,
    balances,
    stakingData,
    loading: dataLoading,
  } = useAccountData();
  const { data: balanceHistory, isLoading: balanceHistoryLoading } =
    useBalanceHistory();
  const { data: stakedHistory, isLoading: stakedHistoryLoading } =
    useStakedBalanceHistory();
  const { openAction } = useActionModal();
  const { chain } = useConfig();
  const { copyToClipboard } = useCopyToClipboard();

  const symbol   = chain?.denom?.symbol   || "CNPY";
  const decimals = chain?.denom?.decimals ?? 6;
  const divisor  = Math.pow(10, decimals);

  const [searchTerm, setSearchTerm] = useState("");
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");

  // ── Derived aggregates ────────────────────────────────────────────────────
  const stakingRate  = totalBalance > 0 ? (totalStaked / totalBalance) * 100 : 0;
  const stakingCount = stakingData.filter(s => (s.staked || 0) > 0).length;
  const liquidCount  = accounts.length - stakingCount;

  const balanceChangePercentage = balanceHistory?.changePercentage ?? 0;
  const stakedChangePercentage  = stakedHistory?.changePercentage  ?? 0;

  // ── Helpers ───────────────────────────────────────────────────────────────
  const fmt = (raw: number) =>
    (raw / divisor).toLocaleString("en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });

  const fmtAddress = (addr: string) =>
    `${addr.slice(0, 5)}…${addr.slice(-6)}`;

  const getAccountSymbol = (index: number) => getCanopySymbol(index);

  const getRealTotal = (address: string) => {
    const liquid = balances.find(b => b.address === address)?.amount ?? 0;
    const staked = stakingData.find(s => s.address === address)?.staked ?? 0;
    return { liquid, staked, total: liquid + staked };
  };

  const getStatusInfo = (address: string) => {
    const staked = stakingData.find(s => s.address === address)?.staked ?? 0;
    return staked > 0
      ? { label: "Staked",  cls: WALLET_BADGE_TONE }
      : { label: "Liquid",  cls: WALLET_BADGE_TONE };
  };

  const processedAddresses = useMemo(() => accounts
    .map((account, index) => {
      const { liquid, staked, total } = getRealTotal(account.address);
      const { label: statusLabel, cls: statusCls } = getStatusInfo(account.address);
      const symbolSrc = getAccountSymbol(index);
      return {
        id:               account.address,
        fullAddress:      account.address,
        address:          fmtAddress(account.address),
        nickname:         account.nickname || fmtAddress(account.address),
        publicKey:        account.publicKey || "",
        total,
        liquid,
        staked,
        stakedPct:        total > 0 ? (staked / total) * 100 : 0,
        liquidPct:        total > 0 ? (liquid / total) * 100 : 0,
        statusLabel,
        statusCls,
        symbolSrc,
      };
    })
    .sort((a, b) => a.nickname.localeCompare(b.nickname)), [accounts, balances, stakingData]);

  const filteredAddresses = processedAddresses.filter(addr => {
    const term = searchTerm.toLowerCase();
    return addr.address.toLowerCase().includes(term) ||
      addr.nickname.toLowerCase().includes(term) ||
      addr.fullAddress.toLowerCase().includes(term) ||
      addr.publicKey.toLowerCase().includes(term);
  });

  const handleSort = useCallback((column: string) => {
    setSortColumn((currentColumn) => {
      if (currentColumn === column) {
        setSortDirection((currentDirection) => currentDirection === "desc" ? "asc" : "desc");
        return currentColumn;
      }
      setSortDirection("desc");
      return column;
    });
  }, []);

  const sortedAddresses = useMemo(() => {
    if (!sortColumn) return filteredAddresses;

    const sorted = [...filteredAddresses];
    sorted.sort((a, b) => {
      let comparison = 0;

      switch (sortColumn) {
        case "Address":
          comparison = a.fullAddress.localeCompare(b.fullAddress, undefined, { numeric: true, sensitivity: "base" });
          break;
        case "Total":
          comparison = a.total - b.total;
          break;
        case "Staked":
          comparison = a.staked - b.staked;
          break;
        case "Liquid":
          comparison = a.liquid - b.liquid;
          break;
        case "Status":
          comparison = a.statusLabel.localeCompare(b.statusLabel, undefined, { numeric: true, sensitivity: "base" });
          break;
        default:
          comparison = 0;
      }

      return sortDirection === "asc" ? comparison : -comparison;
    });

    return sorted;
  }, [filteredAddresses, sortColumn, sortDirection]);

  const columns = useMemo(() => ([
    { label: "Address", sortable: true },
    { label: "Total", sortable: true },
    { label: "Staked", sortable: true },
    { label: "Liquid", sortable: true },
    { label: "Status", sortable: true },
    { label: "Actions", sortable: false },
  ]), []);

  const handleSendAction = (address: string) => {
    const account = accounts.find(a => a.address === address);
    if (account && selectedAccount !== account) switchAccount(account.id);
    openAction("send", {
      onFinish: () => { console.log("Send completed"); },
    });
  };

  const handleReceiveAction = (address: string) => {
    const account = accounts.find(a => a.address === address);
    if (account && selectedAccount !== account) switchAccount(account.id);
    openAction("receive");
  };

  // ── Loading skeleton ──────────────────────────────────────────────────────
  if (accountsLoading || dataLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 rounded-md skeleton" />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[0, 1, 2].map(i => (
            <div key={i} className="canopy-card p-5 h-32 skeleton" />
          ))}
        </div>
        <div className="canopy-card h-64 skeleton" />
      </div>
    );
  }

  // ── Change pill helper ────────────────────────────────────────────────────
  const ChangePill = ({
    loading,
    pct,
    label = "24h",
  }: {
    loading: boolean;
    pct: number;
    label?: string;
  }) => {
    if (loading) return <div className="h-4 w-20 rounded skeleton" />;
    const pos = pct >= 0;
    return (
      <span className={`flex items-center gap-1 text-xs font-medium ${pos ? "text-primary" : "text-destructive"}`}>
        {pos ? <TrendingUp style={{ width: 11, height: 11 }} /> : <TrendingDown style={{ width: 11, height: 11 }} />}
        {pos ? "+" : ""}{pct.toFixed(2)}%
        <span className="text-muted-foreground font-normal">{label}</span>
      </span>
    );
  };

  return (
    <motion.div
      className="space-y-6"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      {/* ── Page header ── */}
      <PageHeader
        title="Accounts"
        subtitle={`${accounts.length} address${accounts.length !== 1 ? "es" : ""} across your keystore`}
        actions={
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
            <input
              type="text"
              placeholder="Search addresses…"
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              className="h-9 w-72 rounded-lg border border-border/60 bg-secondary/80 pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary/40 transition-colors"
            />
          </div>
        }
      />

      {/* ── Summary cards ── */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">

        {/* Card 1 — Total Balance */}
        <motion.div
          className="canopy-card p-5 flex flex-col gap-3"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.04 }}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="wallet-card-title">
                Total Balance
              </span>
            </div>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              <AnimatedNumber value={totalBalance / divisor} format={{ notation: "standard", maximumFractionDigits: 2 }} />
            </span>
            <span className="text-sm text-muted-foreground/50">{symbol}</span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <ChangePill loading={balanceHistoryLoading} pct={balanceChangePercentage} label={balanceHistory?.periodLabel ?? "24h"} />
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Droplets style={{ width: 11, height: 11 }} className="text-blue-400/70" />
              <span className="text-foreground/70">{fmt(totalLiquid)}</span>
              <span className="text-muted-foreground/50">liquid</span>
            </div>
          </div>
        </motion.div>

        {/* Card 2 — Total Staked */}
        <motion.div
          className="canopy-card p-5 flex flex-col gap-3"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.08 }}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="wallet-card-title">
                Total Staked
              </span>
            </div>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              <AnimatedNumber value={totalStaked / divisor} format={{ notation: "standard", maximumFractionDigits: 2 }} />
            </span>
            <span className="text-sm text-muted-foreground/50">{symbol}</span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <ChangePill loading={stakedHistoryLoading} pct={stakedChangePercentage} label={stakedHistory?.periodLabel ?? "24h"} />
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Percent style={{ width: 11, height: 11 }} className="text-primary/60" />
              <span className="text-foreground/70">{stakingRate.toFixed(1)}%</span>
              <span className="text-muted-foreground/50">of total</span>
            </div>
          </div>
        </motion.div>

        {/* Card 3 — Portfolio */}
        <motion.div
          className="canopy-card p-5 flex flex-col gap-3"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.12 }}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="wallet-card-title">
                Portfolio
              </span>
            </div>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              <AnimatedNumber value={accounts.length} format={{ notation: "standard", maximumFractionDigits: 0 }} />
            </span>
            <span className="text-sm text-muted-foreground/50">
              address{accounts.length !== 1 ? "es" : ""}
            </span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <span className="w-1.5 h-1.5 rounded-full bg-primary/70 flex-shrink-0" />
              <span className="text-foreground/70">{stakingCount}</span>
              <span className="text-muted-foreground/50">staking</span>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/40 flex-shrink-0" />
              <span className="text-foreground/70">{liquidCount}</span>
              <span className="text-muted-foreground/50">liquid only</span>
            </div>
          </div>
        </motion.div>
      </div>

      {/* ── Address portfolio table ── */}
      <motion.div
        className="canopy-card p-5"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.16 }}
      >
        {/* Table header */}
        <div className="mb-5 flex flex-col items-start justify-between gap-3 leading-none sm:flex-row sm:items-center sm:gap-4">
          <h2 className="wallet-card-title tracking-tight">
            Address Portfolio
          </h2>
          <LatestUpdated className="self-end sm:self-auto" />
        </div>

        <div className="overflow-x-auto">
          <table
            className="w-full min-w-[850px]"
            style={{ tableLayout: "auto", borderCollapse: "separate", borderSpacing: "0 4px" }}
          >
            <thead>
              <tr>
                {columns.map(({ label, sortable }) => {
                  const isActive = sortColumn === label;

                  return (
                    <th
                      key={label}
                      className={`px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4 ${sortable ? "cursor-pointer select-none hover:text-white/80" : ""}`}
                      onClick={() => sortable ? handleSort(label) : undefined}
                    >
                      <div className="flex items-center gap-1">
                        {label}
                        {sortable && (
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
                        )}
                      </div>
                    </th>
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {filteredAddresses.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-5 py-10 text-center text-sm text-muted-foreground">
                    No addresses found
                  </td>
                </tr>
              ) : (
                sortedAddresses.map((addr, index) => (
                  <motion.tr
                    key={addr.id}
                    className="group"
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.18 + index * 0.04 }}
                  >
                    {/* Address */}
                    <td
                      className={desktopRowCellClass}
                      style={{ borderTopLeftRadius: "10px", borderBottomLeftRadius: "10px" }}
                    >
                      <div className="flex items-center gap-3">
                        <img src={addr.symbolSrc} alt="" className="w-8 h-8 rounded-lg object-contain flex-shrink-0" />
                        <div>
                          <div className="text-sm font-medium text-foreground leading-tight">
                            {addr.nickname}
                          </div>
                          <div className="flex items-center gap-1 mt-0.5">
                            <span className="text-[11px] text-muted-foreground leading-tight">
                              {addr.address}
                            </span>
                            <button
                              onClick={() => copyToClipboard(addr.fullAddress, "Address")}
                              className="rounded p-0.5 text-white/40 transition-colors hover:bg-[#272729] hover:text-white"
                            >
                              <Copy style={{ width: 10, height: 10 }} />
                            </button>
                          </div>
                        </div>
                      </div>
                    </td>

                    {/* Total */}
                    <td className={desktopRowCellClass}>
                      <span className="text-sm text-foreground tabular-nums">
                        {fmt(addr.total)}
                      </span>
                      <span className="text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                    </td>

                    {/* Staked */}
                    <td className={desktopRowCellClass}>
                      <div>
                        <span className="text-sm text-foreground tabular-nums">{fmt(addr.staked)}</span>
                        <span className="text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                      </div>
                    </td>

                    {/* Liquid */}
                    <td className={desktopRowCellClass}>
                      <div>
                        <span className="text-sm text-foreground tabular-nums">{fmt(addr.liquid)}</span>
                        <span className="text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                      </div>
                    </td>

                    {/* Status */}
                    <td className={desktopRowCellClass}>
                      <span className={`${WALLET_BADGE_CLASS} leading-none ${addr.statusCls}`}>
                        {addr.statusLabel}
                      </span>
                    </td>

                    {/* Actions */}
                    <td
                      className={desktopRowCellClass}
                      style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
                    >
                      <div className="flex items-center gap-1.5">
                        <ActionTooltip
                          label="Send"
                          description="Transfer funds from this address."
                        >
                          <button
                            className="group rounded-lg border border-[#272729] p-2 transition-all duration-150 hover:border-white/15 hover:bg-[#272729]"
                            onClick={() => handleSendAction(addr.fullAddress)}
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
                            className="group rounded-lg border border-[#272729] p-2 transition-all duration-150 hover:border-white/15 hover:bg-[#272729]"
                            onClick={() => handleReceiveAction(addr.fullAddress)}
                            aria-label="Receive"
                          >
                            <ArrowDown className="h-3.5 w-3.5 text-muted-foreground transition-colors group-hover:text-[#35cd48]" />
                          </button>
                        </ActionTooltip>
                      </div>
                    </td>
                  </motion.tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </motion.div>
    </motion.div>
  );
};
