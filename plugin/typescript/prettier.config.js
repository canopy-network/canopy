module.exports = {
    // Basic formatting
    printWidth: 100,
    tabWidth: 2,
    useTabs: false,
    semi: true,
    singleQuote: true,
    quoteProps: 'as-needed',

    // Trailing commas (ES5 safe)
    trailingComma: 'es5',

    // Bracket spacing
    bracketSpacing: true,
    bracketSameLine: false,

    // Arrow function parentheses
    arrowParens: 'always',

    // Range formatting
    rangeStart: 0,
    rangeEnd: Infinity,

    // Parser configuration
    requirePragma: false,
    insertPragma: false,
    proseWrap: 'preserve',

    // HTML whitespace sensitivity
    htmlWhitespaceSensitivity: 'css',

    // Vue files
    vueIndentScriptAndStyle: false,

    // End of line
    endOfLine: 'lf',

    // Embedded language formatting
    embeddedLanguageFormatting: 'auto',

    // Single attribute per line in HTML/JSX
    singleAttributePerLine: false,

    // Override for specific file types
    overrides: [
        {
            files: '*.json',
            options: {
                printWidth: 200,
                tabWidth: 2
            }
        },
        {
            files: '*.md',
            options: {
                proseWrap: 'always',
                printWidth: 80
            }
        },
        {
            files: ['*.yml', '*.yaml'],
            options: {
                singleQuote: false
            }
        }
    ]
};
