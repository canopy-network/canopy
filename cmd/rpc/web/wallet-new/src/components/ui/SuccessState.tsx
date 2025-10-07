import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

export const SuccessState: React.FC = () => {
    const { getText } = useManifest();
    
    return (
        <motion.div
            initial={{ scale: 0.9, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            className="text-center py-8"
        >
            <div className="w-16 h-16 bg-green-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
                <i className="fa-solid fa-check text-green-400 text-2xl"></i>
            </div>
                    <h3 className="text-lg font-semibold text-text-primary mb-2">
                        {getText('ui.modals.success.title', 'Transaction Successful!')}
                    </h3>
                    <p className="text-text-muted">
                        {getText('ui.modals.success.message', 'Your transaction has been sent successfully')}
                    </p>
        </motion.div>
    );
};
