import React, { createContext, useContext, useState, useCallback, useMemo, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import ActionRunner from '@/actions/ActionRunner';
import { useManifest } from '@/hooks/useManifest';
import { XIcon } from 'lucide-react';
import { cx } from '@/ui/cx';
import { ModalTabs, Tab } from '@/actions/ModalTabs';
import {LucideIcon} from "@/components/ui/LucideIcon";

interface ActionModalContextType {
    openAction: (actionId: string, options?: ActionModalOptions) => void;
    closeAction: () => void;
    isOpen: boolean;
    currentActionId: string | null;
}

interface ActionModalOptions {
    onFinish?: () => void;
    onClose?: () => void;
    prefilledData?: Record<string, any>;
    relatedActions?: string[]; // IDs of related actions to show as tabs
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
    const [selectedTab, setSelectedTab] = useState<Tab | undefined>(undefined);
    const [options, setOptions] = useState<ActionModalOptions>({});
    const { manifest } = useManifest();

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
            setSelectedTab(undefined);
            setOptions({});
        }, 300);
    }, [options]);

    const handleFinish = useCallback(() => {
        if (options.onFinish) {
            options.onFinish();
        }
        closeAction();
    }, [options, closeAction]);

    // Build tabs from current action and related actions
    const availableTabs = useMemo(() => {
        if (!currentActionId || !manifest) return [];

        const currentAction = manifest.actions.find(a => a.id === currentActionId);
        if (!currentAction) return [];

        const tabs: Tab[] = [{
            value: currentAction.id,
            label: currentAction.title || currentAction.id,
            icon: currentAction.icon
        }];

        // Add related actions from options or manifest
        const relatedActionIds = options.relatedActions || currentAction.relatedActions || [];
        relatedActionIds.forEach(relatedId => {
            const relatedAction = manifest.actions.find(a => a.id === relatedId);
            if (relatedAction) {
                tabs.push({
                    value: relatedAction.id,
                    label: relatedAction.title || relatedAction.id,
                    icon: relatedAction.icon
                });
            }
        });

        return tabs;
    }, [currentActionId, manifest, options.relatedActions]);

    // Set initial selected tab when tabs change
    useEffect(() => {
        if (availableTabs.length > 0 && !selectedTab) {
            setSelectedTab(availableTabs[0]);
        }
    }, [availableTabs, selectedTab]);

    // Get active action ID from selected tab or current action
    const activeActionId = selectedTab?.value || currentActionId;

    // Get modal slot configuration from manifest for active action
    const modalSlot = useMemo(() => {
        return manifest?.actions?.find(a => a.id === activeActionId)?.ui?.slots?.modal;
    }, [activeActionId, manifest]);

    const modalClassName = modalSlot?.className;
    const modalStyle: React.CSSProperties | undefined = modalSlot?.style;

    // Prevent body scroll when modal is open
    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = 'hidden';
            return () => {
                document.body.style.overflow = 'auto';
            };
        }
    }, [isOpen]);

    return (
        <ActionModalContext.Provider value={{ openAction, closeAction, isOpen, currentActionId }}>
            {children}

            {/* Modal Overlay */}
            <AnimatePresence mode="wait">
                {isOpen && currentActionId && (
                    <motion.div
                        key="action-modal"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
                        onClick={closeAction}
                    >
                        <motion.div
                            key="action-modal-content"
                            exit={{ scale: 0.9, opacity: 0 }}
                            transition={{
                                duration: 0.3,
                                ease: 'easeInOut',
                                width: { duration: 0.3, ease: 'easeInOut' }
                            }}
                            className={cx(
                                'relative bg-bg-secondary rounded-xl border border-bg-accent p-6 max-h-[95vh] max-w-[40vw]',
                                modalClassName
                            )}
                            style={modalStyle}
                            onClick={(e) => e.stopPropagation()}
                        >
                            {/* Close Button */}
                            <XIcon
                                onClick={closeAction}
                                className="absolute top-4 right-4 text-text-muted cursor-pointer hover:text-white z-10"
                            />

                            {/* Tabs - only show if there are multiple actions */}
                            {availableTabs.length > 1 ? (
                                <ModalTabs
                                    activeTab={selectedTab}
                                    onTabChange={setSelectedTab}
                                    tabs={availableTabs}
                                />
                            ) : (
                                /* Single action title */
                                availableTabs.length === 1 && (
                                    <div className="mb-6 flex items-center gap-3">
                                        {availableTabs[0].icon && (
                                            <div className="flex items-center justify-center w-10 h-10 rounded-lg ">
                                                <LucideIcon name={availableTabs[0].icon} className="w-6 h-6 text-primary" />
                                            </div>
                                        )}
                                        <h2 className="text-2xl font-semibold text-white">
                                            {availableTabs[0].label}
                                        </h2>
                                    </div>
                                )
                            )}

                            {/* Action Runner with scroll */}
                            {selectedTab && (
                                <motion.div
                                    key={selectedTab.value}
                                    initial={{ opacity: 0, y: 20 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    transition={{ duration: 0.5, delay: 0.4 }}
                                    className="max-h-[80vh] overflow-y-auto scrollbar-hide hover:scrollbar-default"
                                >
                                    <ActionRunner
                                        actionId={selectedTab.value}
                                        onFinish={handleFinish}
                                        className="p-4"
                                        prefilledData={options.prefilledData}
                                    />
                                </motion.div>
                            )}
                        </motion.div>
                    </motion.div>
                )}
            </AnimatePresence>
        </ActionModalContext.Provider>
    );
};
