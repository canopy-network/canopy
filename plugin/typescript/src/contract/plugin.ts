/* This file contains boilerplate logic to interact with the Canopy FSM via socket file */

import * as net from 'net';
import * as path from 'path';
import * as fs from 'fs/promises';
import Long from 'long';

import { types } from '../proto/types.js';

// Normalize message IDs to consistent string format
// protobufjs may return uint64 as Long or number depending on value
function normalizeId(id: Long | number | undefined): string {
    if (id === undefined || id === null) {
        return '0';
    }
    if (Long.isLong(id)) {
        return id.toString();
    }
    return String(id);
}

import {
    IPluginError,
    ErrPluginTimeout,
    ErrMarshal,
    ErrUnmarshal,
    ErrFailedPluginWrite,
    ErrInvalidPluginRespId,
    ErrUnexpectedFSMToPlugin,
    ErrInvalidFSMToPluginMMessage,
    ErrFromAny,
    ErrInvalidMessageCast
} from './error.js';

// Forward declaration - Contract will be set after import
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let ContractClass: any;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let ContractConfigValue: any;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let ContractAsyncClass: any;

// socketPath is the name of the plugin socket exposed by the base SDK
const socketPath = 'plugin.sock';

// CONFIG IMPLEMENTATION

export interface Config {
    ChainId: number;
    DataDirPath: string;
}

// DefaultConfig() returns the default configuration
export function DefaultConfig(): Config {
    return {
        ChainId: 1,
        DataDirPath: '/tmp/plugin/'
    };
}

// NewConfigFromFile() populates a Config object from a JSON file
export async function NewConfigFromFile(filepath: string): Promise<Config> {
    const fileBytes = await fs.readFile(filepath, 'utf-8');
    const c = DefaultConfig();
    const parsed = JSON.parse(fileBytes) as Partial<Config>;
    return { ...c, ...parsed };
}

