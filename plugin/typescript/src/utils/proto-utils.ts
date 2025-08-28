/**
 * Protobuf utility functions for marshaling and unmarshaling
 * Provides type-safe protobuf operations for the Canopy blockchain plugin
 */

import Long from 'long';
import * as protobuf from '../proto/index.js';

// Protobuf message interface for type safety
interface ProtobufMessage {
  create(obj: unknown): unknown;
  encode(obj: unknown): { finish(): Uint8Array };
  decode(bytes: Uint8Array): unknown;
}

// Any message interface (protobuf Any type)
interface AnyMessage {
  readonly typeUrl?: string;
  readonly type_url?: string;
  readonly value?: Uint8Array;
}

// Known message types for fromAny function
interface MessageSend {
  readonly fromAddress: Uint8Array;
  readonly toAddress: Uint8Array;
  readonly amount: number | Long | string;
}

interface Transaction {
  readonly fee: number | Long;
  readonly msg: AnyMessage;
}

(BigInt.prototype as any).toJSON = function() {
    return this.toString();
};

type KnownMessageTypes = MessageSend | Transaction;

/**
 * Marshal object to protobuf bytes
 * Marshal object to protobuf bytes
 */
export function marshal<T extends object>(messageType: ProtobufMessage, data: T): Buffer {
  try {
    const message = messageType.create(data);
    return Buffer.from(messageType.encode(message).finish());
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    throw new Error(`Marshal failed: ${error.message}`);
  }
}

/**
 * Unmarshal bytes to protobuf message
 * Unmarshal bytes to protobuf message
 */
export function unmarshal<T>(messageType: ProtobufMessage, bytes?: Uint8Array | null): T | null {
  try {
    if (!bytes || bytes.length === 0) {
      return null;
    }
    return messageType.decode(bytes) as T;
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    throw new Error(`Unmarshal failed: ${error.message}`);
  }
}

/**
 * Convert protobuf Any type to concrete message
 */
export function fromAny(anyMessage: AnyMessage): KnownMessageTypes {
  try {
    if (!anyMessage) {
      throw new Error('Any message is null or undefined');
    }

    // Handle both typeUrl and type_url field names
    const typeUrl = anyMessage.typeUrl ?? anyMessage.type_url;
    const value = anyMessage.value;

    if (!typeUrl) {
      throw new Error('Any message missing type URL');
    }

    if (!value) {
      throw new Error('Any message missing value');
    }

    // Extract message type from URL (e.g., "types.MessageSend" from full URL)
    const typeName = typeUrl.split('/').pop() ?? typeUrl;

    // Map known message types
    switch (typeName) {
      case 'MessageSend':
      case 'types.MessageSend': {
        const msg = unmarshal<MessageSend>(protobuf.types.MessageSend, value);
        if (!msg) {
          throw new Error('Failed to unmarshal MessageSend');
        }
        return {
          ...msg,
          amount: Long.isLong(msg.amount) ? msg.amount : Long.fromString(msg.amount.toString()),
        };
      }
      case 'Transaction':
      case 'types.Transaction': {
        const msg = unmarshal<Transaction>(protobuf.types.Transaction, value);
        if (!msg) {
          throw new Error('Failed to unmarshal Transaction');
        }
        return msg;
      }
      default:
        throw new Error(`Unknown message type in Any: ${typeName}`);
    }
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    throw new Error(`FromAny failed: ${error.message}`);
  }
}

/**
 * Join byte arrays with length prefixes
 * Join byte arrays with length prefixes for state key generation
 */
export function joinLenPrefix(...items: readonly (Uint8Array | Buffer | null | undefined)[]): Buffer {
  // Calculate total length first
  let totalLen = 0;
  for (const item of items) {
    if (item && item.length > 0) {
      totalLen += 1 + item.length; // 1 byte for length + item length
    }
  }

  // Create result buffer
  const result = Buffer.allocUnsafe(totalLen);
  let offset = 0;

  // Append each item with length prefix
  for (const item of items) {
    if (!item || item.length === 0) {
      continue;
    }

    // Write length byte
    result.writeUInt8(item.length, offset);
    offset += 1;

    // Write item bytes
    if (Buffer.isBuffer(item)) {
      item.copy(result, offset);
    } else {
      Buffer.from(item).copy(result, offset);
    }
    offset += item.length;
  }

  return result;
}

/**
 * Format uint64 as big-endian bytes
 * Format uint64 as big-endian bytes for state storage
 */
export function formatUint64(value: Long | number | string): Buffer {
  const buffer = Buffer.allocUnsafe(8);

  if (Long.isLong(value)) {
    // Convert Long to BigInt for buffer operations
    const bigIntValue = BigInt(value.toString());
    buffer.writeBigUInt64BE(bigIntValue, 0);
  } else {
    buffer.writeBigUInt64BE(BigInt(value), 0);
  }

  return buffer;
}

// Export types for use in other modules
export type {
  ProtobufMessage,
  AnyMessage,
  MessageSend,
  Transaction,
  KnownMessageTypes,
};
