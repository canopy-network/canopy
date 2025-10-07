import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { SendModal } from '@/components/ui/SendModal';
import { useManifest } from '@/hooks/useManifest';

interface QuickActionsCardProps {
    manifest?: any;
}

export const QuickActionsCard = ({ manifest }: QuickActionsCardProps) => {
    const { getText } = useManifest();
    const [sendModalOpen, setSendModalOpen] = useState(false);
    const [receiveModalOpen, setReceiveModalOpen] = useState(false);
    const [stakeModalOpen, setStakeModalOpen] = useState(false);
    const [swapModalOpen, setSwapModalOpen] = useState(false);

    const actions = [
        {
            id: 'send',
            label: getText('ui.tabs.send', 'Send'),
            icon: "fa-solid fa-paper-plane text-muted text-2xl",
            color: 'bg-primary hover:bg-primary/90 text-muted',
            textColor: 'text-muted',
            action: () => setSendModalOpen(true)
        },
        {
            id: 'receive',
            label: getText('ui.tabs.receive', 'Receive'),
            icon: "fa-solid fa-qrcode text-primary text-2xl",
            color: 'bg-bg-tertiary hover:bg-bg-accent',
            textColor: 'text-white',
            action: () => setReceiveModalOpen(true)
        },
        {
            id: 'stake',
            label: getText('ui.tabs.stake', 'Stake'),
            icon: "fa-solid fa-lock text-primary text-2xl",
            color: 'bg-bg-tertiary hover:bg-bg-accent',
            textColor: 'text-white',
            action: () => setStakeModalOpen(true)
        },
        {
            id: 'swap',
            label: getText('ui.tabs.swap', 'Swap'),
            icon: "fa-solid fa-exchange-alt text-primary text-2xl",
            color: 'bg-bg-tertiary hover:bg-bg-accent',
            textColor: 'text-white',
            action: () => setSwapModalOpen(true)
        }
    ];

    return (
        <motion.div
            className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
        >
            {/* Title */}
            <h3 className="text-text-muted text-sm font-medium mb-6">
                {getText('ui.quickActions.title', 'Quick Actions')}
            </h3>

            {/* Actions Grid */}
            <div className="grid grid-cols-2 gap-3">
                {actions.map((action, index) => {
                    return (
                        <motion.button
                            key={action.id}
                            className={`${action.color} text-text-primary rounded-lg p-4 flex flex-col items-center gap-2 transition-all duration-200`}
                            onClick={action.action}
                            initial={{ opacity: 0, scale: 0.8 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{
                                duration: 0.3,
                                delay: 0.3 + (index * 0.1),
                                type: "spring",
                                stiffness: 200
                            }}
                            whileHover={{
                                scale: 1.05,
                                transition: { duration: 0.2 }
                            }}
                            whileTap={{ scale: 0.95 }}
                        >
                            <i className={`${action.icon}`}></i>
                            <span className={`text-sm font-medium ${action.textColor}`}>{action.label}</span>
                        </motion.button>
                    );
                })}
            </div>

            {/* Send Modal */}
            <SendModal
                isOpen={sendModalOpen}
                onClose={() => setSendModalOpen(false)}
            />

            {/* Receive Modal */}
            <SendModal
                isOpen={receiveModalOpen}
                onClose={() => setReceiveModalOpen(false)}
                defaultTab="receive"
            />

                    {/* Stake Modal */}
                    <SendModal
                        isOpen={stakeModalOpen}
                        onClose={() => setStakeModalOpen(false)}
                        defaultTab="stake"
                    />

                    {/* Swap Modal */}
                    <SendModal
                        isOpen={swapModalOpen}
                        onClose={() => setSwapModalOpen(false)}
                        defaultTab="swap"
                    />
        </motion.div>
    );
};