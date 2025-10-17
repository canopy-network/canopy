import React from 'react'
import type { Field, FieldOp, SelectField, SourceRef } from '@/manifest/types'
import { useFieldDs } from '@/actions/useFieldsDs'
import { template } from '@/core/templater'
import { cx } from '@/ui/cx'

type Props = {
    f: Field
    value: Record<string, any>
    errors: Record<string, string>
    templateContext: Record<string, any>
    setVal: (field: Field | string, v: any) => void
    setLocalDs?: React.Dispatch<React.SetStateAction<Record<string, any>>>
}

const getByPath = (obj: any, selector?: string) => {
    if (!selector || !obj) return obj
    return selector.split('.').reduce((acc, k) => acc?.[k], obj)
}

const toOptions = (raw: any): Array<{ label: string; value: string }> => {
    if (!raw) return []
    // array de strings
    if (Array.isArray(raw) && raw.every((x) => typeof x === 'string')) {
        return raw.map((s) => ({ label: s, value: s }))
    }
    // array de objetos
    if (Array.isArray(raw) && raw.every((x) => typeof x === 'object')) {
        return raw.map((o, i) => ({
            label:
                o.label ??
                o.name ??
                o.id ??
                o.value ??
                o.address ??
                String(i + 1),
            value: String(o.value ?? o.id ?? o.address ?? o.key ?? i),
        }))
    }
    // objeto tipo map
    if (typeof raw === 'object') {
        return Object.entries(raw).map(([k, v]) => ({
            label: String((v as any)?.label ?? (v as any)?.name ?? k),
            value: String((v as any)?.value ?? k),
        }))
    }
    return []
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
        <div className="flex items-center gap-2">
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

export const FieldControl: React.FC<Props> = ({
                                                  f,
                                                  value,
                                                  errors,
                                                  templateContext,
                                                  setVal,
                                                  setLocalDs,
                                              }) => {
    const resolveTemplate = React.useCallback(
        (s?: any) => (typeof s === 'string' ? template(s, templateContext) : s),
        [templateContext]
    )

    const common =
        'w-full bg-transparent border placeholder-text-muted text-white rounded px-3 py-2 focus:outline-none'
    const border = errors[f.name]
        ? 'border-red-600'
        : 'border-muted-foreground border-opacity-50'
    const help = errors[f.name] || resolveTemplate(f.help)
    const v = value[f.name] ?? ''

    // DS: siempre llama hook, controla con enabled dentro del hook (ya arreglado)
    const dsField = useFieldDs(f, templateContext)
    const dsValue = dsField?.data

    React.useEffect(() => {
        if (!setLocalDs) return
        // Si este field tiene ds, actualiza el contexto ds local para otras templates
        // (no impacta a menos que definas setLocalDs arriba en FormRenderer)
        const hasDs = (f as any)?.ds && typeof (f as any).ds === 'object'
        if (hasDs && dsValue !== undefined) {
            const dsKey = Object.keys((f as any).ds)[0]
            setLocalDs((prev) => {
                if (JSON.stringify(prev?.[dsKey]) === JSON.stringify(dsValue)) return prev
                return { ...prev, [dsKey]: dsValue }
            })
        }
    }, [dsValue, f, setLocalDs])

    const wrap = (child: React.ReactNode) => (
        <div className="col-span-12">
            <label className="block">
                {resolveTemplate(f.label) && (
                    <div className="text-sm mb-1 text-text-muted">
                        {resolveTemplate(f.label)}
                    </div>
                )}

                <div className="flex items-stretch gap-1">
                    {child}

                    {f.features?.length ? (
                        <FieldFeatures
                            fieldId={f.name}
                            features={f.features}
                            ctx={templateContext}
                            setVal={(id, val) => setVal(id, val)}
                        />
                    ) : null}
                </div>

                {help && (
                    <div
                        className={cx(
                            'text-xs mt-1',
                            errors[f.name] ? 'text-red-400' : 'text-text-muted'
                        )}
                    >
                        {help}
                    </div>
                )}
            </label>
        </div>
    )

    // TEXT / TEXTAREA
    if (f.type === 'text' || f.type === 'textarea') {
        const Comp: any = f.type === 'text' ? 'input' : 'textarea'
        const resolvedValue = resolveTemplate(f.value)
        const val =
            v === '' && resolvedValue != null
                ? resolvedValue
                : v || (dsValue?.amount ?? dsValue?.value ?? '')

        return wrap(
            <Comp
                className={cx(common, border)}
                placeholder={resolveTemplate(f.placeholder)}
                value={val ?? ''}
                readOnly={f.readOnly}
                required={f.required}
                onChange={(e: any) => setVal(f, e.currentTarget.value)}
            />
        )
    }

    // AMOUNT
    if (f.type === 'amount') {
        const val = v ?? (dsValue?.amount ?? dsValue?.value ?? '')
        return wrap(
            <input
                type="number"
                step="any"
                className={cx(common, border)}
                placeholder={resolveTemplate(f.placeholder)}
                value={val ?? ''}
                readOnly={f.readOnly}
                required={f.required}
                onChange={(e) => setVal(f, e.currentTarget.value)}
                min={(f as any).min}
                max={(f as any).max}
            />
        )
    }

    // ADDRESS
    if (f.type === 'address') {
        const resolved = resolveTemplate(f.value)
        const val = v === '' && resolved != null ? resolved : v
        return wrap(
            <input
                className={cx(common, border)}
                placeholder={resolveTemplate(f.placeholder) ?? 'address'}
                value={val ?? ''}
                readOnly={f.readOnly}
                required={f.required}
                onChange={(e) => setVal(f, e.target.value)}
            />
        )
    }

    // SELECT
    if (f.type === 'select') {
        const select = f as SelectField
        const staticOpts = select.options ?? []
        let dynamicOpts: Array<{ label: string; value: string }> = []

        if (select.source) {
            const src = select.source as SourceRef
            // lee desde templateContext (chain/ds/fees/form/account/session)
            const base = templateContext?.[src.uses]
            const picked = getByPath(base, src.selector)
            dynamicOpts = toOptions(picked)
        }

        const opts = (staticOpts.length ? staticOpts : dynamicOpts) ?? []
        const resolved = resolveTemplate(f.value)
        const val = v === '' && resolved != null ? resolved : v

        return wrap(
            <select
                className={cx(common, border)}
                value={val ?? ''}
                onChange={(e) => setVal(f, e.currentTarget.value)}
                disabled={f.readOnly}
                required={f.required}
            >
                {/* opción vacía si el field no es required */}
                {!f.required && <option value="">Select…</option>}
                {opts.map((o) => (
                    <option key={o.value} value={o.value}>
                        {o.label}
                    </option>
                ))}
            </select>
        )
    }

    // fallback
    return (
        <div className="col-span-12 text-sm text-text-muted">
            Unsupported field type: {(f as any)?.type}
        </div>
    )
}
