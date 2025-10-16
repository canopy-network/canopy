import { templateFns } from './templaterFunctions'

function replaceBalanced(input: string, resolver: (expr: string) => string): string {
    let out = ''
    let i = 0
    while (i < input.length) {
        const start = input.indexOf('{{', i)
        if (start === -1) {
            out += input.slice(i)
            break
        }
        // texto antes del bloque
        out += input.slice(i, start)

        // buscar cierre balanceado
        let j = start + 2
        let depth = 1
        while (j < input.length && depth > 0) {
            if (input.startsWith('{{', j)) {
                depth += 1
                j += 2
                continue
            }
            if (input.startsWith('}}', j)) {
                depth -= 1
                j += 2
                if (depth === 0) break
                continue
            }
            j += 1
        }

        // si no se cerró, copia resto y corta
        if (depth !== 0) {
            out += input.slice(start)
            break
        }

        const exprRaw = input.slice(start + 2, j - 2)
        const replacement = resolver(exprRaw.trim())
        out += replacement
        i = j
    }
    return out
}

/** Evalúa una expresión: función tipo fn<...> o ruta a datos a.b.c */
function evalExpr(expr: string, ctx: any): string {
    // funciones: ej. formatToCoin<{{ds.account.amount}}>
    const funcMatch = expr.match(/^(\w+)<([\s\S]*)>$/)
    if (funcMatch) {
        const [, fnName, innerExpr] = funcMatch
        // evalúa el interior tal cual (puede contener {{...}} anidados)
        const innerVal = template(innerExpr, ctx)
        const fn = templateFns[fnName]
        if (typeof fn === 'function') {
            try {
                return String(fn(innerVal))
            } catch (e) {
                console.error(`template fn ${fnName} error:`, e)
                return ''
            }
        }
        console.warn(`template function not found: ${fnName}`)
        return ''
    }

    // ruta normal: a.b.c
    const path = expr.split('.').map(s => s.trim()).filter(Boolean)
    let val: any = ctx
    for (const p of path) val = val?.[p]

    if (val == null) return ''
    if (typeof val === 'object') {
        try { return JSON.stringify(val) } catch { return '' }
    }
    return String(val)
}

export function template(str: unknown, ctx: any): string {
    if (str == null) return ''
    const input = String(str)


    const out = replaceBalanced(input, (expr) => evalExpr(expr, ctx))
    return out
}
