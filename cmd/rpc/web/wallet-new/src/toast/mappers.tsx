// toast/mappers.tsx
import React from "react";
import { ToastTemplateOptions } from "./types";
import { Pause, Play } from "lucide-react";

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
        "We couldn't complete your request.";
    return {
        variant: "error",
        title: `Something went wrong (${code})`,
        description: msg,
        ctx,
        sticky: true,
    };
};

// Mapper for pause validator action
export const pauseValidatorMap = <R extends { ok?: boolean; status?: number; error?: any; data?: any }>(
    r: R,
    ctx: any
): ToastTemplateOptions => {
    if (r.ok) {
        const validatorAddr = ctx?.form?.validatorAddress || "Validator";
        const shortAddr = validatorAddr.length > 12
            ? `${validatorAddr.slice(0, 6)}...${validatorAddr.slice(-4)}`
            : validatorAddr;

        return {
            variant: "success",
            title: "Validator Paused Successfully",
            description: `Validator ${shortAddr} has been paused. The validator will stop producing blocks until resumed.`,
            icon: <Pause className="h-5 w-5" />,
            ctx,
            durationMs: 5000,
        };
    }

    const code = r.status ?? r.error?.code ?? "ERR";
    const msg =
        r.error?.message ??
        r.error?.reason ??
        r.data?.message ??
        "Failed to pause validator. Please check your connection and try again.";

    return {
        variant: "error",
        title: "Pause Failed",
        description: `${msg} (${code})`,
        icon: <Pause className="h-5 w-5" />,
        ctx,
        sticky: true,
    };
};

// Mapper for unpause validator action
export const unpauseValidatorMap = <R extends { ok?: boolean; status?: number; error?: any; data?: any }>(
    r: R,
    ctx: any
): ToastTemplateOptions => {
    if (r.ok) {
        const validatorAddr = ctx?.form?.validatorAddress || "Validator";
        const shortAddr = validatorAddr.length > 12
            ? `${validatorAddr.slice(0, 6)}...${validatorAddr.slice(-4)}`
            : validatorAddr;

        return {
            variant: "success",
            title: "Validator Resumed Successfully",
            description: `Validator ${shortAddr} is now active and will resume producing blocks.`,
            icon: <Play className="h-5 w-5" />,
            ctx,
            durationMs: 5000,
        };
    }

    const code = r.status ?? r.error?.code ?? "ERR";
    const msg =
        r.error?.message ??
        r.error?.reason ??
        r.data?.message ??
        "Failed to resume validator. Please check your connection and try again.";

    return {
        variant: "error",
        title: "Resume Failed",
        description: `${msg} (${code})`,
        icon: <Play className="h-5 w-5" />,
        ctx,
        sticky: true,
    };
};
