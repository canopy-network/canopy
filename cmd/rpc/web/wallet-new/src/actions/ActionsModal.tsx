import React, {useEffect, useMemo, useState} from 'react';
import {motion, AnimatePresence} from 'framer-motion';
import {ModalTabs, Tab} from "./ModalTabs";
import {Action as ManifestAction} from "@/manifest/types";
import ActionRunner from "@/actions/ActionRunner";
import {XIcon} from "lucide-react";
import {cx} from "@/ui/cx";

interface ActionModalProps {
    actions?: ManifestAction[]
    isOpen: boolean;
    onClose: () => void;
}

export const ActionsModal: React.FC<ActionModalProps> = (
    {
        actions,
        isOpen,
        onClose
    }) => {

    const [selectedTab, setSelectedTab] = useState<Tab | undefined>(undefined);


    const modalClassName =  useMemo(() => {
        return actions?.find(a => a.id === selectedTab?.value)?.ui?.slots?.modal?.className;
    }, [selectedTab, actions])


    const availableTabs = useMemo(() => {

        let tabs: Tab[];
        tabs = actions?.map(a => ({
            value: a.id,
            label: a.title || a.id,
            icon: a.icon
        })) || [];

        return tabs;
    }, [actions])

    useEffect(() => {
        if (availableTabs.length > 0) {
            setSelectedTab(availableTabs[0]);
        }
    }, [availableTabs]);


    return (
        <AnimatePresence mode="wait">
            {isOpen && (
                <motion.div
                    key="actions-modal"
                    initial={{opacity: 0}}
                    animate={{opacity: 1}}
                    exit={{opacity: 0}}
                    className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
                    onClick={onClose}
                >
                    <motion.div
                        key="actions-modal-content"
                        exit={{scale: 0.9, opacity: 0}}
                        transition={{
                            duration: 0.3,
                            ease: "easeInOut",
                            width: {duration: 0.3, ease: "easeInOut"}
                        }}
                        className={cx(`relative bg-bg-secondary rounded-xl border border-bg-accent p-6 w-[26dvw] `, modalClassName)}
                        onClick={(e) => e.stopPropagation()}
                    >
                        <XIcon onClick={onClose} className="absolute top-4 right-4 text-text-muted cursor-pointer hover:text-white" />
                        {/* Header with Tabs */}
                        <ModalTabs
                            activeTab={selectedTab}
                            onTabChange={setSelectedTab}
                            tabs={availableTabs}
                        />

                        {selectedTab && (
                            <motion.div

                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ duration: 0.5, delay: 0.4 }}
                            >
                            <ActionRunner actionId={selectedTab.value} onFinish={onClose} />
                            </motion.div>
                        )}
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
}