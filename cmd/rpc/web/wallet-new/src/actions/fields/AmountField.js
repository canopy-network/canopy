import { jsx as _jsx } from "react/jsx-runtime";
import { cx } from '@/ui/cx';
import { FieldWrapper } from './FieldWrapper';
export const AmountField = ({ field, value, error, templateContext, dsValue, onChange, resolveTemplate, setVal, }) => {
    const currentValue = value ?? (dsValue?.amount ?? dsValue?.value ?? '');
    const hasFeatures = !!(field.features?.length);
    const common = 'w-full bg-transparent border placeholder-text-muted text-white rounded px-3 py-2 focus:outline-none';
    const paddingRight = hasFeatures ? 'pr-20' : '';
    const border = error ? 'border-red-600' : 'border-muted-foreground border-opacity-50';
    return (_jsx(FieldWrapper, { field: field, error: error, templateContext: templateContext, resolveTemplate: resolveTemplate, hasFeatures: hasFeatures, setVal: setVal, children: _jsx("input", { type: "number", step: "any", className: cx(common, border, paddingRight), placeholder: resolveTemplate(field.placeholder), value: currentValue ?? '', readOnly: field.readOnly, required: field.required, onChange: (e) => onChange(e.currentTarget.value), min: field.min, max: field.max }) }));
};
