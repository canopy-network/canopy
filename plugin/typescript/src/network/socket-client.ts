import Long from 'long';
import { Socket, connect } from 'net';
import { EventEmitter } from 'events';
import { join } from 'path';
import * as proto from '../proto/index.js';
const { types } = proto;
import { Logger } from '../utils/index.ts';
import Config from '../config/index.ts';
import { Contract } from '../core/contract.ts';

/**
 * Promise resolver interface for pending requests
 */
export interface PendingRequest {
    readonly resolve: (value: types.FSMToPlugin) => void;
    readonly reject: (reason: Error) => void;
}

/**
 * Socket client options for constructor
 */
export interface SocketClientOptions {
    readonly config: Config;
    readonly reconnectInterval?: number;
    readonly requestTimeout?: number;
    readonly connectionTimeout?: number;
}

/**
 * Async channel result for plugin messages
 */
export interface AsyncChannelResult {
    readonly channel: Promise<types.FSMToPlugin>;
    readonly requestId: string;
}

/**
 * Message routing information for debugging
 */
export interface MessageRouting {
    readonly messageId: string;
    readonly messageTypes: readonly string[];
    readonly isPending: boolean;
}

/**
 * Contract creation parameters
 */
export interface ContractParams {
    readonly config: Config;
    readonly plugin: SocketClient;
    readonly fsmId: bigint | Long;
}

/**
 * Unix socket client that communicates with Canopy FSM using length-prefixed protobuf messages
 * Provides full TypeScript type safety for blockchain communication
 */
export class SocketClient extends EventEmitter {
    private readonly config: Config;
    private readonly logger: Logger;
    private readonly socketPath: string;
    private readonly reconnectInterval: number;
    private readonly requestTimeout: number;
    private readonly connectionTimeout: number;

    private conn: Socket | null = null;
    private readonly pending = new Map<string, PendingRequest>();
    private readonly requestContract = new Map<string, Contract>();

    private isConnected = false;
    private isReconnecting = false;

    constructor(config: SocketClientOptions) {
        super();
        this.config = config;
        this.logger = new Logger('SocketClient', process.env.LOG_LEVEL || 'debug');
        this.socketPath = join(this.config.dataDirPath, 'plugin.sock');
        this.reconnectInterval = config.reconnectInterval ?? 3000;
        this.requestTimeout = config.requestTimeout ?? 10000;
        this.connectionTimeout = config.connectionTimeout ?? 5000;
    }

    /**
     * Start the socket client and connect to FSM
     */
    async start(): Promise<void> {
        await this.connectWithRetry();
        this.startListening();
        await this.handshake();
        this.logger.info('Socket client started and connected to FSM');
    }

    /**
     * Connect to Unix socket with retry logic
     */
    private async connectWithRetry(): Promise<void> {
        if (this.isReconnecting) {
            return;
        }
        this.isReconnecting = true;

        while (!this.isConnected) {
            try {
                await this.attemptConnection();
                this.isReconnecting = false;
                return;
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : String(err);
                this.logger.warn(`Error connecting to plugin socket: ${errorMessage}`);

                this.cleanupFailedConnection();
                await this.waitForReconnect();
            }
        }

        this.isReconnecting = false;
    }

    /**
     * Clean up failed connection attempt
     */
    private cleanupFailedConnection(): void {
        if (this.conn) {
            this.conn.destroy();
            this.conn = null;
        }
        this.isConnected = false;
    }

    /**
     * Wait before retrying connection
     */
    private async waitForReconnect(): Promise<void> {
        await new Promise<void>((resolve) => {
            setTimeout(resolve, this.reconnectInterval);
        });
    }

    /**
     * Attempt a single connection with proper error handling
     */
    private async attemptConnection(): Promise<void> {
        const conn = connect(this.socketPath);
        this.conn = conn;

        // Remove existing listeners to prevent memory leaks
        conn.removeAllListeners();

        conn.on('connect', () => {
            this.isConnected = true;
            this.logger.connection(`established ${this.socketPath}`);
        });

        conn.on('error', () => {
            this.isConnected = false;
        });

        conn.on('close', () => {
            this.isConnected = false;
            this.logger.connection('closed');

            if (!this.isReconnecting) {
                setTimeout(() => {
                    void this.connectWithRetry();
                }, this.reconnectInterval);
            }
        });

        await new Promise<void>((resolve, reject) => {
            const timeout = setTimeout(() => {
                conn.destroy();
                reject(new Error('Connection timeout'));
            }, this.connectionTimeout);

            conn.once('connect', () => {
                clearTimeout(timeout);
                resolve();
            });

            conn.once('error', (err: Error) => {
                clearTimeout(timeout);
                reject(err);
            });
        });
    }

