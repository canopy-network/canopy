import { useQuery, useQueryClient } from '@tanstack/react-query';
import React from 'react';
import {
    Blocks,
    Transactions,
    AllTransactions,
    getTransactionsWithRealPagination,
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
    rpcURL
} from '../lib/api';

// Query Keys
export const queryKeys = {
    blocks: (page: number, perPage?: number, filter?: string) => ['blocks', page, perPage, filter],
    transactions: (page: number, height: number) => ['transactions', page, height],
    allTransactions: (page: number, perPage: number, filters?: any) => ['allTransactions', page, perPage, filters],
    realPaginationTransactions: (page: number, perPage: number, filters?: any) => ['realPaginationTransactions', page, perPage, filters],
    accounts: (page: number) => ['accounts', page],
    validators: (page: number) => ['validators', page],
    validatorsWithFilters: (page: number, unstaking: number, paused: number, delegate: number, committee: number) => ['validatorsWithFilters', page, unstaking, paused, delegate, committee],
    committee: (page: number, chainId: number) => ['committee', page, chainId],
    dao: (height: number) => ['dao', height],
    account: (height: number, address: string) => ['account', height, address],
    accountWithTxs: (height: number, address: string, page: number) => ['accountWithTxs', height, address, page],
    params: (height: number) => ['params', height],
    supply: (height: number) => ['supply', height],
    validator: (height: number, address: string) => ['validator', height, address],
    blockByHeight: (height: number) => ['blockByHeight', height],
    blockByHash: (hash: string) => ['blockByHash', hash],
    txByHash: (hash: string) => ['txByHash', hash],
    transactionsBySender: (page: number, sender: string) => ['transactionsBySender', page, sender],
    transactionsByRec: (page: number, rec: string) => ['transactionsByRec', page, rec],
    pending: (page: number) => ['pending', page],
    ecoParams: (chainId: number) => ['ecoParams', chainId],
    orders: (chainId: number) => ['orders', chainId],
    config: () => ['config'],
    modalData: (query: string | number, page: number) => ['modalData', query, page],
    cardData: () => ['cardData'],
    tableData: (page: number, category: number, committee?: number) => ['tableData', page, category, committee],
};

// Hooks for Blocks
export const useBlocks = (page: number, perPage: number = 10, filter: string = 'all') => {
    // Load more blocks if the filter is week or 24h to have enough data to filter
    const blockCount = filter === 'week' ? 50 : filter === '24h' ? 30 : perPage;

    return useQuery({
        queryKey: queryKeys.blocks(page, blockCount, filter),
        queryFn: () => Blocks(page, blockCount),
        staleTime: 300000, // Cache for 5 minutes (increased from 30 seconds)
        refetchInterval: 600000, // Refetch every 10 minutes
        refetchOnWindowFocus: false, // Don't refetch when window regains focus
        gcTime: 600000 // Keep in cache for 10 minutes
    });
};

// Hooks for Transactions
export const useTransactions = (page: number, height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.transactions(page, height),
        queryFn: () => Transactions(page, height),
        staleTime: 300000, // Cache for 5 minutes (increased from 30 seconds)
        refetchInterval: 600000, // Refetch every 10 minutes
        refetchOnWindowFocus: false, // Don't refetch when window regains focus
        gcTime: 600000 // Keep in cache for 10 minutes
    });
};

