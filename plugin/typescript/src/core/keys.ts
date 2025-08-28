/**
 * State key generation functions for the Canopy blockchain plugin
 * Key formats and prefixes maintain compatibility with the blockchain protocol
 */

import Long from 'long';
import { joinLenPrefix, formatUint64 } from '../utils/proto-utils.ts';

// Key prefixes for state storage
export const ACCOUNT_PREFIX = Buffer.from([1]); // store key prefix for accounts
export const POOL_PREFIX = Buffer.from([2]); // store key prefix for pools
export const PARAMS_PREFIX = Buffer.from([7]); // store key prefix for governance parameters

// Type definitions for addresses and amounts
type Address = Buffer | Uint8Array | string | readonly number[];
type ChainId = number | Long | string;
type Amount = number | Long;

/**
 * Generate state database key for an account
 * Generate state database key for an account
 * @param address - Account address (must be 20 bytes when converted to Buffer)
 * @returns Buffer containing the state key
 */
export function keyForAccount(address: Address): Buffer {
    const addressBuffer = Buffer.isBuffer(address) ? address : Buffer.from(address);
    return joinLenPrefix(ACCOUNT_PREFIX, addressBuffer);
}

/**
 * Generate state database key for governance controlled fee parameters
 * Generate state database key for governance controlled fee parameters
 * @returns Buffer containing the fee parameters key
 */
export function keyForFeeParams(): Buffer {
    const suffix = Buffer.from('/f/');
    return joinLenPrefix(PARAMS_PREFIX, suffix);
}

/**
 * Generate state database key for fee pool
 * Generate state database key for fee pool
 * @param chainId - Chain identifier
 * @returns Buffer containing the fee pool key
 */
export function keyForFeePool(chainId: ChainId): Buffer {
    const chainIdBytes = formatUint64(chainId);
    return joinLenPrefix(POOL_PREFIX, chainIdBytes);
}

/**
 * Validate that an address is exactly 20 bytes
 * Used in transaction validation
 * @param address - Address to validate
 * @returns true if address is valid (exactly 20 bytes)
 */
export function validateAddress(address: unknown): address is Address {
    if (!address) {
        return false;
    }

    try {
        const addrBuffer = Buffer.isBuffer(address)
            ? address
            : Buffer.from(address as BufferEncoding);
        return addrBuffer.length === 20;
    } catch {
        return false;
    }
}

/**
 * Validate that an amount is greater than 0
 * Used in transaction validation
 * @param amount - Amount to validate
 * @returns true if amount is valid (greater than 0)
 */
export function validateAmount(amount: unknown): amount is Amount {
    if (Long.isLong(amount)) {
        return amount.greaterThan(0);
    }
    if (typeof amount === 'number') {
        return Number.isFinite(amount) && amount > 0;
    }
    if (typeof amount === 'string') {
        try {
            const longAmount = Long.fromString(amount);
            return longAmount.greaterThan(0);
        } catch {
            return false;
        }
    }
    // Handle objects that might be Long-like
    if (amount && typeof amount === 'object' && 'high' in amount && 'low' in amount) {
        try {
            const longAmount = Long.fromBits(
                (amount as any).low,
                (amount as any).high,
                (amount as any).unsigned || false
            );
            return longAmount.greaterThan(0);
        } catch {
            return false;
        }
    }
    return false;
}

/**
 * Type guard to check if a value is a valid address
 */
export function isValidAddress(value: unknown): value is Address {
    return validateAddress(value);
}

/**
 * Type guard to check if a value is a valid amount
 */
export function isValidAmount(value: unknown): value is Amount {
    return validateAmount(value);
}

/**
 * Convert various address formats to Buffer
 * @param address - Address in various formats
 * @returns Buffer representation of the address
 * @throws Error if address cannot be converted or is invalid length
 */
export function normalizeAddress(address: Address): Buffer {
    if (!validateAddress(address)) {
        throw new Error('Invalid address: must be exactly 20 bytes');
    }

    return Buffer.isBuffer(address) ? address : Buffer.from(address);
}

/**
 * Convert various amount formats to Long for safe arithmetic
 * @param amount - Amount in various formats
 * @returns Long representation of the amount
 * @throws Error if amount cannot be converted or is invalid
 */
export function normalizeAmount(amount: Amount): Long {
    if (!validateAmount(amount)) {
        throw new Error('Invalid amount: must be greater than 0');
    }

    if (Long.isLong(amount)) {
        return amount;
    }

    if (typeof amount === 'number') {
        return Long.fromNumber(amount);
    }

    return Long.fromString(String(amount));
}

// Export types for external use
export type { Address, ChainId, Amount };
