import { useQuery, useQueryClient } from '@tanstack/react-query';
import React from 'react';
import {
    Blocks,
    Transactions,
    AllTransactions,
    getTransactionsWithRealPagination,
    getRecentTransactionsPreview,
    Accounts,
    Validators,
    ValidatorsWithFilters,
    Committee,
    DAO,
    Account,
    AccountWithTxs,
    Params,
    Supply,
    Validator,
    BlockByHeight,
    BlockByHash,
    TxByHash,
    TransactionsBySender,
    TransactionsByRec,
    Pending,
    EcoParams,
    Orders,
    Config,
    getModalData,
    getCardData,
    getTableData,
    Order,
    DexBatch,
    NextDexBatch,
    rpcURL
} from '../lib/api';

const REFRESH_INTERVAL_MS = 20000; // 20 seconds

// Query Keys
export const queryKeys = {
    blocks: (page: number, perPage?: number, filter?: string) => ['blocks', page, perPage, filter],
    transactions: (page: number, height: number) => ['transactions', page, height],
    allTransactions: (page: number, perPage: number, filters?: any) => ['allTransactions', page, perPage, filters],
    realPaginationTransactions: (page: number, perPage: number, filters?: any) => ['realPaginationTransactions', page, perPage, filters],
    accounts: (page: number, perPage?: number) => ['accounts', page, perPage],
    validators: (page: number, perPage?: number) => ['validators', page, perPage],
    validatorsWithFilters: (page: number, unstaking: number, paused: number, delegate: number, committee: number) => ['validatorsWithFilters', page, unstaking, paused, delegate, committee],
    committee: (page: number, chainId: number) => ['committee', page, chainId],
    dao: (height: number) => ['dao', height],
    account: (height: number, address: string) => ['account', height, address],
    accountWithTxs: (height: number, address: string, page: number, perPage: number = 10) => ['accountWithTxs', height, address, page, perPage],
    params: (height: number) => ['params', height],
    supply: (height: number) => ['supply', height],
    validator: (height: number, address: string) => ['validator', height, address],
    blockByHeight: (height: number) => ['blockByHeight', height],
    blockByHash: (hash: string) => ['blockByHash', hash],
    txByHash: (hash: string) => ['txByHash', hash],
    transactionsBySender: (page: number, sender: string) => ['transactionsBySender', page, sender],
    transactionsByRec: (page: number, rec: string) => ['transactionsByRec', page, rec],
    pending: (page: number, perPage: number) => ['pending', page, perPage],
    ecoParams: (chainId: number) => ['ecoParams', chainId],
    orders: (chainId: number) => ['orders', chainId],
    config: () => ['config'],
    modalData: (query: string | number, page: number) => ['modalData', query, page],
    cardData: () => ['cardData'],
    tableData: (page: number, category: number, committee?: number) => ['tableData', page, category, committee],
};

// Block polling interval (ms). Defaults to 2s so a plain explorer/wallet —
// especially a simple private deployment — doesn't poll the node aggressively.
// Override with VITE_BLOCKS_POLL_MS in the environment for snappier updates.
const DEFAULT_BLOCKS_POLL_MS = 2000;
const MIN_BLOCKS_POLL_MS = 500;
const BLOCKS_POLL_MS = (() => {
    const raw = Number(import.meta.env.VITE_BLOCKS_POLL_MS);
    return Number.isFinite(raw) && raw > 0
        ? Math.max(MIN_BLOCKS_POLL_MS, raw)
        : DEFAULT_BLOCKS_POLL_MS;
})();

// Backoff helper: when a polling query keeps failing, slow it down so we
// don't hammer a degraded RPC node (and so we don't flood the browser
// console with one [Error] line per request — those are emitted natively
// by the browser for every 4xx/5xx and can't be silenced from JS).
const errorAwareInterval = (baseMs: number, maxMs: number = 60000) =>
    (query: { state: { error: unknown; fetchFailureCount: number } }) => {
        const fails = query.state.fetchFailureCount || 0;
        if (!query.state.error || fails <= 0) return baseMs;
        return Math.min(baseMs * Math.pow(2, fails), maxMs);
    };

