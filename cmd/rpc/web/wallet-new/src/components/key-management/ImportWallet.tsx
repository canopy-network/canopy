import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle, Eye, EyeOff, KeyRound, FileJson } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useToast } from '@/toast/ToastContext';
import { useDSFetcher } from '@/core/dsFetch';
import { useQueryClient } from '@tanstack/react-query';

export const ImportWallet = (): JSX.Element => {
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
            className="bg-card rounded-2xl p-6 border border-border/80 w-full shadow-[0_14px_34px_rgba(0,0,0,0.2)]"
        >
            <div className="flex items-center justify-between gap-2 mb-6">
                <div>
                    <h2 className="text-xl font-bold text-foreground">Import Wallet</h2>
                    <p className="text-xs text-muted-foreground mt-1">Bring an existing key into this node securely.</p>
                </div>
                <span className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-[10px] font-semibold text-primary uppercase tracking-wider">
                    <KeyRound className="w-3 h-3" />
                    Recovery
                </span>
            </div>

            <div className="grid grid-cols-2 gap-2 mb-6">
                <button
                    onClick={() => setActiveTab('key')}
                    className={`px-3 py-2.5 text-sm font-medium rounded-lg transition-all border ${activeTab === 'key'
                        ? 'text-primary border-primary/40 bg-primary/10'
                        : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/60'
                        }`}
                >
                    Private Key
                </button>
                <button
                    onClick={() => setActiveTab('keystore')}
                    className={`px-3 py-2.5 text-sm font-medium rounded-lg transition-all border ${activeTab === 'keystore'
                        ? 'text-primary border-primary/40 bg-primary/10'
                        : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/60'
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
                                className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground pr-10 placeholder:font-mono focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35"
                            />
                            <button
                                onClick={() => setShowPrivateKey(!showPrivateKey)}
                                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                {showPrivateKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
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
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-semibold"
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
                        <div className="mb-2 inline-flex items-center gap-2 rounded-md border border-border bg-muted/30 px-2 py-1 text-[11px] text-muted-foreground">
                            <FileJson className="w-3.5 h-3.5" />
                            Upload encrypted JSON keystore
                        </div>
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
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-semibold"
                    >
                        Import Keystore
                    </Button>
                </div>
            )}
        </motion.div>
    );
};