    /**
     * Start listening for inbound messages from FSM with proper buffer management
     */
    private startListening(): void {
        if (!this.conn) {
            throw new Error('No connection available for listening');
        }

        let buffer = Buffer.alloc(0);
        let expectedLength: number | null = null;

        this.conn.on('data', (data: Buffer) => {
            buffer = Buffer.concat([buffer, data]);

            while (buffer.length > 0) {
                // Read length prefix if we don't have one yet
                if (expectedLength === null && buffer.length >= 4) {
                    expectedLength = buffer.readUInt32BE(0);
                    buffer = buffer.subarray(4);
                }

                // Read message if we have enough data
                if (expectedLength !== null && buffer.length >= expectedLength) {
                    const messageBytes = buffer.subarray(0, expectedLength);
                    buffer = buffer.subarray(expectedLength);
                    expectedLength = null;

                    this.handleInboundMessage(messageBytes);
                } else {
                    // Need more data - wait for next chunk
                    this.logger.debug(
                        `Need more data: have ${buffer.length}, need ${expectedLength ?? 'unknown'}`
                    );
                    break;
                }
            }
        });
    }

    /**
     * Handle inbound protobuf message from FSM with comprehensive type safety
     */
    private handleInboundMessage(messageBytes: Buffer): void {
        try {
            const fsmToPlugin = types.FSMToPlugin.decode(messageBytes);
            const routing = this.analyzeMessageRouting(fsmToPlugin);

            this.logger.debug(`Inbound message types: [${routing.messageTypes.join(',')}]`);

            if (routing.isPending) {
                this.logger.debug(`Routing to handleFSMResponse (id: ${routing.messageId})`);
                this.handleFSMResponse(fsmToPlugin);
            } else {
                this.logger.debug(`Routing to handleFSMRequest (id: ${routing.messageId})`);
                void this.handleFSMRequest(fsmToPlugin);
            }
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : String(err);
            this.logger.error('Failed to decode FSM message', { error: errorMessage });
        }
    }

    /**
     * Analyze message routing information for debugging and type safety
     */
    private analyzeMessageRouting(message: types.FSMToPlugin): MessageRouting {
        const messageTypes: string[] = [];
        if (message.config) messageTypes.push('config');
        if (message.genesis) messageTypes.push('genesis');
        if (message.begin) messageTypes.push('begin');
        if (message.check) messageTypes.push('check');
        if (message.deliver) messageTypes.push('deliver');
        if (message.end) messageTypes.push('end');
        if (message.stateRead) messageTypes.push('stateRead');
        if (message.stateWrite) messageTypes.push('stateWrite');
        if (message.error) messageTypes.push('error');

        const messageId = String(message.id);
        const isPending = this.pending.has(messageId);

        return {
            messageId,
            messageTypes: Object.freeze(messageTypes),
            isPending
        };
    }

    /**
     * Handle response from FSM to our request with proper cleanup
     */
    private handleFSMResponse(message: types.FSMToPlugin): void {
        const messageId = String(message.id);
        const pending = this.pending.get(messageId);

        if (!pending) {
            this.logger.error(`No pending request found for id: ${messageId}`);
            this.logger.error(
                `Available pending requests: [${Array.from(this.pending.keys()).join(',')}]`
            );
            return;
        }

        // Clean up tracking maps
        this.pending.delete(messageId);
        this.requestContract.delete(messageId);

        // Resolve the promise
        pending.resolve(message);
    }

    /**
     * Handle new request from FSM with comprehensive error handling
     */
    private async handleFSMRequest(message: types.FSMToPlugin): Promise<void> {
        try {
            const response = await this.processRequestMessage(message);
            if (response) {
                await this.sendResponseToFSM(message.id, response);
            }
        } catch (err) {
            await this.sendErrorResponse(message.id, err);
        }
    }