// Lightweight hook for the latest block height (polls every BLOCKS_POLL_MS,
// with backoff on errors so a degraded RPC node doesn't spam the console).
export const useLatestBlock = () => {
    return useQuery({
        queryKey: ['latestBlock'],
        queryFn: () => Blocks(1, 0),
        staleTime: 0,
        refetchInterval: errorAwareInterval(BLOCKS_POLL_MS),
        refetchIntervalInBackground: true,
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
        refetchOnMount: 'always',
        retry: false,
    });
};

// Extracts the height of the most recent block from a Blocks(1,0) response.
const extractLatestBlockHeight = (payload: any): number => {
    if (!payload) return 0;
    const totalCount = payload?.totalCount ?? payload?.count;
    if (typeof totalCount === 'number' && totalCount > 0) return totalCount;
    const list = payload?.results || payload?.blocks || payload?.list || payload?.data;
    const first = Array.isArray(list) && list.length > 0 ? list[0] : null;
    return Number(first?.blockHeader?.height ?? first?.height ?? 0) || 0;
};

// Queries that must refresh whenever a new block is produced. Adding a key here
// makes that query participate in the global "new block" invalidation pulse.
const BLOCK_DEPENDENT_QUERY_KEYS: Array<readonly unknown[]> = [
    ['cardData'],
    ['allBlocksCache'],
    ['blocks'],
    ['recentTransactionsPreview'],
    ['realPaginationTransactions'],
    ['allTransactions'],
    ['transactions'],
    ['txsByHeight'],
    ['pending'],
    ['all-validators'],
    ['all-delegators'],
    ['validators'],
    ['validatorsWithFilters'],
    ['supply'],
    ['orders'],
    ['dexBatch'],
    ['nextDexBatch'],
];

// Global subscription: polls the latest block height and, whenever it changes,
// invalidates every dashboard-facing query so the UI reflects the new chain
// state without requiring a manual page refresh.
export const useBlockSubscription = () => {
    const queryClient = useQueryClient();
    const { data } = useLatestBlock();
    const lastHeightRef = React.useRef<number>(0);

    React.useEffect(() => {
        const height = extractLatestBlockHeight(data);
        if (height <= 0) return;
        if (lastHeightRef.current === height) return;

        const previousHeight = lastHeightRef.current;
        lastHeightRef.current = height;

        // Skip invalidation on the very first observation; queries are already
        // fetching their initial data and we want to avoid a redundant burst.
        if (previousHeight === 0) return;

        BLOCK_DEPENDENT_QUERY_KEYS.forEach((key) => {
            queryClient.invalidateQueries({ queryKey: key });
        });
    }, [data, queryClient]);
};

// Hooks for Blocks
export const useBlocks = (page: number, perPage: number = 10, filter: string = 'all') => {
    const blockCount = filter === 'week' ? 50 : filter === '24h' ? 30 : perPage;

    return useQuery({
        queryKey: queryKeys.blocks(page, blockCount, filter),
        queryFn: () => Blocks(page, blockCount),
        staleTime: 0,
        refetchInterval: BLOCKS_POLL_MS,
        refetchOnWindowFocus: true,
        refetchOnMount: 'always',
    });
};

// Hooks for Transactions
export const useTransactions = (page: number, height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.transactions(page, height),
        queryFn: () => Transactions(page, height),
        staleTime: 300000, // Cache for 5 minutes (increased from 30 seconds)
        refetchInterval: REFRESH_INTERVAL_MS,
        refetchOnWindowFocus: false, // Don't refetch when window regains focus
        gcTime: 600000 // Keep in cache for 10 minutes
    });
};

// Hook for transactions at a specific block height with configurable perPage
export const useTransactionsByHeight = (height: number, perPage: number = 1000, enabled: boolean = true) => {
    return useQuery({
        queryKey: ['txsByHeight', height, perPage],
        queryFn: () => Transactions(1, height, perPage),
        staleTime: 30000,
        enabled,
    });
};

// Hook for all transactions with filters
export const useAllTransactions = (page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    fromBlock?: string;
    toBlock?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}) => {
    return useQuery({
        queryKey: queryKeys.allTransactions(page, perPage, filters),
        queryFn: () => AllTransactions(page, perPage, filters),
        staleTime: 30000,
        enabled: true,
    });
};

// Hook for transactions with real pagination (recommended)
export const useTransactionsWithRealPagination = (page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    fromBlock?: string;
    toBlock?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}) => {
    return useQuery({
        queryKey: queryKeys.realPaginationTransactions(page, perPage, filters),
        queryFn: () => getTransactionsWithRealPagination(page, perPage, filters),
        staleTime: 30000,
        enabled: true,
    });
};

