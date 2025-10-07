import React from 'react';
import { motion } from 'framer-motion';

interface StakeFormProps {
    formData: {
        account: string;
        delegate: boolean;
        committees: string;
        netAddress: string;
        amount: string;
        withdrawal: boolean;
        output: string;
        signer: string;
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

export const StakeForm: React.FC<StakeFormProps> = ({
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
            key="stake-content"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
        >
            <form onSubmit={onSubmit} className="space-y-6">
                {/* Stake Form Fields */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* Account */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            account
                        </label>
                        <div className="relative">
                            <select
                                value={formData.account}
                                onChange={(e) => onInputChange('account', e.target.value)}
                                className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none"
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

                    {/* Committees */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            committees
                        </label>
                        <div className="flex items-center gap-2">
                            <button
                                type="button"
                                onClick={() => {
                                    const currentValue = parseInt(formData.committees) || 1;
                                    if (currentValue > 1) {
                                        onInputChange('committees', (currentValue - 1).toString());
                                    }
                                }}
                                className="w-10 h-10 bg-bg-accent hover:bg-bg-accent/80 border border-bg-accent rounded-lg flex items-center justify-center text-text-primary transition-colors"
                            >
                                <i className="fa-solid fa-minus text-sm"></i>
                            </button>
                            <input
                                type="number"
                                value={formData.committees}
                                onChange={(e) => onInputChange('committees', e.target.value)}
                                min="1"
                                className="flex-1 px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors text-center"
                                required
                            />
                            <button
                                type="button"
                                onClick={() => {
                                    const currentValue = parseInt(formData.committees) || 1;
                                    onInputChange('committees', (currentValue + 1).toString());
                                }}
                                className="w-10 h-10 bg-bg-accent hover:bg-bg-accent/80 border border-bg-accent rounded-lg flex items-center justify-center text-text-primary transition-colors"
                            >
                                <i className="fa-solid fa-plus text-sm"></i>
                            </button>
                        </div>
                        <p className="text-text-muted text-xs mt-1">
                            Number of committees to stake for
                        </p>
                    </div>

                    {/* Amount */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            amount
                        </label>
                        <div className="relative">
                            <input
                                type="number"
                                value={formData.amount}
                                onChange={(e) => onInputChange('amount', e.target.value)}
                                placeholder="1,000.000000"
                                step="0.000001"
                                min="0"
                                className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors pr-16"
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
                        <div className="flex justify-between items-center mt-1">
                            <span className="text-xs text-text-primary">
                                uCNPY: {Math.floor(parseFloat(formData.amount || '0') * 1000000).toLocaleString()}
                            </span>
                            <span className="text-xs text-text-primary">
                                Available: {formattedBalance} CNPY <span className="text-primary font-bold">MAX</span>
                            </span>
                        </div>
                    </div>

                    {/* Net Address */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            net-addr
                        </label>
                        <input
                            type="text"
                            value={formData.netAddress}
                            onChange={(e) => onInputChange('netAddress', e.target.value)}
                            placeholder="url of the node"
                            className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                            required
                        />
                    </div>

                    {/* Output */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            output
                        </label>
                        <input
                            type="text"
                            value={formData.output}
                            onChange={(e) => onInputChange('output', e.target.value)}
                            placeholder="02cd4e5eb53ea665702042a6ed6d31d61605"
                            className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                        />
                    </div>

                    {/* Signer */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            signer
                        </label>
                        <div className="relative">
                            <select
                                value={formData.signer}
                                onChange={(e) => onInputChange('signer', e.target.value)}
                                className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none"
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

                    {/* Delegate Switch */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            delegate
                        </label>
                        <div className="flex items-center gap-3">
                            <button
                                type="button"
                                onClick={() => onInputChange('delegate', !formData.delegate)}
                                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-bg-secondary ${
                                    formData.delegate ? 'bg-primary' : 'bg-bg-accent'
                                }`}
                            >
                                <span
                                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                                        formData.delegate ? 'translate-x-6' : 'translate-x-1'
                                    }`}
                                />
                            </button>
                        </div>
                    </div>

                    {/* Withdrawal Switch */}
                    <div>
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            withdrawal
                        </label>
                        <div className="flex items-center gap-3">
                            <button
                                type="button"
                                onClick={() => onInputChange('withdrawal', !formData.withdrawal)}
                                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-bg-secondary ${
                                    formData.withdrawal ? 'bg-primary' : 'bg-bg-accent'
                                }`}
                            >
                                <span
                                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                                        formData.withdrawal ? 'translate-x-6' : 'translate-x-1'
                                    }`}
                                />
                            </button>
                        </div>
                    </div>

                    {/* Memo - Full Width */}
                    <div className="col-span-2">
                        <label className="block text-sm font-medium text-text-primary mb-2">
                            memo
                        </label>
                        <textarea
                            value={formData.memo}
                            onChange={(e) => onInputChange('memo', e.target.value)}
                            placeholder="opt: note attached with the transaction"
                            maxLength={200}
                            rows={3}
                            className="w-full px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors resize-none"
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

                {/* Generate Transaction Button */}
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
                            Generate Transaction
                        </>
                    )}
                </button>
            </form>
        </motion.div>
    );
};
