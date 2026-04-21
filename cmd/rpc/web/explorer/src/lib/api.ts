// API Configuration
// Get environment variables with fallbacks
const getEnvVar = (key: keyof ImportMetaEnv, fallback: string): string => {
    return import.meta.env[key] || fallback;
};

const normalizeBaseURL = (url: string): string => {
    return url.replace(/\/+$/, '');
};

const buildURL = (baseURL: string, endpointPath: string): string => {
    const normalizedBase = normalizeBaseURL(baseURL);
    const normalizedPath = endpointPath.replace(/^\/+/, '');
    return `${normalizedBase}/${normalizedPath}`;
};

// Merkle root used by the blockchain when a block contains no transactions (32 bytes of 0x46 / ASCII 'F')
const EMPTY_TRANSACTION_ROOT = "4646464646464646464646464646464646464646464646464646464646464646";

// Default values
let rpcURL = getEnvVar('VITE_RPC_URL', "http://localhost:50002");
let adminRPCURL = getEnvVar('VITE_ADMIN_RPC_URL', "http://localhost:50003");
let chainId = parseInt(getEnvVar('VITE_CHAIN_ID', "1"));

// Check if we're in production mode and use public URLs
const isProduction = getEnvVar('VITE_NODE_ENV', 'development') === 'production';
if (isProduction) {
    rpcURL = getEnvVar('VITE_PUBLIC_RPC_URL', rpcURL);
    adminRPCURL = getEnvVar('VITE_PUBLIC_ADMIN_RPC_URL', adminRPCURL);
}

// Override with window.__CONFIG__ if available (for network selector)
if (typeof window !== 'undefined' && window.__CONFIG__) {
    rpcURL = window.__CONFIG__.rpcURL;
    adminRPCURL = window.__CONFIG__.adminRPCURL;
    chainId = window.__CONFIG__.chainId;
}

// Function to update API configuration
const updateApiConfig = (newRpcURL: string, newAdminRPCURL: string, newChainId: number) => {
    rpcURL = normalizeBaseURL(newRpcURL);
    adminRPCURL = normalizeBaseURL(newAdminRPCURL);
    chainId = newChainId;
    console.log('API Config Updated:', { rpcURL, adminRPCURL, chainId });

    // Dispatch custom event for React Query invalidation
    if (typeof window !== 'undefined') {
        window.dispatchEvent(new CustomEvent('apiConfigChanged', {
            detail: { rpcURL, adminRPCURL, chainId }
        }));
    }
};

// Legacy support for window.__CONFIG__ (for backward compatibility)
if (typeof window !== "undefined") {
    if (window.__CONFIG__) {
        rpcURL = normalizeBaseURL(window.__CONFIG__.rpcURL);
        adminRPCURL = normalizeBaseURL(window.__CONFIG__.adminRPCURL);
        chainId = Number(window.__CONFIG__.chainId);
    }

    // On Netlify deployment, use same-origin proxy paths to avoid browser CORS blocks.
    if (window.location.hostname === "canopy.nodefleet.net") {
        rpcURL = "/rpc-node1";
        adminRPCURL = "/admin-node1";
    }

    // Replace localhost with current hostname for local development
    if (rpcURL.includes("localhost")) {
        rpcURL = rpcURL.replace("localhost", window.location.hostname);
    }
    if (adminRPCURL.includes("localhost")) {
        adminRPCURL = adminRPCURL.replace("localhost", window.location.hostname);
    }

    // Listen for network changes
    window.addEventListener('networkChanged', (event: any) => {
        const network = event.detail;
        updateApiConfig(network.rpcUrl, network.adminRpcUrl, network.chainId);
    });

    console.log('RPC URL:', rpcURL);
    console.log('Admin RPC URL:', adminRPCURL);
    console.log('Chain ID:', chainId);
} else {
    console.log("Running in SSR mode, using environment variables");
}

