import { template } from "@/core/templater";
import type { Action, Field } from "@/manifest/types";

/** Get fields from manifest */
export const getFieldsFromAction = (action?: Action): Field[] =>
  Array.isArray(action?.form?.fields) ? (action!.form!.fields as Field[]) : [];

/** Hints for field names */
const NUMERIC_HINTS = new Set([
  "amount",
  "receiveAmount",
  "fee",
  "gas",
  "gasPrice",
]);
const BOOL_HINTS = new Set(["delegate", "earlyWithdrawal", "submit"]);

/** Normalize form according to Fields + hints:
 * - number: convert "1,234.56" to 1234.56
 * - boolean (by name): 'true'/'false' to boolean
 */
export function normalizeFormForAction(
  action: Action | undefined,
  form: Record<string, any>,
) {
  const out: Record<string, any> = { ...form };
  const fields = (action?.form?.fields ?? []) as Field[];

  const asNum = (v: any) => {
    if (v === "" || v == null) return v;
    const s = String(v).replace(/,/g, "");
    const n = Number(s);
    return Number.isNaN(n) ? v : n;
  };
  const asBool = (v: any) =>
    v === true || v === "true" || v === 1 || v === "1" || v === "on";

  for (const f of fields) {
    const n = f?.name;
    if (n == null || !(n in out)) continue;

    // por tipo
    if (f.type === "amount" || NUMERIC_HINTS.has(n)) out[n] = asNum(out[n]);
    // por “hint” de nombre (p.ej. select true/false)
    if (BOOL_HINTS.has(n)) out[n] = asBool(out[n]);
  }
  return out;
}

export type BuildPayloadCtx = {
  form: Record<string, any>;
  chain?: any;
  session?: { password?: string };
  account?: any;
  fees?: { raw?: any; amount?: number | string; denom?: string };
  extra?: Record<string, any>;
};

export function buildPayloadFromAction(action: Action, ctx: any) {
  const result: Record<string, any> = {};

  for (const [key, val] of Object.entries(action.payload || {})) {
    // caso 1: simple string => resolver plantilla
    if (typeof val === "string") {
      result[key] = template(val, ctx);
      continue;
    }

    if (typeof val === "object" && val?.value !== undefined) {
      let resolved: any = template(val?.value, ctx);

      if (val?.coerce) {
        switch (val.coerce) {
          case "number":
            //@ts-ignore
            resolved = Number(resolved);
            break;
          case "string":
            resolved = String(resolved);
            break;
          case "boolean":
            const resolvedStr = String(resolved).toLowerCase();
            resolved = resolvedStr === "true" || resolvedStr === "1";
            break;
        }
      }

      result[key] = resolved;
      continue;
    }
    // fallback
    result[key] = val;
  }

  return result;
}

export function buildConfirmSummary(
  action: Action | undefined,
  data: {
    form: Record<string, any>;
    chain?: any;
    fees?: { effective?: number | string };
  },
) {
  const items = action?.confirm?.summary ?? [];
  return items.map((s) => ({ label: s.label, value: template(s.value, data) }));
}

export function selectQuickActions(
  actions: Action[] | undefined,
  chain: any,
  max?: number,
) {
  const limit = max ?? 8;
  const hasFeature = (a: Action) => !a.requiresFeature;
  const rank = (a: Action) =>
    typeof a.priority === "number"
      ? a.priority
      : typeof a.order === "number"
        ? a.order
        : 0;

  return (actions ?? [])
    .filter(
      (a) => !a.hidden && Array.isArray(a.tags) && a.tags.includes("quick"),
    )
    .filter(hasFeature)
    .sort((a, b) => rank(b) - rank(a))
    .slice(0, limit);
}
