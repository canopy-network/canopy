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
function evalExpr(expr: string, ctx: any): any {
    // 🔒 seguridad básica
    const banned = /(constructor|prototype|__proto__|globalThis|window|document|import|Function|eval)\b/
    if (banned.test(expr)) throw new Error('templater: forbidden token')

    // 🔧 soporta funciones tipo formatToCoin<{{...}}>
    const funcMatch = expr.match(/^(\w+)<([\s\S]*)>$/)
    if (funcMatch) {
        const [, fnName, innerExpr] = funcMatch
        const innerVal = template(innerExpr, ctx)
        const fn = templateFns[fnName]
        if (typeof fn === 'function') {
            try {
                return fn(innerVal)
            } catch (e) {
                console.error(`template fn ${fnName} error:`, e)
                return ''
            }
        }
        console.warn(`template function not found: ${fnName}`)
        return ''
    }

    // 💡 NUEVO: detectar si es una expresión JS (contiene operadores)
    const isExpression = /[<>=!+\-*/%&|?:]/.test(expr)

    if (isExpression) {
        try {
            const argNames = Object.keys(ctx)
            const argValues = Object.values(ctx)

            // Ejemplo: new Function("form","chain","account", "return form.isDelegate === false")
            const fn = new Function(...argNames, `return (${expr});`)
            return fn(...argValues)
        } catch (e) {
            console.warn('template eval error:', e)
            return ''
        }
    }

    // 🧭 fallback: acceso tipo path (form.a.b)
    const path = expr.split('.').map(s => s.trim()).filter(Boolean)
    let val: any = ctx
    for (const p of path) val = val?.[p]

    return val
}

export function template(str: unknown, ctx: any): string {
    if (str == null) return ''
    const input = String(str)


    const out = replaceBalanced(input, (expr) => evalExpr(expr, ctx))
    return out
}

export function templateAny(tpl: any, ctx: Record<string, any> = {}): any {
    if (tpl == null) return tpl
    if (typeof tpl !== 'string') return tpl

    const m = tpl.match(/^\s*\{\{([\s\S]+?)\}\}\s*$/)
    if (m) {
        const expr = m[1]
        try { return evalExpr(expr, ctx) } catch { /* cae al modo string */ }
    }

    return tpl.replace(/\{\{([\s\S]+?)\}\}/g, (_m, expr) => {
        try {
            const val = evalExpr(expr, ctx)
            return val == null ? '' : String(val)
        } catch {
            return ''
        }
    })
}

export function templateBool(tpl: any, ctx: Record<string, any> = {}): boolean {
    const v = templateAny(tpl, ctx)
    return toBool(v)
}


export function toBool(v: any): boolean {
    if (typeof v === 'boolean') return v
    if (typeof v === 'number') return v !== 0 && !Number.isNaN(v)
    if (v == null) return false
    if (Array.isArray(v)) return v.length > 0
    if (typeof v === 'object') return Object.keys(v).length > 0
    const s = String(v).trim().toLowerCase()
    if (s === '' || s === '0' || s === 'false' || s === 'no' || s === 'off' || s === 'null' || s === 'undefined') return false
    return true
}