export const useRecentTransactionsPreview = (blocks: any[] | undefined, limit: number = 5) => {
    const latestBlockHeight = Number(blocks?.[0]?.blockHeader?.height ?? blocks?.[0]?.height ?? 0);

    return useQuery({
        queryKey: ['recentTransactionsPreview', latestBlockHeight, limit],
        queryFn: () => getRecentTransactionsPreview(limit, blocks),
        staleTime: 5000,
        refetchInterval: errorAwareInterval(10000),
        refetchOnWindowFocus: true,
        refetchOnMount: 'always',
        enabled: Array.isArray(blocks) && blocks.length > 0,
        placeholderData: (previousData) => previousData,
        retry: false,
    });
};

// Hooks for Accounts
export const useAccounts = (page: number, perPage: number = 10) => {
    return useQuery({
        queryKey: queryKeys.accounts(page, perPage),
        queryFn: () => Accounts(page, 0, perPage),
        staleTime: 30000,
    });
};

// Hooks for Validators
export const useValidators = (page: number, perPage: number = 10) => {
    return useQuery({
        queryKey: queryKeys.validators(page, perPage),
        queryFn: () => Validators(page, 0, perPage),
        staleTime: 30000,
    });
};

// Hook to get all validators at once
export const useAllValidators = () => {
    return useQuery({
        queryKey: ['all-validators'],
        queryFn: async () => {
            // Get all pages of validators
            const allValidators = []
            let page = 1
            let hasMore = true

            while (hasMore) {
                const response = await Validators(page, 0)
                const validators = response.results || response.validators || response.list || response.data || response

                if (Array.isArray(validators) && validators.length > 0) {
                    allValidators.push(...validators)
                    page++

                    // Check if we have more pages
                    const totalPages = response.totalPages || Math.ceil((response.totalCount || 0) / 10)
                    hasMore = page <= totalPages
                } else {
                    hasMore = false
                }
            }

            return {
                results: allValidators,
                totalCount: allValidators.length,
                totalPages: Math.ceil(allValidators.length / 10)
            }
        },
        staleTime: 300000, // Cache for 5 minutes (increased from 30 seconds)
        refetchInterval: REFRESH_INTERVAL_MS,
        refetchOnWindowFocus: false, // Don't refetch when window regains focus
        gcTime: 600000 // Keep in cache for 10 minutes
    });
};

// Hook to get all delegators at once (using delegate filter = 1)
export const useAllDelegators = () => {
    return useQuery({
        queryKey: ['all-delegators'],
        queryFn: async () => {
            // Get all pages of delegators with delegate filter = 1 (MustBe)
            const allDelegators = []
            let page = 1
            let hasMore = true

            while (hasMore) {
                const response = await ValidatorsWithFilters(page, 0, 0, 1, 0) // delegate: 1 = MustBe
                const delegators = response.results || response.validators || response.list || response.data || response

                if (Array.isArray(delegators) && delegators.length > 0) {
                    allDelegators.push(...delegators)
                    page++

                    // Check if we have more pages
                    const totalPages = response.totalPages || Math.ceil((response.totalCount || 0) / 10)
                    hasMore = page <= totalPages
                } else {
                    hasMore = false
                }
            }

            return {
                results: allDelegators,
                totalCount: allDelegators.length,
                totalPages: Math.ceil(allDelegators.length / 10)
            }
        },
        staleTime: 300000, // Cache for 5 minutes
        refetchInterval: REFRESH_INTERVAL_MS,
        refetchOnWindowFocus: false, // Don't refetch when window regains focus
        gcTime: 600000 // Keep in cache for 10 minutes
    });
};

// Hook to get validators with server-side filtering
export const useValidatorsWithFilters = (page: number, unstaking: number = 0, paused: number = 0, delegate: number = 0, committee: number = 0) => {
    return useQuery({
        queryKey: queryKeys.validatorsWithFilters(page, unstaking, paused, delegate, committee),
        queryFn: () => ValidatorsWithFilters(page, unstaking, paused, delegate, committee),
        staleTime: 30000,
    });
};

