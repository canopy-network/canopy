import React from 'react';
import { motion } from 'framer-motion';
import { QRCodeSVG } from 'qrcode.react';

interface ReceiveFormProps {
    formData: {
        account: string;
    };
    accounts: Array<{
        address: string;
        nickname: string;
    }>;
    onInputChange: (field: string, value: string | number | boolean) => void;
    getSelectedAccountAddress: () => string;
}

export const ReceiveForm: React.FC<ReceiveFormProps> = ({
    formData,
    accounts,
    onInputChange,
    getSelectedAccountAddress
}) => {
    return (
        <motion.div
            key="receive-content"
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
        >
            {/* Receive Tab Content */}
            <div className="space-y-6">
                {/* Account Selection for Receive */}
                <div>
                    <label className="block text-sm font-medium text-text-primary mb-3">
                        Select Account to Receive
                    </label>
                    <div className="relative">
                        <select
                            value={formData.account}
                            onChange={(e) => onInputChange('account', e.target.value)}
                            className="w-full px-4 py-3 bg-bg-tertiary border border-gray-600 rounded-lg text-text-primary focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors appearance-none"
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

                {/* QR Code Section */}
                <div className="text-center space-y-4">
                    <div className="relative inline-block">
                        <div className="bg-white p-6 rounded-2xl shadow-2xl border-4 border-primary/20">
                            <QRCodeSVG
                                value={getSelectedAccountAddress()}
                                size={200}
                                level="M"
                                includeMargin={true}
                            />
                        </div>
                        <div className="absolute -top-2 -right-2 w-6 h-6 bg-primary rounded-full flex items-center justify-center">
                            <i className="fa-solid fa-qrcode text-white text-xs"></i>
                        </div>
                    </div>

                    <div className="space-y-3">
                        <h3 className="text-lg font-semibold text-text-primary">
                            Your Receive Address
                        </h3>
                        <div className="bg-bg-tertiary rounded-lg p-3 border border-bg-accent">
                            <p className="text-text-primary font-mono text-xs break-all leading-relaxed">
                                {getSelectedAccountAddress()}
                            </p>
                        </div>
                        <div className="flex gap-3 justify-center">
                            <button
                                onClick={() => navigator.clipboard.writeText(getSelectedAccountAddress())}
                                className="bg-primary hover:bg-primary/90 text-muted font-medium py-3 px-6 rounded-lg transition-colors flex items-center gap-2"
                            >
                                <i className="fa-solid fa-copy"></i>
                                Copy Address
                            </button>
                            <button
                                onClick={() => {
                                    const canvas = document.createElement('canvas');
                                    const ctx = canvas.getContext('2d');
                                    canvas.width = 200;
                                    canvas.height = 200;
                                    const img = new Image();
                                    img.onload = () => {
                                        ctx?.drawImage(img, 0, 0);
                                        const link = document.createElement('a');
                                        link.download = `qr-${getSelectedAccountAddress().slice(0, 8)}.png`;
                                        link.href = canvas.toDataURL();
                                        link.click();
                                    };
                                    img.src = `data:image/svg+xml;base64,${btoa(document.querySelector('svg')?.outerHTML || '')}`;
                                }}
                                className="bg-bg-accent hover:bg-bg-accent/80 text-text-primary font-medium py-3 px-6 rounded-lg transition-colors flex items-center gap-2 border border-bg-accent"
                            >
                                <i className="fa-solid fa-download"></i>
                                Download QR
                            </button>
                        </div>
                    </div>
                </div>

                {/* Instructions */}
                <div className="bg-bg-accent rounded-lg p-4 border border-bg-accent/50">
                    <h4 className="text-text-primary font-semibold mb-3 flex items-center gap-2">
                        <i className="fa-solid fa-info-circle text-primary"></i>
                        How to Receive
                    </h4>
                    <ul className="text-text-muted text-sm space-y-2">
                        <li className="flex items-start gap-2">
                            <span className="text-primary mt-1">•</span>
                            <span>Share this QR code or address with the sender</span>
                        </li>
                        <li className="flex items-start gap-2">
                            <span className="text-primary mt-1">•</span>
                            <span>Make sure the sender uses the correct network (Canopy Mainnet)</span>
                        </li>
                        <li className="flex items-start gap-2">
                            <span className="text-primary mt-1">•</span>
                            <span>Transactions typically take ~20 seconds to confirm</span>
                        </li>
                    </ul>
                </div>
            </div>
        </motion.div>
    );
};
