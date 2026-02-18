import React, { useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeftRight,
  Box,
  CheckCircle,
  ChevronDown,
  Circle,
  Layers,
  Lock,
  Search,
  Send,
  Shield,
  Wallet,
} from "lucide-react";
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from "chart.js";
import { Line } from "react-chartjs-2";
import { useAccountData } from "@/hooks/useAccountData";
import { useBalanceHistory } from "@/hooks/useBalanceHistory";
import { useStakedBalanceHistory } from "@/hooks/useStakedBalanceHistory";
import { useBalanceChart } from "@/hooks/useBalanceChart";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useAccounts } from "@/app/providers/AccountsProvider";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
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
    totalStaked,
    balances,
    stakingData,
    loading: dataLoading,
  } = useAccountData();
  const { data: balanceHistory, isLoading: balanceHistoryLoading } =
    useBalanceHistory();
  const { data: stakedHistory, isLoading: stakedHistoryLoading } =
    useStakedBalanceHistory();
  const { data: balanceChartData = [], isLoading: balanceChartLoading } =
    useBalanceChart({ points: 6, type: "balance" });
  const { data: stakedChartData = [], isLoading: stakedChartLoading } =
    useBalanceChart({ points: 6, type: "staked" });
  const { openAction } = useActionModal();

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedNetwork, setSelectedNetwork] = useState("All Networks");

  const formatAddress = (address: string) => {
    return (
      address.substring(0, 5) + "..." + address.substring(address.length - 6)
    );
  };

  const formatBalance = (amount: number) => {
    return (amount / 1000000).toFixed(2);
  };

  const getAccountIcon = (index: number) => {
    const icons = [
      { icon: Wallet, bg: "bg-gradient-to-r from-primary/80 to-primary/40" },
      { icon: Layers, bg: "bg-gradient-to-r from-blue-500/80 to-blue-500/40" },
      {
        icon: ArrowLeftRight,
        bg: "bg-gradient-to-r from-purple-500/80 to-purple-500/40",
      },
      {
        icon: Shield,
        bg: "bg-gradient-to-r from-green-500/80 to-green-500/40",
      },
      { icon: Box, bg: "bg-gradient-to-r from-red-500/80 to-red-500/40" },
    ];
    return icons[index % icons.length];
  };

  const getAccountStatus = (address: string) => {
    const stakingInfo = stakingData.find((data) => data.address === address);
    if (stakingInfo && stakingInfo.staked > 0) {
      return {
        status: "Staked",
        color: "bg-primary/20 text-primary",
      };
    }
    return {
      status: "Liquid",
      color: "bg-muted/20 text-muted-foreground",
    };
  };

  const getStatusColor = (status: string) => {
    const stakedText = "Staked";
    const unstakingText = "Unstaking";
    const liquidText = "Liquid";
    const delegatedText = "Delegated";

    switch (status) {
      case stakedText:
        return "bg-primary/20 text-primary";
      case unstakingText:
        return "bg-orange-500/20 text-orange-400";
      case liquidText:
        return "bg-muted/20 text-muted-foreground";
      case delegatedText:
        return "bg-primary/20 text-primary";
      default:
        return "bg-muted/20 text-muted-foreground";
    }
  };

  const getRealTotal = (address: string) => {
    const balanceInfo = balances.find((b) => b.address === address);
    const stakingInfo = stakingData.find((s) => s.address === address);

    const liquid = balanceInfo?.amount || 0;
    const staked = stakingInfo?.staked || 0;

    return { liquid, staked, total: liquid + staked };
  };

  const getStakedPercentage = (address: string) => {
    const { staked, total } = getRealTotal(address);

    if (total === 0) return 0;
    return (staked / total) * 100;
  };

  const getLiquidPercentage = (address: string) => {
    const { liquid, total } = getRealTotal(address);

    if (total === 0) return 0;
    return (liquid / total) * 100;
  };

  const getLiquidAmount = (address: string) => {
    const { liquid } = getRealTotal(address);
    return liquid;
  };

  // Get real 24h changes from unified history hooks
  const balanceChangePercentage = balanceHistory?.changePercentage || 0;
  const stakedChangePercentage = stakedHistory?.changePercentage || 0;

  // Prepare chart data from useBalanceChart hook
  const balanceChart = {
    labels: balanceChartData.map((d) => d.label),
    datasets: [
      {
        data: balanceChartData.map((d) => d.value / 1000000),
        borderColor: "hsl(var(--primary))",
        backgroundColor: "hsl(var(--primary) / 0.1)",
        borderWidth: 2,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHoverRadius: 4,
      },
    ],
  };

  const stakedChart = {
    labels: stakedChartData.map((d) => d.label),
    datasets: [
      {
        data: stakedChartData.map((d) => d.value / 1000000),
        borderColor: "hsl(var(--primary))",
        backgroundColor: "hsl(var(--primary) / 0.1)",
        borderWidth: 2,
        fill: true,
        tension: 0.4,
        pointRadius: 0,
        pointHoverRadius: 4,
      },
    ],
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        enabled: false,
      },
    },
    scales: {
      x: {
        display: false,
      },
      y: {
        display: false,
      },
    },
    elements: {
      point: {
        radius: 0,
      },
    },
  };

  const handleSendAction = (address: string) => {
    // Set the account as selected before opening the action
    const account = accounts.find((a) => a.address === address);
    if (account && selectedAccount !== account) {
      switchAccount(account.id);
    }
    // Open send action modal with prefilled output address
    openAction("send", {
      prefilledData: {
        output: address,
      },
      onFinish: () => {
        console.log("Send action completed");
      },
    });
  };

  const processedAddresses = accounts.map((account, index) => {
    const balanceInfo = balances.find((b) => b.address === account.address);
    const balance = balanceInfo?.amount || 0;
    const formattedBalance = formatBalance(balance);
    const stakingInfo = stakingData.find(
      (data) => data.address === account.address,
    );
    const staked = stakingInfo?.staked || 0;
    const stakedFormatted = formatBalance(staked);
    const liquidAmount = getLiquidAmount(account.address);
    const liquidFormatted = formatBalance(liquidAmount);
    const stakedPercentage = getStakedPercentage(account.address);
    const liquidPercentage = getLiquidPercentage(account.address);
    const statusInfo = getAccountStatus(account.address);
    const accountIcon = getAccountIcon(index);

    return {
      id: account.address,
      address: formatAddress(account.address),
      fullAddress: account.address,
      nickname: account.nickname || formatAddress(account.address),
      balance: formattedBalance,
      staked: stakedFormatted,
      liquid: liquidFormatted,
      stakedPercentage: stakedPercentage,
      liquidPercentage: liquidPercentage,
      status: statusInfo.status,
      statusColor: getStatusColor(statusInfo.status),
      icon: accountIcon.icon,
      iconBg: accountIcon.bg,
    };
  });

  const filteredAddresses = processedAddresses.filter(
    (addr) =>
      addr.address.toLowerCase().includes(searchTerm.toLowerCase()) ||
      addr.nickname.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  const activeAddressesCount = processedAddresses.filter(
    (addr) => addr.status === "Staked" || addr.status === "Delegated",
  ).length;

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        duration: 0.6,
        staggerChildren: 0.1,
      },
    },
  };

  const cardVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0 },
  };

  if (accountsLoading || dataLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-foreground text-xl">{"Loading accounts..."}</div>
      </div>
    );
  }

  return (
    <motion.div
      className="min-h-screen bg-background"
      initial="hidden"
      animate="visible"
      variants={containerVariants}
    >
      <div className="px-6 py-8">
        {/* Header Section */}
        <motion.div className="mb-8" variants={cardVariants}>
          <div className="flex items-center justify-between mb-6">
            <div>
              <h1 className="text-3xl font-bold text-foreground mb-2">
                All Addresses
              </h1>
              <p className="text-muted-foreground">
                Manage and monitor all your blockchain addresses across
                different networks
              </p>
            </div>

            {/* Search and Filter Bar */}
            <div className="flex items-center gap-4">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
                <input
                  type="text"
                  placeholder={"Search addresses..."}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full bg-card lg:w-96 border border-border rounded-lg pl-10 pr-4 py-2 text-foreground placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary/50"
                />
              </div>
              <div className="relative">
                <select
                  value={selectedNetwork}
                  onChange={(e) => setSelectedNetwork(e.target.value)}
                  className="bg-card border border-border rounded-lg px-4 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 appearance-none pr-8"
                >
                  <option value="All Networks">{"All Networks"}</option>
                  <option value="Canopy Mainnet">{"Canopy Mainnet"}</option>
                </select>
                <ChevronDown className="absolute right-2 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4 pointer-events-none" />
              </div>
            </div>
          </div>
        </motion.div>

        {/* Summary Cards */}
        <motion.div
          className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8"
          variants={cardVariants}
        >
          {/* Total Balance Card */}
          <div className="bg-card rounded-xl p-6 border border-border relative overflow-hidden">
            <div className="flex items-center justify-between">
              <h3 className="text-muted-foreground text-sm font-medium mb-2">
                Total Balance
              </h3>
              <Wallet className="text-primary w-5 h-5" />
            </div>
            <div className="text-3xl font-medium  text-foreground mb-2">
              <AnimatedNumber
                value={totalBalance / 1000000}
                format={{
                  notation: "standard",
                  maximumFractionDigits: 2,
                }}
              />
              &nbsp;CNPY
            </div>
            <div className="flex items-center justify-between">
              {balanceHistoryLoading ? (
                <span className="text-sm text-muted-foreground">Loading...</span>
              ) : balanceHistory ? (
                <span
                  className={`text-sm flex items-center gap-1 ${balanceChangePercentage >= 0 ? "text-primary" : "text-status-error"}`}
                >
                  <svg
                    className={`w-4 h-4 ${balanceChangePercentage < 0 ? "rotate-180" : ""}`}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                  {balanceChangePercentage >= 0 ? "+" : ""}
                  {balanceChangePercentage.toFixed(2)}%
                  <span className="text-muted-foreground ml-1">24h change</span>
                </span>
              ) : (
                <span className="text-sm text-muted-foreground">No data</span>
              )}
              {!balanceChartLoading && balanceChartData.length > 0 && (
                <div className="w-20 h-12">
                  <Line
                    key="balance-chart"
                    data={balanceChart}
                    options={chartOptions}
                  />
                </div>
              )}
            </div>
          </div>

          {/* Total Staked Card */}
          <div className="bg-card rounded-xl p-6 border border-border relative overflow-hidden">
            <div className="flex items-center justify-between">
              <h3 className="text-muted-foreground text-sm font-medium mb-2">
                Total Staked
              </h3>
              <Lock className="text-primary w-5 h-5" />
            </div>
            <div className="text-3xl font-medium text-foreground mb-2">
              <AnimatedNumber
                value={totalStaked / 1000000}
                format={{
                  notation: "standard",
                  maximumFractionDigits: 2,
                }}
              />
              &nbsp;CNPY
            </div>
            <div className="flex items-center justify-between">
              {stakedHistoryLoading ? (
                <span className="text-sm text-muted-foreground">Loading...</span>
              ) : stakedHistory ? (
                <span
                  className={`text-sm flex items-center gap-1 ${stakedChangePercentage >= 0 ? "text-primary" : "text-status-error"}`}
                >
                  <svg
                    className={`w-4 h-4 ${stakedChangePercentage < 0 ? "rotate-180" : ""}`}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M3.293 9.707a1 1 0 010-1.414l6-6a1 1 0 011.414 0l6 6a1 1 0 01-1.414 1.414L11 5.414V17a1 1 0 11-2 0V5.414L4.707 9.707a1 1 0 01-1.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                  {stakedChangePercentage >= 0 ? "+" : ""}
                  {stakedChangePercentage.toFixed(2)}%
                  <span className="text-muted-foreground ml-1">24h change</span>
                </span>
              ) : (
                <span className="text-sm text-muted-foreground">No data</span>
              )}
              {!stakedChartLoading && stakedChartData.length > 0 && (
                <div className="w-20 h-12">
                  <Line
                    key="staked-chart"
                    data={stakedChart}
                    options={chartOptions}
                  />
                </div>
              )}
            </div>
          </div>

          {/* Active Addresses Card */}
          <div className="bg-card rounded-xl p-6 border border-border relative overflow-hidden flex flex-col justify-between">
            <div className="flex items-center justify-between">
              <h3 className="text-muted-foreground text-sm font-medium mb-2">
                Active Addresses
              </h3>
              <CheckCircle className="text-primary w-5 h-5" />
            </div>

            <div className="text-3xl font-medium text-foreground mb-2">
              {activeAddressesCount} of {accounts.length}
            </div>
            <div className="flex items-center gap-2">
              <Circle className="text-primary w-3 h-3" />
              <span className="text-muted-foreground text-sm font-medium">
                All Validators Synced
              </span>
            </div>
          </div>
        </motion.div>

        {/* Address Portfolio Section */}
        <motion.div
          className="bg-card rounded-xl border border-border overflow-hidden"
          variants={cardVariants}
        >
          <div className="p-4 md:p-6 border-b border-border">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <h2 className="text-xl font-bold text-foreground">
                Address Portfolio
              </h2>
              <div className="flex items-center gap-2">
                <div className="bg-primary/20 text-primary px-3 py-1 rounded-full text-sm font-medium flex items-center gap-2">
                  Live
                </div>
              </div>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto">
            <table className="w-full min-w-[800px]">
              <thead className="bg-muted">
                <tr className="text-sm">
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Address
                  </th>
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Total Balance
                  </th>
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Staked
                  </th>
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Liquid
                  </th>
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Status
                  </th>
                  <th className="text-left p-3 md:p-4 text-muted-foreground font-medium">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredAddresses.map((address, index) => {
                  return (
                    <motion.tr
                      key={address.id}
                      className="border-b border-border/50 hover:bg-muted/30 transition-colors"
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: index * 0.1 }}
                    >
                      <td className="p-3 md:p-4">
                        <div className="flex items-center gap-3">
                          <div
                            className={`w-8 h-8 md:w-10 md:h-10 ${address.iconBg} rounded-full flex items-center justify-center flex-shrink-0`}
                          >
                            <address.icon className="text-foreground w-3 h-3 md:w-4 md:h-4" />
                          </div>
                          <div className="min-w-0">
                            <div className="text-foreground font-medium text-sm md:text-base truncate">
                              {address.nickname}
                            </div>
                            <div className="text-muted-foreground text-xs font-mono truncate">
                              {address.address}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="p-3 md:p-4">
                        <div>
                          <div className="text-foreground font-medium font-mono text-sm md:text-base whitespace-nowrap">
                            {Number(address.balance).toLocaleString()} CNPY
                          </div>
                        </div>
                      </td>
                      <td className="p-3 md:p-4">
                        <div>
                          <div className="text-foreground font-medium font-mono text-sm md:text-base whitespace-nowrap">
                            {Number(address.staked).toLocaleString()} CNPY
                          </div>
                          <div className="text-muted-foreground text-xs">
                            {address.stakedPercentage.toFixed(2)}%
                          </div>
                        </div>
                      </td>
                      <td className="p-3 md:p-4">
                        <div>
                          <div className="text-foreground font-medium font-mono text-sm md:text-base whitespace-nowrap">
                            {Number(address.liquid).toLocaleString()} CNPY
                          </div>
                          <div className="text-muted-foreground text-xs">
                            {address.liquidPercentage.toFixed(2)}%
                          </div>
                        </div>
                      </td>
                      <td className="p-3 md:p-4">
                        <span
                          className={`px-2 md:px-3 py-1 rounded-full text-xs md:text-sm font-medium ${address.statusColor} whitespace-nowrap`}
                        >
                          {address.status}
                        </span>
                      </td>
                      <td className="p-3 md:p-4">
                        <div className="flex items-center gap-1 md:gap-2">
                          {/*<button*/}
                          {/*    className="p-1.5 md:p-2 hover:bg-muted rounded-lg transition-colors group"*/}
                          {/*    onClick={() => handleViewDetails(address.fullAddress)}*/}
                          {/*    title="View Details"*/}
                          {/*>*/}
                          {/*    <i className="fa-solid fa-eye w-3 h-3 md:w-4 md:h-4 text-muted-foreground group-hover:text-primary"></i>*/}
                          {/*</button>*/}
                          <button
                            className="p-1.5 md:p-2 hover:bg-muted rounded-lg transition-colors group"
                            onClick={() =>
                              handleSendAction(address.fullAddress)
                            }
                            title="Send"
                          >
                            <Send className="w-3 h-3 md:w-4 md:h-4 text-muted-foreground group-hover:text-primary" />
                          </button>
                          {/*<button*/}
                          {/*    className="p-1.5 md:p-2 hover:bg-muted rounded-lg transition-colors group"*/}
                          {/*    onClick={() => handleMoreActions(address.fullAddress)}*/}
                          {/*    title="More Actions"*/}
                          {/*>*/}
                          {/*    <i className="fa-solid fa-ellipsis-h w-3 h-3 md:w-4 md:h-4 text-muted-foreground group-hover:text-primary"></i>*/}
                          {/*</button>*/}
                        </div>
                      </td>
                    </motion.tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </motion.div>
      </div>
    </motion.div>
  );
};

