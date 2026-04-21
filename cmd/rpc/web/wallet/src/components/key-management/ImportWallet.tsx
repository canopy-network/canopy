import React, { useState, useRef } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle, Eye, EyeOff, FileJson } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { useToast } from '@/toast/ToastContext';
import { useDSFetcher } from '@/core/dsFetch';
import { useQueryClient } from '@tanstack/react-query';

interface EncryptedKeyFile {
    publicKey: string;
    salt: string;
    encrypted: string;
    keyAddress: string;
    keyNickname?: string;
}

export const ImportWallet = ({ embedded = false }: { embedded?: boolean }): JSX.Element => {
    const toast = useToast();
    const dsFetch = useDSFetcher();
    const queryClient = useQueryClient();

    const [showPrivateKey, setShowPrivateKey] = useState(false);
    const [activeTab, setActiveTab] = useState<'key' | 'keystore'>('key');
    const [importForm, setImportForm] = useState({
        privateKey: '',
        password: '',
        confirmPassword: '',
        nickname: ''
    });

    const [keystoreForm, setKeystoreForm] = useState({ nickname: '' });
    const [keystoreFile, setKeystoreFile] = useState<EncryptedKeyFile | null>(null);
    const [keystoreFileName, setKeystoreFileName] = useState('');
    const fileInputRef = useRef<HTMLInputElement>(null);

    const panelVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4 }
        }
    };

    const invalidateKeystore = async () => {
        await queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });
        setTimeout(() => {
            queryClient.invalidateQueries({ queryKey: ['ds', 'keystore'] });
        }, 500);
    };

    const handleImportWallet = async () => {
        if (!importForm.privateKey) {
            toast.error({ title: 'Missing private key', description: 'Please enter a private key.' });
            return;
        }

        if (!importForm.nickname) {
            toast.error({ title: 'Missing wallet name', description: 'Please enter a wallet name.' });
            return;
        }

        if (!importForm.password) {
            toast.error({ title: 'Missing password', description: 'Please enter a password.' });
            return;
        }

        if (importForm.password !== importForm.confirmPassword) {
            toast.error({ title: 'Password mismatch', description: 'Passwords do not match.' });
            return;
        }

        const cleanPrivateKey = importForm.privateKey.trim().replace(/^0x/, '');
        if (!/^[0-9a-fA-F]{64,128}$/.test(cleanPrivateKey)) {
            toast.error({
                title: 'Invalid private key',
                description: 'Private key must be 64-128 hexadecimal characters.'
            });
            return;
        }

        const loadingToast = toast.info({
            title: 'Importing wallet...',
            description: 'Please wait while your wallet is imported.',
            sticky: true,
        });

        try {
            await dsFetch('keystoreImportRaw', {
                nickname: importForm.nickname,
                password: importForm.password,
                privateKey: cleanPrivateKey
            });

            await invalidateKeystore();

            toast.dismiss(loadingToast);
            toast.success({
                title: 'Wallet imported',
                description: `Wallet "${importForm.nickname}" has been imported successfully.`,
            });

            setImportForm({ privateKey: '', password: '', confirmPassword: '', nickname: '' });
        } catch (error) {
            toast.dismiss(loadingToast);
            toast.error({
                title: 'Error importing wallet',
                description: error instanceof Error ? error.message : String(error)
            });
        }
    };

    const handleKeystoreFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        const reader = new FileReader();
        reader.onload = (ev) => {
            try {
                const parsed = JSON.parse(ev.target?.result as string) as Record<string, unknown>;

                let epk: EncryptedKeyFile | null = null;

                if (parsed.addressMap && typeof parsed.addressMap === 'object') {
                    const entries = Object.values(parsed.addressMap as Record<string, EncryptedKeyFile>);
                    if (entries.length > 0) epk = entries[0];
                } else if (parsed.publicKey && parsed.salt && parsed.encrypted && parsed.keyAddress) {
                    epk = parsed as unknown as EncryptedKeyFile;
                }

                if (!epk || !epk.publicKey || !epk.salt || !epk.encrypted || !epk.keyAddress) {
                    toast.error({
                        title: 'Invalid keystore file',
                        description: 'File must be an exported keyfile with addressMap or contain publicKey, salt, encrypted, and keyAddress fields.'
                    });
                    setKeystoreFile(null);
                    setKeystoreFileName('');
                    return;
                }

                setKeystoreFile(epk);
                setKeystoreFileName(file.name);

                const nickname = epk.keyNickname
                    || (parsed.nicknameMap ? Object.keys(parsed.nicknameMap as Record<string, string>)[0] : '');
                if (nickname && !keystoreForm.nickname) {
                    setKeystoreForm(prev => ({ ...prev, nickname }));
                }
            } catch {
                toast.error({ title: 'Invalid file', description: 'Could not parse JSON keystore file.' });
                setKeystoreFile(null);
                setKeystoreFileName('');
            }
        };
        reader.readAsText(file);
    };

    const handleImportKeystore = async () => {
        if (!keystoreFile) {
            toast.error({ title: 'Missing keystore file', description: 'Please select a keystore JSON file.' });
            return;
        }

        const nickname = keystoreForm.nickname || keystoreFile.keyNickname || '';
        if (!nickname) {
            toast.error({ title: 'Missing wallet name', description: 'Please enter a wallet name.' });
            return;
        }

        const loadingToast = toast.info({
            title: 'Importing keystore...',
            description: 'Please wait while your keystore is imported.',
            sticky: true,
        });

        try {
            await dsFetch('keystoreImport', {
                nickname,
                address: keystoreFile.keyAddress,
                publicKey: keystoreFile.publicKey,
                salt: keystoreFile.salt,
                encrypted: keystoreFile.encrypted,
                keyAddress: keystoreFile.keyAddress,
            });

            await invalidateKeystore();

            toast.dismiss(loadingToast);
            toast.success({
                title: 'Keystore imported',
                description: `Wallet "${nickname}" has been imported successfully.`,
            });

            setKeystoreForm({ nickname: '' });
            setKeystoreFile(null);
            setKeystoreFileName('');
            if (fileInputRef.current) fileInputRef.current.value = '';
        } catch (error) {
            toast.dismiss(loadingToast);
            toast.error({
                title: 'Error importing keystore',
                description: error instanceof Error ? error.message : String(error)
            });
        }
    };

    const content = (
        <>
            <div className="mb-6">
                <div>
                    <h2 className="text-xl font-bold text-foreground">Import Wallet</h2>
                    <p className="text-xs text-muted-foreground mt-1">Bring an existing key into this node securely.</p>
                </div>
            </div>

            <div className="grid grid-cols-2 gap-2 mb-6">
                <button
                    onClick={() => setActiveTab('key')}
                    className={`px-3 py-2.5 text-sm font-medium rounded-lg transition-all border ${activeTab === 'key'
                        ? 'border-[#272729] bg-[#0f0f0f] text-foreground'
                        : 'border-[#272729] bg-transparent text-muted-foreground hover:bg-[#0f0f0f] hover:text-foreground'
                        }`}
                >
                    Private Key
                </button>
                <button
                    onClick={() => setActiveTab('keystore')}
                    className={`px-3 py-2.5 text-sm font-medium rounded-lg transition-all border ${activeTab === 'keystore'
                        ? 'border-[#272729] bg-[#0f0f0f] text-foreground'
                        : 'border-[#272729] bg-transparent text-muted-foreground hover:bg-[#0f0f0f] hover:text-foreground'
                        }`}
                >
                    Keystore
                </button>
            </div>

            {activeTab === 'key' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Imported Wallet"
                            value={importForm.nickname}
                            onChange={(e) => setImportForm({ ...importForm, nickname: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Private Key
                        </label>
                        <div className="relative">
                            <input
                                type={showPrivateKey ? "text" : "password"}
                                placeholder="Enter your private key..."
                                value={importForm.privateKey}
                                onChange={(e) => setImportForm({ ...importForm, privateKey: e.target.value })}
                                className="w-full rounded-lg border border-border bg-muted px-3 py-2.5 pr-10 text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/10"
                            />
                            <button
                                onClick={() => setShowPrivateKey(!showPrivateKey)}
                                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                {showPrivateKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Password
                        </label>
                        <input
                            type="password"
                            placeholder="Password"
                            value={importForm.password}
                            onChange={(e) => setImportForm({ ...importForm, password: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Confirm Password
                        </label>
                        <input
                            type="password"
                            placeholder="Confirm your password...."
                            value={importForm.confirmPassword}
                            onChange={(e) => setImportForm({ ...importForm, confirmPassword: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2.5 text-foreground"
                        />
                    </div>

                    <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
                        <div className="flex items-start gap-3">
                            <AlertTriangle className="mt-0.5 h-5 w-5 text-white/60" />
                            <div>
                                <h4 className="mb-1 font-medium text-foreground">Import Security Warning</h4>
                                <p className="text-sm text-muted-foreground">
                                    Only import wallets from trusted sources. Verify all information before proceeding.
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportWallet}
                        className="h-11 w-full"
                    >
                        Import Wallet
                    </Button>
                </div>
            )}

            {activeTab === 'keystore' && (
                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Keystore File
                        </label>
                        <div className="mb-2 inline-flex items-center gap-2 rounded-md border border-border bg-muted/30 px-2 py-1 text-[11px] text-muted-foreground">
                            <FileJson className="w-3.5 h-3.5" />
                            {keystoreFileName || 'Upload encrypted JSON keystore'}
                        </div>
                        <input
                            ref={fileInputRef}
                            type="file"
                            accept=".json"
                            onChange={handleKeystoreFileChange}
                            className="w-full rounded-lg border border-border bg-muted px-3 py-2 text-foreground file:mr-4 file:rounded-lg file:border file:border-[#272729] file:bg-[#0f0f0f] file:px-4 file:py-2 file:text-sm file:font-medium file:text-foreground hover:file:bg-[#272729]"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-foreground/80 mb-2">
                            Wallet Name
                        </label>
                        <input
                            type="text"
                            placeholder="Imported Wallet"
                            value={keystoreForm.nickname}
                            onChange={(e) => setKeystoreForm({ ...keystoreForm, nickname: e.target.value })}
                            className="w-full bg-muted border border-border rounded-lg px-3 py-2 text-foreground"
                        />
                    </div>

                    <div className="rounded-lg border border-[#272729] bg-[#0f0f0f] p-4">
                        <div className="flex items-start gap-3">
                            <AlertTriangle className="mt-0.5 h-5 w-5 text-white/60" />
                            <div>
                                <h4 className="mb-1 font-medium text-foreground">Import Security Warning</h4>
                                <p className="text-sm text-muted-foreground">
                                    Only import wallets from trusted sources. Verify all information before proceeding.
                                </p>
                            </div>
                        </div>
                    </div>

                    <Button
                        onClick={handleImportKeystore}
                        className="h-11 w-full"
                    >
                        Import Keystore
                    </Button>
                </div>
            )}
        </>
    );

    if (embedded) {
        return <div className="w-full">{content}</div>;
    }

    return (
        <motion.div
            variants={panelVariants}
            className="bg-card rounded-2xl p-6 border border-border/80 w-full shadow-[0_14px_34px_rgba(0,0,0,0.2)]"
        >
            {content}
        </motion.div>
    );
};
