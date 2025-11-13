import { useQuery } from '@tanstack/react-query';
import { Validators as ValidatorsAPI } from '@/core/api';
import {useAccounts} from "@/app/providers/AccountsProvider";

interface Validator {
    address: string;
    publicKey: string;
    stakedAmount: number;
    unstakingAmount: number;
    unstakingHeight: number;
    pausedHeight: number;
    unstaking: boolean;
    paused: boolean;
    delegate: boolean;
    blocksProduced: number;
    rewards24h: number;
    stakeWeight: number;
    weightChange: number;
    nickname?: string;
}

async function fetchValidators(accounts: any[]): Promise<Validator[]> {
    try {
        // Get all validators from the network
        const allValidatorsResponse = await ValidatorsAPI(0);
        const allValidators = allValidatorsResponse.results || [];
        
        // Filter validators that belong to our accounts
        const accountAddresses = accounts.map(acc => acc.address);
        const ourValidators = allValidators.filter((validator: any) => 
            accountAddresses.includes(validator.address)
        );
        
        // Map to our interface
        const validators: Validator[] = ourValidators.map((validator: any) => {
            const account = accounts.find(acc => acc.address === validator.address);
            return {
                address: validator.address,
                publicKey: validator.publicKey || '',
                stakedAmount: validator.stakedAmount || 0,
                unstakingAmount: validator.unstakingAmount || 0,
                unstakingHeight: validator.unstakingHeight || 0,
                pausedHeight: validator.maxPausedHeight || 0,
                unstaking: validator.unstakingHeight > 0,
                paused: validator.maxPausedHeight > 0,
                delegate: validator.delegate || false,
                blocksProduced: 0, // This would need to be calculated separately
                rewards24h: 0, // This would need to be calculated separately
                stakeWeight: 0, // This would need to be calculated separately
                weightChange: 0, // This would need to be calculated separately
                nickname: account?.nickname
            };
        });
        
        return validators;
    } catch (error) {
        console.error('Error fetching validators:', error);
        return [];
    }
}

export function useValidators() {
    const { accounts, loading: accountsLoading } = useAccounts();

    return useQuery({
        queryKey: ['validators', accounts.map(acc => acc.address)],
        enabled: !accountsLoading && accounts.length > 0,
        queryFn: () => fetchValidators(accounts),
        staleTime: 10000,
        retry: 2,
        retryDelay: 1000,
    });
}

