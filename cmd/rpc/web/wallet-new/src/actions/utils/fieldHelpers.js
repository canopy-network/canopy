export const getByPath = (obj, selector) => {
    if (!selector || !obj)
        return obj;
    return selector.split('.').reduce((acc, k) => acc?.[k], obj);
};
export const toOptions = (raw, f, templateContext, resolveTemplate) => {
    if (!raw)
        return [];
    const map = f?.map ?? {};
    const evalDynamic = (expr, item) => {
        if (!resolveTemplate || typeof expr !== 'string')
            return expr;
        const localCtx = { ...templateContext, row: item, item, ...item };
        try {
            if (/{{.*}}/.test(expr)) {
                return resolveTemplate(expr, localCtx);
            }
            else {
                const fn = new Function(...Object.keys(localCtx), `return (${expr})`);
                return fn(...Object.values(localCtx));
            }
        }
        catch (err) {
            console.warn('Error evaluating map expression:', expr, err);
            return '';
        }
    };
    const makeLabel = (item) => {
        if (map.label)
            return evalDynamic(map.label, item);
        return (item.label ??
            item.name ??
            item.id ??
            item.value ??
            item.address ??
            JSON.stringify(item));
    };
    const makeValue = (item) => {
        if (map.value)
            return evalDynamic(map.value, item);
        return String(item.value ?? item.id ?? item.address ?? item.key ?? item);
    };
    if (Array.isArray(raw)) {
        return raw.map((item) => ({
            label: String(makeLabel(item) ?? ''),
            value: String(makeValue(item) ?? ''),
        }));
    }
    if (typeof raw === 'object') {
        return Object.entries(raw).map(([k, v]) => ({
            label: String(makeLabel(v) ?? k),
            value: String(makeValue(v) ?? k),
        }));
    }
    return [];
};
const SPAN_MAP = {
    1: 'col-span-1',
    2: 'col-span-2',
    3: 'col-span-3',
    4: 'col-span-4',
    5: 'col-span-5',
    6: 'col-span-6',
    7: 'col-span-7',
    8: 'col-span-8',
    9: 'col-span-9',
    10: 'col-span-10',
    11: 'col-span-11',
    12: 'col-span-12',
};
const RSP = (n) => {
    const c = Math.max(1, Math.min(12, Number(n || 12)));
    return SPAN_MAP[c] || 'col-span-12';
};
export const spanClasses = (f, layout) => {
    const conf = f?.span ?? f?.ui?.grid?.colSpan ?? layout?.grid?.defaultSpan;
    const base = typeof conf === 'number' ? { base: conf } : (conf || {});
    const b = RSP(base.base ?? 12);
    const sm = base.sm != null ? `sm:${RSP(base.sm)}` : '';
    const md = base.md != null ? `md:${RSP(base.md)}` : '';
    const lg = base.lg != null ? `lg:${RSP(base.lg)}` : '';
    const xl = base.xl != null ? `xl:${RSP(base.xl)}` : '';
    return [b, sm, md, lg, xl].filter(Boolean).join(' ');
};
