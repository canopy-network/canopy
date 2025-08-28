/**
 * Error handling for the Canopy blockchain plugin
 * Provides structured error types with codes for blockchain operations
 */

const DEFAULT_MODULE = 'plugin' as const;

/** Error code constants for plugin operations */
const PluginErrorCode = {
  PLUGIN_TIMEOUT: 1,
  MARSHAL: 2,
  UNMARSHAL: 3,
  FAILED_PLUGIN_READ: 4,
  FAILED_PLUGIN_WRITE: 5,
  INVALID_PLUGIN_RESP_ID: 6,
  UNEXPECTED_FSM_TO_PLUGIN: 7,
  INVALID_FSM_TO_PLUGIN_MESSAGE: 8,
  INSUFFICIENT_FUNDS: 9,
  FROM_ANY: 10,
  INVALID_MESSAGE_CAST: 11,
  INVALID_ADDRESS: 12,
  INVALID_AMOUNT: 13,
  TX_FEE_BELOW_STATE_LIMIT: 14,
} as const;

/** Protobuf error format for FSM communication */
interface ProtoError {
  readonly code: number;
  readonly module: string;
  readonly msg: string;
}

/** Interface for errors that can be converted to string representation */
interface ErrorLike {
  readonly message?: string;
}

/**
 * Plugin Error class for structured error handling
 *
 * @example
 * ```typescript
 * const error = new PluginError(1, 'auth', 'Authentication failed');
 * console.log(error.toString()); // Formatted error string
 * const protoError = error.toProtoError(); // For FSM communication
 * ```
 */
class PluginError extends Error {
  public readonly code: number;
  public readonly module: string;
  public readonly msg: string;
  public readonly name = 'PluginError' as const;

  constructor(code: number, module: string, message: string) {
    super(message);
    this.code = code;
    this.module = module;
    this.msg = message;

    // Maintains proper stack trace for where our error was thrown (V8 only)
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, PluginError);
    }
  }

  /**
   * Format error string for logging and debugging
   *
   * @returns Formatted error string with module, code, and message
   */
  public toString(): string {
    return `\nModule:  ${this.module}\nCode:    ${this.code}\nMessage: ${this.msg}`;
  }

  /**
   * Convert to protobuf error format for FSM communication
   *
   * @returns Object compatible with protobuf error message format
   */
  public toProtoError(): ProtoError {
    return {
      code: this.code,
      module: this.module,
      msg: this.msg,
    } as const;
  }
}

/**
 * Create a new plugin error
 * Create a new plugin error with structured information
 *
 * @param code - Numeric error code
 * @param module - Module name where error occurred
 * @param message - Human-readable error message
 * @returns New PluginError instance
 */
function newError(code: number, module: string, message: string): PluginError {
  return new PluginError(code, module, message);
}

// Error factory functions for common plugin errors
// Each function returns identical error codes and messages

/**
 * Plugin timeout error
 * @returns PluginError with timeout message
 */
function errPluginTimeout(): PluginError {
  return newError(
    PluginErrorCode.PLUGIN_TIMEOUT,
    DEFAULT_MODULE,
    'a plugin timeout occurred'
  );
}

/**
 * Marshal operation failed error
 * @param err - The underlying error that caused marshal failure
 * @returns PluginError with marshal failure message
 */
function errMarshal(err: ErrorLike | string): PluginError {
  const errorMsg = typeof err === 'string' ? err : (err.message ?? String(err));
  return newError(
    PluginErrorCode.MARSHAL,
    DEFAULT_MODULE,
    `marshal() failed with err: ${errorMsg}`
  );
}

/**
 * Unmarshal operation failed error
 * @param err - The underlying error that caused unmarshal failure
 * @returns PluginError with unmarshal failure message
 */
function errUnmarshal(err: ErrorLike | string): PluginError {
  const errorMsg = typeof err === 'string' ? err : (err.message ?? String(err));
  return newError(
    PluginErrorCode.UNMARSHAL,
    DEFAULT_MODULE,
    `unmarshal() failed with err: ${errorMsg}`
  );
}

/**
 * Plugin read operation failed error
 * @param err - The underlying error that caused read failure
 * @returns PluginError with read failure message
 */
