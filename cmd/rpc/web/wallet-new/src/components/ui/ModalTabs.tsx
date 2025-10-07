import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

interface ModalTabsProps {
    activeTab: 'send' | 'receive' | 'stake' | 'swap';
    onTabChange: (tab: 'send' | 'receive' | 'stake' | 'swap') => void;
    onClose: () => void;
}

export const ModalTabs: React.FC<ModalTabsProps> = ({
    activeTab,
    onTabChange,
    onClose
}) => {
    const { getText } = useManifest();
    return (
        <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-6">
                <button
                    onClick={() => onTabChange('send')}
                    className={`flex items-center gap-2 font-semibold transition-all duration-300 ease-in-out ${activeTab === 'send'
                            ? 'text-primary border-b-2 border-primary pb-2'
                            : 'text-text-muted hover:text-text-primary'
                        }`}
                >
                    <motion.i
                        className="fa-solid fa-paper-plane text-lg"
                        animate={{
                            scale: activeTab === 'send' ? 1.1 : 1,
                            rotate: activeTab === 'send' ? 0 : 0
                        }}
                        transition={{ duration: 0.2 }}
                    ></motion.i>
                            {getText('ui.tabs.send', 'Send')}
                </button>
                <button
                    onClick={() => onTabChange('receive')}
                    className={`flex items-center gap-2 font-semibold transition-all duration-300 ease-in-out ${activeTab === 'receive'
                            ? 'text-primary border-b-2 border-primary pb-2'
                            : 'text-text-muted hover:text-text-primary'
                        }`}
                >
                    <motion.i
                        className="fa-solid fa-qrcode text-lg"
                        animate={{
                            scale: activeTab === 'receive' ? 1.1 : 1,
                            rotate: activeTab === 'receive' ? 0 : 0
                        }}
                        transition={{ duration: 0.2 }}
                    ></motion.i>
                            {getText('ui.tabs.receive', 'Receive')}
                </button>
                        <button
                            onClick={() => onTabChange('stake')}
                            className={`flex items-center gap-2 font-semibold transition-all duration-300 ease-in-out ${activeTab === 'stake'
                                ? 'text-primary border-b-2 border-primary pb-2'
                                : 'text-text-muted hover:text-text-primary'
                                }`}
                        >
                            <motion.i
                                className="fa-solid fa-lock text-lg"
                                animate={{
                                    scale: activeTab === 'stake' ? 1.1 : 1,
                                    rotate: activeTab === 'stake' ? 0 : 0
                                }}
                                transition={{ duration: 0.2 }}
                            ></motion.i>
                            {getText('ui.tabs.stake', 'Stake')}
                        </button>
                        <button
                            onClick={() => onTabChange('swap')}
                            className={`flex items-center gap-2 font-semibold transition-all duration-300 ease-in-out ${activeTab === 'swap'
                                ? 'text-primary border-b-2 border-primary pb-2'
                                : 'text-text-muted hover:text-text-primary'
                                }`}
                        >
                            <motion.i
                                className="fa-solid fa-exchange-alt text-lg"
                                animate={{
                                    scale: activeTab === 'swap' ? 1.1 : 1,
                                    rotate: activeTab === 'swap' ? 0 : 0
                                }}
                                transition={{ duration: 0.2 }}
                            ></motion.i>
                            {getText('ui.tabs.swap', 'Swap')}
                        </button>
            </div>
            <button
                onClick={onClose}
                className="text-text-muted hover:text-text-primary transition-colors"
            >
                <i className="fa-solid fa-times text-lg"></i>
            </button>
        </div>
    );
};
