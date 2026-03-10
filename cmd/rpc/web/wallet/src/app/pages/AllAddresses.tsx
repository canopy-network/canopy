import React, { useState, useMemo } from "react";
import { motion } from "framer-motion";
import { Search, Wallet, Copy } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useAccounts } from "@/app/providers/AccountsProvider";

export const AllAddresses = () => {
  const { accounts, loading: accountsLoading } = useAccounts();
  const { balances, stakingData } = useAccountData();
  const { copyToClipboard } = useCopyToClipboard();

  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");

  const formatAddress = (address: string) => {
    return (
      address.substring(0, 12) + "..." + address.substring(address.length - 12)
    );
  };

  const formatBalance = (amount: number) => {
    return (amount / 1000000).toLocaleString("en-US", {
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
    switch (status) {
      case "Staked":
        return "bg-primary/20 text-primary border border-primary/40";
      case "Unstaking":
        return "bg-orange-500/20 text-orange-400 border border-orange-500/40";
      case "Liquid":
        return "bg-muted/20 text-muted-foreground border border-border/40";
      default:
        return "bg-muted/20 text-muted-foreground border border-border/40";
    }
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
          <h1 className="text-3xl font-bold text-foreground mb-2">
            All Addresses
          </h1>
          <p className="text-muted-foreground">
            Manage all your wallet addresses and their balances
          </p>
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
              {formatBalance(totalBalance)} CNPY
            </div>
          </div>
          <div className="bg-card rounded-xl p-4 border border-border">
            <div className="text-sm text-muted-foreground mb-1">Total Staked</div>
            <div className="text-2xl font-bold text-primary">
              {formatBalance(totalStaked)} CNPY
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
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-accent/30">
                <tr>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Address
                  </th>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Nickname
                  </th>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Liquid Balance
                  </th>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Staked
                  </th>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Total
                  </th>
                  <th className="text-left px-6 py-4 text-sm font-medium text-muted-foreground">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredAddresses.length > 0 ? (
                  filteredAddresses.map((addr, i) => (
                    <motion.tr
                      key={addr.id}
                      className="border-b border-border/30 hover:bg-accent/20 transition-colors"
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.3, delay: i * 0.05 }}
                    >
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 bg-gradient-to-r from-primary/80 to-primary/40 rounded-full flex items-center justify-center flex-shrink-0">
                            <Wallet className="text-foreground w-4 h-4" />
                          </div>
                          <div>
                            <div className="text-sm text-foreground font-mono">
                              {formatAddress(addr.address)}
                            </div>
                            <button
                              onClick={() =>
                                copyToClipboard(
                                  addr.address,
                                  `Address ${addr.nickname}`,
                                )
                              }
                              className="text-xs text-muted-foreground hover:text-primary transition-colors flex items-center"
                            >
                              <Copy className="w-3 h-3 mr-1" />
                              Copy
                            </button>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="text-sm text-foreground">
                          {addr.nickname}
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="text-sm text-foreground">
                          {formatBalance(addr.balance)} CNPY
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="text-sm text-primary">
                          {formatBalance(addr.staked)} CNPY
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="text-sm font-medium text-foreground">
                          {formatBalance(addr.total)} CNPY
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(addr.status)}`}
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

