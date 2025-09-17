// API Configuration
let rpcURL = "http://localhost:50002"; // default value for the RPC URL
let adminRPCURL = "http://localhost:50003"; // default Admin RPC URL
let chainId = 1; // default chain id

if (typeof window !== "undefined") {
    if (window.__CONFIG__) {
        rpcURL = window.__CONFIG__.rpcURL;
        adminRPCURL = window.__CONFIG__.adminRPCURL;
        chainId = Number(window.__CONFIG__.chainId);
    }
    rpcURL = rpcURL.replace("localhost", window.location.hostname);
    adminRPCURL = adminRPCURL.replace("localhost", window.location.hostname);
    console.log(rpcURL);
} else {
    console.log("config undefined");
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
export function Blocks(page: number, _: number) {
    return POST(rpcURL, pageHeightReq(page, 0), blocksPath);
}

export function Transactions(page: number, height: number) {
    return POST(rpcURL, pageHeightReq(page, height), txsByHeightPath);
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
    let result: any = {};
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
    let cardData: any = {};
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
