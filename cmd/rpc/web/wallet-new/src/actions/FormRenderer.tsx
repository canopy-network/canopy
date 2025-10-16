import React from 'react'
import type { Field, FieldOp } from '@/manifest/types'
import { normalizeEvmAddress } from '@/core/address'
import { cx } from '@/ui/cx'
import { validateField } from './validators'
import { template } from '@/core/templater'
import { useSession } from '@/state/session'
import {useFieldDs} from "@/actions/useFieldsDs";

const looksLikeJson = (s: any) => typeof s === 'string' && /^\s*[\[{]/.test(s)
const jsonMaybe = (s: any) => { try { return JSON.parse(s) } catch { return s } }

const Grid: React.FC<{ cols: number; children: React.ReactNode }> = ({ cols, children }) => (
    <div className={cx('grid gap-4', `grid-cols-${cols}`)}>{children}</div>
)

type Props = {
    fields: Field[]
    value: Record<string, any>
    onChange: (patch: Record<string, any>) => void
    gridCols?: number
    /** ctx opcional extra: { fees, ds, ... }  */
    ctx?: Record<string, any>
}

const FieldFeatures: React.FC<{
    fieldId: string
    features?: FieldOp[]
    ctx: Record<string, any>
    setVal: (fieldId: string, v: any) => void
}> = ({ features, ctx, setVal, fieldId }) => {
    if (!features?.length) return null

    const resolve = (s?: any) => (typeof s === 'string' ? template(s, ctx) : s)

    const labelFor = (op: FieldOp) => {
        if (op.op === 'copy') return 'Copy'
        if (op.op === 'paste') return 'Paste'
        if (op.op === 'set') return 'Max'
        return op.op
    }

    const handle = async (op: FieldOp) => {
        switch (op.op) {
            case 'copy': {
                const txt = String(resolve(op.from) ?? '')
                await navigator.clipboard.writeText(txt)
                return
            }
            case 'paste': {
                const txt = await navigator.clipboard.readText()
                setVal(fieldId, txt)
                return
            }
            case 'set': {
                const v = resolve(op.value)
                setVal(op.field ?? fieldId, v)
                return
            }
        }
    }

    return (
        <div className="flex items-center  gap-2">
            {features.map((op) => (
                <button
                    key={op.id}
                    type="button"
                    onClick={() => handle(op)}
                    className="text-xs px-2 py-1 h-full rounded font-bold border border-primary text-primary hover:bg-primary hover:text-secondary"
                >
                    {labelFor(op)}
                </button>
            ))}
        </div>
    )
}

export default function FormRenderer({ fields, value, onChange, gridCols = 12, ctx }: Props) {
    const [errors, setErrors] = React.useState<Record<string, string>>({})
    const [localDs, setLocalDs] = React.useState<Record<string, any>>({})
    const { chain, account } = (window as any).__configCtx ?? {}
    const session = useSession()

    const templateContext = React.useMemo(() => ({
        form: value,
        chain,
        account,
        // 🔴 importante: merge con lo que venga en ctx
        ds: { ...(ctx?.ds || {}), ...localDs },
        ...(ctx || {}),
        session: { password: session?.password },
    }), [value, chain, account, ctx?.ds, ctx, session?.password, localDs])

    const resolveTemplate = React.useCallback(
        (s?: any) => (typeof s === 'string' ? template(s, templateContext) : s),
        [templateContext]
    )

    /** Normaliza fields con una key estable (tab:group:name) */
    const fieldsKeyed = React.useMemo(
        () =>
            fields.map((f: any) => ({
                ...f,
                __key: `${f.tab ?? 'default'}:${f.group ?? ''}:${f.name}`,
            })),
        [fields]
    )

    /** setVal + async validation */
    const setVal = React.useCallback(
        (fOrName: Field | string, v: any) => {
            const name = typeof fOrName === 'string' ? fOrName : (fOrName as any).name
            onChange({ [name]: v })

            void (async () => {
                const f = typeof fOrName === 'string'
                    ? (fieldsKeyed.find(x => x.name === fOrName) as Field | undefined)
                    : (fOrName as Field)

                const e = await validateField((f as any) ?? {}, v, { chain })
                setErrors((prev) =>
                    prev[name] === (e?.message ?? '') ? prev : { ...prev, [name]: e?.message ?? '' }
                )
            })()
        },
        [onChange, chain, fieldsKeyed]
    )

    const tabs = React.useMemo(
        () =>
            Array.from(new Set(fieldsKeyed.map((f: any) => f.tab).filter(Boolean))) as string[],
        [fieldsKeyed]
    )
    const [activeTab, setActiveTab] = React.useState(tabs[0] ?? 'default')
    const fieldsInTab = React.useCallback(
        (t?: string) => fieldsKeyed.filter((f: any) => (tabs.length ? f.tab === t : true)),
        [fieldsKeyed, tabs]
    )



    const renderControl = React.useCallback(
        (f: any) => {
            const common =
                'w-full bg-transparent border placeholder-text-muted text-white rounded px-3 py-2 rounded-md focus:outline-none'

            const border = errors[f.name] ? 'border-red-600' : 'border-muted-foreground border-opacity-50'
            const help = errors[f.name] || resolveTemplate(f.help)
            const v = value[f.name] ?? ''

            const dsField = useFieldDs(f, templateContext)
            const dsValue = dsField?.data

            React.useEffect(() => {
                if (f.ds && dsValue !== undefined) {
                    const dsKey = Object.keys(f.ds)[0]
                    setLocalDs(prev => {
                        if (JSON.stringify(prev?.[dsKey]) === JSON.stringify(dsValue)) return prev
                        return { ...prev, [dsKey]: dsValue }
                    })
                }
            }, [dsValue, f.ds])

            const wrap = (child: React.ReactNode) => (
                <div key={f.__key} className={cx(`col-span-${f.colSpan ?? 12}`)}>
                    <label className="block">
                        {resolveTemplate(f.label) && <div className="text-sm mb-1 text-text-muted ">{resolveTemplate(f.label)}</div>}
                        <div className="flex items-stretch gap-1">
                            {resolveTemplate(f.prefix) && (
                                <span className="px-2 py-2 bg-canopy-400 rounded">{resolveTemplate(f.prefix)}</span>
                            )}

                            {/* campo principal */}
                            {child}

                            {resolveTemplate(f.suffix) && (
                                <span className="px-2 py-2 bg-canopy-600 text-canopy-50 font-semibold rounded">
                  {resolveTemplate(f.suffix)}
                </span>
                            )}

                            {/* features (Copy/Max/Paste) */}
                            {f.features?.length ? (
                                <FieldFeatures
                                    fieldId={f.name}
                                    features={f.features}
                                    ctx={templateContext}
                                    setVal={setVal}
                                />
                            ) : null}
                        </div>

                        {help && (
                            <div
                                className={cx('text-xs mt-1', errors[f.name] ? 'text-red-400' : 'text-text-muted')}
                            >
                                {help}
                            </div>
                        )}
                    </label>
                </div>
            )

            /** TEXT & TEXTAREA */
            if (f.type === 'text' || f.type === 'textarea') {
                const Comp: any = f.type === 'text' ? 'input' : 'textarea'
                const resolved = resolveTemplate(f.value)
                const resolvedValue = resolveTemplate(f.value)
                const val = v === '' && resolvedValue != null
                    ? resolvedValue
                    : v || (dsValue?.amount ?? dsValue?.value ?? '')
                return wrap(
                    <Comp
                        className={cx(common, border)}
                        placeholder={resolveTemplate(f.placeholder)}
                        value={val}
                        readOnly={f.readOnly}
                        disabled={f.disabled}
                        onChange={(e: any) => setVal(f, e.target.value)}
                    />
                )
            }

            /** SELECT (una sola implementación) */
            if (f.type === 'select') {
                // f.options puede ser:
                // - array [{label,value}], o
                // - string (plantilla) que resuelve a array o JSON
                const raw = typeof f.options === 'string' ? resolveTemplate(f.options) : f.options
                const src = Array.isArray(raw) ? raw : looksLikeJson(raw) ? jsonMaybe(raw) : []
                const baseOpts = Array.isArray(src) ? src : []
                const opts = baseOpts.map((o: any, i: number) => {
                    const label =
                        f.optionLabel ? String(o?.[f.optionLabel] ?? '') : (typeof o?.label === 'string' ? resolveTemplate(o.label) : o?.label)
                    const value =
                        f.optionValue ? o?.[f.optionValue] : (typeof o?.value === 'string' ? resolveTemplate(o.value) : o?.value)
                    return { ...o, label, value, __k: `${f.name}-${String(value ?? i)}` }
                })
                // default value por plantilla
                const resolved = resolveTemplate(f.value)
                const val = v === '' && resolved != null ? resolved : v
                return wrap(
                    <select
                        className={cx(common, border)}
                        value={val}
                        readOnly={f.readOnly as any}
                        disabled={f.disabled}
                        onChange={(e) => setVal(f, e.target.value)}
                    >
                        <option value="" disabled>
                            {resolveTemplate(f.placeholder) ?? 'Choose...'}
                        </option>
                        {opts.map((o: any) => (
                            <option key={o.__k} value={o.value}>
                                {o.label}
                            </option>
                        ))}
                    </select>
                )
            }

            /** NUMBER / AMOUNT */
            if (f.type === 'number' || f.type === 'amount') {
                const resolved = resolveTemplate(f.value)
                const val = v === '' && resolved != null
                    ? resolved
                    : v || (dsValue?.amount ?? dsValue?.value ?? '')
                return wrap(
                    <input
                        type="number"
                        step="any"
                        className={cx(common, border)}
                        placeholder={resolveTemplate(f.placeholder)}
                        value={val ?? ''}
                        readOnly={f.readOnly}
                        disabled={f.disabled}
                        onChange={(e) => setVal(f, e.currentTarget.value)}
                        min={f.min}
                        max={f.max}
                    />
                )
            }

            /** ADDRESS (evm u otros) */
            if (f.type === 'address') {
                const fmt = f.format ?? 'evm'
                const { ok } =
                    fmt === 'evm' ? normalizeEvmAddress(String(v || '')) : { ok: true }
                const resolved = resolveTemplate(f.value)
                const val = v === '' && resolved != null ? resolved : v
                return wrap(
                    <input
                        className={cx(common, ok || !val ? border : 'border-red-600')}
                        placeholder={resolveTemplate(f.placeholder) ?? (fmt === 'evm' ? '0xabc... (or without 0x)' : 'address')}
                        value={val}
                        readOnly={f.readOnly}
                        disabled={f.disabled}
                        onChange={(e) => setVal(f, e.target.value)}
                    />
                )
            }

            return <div className="col-span-12">Unsupported field: {f.type}</div>
        },
        [errors, resolveTemplate, value, setVal, templateContext]
    )

    return (
        <>
            {tabs.length > 0 && (
                <div className="mb-3 flex gap-2 border-b border-neutral-800">
                    {tabs.map((t) => (
                        <button
                            key={t}
                            className={cx(
                                'px-3 py-2 -mb-px border-b-2',
                                activeTab === t
                                    ? 'border-emerald-400 text-emerald-400'
                                    : 'border-transparent text-neutral-400'
                            )}
                            onClick={() => setActiveTab(t)}
                        >
                            {t}
                        </button>
                    ))}
                </div>
            )}
            <Grid cols={gridCols}>
                {(tabs.length ? fieldsInTab(activeTab) : fieldsKeyed).map((f: any) => renderControl(f))}
            </Grid>
        </>
    )
}
