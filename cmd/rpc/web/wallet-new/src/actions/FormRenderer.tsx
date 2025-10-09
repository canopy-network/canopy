import React from 'react'
import type { Field } from '@/manifest/types'
import { normalizeEvmAddress } from '@/core/address'
import { cx } from '@/ui/cx'
import { validateField } from './validators'
import { template } from '@/core/templater'
import { useSession } from '@/state/session'

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
    ctx?: Record<string, any>
}

export default function FormRenderer({ fields, value, onChange, gridCols = 12, ctx }: Props) {
    const [errors, setErrors] = React.useState<Record<string, string>>({})
    const { chain, account } = (window as any).__configCtx ?? {}
    const session = useSession()

    const tctx = React.useMemo(
        () => ({ form: value, chain, account, ...(ctx ?? {}), session: { password: session?.password } }),
        [value, chain, account, ctx, session?.password]
    )
    const tt = React.useCallback((s?: any) => (typeof s === 'string' ? template(s, tctx) : s), [tctx])


    const fieldsKeyed = React.useMemo(
        () => fields.map((f: any) => ({ ...f, __key: `${f.tab ?? 'default'}:${f.group ?? ''}:${f.name}` })),
        [fields]
    )

    const setVal = React.useCallback((f: Field, v: any) => {
        onChange({ [f.name]: v })
        // valida async sin bloquear tipeo
        void (async () => {
            const e = await validateField(f as any, v, { chain })
            setErrors(prev => (prev[f.name] === (e?.message ?? '') ? prev : { ...prev, [f.name]: e?.message ?? '' }))
        })()
    }, [onChange, chain])

    const tabs = React.useMemo(
        () => Array.from(new Set(fieldsKeyed.map((f: any) => f.tab).filter(Boolean))) as string[],
        [fieldsKeyed]
    )
    const [activeTab, setActiveTab] = React.useState(tabs[0] ?? 'default')

    const fieldsInTab = React.useCallback(
        (t?: string) => fieldsKeyed.filter((f: any) => (tabs.length ? f.tab === t : true)),
        [fieldsKeyed, tabs]
    )

    const renderControl = React.useCallback((f: any) => {
        const common = 'w-full  bg-transparent  border placeholder-text-muted text-white  rounded px-3 py-2 rounded-md focus:outline-none'
        const border = errors[f.name] ? 'border-red-600' : 'border-muted-foreground border-opacity-50'
        const help = errors[f.name] || tt(f.help)
        const v = value[f.name] ?? ''



        const wrap = (child: React.ReactNode) => (
            <div key={f.__key} className={cx(`col-span-${f.colSpan ?? 12}`)}>
                <label className="block">
                    {tt(f.label) && <div className="text-sm mb-1 text-text-muted ">{tt(f.label)}</div>}
                    <div className="flex items-stretch gap-1">
                        {tt(f.prefix) && <span className="px-2 py-2 bg-canopy-400 rounded">{tt(f.prefix)}</span>}
                        {child}
                        {tt(f.suffix) && <span className="px-2 py-2  bg-canopy-600 text-canopy-50 font-semibold  rounded">{tt(f.suffix)}</span>}
                    </div>
                    {help && (
                        <div className={cx('text-xs mt-1', errors[f.name] ? 'text-red-400' : 'text-text-muted')}>
                            {help}
                        </div>
                    )}
                </label>
            </div>
        )

        if (f.type === 'text' || f.type === 'textarea') {
            const Comp: any = f.type === 'text' ? 'input' : 'textarea'
            return wrap(
                <Comp
                    className={cx(common, border)}
                    placeholder={tt(f.placeholder)}
                    value={v}
                    disabled={f.disabled}
                    onChange={(e: any) => setVal(f, e.target.value)}
                />
            )
        }

        if (f.type === 'select') {
            const raw = typeof f.options === 'string' ? tt(f.options) : f.options
            const src = Array.isArray(raw) ? raw : (looksLikeJson(raw) ? jsonMaybe(raw) : [])
            const baseOpts = Array.isArray(src) ? src : []
            const opts = baseOpts.map((o: any, i: number) => {
                const label = f.optionLabel ? String(o?.[f.optionLabel] ?? '') : tt(o?.label)
                const value = f.optionValue ? (o?.[f.optionValue]) : (typeof o?.value === 'string' ? tt(o.value) : o?.value)
                return { ...o, label, value, __k: `${f.name}-${String(value ?? i)}` }
            })
            return wrap(
                <select
                    className={cx(common, border)}
                    value={v}
                    disabled={f.disabled}
                    onChange={(e) => setVal(f, e.target.value)}
                >
                    <option value="" disabled>{tt(f.placeholder) ?? 'Choose...'}</option>
                    {opts.map((o: any) => <option key={o.__k} value={o.value}>{o.label}</option>)}
                </select>
            )
        }
        if (f.type === 'coin') {

        }

        if (f.type === 'number') {
            return wrap(
                <input
                    type="number"
                    step="any"
                    className={cx(common, border)}
                    placeholder={tt(f.placeholder)}
                    value={v ?? ''}
                    disabled={f.disabled}
                    onChange={(e) => setVal(f, e.currentTarget.value)}
                />
            )
        }

        if (f.type === 'address') {
            const fmt = f.format ?? 'evm'
            const { ok } = fmt === 'evm' ? normalizeEvmAddress(String(v || '')) : { ok: true }
            return wrap(
                <input
                    className={cx(common, ok || !v ? border : 'border-red-600')}
                    placeholder={tt(f.placeholder) ?? (fmt === 'evm' ? '0xabc... (or without 0x)' : 'address')}
                    value={v}
                    disabled={f.disabled}
                    onChange={(e) => setVal(f, e.target.value)}
                />
            )
        }

        if (f.type === 'select') {
            const opts = (f.options ?? []).map((o: any, i: number) => ({
                ...o,
                label: tt(o.label),
                value: typeof o.value === 'string' ? tt(o.value) : o.value,
                __k: `${f.name}-${String(o.value ?? i)}`
            }))
            return wrap(
                <select
                    className={cx(common, border)}
                    value={v}
                    disabled={f.disabled}
                    onChange={(e) => setVal(f, e.target.value)}
                >
                    <option value="" disabled>{tt(f.placeholder) ?? 'Choose...'}</option>
                    {opts.map((o: any) => <option key={o.__k} value={o.value}>{o.label}</option>)}
                </select>
            )
        }

        return <div className="col-span-12">Unsupported field: {f.type}</div>
    }, [errors, tt, value, setVal])

    return (
        <>
            {tabs.length > 0 && (
                <div className="mb-3 flex gap-2 border-b border-neutral-800">
                    {tabs.map((t) => (
                        <button
                            key={t}
                            className={cx('px-3 py-2 -mb-px border-b-2',
                                activeTab === t ? 'border-emerald-400 text-emerald-400' : 'border-transparent text-neutral-400')}
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
