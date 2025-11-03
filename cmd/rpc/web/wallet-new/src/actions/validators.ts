// validators.ts
import type { Field, AmountField } from "@/manifest/types";
import {template} from "@/core/templater";

type RuleCode =
    | "required"
    | "min"
    | "max"
    | "length.min"
    | "length.max"
    | "minSelected"
    | "maxSelected"
    | "pattern";

export type ValidationResult =
    | { ok: true; [key: string]: any }
    | { ok: true; errors: { [key: string]: string[] } }
    | { ok: false; code: RuleCode; message: string };

const DEFAULT_MESSAGES: Record<RuleCode, string> = {
    required: "This field is required.",
    min: "Minimum allowed is {{min}}.",
    max: "Maximum allowed is {{max}}.",
    minSelected: "Minimum selected is {{min}}.",
    maxSelected: "Maximum selected is {{max}}.",
    "length.min": "Minimum length is {{length.min}} characters.",
    "length.max": "Maximum length is {{length.max}} characters.",
    pattern: "Invalid format.",
};

const isEmpty = (s: any) =>
    s == null || (typeof s === "string" && s.trim() === "");

const get = (o: any, path?: string) =>
    !path ? o : path.split(".").reduce((a, k) => a?.[k], o);

const resolveMsg = (
    overrides: Record<string, string> | undefined,
    code: RuleCode,
    params: Record<string, any>
) => {
    const raw = overrides?.[code] ?? DEFAULT_MESSAGES[code];
    return template(raw, params);
};

function evalNumeric(v: any, ctx: Record<string, any>): number | undefined {
    if (v == null) return undefined;
    if (typeof v === "number") return Number.isFinite(v) ? v : undefined;
    if (typeof v === "string") {
        const raw = v.includes("{{") ? template(v, ctx) : v;

        const match = String(raw)
            .replace(/\u00A0/g, " ") // NBSP
            .match(/[-+]?(?:\d{1,3}(?:[ ,]\d{3})+|\d+)(?:[.,]\d+)?/);

        if (!match) return undefined;

        let num = match[0].trim();

        if (num.includes(",") && num.includes(".")) {
            const lastComma = num.lastIndexOf(",");
            const lastDot = num.lastIndexOf(".");
            if (lastComma > lastDot) {
                num = num.replace(/\./g, "").replace(",", ".");
            } else {
                num = num.replace(/,/g, "");
            }
        } else if (num.includes(",")) {
            num = num.replace(",", ".");
        } else {
            num = num.replace(/\s+/g, "");
        }

        const n = Number(num);
        return Number.isFinite(n) ? n : undefined;
    }
    return undefined;
}

export async function validateField(
    field: Field,
    value: any,
    ctx: Record<string, any> = {}
): Promise<ValidationResult> {
    if (field.type === "switch") return { ok: true };

    // OPTIONCARD
    if (field.type === "optionCard") {
        if (field.required && (value === undefined || value === null || value === "")) {
            return {
                ok: false,
                code: "required",
                message: resolveMsg(
                    (field as any).validation?.messages,
                    "required",
                    { field, value, ...ctx }
                ),
            };
        }
        return { ok: true };
    }

    // TABLESELECT
    if (field.type === "tableSelect") {
        const arr = Array.isArray(value) ? value : value ? [value] : [];
        if (field.required && arr.length === 0) {
            return {
                ok: false,
                code: "required",
                message: resolveMsg(
                    (field as any).validation?.messages,
                    "required",
                    { field, value, ...ctx }
                ),
            };
        }

        const vconf = (field as any).validation ?? {};
        const min = evalNumeric(vconf.min, ctx);
        const max = evalNumeric(vconf.max, ctx);


        if (typeof min === "number" && arr.length < min) {
            return {
                ok: false,
                code: "minSelected",
                message: resolveMsg(vconf.messages, "minSelected", { min, field, value, ...ctx }),
            };
        }
        if (typeof max === "number" && arr.length > max) {
            return {
                ok: false,
                code: "maxSelected",
                message: resolveMsg(vconf.messages, "maxSelected", { max, field, value, ...ctx }),
            };
        }
        return { ok: true };
    }

    // ——— base shared validation ———
    const templatedValue = typeof value === "string" ? template(value, ctx) : value;
    const formattedValue = isEmpty(templatedValue) ? value : templatedValue;
    const vconf = (field as any).validation ?? {};
    const messages: Record<string, string> | undefined = vconf.messages;
    const asString = value == null ? "" : String(value);

    // REQUIRED
    if (field.required && (formattedValue == null || formattedValue === "")) {
        return {
            ok: false,
            code: "required",
            message: resolveMsg(messages, "required", { field, value, ...ctx }),
        };
    }

    // AMOUNT
    if (field.type === "amount") {
        const f = field as AmountField;

        const n = typeof formattedValue === "string"
            ? Number(formattedValue.trim().replace(/,/g, ""))
            : Number(formattedValue);

        const safeValue = Number.isNaN(n) ? 0 : n;

        const min = evalNumeric(f.min ?? vconf.validation.min, ctx);
        const max = evalNumeric(f.max ?? vconf.max, ctx);

        if (typeof min === "number" && safeValue < min) {
            return {
                ok: false,
                code: "min",
                message: resolveMsg(messages, "min", { min, field, value: safeValue, ...ctx }),
            };
        }

        if (typeof max === "number" && safeValue > max) {
            return {
                ok: false,
                code: "max",
                message: resolveMsg(messages, "max", { max, field, value: safeValue, ...ctx }),
            };
        }
    }

    // LENGTH (ahora soporta min/max templated)
    if (vconf.length && typeof asString === "string") {
        const lmin = evalNumeric(get(vconf, "length.min"), ctx);
        const lmax = evalNumeric(get(vconf, "length.max"), ctx);
        if (typeof lmin === "number" && asString.length < lmin) {
            return {
                ok: false,
                code: "length.min",
                message: resolveMsg(messages, "length.min", {
                    length: { min: lmin, max: lmax },
                    field,
                    value: formattedValue,
                    ...ctx,
                }),
            };
        }
        if (typeof lmax === "number" && asString.length > lmax) {
            return {
                ok: false,
                code: "length.max",
                message: resolveMsg(messages, "length.max", {
                    length: { min: lmin, max: lmax },
                    field,
                    value: formattedValue,
                    ...ctx,
                }),
            };
        }
    }

    // PATTERN
    if (vconf.pattern) {
        const pattern = template(vconf.pattern, ctx);

        const rx =
                 new RegExp(pattern)

        if (!rx.test(asString)) {
            return {
                ok: false,
                code: "pattern",
                message: resolveMsg(messages, "pattern", {
                    field,
                    value: formattedValue,
                    ...ctx,
                }),
            };
        }
    }

    return { ok: true };
}
