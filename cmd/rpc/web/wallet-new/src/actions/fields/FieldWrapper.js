import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { cx } from '@/ui/cx';
import { spanClasses } from '@/actions/utils/fieldHelpers';
import { FieldFeatures } from '@/actions/components/FieldFeatures';
export const FieldWrapper = ({ field, error, templateContext, resolveTemplate, hasFeatures, setVal, children, }) => {
    const help = error || resolveTemplate(field.help);
    return (_jsx("div", { className: spanClasses(field, templateContext?.layout), children: _jsxs("label", { className: "block", children: [resolveTemplate(field.label) && (_jsx("div", { className: "text-sm mb-1 text-text-muted", children: resolveTemplate(field.label) })), _jsxs("div", { className: "relative", children: [children, hasFeatures && field.features && setVal && (_jsx(FieldFeatures, { fieldId: field.name, features: field.features, ctx: templateContext, setVal: setVal }))] }), help && (_jsx("div", { className: cx('text-xs mt-1 break-words', error ? 'text-red-400' : 'text-text-muted'), children: help }))] }) }));
};
