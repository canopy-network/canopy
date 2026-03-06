import React from 'react';
import {motion} from 'framer-motion';
import {ValidatorCard} from './ValidatorCard';

interface Validator {
    address: string;
    nickname?: string;
    stakedAmount: number;
    status: 'Staked' | 'Paused' | 'Unstaking';
    rewards24h: number;
    chains?: string[];
    isSynced: boolean;
}

interface ValidatorListProps {
    validators: Validator[];
}

const itemVariants = {
    hidden: {opacity: 0, y: 20},
    visible: {opacity: 1, y: 0, transition: {duration: 0.4}}
};

export const ValidatorList: React.FC<ValidatorListProps> = ({ validators }) => {

    if (validators.length === 0) {
        return (
            <motion.div
                variants={itemVariants}
                className="bg-card rounded-xl p-12 border border-border/60"
            >
                <div className="text-center text-muted-foreground">
                    {'No validators found'}
                </div>
            </motion.div>
        );
    }

    return (
        <div className="space-y-4">
            {validators.map((validator, index) => (
                <ValidatorCard
                    key={validator.address}
                    validator={validator}
                    index={index}
                />
            ))}
        </div>
    );
};

