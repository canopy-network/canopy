import React, { createContext, useContext, useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import ActionRunner from '@/actions/ActionRunner';

interface ActionModalContextType {
    openAction: (actionId: string, options?: ActionModalOptions) => void;
    closeAction: () => void;
    isOpen: boolean;
    currentActionId: string | null;
}

interface ActionModalOptions {
    onFinish?: () => void;
    onClose?: () => void;
}

const ActionModalContext = createContext<ActionModalContextType | undefined>(undefined);

export const useActionModal = () => {
    const context = useContext(ActionModalContext);
    if (!context) {
        throw new Error('useActionModal must be used within ActionModalProvider');
    }
    return context;
};

export const ActionModalProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [isOpen, setIsOpen] = useState(false);
    const [currentActionId, setCurrentActionId] = useState<string | null>(null);
    const [options, setOptions] = useState<ActionModalOptions>({});

    const openAction = useCallback((actionId: string, opts: ActionModalOptions = {}) => {
        setCurrentActionId(actionId);
        setOptions(opts);
        setIsOpen(true);
    }, []);

    const closeAction = useCallback(() => {
        setIsOpen(false);
        if (options.onClose) {
            options.onClose();
        }
        // Clear state after animation
        setTimeout(() => {
            setCurrentActionId(null);
            setOptions({});
        }, 300);
    }, [options]);

    const handleFinish = useCallback(() => {
        if (options.onFinish) {
            options.onFinish();
        }
        closeAction();
    }, [options, closeAction]);

    return (
        <ActionModalContext.Provider value={{ openAction, closeAction, isOpen, currentActionId }}>
            {children}

            {/* Modal Overlay */}
            <AnimatePresence>
                {isOpen && currentActionId && (
                    <>
                        {/* Backdrop */}
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            transition={{ duration: 0.2 }}
                            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
                            onClick={closeAction}
                        />

                        {/* Modal Content */}
                        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 pointer-events-none">
                            <motion.div
                                initial={{ opacity: 0, scale: 0.95, y: 20 }}
                                animate={{ opacity: 1, scale: 1, y: 0 }}
                                exit={{ opacity: 0, scale: 0.95, y: 20 }}
                                transition={{ duration: 0.2, ease: 'easeOut' }}
                                className="bg-bg-secondary rounded-2xl border border-bg-accent shadow-2xl max-w-2xl w-full overflow-hidden pointer-events-auto"
                                onClick={(e) => e.stopPropagation()}
                            >
                                {/* Close Button */}
                                <div className="absolute top-4 right-4 z-10">
                                    <button
                                        onClick={closeAction}
                                        className="p-2 rounded-lg bg-bg-tertiary/50 hover:bg-bg-tertiary border border-bg-accent transition-colors"
                                    >
                                        <i className="fa-solid fa-times text-text-muted w-5 h-5"></i>
                                    </button>
                                </div>

                                {/* Action Runner */}
                                <ActionRunner
                                    actionId={currentActionId}
                                    onFinish={handleFinish}
                                    className="p-6"
                                />
                            </motion.div>
                        </div>
                    </>
                )}
            </AnimatePresence>
        </ActionModalContext.Provider>
    );
};
