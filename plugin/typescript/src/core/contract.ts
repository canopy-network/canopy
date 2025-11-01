/**
 * Contract implementation for Canopy blockchain plugin
 * Handles send transaction validation and execution with state management
 */

import Long from 'long';
import * as protobuf from '../proto/index.js';
import { marshal, unmarshal, fromAny } from '../utils/proto-utils.ts';
import {
    keyForAccount,
    keyForFeePool,
    keyForFeeParams,
    validateAddress,
    validateAmount,
    normalizeAddress,
    normalizeAmount,
    type Address,
    type ChainId,
    type Amount
} from './keys.ts';
import {
    errInvalidAddress,
    errInvalidAmount,
    errInsufficientFunds,
    errInvalidMessageCast,
    errTxFeeBelowStateLimit,
    errFromAny,
    errMarshal,
    errUnmarshal,
    type PluginError
} from '../utils/errors.ts';
import type Config from '../config/index.ts';

// Type definitions for contract interfaces
interface SocketClientPlugin {
    stateRead(contract: Contract, request: StateReadRequest): Promise<StateReadResponse>;
    stateWrite(contract: Contract, request: StateWriteRequest): Promise<StateWriteResponse>;
}

interface ContractOptions {
    readonly config?: Config;
    readonly fsmConfig?: unknown;
    readonly plugin?: SocketClientPlugin;
    readonly fsmId?: Long | string;
}

// Protobuf message interfaces
interface ProtoError {
    readonly code: number;
    readonly module: string;
    readonly msg: string;
}

interface TransactionRequest {
    readonly tx: {
        readonly fee: number | Long;
        readonly msg: {
            readonly typeUrl?: string;
            readonly type_url?: string;
            readonly value?: Uint8Array;
        };
    };
}

interface MessageSend {
    readonly fromAddress: Address;
    readonly toAddress: Address;
    readonly amount: Amount;
}

interface Account {
    readonly address?: Address;
    readonly amount: number | Long | string;
}

interface Pool {
    readonly id?: ChainId;
    readonly amount: number | Long | string;
}

interface FeeParams {
    readonly sendFee: number | Long;
}

// State operation interfaces
interface StateReadRequest {
    readonly keys: readonly StateKeyQuery[];
}

interface StateKeyQuery {
    readonly queryId: Long;
    readonly key: Buffer;
}

interface StateReadResponse {
    readonly error?: ProtoError | null;
    readonly results: readonly StateQueryResult[];
}

interface StateQueryResult {
    readonly queryId: Long;
    readonly entries: readonly StateEntry[];
}

interface StateEntry {
    readonly key?: Buffer;
    readonly value?: Buffer;
}

interface StateWriteRequest {
    readonly sets?: readonly StateSetOperation[];
    readonly deletes?: readonly StateDeleteOperation[];
}

interface StateSetOperation {
    readonly key: Buffer;
    readonly value: Buffer;
}

interface StateDeleteOperation {
    readonly key: Buffer;
}

interface StateWriteResponse {
    readonly error?: ProtoError | null;
}

// Response interfaces
interface GenesisResponse {
    readonly error: ProtoError | null;
}

interface BeginBlockResponse {
    readonly error: ProtoError | null;
}

interface EndBlockResponse {
    readonly error: ProtoError | null;
}

interface CheckTxResponse {
    readonly recipient?: Address;
    readonly authorizedSigners?: readonly Address[];
    readonly error: ProtoError | null;
}

interface DeliverTxResponse {
    readonly error: ProtoError | null;
}

/**
 * Plugin configuration for the Canopy contract
 */
export const CONTRACT_CONFIG = {
    name: 'send',
    id: 1,
    version: 1,
    supportedTransactions: ['send']
} as const;

/**
 * Contract class for handling Canopy blockchain transactions
 * Handles blockchain transaction validation and execution
 */