    /**
     * Process specific request message types with comprehensive error handling and type safety
     */
    private async processRequestMessage(
        message: types.FSMToPlugin
    ): Promise<Partial<types.PluginToFSM> | null> {
        const messageId = String(message.id);
        let contract: Contract;

        try {
            contract = this.createContractInstance(message.id);
        } catch (error) {
            const errorMsg = error instanceof Error ? error.message : String(error);
            throw new Error(`Failed to create contract instance for message ${messageId}: ${errorMsg}`);
        }

        try {
            // Handle config message
            if (message.config) {
                this.logger.debug(`Processing config message (id: ${messageId})`);
                return null; // No response needed
            }

            // Handle genesis message
            if (message.genesis) {
                this.logger.info(`Processing genesis request (id: ${messageId})`);
                try {
                    const result = contract.genesis(message.genesis);
                    this.logger.info(`Genesis request processed successfully (id: ${messageId})`);
                    return { genesis: result };
                } catch (error) {
                    const errorMsg = error instanceof Error ? error.message : String(error);
                    throw new Error(`Genesis processing failed for message ${messageId}: ${errorMsg}`);
                }
            }

            // Handle begin block message
            if (message.begin) {
                const blockHeight = message.begin.height ? String(message.begin.height) : 'unknown';
                this.logger.debug(`Processing begin block request (id: ${messageId}, height: ${blockHeight})`);
                try {
                    const result = contract.beginBlock(message.begin);

                    // Check if the result contains an error
                    if (result.error) {
                        this.logger.error(`Begin block returned error (id: ${messageId}, height: ${blockHeight})`, {
                            code: result.error.code,
                            module: result.error.module,
                            message: result.error.msg
                        });
                        // Return the result with error - FSM needs to handle the error response
                        return { begin: result };
                    }

                    this.logger.debug(`Begin block request processed successfully (id: ${messageId})`);
                    return { begin: result };
                } catch (error) {
                    const errorMsg = error instanceof Error ? error.message : String(error);
                    throw new Error(`Begin block processing failed for message ${messageId}, height ${blockHeight}: ${errorMsg}`);
                }
            }

            // Handle check tx message
            if (message.check) {
                const txType = message.check.tx?.messageType ?? 'unknown';
                this.logger.debug(`Processing check tx request (id: ${messageId}, type: ${txType})`);
                try {
                    const result = await contract.checkTx(message.check);

                    // Check if the result contains an error
                    if (result.error) {
                        this.logger.error(`Check tx returned error (id: ${messageId}, type: ${txType})`, {
                            code: result.error.code,
                            module: result.error.module,
                            message: result.error.msg
                        });
                        // Return the result with error - FSM needs to handle the error response
                        return { check: result };
                    }

                    this.logger.debug(`Check tx request processed successfully (id: ${messageId})`);
                    return { check: result };
                } catch (error) {
                    const errorMsg = error instanceof Error ? error.message : String(error);
                    throw new Error(`Check tx processing failed for message ${messageId}, tx type ${txType}: ${errorMsg}`);
                }
            }

            // Handle deliver tx message
            if (message.deliver) {
                const txType = message.deliver.tx?.messageType ?? 'unknown';
                this.logger.debug(`Processing deliver tx request (id: ${messageId}, type: ${txType})`);
                try {
                    const result = await contract.deliverTx(message.deliver);

                    // Check if the result contains an error
                    if (result.error) {
                        this.logger.error(`Deliver tx returned error (id: ${messageId}, type: ${txType})`, {
                            code: result.error.code,
                            module: result.error.module,
                            message: result.error.msg
                        });
                        // Return the result with error - FSM needs to handle the error response
                        return { deliver: result };
                    }

                    this.logger.debug(`Deliver tx request processed successfully (id: ${messageId})`);
                    return { deliver: result };
                } catch (error) {
                    const errorMsg = error instanceof Error ? error.message : String(error);
                    throw new Error(`Deliver tx processing failed for message ${messageId}, tx type ${txType}: ${errorMsg}`);
                }
            }

            // Handle end block message
            if (message.end) {
                const blockHeight = message.end.height ? String(message.end.height) : 'unknown';
                this.logger.debug(`Processing end block request (id: ${messageId}, height: ${blockHeight})`);
                try {
                    const result = contract.endBlock(message.end);

                    // Check if the result contains an error
                    if (result.error) {
                        this.logger.error(`End block returned error (id: ${messageId}, height: ${blockHeight})`, {
                            code: result.error.code,
                            module: result.error.module,
                            message: result.error.msg
                        });
                        // Return the result with error - FSM needs to handle the error response
                        return { end: result };
                    }

                    this.logger.debug(`End block request processed successfully (id: ${messageId})`);
                    return { end: result };
                } catch (error) {
                    const errorMsg = error instanceof Error ? error.message : String(error);
                    throw new Error(`End block processing failed for message ${messageId}, height ${blockHeight}: ${errorMsg}`);
                }
            }

            // Handle state read response (should not be here)
            if (message.stateRead) {
                this.logger.debug(`Received state read response from FSM (id: ${messageId})`);
                return null; // This is a response, not a request
            }

            // Handle state write response (should not be here)
            if (message.stateWrite) {
                this.logger.error(`stateWrite response incorrectly routed to handleFSMRequest (id: ${messageId})`);
                return null;
            }

            // Handle error message
            if (message.error) {
                this.logger.error(`Received error from FSM (id: ${messageId})`, {
                    code: message.error.code,
                    module: message.error.module,
                    message: message.error.msg
                });
                return null; // Error responses don't need replies
            }

            // No valid message type found
            const availableTypes = Object.keys(message).filter(key => key !== 'id' && message[key as keyof typeof message]);
            throw new Error(`Invalid FSM to plugin message type (id: ${messageId}). Available types: [${availableTypes.join(', ')}]`);

        } catch (error) {
            // Re-throw with additional context if not already enhanced
            if (error instanceof Error && error.message.includes(messageId)) {
                throw error;
            }
            const errorMsg = error instanceof Error ? error.message : String(error);
            throw new Error(`Message processing failed for ${messageId}: ${errorMsg}`);
        }
    }

