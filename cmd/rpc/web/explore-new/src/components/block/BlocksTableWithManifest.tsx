import React from 'react';
import TableCard from '../Home/TableCard';
import { useBlocks } from '../../hooks/useApi';

interface BlocksTableWithManifestProps {
    currentPage?: number;
    onPageChange?: (page: number) => void;
    totalCount?: number;
    loading?: boolean;
}

const BlocksTableWithManifest: React.FC<BlocksTableWithManifestProps> = ({
    currentPage = 1,
    onPageChange,
    totalCount = 0,
    loading = false
}) => {
    // Fetch blocks data
    const { data: blocksData } = useBlocks(currentPage, 10);

    // Transform blocks data to match manifest structure
    const blocks = React.useMemo(() => {
        if (!blocksData?.results) return [];

        return blocksData.results.map((block: any) => ({
            height: block.blockHeader?.height || 0,
            timestamp: block.blockHeader?.time || '',
            age: block.blockHeader?.time || '',
            hash: block.blockHeader?.hash || '',
            producer: block.blockHeader?.proposerAddress || '',
            transactions: block.blockHeader?.totalTxs || block.blockHeader?.numTxs || 0,
            gasPrice: 0, // Not available in current API
            blockTime: 6 // Estimated block time
        }));
    }, [blocksData]);

    return (
        <TableCard
            title="Blocks"
            manifestKey="blocks"
            data={blocks}
            loading={loading}
            paginate={true}
            currentPage={currentPage}
            totalCount={totalCount}
            onPageChange={onPageChange}
            spacing={4}
            pageSize={10}
            columns={[]}
            rows={[]}
        />
    );
};

export default BlocksTableWithManifest;
