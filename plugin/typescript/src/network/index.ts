/**
 * Network module exports
 * Centralized exports for network communication functionality
 */

// Socket client implementation
import { SocketClient } from './socket-client.ts';
import type {
    SocketClientOptions,
    PendingRequest,
    AsyncChannelResult,
    MessageRouting,
    ContractParams
} from './socket-client.ts';

// Re-export all network utilities
export {
    SocketClient
};

// Type exports
export type {
    SocketClientOptions,
    PendingRequest,
    AsyncChannelResult,
    MessageRouting,
    ContractParams
};

// Default export for backward compatibility
export default SocketClient;