import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Button } from '@/components/ui/Button';
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useToast } from '@/toast/ToastContext';

export const NewKey = (): JSX.Element => {
    const { createNewAccount } = useAccounts();
    const toast = useToast();

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
        if (!newKeyForm.password) {
            toast.error({ title: 'Missing password', description: 'Please enter a password.' });
            return;
        }

        if (!newKeyForm.walletName) {
            toast.error({ title: 'Missing wallet name', description: 'Please enter a wallet name.' });
            return;
        }

        const loadingToast = toast.info({
            title: 'Creating wallet...',
            description: 'Please wait while your wallet is created.',
            sticky: true,
        });

        try {
            await createNewAccount(newKeyForm.walletName, newKeyForm.password);
            toast.dismiss(loadingToast);
            toast.success({
                title: 'Wallet created',
                description: `Wallet "${newKeyForm.walletName}" is ready.`,
            });
            setNewKeyForm({ password: '', walletName: '' });
        } catch (error) {
            toast.dismiss(loadingToast);
            toast.error({
                title: 'Error creating wallet',
                description: String(error),
            });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent h-full"
        >
            <div className="flex items-center gap-2 mb-6">
                <h2 className="text-xl font-bold text-white">New Key</h2>
            </div>

            <div className="flex flex-col justify-between h-[90%]">
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Primary Wallet"
                            value={newKeyForm.walletName}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, walletName: e.target.value })}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            Password
                        </label>
                        <input
                            type="password"
                            placeholder="Password"
                            value={newKeyForm.password}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, password: e.target.value })}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>
                </div>

                <Button
                    onClick={handleCreateWallet}
                    className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium "
                >
                    Create Wallet
                </Button>
            </div>
        </motion.div>
    );
};
