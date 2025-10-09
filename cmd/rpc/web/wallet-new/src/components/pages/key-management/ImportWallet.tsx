import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { FileText, Eye, EyeOff, AlertTriangle } from 'lucide-react';
import toast from 'react-hot-toast';
import { useAccounts } from '@/hooks/useAccounts';
import { useManifest } from '@/hooks/useManifest';
import { Button } from '@/components/ui/Button';

export const ImportWallet = (): JSX.Element => {
    const { createNewAccount } = useAccounts();
    const { getText } = useManifest();

    const [showPrivateKey, setShowPrivateKey] = useState(false);
    const [activeTab, setActiveTab] = useState<'key' | 'keystore'>('key');
    const [importForm, setImportForm] = useState({
        privateKey: '',
        password: '',
        confirmPassword: ''
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
            toast.error(getText('ui.importWallet.errors.privateKeyRequired', 'Please enter a private key'));
            return;
        }

        if (!importForm.password) {
            toast.error(getText('ui.importWallet.errors.passwordRequired', 'Please enter a password'));
            return;
        }

        if (importForm.password !== importForm.confirmPassword) {
            toast.error(getText('ui.importWallet.errors.passwordsDoNotMatch', 'Passwords do not match'));
            return;
        }

        const loadingToast = toast.loading(getText('ui.importWallet.loading', 'Importing wallet...'));

        try {
            // Here you would implement the import functionality
            // For now, we'll create a new account with the provided name
            await createNewAccount(importForm.password, getText('ui.importWallet.defaultName', 'Imported Wallet'));
            toast.success(getText('ui.importWallet.success', 'Wallet imported successfully'), { id: loadingToast });
            setImportForm({ privateKey: '', password: '', confirmPassword: '' });
        } catch (error) {
            toast.error(getText('ui.importWallet.errors.importError', `Error importing wallet: ${error}`), { id: loadingToast });
        }
    };

    return (
        <motion.div
            variants={panelVariants}
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent w-full"
        >
            <div className="flex items-center gap-2 mb-6">
                <h2 className="text-xl font-bold text-white">{getText('ui.importWallet.title', 'Import Wallet')}</h2>
            </div>

            <div className="flex gap-2 mb-6 lg:w-6/12 w-full justify-between">
                <button
                    onClick={() => setActiveTab('key')}
                    className={`px-4 py-2 text-sm font-medium transition-colors bg-transparent w-full ${activeTab === 'key'
                        ? 'text-primary border-white border-b-2'
                        : 'text-gray-400'
                        }`}
                >
                    {getText('ui.importWallet.tabs.key', 'Key')}
                </button>
                <button
                    onClick={() => setActiveTab('keystore')}
                    className={`px-4 py-2  text-sm font-medium transition-colors bg-transparent w-full ${activeTab === 'keystore'
                        ? 'text-primary border-white border-b-2'
                        : 'text-gray-400 '
                        }`}
                >
                    {getText('ui.importWallet.tabs.keystore', 'Keystore')}
                </button>
            </div>

            {activeTab === 'key' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.privateKey', 'Private Key')}
                        </label>
                        <div className="relative">
                            <input
                                type="password"
                                placeholder={getText('ui.importWallet.placeholders.privateKey', 'Enter your private key...')}
                                value={importForm.privateKey}
                                onChange={(e) => setImportForm({ ...importForm, privateKey: e.target.value })}
                                className="w-full bg-bg-tertiary border border-gray-600 rounded-lg px-3 py-2.5 text-white pr-10 placeholder:font-mono"
                            />
                            <button
                                onClick={() => setShowPrivateKey(!showPrivateKey)}
                                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-white"
                            >
                                {showPrivateKey ? <i className="fa-solid fa-eye-slash text-gray-400 text-md"></i> : <i className="fa-solid fa-eye text-gray-400 text-md"></i>}
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.walletPassword', 'Wallet Password')}
                        </label>
                        <input
                            type="password"
                            placeholder={getText('ui.importWallet.placeholders.password', 'Password')}
                            value={importForm.password}
                            onChange={(e) => setImportForm({ ...importForm, password: e.target.value })}
                            className="w-full bg-bg-tertiary border border-gray-600 rounded-lg px-3 py-2.5 text-white"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.confirmPassword', 'Confirm Password')}
                        </label>
                        <input
                            type="password"
                            placeholder={getText('ui.importWallet.placeholders.confirmPassword', 'Confirm your password....')}
                            value={importForm.confirmPassword}
                            onChange={(e) => setImportForm({ ...importForm, confirmPassword: e.target.value })}
                            className="w-full bg-bg-tertiary border border-gray-600 rounded-lg px-3 py-2.5 text-white"
                        />
                    </div>

                    <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
                        <div className="flex items-start gap-3">
                            <i className="fa-solid fa-triangle-exclamation text-red-500 text-md translate-y-1"></i>
                            <div>
                                <h4 className="text-red-400 font-medium mb-1">{getText('ui.importWallet.securityWarning.title', 'Import Security Warning')}</h4>
                                <p className="text-red-300 text-sm">
                                    {getText('ui.importWallet.securityWarning.message', 'Only import wallets from trusted sources. Verify all information before proceeding.')}
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportWallet}
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium"
                    >
                        {getText('ui.importWallet.buttons.importWallet', 'Import Wallet')}
                    </Button>
                </div>
            )}

            {activeTab === 'keystore' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.keystoreFile', 'Keystore File')}
                        </label>
                        <input
                            type="file"
                            accept=".json"
                            className="w-full bg-bg-tertiary border border-gray-600 rounded-lg px-3 py-2 text-white file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.keystorePassword', 'Keystore Password')}
                        </label>
                        <input
                            type="password"
                            placeholder={getText('ui.importWallet.placeholders.keystorePassword', 'Enter keystore password')}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-2">
                            {getText('ui.importWallet.fields.walletName', 'Wallet Name')}
                        </label>
                        <input
                            type="text"
                            placeholder={getText('ui.importWallet.placeholders.walletName', 'Imported Wallet')}
                            className="w-full bg-bg-tertiary border border-bg-accent rounded-lg px-3 py-2 text-white"
                        />
                    </div>

                    <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
                        <div className="flex items-start gap-3">
                            <AlertTriangle className="w-5 h-5 text-red-500 mt-0.5" />
                            <div>
                                <h4 className="text-red-400 font-medium mb-1">{getText('ui.importWallet.securityWarning.title', 'Import Security Warning')}</h4>
                                <p className="text-red-300 text-sm">
                                    {getText('ui.importWallet.securityWarning.message', 'Only import wallets from trusted sources. Verify all information before proceeding.')}
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportWallet}
                        className="w-full bg-primary text-primary-foreground hover:bg-primary/90 h-11 font-medium"
                    >
                        {getText('ui.importWallet.buttons.importKeystore', 'Import Keystore')}
                    </Button>
                </div>
            )}
        </motion.div>
    );
};
