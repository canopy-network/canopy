import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, Info, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/Button';

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
          icon: XCircle,
          iconColor: 'text-[#ff1845]',
          iconBg: 'bg-[#ff1845]/12',
          buttonColor: 'bg-[#ff1845] hover:bg-[#ff1845]/90 text-white',
          borderColor: 'border-[#ff1845]/35'
        };
      case 'warning':
        return {
          icon: AlertTriangle,
          iconColor: 'text-[#ddb228]',
          iconBg: 'bg-[#ddb228]/12',
          buttonColor: 'bg-[#ddb228] hover:bg-[#ddb228]/90 text-[#0f0f0f]',
          borderColor: 'border-[#ddb228]/35'
        };
      case 'info':
        return {
          icon: Info,
          iconColor: 'text-[#216cd0]',
          iconBg: 'bg-[#216cd0]/12',
          buttonColor: 'bg-[#216cd0] hover:bg-[#216cd0]/90 text-white',
          borderColor: 'border-[#216cd0]/35'
        };
      default:
        return {
          icon: AlertTriangle,
          iconColor: 'text-[#ddb228]',
          iconBg: 'bg-[#ddb228]/12',
          buttonColor: 'bg-[#ddb228] hover:bg-[#ddb228]/90 text-[#0f0f0f]',
          borderColor: 'border-[#ddb228]/35'
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
        className="fixed inset-0 bg-[#0f0f0f]/80 backdrop-blur-md flex items-center justify-center z-50 p-4"
        onClick={onClose}
      >
        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.9, opacity: 0 }}
          className={`bg-[#171717] rounded-2xl border ${styles.borderColor} p-6 w-full max-w-md shadow-[0_24px_72px_rgba(0,0,0,0.55)]`}
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center gap-4 mb-4">
            <div className={`w-12 h-12 ${styles.iconBg} rounded-full flex items-center justify-center`}>
              <styles.icon className={`${styles.iconColor} w-5 h-5`} />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-foreground">{title}</h3>
            </div>
          </div>

          <div className="mb-6">
            <p className="text-muted-foreground leading-relaxed whitespace-pre-line">{message}</p>
          </div>

          <div className="flex gap-3 justify-end">
            <Button
              onClick={onClose}
              variant="secondary"
              className="px-4 py-2 border-[#272729] bg-[#0f0f0f] text-white hover:bg-[#272729]"
            >
              {cancelText}
            </Button>
            <Button
              onClick={handleConfirm}
              className={`px-4 py-2 ${styles.buttonColor}`}
              variant={type === 'danger' ? 'destructive' : 'default'}
            >
              {confirmText}
            </Button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
};
