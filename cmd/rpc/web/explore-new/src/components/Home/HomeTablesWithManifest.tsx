import React from 'react';
import TableCard from './TableCard';
import { useValidators, useTransactions } from '../../hooks/useApi';

const HomeTablesWithManifest: React.FC = () => {
    // Fetch data
    const { data: validatorsData } = useValidators(1);
    const { data: transactionsData } = useTransactions(1, 5);

    // Transform validators data for home table
    const homeValidators = React.useMemo(() => {
        if (!validatorsData?.results) return [];
        
        return validatorsData.results.slice(0, 5).map((validator: any, index: number) => ({
            rank: index + 1,
            name: validator.name || `Validator ${index + 1}`,
            rewards: (validator.stakedAmount / 1000000000) * 0.1, // Simplified calculation
            change24h: validator.unstakingHeight ? 'Inactive' : 'Active',
            blocksProduced: Math.floor(Math.random() * 100), // Simulated
            totalWeight: validator.stakedAmount || 0,
            weightChange: validator.unstakingHeight ? -0.5 : Math.floor(Math.random() * 100) * 0.1
        }));
    }, [validatorsData]);

    // Transform transactions data for home table
    const homeTransactions = React.useMemo(() => {
        if (!transactionsData?.results) return [];
        
        return transactionsData.results.slice(0, 5).map((tx: any) => ({
            hash: tx.txHash || '',
            type: tx.messageType || 'unknown',
            from: tx.from || '',
            to: tx.to || '',
            amount: tx.amount || 0,
            age: tx.timestamp || ''
        }));
    }, [transactionsData]);

    return (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Validator Ranking Table */}
            <TableCard
                title="Validator Ranking"
                manifestKey="home-validators"
                data={homeValidators}
                loading={!validatorsData}
                paginate={false}
                viewAllPath="/validators"
                columns={[]}
                rows={[]}
            />

            {/* Recent Transactions Table */}
            <TableCard
                title="Recent Transactions"
                manifestKey="home-transactions"
                data={homeTransactions}
                loading={!transactionsData}
                paginate={false}
                viewAllPath="/transactions"
                columns={[]}
                rows={[]}
            />
        </div>
    );
};

export default HomeTablesWithManifest;
