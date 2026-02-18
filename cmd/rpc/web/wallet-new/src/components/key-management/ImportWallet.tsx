import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useToast } from '@/toast/ToastContext';
import { useDSFetcher } from '@/core/dsFetch';
import { useQueryClient } from '@tanstack/react-query';

export const ImportWallet = (): JSX.Element => {
    const { switchAccount } = useAccounts();
    const toast = useToast();
    const dsFetch = useDSFetcher();
    const queryClient = useQueryClient();

    const [showPrivateKey, setShowPrivateKey] = useState(false);
    const [activeTab, setActiveTab] = useState<'key' | 'keystore'>('key');
    const [importForm, setImportForm] = useState({
        privateKey: '',
        password: '',
        confirmPassword: '',
        nickname: ''
    });

    const panelVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const handleImportWallet = async () => {
        if (!importForm.privateKey) {
            toast.error({ title: 'Missing private key', description: 'Please enter a private key.' });
            return;
        }

        if (!importForm.nickname) {
            toast.error({ title: 'Missing wallet name', description: 'Please enter a wallet name.' });
            return;
        }

        if (!importForm.password) {
            toast.error({ title: 'Missing password', description: 'Please enter a password.' });
            return;
        }

        if (importForm.password !== importForm.confirmPassword) {
            toast.error({ title: 'Password mismatch', description: 'Passwords do not match.' });
            return;
        }

        // Validate private key format (should be hex, 64-128 chars)
        const cleanPrivateKey = importForm.privateKey.trim().replace(/^0x/, '');
        if (!/^[0-9a-fA-F]{64,128}$/.test(cleanPrivateKey)) {
            toast.error({
                title: 'Invalid private key',
                description: 'Private key must be 64-128 hexadecimal characters.'
            });
            return;
        }

        const loadingToast = toast.info({
            title: 'Importing wallet...',
            description: 'Please wait while your wallet is imported.',
            sticky: true,
        });

        try {
            const response = await dsFetch('keystoreImportRaw', {
                nickname: importForm.nickname,
                password: importForm.password,
                privateKey: cleanPrivateKey
            });

            // Invalidate keystore cache to refetch
            await queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });

            toast.dismiss(loadingToast);
            toast.success({
                title: 'Wallet imported',
                description: `Wallet "${importForm.nickname}" has been imported successfully.`,
            });

            setImportForm({ privateKey: '', password: '', confirmPassword: '', nickname: '' });

            // Switch to the newly imported account if response contains address
            const newAddress = typeof response === 'string' ? response : (response as any)?.address;
            if (newAddress) {
                // Wait a bit for keystore to update, then try to switch
                setTimeout(() => {
                    queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });
                }, 500);
            }
        } catch (error) {
            toast.dismiss(loadingToast);
            toast.error({
                title: 'Error importing wallet',
                description: error instanceof Error ? error.message : String(error)
            });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-card rounded-lg p-6 border border-border w-full"
        >
            <div className="flex items-center gap-2 mb-6">
                <h2 className="text-xl font-bold text-foreground">Import Wallet</h2>
            </div>

            <div className="flex gap-2 mb-6 lg:w-6/12 w-full justify-between">
                <button
                    onClick={() => setActiveTab('key')}
                    className={`px-4 py-2 text-sm font-medium transition-colors bg-transparent w-full ${activeTab === 'key'
                        ? 'text-primary border-white border-b-2'
                        : 'text-muted-foreground'
                        }`}
                >
                    Key
                </button>
                <button
                    onClick={() => setActiveTab('keystore')}
                    className={`px-4 py-2  text-sm font-medium transition-colors bg-transparent w-full ${activeTab === 'keystore'
                        ? 'text-primary border-white border-b-2'
                        : 'text-muted-foreground '
                        }`}
                >
                    Keystore
                </button>
            </div>

            {activeTab === 'key' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Imported Wallet"
                            value={importForm.nickname}
                            onChange={(e) => setImportForm({ ...importForm, nickname: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Private Key
                        </label>
                        <div className="relative">
                            <input
                                type={showPrivateKey ? "text" : "password"}
                                placeholder="Enter your private key..."
                                value={importForm.privateKey}
                                onChange={(e) => setImportForm({ ...importForm, privateKey: e.target.value })}
                                className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground pr-10 placeholder:font-mono"
                            />
                            <button
                                onClick={() => setShowPrivateKey(!showPrivateKey)}
                                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                {showPrivateKey ? <i className="fa-solid fa-eye-slash text-muted-foreground text-md"></i> : <i className="fa-solid fa-eye text-muted-foreground text-md"></i>}
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Password
                        </label>
                        <input
                            type="password"
                            placeholder="Password"
                            value={importForm.password}
                            onChange={(e) => setImportForm({ ...importForm, password: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Confirm Password
                        </label>
                        <input
                            type="password"
                            placeholder="Confirm your password...."
                            value={importForm.confirmPassword}
                            onChange={(e) => setImportForm({ ...importForm, confirmPassword: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
                        <div className="flex items-start gap-3">
                            <i className="fa-solid fa-triangle-exclamation text-red-500 text-md translate-y-1"></i>
                            <div>
                                <h4 className="text-red-400 font-medium mb-1">Import Security Warning</h4>
                                <p className="text-red-300 text-sm">
                                    Only import wallets from trusted sources. Verify all information before proceeding.
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportWallet}
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium"
                    >
                        Import Wallet
                    </Button>
                </div>
            )}

            {activeTab === 'keystore' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Keystore File
                        </label>
                        <input
                            type="file"
                            accept=".json"
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-foreground file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Keystore Password
                        </label>
                        <input
                            type="password"
                            placeholder="Enter keystore password"
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-foreground"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Imported Wallet"
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-foreground"
                        />
                    </div>

                    <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
                        <div className="flex items-start gap-3">
                            <AlertTriangle className="w-5 h-5 text-red-500 mt-0.5" />
                            <div>
                                <h4 className="text-red-400 font-medium mb-1">Import Security Warning</h4>
                                <p className="text-red-300 text-sm">
                                    Only import wallets from trusted sources. Verify all information before proceeding.
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportWallet}
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium"
                    >
                        Import Keystore
                    </Button>
                </div>
            )}
        </motion.div>
    );
};

