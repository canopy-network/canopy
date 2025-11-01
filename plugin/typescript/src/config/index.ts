import { promises as fs } from 'fs';
import { join } from 'path';

/**
 * Configuration options for creating a Config instance
 */
interface ConfigOptions {
    readonly chainId?: number;
    readonly dataDirPath?: string;
}

/**
 * Serializable configuration data for JSON persistence
 */
interface ConfigData {
    readonly chainId: number;
    readonly dataDirPath: string;
}

/**
 * Configuration management for the Canopy blockchain plugin
 * Provides type-safe configuration handling with validation
 */
export default class Config {
    public readonly chainId: number;
    public readonly dataDirPath: string;

    private static readonly DEFAULT_CHAIN_ID = 1 as const;
    private static readonly DEFAULT_DATA_DIR = '/tmp/plugin/' as const;

    constructor(options: ConfigOptions = {}) {
        this.chainId = options.chainId ?? Config.DEFAULT_CHAIN_ID;
        this.dataDirPath = options.dataDirPath ?? Config.DEFAULT_DATA_DIR;

        // Validate configuration
        this.validate();
    }

    /**
     * Validate configuration parameters
     * @throws {Error} If configuration is invalid
     */
    private validate(): void {
        if (!Number.isInteger(this.chainId) || this.chainId < 1) {
            throw new Error(`Invalid chainId: ${this.chainId}. Must be a positive integer.`);
        }

        if (typeof this.dataDirPath !== 'string' || this.dataDirPath.trim() === '') {
            throw new Error(
                `Invalid dataDirPath: ${this.dataDirPath}. Must be a non-empty string.`
            );
        }
    }

    /**
     * Create default configuration
     * Create default configuration
     */
    static defaultConfig(): Config {
        return new Config({
            chainId: Config.DEFAULT_CHAIN_ID,
            dataDirPath: join(Config.DEFAULT_DATA_DIR)
        });
    }

    /**
     * Load configuration from JSON file
     * Load configuration from a JSON file
     * @param filepath - Path to the configuration file
     * @returns Promise resolving to a Config instance
     * @throws {Error} If file cannot be read or parsed
     */
    static async fromFile(filepath: string): Promise<Config> {
        if (typeof filepath !== 'string' || filepath.trim() === '') {
            throw new Error('Filepath must be a non-empty string');
        }

        try {
            const fileBytes = await fs.readFile(filepath, 'utf8');
            const configData = JSON.parse(fileBytes) as Partial<ConfigData>;

            // Start with default config and override with file data
            const defaultConfig = Config.defaultConfig();

            return new Config({
                chainId: configData.chainId ?? defaultConfig.chainId,
                dataDirPath: configData.dataDirPath ?? defaultConfig.dataDirPath
            });
        } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            throw new Error(`Failed to load config from ${filepath}: ${error.message}`);
        }
    }

    /**
     * Save configuration to JSON file
     * @param filepath - Path where to save the configuration
     * @throws {Error} If file cannot be written
     */
    async saveToFile(filepath: string): Promise<void> {
        if (typeof filepath !== 'string' || filepath.trim() === '') {
            throw new Error('Filepath must be a non-empty string');
        }

        const configData: ConfigData = {
            chainId: this.chainId,
            dataDirPath: this.dataDirPath
        };

        try {
            await fs.writeFile(filepath, JSON.stringify(configData, null, 2), 'utf8');
        } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            throw new Error(`Failed to save config to ${filepath}: ${error.message}`);
        }
    }

    /**
     * Create a copy of this configuration with updated values
     * @param updates - Partial configuration options to merge
     * @returns New Config instance with updated values
     */
    update(updates: Partial<ConfigOptions>): Config {
        return new Config({
            chainId: updates.chainId ?? this.chainId,
            dataDirPath: updates.dataDirPath ?? this.dataDirPath
        });
    }

    /**
     * Convert configuration to plain object for serialization
     */
    toJSON(): ConfigData {
        return {
            chainId: this.chainId,
            dataDirPath: this.dataDirPath
        };
    }

    /**
     * Create a string representation of the configuration
     */
    toString(): string {
        return `Config(chainId=${this.chainId}, dataDirPath="${this.dataDirPath}")`;
    }

    /**
     * Check if this configuration equals another
     */
    equals(other: Config): boolean {
        return (
            other instanceof Config &&
            this.chainId === other.chainId &&
            this.dataDirPath === other.dataDirPath
        );
    }
}

// Export types for external use
export type { ConfigOptions, ConfigData };
