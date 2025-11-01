/**
 * Utils module exports
 * Centralized exports for all utility functions and classes
 */

// Error handling
import {
    PluginError,
    newError,
    errPluginTimeout,
    errMarshal,
    errUnmarshal,
    errFailedPluginRead,
    errFailedPluginWrite,
    errInvalidPluginRespId,
    errUnexpectedFSMToPlugin,
    errInvalidFSMToPluginMessage,
    errInsufficientFunds,
    errFromAny,
    errInvalidMessageCast,
    errInvalidAddress,
    errInvalidAmount,
    errTxFeeBelowStateLimit,
    type ErrorFactory
} from './errors.ts';

// Logging
import Logger from './logger.ts';

// Protobuf utilities
import { marshal, unmarshal, fromAny, joinLenPrefix, formatUint64 } from './proto-utils.ts';

// Re-export all utilities with proper typing
export {
    // Error classes and functions
    PluginError,
    newError,
    type ErrorFactory,

    // Error factory functions
    errPluginTimeout,
    errMarshal,
    errUnmarshal,
    errFailedPluginRead,
    errFailedPluginWrite,
    errInvalidPluginRespId,
    errUnexpectedFSMToPlugin,
    errInvalidFSMToPluginMessage,
    errInsufficientFunds,
    errFromAny,
    errInvalidMessageCast,
    errInvalidAddress,
    errInvalidAmount,
    errTxFeeBelowStateLimit,

    // Logger
    Logger,

    // Protobuf utilities
    marshal,
    unmarshal,
    fromAny,
    joinLenPrefix,
    formatUint64
};

// Default export for backward compatibility
export default {
    PluginError,
    newError,
    errPluginTimeout,
    errMarshal,
    errUnmarshal,
    errFailedPluginRead,
    errFailedPluginWrite,
    errInvalidPluginRespId,
    errUnexpectedFSMToPlugin,
    errInvalidFSMToPluginMessage,
    errInsufficientFunds,
    errFromAny,
    errInvalidMessageCast,
    errInvalidAddress,
    errInvalidAmount,
    errTxFeeBelowStateLimit,
    Logger,
    marshal,
    unmarshal,
    fromAny,
    joinLenPrefix,
    formatUint64
} as const;