export class Contract {
    public readonly config?: Config;
    public readonly fsmConfig?: unknown;
    public readonly plugin?: SocketClientPlugin;
    public readonly fsmId?: Long | string;

    constructor(options: ContractOptions = {}) {
        this.config = options.config;
        this.fsmConfig = options.fsmConfig;
        this.plugin = options.plugin;
        this.fsmId = options.fsmId;
    }

    /**
     * Genesis implementation - initializes the contract state
     * @param _request - Genesis request (unused)
     * @returns Genesis response with no error
     */
    genesis(_request: unknown): GenesisResponse {
        return { error: null };
    }

    /**
     * BeginBlock implementation - called at the start of each block
     * @param _request - Begin block request (unused)
     * @returns Begin block response with no error
     */
    beginBlock(_request: unknown): BeginBlockResponse {
        return { error: null };
    }

    /**
     * CheckTx - validate transaction without state changes
     * @param request - Transaction validation request
     * @returns Promise resolving to validation result
     */
    async checkTx(request: TransactionRequest): Promise<CheckTxResponse> {
        try {
            if (!this.plugin || !this.config) {
                return {
                    error: { code: 1, module: 'contract', msg: 'Plugin or config not initialized' }
                };
            }

            // Validate fee against state parameters
          const feeParamsResponse = await this.plugin.stateRead(this, {
            keys: [
              {
                queryId: Long.fromNumber(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER), true),
                key: keyForFeeParams()
              }
            ]
          });

            if (feeParamsResponse.error) {
                return { error: feeParamsResponse.error };
            }

            // Convert bytes into fee parameters
            const feeParamsBytes = feeParamsResponse.results[0]?.entries[0]?.value;
            if (!feeParamsBytes) {
                return { error: { code: 1, module: 'contract', msg: 'Fee parameters not found' } };
            }

            const minFees = unmarshal<FeeParams>(protobuf.types.FeeParams, feeParamsBytes);

            if (!minFees) {
                return {
                    error: errUnmarshal(new Error('Failed to decode fee parameters')).toProtoError()
                };
            }

            // Check for minimum fee
            let requestFee: Long;
            try {
                requestFee = Long.fromValue(normalizeAmount(request.tx.fee), true);
            } catch (error) {
                throw new Error(`Failed to normalize request.tx.fee: ${error.message}. Value: ${JSON.stringify(request.tx.fee)}`);
            }

            let minSendFee: Long;
            try {
                minSendFee = Long.fromValue(normalizeAmount(minFees.sendFee), true);
            } catch (error) {
                throw new Error(`Failed to normalize minFees.sendFee: ${error.message}. Value: ${JSON.stringify(minFees.sendFee)}`);
            }

            if (requestFee.lt(minSendFee)) {
                return { error: errTxFeeBelowStateLimit().toProtoError() };
            }

            // Get the message from protobuf Any type
            let msg: MessageSend;
            try {
                msg = fromAny(request.tx.msg) as MessageSend;
            } catch (err) {
                const error = err instanceof Error ? err : new Error(String(err));
                return { error: errFromAny(error).toProtoError() };
            }

            // Handle the message based on type
            if (this.isMessageSend(msg)) {
                const val = this.checkMessageSend(msg);
                return val;
            } else {
                return { error: errInvalidMessageCast().toProtoError() };
            }
        } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            return { error: errUnmarshal(error).toProtoError() };
        }
    }

    /**
     * DeliverTx - execute transaction with state changes
     * @param request - Transaction execution request
     * @returns Promise resolving to execution result
     */
    async deliverTx(request: TransactionRequest): Promise<DeliverTxResponse> {
        try {
            // Get the message from protobuf Any type
            let msg: MessageSend;
            try {
                msg = fromAny(request.tx.msg) as MessageSend;
            } catch (err) {
                const error = err instanceof Error ? err : new Error(String(err));
                return { error: errFromAny(error).toProtoError() };
            }

            // Handle the message based on type
            if (this.isMessageSend(msg)) {
                try {
                    const response = await this.deliverMessageSend(msg, request.tx.fee);
                    return response;
                } catch (error) {
                    const err = error instanceof Error ? error : new Error(String(error));
                    return { error: { code: 1, module: 'contract', msg: err.message } };
                }
            } else {
                return { error: errInvalidMessageCast().toProtoError() };
            }
        } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            return { error: errUnmarshal(error).toProtoError() };
        }
    }

    /**
     * EndBlock implementation - called at the end of each block
     * @param _request - End block request (unused)
     * @returns End block response with no error
     */
    endBlock(_request: unknown): EndBlockResponse {
        return { error: null };
    }

    /**
     * Type guard to check if message is MessageSend
     * @param msg - Message to check
     * @returns true if message is MessageSend
     */
    private isMessageSend(msg: unknown): msg is MessageSend {
        const message = msg as Partial<MessageSend>;
        return (
            message.fromAddress !== undefined &&
            message.toAddress !== undefined &&
            message.amount !== undefined
        );
    }

    /**
     * Validate MessageSend without state changes
     * @param msg - MessageSend to validate
     * @returns Validation result with authorized signers
     */
    private checkMessageSend(msg: MessageSend): CheckTxResponse {
        // Check sender address (must be exactly 20 bytes)
        if (!validateAddress(msg.fromAddress)) {
            return { error: errInvalidAddress().toProtoError() };
        }

        // Check recipient address (must be exactly 20 bytes)
        if (!validateAddress(msg.toAddress)) {
            return { error: errInvalidAddress().toProtoError() };
        }

        // Check amount (must be greater than 0)
        if (!validateAmount(msg.amount)) {
            return { error: errInvalidAmount().toProtoError() };
        }

        // Return authorized signers (sender must sign)
        return {
            recipient: msg.toAddress,
            authorizedSigners: [msg.fromAddress],
            error: null
        };
    }

    /**
     * Generate random query IDs for batch state operations
     * @returns Object containing fromQueryId, toQueryId, and feeQueryId
     */
    private generateQueryIds(): { fromQueryId: Long; toQueryId: Long; feeQueryId: Long } {
        return {
            fromQueryId: Long.fromNumber(
                Math.floor(Math.random() * Number.MAX_SAFE_INTEGER),
                true
            ),
            toQueryId: Long.fromNumber(
                Math.floor(Math.random() * Number.MAX_SAFE_INTEGER),
                true
            ),
            feeQueryId: Long.fromNumber(
                Math.floor(Math.random() * Number.MAX_SAFE_INTEGER),
                true
            )
        };
    }

    /**
     * Read accounts and fee pool from state
     * @param msg - MessageSend containing addresses
     * @returns Promise resolving to account and fee pool bytes
     */
    private async readAccountsAndFeePool(msg: MessageSend): Promise<{
        fromBytes: Buffer | null;
        toBytes: Buffer | null;
        feePoolBytes: Buffer | null;
    }> {
        if (!this.plugin || !this.config) {
            throw new Error('Plugin or config not initialized');
        }

        const { fromQueryId, toQueryId, feeQueryId } = this.generateQueryIds();

        // Calculate state keys
        const fromKey = keyForAccount(msg.fromAddress);
        const toKey = keyForAccount(msg.toAddress);
        const feePoolKey = keyForFeePool(this.config.chainId);

        // Batch read accounts and fee pool from state
        const response = await this.plugin.stateRead(this, {
            keys: [
                { queryId: feeQueryId, key: feePoolKey },
                { queryId: fromQueryId, key: fromKey },
                { queryId: toQueryId, key: toKey }
            ]
        });

        if (response.error) {
            throw new Error(`State read error: ${response.error.msg}`);
        }

        // Parse response results by query ID
        let fromBytes: Buffer | null = null;
        let toBytes: Buffer | null = null;
        let feePoolBytes: Buffer | null = null;

        for (const result of response.results) {
            switch (true) {
                case result.queryId.eq(fromQueryId):
                    fromBytes = result.entries[0]?.value ?? null;
                    break;
                case result.queryId.eq(toQueryId):
                    toBytes = result.entries[0]?.value ?? null;
                    break;
                case result.queryId.eq(feeQueryId):
                    feePoolBytes = result.entries[0]?.value ?? null;
                    break;
            }
        }

        return { fromBytes, toBytes, feePoolBytes };
    }

    /**
     * Unmarshal account and pool data from state bytes
     * @param fromBytes - From account bytes
     * @param toBytes - To account bytes
     * @param feePoolBytes - Fee pool bytes
     * @param msg - MessageSend containing addresses
     * @returns Unmarshaled accounts, pool, and normalized fromAmount
     */
    private unmarshalAccountsAndPool(
        fromBytes: Buffer | null,
        toBytes: Buffer | null,
        feePoolBytes: Buffer | null,
        msg: MessageSend
    ): { fromAccount: Account; toAccount: Account; feePool: Pool; fromAmount: Long } {
        const fromAccount: Account = unmarshal<Account>(protobuf.types.Account, fromBytes);

        let toAccount: Account;
        try {
            toAccount = unmarshal<Account>(protobuf.types.Account, toBytes) || {
                address: msg.toAddress,
                amount: Long.UZERO
            };
        } catch {
            toAccount = {
                address: msg.toAddress,
                amount: Long.UZERO
            };
        }

        let feePool: Pool;
        try {
            feePool = unmarshal<Pool>(protobuf.types.Pool, feePoolBytes) || {
                amount: Long.UZERO
            };
        } catch {
            feePool = {
                amount: Long.UZERO
            };
        }

        // Assume fromAccount.amount is already a Long from protobuf unmarshaling
        const fromAmount: Long = fromAccount.amount as Long || Long.UZERO;

        return { fromAccount, toAccount, feePool, fromAmount };
    }

    /**
     * Calculate updated account balances for sender and recipient
     * @param fromAmount - Sender's current balance
     * @param toAccount - Recipient account
     * @param msg - MessageSend containing addresses and amount
     * @param transactionFee - Transaction fee
     * @param fromKey - Sender's state key
     * @param toKey - Recipient's state key
     * @returns Updated from and to accounts
     */
    private calculateUpdatedBalances(
        fromAmount: Long,
        toAccount: Account,
        msg: MessageSend,
        transactionFee: Long,
        fromKey: Buffer,
        toKey: Buffer
    ): { updatedFromAccount: Account; updatedToAccount: Account } {
        const messageAmount = Long.fromValue(msg.amount, true);
        const amountToDeduct = messageAmount.add(transactionFee);

        const updatedFromAccount: Account = {
            address: normalizeAddress(msg.fromAddress),
            amount: fromAmount.sub(amountToDeduct)
        };

        // Handle self-transfer optimization
        const isSelfTransfer = Buffer.compare(fromKey, toKey) === 0;
        let updatedToAccount: Account;

        if (isSelfTransfer) {
            updatedToAccount = {
                address: normalizeAddress(msg.toAddress),
                amount: fromAmount.sub(transactionFee) // Only deduct fee for self-transfer
            };
        } else {
            // Update balances - assume toAccount.amount is already a Long from protobuf
            updatedToAccount = {
                address: normalizeAddress(msg.toAddress),
                amount: (toAccount.amount as Long || Long.UZERO).add(messageAmount)
            };
        }

        return { updatedFromAccount, updatedToAccount };
    }

    /**
     * Prepare state write operations for accounts and fee pool
     * @param updatedFromAccount - Updated sender account
     * @param updatedToAccount - Updated recipient account
     * @param updatedFeePool - Updated fee pool
     * @param keys - State keys for accounts and fee pool
     * @param isSelfTransfer - Whether this is a self-transfer
     * @returns State operation arrays for sets and deletes
     */
    private prepareStateOperations(
        updatedFromAccount: Account,
        updatedToAccount: Account,
        updatedFeePool: Pool,
        keys: { fromKey: Buffer; toKey: Buffer; feePoolKey: Buffer },
        isSelfTransfer: boolean
    ): { sets: StateSetOperation[]; deletes: StateDeleteOperation[] } {
        // Marshal updated data
        const updatedFromBytes = marshal(protobuf.types.Account, updatedFromAccount);
        const updatedToBytes = marshal(protobuf.types.Account, updatedToAccount);
        const updatedFeePoolBytes = marshal(protobuf.types.Pool, updatedFeePool);

        // Prepare state write operations
        const sets: StateSetOperation[] = [{ key: keys.feePoolKey, value: updatedFeePoolBytes }];
        const deletes: StateDeleteOperation[] = [];

        // Handle account deletion when balance reaches zero
        if ((updatedFromAccount.amount as Long).eq(Long.UZERO) && !isSelfTransfer) {
            deletes.push({ key: keys.fromKey });
        } else {
            sets.push({ key: keys.fromKey, value: updatedFromBytes });
        }

        if (!isSelfTransfer) {
            sets.push({ key: keys.toKey, value: updatedToBytes });
        }

        return { sets, deletes };
    }

    /**
     * Process MessageSend with state changes
     * Execute a send transaction with state updates
     * @param msg - MessageSend to execute
     * @param fee - Transaction fee
     * @returns Promise resolving to execution result
     */
    private async deliverMessageSend(
        msg: MessageSend,
        fee: number | Long
    ): Promise<DeliverTxResponse> {
        try {
            const transactionFee = Long.fromValue(normalizeAmount(fee), true);

            if (!this.plugin || !this.config) {
                return {
                    error: { code: 1, module: 'contract', msg: 'Plugin or config not initialized' }
                };
            }

            // Read accounts and fee pool from state
            const { fromBytes, toBytes, feePoolBytes } = await this.readAccountsAndFeePool(msg);

            // Unmarshal account and pool data
            const { fromAccount, toAccount, feePool, fromAmount } = this.unmarshalAccountsAndPool(
                fromBytes,
                toBytes,
                feePoolBytes,
                msg
            );

            // Calculate amount to deduct (message amount + fee)
            const messageAmount = Long.fromValue(msg.amount, true);
            const amountToDeduct = messageAmount.add(transactionFee);

            // Check sufficient funds
            if (fromAmount.lt(amountToDeduct)) {
                return { error: errInsufficientFunds().toProtoError() };
            }

            // Calculate state keys for balance calculations
            const fromKey = keyForAccount(msg.fromAddress);
            const toKey = keyForAccount(msg.toAddress);
            const feePoolKey = keyForFeePool(this.config.chainId);

            // Calculate updated balances
            const { updatedFromAccount, updatedToAccount } = this.calculateUpdatedBalances(
                fromAmount,
                toAccount,
                msg,
                transactionFee,
                fromKey,
                toKey
            );

            // Update fee pool - assume feePool.amount is already a Long from protobuf
            const updatedFeePool: Pool = {
                id: this.config.chainId,
                amount: (feePool.amount as Long || Long.UZERO).add(transactionFee)
            };

            // Check if this is a self-transfer
            const isSelfTransfer = Buffer.compare(fromKey, toKey) === 0;

            // Prepare state write operations
            const { sets, deletes } = this.prepareStateOperations(
                updatedFromAccount,
                updatedToAccount,
                updatedFeePool,
                { fromKey, toKey, feePoolKey },
                isSelfTransfer
            );

            // Execute batch state write
            const writeResponse = await this.plugin.stateWrite(this, {
                sets,
                deletes
            });

            return { error: writeResponse.error || null };
        } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            return { error: errMarshal(error).toProtoError() };
        }
    }
}

// Export types for external use
export type {
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
