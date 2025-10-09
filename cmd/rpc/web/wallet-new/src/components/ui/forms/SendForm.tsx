import React from 'react';
import { motion } from 'framer-motion';

interface SendFormProps {
    formData: {
        account: string;
        recipient: string;
        amount: string;
        memo: string;
        fee: number;
    };
    accounts: Array<{
        address: string;
        nickname: string;
    }>;
    formattedBalance: string;
    isLoading: boolean;
    onInputChange: (field: string, value: string | number | boolean) => void;
    onMaxAmount: () => void;
    onSubmit: (e: React.FormEvent) => void;
}

export const SendForm: React.FC<SendFormProps> = ({
    formData,
    accounts,
    formattedBalance,
    isLoading,
    onInputChange,
    onMaxAmount,
    onSubmit
}) => {
    return (
        <motion.div
            key="send-form"
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
        >
            <form onSubmit={onSubmit} className="space-y-6">
                {/* Two Column Layout */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* To Address */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            To Address
                        </label>
                        <div className="relative">
                            <input
                                type="text"
                                value={formData.recipient}
                                onChange={(e) => onInputChange('recipient', e.target.value)}
                                placeholder="Enter recipient address"
                                className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors pr-20"
                                required
                                minLength={40}
                                maxLength={40}
                            />
                            <button
                                type="button"
                                className="absolute right-2 top-1/2 transform -translate-y-1/2 bg-primary text-muted px-3 py-1 rounded text-sm font-medium hover:bg-primary/90 transition-colors"
                            >
                                Paste
                            </button>
                        </div>
                        <button
                            type="button"
                            className="flex items-center gap-2 text-primary hover:text-primary/80 text-sm mt-2 transition-colors"
                        >
                            <i className="fa-solid fa-address-book text-xs"></i>
                            Choose from Address Book
                        </button>
                    </div>

                    {/* Chain (Accounts) */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            Chain
                        </label>
                        <div className="relative">
                            <select
                                value={formData.account}
                                onChange={(e) => onInputChange('account', e.target.value)}
                                className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none"
                                required
                            >
                                {accounts.map((account) => (
                                    <option key={account.address} value={account.nickname}>
                                        {account.nickname}
                                    </option>
                                ))}
                            </select>
                            <i className="fa-solid fa-chevron-down absolute right-3 top-1/2 transform -translate-y-1/2 text-text-muted"></i>
                        </div>
                    </div>

                    {/* Asset */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            Asset
                        </label>
                        <div className="relative">
                            <select className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none">
                                <option value="cnpy">CNPY (Balance: {formattedBalance})</option>
                            </select>
                            <i className="fa-solid fa-chevron-down absolute right-3 top-1/2 transform -translate-y-1/2 text-text-muted"></i>
                        </div>
                    </div>

                    {/* Amount */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            Amount
                        </label>
                        <div className="relative">
                            <input
                                type="number"
                                value={formData.amount}
                                onChange={(e) => onInputChange('amount', e.target.value)}
                                placeholder="0.00"
                                step="0.000001"
                                min="0"
                                className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors pr-16"
                                required
                            />
                            <button
                                type="button"
                                onClick={onMaxAmount}
                                className="absolute right-2 top-1/2 transform -translate-y-1/2 bg-primary text-muted px-3 py-1 rounded text-sm font-medium hover:bg-primary/90 transition-colors"
                            >
                                Max
                            </button>
                        </div>
                        <p className="text-text-muted text-sm mt-1">
                            ≈ $0.00 USD
                        </p>
                    </div>

                    {/* Memo */}
                    <div className="col-span-2">
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            Memo
                        </label>
                        <textarea
                            value={formData.memo}
                            onChange={(e) => onInputChange('memo', e.target.value)}
                            placeholder="Optional note attached with the transaction"
                            maxLength={200}
                            rows={3}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors resize-none"
                        />
                    </div>
                </div>

                {/* Network Fee Section */}
                <div className="bg-[#1A1B23] rounded-lg p-4">
                    <h4 className="text-text-primary font-semibold mb-3">Network Fee</h4>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-text-muted text-sm">Estimated Fee:</span>
                        <span className="text-text-primary text-sm">0.01 CNPY</span>
                    </div>
                    <div className="flex justify-between items-center">
                        <span className="text-text-muted text-sm">Estimated Time:</span>
                        <span className="text-text-primary text-sm">~20 seconds</span>
                    </div>
                </div>

                {/* Send Transaction Button */}
                <button
                    type="submit"
                    disabled={isLoading}
                    className="w-full bg-primary hover:bg-primary/90 disabled:bg-primary/50 text-muted font-semibold py-4 px-4 rounded-lg transition-colors flex items-center justify-center gap-3"
                >
                    {isLoading ? (
                        <>
                            <i className="fa-solid fa-spinner fa-spin"></i>
                            Processing...
                        </>
                    ) : (
                        <>
                            <i className="fa-solid fa-paper-plane"></i>
                            Send Transaction
                        </>
                    )}
                </button>
            </form>
        </motion.div>
    );
};
