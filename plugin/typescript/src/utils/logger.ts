/**
 * Logger class for the Express.ts Canopy plugin
 * Provides structured logging with different log levels
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

interface LogLevels {
    readonly debug: 0;
    readonly info: 1;
    readonly warn: 2;
    readonly error: 3;
}

interface LogData {
    readonly [key: string]: unknown;
}

type LogDataValue = LogData | string | number | boolean | null | undefined;

export default class Logger {
    private readonly name: string;
    private level: LogLevel;
    private currentLevel: number;

    private static readonly levels: LogLevels = {
        debug: 0,
        info: 1,
        warn: 2,
        error: 3
    } as const;

    constructor(name: string = 'SocketClient', level: LogLevel = 'info') {
        this.name = name;
        this.level = level;
        this.currentLevel = Logger.levels[level] ?? Logger.levels.info;
    }

    /**
     * Set the logging level
     */
    setLevel(level: LogLevel): void {
        if (level in Logger.levels) {
            this.level = level;
            this.currentLevel = Logger.levels[level];
        }
    }

    /**
     * Check if a level should be logged
     */
    private shouldLog(level: LogLevel): boolean {
        return Logger.levels[level] >= this.currentLevel;
    }

    /**
     * Format log message with timestamp and level
     */
    private formatMessage(level: LogLevel, message: string, data?: LogDataValue): string {
        const timestamp = new Date().toISOString();
        const prefix = `[${timestamp}] ${level.toUpperCase()} [${this.name}]`;

        if (data !== null && data !== undefined) {
            return `${prefix} ${message} ${this.formatData(data)}`;
        }
        return `${prefix} ${message}`;
    }

    /**
     * Format additional data for logging
     */
    private formatData(data: LogDataValue): string {
        if (typeof data === 'object' && data !== null) {
            try {
                return JSON.stringify(data, null, 2);
            } catch (err) {
                return '[Circular/Invalid Object]';
            }
        }
        return String(data);
    }

    /**
     * Debug level logging - for detailed internal state
     */
    debug(message: string, data?: LogDataValue): void {
        if (this.shouldLog('debug')) {
            // Use process.stdout.write to bypass Jest console mocking during tests
            if (process.env.NODE_ENV === 'test' && !process.env.VERBOSE_TESTS) {
                process.stdout.write(this.formatMessage('debug', message, data) + '\n');
            } else {
                console.log(this.formatMessage('debug', message, data));
            }
        }
    }

    /**
     * Info level logging - for general information
     */
    info(message: string, data?: LogDataValue): void {
        if (this.shouldLog('info')) {
            console.log(this.formatMessage('info', message, data));
        }
    }

    /**
     * Warning level logging - for concerning but non-fatal issues
     */
    warn(message: string, data?: LogDataValue): void {
        if (this.shouldLog('warn')) {
            console.warn(this.formatMessage('warn', message, data));
        }
    }

    /**
     * Error level logging - for errors and failures
     */
    error(message: string, data?: LogDataValue): void {
        if (this.shouldLog('error')) {
            console.error(this.formatMessage('error', message, data));
        }
    }

    /**
     * Log connection events specifically
     */
    connection(event: string, details?: LogDataValue): void {
        this.info(`Connection ${event}`, details);
    }

    /**
     * Log message handling events
     */
    message(direction: 'sent' | 'received', messageType: string, messageId?: string | null): void {
        const details = messageId !== null && messageId !== undefined ? { messageId } : undefined;
        this.debug(`Message ${direction}: ${messageType}`, details);
    }

    /**
     * Log FSM protocol events
     */
    protocol(event: string, details?: LogDataValue): void {
        this.debug(`Protocol: ${event}`, details);
    }
}

export type { LogLevel, LogData, LogDataValue };
