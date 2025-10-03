import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';

interface ConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  type?: 'warning' | 'danger' | 'info';
}

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  type = 'warning'
}) => {
  const getTypeStyles = () => {
    switch (type) {
      case 'danger':
        return {
          icon: 'fa-solid fa-exclamation-triangle',
          iconColor: 'text-red-400',
          iconBg: 'bg-red-500/20',
          buttonColor: 'bg-red-500 hover:bg-red-600',
          borderColor: 'border-red-500/30'
        };
      case 'warning':
        return {
          icon: 'fa-solid fa-exclamation-triangle',
          iconColor: 'text-yellow-400',
          iconBg: 'bg-yellow-500/20',
          buttonColor: 'bg-yellow-500 hover:bg-yellow-600',
          borderColor: 'border-yellow-500/30'
        };
      case 'info':
        return {
          icon: 'fa-solid fa-info-circle',
          iconColor: 'text-blue-400',
          iconBg: 'bg-blue-500/20',
          buttonColor: 'bg-blue-500 hover:bg-blue-600',
          borderColor: 'border-blue-500/30'
        };
      default:
        return {
          icon: 'fa-solid fa-exclamation-triangle',
          iconColor: 'text-yellow-400',
          iconBg: 'bg-yellow-500/20',
          buttonColor: 'bg-yellow-500 hover:bg-yellow-600',
          borderColor: 'border-yellow-500/30'
        };
    }
  };

  const styles = getTypeStyles();

  const handleConfirm = () => {
    onConfirm();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
        onClick={onClose}
      >
        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.9, opacity: 0 }}
          className={`bg-bg-secondary rounded-xl border ${styles.borderColor} p-6 w-full max-w-md`}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center gap-4 mb-4">
            <div className={`w-12 h-12 ${styles.iconBg} rounded-full flex items-center justify-center`}>
              <i className={`${styles.icon} ${styles.iconColor} text-xl`}></i>
            </div>
            <div>
              <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
            </div>
          </div>

          <div className="mb-6">
            <p className="text-text-muted leading-relaxed whitespace-pre-line">{message}</p>
          </div>

          <div className="flex gap-3 justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-bg-tertiary hover:bg-bg-accent text-text-primary font-medium rounded-lg transition-colors"
            >
              {cancelText}
            </button>
            <button
              onClick={handleConfirm}
              className={`px-4 py-2 ${styles.buttonColor} text-white font-medium rounded-lg transition-colors`}
            >
              {confirmText}
            </button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
};
