import { template } from '@/core/templater'
import type { Action, Field, Manifest } from '@/manifest/types'

/** Lee los fields declarados en el manifest para la acción */
export const getFieldsFromAction = (action?: Action): Field[] =>
    Array.isArray(action?.form?.fields) ? (action!.form!.fields as Field[]) : []

/** Hints por nombre para normalizar valores numéricos/booleanos */
const NUMERIC_HINTS = new Set(['amount','receiveAmount','fee','gas','gasPrice'])
const BOOL_HINTS    = new Set(['delegate','earlyWithdrawal','submit'])

/** Normaliza el form según Fields + hints:
 * - number: convierte "1,234.56" -> 1234.56
 * - boolean (por nombre): 'true'/'false' -> boolean
 */
export function normalizeFormForAction(action: Action | undefined, form: Record<string, any>) {
    const out: Record<string, any> = { ...form }
    const fields = (action?.form?.fields ?? []) as Field[]

    const asNum  = (v:any) => {
        if (v === '' || v == null) return v
        const s = String(v).replace(/,/g, '')
        const n = Number(s)
        return Number.isNaN(n) ? v : n
    }
    const asBool = (v:any) =>
        v === true || v === 'true' || v === 1 || v === '1' || v === 'on'

    for (const f of fields) {
        const n = f?.name
        if (n == null || !(n in out)) continue

        // por tipo
        if (f.type === 'number' || NUMERIC_HINTS.has(n)) out[n] = asNum(out[n])
        // por “hint” de nombre (p.ej. select true/false)
        if (BOOL_HINTS.has(n)) out[n] = asBool(out[n])
    }
    return out
}

/** Contexto para construir payload desde el manifest */
export type BuildPayloadCtx = {
    form: Record<string, any>
    chain?: any
    session?: { password?: string }
    fees?: { effective?: number | string }
    extra?: Record<string, any>
}

/** Interpola el payload del manifest (rpc.payload) con template(...) */
export function buildPayloadFromAction(action: Action | undefined, ctx: BuildPayloadCtx) {
    const payloadTmpl = (action as any)?.rpc?.payload ?? {}
    return template(payloadTmpl, {
        ...ctx.extra,
        form: ctx.form,
        chain: ctx.chain,
        session: ctx.session,
        fees: ctx.fees,
    })
}

/** Construye el summary de confirmación con template(...) */
export function buildConfirmSummary(
    action: Action | undefined,
    data: { form: Record<string, any>; chain?: any; fees?: { effective?: number | string } }
) {
    const items = action?.confirm?.summary ?? []
    return items.map(s => ({ label: s.label, value: template(s.value, data) }))
}

/** Selección de Quick Actions usando tags + prioridad */
export function selectQuickActions(manifest: Manifest | undefined, chain: any, max?: number) {
    const limit = max ?? manifest?.ui?.quickActions?.max ?? 8
    const hasFeature = (a: Action) => !a.requiresFeature || chain?.features?.includes(a.requiresFeature)
    const rank = (a: Action) => (typeof a.priority === 'number' ? a.priority : (typeof a.order === 'number' ? a.order : 0))

    return (manifest?.actions ?? [])
        .filter(a => !a.hidden && Array.isArray(a.tags) && a.tags.includes('quick'))
        .filter(hasFeature)
        .sort((a, b) => rank(b) - rank(a))
        .slice(0, limit)
}
