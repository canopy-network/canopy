import React, { useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeftRight,
  Box,
  Layers,
  Lock,
  Search,
  Send,
  Shield,
  Wallet,
  TrendingUp,
  TrendingDown,
  Users,
  Droplets,
  Percent,
} from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceHistory } from "@/hooks/useBalanceHistory";
import { useStakedBalanceHistory } from "@/hooks/useStakedBalanceHistory";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useConfig } from "@/app/providers/ConfigProvider";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

export const Accounts = () => {
  const {
    accounts,
    loading: accountsLoading,
    selectedAccount,
    switchAccount,
  } = useAccounts();
  const {
    totalBalance,
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

  const symbol   = chain?.denom?.symbol   || "CNPY";
  const decimals = chain?.denom?.decimals ?? 6;
  const divisor  = Math.pow(10, decimals);

  const [searchTerm, setSearchTerm] = useState("");

  // ── Derived aggregates ────────────────────────────────────────────────────
  const totalLiquid  = totalBalance - totalStaked;
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

  const getAccountIcon = (index: number) => {
    const icons = [
      { icon: Wallet,          bg: "bg-gradient-to-br from-primary/80 to-primary/40"       },
      { icon: Layers,          bg: "bg-gradient-to-br from-blue-500/80 to-blue-500/40"      },
      { icon: ArrowLeftRight,  bg: "bg-gradient-to-br from-purple-500/80 to-purple-500/40"  },
      { icon: Shield,          bg: "bg-gradient-to-br from-green-500/80 to-green-500/40"    },
      { icon: Box,             bg: "bg-gradient-to-br from-red-500/80 to-red-500/40"        },
    ];
    return icons[index % icons.length];
  };

  const getRealTotal = (address: string) => {
    const liquid = balances.find(b => b.address === address)?.amount ?? 0;
    const staked = stakingData.find(s => s.address === address)?.staked ?? 0;
    return { liquid, staked, total: liquid + staked };
  };

  const getStatusInfo = (address: string) => {
    const staked = stakingData.find(s => s.address === address)?.staked ?? 0;
    return staked > 0
      ? { label: "Staked",  cls: "bg-primary/15 text-primary border border-primary/20"         }
      : { label: "Liquid",  cls: "bg-muted/40 text-muted-foreground border border-border/60"    };
  };

  const processedAddresses = accounts.map((account, index) => {
    const { liquid, staked, total } = getRealTotal(account.address);
    const { label: statusLabel, cls: statusCls } = getStatusInfo(account.address);
    const { icon, bg } = getAccountIcon(index);
    return {
      id:               account.address,
      fullAddress:      account.address,
      address:          fmtAddress(account.address),
      nickname:         account.nickname || fmtAddress(account.address),
      total,
      liquid,
      staked,
      stakedPct:        total > 0 ? (staked / total) * 100 : 0,
      liquidPct:        total > 0 ? (liquid / total) * 100 : 0,
      statusLabel,
      statusCls,
      icon,
      iconBg: bg,
    };
  });

  const filteredAddresses = processedAddresses.filter(addr =>
    addr.address.toLowerCase().includes(searchTerm.toLowerCase()) ||
    addr.nickname.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  const handleSendAction = (address: string) => {
    const account = accounts.find(a => a.address === address);
    if (account && selectedAccount !== account) switchAccount(account.id);
    openAction("send", {
      prefilledData: { output: address },
      onFinish: () => { console.log("Send completed"); },
    });
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
      <span className={`flex items-center gap-1 text-xs font-mono font-medium ${pos ? "text-primary" : "text-destructive"}`}>
        {pos ? <TrendingUp style={{ width: 11, height: 11 }} /> : <TrendingDown style={{ width: 11, height: 11 }} />}
        {pos ? "+" : ""}{pct.toFixed(2)}%
        <span className="text-muted-foreground font-body font-normal">{label}</span>
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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="font-display text-2xl font-bold text-foreground tracking-tight">
            Accounts
          </h1>
          <p className="font-body text-sm text-muted-foreground mt-0.5">
            {accounts.length} address{accounts.length !== 1 ? "es" : ""} across your keystore
          </p>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground pointer-events-none" />
          <input
            type="text"
            placeholder="Search addresses…"
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
            className="h-9 w-72 rounded-lg border border-border/60 bg-secondary/80 pl-9 pr-3 text-sm font-body text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary/40 transition-colors"
          />
        </div>
      </div>

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
              <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                <Wallet className="text-primary" style={{ width: 13, height: 13 }} />
              </div>
              <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                Total Balance
              </span>
            </div>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="font-mono text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              <AnimatedNumber value={totalBalance / divisor} format={{ notation: "standard", maximumFractionDigits: 2 }} />
            </span>
            <span className="font-mono text-sm text-muted-foreground/50">{symbol}</span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <ChangePill loading={balanceHistoryLoading} pct={balanceChangePercentage} />
            <div className="flex items-center gap-1.5 text-xs font-body text-muted-foreground">
              <Droplets style={{ width: 11, height: 11 }} className="text-blue-400/70" />
              <span className="font-mono text-foreground/70">{fmt(totalLiquid)}</span>
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
              <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                <Lock className="text-primary" style={{ width: 13, height: 13 }} />
              </div>
              <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                Total Staked
              </span>
            </div>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="font-mono text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              <AnimatedNumber value={totalStaked / divisor} format={{ notation: "standard", maximumFractionDigits: 2 }} />
            </span>
            <span className="font-mono text-sm text-muted-foreground/50">{symbol}</span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <ChangePill loading={stakedHistoryLoading} pct={stakedChangePercentage} />
            <div className="flex items-center gap-1.5 text-xs font-body text-muted-foreground">
              <Percent style={{ width: 11, height: 11 }} className="text-primary/60" />
              <span className="font-mono text-foreground/70">{stakingRate.toFixed(1)}%</span>
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
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
              <Users className="text-primary" style={{ width: 13, height: 13 }} />
            </div>
            <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
              Portfolio
            </span>
          </div>

          <div className="flex items-baseline gap-1.5">
            <span className="font-mono text-[1.9rem] font-semibold text-foreground tabular-nums leading-none">
              {accounts.length}
            </span>
            <span className="font-mono text-sm text-muted-foreground/50">
              address{accounts.length !== 1 ? "es" : ""}
            </span>
          </div>

          <div className="flex items-center justify-between pt-2 border-t border-border/40">
            <div className="flex items-center gap-1.5 text-xs font-body text-muted-foreground">
              <span className="w-1.5 h-1.5 rounded-full bg-primary/70 flex-shrink-0" />
              <span className="font-mono text-foreground/70">{stakingCount}</span>
              <span className="text-muted-foreground/50">staking</span>
            </div>
            <div className="flex items-center gap-1.5 text-xs font-body text-muted-foreground">
              <span className="w-1.5 h-1.5 rounded-full bg-muted-foreground/40 flex-shrink-0" />
              <span className="font-mono text-foreground/70">{liquidCount}</span>
              <span className="text-muted-foreground/50">liquid only</span>
            </div>
          </div>
        </motion.div>
      </div>

      {/* ── Address portfolio table ── */}
      <motion.div
        className="canopy-card overflow-hidden"
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.16 }}
      >
        {/* Table header */}
        <div className="px-5 py-3.5 border-b border-border/60 flex items-center justify-between">
          <h2 className="font-display text-sm font-semibold text-foreground tracking-tight">
            Address Portfolio
          </h2>
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-primary/10 border border-primary/20">
            <span className="relative flex h-1.5 w-1.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-70" />
              <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-primary" />
            </span>
            <span className="font-mono text-[10px] font-semibold text-primary">Live</span>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[700px]">
            <thead>
              <tr className="border-b border-border/40">
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Address</th>
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Total</th>
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Staked</th>
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Liquid</th>
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Status</th>
                <th className="px-5 py-3 text-left font-display text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filteredAddresses.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-5 py-10 text-center font-body text-sm text-muted-foreground">
                    No addresses found
                  </td>
                </tr>
              ) : (
                filteredAddresses.map((addr, index) => (
                  <motion.tr
                    key={addr.id}
                    className="border-b border-border/40 last:border-0 hover:bg-muted/20 transition-colors"
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.18 + index * 0.04 }}
                  >
                    {/* Address */}
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-3">
                        <div className={`w-8 h-8 ${addr.iconBg} rounded-lg flex items-center justify-center flex-shrink-0`}>
                          <addr.icon className="text-white/90 w-3.5 h-3.5" />
                        </div>
                        <div>
                          <div className="font-body text-sm font-medium text-foreground leading-tight">
                            {addr.nickname}
                          </div>
                          <div className="font-mono text-[11px] text-muted-foreground leading-tight mt-0.5">
                            {addr.address}
                          </div>
                        </div>
                      </div>
                    </td>

                    {/* Total */}
                    <td className="px-5 py-3.5">
                      <span className="font-mono text-sm text-foreground tabular-nums">
                        {fmt(addr.total)}
                      </span>
                      <span className="font-mono text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                    </td>

                    {/* Staked */}
                    <td className="px-5 py-3.5">
                      <div>
                        <span className="font-mono text-sm text-foreground tabular-nums">{fmt(addr.staked)}</span>
                        <span className="font-mono text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                        <div className="font-mono text-[11px] text-muted-foreground/60 mt-0.5">
                          {addr.stakedPct.toFixed(1)}%
                        </div>
                      </div>
                    </td>

                    {/* Liquid */}
                    <td className="px-5 py-3.5">
                      <div>
                        <span className="font-mono text-sm text-foreground tabular-nums">{fmt(addr.liquid)}</span>
                        <span className="font-mono text-xs text-muted-foreground/50 ml-1">{symbol}</span>
                        <div className="font-mono text-[11px] text-muted-foreground/60 mt-0.5">
                          {addr.liquidPct.toFixed(1)}%
                        </div>
                      </div>
                    </td>

                    {/* Status */}
                    <td className="px-5 py-3.5">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-[11px] font-mono font-medium ${addr.statusCls}`}>
                        {addr.statusLabel}
                      </span>
                    </td>

                    {/* Actions */}
                    <td className="px-5 py-3.5">
                      <button
                        className="p-2 rounded-lg border border-border/50 hover:border-primary/30 hover:bg-primary/8 transition-all duration-150 group"
                        onClick={() => handleSendAction(addr.fullAddress)}
                        title="Send"
                      >
                        <Send className="w-3.5 h-3.5 text-muted-foreground group-hover:text-primary transition-colors" />
                      </button>
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
