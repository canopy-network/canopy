import React from 'react'
import {AdvancedSelectField, Field, FieldOp, OptionField, SelectField, SourceRef} from '@/manifest/types'
import {collectDepsFromObject, template, templateAny} from '@/core/templater'
import { cx } from '@/ui/cx'
import * as Switch from '@radix-ui/react-switch';
import { OptionCard, OptionCardOpt } from "@/actions/OptionCard";
import TableSelect from "@/actions/TableSelect";
import { templateBool } from '@/core/templater';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import ComboSelectRadix from "@/actions/ComboSelect";
import {OptionItem, Option} from "@/actions/Option";
import {useFieldsDs} from "@/actions/useFieldsDs";

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

const toOptions = (
    raw: any,
    f?: any,
    templateContext?: Record<string, any>,
    resolveTemplate?: (s: any, ctx?: any) => any
): Array<{ label: string; value: string }> => {
    if (!raw) return []
    const map = f?.map ?? {}

    // Helper: evaluates complex expressions in the map using the same templater
    const evalDynamic = (expr: string, item?: any) => {
        if (!resolveTemplate || typeof expr !== 'string') return expr
        // Combine global context, current item, and safe global access (window-safe)
        const localCtx = { ...templateContext, row: item, item, ...item }
        try {
            // Try to evaluate as a full JS expression (allows map, filter, etc.)
            // Use Function instead of eval for safety
            if (/{{.*}}/.test(expr)) {
                return resolveTemplate(expr, localCtx)
            } else {
                // Allows expressions without braces if someone passes "Object.keys(ds.keystore?.addressMap)"
                // eslint-disable-next-line no-new-func
                const fn = new Function(...Object.keys(localCtx), `return (${expr})`)
                return fn(...Object.values(localCtx))
            }
        } catch (err) {
            console.warn('Error evaluating map expression:', expr, err)
            return ''
        }
    }

    const makeLabel = (item: any) => {
        if (map.label) return evalDynamic(map.label, item)
        return (
            item.label ??
            item.name ??
            item.id ??
            item.value ??
            item.address ??
            JSON.stringify(item)
        )
    }

    const makeValue = (item: any) => {
        if (map.value) return evalDynamic(map.value, item)
        return String(item.value ?? item.id ?? item.address ?? item.key ?? item)
    }

    if (Array.isArray(raw)) {
        return raw.map((item) => ({
            label: String(makeLabel(item) ?? ''),
            value: String(makeValue(item) ?? ''),
        }))
    }

    if (typeof raw === 'object') {
        // If it's a map type { id: {...}, id2: {...} }
        return Object.entries(raw).map(([k, v]) => ({
            label: String(makeLabel(v) ?? k),
            value: String(makeValue(v) ?? k),
        }))
    }

    return []
}

const SPAN_MAP = {
    1:'col-span-1',2:'col-span-2',3:'col-span-3',4:'col-span-4',5:'col-span-5',6:'col-span-6',
    7:'col-span-7',8:'col-span-8',9:'col-span-9',10:'col-span-10',11:'col-span-11',12:'col-span-12',
}
const RSP = (n?: number) => {
    const c = Math.max(1, Math.min(12, Number(n || 12)))
    return SPAN_MAP[c as keyof typeof SPAN_MAP] || 'col-span-12'
}