    /**
     * Send response back to FSM with proper message structure
     */
    private async sendResponseToFSM(
        messageId: Long,
        response: Partial<types.PluginToFSM>
    ): Promise<void> {
        const pluginToFSM: types.PluginToFSM = {
            id: messageId,
            ...response
        };

        await this.sendProtoMsg(types.PluginToFSM, pluginToFSM);
    }

    /**
     * Send error response with proper error structure
     */
    private async sendErrorResponse(messageId: Long, error: unknown): Promise<void> {
        const errorMessage = error instanceof Error ? error.message : String(error);
        this.logger.error('Error handling FSM request', { error: errorMessage });

        const errorResponse: types.PluginToFSM = {
            id: messageId,
            error: {
                code: 1,
                module: 'plugin',
                msg: errorMessage
            }
        };

        await this.sendProtoMsg(types.PluginToFSM, errorResponse);
    }

    /**
     * Create contract instance with proper typing
     */
    private createContractInstance(fsmId: Long): Contract {
        const contractParams: ContractParams = {
            config: this.config,
            plugin: this,
            fsmId: fsmId
        };

        return new Contract(contractParams);
    }

    /**
     * Create contract instance specifically for handshake operations
     */
    private createContractInstanceForHandshake(): Contract {
        const contractParams: ContractParams = {
            config: this.config,
            plugin: this,
            fsmId: Long.fromNumber(999) // Use consistent BigInt for handshake
        };

        return new Contract(contractParams);
    }

    /**
     * Perform handshake with FSM using proper protobuf types
     */
    private async handshake(): Promise<void> {
        this.logger.info('Starting FSM handshake');

        const pluginConfig: types.PluginConfig = types.PluginConfig.create({
            name: 'send',
            id: 1,
            version: 1,
            supportedTransactions: ['send']
        });

        const contract = this.createContractInstanceForHandshake();
        const response = await this.sendToPluginSync(contract, { config: pluginConfig });

        if (!Object.prototype.hasOwnProperty.call(response, 'config')) {
            throw new Error('Unexpected FSM response during handshake');
        }

        this.logger.info('FSM handshake completed successfully');
    }

    /**
     * Send message to FSM and wait for response with proper timeout handling
     */
    async sendToPluginSync(
        contract: Contract,
        payload: Partial<types.PluginToFSM>
    ): Promise<types.FSMToPlugin> {
        const { channel, requestId } = this.sendToPluginAsync(contract, payload);

        try {
            return await this.waitForResponse(channel, requestId);
        } finally {
            this.requestContract.delete(requestId);
        }
    }

    /**
     * Send message to FSM without waiting for response
     */
    private sendToPluginAsync(
        contract: Contract,
        payload: Partial<types.PluginToFSM>
    ): AsyncChannelResult {
        const requestId = this.generateRequestId(contract);

        this.logger.debug(
            `Sending message to FSM (id: ${requestId}, payload: [${Object.keys(payload).join(',')}])`
        );

        const promise = new Promise<types.FSMToPlugin>((resolve, reject) => {
            this.pending.set(requestId, { resolve, reject });
        });

        this.requestContract.set(requestId, contract);

        const messageId = this.convertToMessageId(requestId);
        const message: types.PluginToFSM = {
            id: messageId,
            ...payload
        };

        this.logger.protocol(`Outbound message (id: ${messageId})`);
        void this.sendProtoMsg(types.PluginToFSM, message);

        return { channel: promise, requestId };
    }

