// API Response Types

export interface BlockHeader {
    height: number;
    hash: string;
    time: number;
    numTxs: string;
    totalTxs: string;
    proposerAddress: string;
}

export interface Block {
    blockHeader: BlockHeader;
}

export interface Transaction {
    sender: string;
    recipient: string;
    messageType: string;
    height: number;
    index: number;
    txHash: string;
    fee: number;
    sequence: number;
}

export interface Account {
    address: string;
    amount: number;
}

export interface Validator {
    address: string;
    publicKey: string;
    committees: string;
    netAddress: string;
    stakedAmount: number;
    maxPausedHeight: number;
    unstakingHeight: number;
    output: string;
    delegate: boolean;
    compound: boolean;
}

export interface Order {
    Id: string;
    Chain: string;
    Data: string;
    AmountForSale: number;
    Rate: string;
    RequestedAmount: number;
    SellerReceiveAddress: string;
    SellersSendAddress: string;
    BuyerSendAddress: string;
    Status: string;
    BuyerReceiveAddress: string;
    BuyerChainDeadline: number;
}

export interface PaginatedResponse<T> {
    pageNumber: number;
    perPage: number;
    results: T[];
    type: string;
    count: number;
    totalPages: number;
    totalCount: number;
}

export interface Supply {
    totalSupply: number;
    stakedSupply: number;
    delegateSupply: number;
}

export interface Params {
    consensus: Record<string, any>;
    validator: Record<string, any>;
    fee: Record<string, any>;
    governance: Record<string, any>;
}

export interface EcoParams {
    chainId: number;
    params: Record<string, any>;
}

export interface Pool {
    id: number;
    data: any;
}

export interface Config {
    networkId: string;
    chainId: number;
    rpcURL: string;
    adminRPCURL: string;
}

// Specific response types
export type BlocksResponse = PaginatedResponse<Block>;
export type TransactionsResponse = PaginatedResponse<Transaction>;
export type AccountsResponse = PaginatedResponse<Account>;
export type ValidatorsResponse = PaginatedResponse<Validator>;
export type OrdersResponse = Order[];

// Card data type
export interface CardData {
    blocks: BlocksResponse;
    canopyCommittee: ValidatorsResponse;
    supply: Supply;
    pool: Pool;
    params: Params;
    ecoParams: EcoParams;
}

// Modal data type
export interface ModalData {
    block?: Block;
    validator?: Validator;
    account?: Account;
    sent_transactions?: TransactionsResponse;
    rec_transactions?: TransactionsResponse;
}
