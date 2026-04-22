import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useToast } from '@/toast/ToastContext';
import { useDSFetcher } from '@/core/dsFetch';
import { useQueryClient } from '@tanstack/react-query';

export const NewKey = ({ embedded = false }: { embedded?: boolean }): JSX.Element => {
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

            await queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });

            toast.dismiss(loadingToast);
            toast.success({
                title: 'Wallet created',
                description: `Wallet "${newKeyForm.walletName}" is ready.`,
            });

            setNewKeyForm({ password: '', walletName: '' });

            const newAddress = typeof response === 'string' ? response : (response as any)?.address;
            if (newAddress) {
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

    const content = (
        <>
            <div className="mb-6">
                <div>
                    <h2 className="text-xl font-bold text-foreground">Create New Key</h2>
                    <p className="mt-1 text-xs text-muted-foreground">Generate a fresh encrypted wallet identity.</p>
                </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col justify-between">
                <div className="space-y-5">
                    <div>
                        <label className="mb-2 block text-sm font-medium text-foreground/80">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Primary Wallet"
                            value={newKeyForm.walletName}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, walletName: e.target.value })}
                            className="w-full rounded-lg border border-border bg-muted px-3 py-2.5 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35"
                        />
                    </div>
                    <div>
                        <label className="mb-2 block text-sm font-medium text-foreground/80">
                            Password
                        </label>
                        <input
                            type="password"
                            placeholder="Password"
                            value={newKeyForm.password}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, password: e.target.value })}
                            className="w-full rounded-lg border border-border bg-muted px-3 py-2.5 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/35"
                        />
                    </div>
                    <div className="flex items-start gap-2 rounded-lg border border-[#35cd48]/25 bg-[#35cd48]/10 px-3 py-2.5 text-xs text-[#35cd48]">
                        <ShieldCheck className="mt-0.5 h-4 w-4 text-[#35cd48]" />
                        <span className="text-[#35cd48]">
                            This key is encrypted using your password and stored in the local keystore.
                        </span>
                    </div>
                </div>

                <Button
                    onClick={handleCreateWallet}
                    className="mt-5 h-11 w-full"
                >
                    Create Wallet
                </Button>
            </div>
        </>
    );

    if (embedded) {
        return <div className="w-full">{content}</div>;
    }

    return (
        <motion.div
            variants={panelVariants}
            className="bg-card rounded-2xl border border-border/80 p-6 shadow-[0_14px_34px_rgba(0,0,0,0.2)]"
        >
            {content}
        </motion.div>
    );
};
