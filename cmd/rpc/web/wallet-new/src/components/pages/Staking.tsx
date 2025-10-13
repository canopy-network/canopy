import React, { useEffect, useRef, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { useManifest } from '@/hooks/useManifest';
import { useStakingData } from '@/hooks/useStakingData';
import { useValidators } from '@/hooks/useValidators';
import { useAccountData } from '@/hooks/useAccountData';
import { useMultipleBlockProducerData } from '@/hooks/useBlockProducerData';
import { Validators as ValidatorsAPI } from '@/core/api';
import { PauseUnpauseModal } from '@/components/ui/PauseUnpauseModal';
import { SendModal } from '@/components/ui/SendModal';
import { StatsCards } from './staking/StatsCards';
import { Toolbar } from './staking/Toolbar';
import { ValidatorList } from './staking/ValidatorList';

type ValidatorRow = {
    address: string;
    nickname?: string;
    stakedAmount: number;
    status: 'Staked' | 'Paused' | 'Unstaking';
    rewards24h: number;
    chains?: string[];
    isSynced: boolean;
    // Additional validator information
    committees?: number[];
    compound?: boolean;
    delegate?: boolean;
    maxPausedHeight?: number;
    netAddress?: string;
    output?: string;
    publicKey?: string;
    unstakingHeight?: number;
};

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
                stakedAmount: v.stakedAmount || 0,
                status: v.unstaking ? 'Unstaking' : v.paused ? 'Paused' : 'Staked',
                rewards24h: blockProducerData[v.address]?.rewards24h || 0,
                chains: v.committees?.map((id: number) => chainLabels[id % chainLabels.length]) || [],
                isSynced: !(v.paused),
            // Additional validator information
            committees: v.committees,
            compound: v.compound,
            delegate: v.delegate,
            maxPausedHeight: v.maxPausedHeight,
            netAddress: v.netAddress,
            output: v.output,
            publicKey: v.publicKey,
            unstakingHeight: v.unstakingHeight,
            }));
    }, [validators, blockProducerData]);

    const filtered = rows.filter(r =>
        (r.nickname || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        r.address.toLowerCase().includes(searchTerm.toLowerCase())
    );


    const containerVariants = {
        hidden: { opacity: 0 },
        visible: {
            opacity: 1,
            transition: { duration: 0.6, staggerChildren: 0.1 }
        }
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
    
    const handlePauseUnpause = (address: string, nickname?: string, action?: 'pause' | 'unpause') => {
        setPauseModal({
            isOpen: true,
            action: action || 'pause',
            address,
            nickname
        });
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
                <StatsCards
                    totalStaked={totalStaked}
                    totalRewards={staking.totalRewards || 0}
                    validatorsCount={validators.length}
                    chainCount={chainCount}
                    activeValidatorsCount={validators.filter(v => !v.paused).length}
                />
                <div className="flex flex-col bg-bg-secondary rounded-xl p-6 border border-bg-accent">
                {/* Toolbar */}
                    <Toolbar
                        searchTerm={searchTerm}
                        onSearchChange={setSearchTerm}
                        onAddStake={() => setAddStakeOpen(true)}
                        onExportCSV={exportCSV}
                        activeValidatorsCount={validators.filter(v => !v.paused).length}
                    />

                    {/* Validator List */}
                    <ValidatorList
                        validators={filtered}
                        onPauseUnpause={handlePauseUnpause}
                    />
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


