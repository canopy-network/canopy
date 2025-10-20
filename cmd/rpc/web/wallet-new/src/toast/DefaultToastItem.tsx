// toast/DefaultToastItem.tsx
import React from "react";
import { ToastAction, ToastRenderData } from "./types";
import { X } from "lucide-react";

const VARIANT_CLASSES: Record<NonNullable<ToastRenderData["variant"]>, string> = {
    success: "border-status-success bg-primary-foreground",
    error: "border-status-error bg-primary-foreground",
    warning: "border-status-warning bg-primary-foreground",
    info: "border-status-info bg-primary-foreground",
    neutral: "border-muted bg-primary-foreground",
};

export const DefaultToastItem: React.FC<{
    data: Required<ToastRenderData>;
    onClose: () => void;
}> = ({ data, onClose }) => {
    const color = VARIANT_CLASSES[data.variant ?? "neutral"];
    return (
        <div className={`w-[380px] max-w-[92vw] rounded-md border shadow-sm p-3 ${color}`}>
            <div className="flex items-start gap-3">
                {data.icon && <div className="mt-0.5">{data.icon}</div>}
                <div className="flex-1">
                    {data.title && <div className="font-semibold leading-5 text-canopy-50">{data.title}</div>}
                    {data.description && <div className="mt-0.5 text-sm text-canopy-50 text-wrap break-all">{data.description}</div>}
                    {!!data.actions?.length && (
                        <div className="mt-2 flex flex-wrap gap-2">
                            {data.actions.map((a, i) =>
                                a.type === "link" ? (
                                    <a
                                        key={i}
                                        href={a.href}
                                        target={a.newTab ? "_blank" : undefined}
                                        rel={a.newTab ? "noreferrer" : undefined}
                                        className="text-sm underline underline-offset-2 hover:opacity-80 text-white"
                                    >
                                        {a.label}
                                    </a>
                                ) : (
                                    <button
                                        key={i}
                                        onClick={a.onClick}
                                        className="text-sm rounded-xl px-2 py-1 border bg-white hover:bg-zinc-50 active:scale-[0.98]"
                                    >
                                        {a.label}
                                    </button>
                                )
                            )}
                        </div>
                    )}
                </div>
                <button
                    onClick={onClose}
                    aria-label="Close"
                    className="rounded-full p-1 hover:bg-black/5 active:scale-95"
                >
                    <X className="h-4 w-4 text-canopy-900" />
                </button>
            </div>
        </div>
    );
};