// RPC PATHS
const blocksPath = "/v1/query/blocks";
const blockByHashPath = "/v1/query/block-by-hash";
const blockByHeightPath = "/v1/query/block-by-height";
const txByHashPath = "/v1/query/tx-by-hash";
const txsBySender = "/v1/query/txs-by-sender";
const txsByRec = "/v1/query/txs-by-rec";
const txsByHeightPath = "/v1/query/txs-by-height";
const pendingPath = "/v1/query/pending";
const ecoParamsPath = "/v1/query/eco-params";
const validatorsPath = "/v1/query/validators";
const accountsPath = "/v1/query/accounts";
const poolPath = "/v1/query/pool";
const accountPath = "/v1/query/account";
const validatorPath = "/v1/query/validator";
const paramsPath = "/v1/query/params";
const supplyPath = "/v1/query/supply";
const ordersPath = "/v1/query/orders";
const orderPath = "/v1/query/order";
const dexBatchPath = "/v1/query/dex-batch";
const nextDexBatchPath = "/v1/query/next-dex-batch";
const configPath = "/v1/admin/config";

// Pool IDs
// 2 * MaxUint16 + 1, placed above the max chainId + EscrowPoolAddend range to avoid collisions
const DAO_POOL_ID = 131071;

// HTTP Methods
export async function POST(url: string, request: string, path: string) {
    return fetch(buildURL(url, path), {
        method: "POST",
        headers: {
            'Content-Type': 'application/json',
        },
        body: request,
    })
        .then(async (response) => {
            if (!response.ok) {
                return Promise.reject(response);
            }
            return response.json();
        })
        .catch((rejected) => {
            console.log(rejected);
            return Promise.reject(rejected);
        });
}

export async function GET(url: string, path: string) {
    return fetch(buildURL(url, path), {
        method: "GET",
    })
        .then(async (response) => {
            if (!response.ok) {
                return Promise.reject(response);
            }
            return response.json();
        })
        .catch((rejected) => {
            console.log(rejected);
            return Promise.reject(rejected);
        });
}

// Request Objects
function chainRequest(chain_id: number) {
    return JSON.stringify({ chainId: chain_id });
}

function heightRequest(height: number) {
    return JSON.stringify({ height: height });
}

function hashRequest(hash: string) {
    return JSON.stringify({ hash: hash });
}

function pageAddrReq(page: number, addr: string) {
    return JSON.stringify({ pageNumber: page, perPage: 10, address: addr });
}

function heightAndAddrRequest(height: number, address: string) {
    return JSON.stringify({ height: height, address: address });
}

function heightAndIDRequest(height: number, id: number) {
    return JSON.stringify({ height: height, id: id });
}

function pageHeightReq(page: number, height: number, perPage: number = 10) {
    return JSON.stringify({ pageNumber: page, perPage: perPage, height: height });
}

function validatorsReq(page: number, height: number, committee: number) {
    return JSON.stringify({ height: height, pageNumber: page, perPage: 1000, committee: committee });
}

// API Calls
export function Blocks(page: number, perPage: number = 10) {
    return POST(rpcURL, JSON.stringify({ pageNumber: page, perPage: perPage }), blocksPath);
}

export function Transactions(page: number, height: number, perPage: number = 10) {
    return POST(rpcURL, pageHeightReq(page, height, perPage), txsByHeightPath);
}

