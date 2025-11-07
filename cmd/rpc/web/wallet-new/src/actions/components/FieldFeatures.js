import { jsx as _jsx } from "react/jsx-runtime";
import { template } from '@/core/templater';
export const FieldFeatures = ({ features, ctx, setVal, fieldId }) => {
    if (!features?.length)
        return null;
    const resolve = (s) => (typeof s === 'string' ? template(s, ctx) : s);
    const labelFor = (op) => {
        if (op.op === 'copy')
            return 'Copy';
        if (op.op === 'paste')
            return 'Paste';
        if (op.op === 'set')
            return 'Max';
        return op.op;
    };
    const handle = async (op) => {
        const opAny = op;
        switch (opAny.op) {
            case 'copy': {
                const txt = String(resolve(opAny.from) ?? '');
                await navigator.clipboard.writeText(txt);
                return;
            }
            case 'paste': {
                const txt = await navigator.clipboard.readText();
                setVal(fieldId, txt);
                return;
            }
            case 'set': {
                const v = resolve(opAny.value);
                setVal(opAny.field ?? fieldId, v);
                return;
            }
        }
    };
    return (_jsx("div", { className: "absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1", children: features.map((op) => (_jsx("button", { type: "button", onClick: () => handle(op), className: "text-xs px-2 py-1 rounded font-semibold border border-primary text-primary hover:bg-primary hover:text-secondary transition-colors", children: labelFor(op) }, op.id))) }));
};
