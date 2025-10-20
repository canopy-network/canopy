import React from 'react'
import type { Field, FieldOp, SelectField, SourceRef } from '@/manifest/types'
import { useFieldDs } from '@/actions/useFieldsDs'
import { template } from '@/core/templater'
import { cx } from '@/ui/cx'
import * as Switch from '@radix-ui/react-switch';
import {OptionCard, OptionCardOpt} from "@/actions/OptionCard";

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

    // DS: Always call hook, control with enabled inside the hook (already regulated)
    const dsField = useFieldDs(f, templateContext)
    const dsValue = dsField?.data

    React.useEffect(() => {
        if (!setLocalDs) return
        // If this field has a data source, update the local ds context for other templates
        // (does not affect anything unless setLocalDs is defined above in FormRenderer)
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

    // SWITCH
    if (f.type === 'switch') {
        const checked = Boolean(v ?? resolveTemplate(f.value) ?? false)
        return (
            <div className="col-span-12 flex flex-col ">
                <div className={"flex items-center justify-between"}>
                    <div className="text-sm mb-1 text-canopy-50 ">{resolveTemplate(f.label)}</div>
                    <Switch.Root
                        id={f.id}
                        checked={checked}
                        disabled={f.readOnly}
                        onCheckedChange={(next) => setVal(f, next)}
                        className="relative h-5 w-9 rounded-full bg-neutral-700 data-[state=checked]:bg-emerald-500 outline-none shadow-inner transition-colors"
                        aria-label={String(resolveTemplate(f.label) ?? f.name)}
                    >
                        <Switch.Thumb className="block h-4 w-4 translate-x-0.5 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-[18px]" />
                    </Switch.Root>
                </div>


                {f.help && <span className="text-xs text-text-muted">{resolveTemplate(f.help)}</span>}
            </div>
        )
    }

    // OPTION CARD
    if (f.type === 'optionCard') {
        const opts: OptionCardOpt[] = Array.isArray((f as any).options) ? (f as any).options : [];
        const resolvedDefault = resolveTemplate(f.value);
        const current = (v === '' || v == null) && resolvedDefault != null ? resolvedDefault : v;

        return wrap(
            <div role="radiogroup" aria-label={String(resolveTemplate(f.label) ?? f.name)} className="grid grid-cols-12 gap-3 w-full">
                {opts.map((o, i) => {
                    const label = resolveTemplate(o.label);
                    const help = resolveTemplate(o.help);
                    const val = String(resolveTemplate(o.value) ?? i);
                    const selected = String(current ?? '') === val;

                    return (
                        <div key={val} className="col-span-12 ">
                            <OptionCard
                                selected={selected}
                                disabled={f.readOnly}
                                onSelect={() => setVal(f, val)}
                                label={label}
                                help={help}
                            />
                        </div>
                    );
                })}
            </div>
        );
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
