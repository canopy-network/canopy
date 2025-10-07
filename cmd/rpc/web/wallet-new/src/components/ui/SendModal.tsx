import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useAccounts } from '@/hooks/useAccounts';
import { useAccountData } from '@/hooks/useAccountData';
import { TxSend, TxStake, TxCreateOrder } from '@/core/api';
import { AlertModal } from './AlertModal';
import { PasswordModal } from './PasswordModal';
import { ModalTabs } from './ModalTabs';
import { SuccessState } from './SuccessState';
import { SendForm } from './forms/SendForm';
import { ReceiveForm } from './forms/ReceiveForm';
import { StakeForm } from './forms/StakeForm';
import { SwapForm } from './forms/SwapForm';

interface SendModalProps {
    isOpen: boolean;
    onClose: () => void;
    defaultTab?: 'send' | 'receive' | 'stake' | 'swap';
}

export const SendModal: React.FC<SendModalProps> = ({
    isOpen,
    onClose,
    defaultTab = 'send'
}) => {
    const { accounts, loading: accountsLoading } = useAccounts();
    const { balances, loading: dataLoading } = useAccountData();

    const [activeTab, setActiveTab] = useState<'send' | 'receive' | 'stake' | 'swap'>(defaultTab);

    const [formData, setFormData] = useState({
        account: accounts[0]?.nickname || '',
        recipient: '',
        amount: '',
        memo: '',
        fee: 0.01,
        // Stake fields
        delegate: true,
        committees: '1',
        netAddress: '',
        withdrawal: true,
        output: '',
        signer: accounts[0]?.nickname || '',
        // Swap fields
        chainId: '',
        data: '',
        receiveAmount: '',
        receiveAddress: ''
    });

    const [passwordModal, setPasswordModal] = useState({
        isOpen: false,
        password: ''
    });

    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const [alertModal, setAlertModal] = useState<{
        isOpen: boolean;
        title: string;
        message: string;
        type: 'success' | 'error' | 'warning' | 'info';
    }>({
        isOpen: false,
        title: '',
        message: '',
        type: 'info'
    });

    // Update form data when accounts change
    useEffect(() => {
        if (accounts.length > 0 && !formData.account) {
            setFormData(prev => ({
                ...prev,
                account: accounts[0].nickname
            }));
        }
    }, [accounts, formData.account]);

    // Update active tab when defaultTab changes
    useEffect(() => {
        setActiveTab(defaultTab);
    }, [defaultTab]);

    const handleInputChange = (field: string, value: string | number | boolean) => {
        setFormData(prev => ({
            ...prev,
            [field]: value
        }));
    };

    const getAvailableBalance = () => {
        const selectedAccount = accounts.find(acc => acc.nickname === formData.account);
        if (!selectedAccount) return 0;

        const balanceInfo = balances.find(b => b.address === selectedAccount.address);
        return balanceInfo?.amount || 0;
    };

    const getSelectedAccountAddress = () => {
        const selectedAccount = accounts.find(acc => acc.nickname === formData.account);
        return selectedAccount?.address || '';
    };

    const formatBalance = (amount: number) => {
        return (amount / 1000000).toFixed(2);
    };

    const handleMaxAmount = () => {
        const available = getAvailableBalance();
        const feeInMicroUnits = formData.fee * 1000000;
        const maxAmount = Math.max(0, available - feeInMicroUnits);
        setFormData(prev => ({
            ...prev,
            amount: formatBalance(maxAmount)
        }));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (activeTab === 'send') {
            // Validate send form
            if (!formData.recipient || !formData.amount) {
                setAlertModal({
                    isOpen: true,
                    title: 'Missing Information',
                    message: 'Please fill in all required fields.',
                    type: 'error'
                });
                return;
            }

            // Validate recipient address
            if (formData.recipient.length !== 40) {
                setAlertModal({
                    isOpen: true,
                    title: 'Invalid Recipient',
                    message: 'Recipient address must be 40 characters long.',
                    type: 'error'
                });
                return;
            }

            // Validate amount
            const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
            const feeInMicroUnits = 0.01 * 1000000; // Fixed fee
            const availableBalance = getAvailableBalance();

            if (amountInMicroUnits + feeInMicroUnits > availableBalance) {
                setAlertModal({
                    isOpen: true,
                    title: 'Insufficient Balance',
                    message: 'The amount plus fee exceeds your available balance.',
                    type: 'error'
                });
                return;
            }
                } else if (activeTab === 'stake') {
                    // Validate stake form
                    if (!formData.amount) {
                        setAlertModal({
                            isOpen: true,
                            title: 'Missing Information',
                            message: 'Please fill in the amount field.',
                            type: 'error'
                        });
                        return;
                    }

                    // Validate amount
                    const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
                    const availableBalance = getAvailableBalance();

                    if (amountInMicroUnits > availableBalance) {
                        setAlertModal({
                            isOpen: true,
                            title: 'Insufficient Balance',
                            message: 'The amount exceeds your available balance.',
                            type: 'error'
                        });
                        return;
                    }
                } else if (activeTab === 'swap') {
                    // Validate swap form
                    if (!formData.chainId || !formData.amount || !formData.receiveAmount || !formData.receiveAddress) {
                        setAlertModal({
                            isOpen: true,
                            title: 'Missing Information',
                            message: 'Please fill in all required fields.',
                            type: 'error'
                        });
                        return;
                    }

                    // Validate receive address
                    if (formData.receiveAddress.length !== 40) {
                        setAlertModal({
                            isOpen: true,
                            title: 'Invalid Receive Address',
                            message: 'Receive address must be 40 characters long.',
                            type: 'error'
                        });
                        return;
                    }

                    // Validate amount
                    const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
                    const feeInMicroUnits = 0.01 * 1000000; // Fixed fee
                    const availableBalance = getAvailableBalance();

                    if (amountInMicroUnits + feeInMicroUnits > availableBalance) {
                        setAlertModal({
                            isOpen: true,
                            title: 'Insufficient Balance',
                            message: 'The amount plus fee exceeds your available balance.',
                            type: 'error'
                        });
                        return;
                    }
                }

        // Open password modal
        setPasswordModal({ isOpen: true, password: '' });
    };

    const handlePasswordSubmit = async () => {
        if (!passwordModal.password) {
            setAlertModal({
                isOpen: true,
                title: 'Password Required',
                message: 'Please enter your password.',
                type: 'error'
            });
            return;
        }

        setIsLoading(true);
        setPasswordModal({ isOpen: false, password: '' });

        try {
            // Find the account by nickname
            const account = accounts.find(acc => acc.nickname === formData.account);

            if (!account) {
                setAlertModal({
                    isOpen: true,
                    title: 'Account Not Found',
                    message: 'The selected account was not found. Please check your selection.',
                    type: 'error'
                });
                return;
            }

            if (activeTab === 'send') {
                const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
                const feeInMicroUnits = 0.01 * 1000000; // Fixed fee

                // Execute the send transaction
                await TxSend(
                    account.address,
                    formData.recipient,
                    amountInMicroUnits,
                    formData.memo,
                    feeInMicroUnits,
                    passwordModal.password,
                    true
                );
            } else if (activeTab === 'stake') {
                const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
                const feeInMicroUnits = 0.01 * 1000000; // Fixed fee
                
                // Find the signer account
                const signerAccount = accounts.find(acc => acc.nickname === formData.signer);
                if (!signerAccount) {
                    setAlertModal({
                        isOpen: true,
                        title: 'Signer Not Found',
                        message: 'The selected signer account was not found.',
                        type: 'error'
                    });
                    return;
                }
                
                        // Execute the stake transaction
                        await TxStake(
                            account.address,
                            account.publicKey || '', // Public key
                            formData.committees, // Already a string
                            formData.netAddress,
                            amountInMicroUnits,
                            formData.delegate,
                            formData.withdrawal,
                            formData.output,
                            signerAccount.address,
                            formData.memo,
                            feeInMicroUnits,
                            passwordModal.password,
                            true
                        );
                    } else if (activeTab === 'swap') {
                        const amountInMicroUnits = parseFloat(formData.amount) * 1000000;
                        const receiveAmountInMicroUnits = parseFloat(formData.receiveAmount) * 1000000;
                        const feeInMicroUnits = 0.01 * 1000000; // Fixed fee

                        // Execute the create order transaction
                        await TxCreateOrder(
                            account.address,
                            formData.chainId,
                            formData.data,
                            amountInMicroUnits,
                            receiveAmountInMicroUnits,
                            formData.receiveAddress,
                            formData.memo,
                            feeInMicroUnits,
                            passwordModal.password,
                            true
                        );
                    }

                    setSuccess(true);
            setTimeout(() => {
                onClose();
                setSuccess(false);
                        setFormData({
                            account: accounts[0]?.nickname || '',
                            recipient: '',
                            amount: '',
                            memo: '',
                            fee: 0.01,
                            delegate: true,
                            committees: '1',
                            netAddress: '',
                            withdrawal: true,
                            output: '',
                            signer: accounts[0]?.nickname || '',
                            chainId: '',
                            data: '',
                            receiveAmount: '',
                            receiveAddress: ''
                        });
            }, 2000);
        } catch (err) {
            setAlertModal({
                isOpen: true,
                title: 'Transaction Failed',
                message: err instanceof Error ? err.message : 'An unexpected error occurred while processing the transaction.',
                type: 'error'
            });
        } finally {
            setIsLoading(false);
        }
    };

    if (!isOpen) return null;

    const availableBalance = getAvailableBalance();
    const formattedBalance = formatBalance(availableBalance);

    return (
        <AnimatePresence mode="wait">
            {isOpen && (
                <motion.div
                    key="send-modal"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
                    onClick={onClose}
                >
                    <motion.div
                        initial={{ scale: 0.9, opacity: 0 }}
                        animate={{
                            scale: 1,
                            opacity: 1,
                            width: activeTab === 'receive' ? '28rem' : '56rem'
                        }}
                        exit={{ scale: 0.9, opacity: 0 }}
                        transition={{
                            duration: 0.3,
                            ease: "easeInOut",
                            width: { duration: 0.3, ease: "easeInOut" }
                        }}
                        className="bg-bg-secondary rounded-xl border border-bg-accent p-6 w-full"
                        style={{ maxWidth: activeTab === 'receive' ? '28rem' : '56rem' }}
                        onClick={(e) => e.stopPropagation()}
                    >
                        {/* Header with Tabs */}
                        <ModalTabs
                            activeTab={activeTab}
                            onTabChange={setActiveTab}
                            onClose={onClose}
                        />

                        {success ? (
                            <SuccessState />
                        ) : (
                            <AnimatePresence mode="wait">
                                {activeTab === 'send' && (
                                    <SendForm
                                        formData={formData}
                                        accounts={accounts}
                                        formattedBalance={formattedBalance}
                                        isLoading={isLoading}
                                        onInputChange={handleInputChange}
                                        onMaxAmount={handleMaxAmount}
                                        onSubmit={handleSubmit}
                                    />
                                )}
                                {activeTab === 'receive' && (
                                    <ReceiveForm
                                        formData={formData}
                                        accounts={accounts}
                                        onInputChange={handleInputChange}
                                        getSelectedAccountAddress={getSelectedAccountAddress}
                                    />
                                )}
                                        {activeTab === 'stake' && (
                                            <StakeForm
                                                formData={formData}
                                                accounts={accounts}
                                                formattedBalance={formattedBalance}
                                                isLoading={isLoading}
                                                onInputChange={handleInputChange}
                                                onMaxAmount={handleMaxAmount}
                                                onSubmit={handleSubmit}
                                            />
                                        )}
                                        {activeTab === 'swap' && (
                                            <SwapForm
                                                formData={formData}
                                                accounts={accounts}
                                                formattedBalance={formattedBalance}
                                                isLoading={isLoading}
                                                onInputChange={handleInputChange}
                                                onMaxAmount={handleMaxAmount}
                                                onSubmit={handleSubmit}
                                            />
                                        )}
                            </AnimatePresence>
                        )}

                        {/* Alert Modal */}
                        <AlertModal
                            isOpen={alertModal.isOpen}
                            onClose={() => setAlertModal(prev => ({ ...prev, isOpen: false }))}
                            title={alertModal.title}
                            message={alertModal.message}
                            type={alertModal.type}
                        />

                        {/* Password Modal */}
                        <PasswordModal
                            isOpen={passwordModal.isOpen}
                            password={passwordModal.password}
                            isLoading={isLoading}
                            onPasswordChange={(password) => setPasswordModal(prev => ({ ...prev, password }))}
                            onSubmit={handlePasswordSubmit}
                            onClose={() => setPasswordModal({ isOpen: false, password: '' })}
                        />
                    </motion.div>
                </motion.div>
            )}
        </AnimatePresence>
    );
};
