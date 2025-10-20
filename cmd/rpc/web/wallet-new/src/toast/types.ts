export type ToastVariant = "success" | "error" | "warning" | "info" | "neutral";

export type ToastAction =
    | { type: "link"; label: string; href: string; newTab?: boolean }
    | { type: "button"; label: string; onClick: () => void };

export type ToastRenderData = {
    id: string;
    title?: React.ReactNode;
    description?: React.ReactNode;
    icon?: React.ReactNode;
    actions?: ToastAction[];
    variant?: ToastVariant;
    durationMs?: number;     // auto-dismiss
    sticky?: boolean;        // no auto-dismiss
};

export type ToastTemplateInput =
    | string                    // "Hello {{user.name}}"
    | ((ctx: any) => string)    // (ctx) => `Hello ${ctx.user.name}`
    | React.ReactNode;          // <span>…</span>

export type ToastTemplateOptions = Omit<ToastRenderData, "title" | "description"> & {
    title?: ToastTemplateInput;
    description?: ToastTemplateInput;
    ctx?: any; // Action Runner ctx
};

export type ToastFromResultOptions<R = any> = {
    result: R;
    ctx?: any;
    map?: (r: R, ctx: any) => ToastTemplateOptions | null | undefined;
    fallback?: ToastTemplateOptions;
};

export type ToastApi = {
    toast: (t: ToastTemplateOptions) => string;
    success: (t: ToastTemplateOptions) => string;
    error: (t: ToastTemplateOptions) => string;
    info: (t: ToastTemplateOptions) => string;
    warning: (t: ToastTemplateOptions) => string;
    neutral: (t: ToastTemplateOptions) => string;
    fromResult: <R = any>(o: ToastFromResultOptions<R>) => string | null;
    dismiss: (id: string) => void;
    clear: () => void;
};