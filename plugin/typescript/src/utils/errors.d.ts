declare module 'plugin-error' {
    export interface ProtoError {
        readonly code: number;
        readonly module: string;
        readonly msg: string;
    }

    export interface ErrorLike {
        readonly message?: string;
    }

    export const enum PluginErrorCode {
        PLUGIN_TIMEOUT = 1,
        MARSHAL = 2,
        UNMARSHAL = 3,
        FAILED_PLUGIN_READ = 4,
        FAILED_PLUGIN_WRITE = 5,
        INVALID_PLUGIN_RESP_ID = 6,
        UNEXPECTED_FSM_TO_PLUGIN = 7,
        INVALID_FSM_TO_PLUGIN_MESSAGE = 8,
        INSUFFICIENT_FUNDS = 9,
        FROM_ANY = 10,
        INVALID_MESSAGE_CAST = 11,
        INVALID_ADDRESS = 12,
        INVALID_AMOUNT = 13,
        TX_FEE_BELOW_STATE_LIMIT = 14
    }

    export class PluginError extends Error {
        readonly code: number;
        readonly module: string;
        readonly msg: string;
        readonly name: 'PluginError';

        constructor(code: number, module: string, message: string);
        toString(): string;
        toProtoError(): ProtoError;
    }

    export function newError(code: number, module: string, message: string): PluginError;

    // Error factory functions
    export function errPluginTimeout(): PluginError;
    export function errMarshal(err: ErrorLike | string): PluginError;
    export function errUnmarshal(err: ErrorLike | string): PluginError;
    export function errFailedPluginRead(err: ErrorLike | string): PluginError;
    export function errFailedPluginWrite(err: ErrorLike | string): PluginError;
    export function errInvalidPluginRespId(): PluginError;
    export function errUnexpectedFSMToPlugin(type: string | number): PluginError;
    export function errInvalidFSMToPluginMessage(type: string | number): PluginError;
    export function errInsufficientFunds(): PluginError;
    export function errFromAny(err: ErrorLike | string): PluginError;
    export function errInvalidMessageCast(): PluginError;
    export function errInvalidAddress(): PluginError;
    export function errInvalidAmount(): PluginError;
    export function errTxFeeBelowStateLimit(): PluginError;
}
