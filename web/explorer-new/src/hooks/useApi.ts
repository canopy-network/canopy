import { useQuery } from '@tanstack/react-query';
import {
    Blocks,
    Transactions,
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
    getTableData
} from '../lib/api';

// Query Keys
export const queryKeys = {
    blocks: (page: number) => ['blocks', page],
    transactions: (page: number, height: number) => ['transactions', page, height],
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
export const useBlocks = (page: number) => {
    return useQuery({
        queryKey: queryKeys.blocks(page),
        queryFn: () => Blocks(page, 0),
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

// Hooks for Orders
export const useOrders = (chainId: number) => {
    return useQuery({
        queryKey: queryKeys.orders(chainId),
        queryFn: () => Orders(chainId),
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
