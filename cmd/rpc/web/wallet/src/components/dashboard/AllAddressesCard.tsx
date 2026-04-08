import React, { useMemo, useCallback } from 'react';
import { motion } from 'framer-motion';
import { ChevronRight, WalletCards} from 'lucide-react';
import { useAccountData } from '@/hooks/useAccountData';
import { useAccountsList } from '@/app/providers/AccountsProvider';
import { NavLink } from 'react-router-dom';
import { StatusBadge } from '@/components/ui/StatusBadge';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';

const shortAddr = (address: string) => `${address.slice(0, 6)}…${address.slice(-4)}`;

interface AddressData {
    id: string;
    address: string;
    nickname: string;
    totalValue: string;
    status: string;
}

const AddressRow = React.memo<{ address: AddressData; index: number }>(({ address, index }) => (
    <motion.div
        className="flex items-center gap-2.5 px-3 py-2 rounded-lg border border-border/50 hover:border-primary/20 hover:bg-primary/3 transition-all duration-150"
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.18, delay: index * 0.04 }}
    >
        <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-primary/70 to-primary/30 flex items-center justify-center flex-shrink-0">
            <span className="font-mono text-[10px] font-bold text-primary-foreground">
                {address.nickname.charAt(0).toUpperCase()}
            </span>
        </div>

        <div className="flex-1 min-w-0">
            <div className="text-sm font-body font-medium text-foreground truncate leading-tight">{address.nickname}</div>
            <div className="text-xs font-mono text-muted-foreground/60 mt-0.5">{address.address}</div>
        </div>

        <div className="flex flex-col items-end gap-1 flex-shrink-0">
            <span className="text-xs font-mono font-semibold text-foreground tabular-nums">{Number(address.totalValue).toLocaleString()}</span>
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
        return info && info.staked > 0 ? 'Staked' : 'Liquid';
    }, [stakingData]);

    const processedAddresses = useMemo((): AddressData[] =>
        accounts.map(account => {
            const balance = balances.find(b => b.address === account.address)?.amount || 0;
            return {
                id: account.address,
                address: shortAddr(account.address),
                nickname: account.nickname || 'Unnamed',
                totalValue: formatBalance(balance),
                status: getStatus(account.address),
            };
        }),
        [accounts, balances, formatBalance, getStatus]
    );

    if (accountsLoading || dataLoading) {
        return (
            <motion.div
                className="canopy-card p-5 h-full"
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.35, delay: 0.22 }}
            >
                <LoadingState message="Loading addresses…" size="md" />
            </motion.div>
        );
    }

    return (
        <motion.div
            className="canopy-card p-5 h-full flex flex-col"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.35, delay: 0.22 }}
        >
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <div className="w-7 h-7 rounded-lg bg-primary/10 border border-primary/15 flex items-center justify-center">
                        <WalletCards className="text-primary" style={{ width: 13, height: 13 }} />
                    </div>
                    <span className="font-display text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                        Addresses
                    </span>
                </div>
                <NavLink
                    to="/all-addresses"
                    className="text-xs font-body text-muted-foreground hover:text-primary transition-colors font-medium flex items-center gap-1"
                >
                    All ({processedAddresses.length})
                    <ChevronRight style={{ width: 12, height: 12 }} />
                </NavLink>
            </div>

            <div className="space-y-1.5 flex-1">
                {processedAddresses.length > 0 ? (
                    processedAddresses.slice(0, 4).map((address, index) => (
                        <AddressRow key={address.id} address={address} index={index} />
                    ))
                ) : (
                    <EmptyState icon="MapPin" title="No addresses" description="Add an address to get started" size="sm" />
                )}
            </div>
        </motion.div>
    );
});

AllAddressesCard.displayName = 'AllAddressesCard';
