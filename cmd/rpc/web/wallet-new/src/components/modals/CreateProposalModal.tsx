import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';
import { useAccounts } from '@/hooks/useAccounts';

interface CreateProposalModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSuccess: () => void;
}


export default function CreateProposalModal({ isOpen, onClose, onSuccess }: CreateProposalModalProps): JSX.Element {
    const { getText, manifest } = useManifest();
    const { activeAccount } = useAccounts();

    // Obtener campos del manifest
    const getFieldsFromManifest = () => {
        const governanceAction = manifest?.actions?.find((action: any) => action.id === 'Governance');
        const createProposalAction = governanceAction?.actions?.find((action: any) => action.id === 'CreateProposal');
        return (createProposalAction as any)?.form?.fields || [];
    };

    const fields = getFieldsFromManifest();

    // Función para obtener la altura actual del bloque
    const getCurrentHeight = async () => {
        try {
            const response = await fetch(`http://localhost:${queryPort}/v1/query/height`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: '{}'
            });
            const height = await response.text();
            return parseInt(height.trim()) || 0;
        } catch (error) {
            console.error('Error getting current height:', error);
            return 0;
        }
    };

    // Estado inicial basado en el manifest
    const getInitialFormData = async () => {
        const initialData: any = {};
        const currentHeight = await getCurrentHeight();

        fields.forEach((field: any) => {
            if (field.name === 'startHeight') {
                // Usar altura actual + 100 como startHeight
                initialData[field.name] = (currentHeight + 100).toString();
            } else if (field.name === 'endHeight') {
                // Usar altura actual + 200 como endHeight
                initialData[field.name] = (currentHeight + 200).toString();
            } else {
                // Usar defaultValue del manifest o valor por defecto
                initialData[field.name] = field.defaultValue || '';
            }
        });
        return initialData;
    };

    const [formData, setFormData] = useState({
        paramSpace: 'fee',
        paramKey: 'sendFee',
        paramValue: '1000',
        startHeight: '2200',
        endHeight: '2300',
        memo: '',
        password: ''
    });

    // Función para obtener opciones por paramSpace desde el manifest
    const getFilteredParamKeyOptions = (paramSpace: string) => {
        const paramKeyField = fields.find((field: any) => field.name === 'paramKey');
        const optionsBySpace = paramKeyField?.optionsBySpace || {};
        return optionsBySpace[paramSpace] || [];
    };

    // Inicializar con opciones filtradas
    const getInitialParamKeyOptions = () => {
        return getFilteredParamKeyOptions('fee'); // Usar 'fee' como default
    };

    const [paramKeyOptions, setParamKeyOptions] = useState(getInitialParamKeyOptions());

    // Usar puertos por defecto (Node 1)
    const adminPort = '50003';
    const queryPort = '50002';

    const { createProposal } = useGovernanceActions(adminPort, queryPort);

    // Actualizar formData cuando el manifest se cargue
    useEffect(() => {
        if (manifest && fields.length > 0) {
            const loadInitialData = async () => {
                const newFormData = await getInitialFormData();
                setFormData(newFormData);
            };
            loadInitialData();
        }
    }, [manifest]);

    // Actualizar opciones de paramKey cuando cambie paramSpace
    useEffect(() => {
        if (formData.paramSpace) {
            const filteredOptions = getFilteredParamKeyOptions(formData.paramSpace);
            setParamKeyOptions(filteredOptions);

            // Reset paramKey si no está disponible en el nuevo espacio
            if (filteredOptions.length > 0 && !filteredOptions.find((opt: any) => opt.value === formData.paramKey)) {
                setFormData((prev: any) => ({ ...prev, paramKey: filteredOptions[0].value }));
            }
        }
    }, [formData.paramSpace, manifest]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (!activeAccount) {
            toast.error('No active account selected');
            return;
        }

        try {
            const result = await createProposal.mutateAsync({
                paramSpace: formData.paramSpace,
                paramKey: formData.paramKey,
                paramValue: formData.paramValue,
                startHeight: parseInt(formData.startHeight),
                endHeight: parseInt(formData.endHeight),
                memo: formData.memo,
                password: formData.password
            });

            // Crear el objeto completo como el viejo explorador
            const proposalObject = {
                type: "changeParameter",
                msg: {
                    parameterSpace: formData.paramSpace,
                    parameterKey: formData.paramKey,
                    parameterValue: parseInt(formData.paramValue),
                    startHeight: parseInt(formData.startHeight),
                    endHeight: parseInt(formData.endHeight),
                    signer: activeAccount?.address || ""
                },
                signature: {
                    publicKey: activeAccount?.publicKey || "",
                    signature: result || ""
                },
                time: Date.now() * 1000, // Microsegundos
                createdHeight: 0, // Se llenará cuando se confirme
                fee: 10000,
                memo: formData.memo,
                networkID: 1,
                chainID: 1
            };

            // Mostrar en consola como el viejo explorador
            console.log("Proposal Created:", proposalObject);

            // Show success toast
            toast.success(`✅ Propuesta creada exitosamente!\nHash: ${result}`, {
                duration: 5000,
                style: {
                    fontSize: '14px',
                    whiteSpace: 'pre-line'
                }
            });

            onSuccess();
            onClose();
            // Reset form data
            const resetData = async () => {
                const newFormData = await getInitialFormData();
                setFormData(newFormData);
            };
            resetData();
        } catch (error: any) {
            console.error('Error creating proposal:', error);

            // Show error toast
            toast.error(`❌ Error al crear propuesta:\n${error.message || 'Error desconocido'}`, {
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
            case 'select':
                // Usar opciones dinámicas para paramKey
                const options = field.name === 'paramKey' ? paramKeyOptions : field.options;
                return (
                    <select {...commonProps}>
                        {options?.map((option: any) => (
                            <option key={option.value} value={option.value}>
                                {option.label}
                            </option>
                        ))}
                    </select>
                );
            case 'textarea':
                return (
                    <textarea
                        {...commonProps}
                        className="w-full bg-bg-secondary border border-gray-600 rounded-md px-3 py-2 text-white h-20 resize-none"
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
                    className="fixed -inset-10 bg-black/50 flex items-center justify-center z-50"
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
                                {getText('ui.governance.createProposal', 'Create Proposal')}
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
                            transition={{ delay: 0.3, duration: 0.3 }}
                        >
                            {fields.map((field: any, index: number) => (
                                <motion.div
                                    key={field.name}
                                    className={field.colSpan === 6 ? "grid grid-cols-2 gap-4" : ""}
                                    initial={{ opacity: 0, x: -20 }}
                                    animate={{ opacity: 1, x: 0 }}
                                    transition={{ delay: 0.3 + (index * 0.1), duration: 0.3 }}
                                >
                                    <div className={field.colSpan === 6 ? "" : ""}>
                                        <label className="block text-text-muted text-sm mb-2">
                                            {getText(`ui.governance.${field.name}`, field.label)}
                                        </label>
                                        <motion.div
                                            whileFocus={{ scale: 1.02 }}
                                            transition={{ duration: 0.2 }}
                                        >
                                            {renderField(field)}
                                        </motion.div>
                                    </div>
                                </motion.div>
                            ))}

                            <motion.div
                                className="flex gap-3 pt-4"
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ delay: 0.8, duration: 0.3 }}
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
                                    disabled={createProposal.isPending}
                                    className="flex-1 bg-primary text-muted px-4 py-2 rounded-md hover:bg-primary/80 disabled:opacity-50"
                                    whileHover={{ scale: 1.02 }}
                                    whileTap={{ scale: 0.98 }}
                                >
                                    <i className="fa-solid fa-download"></i>
                                    {createProposal.isPending ? getText('ui.common.creating', 'Creating...') : getText('ui.governance.create', 'Create')}
                                </motion.button>
                            </motion.div>
                        </motion.form>
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
}
