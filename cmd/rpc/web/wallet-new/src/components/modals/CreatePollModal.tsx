import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';

interface CreatePollModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: () => void;
}

// Campos del manifest.json para CreatePoll
const pollFields = [
    {
        name: "proposal",
        label: "Proposal",
        type: "textarea",
        required: true,
        placeholder: "Describe your proposal...",
        colSpan: 12
    },
    {
        name: "endBlock",
        label: "End Block",
        type: "number",
        required: true,
        placeholder: "e.g., 100",
        colSpan: 12
    },
    {
        name: "url",
        label: "Discussion URL",
        type: "url",
        required: true,
        placeholder: "https://discord.com/link-to-thread",
        colSpan: 12
    },
    {
        name: "password",
        label: "Password",
        type: "password",
        required: true,
        colSpan: 12
    }
];

export default function CreatePollModal({ isOpen, onClose, onSuccess }: CreatePollModalProps): JSX.Element {
    const { getText } = useManifest();
    const { createPoll } = useGovernanceActions('50003', '50002');
    const [formData, setFormData] = useState({
        proposal: '',
        endBlock: '100',
        url: 'https://discord.com/link-to-thread',
        password: ''
    });

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        try {
            const result = await createPoll.mutateAsync({
                proposal: formData.proposal,
                endBlock: parseInt(formData.endBlock),
                url: formData.url,
                password: formData.password
            });

            // Show success toast
            toast.success(`✅ Poll creado exitosamente!\nHash: ${result}`, {
                duration: 5000,
                style: {
                    fontSize: '14px',
                    whiteSpace: 'pre-line'
                }
            });

            onSuccess();
            onClose();
            setFormData({
                proposal: '',
                endBlock: '100',
                url: 'https://discord.com/link-to-thread',
                password: ''
            });
        } catch (error: any) {
            console.error('Error creating poll:', error);
            
            // Show error toast
            toast.error(`❌ Error al crear poll:\n${error.message || 'Error desconocido'}`, {
                duration: 6000,
                style: {
                    fontSize: '14px',
                    whiteSpace: 'pre-line'
                }
            });
        }
    };

    const renderField = (field: any) => {
        const commonProps = {
            value: formData[field.name as keyof typeof formData] || '',
            onChange: (e: any) => setFormData({ ...formData, [field.name]: e.target.value }),
            className: "w-full bg-bg-secondary border border-gray-600 rounded-md px-3 py-2 text-white",
            required: field.required,
            placeholder: field.placeholder
        };

        switch (field.type) {
            case 'textarea':
                return (
                    <textarea
                        {...commonProps}
                        className="w-full bg-bg-secondary border border-gray-600 rounded-md px-3 py-2 text-white h-24 resize-none"
                    />
                );
            default:
                return <input type={field.type} {...commonProps} />;
        }
    };

    return (
        <AnimatePresence>
            {isOpen && (
                <motion.div
                    className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.2 }}
                >
                    <motion.div
                        className="bg-bg-secondary rounded-xl border border-bg-accent p-6 w-full max-w-md"
                        initial={{ scale: 0.9, opacity: 0, y: 20 }}
                        animate={{ scale: 1, opacity: 1, y: 0 }}
                        exit={{ scale: 0.9, opacity: 0, y: 20 }}
                        transition={{ duration: 0.3, ease: "easeOut" }}
                    >
                        <motion.div
                            className="flex items-center justify-between mb-4"
                            initial={{ opacity: 0, y: -10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.1, duration: 0.3 }}
                        >
                            <h2 className="text-white text-lg font-bold">
                                {getText('ui.governance.createPoll', 'Create Poll')}
                            </h2>
                            <motion.button
                                onClick={onClose}
                                className="text-text-muted hover:text-white"
                                whileHover={{ scale: 1.1 }}
                                whileTap={{ scale: 0.9 }}
                            >
                                <i className="fa-solid fa-times"></i>
                            </motion.button>
                        </motion.div>

                        <motion.form
                            onSubmit={handleSubmit}
                            className="space-y-4"
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            transition={{ delay: 0.2, duration: 0.3 }}
                        >
                            {pollFields.map((field, index) => (
                                <motion.div
                                    key={field.name}
                                    initial={{ opacity: 0, x: -20 }}
                                    animate={{ opacity: 1, x: 0 }}
                                    transition={{ delay: 0.3 + (index * 0.1), duration: 0.3 }}
                                >
                                    <label className="block text-text-muted text-sm mb-2">
                                        {getText(`ui.governance.${field.name}`, field.label)}
                                    </label>
                                    <motion.div
                                        whileFocus={{ scale: 1.02 }}
                                        transition={{ duration: 0.2 }}
                                    >
                                        {renderField(field)}
                                    </motion.div>
                                </motion.div>
                            ))}

                            <motion.div
                                className="flex gap-3 pt-4"
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.7, duration: 0.3 }}
                            >
                                <motion.button
                                    type="button"
                                    onClick={onClose}
                                    className="flex-1 bg-gray-500/20 text-gray-300 px-4 py-2 rounded-md hover:bg-gray-500/30"
                                    whileHover={{ scale: 1.02 }}
                                    whileTap={{ scale: 0.98 }}
                                >
                                    {getText('ui.common.cancel', 'Cancel')}
                                </motion.button>
                                <motion.button
                                    type="submit"
                                    disabled={createPoll.isPending}
                                    className="flex-1 bg-primary text-muted px-4 py-2 rounded-md hover:bg-primary/80 disabled:opacity-50"
                                    whileHover={{ scale: 1.02 }}
                                    whileTap={{ scale: 0.98 }}
                                >
                                    <i className="fa-solid fa-download"></i>
                                    {createPoll.isPending ? getText('ui.common.creating', 'Creating...') : getText('ui.governance.create', 'Create')}
                                </motion.button>
                            </motion.div>
                        </motion.form>
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
}