// Hooks for Committee
export const useCommittee = (page: number, chainId: number) => {
    return useQuery({
        queryKey: queryKeys.committee(page, chainId),
        queryFn: () => Committee(page, chainId),
        staleTime: 30000,
    });
};

// Hooks for DAO
export const useDAO = (height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.dao(height),
        queryFn: () => DAO(height, 0),
        staleTime: 30000,
    });
};

// Hooks for Account
export const useAccount = (height: number, address: string) => {
    return useQuery({
        queryKey: queryKeys.account(height, address),
        queryFn: () => Account(height, address),
        staleTime: 30000,
        enabled: !!address,
    });
};

// Hooks for Account with Transactions
export const useAccountWithTxs = (height: number, address: string, page: number, perPage: number = 10) => {
    return useQuery({
        queryKey: queryKeys.accountWithTxs(height, address, page, perPage),
        queryFn: () => AccountWithTxs(height, address, page, perPage),
        staleTime: 30000,
        enabled: !!address,
    });
};

// Hooks for Params
export const useParams = (height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.params(height),
        queryFn: () => Params(height, 0),
        staleTime: 30000,
    });
};

// Hooks for Supply
export const useSupply = (height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.supply(height),
        queryFn: () => Supply(height, 0),
        staleTime: 30000,
    });
};

// Hooks for Validator
export const useValidator = (height: number, address: string) => {
    return useQuery({
        queryKey: queryKeys.validator(height, address),
        queryFn: () => Validator(height, address),
        staleTime: 30000,
        enabled: !!address,
    });
};

// Hooks for Block by Height
export const useBlockByHeight = (height: number) => {
    return useQuery({
        queryKey: queryKeys.blockByHeight(height),
        queryFn: () => BlockByHeight(height),
        staleTime: 30000,
        enabled: height > 0,
    });
};

// Hooks for Block by Hash
export const useBlockByHash = (hash: string) => {
    return useQuery({
        queryKey: queryKeys.blockByHash(hash),
        queryFn: () => BlockByHash(hash),
        staleTime: 30000,
        enabled: !!hash,
    });
};

// Hooks for Transaction by Hash
export const useTxByHash = (hash: string) => {
    return useQuery({
        queryKey: queryKeys.txByHash(hash),
        queryFn: () => TxByHash(hash),
        staleTime: 30000,
        enabled: !!hash,
    });
};

// Hooks for Transactions by Sender
export const useTransactionsBySender = (page: number, sender: string) => {
    return useQuery({
        queryKey: queryKeys.transactionsBySender(page, sender),
        queryFn: () => TransactionsBySender(page, sender),
        staleTime: 30000,
        enabled: !!sender,
    });
};

// Hooks for Transactions by Receiver
export const useTransactionsByRec = (page: number, rec: string) => {
    return useQuery({
        queryKey: queryKeys.transactionsByRec(page, rec),
        queryFn: () => TransactionsByRec(page, rec),
        staleTime: 30000,
        enabled: !!rec,
    });
};

// Hooks for Pending Transactions
export const usePending = (page: number, perPage: number = 10) => {
    return useQuery({
        queryKey: queryKeys.pending(page, perPage),
        queryFn: () => Pending(page, perPage),
        staleTime: 10000,
    });
};

// Hooks for Eco Params
export const useEcoParams = (chainId: number) => {
    return useQuery({
        queryKey: queryKeys.ecoParams(chainId),
        queryFn: () => EcoParams(chainId),
        staleTime: 30000,
    });
};


// Hooks for Config
export const useConfig = () => {
    return useQuery({
        queryKey: queryKeys.config(),
        queryFn: () => Config(),
        staleTime: 60000, // Longer stale time for config
    });
};

// Hooks for Modal Data
export const useModalData = (query: string | number, page: number) => {
    return useQuery({
        queryKey: queryKeys.modalData(query, page),
        queryFn: () => getModalData(query, page),
        staleTime: 30000,
        enabled: !!query,
    });
};

// Hooks for Card Data
export const useCardData = () => {
    const queryClient = useQueryClient();
    const queryKey = [...queryKeys.cardData(), rpcURL];

    return useQuery({
        queryKey,
        queryFn: () => {
            const previousCardData = queryClient.getQueryData(queryKey);
            return getCardData(previousCardData);
        },
        staleTime: 0,
        refetchInterval: errorAwareInterval(BLOCKS_POLL_MS),
        refetchOnWindowFocus: true,
        refetchOnMount: 'always',
        placeholderData: (previousData) => previousData,
        retry: false,
    });
};

