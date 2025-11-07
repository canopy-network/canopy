import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import * as Switch from '@radix-ui/react-switch';
export const SwitchField = ({ field, value, onChange, resolveTemplate, }) => {
    const checked = Boolean(value ?? resolveTemplate(field.value) ?? false);
    return (_jsxs("div", { className: "col-span-12 flex flex-col", children: [_jsxs("div", { className: "flex items-center justify-between", children: [_jsx("div", { className: "text-sm mb-1 text-canopy-50", children: resolveTemplate(field.label) }), _jsx(Switch.Root, { id: field.id, checked: checked, disabled: field.readOnly, onCheckedChange: (next) => onChange(next), className: "relative h-5 w-9 rounded-full bg-neutral-700 data-[state=checked]:bg-emerald-500 outline-none shadow-inner transition-colors", "aria-label": String(resolveTemplate(field.label) ?? field.name), children: _jsx(Switch.Thumb, { className: "block h-4 w-4 translate-x-0.5 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-[18px]" }) })] }), field.help && _jsx("span", { className: "text-xs text-text-muted", children: resolveTemplate(field.help) })] }));
};
