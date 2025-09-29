import React from 'react';
import TableCard from '../Home/TableCard';
import { useTransactions } from '../../hooks/useApi';

interface TransactionsTableWithManifestProps {
    currentPage?: number;
    onPageChange?: (page: number) => void;
    totalCount?: number;
    loading?: boolean;
}

const TransactionsTableWithManifest: React.FC<TransactionsTableWithManifestProps> = ({
    currentPage = 1,
    onPageChange,
    totalCount = 0,
    loading = false
}) => {
    // Fetch transactions data
    const { data: transactionsData } = useTransactions(currentPage, 10);

    // Transform transactions data to match manifest structure
    const transactions = React.useMemo(() => {
        if (!transactionsData?.results) return [];
        
        return transactionsData.results.map((tx: any) => ({
            hash: tx.txHash || '',
            type: tx.messageType || 'unknown',
            from: tx.from || '',
            to: tx.to || '',
            amount: tx.amount || 0,
            fee: tx.fee || 0,
            block: tx.blockHeight || 0,
            age: tx.timestamp || ''
        }));
    }, [transactionsData]);

    return (
        <TableCard
            title="Transactions"
            manifestKey="transactions"
            data={transactions}
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

export default TransactionsTableWithManifest;
