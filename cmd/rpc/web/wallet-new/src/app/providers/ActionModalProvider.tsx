import React, { createContext, useContext, useState, useCallback, useMemo, useEffect, Suspense } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { XIcon, Loader2 } from 'lucide-react';
import { cx } from '@/ui/cx';
import { ModalTabs, Tab } from '@/actions/ModalTabs';
import { LucideIcon } from '@/components/ui/LucideIcon';

const ActionRunner = React.lazy(() => import('@/actions/ActionRunner'));

const ActionRunnerFallback = () => (
  <div className="flex flex-col items-center justify-center py-12 gap-3">
    <Loader2 className="w-8 h-8 text-primary animate-spin" />
    <span className="text-muted-foreground text-sm">Loading action...</span>
  </div>
);

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
  relatedActions?: string[];
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

  const availableTabs = useMemo(() => {
    if (!currentActionId || !manifest) return [];

    const currentAction = manifest.actions.find((a) => a.id === currentActionId);
    if (!currentAction) return [];

    const tabs: Tab[] = [
      {
        value: currentAction.id,
        label: currentAction.title || currentAction.id,
        icon: currentAction.icon,
      },
    ];

    const relatedActionIds = options.relatedActions || currentAction.relatedActions || [];
    relatedActionIds.forEach((relatedId) => {
      const relatedAction = manifest.actions.find((a) => a.id === relatedId);
      if (relatedAction) {
        tabs.push({
          value: relatedAction.id,
          label: relatedAction.title || relatedAction.id,
          icon: relatedAction.icon,
        });
      }
    });

    return tabs;
  }, [currentActionId, manifest, options.relatedActions]);

  useEffect(() => {
    if (availableTabs.length > 0 && !selectedTab) {
      setSelectedTab(availableTabs[0]);
    }
  }, [availableTabs, selectedTab]);

  const activeActionId = selectedTab?.value || currentActionId;

  const modalSlot = useMemo(() => {
    return manifest?.actions?.find((a) => a.id === activeActionId)?.ui?.slots?.modal;
  }, [activeActionId, manifest]);

  const modalClassName = modalSlot?.className;
  const modalStyle: React.CSSProperties | undefined = modalSlot?.style;

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

      <AnimatePresence mode="wait">
        {isOpen && currentActionId && (
          <motion.div
            key="action-modal"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-2 sm:p-4"
            onClick={closeAction}
          >
            <motion.div
              key="action-modal-content"
              exit={{ scale: 0.9, opacity: 0 }}
              transition={{
                duration: 0.3,
                ease: 'easeInOut',
                width: { duration: 0.3, ease: 'easeInOut' },
              }}
              className={cx(
                'relative bg-card border border-border overflow-hidden flex flex-col min-h-0',
                'w-full max-w-[min(100vw-1rem,72rem)] rounded-lg sm:rounded-xl',
                'h-[calc(100dvh-1rem)] max-h-[calc(100dvh-1rem)]',
                'sm:h-auto sm:max-h-[calc(100dvh-2rem)]',
                modalClassName,
              )}
              style={modalStyle}
              onClick={(e) => e.stopPropagation()}
            >
              <XIcon
                onClick={closeAction}
                className="absolute top-3 right-3 sm:top-4 sm:right-4 w-5 h-5 sm:w-6 sm:h-6 text-muted-foreground cursor-pointer hover:text-foreground z-10"
              />

              {availableTabs.length > 1 ? (
                <div className="shrink-0 px-3 pt-3 pb-2 sm:px-5 sm:pt-5 sm:pb-3 md:px-6 md:pt-6">
                  <ModalTabs
                    activeTab={selectedTab}
                    onTabChange={setSelectedTab}
                    tabs={availableTabs}
                  />
                </div>
              ) : (
                availableTabs.length === 1 && (
                  <div className="shrink-0 px-3 pt-3 pb-2 sm:px-5 sm:pt-5 sm:pb-3 md:px-6 md:pt-6">
                    <div className="mb-2 pr-8 sm:mb-3 flex items-start gap-3">
                      {availableTabs[0].icon && (
                        <div className="flex items-center justify-center w-9 h-9 sm:w-10 sm:h-10 rounded-lg">
                          <LucideIcon name={availableTabs[0].icon} className="w-5 h-5 sm:w-6 sm:h-6 text-primary" />
                        </div>
                      )}
                      <h2 className="text-xl sm:text-2xl font-semibold text-foreground break-words">
                        {availableTabs[0].label}
                      </h2>
                    </div>
                  </div>
                )
              )}

              {selectedTab && (
                <motion.div
                  key={selectedTab.value}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.5, delay: 0.4 }}
                  className="flex-1 overflow-y-auto overflow-x-hidden scrollbar-hide hover:scrollbar-default min-h-0 px-3 pb-3 sm:px-5 sm:pb-5 md:px-6 md:pb-6"
                >
                  <Suspense fallback={<ActionRunnerFallback />}>
                    <ActionRunner
                      actionId={selectedTab.value}
                      onFinish={handleFinish}
                      className="p-0"
                      prefilledData={options.prefilledData}
                    />
                  </Suspense>
                </motion.div>
              )}
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </ActionModalContext.Provider>
  );
};
