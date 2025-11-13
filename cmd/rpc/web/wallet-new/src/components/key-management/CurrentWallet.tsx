import React, { useState } from 'react';
import { motion } from 'framer-motion';
import {  Copy, Eye, EyeOff, Download, Key } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard';
import { useToast } from '@/toast/ToastContext';
import {useAccounts} from "@/app/providers/AccountsProvider";

export const CurrentWallet = (): JSX.Element => {
    const {
        accounts,
        selectedAccount,
        switchAccount
    } = useAccounts();

    const [showPrivateKey, setShowPrivateKey] = useState(false);
    const { copyToClipboard } = useCopyToClipboard();
    const toast = useToast();

    const panelVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const handleDownloadKeyfile = () => {
        if (selectedAccount) {
            // Implement keyfile download functionality
            toast.success({
                title: 'Download Ready',
                description: 'Keyfile download functionality would be implemented here',
            });
        } else {
            toast.error({
                title: 'No Account Selected',
                description: 'Please select an active account first',
            });
        }
    };

    const handleRevealPrivateKeys = () => {
        if (confirm('Are you sure you want to reveal your private keys? This is a security risk.')) {
            setShowPrivateKey(!showPrivateKey);
            toast.success({
                title: showPrivateKey ? 'Private Keys Hidden' : 'Private Keys Revealed',
                description: showPrivateKey ? 'Your keys are now hidden' : 'Be careful! Your private keys are visible',
                icon: showPrivateKey ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />,
            });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
        >
            <div className="flex items-center justify-between gap-2 mb-6">
                <h2 className="text-xl font-bold text-white">Current Wallet</h2>
                <i className="fa-solid fa-shield-halved text-primary text-2xl"></i>
            </div>

            <div className="space-y-5">
                <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                        Wallet Name
                    </label>
                    <Select value={selectedAccount?.id || ''} onValueChange={switchAccount}>
                        <SelectTrigger className="w-full bg-bg-tertiary border-bg-accent text-white h-11 rounded-lg">
                            <SelectValue placeholder="Select wallet" />
                        </SelectTrigger>
                        <SelectContent className="bg-bg-tertiary border-bg-accent">
                            {accounts.map((account) => (
                                <SelectItem key={account.id} value={account.id} className="text-white">
                                    {account.nickname}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                        Wallet Address
                    </label>
                    <div className="relative flex items-center justify-between gap-2">
                        <input
                            type="text"
                            value={selectedAccount?.address || ''}
                            readOnly
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10"
                        />
                        <button
                            onClick={() => copyToClipboard(selectedAccount?.address || '', 'Wallet address')}
                            className="text-primary-foreground hover:text-white bg-primary rounded-lg px-3 py-2.5"
                        >
                            <Copy className="w-4 h-4" />
                        </button>
                    </div>
                </div>

                <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                        Public Key
                    </label>
                    <div className="relative flex items-center justify-between gap-2">
                        <input
                            type={showPrivateKey ? 'text' : 'password'}
                            value={selectedAccount?.publicKey || ''}
                            readOnly
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10"
                        />
                        <button
                            onClick={() => setShowPrivateKey(!showPrivateKey)}
                            className="hover:text-primary bg-muted rounded-lg px-3 py-2 text-white"
                        >
                            {showPrivateKey ? <i className="fa-solid fa-eye-slash text-white text-md"></i> : <i className="fa-solid fa-eye text-white text-md"></i>}
                        </button>
                    </div>
                </div>

                <div className="flex gap-2 flex-col">
                    <Button
                        onClick={handleDownloadKeyfile}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 flex-1 py-3"
                    >
                        <Download className="w-4 h-4 mr-2" />
                        Download Keyfile
                    </Button>
                    <Button
                        onClick={handleRevealPrivateKeys}
                        variant="destructive"
                        className="flex-1 py-3"
                    >
                        <Key className="w-4 h-4 mr-2" />
                        Reveal Private Keys
                    </Button>
                </div>

                <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
                    <div className="flex items-start gap-3">
                        <i className="fa-solid fa-triangle-exclamation text-red-500 text-md translate-y-1"></i>
                        <div>
                            <h4 className="text-red-400 font-medium mb-1">Security Warning</h4>
                            <p className="text-red-300 text-sm">
                                Never share your private keys. Anyone with access to them can control your funds.
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </motion.div>
    );
};