// Optimized function to get transactions with real pagination
export async function getTransactionsWithRealPagination(page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    fromBlock?: string;
    toBlock?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}) {
    try {
        // Get the total number of transactions
        const totalTransactionCount = await getTotalTransactionCount();

        // If there are no filters, use a more direct approach
        if (!filters || Object.values(filters).every(v => !v)) {
            // Get blocks sequentially to cover the pagination
            const startIndex = (page - 1) * perPage;
            const endIndex = startIndex + perPage;

            let allTransactions: any[] = [];
            let currentBlockPage = 1;
            const maxPages = 50; // Limit to avoid too many requests

            while (allTransactions.length < endIndex && currentBlockPage <= maxPages) {
                const blocksResponse = await Blocks(currentBlockPage, 0);
                const blocks = blocksResponse?.results || blocksResponse?.blocks || [];

                if (!Array.isArray(blocks) || blocks.length === 0) break;

                for (const block of blocks) {
                    if (block.transactions && Array.isArray(block.transactions)) {
                        const blockTransactions = block.transactions.map((tx: any) => ({
                            ...tx,
                            blockHeight: block.blockHeader?.height || block.height,
                            blockHash: block.blockHeader?.hash || block.hash,
                            blockTime: block.blockHeader?.time || block.time,
                            blockNumber: block.blockHeader?.height || block.height
                        }));
                        allTransactions = allTransactions.concat(blockTransactions);

                        // If we have enough transactions, exit
                        if (allTransactions.length >= endIndex) break;
                    }
                }

                currentBlockPage++;
            }

            // Sort by time (most recent first)
            allTransactions.sort((a, b) => {
                const timeA = a.blockTime || a.time || a.timestamp || 0;
                const timeB = b.blockTime || b.time || b.timestamp || 0;
                return timeB - timeA;
            });

            // Apply pagination
            const paginatedTransactions = allTransactions.slice(startIndex, endIndex);

            return {
                results: paginatedTransactions,
                totalCount: totalTransactionCount.total,
                pageNumber: page,
                perPage: perPage,
                totalPages: Math.ceil(totalTransactionCount.total / perPage),
                hasMore: endIndex < totalTransactionCount.total
            };
        }

        // If there are filters, use the previous method
        return await AllTransactions(page, perPage, filters);

    } catch (error) {
        console.error('Error fetching transactions with real pagination:', error);
        return { results: [], totalCount: 0, pageNumber: page, perPage, totalPages: 0, hasMore: false };
    }
}

// New function to get total transaction count
// Cache for total transaction count
let totalTransactionCountCache: { count: number; last24h: number; tpm: number; timestamp: number } | null = null;
const CACHE_DURATION = 30000; // 30 seconds

export async function getTotalAccountCount(cachedBlocks?: any[]): Promise<{ total: number, last24h: number }> {
    try {
        // Get total accounts
        const accountsResponse = await Accounts(1, 0);
        const totalAccounts = accountsResponse?.totalCount || accountsResponse?.count || 0;

        // Get accounts from last 24h by checking recent blocks
        const twentyFourHoursAgo = Date.now() - 24 * 60 * 60 * 1000;
        let accountsLast24h = 0;

        // if we have cached blocks, use them
        if (cachedBlocks && Array.isArray(cachedBlocks) && cachedBlocks.length > 0) {

            for (const block of cachedBlocks) {
                const blockTime = block.blockHeader?.time || block.time;
                if (blockTime) {
                    let date: Date;
                    try {
                        if (typeof blockTime === 'number') {
                            if (blockTime > 1e15) {
                                date = new Date(blockTime / 1000000);
                            } else if (blockTime > 1e12) {
                                date = new Date(blockTime);
                            } else {
                                date = new Date(blockTime * 1000);
                            }
                        } else {
                            date = new Date(blockTime);
                        }

                        if (date.getTime() >= twentyFourHoursAgo) {
                            // Count accounts from transactions in this block
                            if (block.transactions && Array.isArray(block.transactions)) {
                                for (const tx of block.transactions) {
                                    // Count unique senders as new accounts
                                    if (tx.sender) {
                                        accountsLast24h++;
                                    }
                                }
                            }
                        }
                    } catch (error) {
                        console.log('Invalid block timestamp:', blockTime, error);
                    }
                }
            }
            return {
                total: totalAccounts,
                last24h: accountsLast24h
            };
        }

        return {
            total: totalAccounts,
            last24h: 0
        };
    } catch (error) {
        console.error('Error getting total account count:', error);
        return {
            total: 0,
            last24h: 0
        };
    }
}