// Hooks for Table Data
export const useTableData = (page: number, category: number, committee?: number) => {
    return useQuery({
        queryKey: queryKeys.tableData(page, category, committee),
        queryFn: () => getTableData(page, category, committee),
        staleTime: 30000,
    });
};

// Hook to load the most recent block page and reuse the data across the app.
// Previously this fanned out 10 parallel page fetches every 10s; with a
// degraded upstream that produced 10 browser-emitted error lines per poll.
// One page (10 blocks) is enough for Home, Navbar, and the dashboard tables.
export const useAllBlocksCache = () => {
    return useQuery({
        queryKey: ['allBlocksCache'],
        queryFn: async () => {
            const response = await fetch(`${rpcURL}/v1/query/blocks`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ perPage: 10, pageNumber: 1 }),
            });

            if (!response.ok) {
                throw new Error(`blocks RPC returned ${response.status}`);
            }

            const data = await response.json();
            const blocks: any[] = Array.isArray(data?.results) ? [...data.results] : [];

            // Keep newest block first for Navbar and Home blocks table.
            blocks.sort((a: any, b: any) => {
                const ah = Number((a?.blockHeader?.height ?? a?.height) || 0);
                const bh = Number((b?.blockHeader?.height ?? b?.height) || 0);
                return bh - ah;
            });

            if (typeof data?.totalCount === 'number') {
                (blocks as any).totalCount = data.totalCount;
            }

            if (blocks.length === 0) {
                throw new Error('No blocks returned from RPC');
            }

            return blocks;
        },
        staleTime: 5000,
        refetchInterval: errorAwareInterval(10000),
        refetchOnWindowFocus: true,
        refetchOnMount: 'always',
        retry: false,
    });
};

// Define queryKeys for blocks in range
const blocksInRangeKey = (fromBlock: number, toBlock: number, maxBlocks: number) =>
    ['blocksInRange', fromBlock, toBlock, maxBlocks];

// Hook for fetching blocks within a specific range - now reuses cached data
export const useBlocksInRange = (fromBlock: number, toBlock: number, maxBlocksToFetch: number = 10) => {
    // Use the cache of all blocks
    const { data: allBlocks, isLoading, error } = useAllBlocksCache();

    // Process data on the client without making more requests
    const processedData = React.useMemo(() => {
        if (!allBlocks || !Array.isArray(allBlocks)) {
            return { results: [], totalCount: 0 };
        }

        let filteredBlocks = allBlocks;

        // Filter blocks by height if fromBlock or toBlock are specified
        if (fromBlock > 0 || toBlock > 0) {
            filteredBlocks = allBlocks.filter(block => {
                const blockHeight = block.height || block.blockHeader?.height || 0;
                return blockHeight >= fromBlock && blockHeight <= toBlock;
            });
        }

        // Ensure we don't return more than maxBlocksToFetch
        const finalBlocks = filteredBlocks.slice(0, maxBlocksToFetch);

        return {
            results: finalBlocks,
            totalCount: finalBlocks.length,
        };
    }, [allBlocks, fromBlock, toBlock, maxBlocksToFetch]);

    return {
        data: processedData,
        isLoading,
        error
    };
};


// Hook for Analytics - Get multiple pages of blocks for transaction analysis
export const useBlocksForAnalytics = (numPages: number = 10) => {
    // Use the global blocks cache
    const { data: allBlocks, isLoading, error } = useAllBlocksCache();

    // Process data on the client without making more requests
    const processedData = React.useMemo(() => {
        if (!allBlocks || !Array.isArray(allBlocks)) {
            return { results: [], totalCount: 0 };
        }

        // Limit to a maximum of 100 blocks (10 pages * 10 blocks per page)
        const maxBlocks = Math.min(numPages * 10, 100);
        const finalBlocks = allBlocks.slice(0, maxBlocks);

        return {
            results: finalBlocks,
            totalCount: finalBlocks.length,
        };
    }, [allBlocks, numPages]);

    return {
        data: processedData,
        isLoading,
        error
    };
};

