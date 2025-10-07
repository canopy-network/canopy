import React from 'react';
import { motion } from 'framer-motion';
import { Download } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useManifest } from '@/hooks/useManifest';
import { CurrentWallet } from './key-management/CurrentWallet';
import { ImportWallet } from './key-management/ImportWallet';
import { NewKey } from './key-management/NewKey';



export const KeyManagement = (): JSX.Element => {
    const { getText } = useManifest();
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

    return (
        <div className="bg-bg-primary">
            {/* Main Content */}
            <div className="px-6 py-8">
                <div className="flex justify-between items-center">
                    <motion.div
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ duration: 0.4 }}
                        className="mb-8"
                    >
                        <h1 className="text-3xl font-bold text-white mb-2">{getText('ui.keyManagement.title', 'Key Management')}</h1>
                        <p className="text-gray-400">{getText('ui.keyManagement.subtitle', 'Manage your wallet keys and security settings')}</p>
                    </motion.div>
                    <Button className="bg-primary text-primary-foreground hover:bg-primary/90 font-medium">
                        <Download className="w-4 h-4 mr-2" />
                        {getText('ui.keyManagement.downloadKeys', 'Download Keys')}
                    </Button>
                </div>

                {/* Three Panel Layout */}
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