function errFailedPluginRead(err: ErrorLike | string): PluginError {
  const errorMsg = typeof err === 'string' ? err : (err.message ?? String(err));
  return newError(
    PluginErrorCode.FAILED_PLUGIN_READ,
    DEFAULT_MODULE,
    `a plugin read failed with err: ${errorMsg}`
  );
}

/**
 * Plugin write operation failed error
 * @param err - The underlying error that caused write failure
 * @returns PluginError with write failure message
 */
function errFailedPluginWrite(err: ErrorLike | string): PluginError {
  const errorMsg = typeof err === 'string' ? err : (err.message ?? String(err));
  return newError(
    PluginErrorCode.FAILED_PLUGIN_WRITE,
    DEFAULT_MODULE,
    `a plugin write failed with err: ${errorMsg}`
  );
}

/**
 * Invalid plugin response ID error
 * @returns PluginError indicating invalid response ID
 */
function errInvalidPluginRespId(): PluginError {
  return newError(
    PluginErrorCode.INVALID_PLUGIN_RESP_ID,
    DEFAULT_MODULE,
    'plugin response id is invalid'
  );
}

/**
 * Unexpected FSM to plugin message type error
 * @param type - The unexpected message type received
 * @returns PluginError indicating unexpected FSM message
 */
function errUnexpectedFSMToPlugin(type: string | number): PluginError {
  return newError(
    PluginErrorCode.UNEXPECTED_FSM_TO_PLUGIN,
    DEFAULT_MODULE,
    `unexpected FSM to plugin: ${type}`
  );
}

/**
 * Invalid FSM to plugin message type error
 * @param type - The invalid message type received
 * @returns PluginError indicating invalid FSM message
 */
function errInvalidFSMToPluginMessage(type: string | number): PluginError {
  return newError(
    PluginErrorCode.INVALID_FSM_TO_PLUGIN_MESSAGE,
    DEFAULT_MODULE,
    `invalid FSM to plugin: ${type}`
  );
}

/**
 * Insufficient funds error
 * @returns PluginError indicating insufficient funds
 */
function errInsufficientFunds(): PluginError {
  return newError(
    PluginErrorCode.INSUFFICIENT_FUNDS,
    DEFAULT_MODULE,
    'insufficient funds'
  );
}

/**
 * fromAny operation failed error
 * @param err - The underlying error that caused fromAny failure
 * @returns PluginError with fromAny failure message
 */
function errFromAny(err: ErrorLike | string): PluginError {
  const errorMsg = typeof err === 'string' ? err : (err.message ?? String(err));
  return newError(
    PluginErrorCode.FROM_ANY,
    DEFAULT_MODULE,
    `fromAny() failed with err: ${errorMsg}`
  );
}

/**
 * Invalid message cast error
 * @returns PluginError indicating message cast failure
 */
function errInvalidMessageCast(): PluginError {
  return newError(
    PluginErrorCode.INVALID_MESSAGE_CAST,
    DEFAULT_MODULE,
    'the message cast failed'
  );
}

/**
 * Invalid address error
 * @returns PluginError indicating invalid address
 */
function errInvalidAddress(): PluginError {
  return newError(
    PluginErrorCode.INVALID_ADDRESS,
    DEFAULT_MODULE,
    'address is invalid'
  );
}

/**
 * Invalid amount error
 * @returns PluginError indicating invalid amount
 */
function errInvalidAmount(): PluginError {
  return newError(
    PluginErrorCode.INVALID_AMOUNT,
    DEFAULT_MODULE,
    'amount is invalid'
  );
}

/**
 * Transaction fee below state limit error
 * @returns PluginError indicating fee is below state limit
 */
function errTxFeeBelowStateLimit(): PluginError {
  return newError(
    PluginErrorCode.TX_FEE_BELOW_STATE_LIMIT,
    DEFAULT_MODULE,
    'tx.fee is below state limit'
  );
}

// Export all error functions and classes
export {
  PluginError,
  PluginErrorCode,
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
};

// Export types separately
export type { ProtoError, ErrorFactory };

// For CommonJS compatibility
export default {
  PluginError,
  PluginErrorCode,
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
};