export async function getTotalTransactionCount(cachedBlocks?: any[]): Promise<{ total: number, last24h: number, tpm: number }> {
    try {
        // Check cache
        if (totalTransactionCountCache &&
            (Date.now() - totalTransactionCountCache.timestamp) < CACHE_DURATION) {
            return {
                total: totalTransactionCountCache.count,
                last24h: totalTransactionCountCache.last24h || 0,
                tpm: totalTransactionCountCache.tpm || 0
            };
        }

        let totalCount = 0;
        let last24hCount = 0;
        const twentyFourHoursAgo = Date.now() - 24 * 60 * 60 * 1000;

        // If cached blocks are available, use them
        if (cachedBlocks && Array.isArray(cachedBlocks) && cachedBlocks.length > 0) {
            for (const block of cachedBlocks) {
                if (block.transactions && Array.isArray(block.transactions)) {
                    totalCount += block.transactions.length;

                    // Count transactions from last 24h
                    for (const tx of block.transactions) {
                        const timestamp = tx.time || tx.timestamp || tx.blockTime || block.blockHeader?.time || block.time;
                        if (timestamp) {
                            let date: Date;
                            try {
                                if (typeof timestamp === 'number') {
                                    if (timestamp > 1e15) {
                                        date = new Date(timestamp / 1000000);
                                    } else if (timestamp > 1e12) {
                                        date = new Date(timestamp);
                                    } else {
                                        date = new Date(timestamp * 1000);
                                    }
                                } else if (typeof timestamp === 'string') {
                                    date = new Date(timestamp);
                                } else {
                                    date = new Date(timestamp);
                                }

                                const txTime = date.getTime();
                                if (txTime >= twentyFourHoursAgo) {
                                    last24hCount++;
                                }
                            } catch (error) {
                                console.log('Invalid timestamp:', timestamp, error);
                            }
                        }
                    }
                }
            }

            // Calculate TPM (Transactions Per Minute)
            const minutesIn24h = 24 * 60;
            const tpm = last24hCount > 0 ? last24hCount / minutesIn24h : 0;

            // Update cache
            totalTransactionCountCache = {
                count: totalCount,
                last24h: last24hCount,
                tpm: tpm,
                timestamp: Date.now()
            };

            return {
                total: totalCount,
                last24h: last24hCount,
                tpm: Math.round(tpm * 100) / 100
            };
        }

        return {
            total: 0,
            last24h: 0,
            tpm: 0
        };
    } catch (error) {
        console.error('Error getting total transaction count:', error);
        return {
            total: totalTransactionCountCache?.count || 0,
            last24h: totalTransactionCountCache?.last24h || 0,
            tpm: totalTransactionCountCache?.tpm || 0
        };
    }
}

// Extract transactions from a list of blocks, attaching block metadata to each tx
function extractTransactionsFromBlocks(blocks: any[]): any[] {
    const txs: any[] = [];
    for (const block of blocks) {
        if (block.transactions && Array.isArray(block.transactions)) {
            const blockHeight = block.blockHeader?.height || block.height || 0;
            for (const tx of block.transactions) {
                txs.push({
                    ...tx,
                    blockHeight,
                    blockHash: block.blockHeader?.hash || block.hash,
                    blockTime: block.blockHeader?.time || block.time,
                    blockNumber: blockHeight,
                });
            }
        }
    }
    return txs;
}

import { extractAmountMicro } from './utils';

// Apply non-block-range filters to a list of transactions
function applyTxFilters(txs: any[], filters: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}): any[] {
    return txs.filter(tx => {
        if (filters.type && filters.type !== 'All Types') {
            const txType = tx.messageType || tx.type || 'send';
            if (txType.toLowerCase() !== filters.type.toLowerCase()) return false;
        }
        if (filters.address) {
            const addr = filters.address.toLowerCase();
            const sender = (tx.sender || tx.from || '').toLowerCase();
            const recipient = (tx.recipient || tx.to || '').toLowerCase();
            const hash = (tx.txHash || tx.hash || '').toLowerCase();
            if (!sender.includes(addr) && !recipient.includes(addr) && !hash.includes(addr)) return false;
        }
        if (filters.fromDate || filters.toDate) {
            const txTime = tx.blockTime || tx.time || tx.timestamp;
            if (txTime) {
                const txDate = new Date(txTime > 1e12 ? txTime / 1000 : txTime);
                if (filters.fromDate && txDate < new Date(filters.fromDate)) return false;
                if (filters.toDate) {
                    const toDate = new Date(filters.toDate);
                    toDate.setHours(23, 59, 59, 999);
                    if (txDate > toDate) return false;
                }
            }
        }
        if (filters.minAmount !== undefined || filters.maxAmount !== undefined) {
            const amount = extractAmountMicro(tx as Record<string, unknown>);
            if (filters.minAmount !== undefined && amount < filters.minAmount) return false;
            if (filters.maxAmount !== undefined && amount > filters.maxAmount) return false;
        }
        if (filters.status && filters.status !== 'all') {
            if ((tx.status || 'success') !== filters.status) return false;
        }
        return true;
    });
}

