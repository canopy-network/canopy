// API Configuration
// Get environment variables with fallbacks
const getEnvVar = (key: keyof ImportMetaEnv, fallback: string): string => {
    return import.meta.env[key] || fallback;
};

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

// Legacy support for window.__CONFIG__ (for backward compatibility)
if (typeof window !== "undefined") {
    if (window.__CONFIG__) {
        rpcURL = window.__CONFIG__.rpcURL;
        adminRPCURL = window.__CONFIG__.adminRPCURL;
        chainId = Number(window.__CONFIG__.chainId);
    }

    // Replace localhost with current hostname for local development
    if (rpcURL.includes("localhost")) {
        rpcURL = rpcURL.replace("localhost", window.location.hostname);
    }
    if (adminRPCURL.includes("localhost")) {
        adminRPCURL = adminRPCURL.replace("localhost", window.location.hostname);
    }

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
const configPath = "/v1/admin/config";

// HTTP Methods
export async function POST(url: string, request: string, path: string) {
    return fetch(url + path, {
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
    return fetch(url + path, {
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

function pageHeightReq(page: number, height: number) {
    return JSON.stringify({ pageNumber: page, perPage: 10, height: height });
}

function validatorsReq(page: number, height: number, committee: number) {
    return JSON.stringify({ height: height, pageNumber: page, perPage: 1000, committee: committee });
}

// API Calls
export function Blocks(page: number, perPage: number = 10) {
    return POST(rpcURL, JSON.stringify({ pageNumber: page, perPage: perPage }), blocksPath);
}

export function Transactions(page: number, height: number) {
    return POST(rpcURL, pageHeightReq(page, height), txsByHeightPath);
}

// Función optimizada para obtener transacciones con paginación real
export async function getTransactionsWithRealPagination(page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
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

            // Ordenar por tiempo (más recientes primero)
            allTransactions.sort((a, b) => {
                const timeA = a.blockTime || a.time || a.timestamp || 0;
                const timeB = b.blockTime || b.time || b.timestamp || 0;
                return timeB - timeA;
            });

            // Aplicar paginación
            const paginatedTransactions = allTransactions.slice(startIndex, endIndex);

            return {
                results: paginatedTransactions,
                totalCount: totalTransactionCount,
                pageNumber: page,
                perPage: perPage,
                totalPages: Math.ceil(totalTransactionCount / perPage),
                hasMore: endIndex < totalTransactionCount
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
// Cache para el conteo total de transacciones
let totalTransactionCountCache: { count: number; timestamp: number } | null = null;
const CACHE_DURATION = 30000; // 30 segundos

export async function getTotalTransactionCount(): Promise<number> {
    try {
        // Verificar cache
        if (totalTransactionCountCache &&
            (Date.now() - totalTransactionCountCache.timestamp) < CACHE_DURATION) {
            return totalTransactionCountCache.count;
        }

        // Get information from the latest block to know the total number of transactions
        const latestBlocksResponse = await Blocks(1, 0);
        const latestBlock = latestBlocksResponse?.results?.[0] || latestBlocksResponse?.blocks?.[0];

        let totalCount = 0;

        if (latestBlock?.blockHeader?.totalTxs) {
            totalCount = latestBlock.blockHeader.totalTxs;
        } else {
            // Fallback: get transactions from multiple pages of blocks
            let currentPage = 1;
            const maxPages = 10; // Limit to avoid too many requests

            while (currentPage <= maxPages) {
                const blocksResponse = await Blocks(currentPage, 0);
                const blocks = blocksResponse?.results || blocksResponse?.blocks || [];

                if (!Array.isArray(blocks) || blocks.length === 0) break;

                for (const block of blocks) {
                    if (block.transactions && Array.isArray(block.transactions)) {
                        totalCount += block.transactions.length;
                    }
                }

                currentPage++;
            }
        }

        // Actualizar cache
        totalTransactionCountCache = {
            count: totalCount,
            timestamp: Date.now()
        };

        return totalCount;
    } catch (error) {
        console.error('Error getting total transaction count:', error);
        return totalTransactionCountCache?.count || 0;
    }
}

// new function to get transactions from multiple blocks
export async function AllTransactions(page: number, perPage: number = 10, filters?: {
    type?: string;
    fromDate?: string;
    toDate?: string;
    status?: string;
    address?: string;
    minAmount?: number;
    maxAmount?: number;
}) {
    try {
        // Obtener el conteo total de transacciones
        const totalTransactionCount = await getTotalTransactionCount();

        // Calcular cuántos bloques necesitamos obtener para cubrir la paginación
        // Asumimos un promedio de transacciones por bloque para optimizar
        const estimatedTxsPerBlock = 1; // Ajustar según la realidad de tu blockchain
        const blocksNeeded = Math.ceil((page * perPage) / estimatedTxsPerBlock) + 5; // Buffer extra

        // Obtener múltiples páginas de bloques para asegurar suficientes transacciones
        let allTransactions: any[] = [];
        let currentBlockPage = 1;
        const maxBlockPages = Math.min(blocksNeeded, 20); // Limitar para rendimiento

        while (currentBlockPage <= maxBlockPages && allTransactions.length < (page * perPage)) {
            const blocksResponse = await Blocks(currentBlockPage, 0);
            const blocks = blocksResponse?.results || blocksResponse?.blocks || blocksResponse?.list || [];

            if (!Array.isArray(blocks) || blocks.length === 0) break;

            for (const block of blocks) {
                if (block.transactions && Array.isArray(block.transactions)) {
                    // add block information to each transaction
                    const blockTransactions = block.transactions.map((tx: any) => ({
                        ...tx,
                        blockHeight: block.blockHeader?.height || block.height,
                        blockHash: block.blockHeader?.hash || block.hash,
                        blockTime: block.blockHeader?.time || block.time,
                        blockNumber: block.blockHeader?.height || block.height
                    }));
                    allTransactions = allTransactions.concat(blockTransactions);
                }
            }

            currentBlockPage++;
        }

        // apply filters if provided
        if (filters) {
            allTransactions = allTransactions.filter(tx => {
                // Filtro por tipo
                if (filters.type && filters.type !== 'All Types') {
                    const txType = tx.messageType || tx.type || 'send';
                    if (txType.toLowerCase() !== filters.type.toLowerCase()) {
                        return false;
                    }
                }

                // filter by address (sender or recipient)
                if (filters.address) {
                    const address = filters.address.toLowerCase();
                    const sender = (tx.sender || tx.from || '').toLowerCase();
                    const recipient = (tx.recipient || tx.to || '').toLowerCase();
                    const hash = (tx.txHash || tx.hash || '').toLowerCase();

                    if (!sender.includes(address) && !recipient.includes(address) && !hash.includes(address)) {
                        return false;
                    }
                }

                // filter by date range
                if (filters.fromDate || filters.toDate) {
                    const txTime = tx.blockTime || tx.time || tx.timestamp;
                    if (txTime) {
                        const txDate = new Date(txTime > 1e12 ? txTime / 1000 : txTime);

                        if (filters.fromDate) {
                            const fromDate = new Date(filters.fromDate);
                            if (txDate < fromDate) return false;
                        }

                        if (filters.toDate) {
                            const toDate = new Date(filters.toDate);
                            toDate.setHours(23, 59, 59, 999); // Incluir todo el día
                            if (txDate > toDate) return false;
                        }
                    }
                }

                // filter by amount range
                if (filters.minAmount !== undefined || filters.maxAmount !== undefined) {
                    const amount = tx.amount || tx.value || 0;

                    if (filters.minAmount !== undefined && amount < filters.minAmount) {
                        return false;
                    }

                    if (filters.maxAmount !== undefined && amount > filters.maxAmount) {
                        return false;
                    }
                }

                // filter by status
                if (filters.status && filters.status !== 'all') {
                    const txStatus = tx.status || 'success';
                    if (txStatus !== filters.status) {
                        return false;
                    }
                }

                return true;
            });
        }

        // Ordenar por tiempo (más recientes primero)
        allTransactions.sort((a, b) => {
            const timeA = a.blockTime || a.time || a.timestamp || 0;
            const timeB = b.blockTime || b.time || b.timestamp || 0;
            return timeB - timeA;
        });

        // Aplicar paginación
        const startIndex = (page - 1) * perPage;
        const endIndex = startIndex + perPage;
        const paginatedTransactions = allTransactions.slice(startIndex, endIndex);

        // Usar el conteo total real si no hay filtros, sino usar el conteo filtrado
        const finalTotalCount = filters ? allTransactions.length : totalTransactionCount;

        return {
            results: paginatedTransactions,
            totalCount: finalTotalCount,
            pageNumber: page,
            perPage: perPage,
            totalPages: Math.ceil(finalTotalCount / perPage),
            hasMore: endIndex < finalTotalCount
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

export function Committee(page: number, chain_id: number) {
    return POST(rpcURL, validatorsReq(page, 0, chain_id), validatorsPath);
}

export function DAO(height: number, _: number) {
    return POST(rpcURL, heightAndIDRequest(height, 131071), poolPath);
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

export function Orders(chain_id: number) {
    return POST(rpcURL, heightAndIDRequest(0, chain_id), ordersPath);
}

export function Order(chain_id: number, order_id: string, height: number = 0) {
    return POST(rpcURL, JSON.stringify({ chainId: chain_id, orderId: order_id, height: height }), orderPath);
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

export async function getCardData() {
    const cardData: any = {};
    cardData.blocks = await Blocks(1, 0);
    cardData.canopyCommittee = await Committee(1, chainId);
    cardData.supply = await Supply(0, 0);
    cardData.pool = await DAO(0, 0);
    cardData.params = await Params(0, 0);
    cardData.ecoParams = await EcoParams(0);
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
            return await Orders(committee || 1);
        case 7:
            return await Supply(0, 0);
        default:
            return null;
    }
}
