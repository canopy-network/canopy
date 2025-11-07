import { jsx as _jsx } from "react/jsx-runtime";
import { OptionCard } from '@/actions/OptionCard';
import { FieldWrapper } from './FieldWrapper';
export const OptionCardField = ({ field, value, error, templateContext, onChange, resolveTemplate, }) => {
    const opts = Array.isArray(field.options) ? field.options : [];
    const resolvedDefault = resolveTemplate(field.value);
    const currentValue = (value === '' || value == null) && resolvedDefault != null ? resolvedDefault : value;
    return (_jsx(FieldWrapper, { field: field, error: error, templateContext: templateContext, resolveTemplate: resolveTemplate, children: _jsx("div", { role: "radiogroup", "aria-label": String(resolveTemplate(field.label) ?? field.name), className: "grid grid-cols-12 gap-3 w-full", children: opts.map((o, i) => {
                const label = resolveTemplate(o.label);
                const help = resolveTemplate(o.help);
                const val = String(resolveTemplate(o.value) ?? i);
                const selected = String(currentValue ?? '') === val;
                return (_jsx("div", { className: "col-span-12", children: _jsx(OptionCard, { selected: selected, disabled: field.readOnly, onSelect: () => onChange(val), label: label, help: help }) }, val));
            }) }) }));
};
