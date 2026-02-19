import React from 'react';
import { motion } from 'framer-motion';
import { Download, ShieldCheck, KeyRound, WalletCards } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { CurrentWallet } from '@/components/key-management/CurrentWallet';
import { ImportWallet } from '@/components/key-management/ImportWallet';
import { NewKey } from '@/components/key-management/NewKey';
import { useDS } from '@/core/useDs';
import { downloadJson } from '@/helpers/download';
import { useToast } from '@/toast/ToastContext';



export const KeyManagement = (): JSX.Element => {
    const toast = useToast();
    const { data: keystore } = useDS('keystore', {});
    const walletCount = Object.keys(keystore?.addressMap || {}).length;

    const containerVariants = {
        hidden: { opacity: 0 },
        visible: {
            opacity: 1,
            transition: {
                duration: 0.6,
                staggerChildren: 0.1
            }
        }
    };

    const handleDownloadKeys = () => {
        if (!keystore) {
            toast.error({
                title: 'No keys available',
                description: 'Keystore data has not loaded yet.',
            });
            return;
        }

        downloadJson(keystore, 'keystore');
        toast.success({
            title: 'Download started',
            description: 'Your keystore JSON is on its way.',
        });
    };

    return (
        <div className="bg-background min-h-screen">
            <div className="px-6 py-8 space-y-7">
                <motion.div
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.4 }}
                    className="rounded-2xl border border-primary/30 bg-gradient-to-br from-primary/15 via-card to-card p-5 md:p-6"
                >
                    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-5">
                        <div>
                            <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary mb-3">
                                <ShieldCheck className="w-3.5 h-3.5" />
                                Security Control Center
                            </div>
                            <h1 className="text-3xl md:text-4xl font-bold tracking-tight text-foreground mb-2">Key Management</h1>
                            <p className="text-sm md:text-base text-muted-foreground max-w-2xl">
                                Create, import, protect, and maintain wallet keys with explicit security safeguards.
                            </p>
                        </div>
                        <div className="flex flex-col sm:flex-row gap-3">
                            <div className="rounded-xl border border-border bg-background/80 px-4 py-3 min-w-[150px]">
                                <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-muted-foreground">
                                    <WalletCards className="w-3.5 h-3.5" />
                                    Wallets
                                </div>
                                <div className="text-xl font-bold text-foreground mt-1">{walletCount}</div>
                            </div>
                            <Button
                                className="h-12 px-5 bg-primary text-primary-foreground hover:bg-primary/90 font-semibold"
                                onClick={handleDownloadKeys}
                            >
                                <Download className="w-4 h-4 mr-2" />
                                Download Full Keystore
                            </Button>
                        </div>
                    </div>
                    <div className="mt-4 inline-flex items-center gap-2 text-xs text-muted-foreground">
                        <KeyRound className="w-3.5 h-3.5 text-primary" />
                        Always keep encrypted backups offline before deleting or rotating keys.
                    </div>
                </motion.div>

                <motion.div
                    className="grid grid-cols-1 lg:grid-cols-3 gap-6"
                    variants={containerVariants}
                    initial="hidden"
                    animate="visible"
                >
                    <CurrentWallet />
                    <ImportWallet />
                    <NewKey />
                </motion.div>

            </div>
        </div>
    );
};