// Plugin defines the 'VM-less' extension of the Finite State Machine
export class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    fsmConfig: any = null;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    pluginConfig: any;
    conn: net.Socket;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    pending: Map<string, { resolve: (value: any) => void }> = new Map();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    requestContract: Map<string, any> = new Map();
    config: Config;
    private buffer: Buffer = Buffer.alloc(0);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(config: Config, conn: net.Socket, pluginConfig: any) {
        this.pluginConfig = pluginConfig;
        this.conn = conn;
        this.config = config;
    }

    async Handshake(): Promise<IPluginError | null> {
        console.log('Handshaking with FSM');
        const contract = new ContractClass(this.config, this.fsmConfig, this, Long.ZERO);
        const [response, err] = await this.sendToPluginSync(contract, {
            config: this.pluginConfig
        });
        if (err) {
            return err;
        }
        if (!response || !response.config) {
            return ErrUnexpectedFSMToPlugin(typeof response);
        }
        this.fsmConfig = response.config;
        return null;
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async StateRead(c: any, request: any): Promise<[any | null, IPluginError | null]> {
        const [response, err] = await this.sendToPluginSync(c, { stateRead: request });
        if (err) {
            return [null, err];
        }
        if (!response || !response.stateRead) {
            return [null, ErrUnexpectedFSMToPlugin(typeof response)];
        }
        return [response.stateRead, null];
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async StateWrite(c: any, request: any): Promise<[any | null, IPluginError | null]> {
        const [response, err] = await this.sendToPluginSync(c, { stateWrite: request });
        if (err) {
            return [null, err];
        }
        if (!response || !response.stateWrite) {
            return [null, ErrUnexpectedFSMToPlugin(typeof response)];
        }
        return [response.stateWrite, null];
    }

    ListenForInbound(): void {
        this.conn.on('data', (chunk: Buffer) => {
            this.buffer = Buffer.concat([this.buffer, chunk]);
            this.processBuffer();
        });

        this.conn.on('error', (err) => {
            console.error(`Socket error: ${err.message}`);
            process.exit(1);
        });

        this.conn.on('close', () => {
            console.log('Socket closed');
            process.exit(0);
        });
    }

    private processBuffer(): void {
        while (this.buffer.length >= 4) {
            const messageLength = this.buffer.readUInt32BE(0);
            if (this.buffer.length < 4 + messageLength) {
                break;
            }
            const msgBytes = this.buffer.subarray(4, 4 + messageLength);
            this.buffer = this.buffer.subarray(4 + messageLength);
            this.handleMessage(msgBytes);
        }
    }

    private handleMessage(msgBytes: Buffer): void {
        const [msg, err] = Unmarshal(msgBytes, types.FSMToPlugin);
        if (err || !msg) {
            console.error(`Failed to unmarshal message: ${err?.msg}`);
            process.exit(1);
        }

        // Handle message asynchronously
        this.handleMessageAsync(msg).catch((e) => {
            console.error(`Error handling message: ${e}`);
            process.exit(1);
        });
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    private async handleMessageAsync(msg: any): Promise<void> {
        const c = new ContractClass(this.config, this.fsmConfig, this, msg.id as Long);

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        let response: any = null;

        // Check which payload field is actually set (protobufjs may have empty objects for unset fields)
        // Use the 'payload' field name from the oneof if available, otherwise check object keys

        // Route based on which field has actual content
        if (msg.config && Object.keys(msg.config).length >= 0 && msg.payload === 'config') {
            console.log('Received config response from FSM');
            const handleErr = this.handleFSMResponse(msg);
            if (handleErr) {
                console.error(handleErr.msg);
                process.exit(1);
            }
            return;
        } else if (msg.stateRead && msg.payload === 'stateRead') {
            console.log('Received stateRead response from FSM');
            const handleErr = this.handleFSMResponse(msg);
            if (handleErr) {
                console.error(handleErr.msg);
                process.exit(1);
            }
            return;
        } else if (msg.stateWrite && msg.payload === 'stateWrite') {
            console.log('Received stateWrite response from FSM');
            const handleErr = this.handleFSMResponse(msg);
            if (handleErr) {
                console.error(handleErr.msg);
                process.exit(1);
            }
            return;
        } else if (msg.genesis && msg.payload === 'genesis') {
            console.log('Received genesis request from FSM');
            response = { genesis: c.Genesis(msg.genesis) };
        } else if (msg.begin && msg.payload === 'begin') {
            console.log('Received begin request from FSM');
            response = { begin: c.BeginBlock(msg.begin) };
        } else if (msg.check && msg.payload === 'check') {
            console.log('Received check request from FSM');
            const checkResponse = await ContractAsyncClass.CheckTx(c, msg.check);
            response = { check: checkResponse };
        } else if (msg.deliver && msg.payload === 'deliver') {
            console.log('Received deliver request from FSM');
            const deliverResponse = await ContractAsyncClass.DeliverTx(c, msg.deliver);
            response = { deliver: deliverResponse };
        } else if (msg.end && msg.payload === 'end') {
            console.log('Received end request from FSM');
            response = { end: c.EndBlock(msg.end) };
        } else {
            // Fallback: check which field actually has content
            console.log(`DEBUG: msg.payload=${msg.payload}, keys=${Object.keys(msg).join(',')}`);
            const handleErr = ErrInvalidFSMToPluginMMessage(JSON.stringify(msg));
            console.error(handleErr.msg);
            process.exit(1);
        }

        const sendErr = this.sendProtoMsg(
            types.PluginToFSM.create({
                id: msg.id,
                ...response
            })
        );

        if (sendErr) {
            console.error(sendErr.msg);
            process.exit(1);
        }
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleFSMResponse(msg: any): IPluginError | null {
        const id = normalizeId(msg.id);
        const pending = this.pending.get(id);
        if (!pending) {
            return ErrInvalidPluginRespId();
        }
        this.pending.delete(id);
        this.requestContract.delete(id);
        pending.resolve(msg);
        return null;
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async sendToPluginSync(c: any, payload: any): Promise<[any | null, IPluginError | null]> {
        const [promise, requestId, err] = this.sendToPluginAsync(c, payload);
        if (err) {
            return [null, err];
        }
        const [response, waitErr] = await this.waitForResponse(promise, requestId);
        this.requestContract.delete(requestId);
        return [response, waitErr];
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    sendToPluginAsync(c: any, payload: any): [Promise<any>, string, IPluginError | null] {
        const requestId = normalizeId(c.fsmId);
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        let resolvePromise: (value: any) => void;
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const promise = new Promise<any>((resolve) => {
            resolvePromise = resolve;
        });
        this.pending.set(requestId, { resolve: resolvePromise! });
        this.requestContract.set(requestId, c);
        const err = this.sendProtoMsg(
            types.PluginToFSM.create({
                id: c.fsmId,
                ...payload
            })
        );
        return [promise, requestId, err];
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async waitForResponse(
        promise: Promise<any>,
        requestId: string
    ): Promise<[any | null, IPluginError | null]> {
        const timeoutPromise = new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error('timeout')), 10000)
        );
        try {
            const response = await Promise.race([promise, timeoutPromise]);
            return [response, null];
        } catch {
            this.pending.delete(requestId);
            this.requestContract.delete(requestId);
            return [null, ErrPluginTimeout()];
        }
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    sendProtoMsg(msg: any): IPluginError | null {
        const [bz, err] = Marshal(msg);
        if (err || !bz) {
            return err;
        }
        return this.sendLengthPrefixed(bz);
    }

    sendLengthPrefixed(bz: Uint8Array): IPluginError | null {
        const lengthPrefix = Buffer.alloc(4);
        lengthPrefix.writeUInt32BE(bz.length, 0);
        try {
            this.conn.write(Buffer.concat([lengthPrefix, Buffer.from(bz)]));
        } catch (er) {
            return ErrFailedPluginWrite(er as Error);
        }
        return null;
    }
}

// StartPlugin() creates and starts a plugin
export function StartPlugin(c: Config): void {
    const sockPath = path.join(c.DataDirPath, socketPath);

    const tryConnect = (): void => {
        const conn = net.createConnection(sockPath);

        conn.on('connect', () => {
            console.log('Connected to plugin socket');
            const p = new Plugin(c, conn, ContractConfigValue);
            p.ListenForInbound();
            p.Handshake().then((err) => {
                if (err) {
                    console.error(err.msg);
                    process.exit(1);
                }
            });
        });

        conn.on('error', (err) => {
            console.log(`Failed to connect to plugin socket: ${err.message}`);
            setTimeout(tryConnect, 1000);
        });
    };

    tryConnect();
}

// Initialize contract references after module load
export function initializeContract(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    contractClass: any,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    contractConfig: any,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    contractAsyncClass: any
): void {
    ContractClass = contractClass;
    ContractConfigValue = contractConfig;
    ContractAsyncClass = contractAsyncClass;
}

// CODEC IMPLEMENTATION

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function Marshal(message: any): [Uint8Array | null, IPluginError | null] {
    try {
        const protoBytes = types.PluginToFSM.encode(message).finish();
        return [protoBytes, null];
    } catch (err) {
        return [null, ErrMarshal(err as Error)];
    }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function Unmarshal<T>(
    protoBytes: Uint8Array | Buffer,
    MessageType: any
): [T | null, IPluginError | null] {
    if (!protoBytes || protoBytes.length === 0) {
        return [null, null];
    }
    try {
        return [MessageType.decode(protoBytes), null];
    } catch (err) {
        return [null, ErrUnmarshal(err as Error)];
    }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function FromAny(any: any): [any | null, string | null, IPluginError | null] {
    if (!any || !any.value) {
        return [null, null, ErrFromAny(new Error('any is null or has no value'))];
    }

    // Check both typeUrl and type_url (protobuf field name variations)
    const typeUrl = any.typeUrl || any.type_url || '';

    try {
        if (typeUrl.includes('MessageSend')) {
            return [types.MessageSend.decode(any.value), 'MessageSend', null];
        }
        // NOTE: To add new message types, see TUTORIAL.md
        return [null, null, ErrInvalidMessageCast()];
    } catch (err) {
        return [null, null, ErrFromAny(err as Error)];
    }
}

export function JoinLenPrefix(...toAppend: (Buffer | Uint8Array | null | undefined)[]): Uint8Array {
    let totalLen = 0;
    for (const item of toAppend) {
        if (item) {
            totalLen += 1 + item.length;
        }
    }
    const res = Buffer.alloc(totalLen);
    let offset = 0;
    for (const item of toAppend) {
        if (!item) {
            continue;
        }
        res[offset++] = item.length;
        Buffer.from(item).copy(res, offset);
        offset += item.length;
    }
    return res;
}
