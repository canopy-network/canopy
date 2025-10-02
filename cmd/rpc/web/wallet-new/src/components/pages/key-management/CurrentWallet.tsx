import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Shield, Copy, Eye, EyeOff, Download, Key, AlertTriangle } from 'lucide-react';
import toast from 'react-hot-toast';
import { useAccounts } from '@/hooks/useAccounts';
import { Button } from '@/components/ui/Button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';

export const CurrentWallet = (): JSX.Element => {
    const {
        accounts,
        activeAccount,
        switchAccount
    } = useAccounts();

    const [showPrivateKey, setShowPrivateKey] = useState(false);

    const panelVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const handleDownloadKeyfile = () => {
        if (activeAccount) {
            // Implement keyfile download functionality
            toast.success('Keyfile download functionality would be implemented here');
        } else {
            toast.error('No active account selected');
        }
    };

    const handleRevealPrivateKeys = () => {
        if (confirm('Are you sure you want to reveal your private keys? This is a security risk.')) {
            setShowPrivateKey(!showPrivateKey);
            toast.success(showPrivateKey ? 'Private keys hidden' : 'Private keys revealed');
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        toast.success('Copied to clipboard');
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
                    <Select value={activeAccount?.id || ''} onValueChange={switchAccount}>
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
                            value={activeAccount?.address || ''}
                            readOnly
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2.5 text-white pr-10"
                        />
                        <button
                            onClick={() => copyToClipboard(activeAccount?.address || '')}
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
                            value={activeAccount?.publicKey || ''}
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
