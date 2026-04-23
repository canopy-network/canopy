import React from 'react';
import { motion } from 'framer-motion';
import { Copy } from 'lucide-react';
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard';
import { useManifest } from '@/hooks/useManifest';
import { useDenom } from '@/hooks/useDenom';
import { WALLET_BADGE_CLASS, WALLET_BADGE_TONE } from '@/components/ui/badgeStyles';

interface Address {
    id: string;
    address: string;
    totalBalance: number;
    staked: number;
    liquid: number;
    status: 'Active' | 'Inactive' | 'Pending';
    icon: string;
    iconBg: string;
}

interface AddressRowProps {
    address: Address;
    index: number;
    onViewDetails: (address: string) => void;
    onSend: (address: string) => void;
    onReceive: (address: string) => void;
}

const formatAddress = (address: string) => {
    return address.substring(0, 5) + '...' + address.substring(address.length - 6);
};

const formatBalance = (amount: number, factor: number) => {
    return (amount / factor).toFixed(2);
};

export const AddressRow: React.FC<AddressRowProps> = ({
                                                          address,
                                                          index,
                                                          onViewDetails,
                                                          onSend,
                                                          onReceive
                                                      }) => {
    const { copyToClipboard } = useCopyToClipboard();
    const { symbol, factor } = useDenom();

    const getStatusColor = (status: string) => {
        return WALLET_BADGE_TONE;
    };

    return (
        <motion.tr
            className="border-b border-border/50 hover:bg-muted/30 transition-colors"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
        >
            <td className="p-4">
                <div className="flex items-center gap-3">
                    <div className={`w-10 h-10 ${address.iconBg} rounded-full flex items-center justify-center flex-shrink-0`}>
                        <i className={`${address.icon} text-foreground text-sm`}></i>
                    </div>
                    <div>
                        <div className="flex items-center gap-1.5">
                            <span className="text-foreground font-medium">{formatAddress(address.address)}</span>
                            <button
                                onClick={(e) => { e.stopPropagation(); copyToClipboard(address.address, "Address"); }}
                                className="p-1 rounded hover:bg-accent/60 text-muted-foreground hover:text-foreground transition-colors"
                                aria-label="Copy address"
                            >
                                <Copy className="w-3.5 h-3.5" />
                            </button>
                        </div>
                        <div className="text-muted-foreground text-xs">{address.address}</div>
                    </div>
                </div>
            </td>
            <td className="p-4">
                <div className="text-foreground font-medium">{formatBalance(address.totalBalance, factor)} {symbol}</div>
            </td>
            <td className="p-4">
                <div className="text-foreground font-medium">{formatBalance(address.staked, factor)} {symbol}</div>
            </td>
            <td className="p-4">
                <div className="text-foreground font-medium">{formatBalance(address.liquid, factor)} {symbol}</div>
            </td>
            <td className="p-4">
                <span className={`${WALLET_BADGE_CLASS} ${getStatusColor(address.status)}`}>
                    {address.status}
                </span>
            </td>
            <td className="p-4">
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => onViewDetails(address.address)}
                        className="text-primary hover:text-primary/80 text-sm font-medium"
                    >
                      View Details
                    </button>
                    <button
                        onClick={() => onSend(address.address)}
                        className="text-primary hover:text-primary/80 text-sm font-medium"
                    >
                      Send
                    </button>
                    <button
                        onClick={() => onReceive(address.address)}
                        className="text-primary hover:text-primary/80 text-sm font-medium"
                    >
                        Receive
                    </button>
                </div>
            </td>
        </motion.tr>
    );
};