// Hook para todas las transacciones con filtros
export const useAllTransactions = (page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
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

// Hooks for Accounts
export const useAccounts = (page: number) => {
    return useQuery({
        queryKey: queryKeys.accounts(page),
        queryFn: () => Accounts(page, 0),
        staleTime: 30000,
    });
};

// Hooks for Validators
export const useValidators = (page: number) => {
    return useQuery({
        queryKey: queryKeys.validators(page),
        queryFn: () => Validators(page, 0),
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
        refetchInterval: 600000, // Refetch every 10 minutes
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
        refetchInterval: 600000, // Refetch every 10 minutes
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
export const useAccountWithTxs = (height: number, address: string, page: number) => {
    return useQuery({
        queryKey: queryKeys.accountWithTxs(height, address, page),
        queryFn: () => AccountWithTxs(height, address, page),
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
export const usePending = (page: number) => {
    return useQuery({
        queryKey: queryKeys.pending(page),
        queryFn: () => Pending(page, 0),
        staleTime: 10000, // Shorter stale time for pending transactions
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
    return useQuery({
        queryKey: [...queryKeys.cardData(), rpcURL], // Include RPC URL to invalidate on network change
        queryFn: () => getCardData(),
        staleTime: 5000, // Reduced stale time for more frequent updates
        refetchOnWindowFocus: true, // Refetch when window regains focus
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

// Hook para cargar TODOS los bloques UNA SOLA VEZ y reutilizar los datos
export const useAllBlocksCache = () => {
    return useQuery({
        queryKey: ['allBlocksCache'],
        queryFn: async () => {
            const allBlocks: any[] = [];
            const perPage = 10; // Max blocks per page from API
            const maxPages = 10; // Maximum 10 pages (100 blocks)

            // Hacer solo los requests necesarios
            const requests = [];
            for (let page = 1; page <= maxPages; page++) {
                requests.push(
                    fetch(`${rpcURL}/v1/query/blocks`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            perPage: perPage,
                            pageNumber: page,
                        }),
                    })
                );
            }

            try {
                // Hacer todos los requests en paralelo
                const responses = await Promise.all(requests);

                // Procesar todas las respuestas
                for (let i = 0; i < responses.length; i++) {
                    const response = responses[i];
                    if (!response.ok) {
                        console.error(`Failed to fetch blocks page ${i + 1}`);
                        throw new Error(`Failed to fetch blocks page ${i + 1}`);
                    }

                    const data = await response.json();
                    if (data.results && Array.isArray(data.results)) {
                        allBlocks.push(...data.results)
                    }
                    (allBlocks as any).totalCount = data.totalCount || 0;
                }

                return allBlocks;
            } catch (error: any) {
                console.error(`Error fetching blocks:`, error);
                throw new Error(`Error fetching blocks: ${error.message}`);
            }
        },
        staleTime: 300000, // Cache for 5 minutes
        refetchInterval: 600000, // Refetch every 10 minutes
        gcTime: 600000, // Keep in cache for 10 minutes
    });
};

// Define queryKeys for blocks in range
const blocksInRangeKey = (fromBlock: number, toBlock: number, maxBlocks: number) =>
    ['blocksInRange', fromBlock, toBlock, maxBlocks];

// Hook for fetching blocks within a specific range - AHORA REUTILIZA LOS DATOS
export const useBlocksInRange = (fromBlock: number, toBlock: number, maxBlocksToFetch: number = 10) => {
    // Usar el cache de todos los bloques
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
    // Usar el cache global de bloques
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
    // Usar el cache global de bloques
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

        // Limit to a maximum of 50 blocks to avoid too many requests
        const limitedBlocks = Math.min(maxBlocksToFetch, 50);
        const finalBlocks = filteredBlocks.slice(0, limitedBlocks);

        const allTransactions: any[] = [];

        // Extraer transacciones de cada bloque
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
export const useOrders = (chainId: number = 1) => {
    return useQuery({
        queryKey: ['orders', chainId],
        queryFn: async () => {
            const response = await Orders(chainId);
            if (!response.ok) {
                throw new Error('Failed to fetch orders');
            }
            return response.json();
        },
        staleTime: 30000, // Cache for 30 seconds
        refetchInterval: 60000, // Refetch every minute
    });
};

// Hook for fetching a specific order
export const useOrder = (chainId: number, orderId: string, height: number = 0) => {
    return useQuery({
        queryKey: ['order', chainId, orderId, height],
        queryFn: async () => {
            const response = await Order(chainId, orderId, height);
            if (!response.ok) {
                throw new Error('Failed to fetch order');
            }
            return response.json();
        },
        enabled: !!orderId, // Only run if orderId is provided
        staleTime: 30000, // Cache for 30 seconds
    });
};

// Hook to handle network changes and invalidate queries
export const useNetworkChangeHandler = () => {
    const queryClient = useQueryClient();

    React.useEffect(() => {
        const handleApiConfigChange = (event: any) => {
            console.log('🔄 Network changed, invalidating queries...', event.detail);

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


