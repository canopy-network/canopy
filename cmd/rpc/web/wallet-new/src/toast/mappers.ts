// toast/mappers.ts
import { ToastTemplateOptions } from "./types";

export const genericResultMap = <R extends { ok?: boolean; status?: number; error?: any; data?: any }>(
    r: R,
    ctx: any
): ToastTemplateOptions => {
    if (r.ok) {
        return {
            variant: "success",
            title: "Done",
            description: typeof r.data?.message === "string"
                ? r.data.message
                : "The operation completed successfully.",
            ctx,
        };
    }
    // error pathway
    const code = r.status ?? r.error?.code ?? "ERR";
    const msg =
        r.error?.message ??
        r.error?.reason ??
        r.data?.message ??
        "We couldn’t complete your request.";
    return {
        variant: "error",
        title: `Something went wrong (${code})`,
        description: msg,
        ctx,
        sticky: true,
    };
};
