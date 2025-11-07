import { jsx as _jsx } from "react/jsx-runtime";
import { FieldWrapper } from './FieldWrapper';
export const DynamicHtmlField = ({ field, error, templateContext, resolveTemplate, }) => {
    const resolvedHtml = resolveTemplate(field.html);
    return (_jsx(FieldWrapper, { field: field, error: error, templateContext: templateContext, resolveTemplate: resolveTemplate, children: _jsx("div", { className: "text-sm text-text-muted w-full", dangerouslySetInnerHTML: { __html: resolvedHtml ?? '' } }) }));
};
