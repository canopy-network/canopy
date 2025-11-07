import { jsx as _jsx } from "react/jsx-runtime";
import { template } from '@/core/templater';
import TableSelect from '@/actions/TableSelect';
export const TableSelectField = ({ field, value, errors, templateContext, onChange, resolveTemplate, }) => {
    return (_jsx(TableSelect, { field: field, currentValue: value, onChange: (next) => onChange(next), errors: errors, resolveTemplate: resolveTemplate, template: template, templateContext: templateContext }));
};
