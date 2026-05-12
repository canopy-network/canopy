import React, { useMemo, useCallback } from 'react';
import { motion } from 'framer-motion';
import { ArrowUp, ChevronRight } from 'lucide-react';
import { useActionModal } from '@/app/providers/ActionModalProvider';
import { useAccountData } from '@/hooks/useAccountData';
import { useAccountsList, useSelectedAccount } from '@/app/providers/AccountsProvider';
import { NavLink, useNavigate } from 'react-router-dom';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';
import { getCanopySymbol } from '@/lib/utils/canopySymbols';
import { useDenom } from '@/hooks/useDenom';
import { CopyableIdentifier } from '@/components/ui/CopyableIdentifier';

const shortAddr = (address: string) => `${address.slice(0, 6)}…${address.slice(-4)}`;

interface AddressData {
    id: string;
    address: string;
    fullAddress: string;
    nickname: string;
    totalValue: string;
}

const AddressRow = React.memo<{
    address: AddressData;
    index: number;
    onClick?: () => void;
    onSend?: (address: AddressData) => void;
}>(({ address, index, onClick, onSend }) => (
    <motion.div
        className="flex items-center gap-3 px-3.5 py-3 rounded-lg hover:bg-primary/3 transition-all duration-150 cursor-pointer"
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.18, delay: index * 0.04 }}
        onClick={onClick}
    >
        <img src={getCanopySymbol(index)} alt="" className="w-7 h-7 rounded-lg object-contain flex-shrink-0" />

        <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-foreground truncate leading-tight">{address.nickname}</div>
            <CopyableIdentifier value={address.fullAddress} label="Address" className="mt-0.5 max-w-full text-xs text-muted-foreground/60">
                {address.address}
            </CopyableIdentifier>
        </div>

        <div className="ml-auto flex items-center gap-2.5 flex-shrink-0">
            <span className="text-xs font-semibold text-foreground tabular-nums whitespace-nowrap">{Number(address.totalValue).toLocaleString()}</span>
            <button
                type="button"
                className="inline-flex items-center justify-center rounded-md border border-border/50 p-1.5 text-muted-foreground transition-colors hover:border-white/15 hover:bg-primary/5 hover:text-foreground"
                onClick={(event) => {
                    event.stopPropagation();
                    onSend?.(address);
                }}
                aria-label={`Send from ${address.nickname}`}
            >
                <ArrowUp style={{ width: 12, height: 12 }} />
            </button>
        </div>
    </motion.div>
));

AddressRow.displayName = 'AddressRow';

export const AllAddressesCard = React.memo(() => {
    const { accounts, loading: accountsLoading } = useAccountsList();
    const { switchAccount } = useSelectedAccount();
    const { openAction } = useActionModal();
    const navigate = useNavigate();
    const { balances, stakingData, loading: dataLoading } = useAccountData();
    const { factor } = useDenom();

    const formatBalance = useCallback((amount: number) => (amount / factor).toFixed(2), [factor]);

    const getStatus = useCallback((address: string) => {
        const info = stakingData.find(d => d.address === address);
        return info && info.staked > 0 ? 'Staked' : 'Liquid';
    }, [stakingData]);

    const handleSendAction = useCallback((address: AddressData) => {
        switchAccount(address.id);
        openAction('send');
    }, [openAction, switchAccount]);

    const processedAddresses = useMemo((): AddressData[] =>
        accounts.map(account => {
            const balance = balances.find(b => b.address === account.address)?.amount || 0;
            return {
                id: account.address,
                address: shortAddr(account.address),
                fullAddress: account.address,
                nickname: account.nickname || 'Unnamed',
                totalValue: formatBalance(balance),
            };
        }),
        [accounts, balances, formatBalance]
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
                    <span className="wallet-card-title">
                        Addresses
                    </span>
                </div>
                <NavLink
                    to="/accounts"
                    className="text-xs text-muted-foreground hover:text-primary transition-colors font-medium flex items-center gap-1"
                >
                    All ({processedAddresses.length})
                    <ChevronRight style={{ width: 12, height: 12 }} />
                </NavLink>
            </div>

            <div className="space-y-1.5 flex-1">
                {processedAddresses.length > 0 ? (
                    processedAddresses.slice(0, 4).map((address, index) => (
                        <AddressRow
                            key={address.id}
                            address={address}
                            index={index}
                            onSend={handleSendAction}
                            onClick={() => {
                                switchAccount(address.id);
                                navigate('/accounts');
                            }}
                        />
                    ))
                ) : (
                    <EmptyState icon="MapPin" title="No addresses" description="Add an address to get started" size="sm" />
                )}
            </div>
        </motion.div>
    );
});

AllAddressesCard.displayName = 'AllAddressesCard';
