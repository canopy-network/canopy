/**
 * Core module exports
 * Centralized exports for core blockchain functionality
 */

// Contract implementation
import { Contract, CONTRACT_CONFIG } from './contract.ts';

// Key generation and validation utilities
import {
    ACCOUNT_PREFIX,
    POOL_PREFIX,
    PARAMS_PREFIX,
    keyForAccount,
    keyForFeeParams,
    keyForFeePool,
    validateAddress,
    validateAmount,
    isValidAddress,
    isValidAmount,
    normalizeAddress,
    normalizeAmount,
    type Address,
    type ChainId,
    type Amount
} from './keys.ts';

// Re-export contract types
import type {
    ContractOptions,
    SocketClientPlugin,
    TransactionRequest,
    MessageSend,
    Account,
    Pool,
    FeeParams,
    CheckTxResponse,
    DeliverTxResponse,
    StateReadRequest,
    StateWriteRequest
} from './contract.ts';

// Named exports
export {
    // Contract
    Contract,
    CONTRACT_CONFIG,

    // Key prefixes (as const assertions for better type safety)
    ACCOUNT_PREFIX,
    POOL_PREFIX,
    PARAMS_PREFIX,

    // Key generation functions
    keyForAccount,
    keyForFeeParams,
    keyForFeePool,

    // Validation functions
    validateAddress,
    validateAmount,
    isValidAddress,
    isValidAmount,

    // Normalization functions
    normalizeAddress,
    normalizeAmount
};

// Type exports
export type {
    // Core types
    Address,
    ChainId,
    Amount,

    // Contract types
    ContractOptions,
    SocketClientPlugin,
    TransactionRequest,
    MessageSend,
    Account,
    Pool,
    FeeParams,
    CheckTxResponse,
    DeliverTxResponse,
    StateReadRequest,
    StateWriteRequest
};

// Default export for backward compatibility
export default {
    Contract,
    CONTRACT_CONFIG,
    ACCOUNT_PREFIX,
    POOL_PREFIX,
    PARAMS_PREFIX,
    keyForAccount,
    keyForFeeParams,
    keyForFeePool,
    validateAddress,
    validateAmount,
    isValidAddress,
    isValidAmount,
    normalizeAddress,
    normalizeAmount
} as const;
