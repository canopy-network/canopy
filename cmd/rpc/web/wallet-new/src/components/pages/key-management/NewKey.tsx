import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { User } from 'lucide-react';
import toast from 'react-hot-toast';
import { useAccounts } from '@/hooks/useAccounts';
import { useManifest } from '@/hooks/useManifest';
import { Button } from '@/components/ui/Button';

export const NewKey = (): JSX.Element => {
    const { createNewAccount } = useAccounts();
    const { getText } = useManifest();

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
            toast.error(getText('ui.newKey.errors.passwordRequired', 'Please enter a password'));
            return;
        }

        if (!newKeyForm.walletName) {
            toast.error(getText('ui.newKey.errors.walletNameRequired', 'Please enter a wallet name'));
            return;
        }

        if (newKeyForm.password.length < 8) {
            toast.error(getText('ui.newKey.errors.passwordTooShort', 'Password must be at least 8 characters long'));
            return;
        }

        const loadingToast = toast.loading(getText('ui.newKey.loading', 'Creating wallet...'));

        try {
            await createNewAccount(newKeyForm.walletName, newKeyForm.password);
            toast.success(getText('ui.newKey.success', 'Wallet created successfully'), { id: loadingToast });
            setNewKeyForm({ password: '', walletName: '' });
        } catch (error) {
            toast.error(getText('ui.newKey.errors.createError', `Error creating wallet: ${error}`), { id: loadingToast });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent h-full"
        >
            <div className="flex items-center gap-2 mb-6">
                <h2 className="text-xl font-bold text-white">{getText('ui.newKey.title', 'New Key')}</h2>
            </div>

            <div className="flex flex-col justify-between h-[90%]">
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.newKey.fields.password', 'Password')}
                        </label>
                        <input
                            type="password"
                            placeholder={getText('ui.newKey.placeholders.password', 'Password')}
                            value={newKeyForm.password}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, password: e.target.value })}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.newKey.fields.walletName', 'Wallet Name')}
                        </label>
                        <input
                            type="text"
                            placeholder={getText('ui.newKey.placeholders.walletName', 'Primary Wallet')}
                            value={newKeyForm.walletName}
                            onChange={(e) => setNewKeyForm({ ...newKeyForm, walletName: e.target.value })}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>
                </div>

                <Button
                    onClick={handleCreateWallet}
                    className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium "
                >
                    {getText('ui.newKey.buttons.createWallet', 'Create Wallet')}
                </Button>
            </div>
        </motion.div>
    );
};