// Fetch transactions from blocks within a specific height range by
// querying each block height directly via the BlockByHeight RPC endpoint.
async function fetchTransactionsForBlockRange(
    fromHeight: number,
    toHeight: number,
    filters: { type?: string; fromDate?: string; toDate?: string; status?: string; address?: string; minAmount?: number; maxAmount?: number },
): Promise<{ transactions: any[]; totalFiltered: number }> {
    const metaResponse = await Blocks(1, 0);
    const latestHeight: number = metaResponse?.totalCount || 0;

    const effectiveFrom = Math.max(fromHeight || 1, 1);
    const effectiveTo = latestHeight > 0
        ? Math.min(toHeight || latestHeight, latestHeight)
        : (toHeight || effectiveFrom);

    if (effectiveFrom > effectiveTo) return { transactions: [], totalFiltered: 0 };

    // Cap the number of heights to query (newest first within the range)
    const maxHeights = 500;
    const rangeSize = effectiveTo - effectiveFrom + 1;
    const cappedFrom = rangeSize > maxHeights ? effectiveTo - maxHeights + 1 : effectiveFrom;

    let allTransactions: any[] = [];
    const batchSize = 10;

    // Fetch blocks by height in parallel batches, newest first
    for (let batchTop = effectiveTo; batchTop >= cappedFrom; batchTop -= batchSize) {
        const batchBottom = Math.max(batchTop - batchSize + 1, cappedFrom);
        const promises: Promise<any>[] = [];

        for (let h = batchTop; h >= batchBottom; h--) {
            promises.push(
                BlockByHeight(h).catch(() => null)
            );
        }

        const results = await Promise.all(promises);

        for (const block of results) {
            if (!block) continue;
            const txs = extractTransactionsFromBlocks([block]);
            allTransactions = allTransactions.concat(txs);
        }
    }

    const filtered = applyTxFilters(allTransactions, filters);
    return { transactions: filtered, totalFiltered: filtered.length };
}

// Fetch transactions from recent blocks (no block-range constraint)
async function fetchRecentTransactions(
    targetCount: number,
    filters: { type?: string; fromDate?: string; toDate?: string; status?: string; address?: string; minAmount?: number; maxAmount?: number },
    hasFilters: boolean,
): Promise<{ transactions: any[]; totalFiltered: number; totalRaw: number }> {
    let allTransactions: any[] = [];
    let currentBlockPage = 1;
    const maxBlockPages = 50;
    const collectAll = hasFilters;

    while (currentBlockPage <= maxBlockPages && (collectAll || allTransactions.length < targetCount)) {
        const blocksResponse = await Blocks(currentBlockPage, 0);
        const blocks = blocksResponse?.results || blocksResponse?.blocks || blocksResponse?.list || [];
        if (!Array.isArray(blocks) || blocks.length === 0) break;
        allTransactions = allTransactions.concat(extractTransactionsFromBlocks(blocks));
        currentBlockPage++;
    }

    const totalRaw = allTransactions.length;
    const filtered = hasFilters ? applyTxFilters(allTransactions, filters) : allTransactions;
    return { transactions: filtered, totalFiltered: filtered.length, totalRaw };
}

