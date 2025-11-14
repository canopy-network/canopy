import React from "react";
import { motion } from "framer-motion";
import { Wallet, Lock, Gift } from "lucide-react";
import { Line } from "react-chartjs-2";
import AnimatedNumber from "@/components/ui/AnimatedNumber";

interface StatsCardsProps {
  totalBalance: number;
  totalStaked: number;
  totalRewards: number;
  balanceChange: number;
  stakingChange: number;
  rewardsChange: number;
  balanceChartData: any;
  stakingChartData: any;
  rewardsChartData: any;
  chartOptions: any;
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

export const StatsCards: React.FC<StatsCardsProps> = ({
  totalBalance,
  totalStaked,
  totalRewards,
  balanceChange,
  stakingChange,
  rewardsChange,
  balanceChartData,
  stakingChartData,
  rewardsChartData,
  chartOptions,
}) => {
  const statsData = [
    {
      id: "totalBalance",
      title: "Total Balance",
      value: totalBalance,
      change: balanceChange,
      chartData: balanceChartData,
      icon: Wallet,
      iconColor: "text-primary",
    },
    {
      id: "totalStaked",
      title: "Total Staked",
      value: totalStaked,
      change: stakingChange,
      chartData: stakingChartData,
      icon: Lock,
      iconColor: "text-primary",
    },
    {
      id: "totalRewards",
      title: "Total Rewards",
      value: totalRewards,
      change: rewardsChange,
      chartData: rewardsChartData,
      icon: Gift,
      iconColor: "text-primary",
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      {statsData.map((stat) => (
        <motion.div
          key={stat.id}
          variants={itemVariants}
          className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden"
        >
          <div className="flex items-center justify-between">
            <h3 className="text-text-muted text-sm font-medium mb-2">
              {stat.title}
            </h3>
            <stat.icon className={`${stat.iconColor} w-5 h-5`} />
          </div>
          <div className="text-3xl font-medium text-white mb-2">
            <AnimatedNumber
              value={stat.value / 1000000}
              format={{
                notation: "standard",
                maximumFractionDigits: 2,
              }}
            />
            &nbsp;CNPY
          </div>
          <div className="flex items-center justify-between">
            <span
              className={`text-sm font-medium ${stat.change >= 0 ? "text-primary" : "text-red-400"}`}
            >
              {stat.change >= 0 ? "+" : ""}
              {stat.change.toFixed(1)}%
              <span className="text-text-muted text-sm font-medium">
                {" "}
                24h change
              </span>
            </span>
            <div className="w-20 h-12">
              <Line data={stat.chartData} options={chartOptions} />
            </div>
          </div>
        </motion.div>
      ))}
    </div>
  );
};
