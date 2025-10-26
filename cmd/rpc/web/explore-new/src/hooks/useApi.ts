import { useQuery } from '@tanstack/react-query';
import React from 'react';
import {
    Blocks,
    Transactions,
    AllTransactions,
    getTransactionsWithRealPagination,
    Accounts,
    Validators,
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
    Order
} from '../lib/api';

// Query Keys
export const queryKeys = {
    blocks: (page: number) => ['blocks', page],
    transactions: (page: number, height: number) => ['transactions', page, height],
    allTransactions: (page: number, perPage: number, filters?: any) => ['allTransactions', page, perPage, filters],
    realPaginationTransactions: (page: number, perPage: number, filters?: any) => ['realPaginationTransactions', page, perPage, filters],
    accounts: (page: number) => ['accounts', page],
    validators: (page: number) => ['validators', page],
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
export const useBlocks = (page: number, perPage: number = 10) => {
    return useQuery({
        queryKey: queryKeys.blocks(page),
        queryFn: () => Blocks(page, perPage),
        staleTime: 30000, // 30 seconds
    });
};

// Hooks for Transactions
export const useTransactions = (page: number, height: number = 0) => {
    return useQuery({
        queryKey: queryKeys.transactions(page, height),
        queryFn: () => Transactions(page, height),
        staleTime: 30000,
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

// Hook para transacciones con paginación real (recomendado)
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
        queryKey: queryKeys.cardData(),
        queryFn: () => getCardData(),
        staleTime: 30000,
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

// Define queryKeys for blocks in range
const blocksInRangeKey = (fromBlock: number, toBlock: number, maxBlocks: number) => 
    ['blocksInRange', fromBlock, toBlock, maxBlocks];

// Hook for fetching blocks within a specific range
export const useBlocksInRange = (fromBlock: number, toBlock: number, maxBlocksToFetch: number = 100) => {
    return useQuery({
        queryKey: blocksInRangeKey(fromBlock, toBlock, maxBlocksToFetch),
        queryFn: async () => {
            const allBlocks: any[] = [];
            let page = 1;
            const perPage = 100; // Max blocks per page from API

            while (allBlocks.length < maxBlocksToFetch) {
                try {
                    const response = await fetch('http://localhost:50002/v1/query/blocks', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            perPage: perPage,
                            pageNumber: page,
                        }),
                    });

                    if (!response.ok) {
                        console.error(`Failed to fetch blocks page ${page}`);
                        throw new Error(`Failed to fetch blocks page ${page}`);
                    }

                    const data = await response.json();
                    if (data.results && Array.isArray(data.results)) {
                        allBlocks.push(...data.results);
                    }

                    // If we got less than perPage blocks, we've reached the end of available blocks
                    if (data.results.length < perPage) {
                        break;
                    }

                    page++;
                } catch (error: any) {
                    console.error(`Error fetching blocks page ${page}:`, error);
                    throw new Error(`Error fetching blocks: ${error.message}`);
                }
            }

            // Filter blocks by height if fromBlock or toBlock are specified
            let filteredBlocks = allBlocks;
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
        },
        staleTime: 60000, // Cache for 1 minute
        refetchInterval: 300000, // Refetch every 5 minutes
    });
};

// Hook for Analytics - Get multiple pages of blocks for transaction analysis
export const useBlocksForAnalytics = (numPages: number = 10) => {
    // Usa el hook de useBlocksInRange para obtener los bloques
    return useBlocksInRange(0, 0, numPages * 100); // Fetch up to numPages * 100 blocks
};

// Hook para extraer transacciones de los bloques en un rango específico
export const useTransactionsInRange = (fromBlock: number, toBlock: number, maxBlocksToFetch: number = 100) => {
    // Reutilizar el hook de useBlocksInRange para obtener los bloques
    const blocksQuery = useBlocksInRange(fromBlock, toBlock, maxBlocksToFetch);
    
    // Procesar los bloques para extraer las transacciones
    const { data: blocksData, isLoading, error } = blocksQuery;
    
    // Transformar los datos de bloques para extraer las transacciones
    const transformedData = React.useMemo(() => {
        if (!blocksData?.results || !Array.isArray(blocksData.results)) {
            return { results: [], totalCount: 0 };
        }
        
        const allTransactions: any[] = [];
        
        // Extraer transacciones de cada bloque
        blocksData.results.forEach((block: any) => {
            if (block.transactions && Array.isArray(block.transactions)) {
                // Agregar información del bloque a cada transacción
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
    }, [blocksData]);
    
    return {
        data: transformedData,
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