// Main entry point: get transactions from multiple blocks with filters and pagination
export async function AllTransactions(page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    fromBlock?: string;
    toBlock?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}) {
    try {
        const fromHeight = filters?.fromBlock ? Number(filters.fromBlock) : 0;
        const toHeight = filters?.toBlock ? Number(filters.toBlock) : 0;
        const hasBlockRange = (fromHeight > 0 || toHeight > 0);

        const nonBlockFilters = {
            type: filters?.type,
            fromDate: filters?.fromDate,
            toDate: filters?.toDate,
            status: filters?.status,
            address: filters?.address,
            minAmount: filters?.minAmount,
            maxAmount: filters?.maxAmount,
        };
        const hasNonBlockFilters = Object.values(nonBlockFilters).some(v => v !== undefined && v !== '');

        let allFiltered: any[];
        let totalCount: number;

        if (hasBlockRange) {
            const result = await fetchTransactionsForBlockRange(fromHeight, toHeight, nonBlockFilters);
            allFiltered = result.transactions;
            totalCount = result.totalFiltered;
        } else {
            const totalTransactionCount = await getTotalTransactionCount();
            const result = await fetchRecentTransactions(page * perPage, nonBlockFilters, hasNonBlockFilters);
            allFiltered = result.transactions;
            totalCount = hasNonBlockFilters ? result.totalFiltered : totalTransactionCount.total;
        }

        // Sort by block height descending (most recent first)
        allFiltered.sort((a, b) => {
            const hA = a.blockHeight || a.height || 0;
            const hB = b.blockHeight || b.height || 0;
            return hB - hA;
        });

        const startIndex = (page - 1) * perPage;
        const endIndex = startIndex + perPage;
        const paginatedTransactions = allFiltered.slice(startIndex, endIndex);

        return {
            results: paginatedTransactions,
            totalCount: totalCount,
            pageNumber: page,
            perPage: perPage,
            totalPages: Math.ceil(totalCount / perPage),
            hasMore: endIndex < totalCount,
        };
    } catch (error) {
        console.error('Error fetching all transactions:', error);
        return { results: [], totalCount: 0, pageNumber: page, perPage, totalPages: 0, hasMore: false };
    }
}

export function Accounts(page: number, _: number) {
    return POST(rpcURL, pageHeightReq(page, 0), accountsPath);
}

export function Validators(page: number, _: number) {
    return POST(rpcURL, pageHeightReq(page, 0), validatorsPath);
}

export function ValidatorsWithFilters(page: number, unstaking: number = 0, paused: number = 0, delegate: number = 0, committee: number = 0, perPage: number = 1000) {
    const request = {
        height: 0,
        perPage: perPage,
        pageNumber: page,
        unstaking,
        paused,
        delegate,
        committee
    };
    return POST(rpcURL, JSON.stringify(request), validatorsPath);
}

export function Committee(page: number, chain_id: number) {
    return POST(rpcURL, validatorsReq(page, 0, chain_id), validatorsPath);
}

export function DAO(height: number, _: number) {
    return POST(rpcURL, heightAndIDRequest(height, DAO_POOL_ID), poolPath);
}

export function Account(height: number, address: string) {
    return POST(rpcURL, heightAndAddrRequest(height, address), accountPath);
}

export async function AccountWithTxs(height: number, address: string, page: number) {
    const result: any = {};
    result.account = await Account(height, address);
    result.sent_transactions = await TransactionsBySender(page, address);
    result.rec_transactions = await TransactionsByRec(page, address);
    return result;
}

export function Params(height: number, _: number) {
    return POST(rpcURL, heightRequest(height), paramsPath);
}

export function Supply(height: number, _: number) {
    return POST(rpcURL, heightRequest(height), supplyPath);
}

export function Validator(height: number, address: string) {
    return POST(rpcURL, heightAndAddrRequest(height, address), validatorPath);
}

export function BlockByHeight(height: number) {
    return POST(rpcURL, heightRequest(height), blockByHeightPath);
}

export function BlockByHash(hash: string) {
    return POST(rpcURL, hashRequest(hash), blockByHashPath);
}

export function TxByHash(hash: string) {
    return POST(rpcURL, hashRequest(hash), txByHashPath);
}

export function TransactionsBySender(page: number, sender: string) {
    return POST(rpcURL, pageAddrReq(page, sender), txsBySender);
}

export function TransactionsByRec(page: number, rec: string) {
    return POST(rpcURL, pageAddrReq(page, rec), txsByRec);
}

export function Pending(page: number, _: number) {
    return POST(rpcURL, pageAddrReq(page, ""), pendingPath);
}

