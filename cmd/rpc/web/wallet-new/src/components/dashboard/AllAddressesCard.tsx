import React, { useMemo, useCallback } from "react";
import { motion } from "framer-motion";
import { Wallet } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useAccountsList } from "@/app/providers/AccountsProvider";
import { NavLink } from "react-router-dom";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";

// Helper functions moved outside component
const formatAddress = (address: string) => {
  return address.substring(0, 6) + "..." + address.substring(address.length - 4);
};

// Address data type
interface AddressData {
  id: string;
  address: string;
  fullAddress: string;
  nickname: string;
  balance: string;
  totalValue: string;
  status: string;
}

// Memoized address row component
interface AddressRowProps {
  address: AddressData;
  index: number;
}

const AddressRow = React.memo<AddressRowProps>(({ address, index }) => (
  <motion.div
    className="p-3 bg-bg-tertiary/30 rounded-lg hover:bg-bg-tertiary/50 transition-colors"
    initial={{ opacity: 0, y: 10 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, delay: index * 0.05 }}
  >
    <div className="flex items-start gap-3">
      {/* Icon */}
      <div className="w-10 h-10 bg-gradient-to-r from-primary/80 to-primary/40 rounded-full flex items-center justify-center flex-shrink-0">
        <Wallet className="text-white w-4 h-4" />
      </div>

      {/* Content Container */}
      <div className="flex-1 min-w-0 space-y-2">
        {/* Top Row: Nickname and Address */}
        <div>
          <div className="text-text-primary text-sm font-medium mb-1 truncate">
            {address.nickname}
          </div>
          <div className="text-text-muted text-xs font-mono truncate">
            {address.address}
          </div>
        </div>

        {/* Bottom Row: Balance and Status */}
        <div className="flex items-center justify-between gap-3">
          <div className="text-text-primary text-sm font-medium whitespace-nowrap">
            {address.totalValue} CNPY
          </div>
          <StatusBadge label={address.status} size="sm" />
        </div>
      </div>
    </div>
  </motion.div>
));

AddressRow.displayName = 'AddressRow';

export const AllAddressesCard = React.memo(() => {
  // Use granular hook - only re-renders when accounts list changes
  const { accounts, loading: accountsLoading } = useAccountsList();
  const { balances, stakingData, loading: dataLoading } = useAccountData();

  const formatBalance = useCallback((amount: number) => {
    return (amount / 1000000).toFixed(2);
  }, []);

  const getAccountStatus = useCallback((address: string) => {
    const stakingInfo = stakingData.find((data) => data.address === address);
    if (stakingInfo && stakingInfo.staked > 0) {
      return "Staked";
    }
    return "Liquid";
  }, [stakingData]);

  const processedAddresses = useMemo((): AddressData[] => {
    return accounts.map((account) => {
      const balanceInfo = balances.find((b) => b.address === account.address);
      const balance = balanceInfo?.amount || 0;
      const formattedBalance = formatBalance(balance);
      const status = getAccountStatus(account.address);

      return {
        id: account.address,
        address: formatAddress(account.address),
        fullAddress: account.address,
        nickname: account.nickname || "Unnamed",
        balance: `${formattedBalance} CNPY`,
        totalValue: formattedBalance,
        status: status,
      };
    });
  }, [accounts, balances, formatBalance, getAccountStatus]);

  if (accountsLoading || dataLoading) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.4 }}
      >
        <LoadingState message="Loading addresses..." size="md" />
      </motion.div>
    );
  }

  return (
    <motion.div
      className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.4 }}
    >
      {/* Title with See All link */}
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-text-primary text-lg font-semibold">
          All Addresses
        </h3>
        <NavLink to="/all-addresses" className="text-text-muted hover:text-primary/80 text-sm font-medium transition-colors">
          See All ({processedAddresses.length})
        </NavLink>
      </div>

      {/* Addresses List */}
      <div className="space-y-3">
        {processedAddresses.length > 0 ? (
          processedAddresses.slice(0, 4).map((address, index) => (
            <AddressRow key={address.id} address={address} index={index} />
          ))
        ) : (
          <EmptyState
            icon="Wallet"
            title="No addresses found"
            description="Add an address to get started"
            size="sm"
          />
        )}
      </div>
    </motion.div>
  );
});

AllAddressesCard.displayName = 'AllAddressesCard';
