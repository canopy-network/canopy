import React from 'react'
import type { Field, FieldOp } from '@/manifest/types'
import { cx } from '@/ui/cx'
import { validateField } from './validators'
import { template } from '@/core/templater'
import { useSession } from '@/state/session'
import {FieldControl} from "@/actions/FieldControl";
import { motion } from "framer-motion"

const looksLikeJson = (s: any) => typeof s === 'string' && /^\s*[\[{]/.test(s)
const jsonMaybe = (s: any) => { try { return JSON.parse(s) } catch { return s } }

const Grid: React.FC<{ cols: number; children: React.ReactNode }> = ({ cols, children }) => (
    <motion.div className={cx('grid gap-4', `grid-cols-${cols}`)}>{children}</motion.div>
)

type Props = {
    fields: Field[]
    value: Record<string, any>
    onChange: (patch: Record<string, any>) => void
    gridCols?: number
    /** ctx opcional extra: { fees, ds, ... }  */
    ctx?: Record<string, any>
    onErrorsChange?: (errors: Record<string,string>, hasErrors: boolean) => void   // 👈 NUEVO

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

export default function FormRenderer({ fields, value, onChange, gridCols = 12, ctx, onErrorsChange }: Props) {
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

                const e = await validateField((f as any) ?? {}, v, templateContext)
                setErrors((prev) =>
                    prev[name] === (e?.message ?? '') ? prev : { ...prev, [name]: e?.message ?? '' }
                )
            })()
        },
        [onChange, chain, fieldsKeyed]
    )

    const hasActiveErrors = React.useMemo(() => {
        const anyMsg = Object.values(errors).some((m) => !!m)
        const requiredMissing = fields.some((f) => f.required && (value[f.name] == null || value[f.name] === ''))
        return anyMsg || requiredMissing
    }, [errors, fields, value])

    React.useEffect(() => {
        onErrorsChange?.(errors, hasActiveErrors)
    }, [errors, hasActiveErrors, onErrorsChange])

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
                {(tabs.length ? fieldsInTab(activeTab) : fieldsKeyed).map((f: any) => (
                    <FieldControl
                        key={f.__key}
                        f={f}
                        value={value}
                        errors={errors}
                        templateContext={templateContext}
                        setVal={setVal}
                        setLocalDs={setLocalDs}
                    />
                ))}
            </Grid>
        </>
    )
}
