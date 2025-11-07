import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { template, templateAny } from '@/core/templater';
import { toOptions } from '@/actions/utils/fieldHelpers';
import { FieldWrapper } from './FieldWrapper';
export const SelectField = ({ field, value, error, templateContext, dsValue, onChange, resolveTemplate, }) => {
    const select = field;
    const staticOptions = Array.isArray(select.options) ? select.options : [];
    const rawOptions = dsValue && Object.keys(dsValue).length ? dsValue : staticOptions;
    let mappedFromExpr = null;
    if (typeof select.map === 'string') {
        try {
            const out = templateAny(select.map, templateContext);
            if (Array.isArray(out)) {
                mappedFromExpr = out;
            }
            else if (typeof out === 'string') {
                try {
                    const maybe = JSON.parse(out);
                    if (Array.isArray(maybe))
                        mappedFromExpr = maybe;
                }
                catch { }
            }
        }
        catch (err) {
            console.warn('select.map expression error:', err);
        }
    }
    const builtOptions = mappedFromExpr
        ? mappedFromExpr.map((o) => ({
            label: String(o?.label ?? ''),
            value: String(o?.value ?? ''),
        }))
        : toOptions(rawOptions, field, templateContext, template);
    const resolvedDefault = resolveTemplate(field.value);
    const currentValue = value === '' && resolvedDefault != null ? resolvedDefault : value;
    return (_jsx(FieldWrapper, { field: field, error: error, templateContext: templateContext, resolveTemplate: resolveTemplate, children: _jsxs(Select, { value: currentValue ?? '', onValueChange: (val) => onChange(val), disabled: field.readOnly, required: field.required, children: [_jsx(SelectTrigger, { className: "w-full bg-bg-tertiary border-bg-accent text-white h-11 rounded-lg", children: _jsx(SelectValue, { placeholder: field.placeholder }) }), _jsx(SelectContent, { className: "bg-bg-tertiary border-bg-accent", children: builtOptions.map((o) => (_jsx(SelectItem, { value: o.value, className: "text-white", children: o.label }, o.value))) })] }) }));
};