const spanClasses = (f: any, layout?: any) => {
    const conf = f?.span ?? f?.ui?.grid?.colSpan ?? layout?.grid?.defaultSpan
    const base = typeof conf === 'number' ? { base: conf } : (conf || {})
    const b  = RSP(base.base ?? 12)
    const sm = base.sm != null ? `sm:${RSP(base.sm)}` : ''
    const md = base.md != null ? `md:${RSP(base.md)}` : ''
    const lg = base.lg != null ? `lg:${RSP(base.lg)}` : ''
    const xl = base.xl != null ? `xl:${RSP(base.xl)}` : ''
    return [b, sm, md, lg, xl].filter(Boolean).join(' ')
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

    const manualWatch: string[] = React.useMemo(() => {
        const dsObj: any = (f as any)?.ds
        const watch = dsObj?.__options?.watch
        return Array.isArray(watch) ? watch : []
    }, [f])

    // 2) auto watch desde templates en el DS
    const autoWatchAllRoots: string[] = React.useMemo(() => {
        const dsObj: any = (f as any)?.ds
        return collectDepsFromObject(dsObj) // e.g. ["form.operator","form.output","chain.denom"]
    }, [f])

    // 3) limita a form.* y normaliza
    const autoWatchFormOnly: string[] = React.useMemo(() => {
        return autoWatchAllRoots
            .filter(p => p.startsWith("form."))
            .map(p => p.replace(/^form\.\??/, "form.")) // por si vino "form?.x"
    }, [autoWatchAllRoots])


    const watchPaths: string[] = React.useMemo(() => {
        const merged = new Set<string>([...manualWatch, ...autoWatchFormOnly])
        return Array.from(merged)
    }, [manualWatch, autoWatchFormOnly])

    // 5) snapshot + token
    const watchSnapshot = React.useMemo(() => {
        const snap: Record<string, any> = {}
        for (const p of watchPaths) snap[p] = getByPath(templateContext, p)
        return snap
    }, [watchPaths, templateContext])

    const watchToken = React.useMemo(() => {
        try { return JSON.stringify(watchSnapshot) } catch { return "" }
    }, [watchSnapshot])

    const { dsValue, dsLoading, dsError } = useFieldsDs((f as any).ds, templateContext, {
        keyScope: `${f.id}:${watchToken}`,
    })

    const isVisible = (f as any).showIf == null
        ? true
        : templateBool((f as any).showIf, templateContext);

    if (!isVisible) return null;

    const common =
        'w-full bg-transparent border placeholder-text-muted text-white rounded px-3 py-2 focus:outline-none'
    const border = errors[f.name]
        ? 'border-red-600'
        : 'border-muted-foreground border-opacity-50'
    const help = errors[f.name] || resolveTemplate(f.help)
    const v = value[f.name] ?? ''



    const ctxWithDs = React.useMemo(() => {
        return {
            ...templateContext,
            ds: { ...(templateContext?.ds || {}), ...(dsValue || {}) },
        };
    }, [templateContext, dsValue]);

    const stable = (obj: any) => {
        try { return JSON.stringify(obj, Object.keys(obj || {}).sort()); }
        catch { return JSON.stringify(obj || {}); }
    };


    React.useEffect(() => {
        if (!setLocalDs) return;

        const fieldDs = (f as any)?.ds;
        const declaredKeys = fieldDs && typeof fieldDs === "object"
            ? Object.keys(fieldDs)
            : [];

        if (declaredKeys.length === 0 || dsValue == null) return;

        setLocalDs(prev => {
            const next = { ...(prev || {}) };
            let changed = false;

            for (const key of declaredKeys) {
                const incoming = (dsValue as any)[key];
                // Si aún no hay data para esa key, no tocar
                if (incoming === undefined) continue;

                const prevForKey = (prev as any)?.[key];
                const same = stable(prevForKey) === stable(incoming);
                if (!same) {
                    next[key] = incoming;
                    changed = true;
                }
            }

            return changed ? next : prev;
        });
    }, [
        setLocalDs,
        f?.ds ? JSON.stringify(Object.keys((f as any).ds).sort()) : "no-ds",
        stable(dsValue),
    ]);

    const wrap = (child: React.ReactNode) => (
        <div className={spanClasses(f, templateContext?.layout)}>
            <label className="block">
                {resolveTemplate(f.label) && (
                    <div className="text-sm mb-1 text-text-muted">{resolveTemplate(f.label)}</div>
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
                    <div className={cx('text-xs mt-1 break-words'
                        , errors[f.name] ? 'text-red-400' : 'text-text-muted')}>
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

    if (f.type === "advancedSelect") {
        const select = f as AdvancedSelectField

        const dsData = dsValue
        const staticOptions = Array.isArray(select.options) ? select.options : []

        // Default main source: ds > static
        const rawOptions = dsData && Object.keys(dsData).length ? dsData : staticOptions

        // If map is a string (FULL EXPRESSION), evaluate it with templateAny (returns real type)
        let mappedFromExpr: any[] | null = null;
        if (typeof (select as any).map === 'string') {
            try {
                const out = templateAny((select as any).map, ctxWithDs); // <— ctx con ds
                if (Array.isArray(out)) {
                    mappedFromExpr = out;
                } else if (typeof out === 'string') {
                    try {
                        const maybe = JSON.parse(out);
                        if (Array.isArray(maybe)) mappedFromExpr = maybe;
                    } catch { /* ignore */ }
                }
            } catch (err) {
                console.warn('select.map expression error:', err);
            }
        }

        // Build options:
        // - if map string returned an array => use it as-is
        // - otherwise, use toOptions (respects map {label,value} and/or raw structure)
        const builtOptions = mappedFromExpr
            ? mappedFromExpr.map((o) => ({
                label: String(o?.label ?? ''),
                value: String(o?.value ?? ''),
            }))
            : toOptions(rawOptions, f, templateContext, template)

        // Current value (resolves templated default)
        const resolvedDefault = resolveTemplate(f.value)
        const val = v === '' && resolvedDefault != null ? resolvedDefault : v


        return wrap(
            <ComboSelectRadix
                id={f.id}
                value={val}
                options={builtOptions}
                onChange={(val) => {
                    setVal( f, val);
                }}
                placeholder={f.placeholder}
                allowCreate={f.allowCreate}
                allowFreeInput={f.allowFreeInput}
                disabled={f.disabled}
            />
        );
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

    if (f.type === "option") {
        const field = f as OptionField;
        const isInLine = field.inLine;
        const opts: OptionItem[] = Array.isArray((f as any).options)
            ? (f as any).options
            : [];
        const resolvedDefault = resolveTemplate(f.value);
        const current =
            (v === "" || v == null) && resolvedDefault != null
                ? resolvedDefault
                : v;

        return wrap(
            <div
                role="radiogroup"
                aria-label={String(resolveTemplate(f.label) ?? f.name)}
                className={cx(
                    "w-full gap-3",
                    isInLine
                        ? // Distribución fluida horizontal
                        "flex flex-wrap justify-between items-center"
                        : "grid grid-cols-12"
                )}
            >
                {opts.map((o, i) => {
                    const label = resolveTemplate(o.label);
                    const help = resolveTemplate(o.help);
                    const val = String(resolveTemplate(o.value) ?? i);
                    const selected = String(current ?? "") === val;

                    return (
                        <div
                            key={val}
                            className={cx(
                                isInLine
                                    ?
                                    "flex-1 min-w-[120px] max-w-full"
                                    : "col-span-12"
                            )}
                        >
                            <Option
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
    if (f.type === 'tableSelect') {
        return (
            <TableSelect
                field={f}
                currentValue={v}
                onChange={(next) => setVal(f, next)}
                errors={errors}
                resolveTemplate={resolveTemplate}
                template={template}
                templateContext={templateContext}
            />
        )
    }

    // SELECT
    if (f.type === 'select') {
        const select = f as SelectField

        const dsData = dsValue
        const staticOptions = Array.isArray(select.options) ? select.options : []

        // Default main source: ds > static
        const rawOptions = dsData && Object.keys(dsData).length ? dsData : staticOptions

        // If map is a string (FULL EXPRESSION), evaluate it with templateAny (returns real type)
        let mappedFromExpr: any[] | null = null
        if (typeof (select as any).map === 'string') {
            try {
                // Use templateAny so it returns a real ARRAY if the expression is one
                const out = templateAny((select as any).map, templateContext)

                if (Array.isArray(out)) {
                    mappedFromExpr = out
                } else if (typeof out === 'string') {
                    // If it came as string (e.g. JSON), try to parse it
                    try {
                        const maybe = JSON.parse(out)
                        if (Array.isArray(maybe)) mappedFromExpr = maybe
                    } catch { /* ignore */ }
                }
            } catch (err) {
                console.warn('select.map expression error:', err)
            }
        }

        // Build options:
        // - if map string returned an array => use it as-is
        // - otherwise, use toOptions (respects map {label,value} and/or raw structure)
        const builtOptions = mappedFromExpr
            ? mappedFromExpr.map((o) => ({
                label: String(o?.label ?? ''),
                value: String(o?.value ?? ''),
            }))
            : toOptions(rawOptions, f, templateContext, template)

        // Current value (resolves templated default)
        const resolvedDefault = resolveTemplate(f.value)
        const val = v === '' && resolvedDefault != null ? resolvedDefault : v

        // Render
        return wrap(
            <Select value={val ?? ''}
                    onValueChange={(val) => setVal(f, val)}
                    disabled={f.readOnly}
                    required={f.required}
            >
                <SelectTrigger className="w-full bg-bg-tertiary border-bg-accent text-white h-11 rounded-lg">
                    <SelectValue placeholder={f.placeholder}/>
                </SelectTrigger>
                <SelectContent className="bg-bg-tertiary border-bg-accent">
                    {builtOptions.map((o) => (
                        <SelectItem key={o.value} value={o.value} className="text-white">
                            {o.label}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        )
    }

    if (f.type === 'dynamicHtml') {
        const resolvedHtml = resolveTemplate(f.html);

        // Evaluar el objeto `ds` (data source interno)
        const dsObj = (f as any).ds;
        let resolvedDs: Record<string, any> | undefined;
        if (dsObj && typeof dsObj === 'object') {
            try {
                // Evaluamos cada expresión templated dentro del DS
                const deepResolve = (obj: any): any => {
                    if (obj == null) return obj;
                    if (typeof obj === 'string') return templateAny(obj, templateContext);
                    if (typeof obj === 'object') {
                        const result: Record<string, any> = {};
                        for (const [k, v] of Object.entries(obj)) {
                            result[k] = deepResolve(v);
                        }
                        return result;
                    }
                    return obj;
                };
                resolvedDs = deepResolve(dsObj);
            } catch (err) {
                console.warn('Error resolving dynamicHtml.ds:', err);
            }
        }

        // Si hay setLocalDs, propagar los valores resueltos al contexto local
        React.useEffect(() => {
            if (!setLocalDs || !resolvedDs) return;
            setLocalDs(prev => ({ ...prev, ...resolvedDs }));
        }, [JSON.stringify(resolvedDs)]);

        // Render dinámico del HTML (seguro)
        return wrap(
            <div
                className="text-sm text-text-muted w-full"
                dangerouslySetInnerHTML={{ __html: resolvedHtml ?? '' }}
            />
        );

    }

    // fallback
    return (
        <div className="col-span-12 text-sm text-text-muted">
            Unsupported field type: {(f as any)?.type}
        </div>
    )
}
