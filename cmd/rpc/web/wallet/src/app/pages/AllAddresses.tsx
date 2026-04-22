import React, { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { ChevronDown, ChevronUp, ChevronsUpDown, Copy, Search } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { getCanopySymbol } from "@/lib/utils/canopySymbols";
import { useDenom } from "@/hooks/useDenom";
import { PageHeader } from "@/components/layouts/PageHeader";
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from "@/components/ui/badgeStyles";

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

export const AllAddresses = () => {
  const { accounts, loading: accountsLoading } = useAccounts();
  const { balances, stakingData } = useAccountData();
  const { copyToClipboard } = useCopyToClipboard();
  const { symbol, factor } = useDenom();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");

  const formatAddress = (address: string) => {
    return (
      address.substring(0, 12) + "..." + address.substring(address.length - 12)
    );
  };

  const formatBalance = (amount: number) => {
    return (amount / factor).toLocaleString("en-US", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    });
  };

  const getAccountStatus = (address: string) => {
    const stakingInfo = stakingData.find((data) => data.address === address);
    if (stakingInfo && stakingInfo.staked > 0) {
      return "Staked";
    }
    return "Liquid";
  };

  const getStatusColor = (status: string) => {
    return WALLET_BADGE_TONE;
  };

  const processedAddresses = useMemo(() => {
    return accounts.map((account) => {
      const balanceInfo = balances.find((b) => b.address === account.address);
      const balance = balanceInfo?.amount || 0;
      const stakingInfo = stakingData.find(
        (data) => data.address === account.address,
      );
      const staked = stakingInfo?.staked || 0;
      const total = balance + staked;

      return {
        id: account.address,
        address: account.address,
        nickname: account.nickname || "Unnamed",
        balance: balance,
        staked: staked,
        total: total,
        status: getAccountStatus(account.address),
      };
    });
  }, [accounts, balances, stakingData]);

  // Filter addresses
  const filteredAddresses = useMemo(() => {
    return processedAddresses.filter((addr) => {
      const matchesSearch =
        searchTerm === "" ||
        addr.address.toLowerCase().includes(searchTerm.toLowerCase()) ||
        addr.nickname.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesStatus =
        filterStatus === "all" || addr.status === filterStatus;

      return matchesSearch && matchesStatus;
    });
  }, [processedAddresses, searchTerm, filterStatus]);

  const handleSort = (column: string) => {
    setSortColumn((currentColumn) => {
      if (currentColumn === column) {
        setSortDirection((currentDirection) => currentDirection === "desc" ? "asc" : "desc");
        return currentColumn;
      }
      setSortDirection("desc");
      return column;
    });
  };

  const sortedAddresses = useMemo(() => {
    if (!sortColumn) return filteredAddresses;

    const sorted = [...filteredAddresses];
    sorted.sort((a, b) => {
      let comparison = 0;

      switch (sortColumn) {
        case "Address":
          comparison = a.address.localeCompare(b.address, undefined, { numeric: true, sensitivity: "base" });
          break;
        case "Nickname":
          comparison = a.nickname.localeCompare(b.nickname, undefined, { numeric: true, sensitivity: "base" });
          break;
        case "Liquid Balance":
          comparison = a.balance - b.balance;
          break;
        case "Staked":
          comparison = a.staked - b.staked;
          break;
        case "Total":
          comparison = a.total - b.total;
          break;
        case "Status":
          comparison = a.status.localeCompare(b.status, undefined, { numeric: true, sensitivity: "base" });
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
    { label: "Nickname", sortable: true },
    { label: "Liquid Balance", sortable: true },
    { label: "Staked", sortable: true },
    { label: "Total", sortable: true },
    { label: "Status", sortable: true },
  ]), []);

  // Calculate totals
  const totalBalance = useMemo(() => {
    return filteredAddresses.reduce((sum, addr) => sum + addr.balance, 0);
  }, [filteredAddresses]);

  const totalStaked = useMemo(() => {
    return filteredAddresses.reduce((sum, addr) => sum + addr.staked, 0);
  }, [filteredAddresses]);

  if (accountsLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-foreground text-xl">Loading addresses...</div>
      </div>
    );
  }

  return (
    <motion.div
      className="min-h-screen bg-background"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.6 }}
    >
      <div className="px-6 py-8">
        {/* Header */}
        <div className="mb-8">
          <PageHeader
            title="All Addresses"
            subtitle="Manage all your wallet addresses and their balances"
          />
        </div>

        {/* Filters */}
        <div className="bg-card rounded-xl p-6 border border-border mb-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground w-4 h-4" />
              <input
                type="text"
                placeholder="Search by address or nickname..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-text-muted focus:outline-none focus:border-primary/40 transition-colors"
              />
            </div>

            {/* Status Filter */}
            <div>
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground focus:outline-none focus:border-primary/40 transition-colors"
              >
                <option value="all">All Status</option>
                <option value="Staked">Staked</option>
                <option value="Liquid">Liquid</option>
              </select>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-card rounded-xl p-4 border border-border">
            <div className="text-sm text-muted-foreground mb-1">Total Addresses</div>
            <div className="text-2xl font-bold text-foreground">
              {accounts.length}
            </div>
          </div>
          <div className="bg-card rounded-xl p-4 border border-border">
            <div className="text-sm text-muted-foreground mb-1">Total Balance</div>
            <div className="text-2xl font-bold text-foreground">
              {formatBalance(totalBalance)} {symbol}
            </div>
          </div>
          <div className="bg-card rounded-xl p-4 border border-border">
            <div className="text-sm text-muted-foreground mb-1">Total Staked</div>
            <div className="text-2xl font-bold text-primary">
              {formatBalance(totalStaked)} {symbol}
            </div>
          </div>
          <div className="bg-card rounded-xl p-4 border border-border">
            <div className="text-sm text-muted-foreground mb-1">Filtered Results</div>
            <div className="text-2xl font-bold text-foreground">
              {filteredAddresses.length}
            </div>
          </div>
        </div>

        {/* Addresses Table */}
        <div className="bg-card rounded-xl border border-border overflow-hidden">
          <div className="flex flex-col items-start justify-between gap-3 px-6 py-4 leading-none sm:flex-row sm:items-center sm:gap-4">
            <h2 className="wallet-card-title tracking-tight">
              Addresses
            </h2>
            <LatestUpdated className="self-end sm:self-auto" />
          </div>
          <div className="overflow-x-auto">
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
                        className={`px-2 py-1.5 text-left text-[11px] font-medium tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4 ${sortable ? "cursor-pointer select-none hover:text-white/80" : ""}`}
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
                {sortedAddresses.length > 0 ? (
                  sortedAddresses.map((addr, i) => (
                    <motion.tr
                      key={addr.id}
                      className="group"
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.3, delay: i * 0.05 }}
                    >
                      <td
                        className={desktopRowCellClass}
                        style={{ borderTopLeftRadius: "10px", borderBottomLeftRadius: "10px" }}
                      >
                        <div className="flex items-center gap-3">
                          <img src={getCanopySymbol(i)} alt="" className="w-10 h-10 rounded-full object-contain flex-shrink-0" />
                          <div>
                            <div className="text-sm text-foreground">
                              {formatAddress(addr.address)}
                            </div>
                            <button
                              onClick={() =>
                                copyToClipboard(
                                  addr.address,
                                  `Address ${addr.nickname}`,
                                )
                              }
                              className="flex items-center text-xs text-white/60 transition-colors hover:text-[#35cd48]"
                            >
                              <Copy className="w-3 h-3 mr-1" />
                              Copy
                            </button>
                          </div>
                        </div>
                      </td>
                      <td className={desktopRowCellClass}>
                        <div className="text-sm text-foreground">
                          {addr.nickname}
                        </div>
                      </td>
                      <td className={desktopRowCellClass}>
                        <div className="text-sm text-foreground">
                          {formatBalance(addr.balance)} {symbol}
                        </div>
                      </td>
                      <td className={desktopRowCellClass}>
                        <div className="text-sm text-primary">
                          {formatBalance(addr.staked)} {symbol}
                        </div>
                      </td>
                      <td className={desktopRowCellClass}>
                        <div className="text-sm font-medium text-foreground">
                          {formatBalance(addr.total)} {symbol}
                        </div>
                      </td>
                      <td
                        className={desktopRowCellClass}
                        style={{ borderTopRightRadius: "10px", borderBottomRightRadius: "10px" }}
                      >
                        <span
                          className={`${WALLET_BADGE_CLASS} leading-none ${getStatusColor(addr.status)}`}
                        >
                          {addr.status}
                        </span>
                      </td>
                    </motion.tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={7}
                      className="px-6 py-12 text-center text-muted-foreground"
                    >
                      No addresses found
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

export default AllAddresses;
