import { useQuery } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';
import { TransactionsBySender, TransactionsByRec, FailedTransactions } from '@/core/api';

interface Transaction {
    hash: string;
    height: number;
    time: number;
    transaction: {
        type: string;
        from?: string;
        to?: string;
        amount?: number;
    };
    fee: number;
    memo?: string;
    status?: string;
}

interface TransactionResponse {
    results: Transaction[];
    total: number;
    pageNumber: number;
    perPage: number;
}

async function fetchTransactionsBySender(address: string): Promise<Transaction[]> {
    try {
        const data: TransactionResponse = await TransactionsBySender(1, address);
        return data.results || [];
    } catch (error) {
        console.error(`Error fetching transactions for address ${address}:`, error);
        return [];
    }
}

async function fetchTransactionsByReceiver(address: string): Promise<Transaction[]> {
    try {
        const data: TransactionResponse = await TransactionsByRec(1, address);
        return data.results || [];
    } catch (error) {
        console.error(`Error fetching received transactions for address ${address}:`, error);
        return [];
    }
}

async function fetchFailedTransactions(address: string): Promise<Transaction[]> {
    try {
        const data: TransactionResponse = await FailedTransactions(1, address);
        return data.results || [];
    } catch (error) {
        console.error(`Error fetching failed transactions for address ${address}:`, error);
        return [];
    }
}

export function useTransactions() {
    const { accounts, loading: accountsLoading } = useAccounts();

    return useQuery({
        queryKey: ['transactions', accounts.map(acc => acc.address)],
        enabled: !accountsLoading && accounts.length > 0,
        queryFn: async () => {
            if (accounts.length === 0) return [];

            try {
                // Fetch transactions for all accounts
                const allTransactions: Transaction[] = [];
                
                for (const account of accounts) {
                    const [sentTxs, receivedTxs, failedTxs] = await Promise.all([
                        fetchTransactionsBySender(account.address),
                        fetchTransactionsByReceiver(account.address),
                        fetchFailedTransactions(account.address)
                    ]);
                    
                    // Add status to transactions
                    sentTxs.forEach(tx => tx.status = 'included');
                    receivedTxs.forEach(tx => tx.status = 'included');
                    failedTxs.forEach(tx => tx.status = 'failed');
                    
                    allTransactions.push(...sentTxs, ...receivedTxs, ...failedTxs);
                }

                // Sort by time (most recent first) and remove duplicates
                const uniqueTransactions = allTransactions
                    .filter((tx, index, self) => 
                        index === self.findIndex(t => t.hash === tx.hash)
                    )
                    .sort((a, b) => b.time - a.time)
                    .slice(0, 10); // Get latest 10 transactions

                return uniqueTransactions;
            } catch (error) {
                console.error('Error fetching transactions:', error);
                return [];
            }
        },
        staleTime: 10000,
        retry: 2,
        retryDelay: 1000,
    });
}
