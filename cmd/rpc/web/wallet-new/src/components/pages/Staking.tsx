import React, { useEffect, useRef, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { useStakingData } from '@/hooks/useStakingData';
import { useValidators } from '@/hooks/useValidators';
import { useAccountData } from '@/hooks/useAccountData';
import { useMultipleBlockProducerData } from '@/hooks/useBlockProducerData';
import { Validators as ValidatorsAPI } from '@/core/api';
import { Button } from '@/components/ui/Button';
import { PauseUnpauseModal } from '@/components/ui/PauseUnpauseModal';
import { SendModal } from '@/components/ui/SendModal';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    Filler
} from 'chart.js';
import { Line } from 'react-chartjs-2';

ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    Filler
);

type ValidatorRow = {
    address: string;
    nickname?: string;
    stakedAmount: number;
    status: 'Staked' | 'Paused' | 'Unstaking';
    rewards24h: number;
};

const formatAmount = (amount: number) => {
    return (amount / 1_000_000).toLocaleString(undefined, { maximumFractionDigits: 2 });
};

const truncateAddress = (address: string) => `${address.substring(0, 4)}…${address.substring(address.length - 4)}`;

const chainLabels = ['DEX', 'CAN'];

export default function Staking(): JSX.Element {
    const { getText } = useManifest();
    const { data: staking = { totalStaked: 0, totalRewards: 0, chartData: [] } as any } = useStakingData();
    const { totalStaked, stakingData } = useAccountData();
    const { data: validators = [] } = useValidators();
    const csvRef = useRef<HTMLAnchorElement>(null);

    const [addStakeOpen, setAddStakeOpen] = useState(false);
    const [pauseModal, setPauseModal] = useState<{ isOpen: boolean; action: 'pause' | 'unpause'; address: string; nickname?: string }>(
        { isOpen: false, action: 'pause', address: '' }
    );
    const [searchTerm, setSearchTerm] = useState('');
    const [isActive, setIsActive] = useState(true);
    const [chainCount, setChainCount] = useState<number>(0);
    
    // Get validator addresses for block producer data
    const validatorAddresses = validators.map(v => v.address);
    const { data: blockProducerData = {} } = useMultipleBlockProducerData(validatorAddresses);

    // Fetch committees count (union across our validators) directly from API
    useEffect(() => {
        const run = async () => {
            try {
                const all = await ValidatorsAPI(0);
                const ourAddresses = new Set(validators.map(v => v.address));
                const committees = new Set<number>();
                (all.results || []).forEach((v: any) => {
                    if (ourAddresses.has(v.address) && Array.isArray(v.committees)) {
                        v.committees.forEach((c: number) => committees.add(c));
                    }
                });
                setChainCount(committees.size);
            } catch {
                setChainCount(0);
            }
        };
        if (validators.length > 0) run();
    }, [validators]);

    const rows: ValidatorRow[] = useMemo(() => {
            return validators.map((v: any) => ({
                address: v.address,
                nickname: v.nickname,
                publicKey: v.publicKey || '',
                stakedAmount: v.stakedAmount || 0,
                status: v.unstaking ? 'Unstaking' : v.paused ? 'Paused' : 'Staked',
                rewards24h: blockProducerData[v.address]?.rewards24h || 0,
                chains: v.committees?.map((id: number) => chainLabels[id % chainLabels.length]) || [],
                isSynced: !(v.paused),
            }));
    }, [validators, blockProducerData]);

    const filtered = rows.filter(r =>
        (r.nickname || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        r.address.toLowerCase().includes(searchTerm.toLowerCase())
    );

    // Chart data configuration
    const rewardsChartData = {
        labels: ['', '', '', '', '', ''],
        datasets: [{
            data: [3, 4, 3.5, 5, 4, 4.5],
            borderColor: '#6fe3b4',
            backgroundColor: 'transparent',
            borderWidth: 1.5,
            tension: 0.4,
            pointRadius: 0
        }]
    };

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { display: false }, tooltip: { enabled: false } },
        scales: { x: { display: false }, y: { display: false } },
    };

    const containerVariants = {
        hidden: { opacity: 0 },
        visible: {
            opacity: 1,
            transition: { duration: 0.6, staggerChildren: 0.1 }
        }
    };

    const itemVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
    };

    // Export to CSV functionality
    const prepareCSVData = () => {
        const header = ['address', 'nickname', 'stakedAmount', 'rewards24h', 'status'];
        const lines = [header.join(',')]
            .concat(
                filtered.map(r => [
                    r.address, 
                    r.nickname || '', 
                    r.stakedAmount, 
                    r.rewards24h, 
                    r.status
                ].join(','))
            );
        return lines.join('\n');
    };

    const exportCSV = () => {
        const csvContent = prepareCSVData();
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const url = URL.createObjectURL(blob);
        
        if (csvRef.current) {
            csvRef.current.href = url;
            csvRef.current.download = 'validators.csv';
            csvRef.current.click();
        }
        
        setTimeout(() => URL.revokeObjectURL(url), 100);
    };
    
    // Format the amount to match the UI (75,234.56)
    const formatStakedAmount = (amount: number) => {
        if (!amount && amount !== 0) return '0.00';
        return (amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    };

    // Format the rewards to match the UI (+678.90)
    const formatRewards = (amount: number) => {
        if (!amount && amount !== 0) return '+0.00';
        return `+${(amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
    };

    return (
        <motion.div
            className="min-h-screen bg-bg-primary"
            initial="hidden"
            animate="visible" 
            variants={containerVariants}
        >
            {/* Hidden link for CSV export */}
            <a ref={csvRef} style={{ display: 'none' }} />
            
            <div className="px-6 py-8">
                {/* Top stats */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                    {/* Total Staked Card */}
                    <motion.div 
                        variants={itemVariants}
                        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden"
                    >
                        <div className="absolute top-4 right-4">
                            <i className="fa-solid fa-coins text-primary text-2xl"></i>
                        </div>
                        <h3 className="text-text-secondary text-sm font-medium mb-4">
                            {getText('ui.staking.totalStaked', 'Total Staked')}
                        </h3>
                        <p className="text-white text-2xl font-bold mb-1">
                            {formatStakedAmount(totalStaked)} CNPY
                        </p>
                        <p className="text-text-muted text-xs">
                            {getText('ui.staking.acrossValidators', 'Across')} {validators.length} {getText('ui.staking.validators', 'validators')}
                        </p>
                    </motion.div>
                    
                    {/* Rewards Earned Card */}
                    <motion.div 
                        variants={itemVariants} 
                        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden"
                    >
                        <div className="absolute top-4 right-4">
                            <button className="text-text-muted">
                                <i className="fa-solid fa-ellipsis text-xl"></i>
                            </button>
                        </div>
                        <h3 className="text-text-secondary text-sm font-medium mb-4">
                            {getText('ui.staking.rewardsEarned', 'Rewards Earned')}
                        </h3>
                        <p className="text-primary text-2xl font-bold mb-1">
                            {formatRewards(staking.totalRewards || 0)} CNPY
                        </p>
                        <p className="text-text-muted text-xs">
                            {getText('ui.staking.last24h', 'Last 24 hours')}
                        </p>
                    </motion.div>
                    
                    {/* Active Validators Card */}
                    <motion.div 
                        variants={itemVariants}
                        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden"
                    >
                        <div className="absolute top-4 right-4">
                            <i className="fa-solid fa-shield text-text-secondary text-2xl"></i>
                        </div>
                        <h3 className="text-text-secondary text-sm font-medium mb-4">
                            {getText('ui.staking.activeValidators', 'Active Validators')}
                        </h3>
                        <p className="text-white text-2xl font-bold mb-1">
                            {validators.length}
                        </p>
                        <p className="text-text-muted text-xs flex items-center gap-1">
                            <span className="inline-block w-2 h-2 bg-green-500 rounded-full"></span>
                            {getText('ui.staking.allOnline', 'All online')}
                        </p>
                    </motion.div>
                    
                    {/* Chains Staked Card */}
                    <motion.div 
                        variants={itemVariants}
                        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent relative overflow-hidden"
                    >
                        <div className="absolute top-4 right-4">
                            <i className="fa-solid fa-link text-text-secondary text-2xl"></i>
                        </div>
                        <h3 className="text-text-secondary text-sm font-medium mb-4">
                            {getText('ui.staking.chainsStaked', 'Chains Staked')}
                        </h3>
                        <p className="text-white text-2xl font-bold mb-1">
                            {chainCount || 0}
                        </p>
                        <div className="flex items-center gap-1 mt-1">
                            <span className="w-4 h-4 rounded-full bg-pink-500"></span>
                            <span className="w-4 h-4 rounded-full bg-orange-500"></span>
                            <span className="w-4 h-4 rounded-full bg-blue-500"></span>
                            <span className="text-text-muted text-xs">+3 more</span>
                        </div>
                    </motion.div>
                </div>

                {/* Toolbar */}
                <motion.div 
                    variants={itemVariants}
                    className="mb-6 flex flex-col md:flex-row items-stretch gap-3 md:items-center md:justify-between"
                >
                    <div className="flex items-center gap-3">
                        <h2 className="text-xl font-bold text-white flex items-center gap-2">
                            {getText('ui.staking.allValidators', 'All Validators')}
                            <span className="bg-green-500/20 text-green-400 text-xs px-2 py-0.5 rounded-full">
                                {validators.filter(v => !v.paused).length} active
                            </span>
                        </h2>
                    </div>
                    <div className="flex items-center gap-2">
                        <div className="relative md:w-64">
                            <input
                                type="text"
                                placeholder={getText('ui.staking.search', 'Search validators...')}
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                                className="w-full bg-bg-secondary border border-bg-accent rounded-lg pl-10 pr-4 py-2 text-white placeholder-text-muted focus:outline-none focus:ring-2 focus:ring-primary/50"
                            />
                            <i className="fa-solid fa-search absolute left-3 top-1/2 transform -translate-y-1/2 text-text-muted"></i>
                        </div>
                        <button
                            onClick={() => setAddStakeOpen(true)}
                            className="flex items-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium"
                        >
                            <i className="fa-solid fa-plus"></i>
                            {getText('ui.staking.addStake', 'Add Stake')}
                        </button>
                        <button
                            onClick={exportCSV}
                            className="flex items-center gap-2 px-3 py-2 bg-bg-secondary hover:bg-bg-accent/50 text-white rounded-lg text-sm font-medium border border-bg-accent"
                        >
                            <i className="fa-solid fa-download"></i>
                            {getText('ui.staking.exportCSV', 'Export CSV')}
                        </button>
                        <button className="p-2 hover:bg-bg-accent/50 rounded-lg">
                            <i className="fa-solid fa-filter text-text-muted"></i>
                        </button>
                    </div>
                </motion.div>

                {/* Validator Cards */}
                <div className="space-y-4">
                    {filtered.map((validator, index) => (
                        <motion.div 
                            key={validator.address}
                            variants={itemVariants}
                            className="bg-bg-secondary rounded-xl border border-bg-accent relative overflow-hidden"
                        >
                            <div className="p-4 flex flex-col lg:flex-row gap-6">
                                {/* Left side - Validator identity */}
                                <div className="flex items-start gap-3">
                                    <div className="flex flex-col items-start">
                                        <div className="text-text-primary font-medium mb-1 flex items-center">
                                            <span className="mr-2">{validator.nickname || `Node ${index + 1}`}</span>
                                            <button className="text-bg-accent">
                                                <i className="fa-solid fa-server text-primary text-xs"></i>
                                            </button>
                                        </div>
                                        <div className="text-text-muted text-sm font-mono">
                                            {truncateAddress(validator.address)}
                                        </div>
                                        <div className="flex items-center mt-1">
                                            <button className="text-primary text-xs" onClick={() => navigator.clipboard.writeText(validator.address)}>
                                                Copy
                                            </button>
                                        </div>
                                        
                                        {/* Chain badges */}
                                        <div className="flex mt-2 gap-1">
                                            {(validator.chains || []).slice(0, 2).map((chain, i) => (
                                                <span key={i} className="px-2 py-0.5 text-xs bg-bg-accent text-white rounded">{chain}</span>
                                            ))}
                                            {(validator.chains || []).length > 2 && (
                                                <span className="text-text-muted text-xs">+{(validator.chains || []).length - 2} more</span>
                                            )}
                                        </div>
                                    </div>
                                </div>
                                
                                {/* Spacer */}
                                <div className="flex-1"></div>
                                
                                {/* Right side - Stats */}
                                <div className="flex items-center justify-between flex-1">
                                    <div className="flex flex-col items-end">
                                        <div className="text-text-primary font-medium text-right">
                                            {formatStakedAmount(validator.stakedAmount)} CNPY
                                        </div>
                                        <div className="text-text-muted text-xs text-right">
                                            {getText('ui.staking.totalStaked', 'Total Staked')}
                                        </div>
                                    </div>
                                    
                                    <div className="flex flex-col items-end mx-4">
                                        <div className="text-primary font-medium text-right">
                                            {formatRewards(validator.rewards24h)}
                                        </div>
                                        <div className="text-text-muted text-xs text-right">
                                            {getText('ui.staking.rewards24h', '24h Rewards')}
                                        </div>
                                    </div>
                                    
                                    <div className="w-36 h-12 mx-6">
                                        <Line data={rewardsChartData} options={chartOptions} />
                                    </div>
                                    
                                    <div className="flex items-center gap-2">
                                        <span className={`${
                                            validator.status === 'Staked' 
                                            ? 'bg-green-500/20 text-green-400' 
                                            : validator.status === 'Paused'
                                            ? 'bg-yellow-500/20 text-yellow-400'
                                            : 'bg-red-500/20 text-red-400'
                                        } text-xs px-2 py-0.5 rounded-full`}>
                                            {validator.status}
                                        </span>
                                        <span className={`w-2 h-2 ${validator.isSynced ? 'bg-green-500' : 'bg-red-500'} rounded-full`}></span>
                                    </div>
                                    
                                    <div className="flex items-center">
                                        <button 
                                            className="p-2 hover:bg-bg-accent rounded-lg transition-colors mx-1"
                                            onClick={() => setPauseModal({ 
                                                isOpen: true, 
                                                action: validator.status === 'Staked' ? 'pause' : 'unpause', 
                                                address: validator.address, 
                                                nickname: validator.nickname 
                                            })}
                                        >
                                            <i className="fa-solid fa-pause text-text-muted"></i>
                                        </button>
                                        <button className="p-2 hover:bg-bg-accent rounded-lg transition-colors">
                                            <i className="fa-solid fa-ellipsis text-text-muted"></i>
                                        </button>
                                    </div>
                                </div>
                            </div>
                        </motion.div>
                    ))}

                    {filtered.length === 0 && (
                        <motion.div 
                            variants={itemVariants}
                            className="bg-bg-secondary rounded-xl p-12 border border-bg-accent"
                        >
                            <div className="text-center text-text-muted">{getText('ui.staking.noValidators', 'No validators found')}</div>
                        </motion.div>
                    )}
                </div>
            </div>

            {/* Modals */}
            <SendModal isOpen={addStakeOpen} onClose={() => setAddStakeOpen(false)} defaultTab="stake" />
            <PauseUnpauseModal
                isOpen={pauseModal.isOpen}
                onClose={() => setPauseModal({ isOpen: false, action: 'pause', address: '' })}
                validatorAddress={pauseModal.address}
                validatorNickname={pauseModal.nickname}
                action={pauseModal.action}
            />
        </motion.div>
    );
}


