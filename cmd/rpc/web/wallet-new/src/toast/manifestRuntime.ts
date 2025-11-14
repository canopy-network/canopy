// toast/manifestRuntime.ts
import { template } from "@/core/templater";
import { ToastTemplateOptions } from "@/toast/types";

const maybeTpl = (v: any, data: any) =>
    typeof v === "string" ? template(v, data) : v;

export type NotificationNode = Partial<ToastTemplateOptions> & {
    actions?: Array<
        | { type: "link"; label: string; href: string; newTab?: boolean }
        | { type: "button"; label: string; onClickId?: string } // opcional: callback id
    >;
};

export function resolveToastFromManifest(
    action: any,
    key: "onInit" | "onBeforeSubmit" | "onSuccess" | "onError" | "onFinally",
    ctx: any,
    result?: any
): ToastTemplateOptions | null {
    const node: NotificationNode | undefined = action?.notifications?.[key];
    if (!node) return null;

    const data = { ...ctx, result };
    const rendered: ToastTemplateOptions = {
        variant: node.variant,
        title: maybeTpl(node.title, data),
        description: maybeTpl(node.description, data),
        icon: node.icon,
        sticky: node.sticky,
        durationMs: node.durationMs,
        actions: node.actions?.map((a) =>
            a.type === "link"
                ? { ...a, href: maybeTpl(a.href, data), label: maybeTpl(a.label, data) }
                : { ...a, label: maybeTpl(a.label, data) }
        ),
        ctx: data
    };
    return rendered;
}

export function resolveRedirectFromManifest(
    action: any,
    ctx: any,
    result: any
): { to: string; delayMs?: number; replace?: boolean } | null {
    const r = action?.redirect;
    if (!r) return null;
    const should =
        r.when === "always" ||
        (r.when === "success" && (result?.ok ?? true)) ||
        (r.when === "error" && !(result?.ok ?? true));
    if (!should) return null;

    const to = template(r.to, { ...ctx, result });
    return { to, delayMs: r.delayMs ?? 0, replace: !!r.replace };
}
