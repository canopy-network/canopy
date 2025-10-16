import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

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

const formatBalance = (amount: number) => {
    return (amount / 1000000).toFixed(2);
};

export const AddressRow: React.FC<AddressRowProps> = ({
                                                          address,
                                                          index,
                                                          onViewDetails,
                                                          onSend,
                                                          onReceive
                                                      }) => {

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'Active':
                return 'bg-green-500/20 text-green-400';
            case 'Inactive':
                return 'bg-red-500/20 text-red-400';
            case 'Pending':
                return 'bg-yellow-500/20 text-yellow-400';
            default:
                return 'bg-gray-500/20 text-gray-400';
        }
    };

    return (
        <motion.tr
            className="border-b border-bg-accent/50 hover:bg-bg-tertiary/30 transition-colors"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
        >
            <td className="p-4">
                <div className="flex items-center gap-3">
                    <div className={`w-10 h-10 ${address.iconBg} rounded-full flex items-center justify-center flex-shrink-0`}>
                        <i className={`${address.icon} text-white text-sm`}></i>
                    </div>
                    <div>
                        <div className="text-white font-medium">{formatAddress(address.address)}</div>
                        <div className="text-text-muted text-xs">{address.address}</div>
                    </div>
                </div>
            </td>
            <td className="p-4">
                <div className="text-white font-medium">{formatBalance(address.totalBalance)} CNPY</div>
            </td>
            <td className="p-4">
                <div className="text-white font-medium">{formatBalance(address.staked)} CNPY</div>
            </td>
            <td className="p-4">
                <div className="text-white font-medium">{formatBalance(address.liquid)} CNPY</div>
            </td>
            <td className="p-4">
                <span className={`text-xs px-2 py-1 rounded-full ${getStatusColor(address.status)}`}>
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
