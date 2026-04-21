import React from 'react';
import { motion } from 'framer-motion';
import { CircleHelp, Download, KeyRound, Sparkles, Wallet } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { CurrentWallet } from '@/components/key-management/CurrentWallet';
import { ImportWallet } from '@/components/key-management/ImportWallet';
import { NewKey } from '@/components/key-management/NewKey';
import { ActionTooltip } from '@/components/ui/ActionTooltip';
import { Dialog, DialogContent } from '@/components/ui/Dialog';
import { useDS } from '@/core/useDs';
import { downloadJson } from '@/helpers/download';
import { useToast } from '@/toast/ToastContext';
import { PageHeader } from '@/components/layouts/PageHeader';

type KeyManagementModal = 'import' | 'create' | null;

export const KeyManagement = (): JSX.Element => {
    const toast = useToast();
    const { data: keystore } = useDS('keystore', {});
    const [activeModal, setActiveModal] = React.useState<KeyManagementModal>(null);

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

    const actionCards = [
        {
            id: 'import' as const,
            title: 'Import Wallet',
            description: 'Recover an existing wallet from a private key or encrypted keystore JSON.',
            help: 'Use this to bring an existing trusted wallet into the local keystore.',
            icon: KeyRound,
        },
        {
            id: 'create' as const,
            title: 'Create New Key',
            description: 'Generate a fresh encrypted wallet identity and store it in the local keystore.',
            help: 'Creates a brand-new wallet and secures it with your chosen password.',
            icon: Sparkles,
        },
    ];

    return (
        <div className="space-y-6 pb-16 lg:pb-8">
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4 }}
                className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"
            >
                <PageHeader
                    title="Key Management"
                    subtitle="Create, import, back up, and manage wallet keys from one place."
                    actions={
                        <Button
                            className="h-12 border border-[#272729] bg-[#0f0f0f] px-5 text-white hover:bg-[#272729]"
                            onClick={handleDownloadKeys}
                        >
                            <Download className="h-4 w-4" />
                            Download Full Keystore
                        </Button>
                    }
                    className="w-full"
                />
            </motion.div>

            <motion.div
                className="grid grid-cols-1 gap-6"
                variants={containerVariants}
                initial="hidden"
                animate="visible"
            >
                <CurrentWallet />

                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                {actionCards.map((item) => {
                    const Icon = item.icon;
                    return (
                        <motion.button
                            key={item.id}
                            whileHover={{ y: -2 }}
                            whileTap={{ scale: 0.99 }}
                            onClick={() => setActiveModal(item.id)}
                            className="group rounded-xl border border-[#272729] bg-[#171717] px-4 py-4 text-left transition-all duration-200 hover:border-white/15 hover:bg-[#0f0f0f]"
                        >
                            <div className="mb-2 flex items-center gap-2">
                                <span className="inline-flex h-7 w-7 items-center justify-center text-[#35cd48]">
                                    <Icon className="h-4 w-4" />
                                </span>
                                <span className="text-sm font-semibold text-foreground">{item.title}</span>
                                <ActionTooltip
                                    label={item.title}
                                    description={item.help}
                                    className="ml-auto"
                                >
                                    <span
                                        className="inline-flex h-5 w-5 items-center justify-center rounded-full border border-[#272729] text-muted-foreground hover:bg-[#272729] hover:text-foreground"
                                        tabIndex={0}
                                        aria-label={`${item.title} help`}
                                    >
                                        <CircleHelp className="h-3.5 w-3.5" />
                                    </span>
                                </ActionTooltip>
                            </div>
                            <p className="text-xs leading-relaxed text-muted-foreground">
                                {item.description}
                            </p>
                        </motion.button>
                    );
                })}
                </div>
            </motion.div>

            <Dialog open={activeModal !== null} onOpenChange={(open) => !open && setActiveModal(null)}>
                <DialogContent
                    title={
                        activeModal === 'import'
                                ? 'Import Wallet'
                                : 'Create New Key'
                    }
                    className="max-h-[90vh] max-w-[min(96vw,56rem)] gap-0 overflow-hidden border-[#272729] bg-[#171717] p-0"
                >
                    <div className="max-h-[90vh] overflow-y-auto p-5 sm:p-6">
                        {activeModal === 'import' ? <ImportWallet embedded /> : null}
                        {activeModal === 'create' ? <NewKey embedded /> : null}
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
};