    /**
     * Generate request ID with proper type handling
     */
    private generateRequestId(contract: Contract): string {
        const fsmId = contract.fsmId;

        if (fsmId !== undefined && fsmId !== null) {
            return String(fsmId);
        }

        return String(Math.floor(Math.random() * Number.MAX_SAFE_INTEGER));
    }

    /**
     * Convert request ID to protobuf-compatible message ID
     */
    private convertToMessageId(requestId: string): Long {
        try {
          const longId = Long.fromString(requestId, true);
            return longId;
        } catch (error) {
            throw new Error(`Invalid request ID for protobuf: ${requestId} ${error}`);
        }
    }

    /**
     * Wait for response with timeout handling
     */
    private async waitForResponse(
        channel: Promise<types.FSMToPlugin>,
        requestId: string
    ): Promise<types.FSMToPlugin> {
        const timeoutPromise = new Promise<never>((_, reject) => {
            setTimeout(() => {
                this.pending.delete(requestId);
                this.requestContract.delete(requestId);
                reject(new Error('Plugin timeout occurred'));
            }, this.requestTimeout);
        });

        try {
            return await Promise.race([channel, timeoutPromise]);
        } catch (err) {
            // Clean up on timeout or error
            this.pending.delete(requestId);
            this.requestContract.delete(requestId);
            throw err;
        }
    }

    /**
     * Send protobuf message with length prefix for blockchain communication
     */
    private async sendProtoMsg<T extends object>(
        messageType: { create: (obj: T) => T; encode: (obj: T) => { finish(): Uint8Array } },
        data: T
    ): Promise<void> {
        if (!this.conn) {
            throw new Error('No connection available for sending message');
        }

        try {
            const message = messageType.create(data);
            const messageBytes = messageType.encode(message).finish();

            // Create 4-byte length prefix (big endian)
            const lengthPrefix = Buffer.allocUnsafe(4);
            lengthPrefix.writeUInt32BE(messageBytes.length, 0);

            // Send length-prefixed message
            const fullMessage = Buffer.concat([lengthPrefix, Buffer.from(messageBytes)]);
            this.conn.write(fullMessage);
        } catch (err) {
            const errorMessage = err instanceof Error ? err.message : String(err);
            throw new Error(`Failed to send plugin message: ${errorMessage}`);
        }
    }

    /**
     * Send state read request to FSM with type safety
     */
    async stateRead(
        contract: Contract,
        request: types.StateReadRequest
    ): Promise<types.StateReadResponse> {
        const response = await this.sendToPluginSync(contract, { stateRead: request });

        if (!response.stateRead) {
            throw new Error('Unexpected FSM response to state read');
        }

        return response.stateRead;
    }

    /**
     * Send state write request to FSM with type safety
     */
    async stateWrite(
        contract: Contract,
        request: types.StateWriteRequest
    ): Promise<types.StateWriteResponse> {
        const response = await this.sendToPluginSync(contract, { stateWrite: request });

        if (!response.stateWrite) {
            throw new Error('Unexpected FSM response to state write');
        }

        return response.stateWrite;
    }

    /**
     * Close the socket connection with proper cleanup
     */
    async close(): Promise<void> {
        if (!this.conn) {
            this.isConnected = false;
            return;
        }

        return new Promise<void>((resolve) => {
            // Set flag to prevent reconnection attempts
            this.isReconnecting = true;

            // Store reference and clean up state
            const originalConn = this.conn!;
            this.conn = null;
            this.isConnected = false;

            // Wait for connection to close cleanly
            originalConn.once('close', () => {
                resolve();
            });

            // Force destroy the connection
            originalConn.destroy();

            // Fallback timeout in case close event doesn't fire
            setTimeout(resolve, 100);
        });
    }

    /**
     * Check if client is currently connected to FSM
     */
    isSocketConnected(): boolean {
        return this.isConnected;
    }

    /**
     * Get current socket path
     */
    getSocketPath(): string {
        return this.socketPath;
    }

    /**
     * Get number of pending requests
     */
    getPendingRequestCount(): number {
        return this.pending.size;
    }
}

export default SocketClient;
