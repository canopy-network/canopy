import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Sparkles, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useToast } from '@/toast/ToastContext';
import { useDSFetcher } from '@/core/dsFetch';
import { useQueryClient } from '@tanstack/react-query';

export const NewKey = (): JSX.Element => {
    const { switchAccount } = useAccounts();
    const toast = useToast();
    const dsFetch = useDSFetcher();
    const queryClient = useQueryClient();

    const [newKeyForm, setNewKeyForm] = useState({
        password: '',
        walletName: ''
    });

    const panelVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const handleCreateWallet = async () => {
        if (!newKeyForm.walletName) {
            toast.error({ title: 'Missing wallet name', description: 'Please enter a wallet name.' });
            return;
        }

        if (!newKeyForm.password) {
            toast.error({ title: 'Missing password', description: 'Please enter a password.' });
            return;
        }

        const loadingToast = toast.info({
            title: 'Creating wallet...',
            description: 'Please wait while your wallet is created.',
            sticky: true,
        });

        try {
            const response = await dsFetch('keystoreNewKey', {
                nickname: newKeyForm.walletName,
                password: newKeyForm.password
            });

            // Invalidate keystore cache to refetch
            await queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });

            toast.dismiss(loadingToast);
            toast.success({
                title: 'Wallet created',
                description: `Wallet "${newKeyForm.walletName}" is ready.`,
            });

            setNewKeyForm({ password: '', walletName: '' });

            // Switch to the newly created account if response contains address
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
                title: 'Error creating wallet',
                description: error instanceof Error ? error.message : String(error),
            });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-card rounded-2xl p-6 border border-border/80 h-full shadow-[0_14px_34px_rgba(0,0,0,0.2)]"
        >
            <div className="flex items-center justify-between gap-2 mb-6">
                <div>
                    <h2 className="text-xl font-bold text-foreground">Create New Key</h2>
                    <p className="text-xs text-muted-foreground mt-1">Generate a fresh encrypted wallet identity.</p>
                </div>
                <span className="inline-flex items-center gap-1 rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-[10px] font-semibold text-primary uppercase tracking-wider">
                    <Sparkles className="w-3 h-3" />
                    New
                </span>
            </div>

            <div className="flex flex-col justify-between h-full min-h-0">
                <div className="space-y-5">
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Primary Wallet"
                            value={newKeyForm.walletName}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, walletName: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Password
                        </label>
                        <input
                            type="password"
                            placeholder="Password"
                            value={newKeyForm.password}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, password: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35"
                        />
                    </div>
                    <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/10 px-3 py-2.5 text-xs text-emerald-300 flex items-start gap-2">
                        <ShieldCheck className="w-4 h-4 mt-0.5" />
                        This key is encrypted using your password and stored in the local keystore.
                    </div>
                </div>

                <Button
                    onClick={handleCreateWallet}
                    className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-semibold mt-5"
                >
                    Create Wallet
                </Button>
            </div>
        </motion.div>
    );
};

