// toast/utils.ts
import {ToastTemplateInput} from "@/toast/types";

export const getAt = (o: any, p?: string) =>
    !p ? o : p.split(".").reduce((a, k) => (a ? a[k] : undefined), o);

const interpolate = (tpl: string, ctx: any) =>
    tpl.replace(/\{\{\s*([^}]+)\s*\}\}/g, (_, path) => {
        const v = getAt(ctx, path.trim());
        return v == null ? "" : String(v);
    });

export const renderTemplate = (input: ToastTemplateInput, ctx?: any): React.ReactNode => {
    if (typeof input === "function") return (input as any)(ctx);
    if (typeof input === "string") return ctx ? interpolate(input, ctx) : input;
    return input; // ReactNode passthrough
};
