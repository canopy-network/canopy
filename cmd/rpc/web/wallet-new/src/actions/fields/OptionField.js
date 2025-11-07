import { jsx as _jsx } from "react/jsx-runtime";
import { cx } from '@/ui/cx';
import { Option } from '@/actions/Option';
import { FieldWrapper } from './FieldWrapper';
export const OptionField = ({ field, value, error, templateContext, onChange, resolveTemplate, }) => {
    const optionField = field;
    const isInLine = optionField.inLine;
    const opts = Array.isArray(field.options) ? field.options : [];
    const resolvedDefault = resolveTemplate(field.value);
    const currentValue = (value === '' || value == null) && resolvedDefault != null ? resolvedDefault : value;
    return (_jsx(FieldWrapper, { field: field, error: error, templateContext: templateContext, resolveTemplate: resolveTemplate, children: _jsx("div", { role: "radiogroup", "aria-label": String(resolveTemplate(field.label) ?? field.name), className: cx('w-full gap-3', isInLine ? 'flex flex-wrap justify-between items-center' : 'grid grid-cols-12'), children: opts.map((o, i) => {
                const label = resolveTemplate(o.label);
                const help = resolveTemplate(o.help);
                const val = String(resolveTemplate(o.value) ?? i);
                const selected = String(currentValue ?? '') === val;
                return (_jsx("div", { className: cx(isInLine ? 'flex-1 min-w-[120px] max-w-full' : 'col-span-12'), children: _jsx(Option, { selected: selected, disabled: field.readOnly, onSelect: () => onChange(val), label: label, help: help }) }, val));
            }) }) }));
};
