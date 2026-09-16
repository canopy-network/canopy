const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { test } = require('node:test')
const { runInNewContext } = require('node:vm')
const ts = require('typescript')

const source = ts.transpileModule(
    readFileSync(`${__dirname}/../src/lib/clipboard.ts`, 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } },
).outputText

function setup({ secure = false, writeText, copyResult = true, copyError } = {}) {
    const calls = []
    class HTMLElement {
        focus() { calls.push('focus') }
    }
    const textarea = {
        style: {},
        setAttribute() {},
        select() { calls.push(['select', this.value]) },
        remove() { calls.push('remove') },
    }
    const context = {
        exports: {},
        HTMLElement,
        window: { isSecureContext: secure },
        navigator: { clipboard: writeText ? { writeText } : undefined },
        document: {
            activeElement: new HTMLElement(),
            createElement(tag) {
                assert.equal(tag, 'textarea')
                return textarea
            },
            body: { appendChild() { calls.push('append') } },
            execCommand(command) {
                assert.equal(command, 'copy')
                calls.push('copy')
                if (copyError) throw copyError
                return copyResult
            },
        },
    }
    runInNewContext(source, context)
    return { copyText: context.exports.copyText, calls }
}

const text = 'full identifier\nwith multiple lines'
const fallbackCalls = ['append', ['select', text], 'copy', 'remove', 'focus']

test('awaits the Clipboard API on secure origins without using a textarea', async () => {
    let finish
    const written = []
    const { copyText, calls } = setup({
        secure: true,
        writeText: value => {
            written.push(value)
            return new Promise(resolve => { finish = resolve })
        },
    })
    let complete = false
    const pending = copyText(text).then(() => { complete = true })
    await Promise.resolve()
    assert.equal(complete, false)
    finish()
    await pending
    assert.deepEqual(written, [text])
    assert.deepEqual(calls, [])
})

test('copies the full text on HTTP without navigator.clipboard', async () => {
    const { copyText, calls } = setup()
    await copyText(text)
    assert.deepEqual(calls, fallbackCalls)
})

test('falls back when a secure origin has no Clipboard API', async () => {
    const { copyText, calls } = setup({ secure: true })
    await copyText(text)
    assert.deepEqual(calls, fallbackCalls)
})

test('falls back when the Clipboard API rejects', async () => {
    const { copyText, calls } = setup({
        secure: true,
        writeText: async () => { throw new Error('Permission denied') },
    })
    await copyText(text)
    assert.deepEqual(calls, fallbackCalls)
})

for (const failure of [{ copyResult: false }, { copyError: new Error('Copy blocked') }]) {
    test(`rejects and cleans up when fallback ${failure.copyError ? 'throws' : 'returns false'}`, async () => {
        const { copyText, calls } = setup(failure)
        await assert.rejects(copyText(text), /Unable to copy text|Copy blocked/)
        assert.deepEqual(calls, fallbackCalls)
    })
}
