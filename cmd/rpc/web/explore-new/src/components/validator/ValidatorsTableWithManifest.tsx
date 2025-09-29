import React from 'react';
import TableCard from '../Home/TableCard';
import { useValidators } from '../../hooks/useApi';

interface ValidatorsTableWithManifestProps {
    currentPage?: number;
    onPageChange?: (page: number) => void;
    totalCount?: number;
    loading?: boolean;
}

const ValidatorsTableWithManifest: React.FC<ValidatorsTableWithManifestProps> = ({
    currentPage = 1,
    onPageChange,
    totalCount = 0,
    loading = false
}) => {
    // Fetch validators data
    const { data: validatorsData } = useValidators(1);

    // Transform validators data to match manifest structure
    const validators = React.useMemo(() => {
        if (!validatorsData?.results) return [];
        
        return validatorsData.results.map((validator: any, index: number) => ({
            rank: index + 1,
            name: validator.name || `Validator ${index + 1}`,
            address: validator.address || '',
            stakedAmount: validator.stakedAmount || 0,
            stakeWeight: (validator.stakedAmount / 1000000000) * 0.1, // Simplified calculation
            estimatedRewardRate: (validator.stakedAmount / 1000000000) * 0.05, // Simplified calculation
            activityScore: validator.unstakingHeight ? 'Unstaking' : 
                          validator.maxPausedHeight ? 'Paused' : 
                          validator.delegate ? 'Delegates' : 'Active',
            blocksProduced: Math.floor(Math.random() * 1000) // Simulated for now
        }));
    }, [validatorsData]);

    return (
        <TableCard
            title="Validators"
            manifestKey="validators"
            data={validators}
            loading={loading}
            paginate={true}
            currentPage={currentPage}
            totalCount={totalCount}
            onPageChange={onPageChange}
            pageSize={10}
            columns={[]}
            rows={[]}
        />
    );
};

export default ValidatorsTableWithManifest;
