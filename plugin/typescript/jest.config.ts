import type { Config } from 'jest';

const config: Config = {
    // Use ts-jest preset for TypeScript support
    preset: 'ts-jest/presets/default-esm',

    // Test environment
    testEnvironment: 'node',

    // Enable ESM support
    extensionsToTreatAsEsm: ['.ts'],
    globals: {
        'ts-jest': {
            useESM: true,
            tsconfig: {
                module: 'ESNext',
                target: 'ES2022'
            }
        }
    },

    // Module name mapping for path aliases
    moduleNameMapping: {
        '^@/(.*)$': '<rootDir>/src/$1',
        '^@/config$': '<rootDir>/src/config/index.ts',
        '^@/core$': '<rootDir>/src/core/index.ts',
        '^@/network$': '<rootDir>/src/network/index.ts',
        '^@/utils$': '<rootDir>/src/utils/index.ts'
    },

    // Test file patterns
    testMatch: [
        '**/test/**/*.test.ts',
        '**/test/**/*.spec.ts',
        '**/__tests__/**/*.ts',
        '**/*.test.ts',
        '**/*.spec.ts'
    ],

    // Files to ignore
    testPathIgnorePatterns: ['/node_modules/', '/dist/', '/coverage/'],

    // Module file extensions
    moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],

    // Transform settings
    transform: {
        '^.+\\.ts$': [
            'ts-jest',
            {
                useESM: true
            }
        ]
    },

    // Setup files
    setupFilesAfterEnv: ['<rootDir>/test/setup.ts'],

    // Coverage settings
    collectCoverageFrom: [
        'src/**/*.ts',
        '!src/**/*.d.ts',
        '!src/proto/**', // Exclude generated protobuf files
        '!src/main.ts', // Exclude main entry point from coverage
        '!src/**/*.test.ts',
        '!src/**/*.spec.ts'
    ],

    coverageDirectory: 'coverage',
    coverageReporters: ['text', 'lcov', 'html', 'json-summary', 'clover'],

    coverageThreshold: {
        global: {
            branches: 80,
            functions: 80,
            lines: 80,
            statements: 80
        }
    },

    // Test timeout (30 seconds for integration tests)
    testTimeout: 30000,

    // Clear mocks between tests
    clearMocks: true,
    restoreMocks: true,
    resetMocks: false,

    // Verbose output
    verbose: true,

    // Error handling
    errorOnDeprecated: true,

    // Test result processors
    reporters: [
        'default',
        [
            'jest-junit',
            {
                outputDirectory: 'test-results',
                outputName: 'junit.xml',
                classNameTemplate: '{classname}',
                titleTemplate: '{title}',
                ancestorSeparator: ' › ',
                usePathForSuiteName: true
            }
        ]
    ],

    // Watch settings
    watchman: true,
    watchPathIgnorePatterns: ['/node_modules/', '/dist/', '/coverage/'],

    // Performance settings
    maxWorkers: '50%',
    maxConcurrency: 5,

    // Bail settings (stop after first test failure in CI)
    bail: process.env.CI ? 1 : 0,

    // Cache settings
    cache: true,
    cacheDirectory: '<rootDir>/node_modules/.cache/jest',

    // Notify settings
    notify: false,
    notifyMode: 'failure-change'
};

export default config;
