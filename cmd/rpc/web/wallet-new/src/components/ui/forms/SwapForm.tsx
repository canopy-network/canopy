import React from 'react';
import { motion } from 'framer-motion';
import { Account } from '@/hooks/useAccounts';
import { useManifest } from '@/hooks/useManifest';

interface SwapFormProps {
    formData: {
        account: string;
        chainId: string;
        data: string;
        amount: string;
        receiveAmount: string;
        receiveAddress: string;
        memo: string;
        fee: number;
    };
    accounts: Account[];
    formattedBalance: string;
    isLoading: boolean;
    onInputChange: (field: string, value: string | number | boolean) => void;
    onMaxAmount: () => void;
    onSubmit: (e: React.FormEvent) => void;
}

export const SwapForm: React.FC<SwapFormProps> = ({
    formData,
    accounts,
    formattedBalance,
    isLoading,
    onInputChange,
    onMaxAmount,
    onSubmit
}) => {
    const { getActionText, getText } = useManifest();
    return (
        <motion.div
            key="swap-content"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
        >
            <form onSubmit={onSubmit} className="space-y-6">
                {/* Swap Form Fields */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* Account */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.account.label', 'account')}
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

                    {/* Chain ID */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.chainId.label', 'commit-Id')}
                        </label>
                        <input
                            type="text"
                            value={formData.chainId}
                            onChange={(e) => onInputChange('chainId', e.target.value)}
                            placeholder={getActionText('Swap', 'form.fields.chainId.placeholder', 'the id of the committee / counter asset')}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                            required
                        />
                    </div>

                    {/* Data */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.data.label', 'data')}
                        </label>
                        <input
                            type="text"
                            value={formData.data}
                            onChange={(e) => onInputChange('data', e.target.value)}
                            placeholder={getActionText('Swap', 'form.fields.data.placeholder', 'optional hex data for sub-asset id')}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                        />
                    </div>

                    {/* Amount */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.amount.label', 'amount')}
                        </label>
                        <div className="relative">
                            <input
                                type="number"
                                value={formData.amount}
                                onChange={(e) => onInputChange('amount', e.target.value)}
                                placeholder={getActionText('Swap', 'form.fields.amount.placeholder', 'amount value for the tx in CNPY')}
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
                                {getText('ui.common.max', 'Max')}
                            </button>
                        </div>
                        <div className="flex justify-between items-center mt-1">
                            <span className="text-xs text-text-primary">
                                {getText('ui.common.uCNPY', 'uCNPY')}: {Math.floor(parseFloat(formData.amount || '0') * 1000000).toLocaleString()}
                            </span>
                            <span className="text-xs text-text-primary">
                                {getText('ui.common.available', 'Available')}: {formattedBalance} CNPY <span className="text-primary font-bold">{getText('ui.common.max', 'MAX')}</span>
                            </span>
                        </div>
                    </div>

                    {/* Receive Amount */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.receiveAmount.label', 'rec-amount')}
                        </label>
                        <input
                            type="number"
                            value={formData.receiveAmount}
                            onChange={(e) => onInputChange('receiveAmount', e.target.value)}
                            placeholder={getActionText('Swap', 'form.fields.receiveAmount.placeholder', 'amount of counter asset to receive')}
                            step="0.000001"
                            min="0"
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                            required
                        />
                    </div>

                    {/* Receive Address */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.receiveAddress.label', 'rec-addr')}
                        </label>
                        <input
                            type="text"
                            value={formData.receiveAddress}
                            onChange={(e) => onInputChange('receiveAddress', e.target.value)}
                            placeholder={getActionText('Swap', 'form.fields.receiveAddress.placeholder', 'the address where the counter asset will be sent')}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                            required
                            minLength={40}
                            maxLength={40}
                        />
                    </div>

                    {/* Memo - Full Width */}
                    <div className="col-span-2">
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            {getActionText('Swap', 'form.fields.memo.label', 'memo')}
                        </label>
                        <textarea
                            value={formData.memo}
                            onChange={(e) => onInputChange('memo', e.target.value)}
                            placeholder={getActionText('Swap', 'form.fields.memo.placeholder', 'opt: note attached with the transaction')}
                            maxLength={200}
                            rows={3}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors resize-none"
                        />
                    </div>
                </div>

                {/* Network Fee Section */}
                <div className="bg-[#1A1B23] rounded-lg p-4">
                    <h4 className="text-text-primary font-semibold mb-3">{getActionText('Swap', 'ui.networkFee.title', 'Network Fee')}</h4>
                    <div className="flex justify-between items-center mb-2">
                        <span className="text-text-muted text-sm">{getActionText('Swap', 'ui.networkFee.estimatedFee', 'Estimated Fee:')}</span>
                        <span className="text-text-primary text-sm">{getActionText('Swap', 'ui.networkFee.feeAmount', '0.01 CNPY')}</span>
                    </div>
                    <div className="flex justify-between items-center">
                        <span className="text-text-muted text-sm">{getActionText('Swap', 'ui.networkFee.estimatedTime', 'Estimated Time:')}</span>
                        <span className="text-text-primary text-sm">{getActionText('Swap', 'ui.networkFee.timeAmount', '~20 seconds')}</span>
                    </div>
                </div>

                {/* Create Order Button */}
                <button
                    type="submit"
                    disabled={isLoading}
                    className="w-full bg-primary hover:bg-primary/90 disabled:bg-primary/50 text-muted font-semibold py-4 px-4 rounded-lg transition-colors flex items-center justify-center gap-3"
                >
                    {isLoading ? (
                        <>
                            <i className="fa-solid fa-spinner fa-spin"></i>
                            {getActionText('Swap', 'ui.buttons.processing', 'Processing...')}
                        </>
                    ) : (
                        <>
                            <i className="fa-solid fa-exchange-alt"></i>
                            {getActionText('Swap', 'ui.buttons.createOrder', 'Create Order')}
                        </>
                    )}
                </button>
            </form>
        </motion.div>
    );
};
