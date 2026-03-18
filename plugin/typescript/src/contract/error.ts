/* This file contains contract level PluginErrors */

import { types } from '../proto/types.js';

const DefaultModule = 'plugin';

// PluginError interface matching the protobuf definition
export interface IPluginError {
    code: number;
    module: string;
    msg: string;
}

// NewError() creates a plugin error
export function NewError(code: number, module: string, message: string): IPluginError {
    return types.PluginError.create({ code, module, msg: message });
}

export function ErrPluginTimeout(): IPluginError {
    return NewError(1, DefaultModule, 'a plugin timeout occurred');
}

export function ErrMarshal(err: Error): IPluginError {
    return NewError(2, DefaultModule, `marshal() failed with err: ${err.message}`);
}

export function ErrUnmarshal(err: Error): IPluginError {
    return NewError(3, DefaultModule, `unmarshal() failed with err: ${err.message}`);
}

export function ErrFailedPluginRead(err: Error): IPluginError {
    return NewError(4, DefaultModule, `a plugin read failed with err: ${err.message}`);
}

export function ErrFailedPluginWrite(err: Error): IPluginError {
    return NewError(5, DefaultModule, `a plugin write failed with err: ${err.message}`);
}

export function ErrInvalidPluginRespId(): IPluginError {
    return NewError(6, DefaultModule, 'plugin response id is invalid');
}

export function ErrUnexpectedFSMToPlugin(t: string): IPluginError {
    return NewError(7, DefaultModule, `unexpected FSM to plugin: ${t}`);
}

export function ErrInvalidFSMToPluginMMessage(t: string): IPluginError {
    return NewError(8, DefaultModule, `invalid FSM to plugin: ${t}`);
}

export function ErrInsufficientFunds(): IPluginError {
    return NewError(9, DefaultModule, 'insufficient funds');
}

export function ErrFromAny(err: Error): IPluginError {
    return NewError(10, DefaultModule, `fromAny() failed with err: ${err.message}`);
}

export function ErrInvalidMessageCast(): IPluginError {
    return NewError(11, DefaultModule, 'the message cast failed');
}

export function ErrInvalidAddress(): IPluginError {
    return NewError(12, DefaultModule, 'address is invalid');
}

export function ErrInvalidAmount(): IPluginError {
    return NewError(13, DefaultModule, 'amount is invalid');
}

export function ErrTxFeeBelowStateLimit(): IPluginError {
    return NewError(14, DefaultModule, 'tx.fee is below state limit');
}
