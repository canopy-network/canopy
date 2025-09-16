module.exports = {
    root: true,
    env: {
        node: true,
        es2022: true,
        jest: true
    },
    extends: [
        'eslint:recommended',
        '@typescript-eslint/recommended',
        '@typescript-eslint/recommended-requiring-type-checking',
        'plugin:node/recommended',
        'plugin:import/recommended',
        'plugin:import/typescript',
        'plugin:security/recommended',
        'prettier' // Must be last to override other formatting rules
    ],
    parser: '@typescript-eslint/parser',
    parserOptions: {
        ecmaVersion: 2022,
        sourceType: 'module',
        project: './tsconfig.json',
        tsconfigRootDir: __dirname
    },
    plugins: ['@typescript-eslint', 'node', 'import', 'security'],
    settings: {
        'import/resolver': {
            typescript: {
                alwaysTryTypes: true,
                project: './tsconfig.json'
            },
            node: {
                extensions: ['.js', '.jsx', '.ts', '.tsx']
            }
        },
        node: {
            tryExtensions: ['.js', '.json', '.ts']
        }
    },
    rules: {
        // TypeScript specific rules
        '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
        '@typescript-eslint/explicit-function-return-type': [
            'warn',
            {
                allowExpressions: true,
                allowTypedFunctionExpressions: true
            }
        ],
        '@typescript-eslint/no-explicit-any': 'warn',
        '@typescript-eslint/prefer-nullish-coalescing': 'error',
        '@typescript-eslint/prefer-optional-chain': 'error',
        '@typescript-eslint/no-unnecessary-type-assertion': 'error',
        '@typescript-eslint/no-non-null-assertion': 'warn',
        '@typescript-eslint/ban-ts-comment': [
            'error',
            {
                'ts-expect-error': 'allow-with-description',
                'ts-ignore': false,
                'ts-nocheck': false,
                'ts-check': false
            }
        ],
        '@typescript-eslint/consistent-type-imports': [
            'error',
            {
                prefer: 'type-imports',
                disallowTypeAnnotations: false
            }
        ],
        '@typescript-eslint/consistent-type-definitions': ['error', 'interface'],
        '@typescript-eslint/array-type': ['error', { default: 'array-simple' }],
        '@typescript-eslint/prefer-readonly': 'error',

        // Import rules
        'import/order': [
            'error',
            {
                groups: [
                    'builtin',
                    'external',
                    'internal',
                    'parent',
                    'sibling',
                    'index',
                    'object',
                    'type'
                ],
                'newlines-between': 'never',
                alphabetize: {
                    order: 'asc',
                    caseInsensitive: true
                }
            }
        ],
        'import/no-unresolved': 'error',
        'import/no-cycle': 'error',
        'import/no-self-import': 'error',
        'import/no-useless-path-segments': 'error',

        // Node.js rules
        'node/no-missing-import': 'off', // TypeScript handles this
        'node/no-unsupported-features/es-syntax': 'off', // We use TypeScript
        'node/no-unpublished-import': [
            'error',
            {
                allowModules: ['@types/jest', '@types/node']
            }
        ],

        // General rules
        'no-console': 'off', // Allow console in server applications
        'no-debugger': 'error',
        'no-alert': 'error',
        'no-var': 'error',
        'prefer-const': 'error',
        'prefer-template': 'error',
        'prefer-arrow-callback': 'error',
        'arrow-spacing': 'error',
        'no-duplicate-imports': 'error',
        'no-useless-rename': 'error',
        'object-shorthand': 'error',
        'prefer-destructuring': [
            'error',
            {
                array: true,
                object: true
            },
            {
                enforceForRenamedProperties: false
            }
        ],

        // Security rules
        'security/detect-object-injection': 'off', // Too many false positives
        'security/detect-non-literal-fs-filename': 'off' // Common in server apps
    },
    overrides: [
        {
            // Test files
            files: ['**/*.test.ts', '**/*.spec.ts', '**/test/**/*.ts'],
            env: {
                jest: true
            },
            rules: {
                '@typescript-eslint/no-explicit-any': 'off',
                '@typescript-eslint/no-non-null-assertion': 'off',
                '@typescript-eslint/no-unsafe-assignment': 'off',
                '@typescript-eslint/no-unsafe-member-access': 'off',
                'node/no-unpublished-import': 'off'
            }
        },
        {
            // Configuration files
            files: ['*.config.js', '*.config.ts', '.eslintrc.js'],
            env: {
                node: true
            },
            rules: {
                'node/no-unpublished-require': 'off',
                '@typescript-eslint/no-var-requires': 'off'
            }
        },
        {
            // Generated protobuf files
            files: ['src/proto/**/*.js', 'src/proto/**/*.d.ts'],
            rules: {
                // Disable all rules for generated files
                '@typescript-eslint/no-explicit-any': 'off',
                '@typescript-eslint/no-unused-vars': 'off',
                'import/no-unresolved': 'off',
                'node/no-missing-import': 'off'
            }
        }
    ],
    ignorePatterns: [
        'dist/',
        'coverage/',
        'node_modules/',
        'test-results/',
        '*.js', // Ignore JS files in root (config files are handled by overrides)
        'src/proto/index.js' // Generated protobuf file
    ]
};
