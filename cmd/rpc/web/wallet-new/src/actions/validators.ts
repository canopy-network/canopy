// validators.ts
import type { Field, AmountField } from "@/manifest/types";
type RuleCode =
    | "required"
    | "min"
    | "max"
    | "length.min"
    | "length.max"
    | "pattern";

export type ValidationResult =
    | { ok: true, [key: string]: any }
    | { ok: true, errors: { [key: string]: string[]}}
    | { ok: false; code: RuleCode; message: string };

const DEFAULT_MESSAGES: Record<RuleCode, string> = {
    required: "This field is required.",
    min: "Minimum allowed is {{min}}.",
    max: "Maximum allowed is {{max}}.",
    "length.min": "Minimum length is {{length.min}} characters.",
    "length.max": "Maximum length is {{length.max}} characters.",
    pattern: "Invalid format.",
};

const isEmpty = (s: string) => s == null || s.trim() === "";

// tiny template helper: replaces {{path}} using ctx
const tmpl = (s: string, ctx: Record<string, any>) =>
    s.replace(/{{\s*([^}]+)\s*}}/g, (_, key) =>
        String(key.split(".").reduce((a: any, k: string) => a?.[k], ctx) ?? "")
    );

// Safe path getter
const get = (o: any, path?: string) =>
    !path ? o : path.split(".").reduce((a, k) => a?.[k], o);

// Utility: look up field-specific override or default
const resolveMsg = (
    overrides: Record<string, string> | undefined,
    code: RuleCode,
    params: Record<string, any>
) => {
    const raw = overrides?.[code] ?? DEFAULT_MESSAGES[code];
    return tmpl(raw, params);
};

export async function validateField(
    field: Field,
    value: any,
    ctx: Record<string, any> = {}
): Promise<ValidationResult> {
    // Optional field-level validation config
    // We don’t change your types; just read if present.

    const templatedValue = tmpl(value, ctx);
    const formattedValue = isEmpty(templatedValue) ? value : templatedValue ;
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

    if (field.type === "amount") {
        const f = field as AmountField;

        const n = typeof formattedValue === "string" ? Number(formattedValue.trim()) : Number(formattedValue);

        const safeValue = Number.isNaN(n) ? 0 : n;

        const min = typeof f.min === "number" ? f.min : 0;
        const max = typeof f.max === "number" ? f.max : undefined;

        if (safeValue < min) {
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

    // GENERIC LENGTH (if provided)
    // Supports: validation.length = { min?: number, max?: number }
    if (vconf.length && typeof asString === "string") {
        const lmin = get(vconf, "length.min");
        const lmax = get(vconf, "length.max");
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

    // GENERIC PATTERN (if provided)
    // Supports: validation.pattern = "^[a-z0-9]+$" or new RegExp(...)
    if (vconf.pattern) {
        const rx =
            typeof vconf.pattern === "string" ? new RegExp(vconf.pattern) : vconf.pattern;
        if (!rx.test(asString)) {
            return {
                ok: false,
                code: "pattern",
                message: resolveMsg(messages, "pattern", { field, value: formattedValue, ...ctx }),
            };
        }
    }

    return { ok: true };
}
