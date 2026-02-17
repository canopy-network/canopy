import React, { useMemo, useCallback } from "react";
import { motion } from "framer-motion";
import { Wallet } from "lucide-react";
import { useAccountData } from "@/hooks/useAccountData";
import { useAccountsList } from "@/app/providers/AccountsProvider";
import { NavLink } from "react-router-dom";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";

const formatAddress = (address: string) =>
    `${address.slice(0, 6)}…${address.slice(-4)}`;

interface AddressData {
  id: string;
  address: string;
  nickname: string;
  totalValue: string;
  status: string;
}

const AddressRow = React.memo<{ address: AddressData; index: number }>(({ address, index }) => (
  <motion.div
    className="flex items-center gap-3 px-3 py-2.5 rounded-xl border border-white/[0.06] hover:border-white/10 hover:bg-white/[0.02] transition-all duration-150"
    initial={{ opacity: 0, y: 8 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, delay: index * 0.05 }}
  >
    <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center flex-shrink-0">
      <Wallet className="w-3.5 h-3.5 text-white" />
    </div>

    <div className="flex-1 min-w-0">
      <div className="text-sm font-medium text-white truncate leading-tight">{address.nickname}</div>
      <div className="text-xs text-back font-mono mt-0.5">{address.address}</div>
    </div>

    <div className="flex flex-col items-end gap-1 flex-shrink-0">
      <span className="text-sm font-semibold text-white tabular-nums">{Number(address.totalValue).toLocaleString()}</span>
      <StatusBadge label={address.status} size="sm" />
    </div>
  </motion.div>
));

AddressRow.displayName = 'AddressRow';

export const AllAddressesCard = React.memo(() => {
  const { accounts, loading: accountsLoading } = useAccountsList();
  const { balances, stakingData, loading: dataLoading } = useAccountData();

  const formatBalance = useCallback((amount: number) => (amount / 1_000_000).toFixed(2), []);

  const getStatus = useCallback((address: string) => {
    const info = stakingData.find(d => d.address === address);
    return info && info.staked > 0 ? "Staked" : "Liquid";
  }, [stakingData]);

  const processedAddresses = useMemo((): AddressData[] =>
    accounts.map(account => {
      const balance = balances.find(b => b.address === account.address)?.amount || 0;
      return {
        id: account.address,
        address: formatAddress(account.address),
        nickname: account.nickname || "Unnamed",
        totalValue: formatBalance(balance),
        status: getStatus(account.address),
      };
    }),
    [accounts, balances, formatBalance, getStatus]
  );

  if (accountsLoading || dataLoading) {
    return (
      <motion.div
        className="rounded-2xl p-6 border border-white/10 h-full"
        style={{ background: '#22232E' }}
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.3 }}
      >
        <LoadingState message="Loading addresses..." size="md" />
      </motion.div>
    );
  }

  return (
    <motion.div
      className="rounded-2xl p-6 border border-white/10 h-full flex flex-col"
      style={{ background: '#22232E' }}
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.3 }}
    >
      <div className="flex items-center justify-between mb-5">
        <span className="text-xs font-medium text-back uppercase tracking-wider">All Addresses</span>
        <NavLink
          to="/all-addresses"
          className="text-xs text-back hover:text-primary transition-colors font-medium"
        >
          See all ({processedAddresses.length})
        </NavLink>
      </div>

      <div className="space-y-2 flex-1">
        {processedAddresses.length > 0 ? (
          processedAddresses.slice(0, 4).map((address, index) => (
            <AddressRow key={address.id} address={address} index={index} />
          ))
        ) : (
          <EmptyState icon="Wallet" title="No addresses found" description="Add an address to get started" size="sm" />
        )}
      </div>
    </motion.div>
  );
});

AllAddressesCard.displayName = 'AllAddressesCard';