// Hook to extract transactions from blocks in a specific range
export const useTransactionsInRange = (fromBlock: number, toBlock: number, maxBlocksToFetch: number = 50) => {
    // Use the global blocks cache
    const { data: allBlocks, isLoading, error } = useAllBlocksCache();

    // Process data on the client without making more requests
    const processedData = React.useMemo(() => {
        if (!allBlocks || !Array.isArray(allBlocks)) {
            return { results: [], totalCount: 0 };
        }

        let filteredBlocks = allBlocks;

        // Filter blocks by height if fromBlock or toBlock are specified
        if (fromBlock > 0 || toBlock > 0) {
            filteredBlocks = allBlocks.filter(block => {
                const blockHeight = block.height || block.blockHeader?.height || 0;
                return blockHeight >= fromBlock && blockHeight <= toBlock;
            });
        }

        // limit blocks to the requested max (capped at 100 to stay reasonable)
        const limitedBlocks = Math.min(maxBlocksToFetch, 100);
        const finalBlocks = filteredBlocks.slice(0, limitedBlocks);

        const allTransactions: any[] = [];

        // Extract transactions from each block
        finalBlocks.forEach((block: any) => {
            if (block.transactions && Array.isArray(block.transactions)) {
                // Add block information to each transaction
                const txsWithBlockInfo = block.transactions.map((tx: any) => ({
                    ...tx,
                    blockHeight: block.blockHeader?.height || block.height,
                    blockTime: block.blockHeader?.time || block.time,
                }));

                allTransactions.push(...txsWithBlockInfo);
            }
        });

        return {
            results: allTransactions,
            totalCount: allTransactions.length
        };
    }, [allBlocks, fromBlock, toBlock, maxBlocksToFetch]);

    return {
        data: processedData,
        isLoading,
        error
    };
};

// Hook for fetching orders (swaps)
export const useOrders = () => {
    return useQuery({
        queryKey: ['orders'],
        queryFn: () => Orders(),
        staleTime: 30000, // Cache for 30 seconds
        refetchInterval: REFRESH_INTERVAL_MS,
    });
};

// Hook for fetching a specific order
export const useOrder = (committee: number, orderId: string, height: number = 0) => {
    return useQuery({
        queryKey: ['order', committee, orderId, height],
        queryFn: () => Order(committee, orderId, height),
        enabled: !!orderId,
        staleTime: 30000,
    });
};

// Hook for fetching the locked dex batch
export const useDexBatch = (chainId: number = 1) => {
    return useQuery({
        queryKey: ['dexBatch', chainId],
        queryFn: () => DexBatch(0, chainId),
        staleTime: REFRESH_INTERVAL_MS,
        refetchInterval: REFRESH_INTERVAL_MS,
    });
};

// Hook for fetching the next dex batch
export const useNextDexBatch = (chainId: number = 1) => {
    return useQuery({
        queryKey: ['nextDexBatch', chainId],
        queryFn: () => NextDexBatch(0, chainId),
        staleTime: REFRESH_INTERVAL_MS,
        refetchInterval: REFRESH_INTERVAL_MS,
    });
};

// Hook to handle network changes and invalidate queries
export const useNetworkChangeHandler = () => {
    const queryClient = useQueryClient();

    React.useEffect(() => {
        const handleApiConfigChange = (event: any) => {
            // Invalidate specific queries that depend on network data
            queryClient.invalidateQueries({ queryKey: ['cardData'] });
            queryClient.invalidateQueries({ queryKey: ['blocks'] });
            queryClient.invalidateQueries({ queryKey: ['transactions'] });
            queryClient.invalidateQueries({ queryKey: ['accounts'] });
            queryClient.invalidateQueries({ queryKey: ['validators'] });
            queryClient.invalidateQueries({ queryKey: ['supply'] });
            queryClient.invalidateQueries({ queryKey: ['params'] });
            queryClient.invalidateQueries({ queryKey: ['ecoParams'] });
            queryClient.invalidateQueries({ queryKey: ['orders'] });

            // Also invalidate all queries as fallback
            queryClient.invalidateQueries();
        };

        // Listen for API config changes
        window.addEventListener('apiConfigChanged', handleApiConfigChange);
        window.addEventListener('networkChanged', handleApiConfigChange);

        return () => {
            window.removeEventListener('apiConfigChanged', handleApiConfigChange);
            window.removeEventListener('networkChanged', handleApiConfigChange);
        };
    }, [queryClient]);
};
