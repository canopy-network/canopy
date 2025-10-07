import React from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

interface PasswordModalProps {
    isOpen: boolean;
    password: string;
    isLoading: boolean;
    onPasswordChange: (password: string) => void;
    onSubmit: () => void;
    onClose: () => void;
}

export const PasswordModal: React.FC<PasswordModalProps> = ({
    isOpen,
    password,
    isLoading,
    onPasswordChange,
    onSubmit,
    onClose
}) => {
    const { getText } = useManifest();
    if (!isOpen) return null;

    return (
        <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 flex items-center justify-center z-60 p-4"
            onClick={onClose}
        >
            <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                exit={{ scale: 0.9, opacity: 0 }}
                className="bg-bg-secondary rounded-xl border border-bg-accent p-6 w-full max-w-md"
                onClick={(e) => e.stopPropagation()}
            >
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-lg font-semibold text-text-primary">
                                {getText('ui.modals.password.title', 'Enter Password')}
                            </h3>
                    <button
                        onClick={onClose}
                        className="text-text-muted hover:text-text-primary transition-colors"
                    >
                        <i className="fa-solid fa-times text-lg"></i>
                    </button>
                </div>

                <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-text-primary mb-2">
                                    {getText('ui.modals.password.passwordLabel', 'Password')}
                                </label>
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => onPasswordChange(e.target.value)}
                                    placeholder={getText('ui.modals.password.passwordPlaceholder', 'Enter your password')}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                            autoFocus
                            onKeyPress={(e) => {
                                if (e.key === 'Enter') {
                                    onSubmit();
                                }
                            }}
                        />
                    </div>

                            <div className="flex gap-3">
                                <button
                                    onClick={onClose}
                                    className="flex-1 px-4 py-3 bg-bg-accent hover:bg-bg-accent/80 text-text-primary font-medium rounded-lg transition-colors"
                                >
                                    {getText('ui.modals.password.cancel', 'Cancel')}
                                </button>
                                <button
                                    onClick={onSubmit}
                                    disabled={isLoading}
                                    className="flex-1 bg-primary hover:bg-primary/90 disabled:bg-primary/50 text-muted font-medium py-3 px-4 rounded-lg transition-colors flex items-center justify-center gap-2"
                                >
                                    {isLoading ? (
                                        <>
                                            <i className="fa-solid fa-spinner fa-spin"></i>
                                            {getText('ui.modals.password.processing', 'Processing...')}
                                        </>
                                    ) : (
                                        getText('ui.modals.password.confirm', 'Confirm')
                                    )}
                                </button>
                            </div>
                </div>
            </motion.div>
        </motion.div>
    );
};