export function EcoParams(chain_id: number) {
    return POST(rpcURL, chainRequest(chain_id), ecoParamsPath);
}

export function Orders() {
    return POST(rpcURL, JSON.stringify({ height: 0 }), ordersPath);
}

export function Order(committee: number, order_id: string, height: number = 0) {
    return POST(rpcURL, JSON.stringify({ committee: committee, orderId: order_id, height: height }), orderPath);
}

export function DexBatch(height: number, chainId: number, points: boolean = false) {
    return POST(rpcURL, JSON.stringify({ height: height, id: chainId, points: points }), dexBatchPath);
}

export function NextDexBatch(height: number, chainId: number, points: boolean = false) {
    return POST(rpcURL, JSON.stringify({ height: height, id: chainId, points: points }), nextDexBatchPath);
}

export function Config() {
    return GET(adminRPCURL, configPath);
}

// Component Specific API Calls
export async function getModalData(query: string | number, page: number) {
    const noResult = "no result found";

    // Handle string query cases
    if (typeof query === "string") {
        // Block by hash
        if (query.length === 64) {
            const block = await BlockByHash(query);
            if (block?.blockHeader?.hash) return { block };

            const tx = await TxByHash(query);
            return tx?.sender ? tx : noResult;
        }

        // Validator or account by address
        if (query.length === 40) {
            const [valResult, accResult] = await Promise.allSettled([Validator(0, query), AccountWithTxs(0, query, page)]);

            const val = valResult.status === "fulfilled" ? valResult.value : null;
            const acc = accResult.status === "fulfilled" ? accResult.value : null;

            if (!acc?.account?.address && !val?.address) return noResult;
            return acc?.account?.address ? { ...acc, validator: val } : { validator: val };
        }

        return noResult;
    }

    // Handle block by height
    const block = await BlockByHeight(query);
    return block?.blockHeader?.hash ? { block } : noResult;
}

export async function getCardData(previousCardData?: any) {
    const fallbackCardData = {
        blocks: previousCardData?.blocks ?? { results: [], totalCount: 0 },
        canopyCommittee: previousCardData?.canopyCommittee ?? null,
        supply: previousCardData?.supply ?? null,
        pool: previousCardData?.pool ?? null,
        params: previousCardData?.params ?? null,
        ecoParams: previousCardData?.ecoParams ?? null,
        hasRealTransactions: previousCardData?.hasRealTransactions ?? false,
    };

    const requests = {
        blocks: () => Blocks(1, 0),
        canopyCommittee: () => Committee(1, chainId),
        supply: () => Supply(0, 0),
        pool: () => DAO(0, 0),
        params: () => Params(0, 0),
        ecoParams: () => EcoParams(0),
    };

    const entries = await Promise.allSettled(
        Object.entries(requests).map(async ([key, loader]) => [key, await loader()] as const)
    );

    const cardData: any = { ...fallbackCardData };

    for (const result of entries) {
        if (result.status === 'fulfilled') {
            const [key, value] = result.value;
            cardData[key] = value;
            continue;
        }

        console.error('❌ Error in getCardData request:', result.reason);
    }

    const blocks = cardData.blocks?.results || cardData.blocks?.blocks || [];
    if (Array.isArray(blocks) && blocks.length > 0) {
        cardData.hasRealTransactions = blocks.some((block: any) => {
            const txRoot = block.blockHeader?.transactionRoot;
            return txRoot && txRoot !== EMPTY_TRANSACTION_ROOT;
        });
    }

    return cardData;
}

export async function getTableData(page: number, category: number, committee?: number) {
    switch (category) {
        case 0:
            return await Blocks(page, 0);
        case 1:
            return await Transactions(page, 0);
        case 2:
            return await Pending(page, 0);
        case 3:
            return await Accounts(page, 0);
        case 4:
            return await Validators(page, 0);
        case 5:
            return await Params(page, 0);
        case 6:
            return await Orders();
        case 7:
            return await Supply(0, 0);
        default:
            return null;
    }
}

// Export rpcURL for use in hooks
export { rpcURL };
