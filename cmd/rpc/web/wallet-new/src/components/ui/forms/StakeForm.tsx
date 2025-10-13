import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';

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
    const { getText } = useManifest();
    const [activeTab, setActiveTab] = useState<'stake' | 'unstake'>('stake');
    const [currentStep, setCurrentStep] = useState(1);
    const totalSteps = 3;

    const handleTabChange = (tab: 'stake' | 'unstake') => {
        setActiveTab(tab);
    };

    const handleContinue = (e: React.FormEvent) => {
        e.preventDefault();
        if (currentStep < totalSteps) {
            setCurrentStep(currentStep + 1);
        } else {
            onSubmit(e);
        }
    };

    const handleBack = () => {
        if (currentStep > 1) {
            setCurrentStep(currentStep - 1);
        }
    };

    return (
        <motion.div
            key="stake-content"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="p-4"
        >
            {/* Tabs */}
            <div className="flex mb-8 border-b border-bg-accent">
                <button
                    className={`pb-2 px-4 font-medium text-base ${activeTab === 'stake' ? 'text-primary border-b-2 border-primary' : 'text-text-secondary'}`}
                    onClick={() => handleTabChange('stake')}
                >
                    {getText('ui.staking.stake', 'Stake')}
                </button>
                <button
                    className={`pb-2 px-4 font-medium text-base ${activeTab === 'unstake' ? 'text-primary border-b-2 border-primary' : 'text-text-secondary'}`}
                    onClick={() => handleTabChange('unstake')}
                >
                    {getText('ui.staking.unstake', 'Unstake')}
                </button>
            </div>

            <div className="mb-8">
                <h2 className="text-2xl font-bold text-white mb-2">
                    {getText('ui.staking.stakeYourTokens', 'Stake Your Tokens')}
                </h2>
                <p className="text-text-secondary text-sm">
                    {getText('ui.staking.completeSetup', 'Complete your staking setup in 3 simple steps')}
                </p>
            </div>

            {/* Progress Steps */}
            <div className="flex items-center justify-between mb-8 relative">
                <div className="absolute left-0 right-0 top-1/2 h-0.5 bg-bg-accent -z-10"></div>
                
                {[1, 2, 3].map((step) => (
                    <div key={step} className="flex flex-col items-center">
                        <div 
                            className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium mb-1
                                ${currentStep === step 
                                    ? 'bg-primary text-primary-foreground' 
                                    : currentStep > step 
                                        ? 'bg-primary text-primary-foreground'
                                        : 'bg-bg-tertiary text-text-secondary'}`}
                        >
                            {currentStep > step ? <i className="fa-solid fa-check"></i> : step}
                        </div>
                        <span className="text-xs text-text-secondary">
                            {step === 1 ? 'Basic Setup' : step === 2 ? 'Configuration' : 'Review'}
                        </span>
                    </div>
                ))}
            </div>

            <form onSubmit={handleContinue}>
                {currentStep === 1 && (
                    <div className="space-y-6">
                        {/* Step 1: Basic Setup */}
                    <div>
                            <label className="block text-sm font-medium text-text-secondary mb-2">
                                {getText('ui.staking.addressToUse', 'Address to Use')}
                        </label>
                        <div className="relative">
                            <select
                                value={formData.account}
                                onChange={(e) => onInputChange('account', e.target.value)}
                                    className="w-full px-4 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none"
                                required
                            >
                                {accounts.map((account) => (
                                    <option key={account.address} value={account.nickname}>
                                            {account.nickname} ({account.address.substring(0, 6)}...{account.address.substring(account.address.length - 4)})
                                    </option>
                                ))}
                            </select>
                                <i className="fa-solid fa-chevron-down absolute right-4 top-1/2 transform -translate-y-1/2 text-text-muted"></i>
                            </div>
                        </div>

                        <div className="space-y-2">
                            <label className="block text-sm font-medium text-text-secondary mb-2">
                                {getText('ui.staking.stakeType', 'Stake Type')}
                            </label>
                            <div className="grid grid-cols-2 gap-4">
                                <div 
                                    className={`border ${formData.delegate ? 'border-primary bg-primary/10' : 'border-bg-accent bg-bg-tertiary'} rounded-lg p-4 cursor-pointer`}
                                    onClick={() => onInputChange('delegate', true)}
                                >
                                    <div className="flex justify-between items-center mb-2">
                                        <span className="text-white font-medium">Validation</span>
                                        <div className={`w-4 h-4 rounded-full border ${formData.delegate ? 'border-primary' : 'border-text-muted'} flex items-center justify-center`}>
                                            {formData.delegate && <div className="w-2 h-2 rounded-full bg-primary"></div>}
                                        </div>
                                    </div>
                                    <p className="text-text-secondary text-xs">Run your own validator</p>
                                </div>
                                
                                <div 
                                    className={`border ${!formData.delegate ? 'border-primary bg-primary/10' : 'border-bg-accent bg-bg-tertiary'} rounded-lg p-4 cursor-pointer`}
                                    onClick={() => onInputChange('delegate', false)}
                                >
                                    <div className="flex justify-between items-center mb-2">
                                        <span className="text-white font-medium">Delegation</span>
                                        <div className={`w-4 h-4 rounded-full border ${!formData.delegate ? 'border-primary' : 'border-text-muted'} flex items-center justify-center`}>
                                            {!formData.delegate && <div className="w-2 h-2 rounded-full bg-primary"></div>}
                                        </div>
                                    </div>
                                    <p className="text-text-secondary text-xs">Delegate to committee</p>
                                </div>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-text-secondary mb-2">
                                {getText('ui.staking.stakeAmount', 'Stake Amount')}
                            </label>
                            <div className="relative">
                                <input
                                    type="number"
                                    value={formData.amount}
                                    onChange={(e) => onInputChange('amount', e.target.value)}
                                    placeholder="0.00"
                                    step="0.000001"
                                    min="0"
                                    className="w-full px-4 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors pr-20"
                                    required
                                />
                                <div className="absolute right-2 top-1/2 transform -translate-y-1/2 flex items-center">
                                    <span className="text-text-secondary mr-2">CNPY</span>
                                    <button
                                        type="button"
                                        onClick={onMaxAmount}
                                        className="bg-primary text-primary-foreground px-2 py-1 rounded text-xs font-medium hover:bg-primary/90 transition-colors"
                                    >
                                        {getText('ui.staking.max', 'Max')}
                                    </button>
                                </div>
                            </div>
                            <div className="flex justify-between items-center mt-1">
                                <span className="text-xs text-text-secondary">
                                    {getText('ui.staking.available', 'Available')}: {formattedBalance} CNPY
                                </span>
                            </div>
                        </div>
                    </div>
                )}

                {currentStep === 2 && (
                    <div className="space-y-6">
                        {/* Step 2: Configuration */}
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* Committees */}
                    <div>
                                <label className="block text-sm font-medium text-text-secondary mb-2">
                                    {getText('ui.staking.chains', 'Chains')}
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
                                        className="w-10 h-10 bg-bg-tertiary hover:bg-bg-accent/80 border border-bg-accent rounded-lg flex items-center justify-center text-white transition-colors"
                            >
                                <i className="fa-solid fa-minus text-sm"></i>
                            </button>
                            <input
                                type="number"
                                value={formData.committees}
                                onChange={(e) => onInputChange('committees', e.target.value)}
                                min="1"
                                        className="flex-1 px-3 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors text-center"
                                required
                            />
                            <button
                                type="button"
                                onClick={() => {
                                    const currentValue = parseInt(formData.committees) || 1;
                                    onInputChange('committees', (currentValue + 1).toString());
                                }}
                                        className="w-10 h-10 bg-bg-tertiary hover:bg-bg-accent/80 border border-bg-accent rounded-lg flex items-center justify-center text-white transition-colors"
                            >
                                <i className="fa-solid fa-plus text-sm"></i>
                            </button>
                        </div>
                        <p className="text-text-muted text-xs mt-1">
                                    {getText('ui.staking.chainsToStake', 'Number of chains to stake for')}
                        </p>
                    </div>

                    {/* Net Address */}
                    <div>
                                <label className="block text-sm font-medium text-text-secondary mb-2">
                                    {getText('ui.staking.netAddress', 'Net Address')}
                        </label>
                        <input
                            type="text"
                            value={formData.netAddress}
                            onChange={(e) => onInputChange('netAddress', e.target.value)}
                            placeholder="url of the node"
                                    className="w-full px-4 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                                    required={formData.delegate}
                                />
                                <p className="text-text-muted text-xs mt-1">
                                    {getText('ui.staking.nodeUrl', 'URL of your validator node')}
                                </p>
                            </div>
                    </div>

                        {/* Autocompound */}
                    <div>
                            <div className="flex justify-between items-center mb-2">
                                <label className="text-sm font-medium text-text-secondary">
                                    {getText('ui.staking.autocompound', 'Autocompound')}
                        </label>
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
                            <p className="text-text-muted text-xs">
                                {getText('ui.staking.autocompoundDesc', 'Automatically restake rewards')}
                            </p>
                        </div>
                        
                        {/* Advanced Options */}
                        <div className="bg-bg-tertiary rounded-lg p-4">
                            <div className="flex justify-between items-center mb-2">
                                <h4 className="text-white font-medium">Advanced Options</h4>
                                <button
                                    type="button"
                                    className="text-text-secondary hover:text-white transition-colors"
                                >
                                    <i className="fa-solid fa-chevron-down"></i>
                                </button>
                            </div>
                            <div className="space-y-4">
                                {/* Output */}
                                <div>
                                    <label className="block text-sm font-medium text-text-secondary mb-2">
                                        {getText('ui.staking.output', 'Output')}
                                    </label>
                                    <input
                                        type="text"
                                        value={formData.output}
                                        onChange={(e) => onInputChange('output', e.target.value)}
                                        placeholder="Optional output address"
                                        className="w-full px-4 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors"
                                    />
                    </div>

                                {/* Memo */}
                                <div>
                                    <label className="block text-sm font-medium text-text-secondary mb-2">
                                        {getText('ui.staking.memo', 'Memo')}
                        </label>
                        <textarea
                            value={formData.memo}
                            onChange={(e) => onInputChange('memo', e.target.value)}
                                        placeholder="Optional note for this transaction"
                            maxLength={200}
                                        rows={2}
                                        className="w-full px-4 py-3 bg-bg-tertiary border border-bg-accent rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors resize-none"
                                    />
                                </div>
                            </div>
                        </div>
                    </div>
                )}

                {currentStep === 3 && (
                    <div className="space-y-6">
                        {/* Step 3: Review */}
                        <div className="bg-bg-tertiary rounded-lg p-4 space-y-4">
                            <h4 className="text-white font-medium">Transaction Summary</h4>
                            
                            <div className="space-y-2">
                                <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                    <span className="text-text-secondary">Account</span>
                                    <span className="text-white">{formData.account}</span>
                                </div>
                                
                                <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                    <span className="text-text-secondary">Type</span>
                                    <span className="text-white">{formData.delegate ? 'Validation' : 'Delegation'}</span>
                                </div>
                                
                                <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                    <span className="text-text-secondary">Amount</span>
                                    <span className="text-white">{formData.amount} CNPY</span>
                                </div>
                                
                                <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                    <span className="text-text-secondary">Chains</span>
                                    <span className="text-white">{formData.committees}</span>
                                </div>
                                
                                {formData.netAddress && (
                                    <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                        <span className="text-text-secondary">Net Address</span>
                                        <span className="text-white">{formData.netAddress}</span>
                                    </div>
                                )}
                                
                                <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                    <span className="text-text-secondary">Autocompound</span>
                                    <span className="text-white">{formData.withdrawal ? 'Enabled' : 'Disabled'}</span>
                                </div>
                                
                                {formData.memo && (
                                    <div className="flex justify-between items-center py-2 border-b border-bg-accent">
                                        <span className="text-text-secondary">Memo</span>
                                        <span className="text-white">{formData.memo}</span>
                                    </div>
                                )}
                    </div>
                </div>

                {/* Network Fee Section */}
                        <div className="bg-bg-tertiary rounded-lg p-4">
                            <h4 className="text-white font-medium mb-3">Network Fee</h4>
                    <div className="flex justify-between items-center mb-2">
                                <span className="text-text-secondary text-sm">Estimated Fee:</span>
                                <span className="text-white text-sm">0.01 CNPY</span>
                    </div>
                    <div className="flex justify-between items-center">
                                <span className="text-text-secondary text-sm">Estimated Time:</span>
                                <span className="text-white text-sm">~20 seconds</span>
                            </div>
                        </div>
                    </div>
                )}

                {/* Navigation Buttons */}
                <div className="flex justify-between mt-8">
                    {currentStep > 1 ? (
                        <button
                            type="button"
                            onClick={handleBack}
                            className="px-6 py-3 border border-bg-accent text-white rounded-lg hover:bg-bg-accent/30 transition-colors"
                        >
                            <i className="fa-solid fa-arrow-left mr-2"></i>
                            {getText('ui.common.back', 'Back')}
                        </button>
                    ) : (
                        <div></div>
                    )}

                <button
                    type="submit"
                    disabled={isLoading}
                        className="px-6 py-3 bg-primary hover:bg-primary/90 disabled:bg-primary/50 text-primary-foreground font-medium rounded-lg transition-colors flex items-center"
                >
                    {isLoading ? (
                        <>
                                <i className="fa-solid fa-spinner fa-spin mr-2"></i>
                                {getText('ui.common.processing', 'Processing...')}
                            </>
                        ) : currentStep < totalSteps ? (
                            <>
                                {getText('ui.common.continue', 'Continue')}
                                <i className="fa-solid fa-arrow-right ml-2"></i>
                        </>
                    ) : (
                        <>
                                <i className="fa-solid fa-paper-plane mr-2"></i>
                                {getText('ui.staking.confirmStake', 'Confirm Stake')}
                        </>
                    )}
                </button>
                </div>
            </form>
        </motion.div>
    );
};