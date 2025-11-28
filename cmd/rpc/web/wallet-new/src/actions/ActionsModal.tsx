// ActionsModal.tsx
import React, {useEffect, useMemo, useState} from 'react'
import {motion, AnimatePresence} from 'framer-motion'
import {ModalTabs, Tab} from './ModalTabs'
import {Action as ManifestAction} from '@/manifest/types'
import ActionRunner from '@/actions/ActionRunner'
import {XIcon} from 'lucide-react'
import {cx} from '@/ui/cx'

interface ActionModalProps {
    actions?: (ManifestAction & { prefilledData?: Record<string, any> })[]
    isOpen: boolean
    onClose: () => void
}

export const ActionsModal: React.FC<ActionModalProps> = ({
                                                             actions,
                                                             isOpen,
                                                             onClose
                                                         }) => {
    const [selectedTab, setSelectedTab] = useState<Tab | undefined>(undefined)

    const modalSlot = useMemo(() => {
        return actions?.find(a => a.id === selectedTab?.value)?.ui?.slots?.modal
    }, [selectedTab, actions])

    const modalClassName = modalSlot?.className
    const modalStyle: React.CSSProperties | undefined = modalSlot?.style

    const availableTabs = useMemo(() => {
        return (
            actions?.map(a => ({
                value: a.id,
                label: a.title || a.id,
                icon: a.icon
            })) || []
        )
    }, [actions])

    useEffect(() => {
        if (availableTabs.length > 0) setSelectedTab(availableTabs[0])
    }, [availableTabs])

    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = 'hidden'
            return () => {
                document.body.style.overflow = 'auto'
            }
        }
    }, [isOpen])

    return (
        <AnimatePresence mode="wait">
            {isOpen && (
                <motion.div
                    key="actions-modal"
                    initial={{opacity: 0}}
                    animate={{opacity: 1}}
                    exit={{opacity: 0}}
                    className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-2 sm:p-4"
                    onClick={onClose}
                >
                    <motion.div
                        key="actions-modal-content"
                        exit={{scale: 0.9, opacity: 0}}
                        transition={{
                            duration: 0.3,
                            ease: 'easeInOut',
                            width: {duration: 0.3, ease: 'easeInOut'}
                        }}
                        // 🧩 base + clases opcionales + estilos inline del manifest
                        className={cx(
                            'relative bg-bg-secondary border border-bg-accent overflow-hidden flex flex-col',
                            // Mobile: casi pantalla completa
                            'w-full h-[96vh] max-h-[96vh] rounded-lg p-3',
                            // Small tablets: un poco más pequeño
                            'sm:h-auto sm:max-h-[92vh] sm:max-w-[90vw] sm:rounded-xl sm:p-5',
                            // Desktop: tamaño controlado
                            'md:w-auto md:max-w-[80vw] md:max-h-[90vh] md:p-6',
                            modalClassName
                        )}
                        style={modalStyle}
                        onClick={e => e.stopPropagation()}
                    >
                        <XIcon
                            onClick={onClose}
                            className="absolute top-3 right-3 sm:top-4 sm:right-4 w-5 h-5 sm:w-6 sm:h-6 text-text-muted cursor-pointer hover:text-white z-10"
                        />

                        <ModalTabs
                            activeTab={selectedTab}
                            onTabChange={setSelectedTab}
                            tabs={availableTabs}
                        />

                        {selectedTab && (
                            <motion.div
                                initial={{opacity: 0, y: 20}}
                                animate={{opacity: 1, y: 0}}
                                transition={{duration: 0.5, delay: 0.4}}
                                className="flex-1 overflow-y-auto scrollbar-hide hover:scrollbar-default min-h-0"
                            >
                                <ActionRunner
                                    actionId={selectedTab.value}
                                    onFinish={onClose}
                                    className="p-2 sm:p-3 md:p-4"
                                    prefilledData={actions?.find(a => a.id === selectedTab.value)?.prefilledData}
                                />
                            </motion.div>
                        )}
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    )
}
