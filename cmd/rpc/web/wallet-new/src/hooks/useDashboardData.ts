import { useState, useEffect } from 'react';

export interface Account {
    address: string;
    balance: string;
    nickname?: string;
    status?: 'staked' | 'unstaking' | 'liquid' | 'delegated';
}

export interface Transaction {
    hash: string;
    time: string;
    action: 'send' | 'receive' | 'stake' | 'swap';
    amount: string;
    status: 'confirmed' | 'pending' | 'open' | 'failed';
    from?: string;
    to?: string;
}

export interface Node {
    address: string;
    stakeAmount: string;
    status: 'staked' | 'unstaking' | 'paused';
    blocksProduced: number;
    rewards24h: string;
    stakeWeight: string;
    weightChange: string;
}

export interface DashboardData {
    totalBalance: string;
    stakedBalance: string;
    balanceChange24h: string;
    accounts: Account[];
    recentTransactions: Transaction[];
    nodes: Node[];
    loading: boolean;
    error: string | null;
}

const API_BASE = 'http://localhost:50002';

export const useDashboardData = () => {
    const [data, setData] = useState<DashboardData>({
        totalBalance: '0',
        stakedBalance: '0',
        balanceChange24h: '0',
        accounts: [],
        recentTransactions: [],
        nodes: [],
        loading: true,
        error: null
    });

    const fetchAccounts = async (): Promise<Account[]> => {
        try {
            const response = await fetch(`${API_BASE}/v1/query/accounts`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ height: 0, address: '' })
            });

            if (!response.ok) throw new Error('Failed to fetch accounts');

            const result = await response.json();
            return result.accounts || [];
        } catch (error) {
            console.error('Error fetching accounts:', error);
            return [];
        }
    };

    const fetchAccountBalance = async (address: string): Promise<string> => {
        try {
            const response = await fetch(`${API_BASE}/v1/query/account`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ height: 0, address })
            });

            if (!response.ok) throw new Error('Failed to fetch account balance');

            const result = await response.json();
            return result.balance || '0';
        } catch (error) {
            console.error('Error fetching account balance:', error);
            return '0';
        }
    };

    const fetchValidators = async (): Promise<Node[]> => {
        try {
            const response = await fetch(`${API_BASE}/v1/query/validators`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ height: 0, address: '' })
            });

            if (!response.ok) throw new Error('Failed to fetch validators');

            const result = await response.json();
            return result.validators || [];
        } catch (error) {
            console.error('Error fetching validators:', error);
            return [];
        }
    };

    const fetchRecentTransactions = async (): Promise<Transaction[]> => {
        try {
            return [];
        } catch (error) {
            console.error('Error fetching transactions:', error);
            return [];
        }
    };

    const loadDashboardData = async () => {
        try {
            setData(prev => ({ ...prev, loading: true, error: null }));

            const [accounts, validators, transactions] = await Promise.all([
                fetchAccounts(),
                fetchValidators(),
                fetchRecentTransactions()
            ]);

            // Calcular balance total
            let totalBalance = 0;
            let stakedBalance = 0;

            for (const account of accounts) {
                const balance = parseFloat(account.balance) || 0;
                totalBalance += balance;

                if (account.status === 'staked' || account.status === 'delegated') {
                    stakedBalance += balance;
                }
            }

            setData({
                totalBalance: totalBalance.toFixed(2),
                stakedBalance: stakedBalance.toFixed(2),
                balanceChange24h: '+2.4', // Simulado
                accounts,
                recentTransactions: transactions,
                nodes: validators,
                loading: false,
                error: null
            });

        } catch (error) {
            setData(prev => ({
                ...prev,
                loading: false,
                error: error instanceof Error ? error.message : 'Failed to load dashboard data'
            }));
        }
    };

    useEffect(() => {
        loadDashboardData();

        // Refrescar datos cada 30 segundos
        const interval = setInterval(loadDashboardData, 30000);
        return () => clearInterval(interval);
    }, []);

    return {
        ...data,
        refetch: loadDashboardData
    };
};
