// toast/ToastContext.tsx
"use client";
import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import { ToastApi, ToastTemplateOptions, ToastFromResultOptions } from "./types";
import { renderTemplate } from "./utils";
import { motion, AnimatePresence } from "framer-motion";
import {DefaultToastItem} from "@/toast/DefaultToastItem";

type ToastState = {
    queue: Array<Required<import("./types").ToastRenderData>>;
};

type ProviderProps = {
    children: React.ReactNode;
    maxVisible?: number;
    position?: "top-right" | "top-left" | "bottom-right" | "bottom-left";
    defaultDurationMs?: number;
    renderItem?: (t: Required<import("./types").ToastRenderData>) => React.ReactNode;
};

const ToastContext = createContext<ToastApi | null>(null);

let _id = 0;
const genId = () => `t_${Date.now()}_${_id++}`;

export const ToastProvider: React.FC<ProviderProps> = ({
                                                           children,
                                                           maxVisible = 4,
                                                           position = "top-right",
                                                           defaultDurationMs = 300000,
                                                           renderItem,
                                                       }) => {
    const [queue, setQueue] = useState<ToastState["queue"]>([]);
    const timers = useRef<Record<string, any>>({});

    const scheduleAutoDismiss = useCallback((id: string, ms?: number, sticky?: boolean) => {
        if (sticky) return;
        const dur = typeof ms === "number" ? ms : defaultDurationMs;
        timers.current[id] = setTimeout(() => {
            setQueue((q) => q.filter((x) => x.id !== id));
            delete timers.current[id];
        }, dur);
    }, [defaultDurationMs]);

    const add = useCallback((opts: ToastTemplateOptions, variant?: import("./types").ToastVariant) => {
        const id = genId();
        const data = {
            id,
            title: opts.title != null ? renderTemplate(opts.title, opts.ctx) : undefined,
            description: opts.description != null ? renderTemplate(opts.description, opts.ctx) : undefined,
            icon: opts.icon,
            actions: opts.actions,
            variant: variant ?? opts.variant ?? "neutral",
            durationMs: opts.durationMs,
            sticky: opts.sticky ?? false,
        } as Required<import("./types").ToastRenderData>;
        setQueue((q) => {
            const next = [data, ...q];
            return next.slice(0, maxVisible);
        });
        scheduleAutoDismiss(id, data.durationMs, data.sticky);
        return id;
    }, [maxVisible, scheduleAutoDismiss]);

    const dismiss = useCallback((id: string) => {
        if (timers.current[id]) {
            clearTimeout(timers.current[id]);
            delete timers.current[id];
        }
        setQueue((q) => q.filter((x) => x.id !== id));
    }, []);

    const clear = useCallback(() => {
        Object.values(timers.current).forEach(clearTimeout);
        timers.current = {};
        setQueue([]);
    }, []);

    const fromResult = useCallback(<R,>({ result, ctx, map, fallback }: ToastFromResultOptions<R>) => {
        const mapped = map?.(result as R, ctx);
        if (!mapped && !fallback) return null;
        return add(mapped ?? fallback!, mapped?.variant);
    }, [add]);

    const api = useMemo<ToastApi>(() => ({
        toast: (t) => add(t, t.variant),
        success: (t) => add({ ...t, variant: "success" }),
        error: (t) => add({ ...t, variant: "error" }),
        info: (t) => add({ ...t, variant: "info" }),
        warning: (t) => add({ ...t, variant: "warning" }),
        neutral: (t) => add({ ...t, variant: "neutral" }),
        fromResult,
        dismiss,
        clear,
    }), [add, dismiss, clear, fromResult]);

    const posClasses = {
        "top-right": "top-3 right-3 sm:top-4 sm:right-4",
        "top-left": "top-3 left-3 sm:top-4 sm:left-4",
        "bottom-right": "bottom-3 right-3 sm:bottom-4 sm:right-4",
        "bottom-left": "bottom-3 left-3 sm:bottom-4 sm:left-4",
    }[position];

    return (
        <ToastContext.Provider value={api}>
            {children}
            {/* Container */}
            <div
                className={`pointer-events-none fixed z-[9999] ${posClasses} flex flex-col gap-2.5 sm:gap-3 w-[min(100vw-1.5rem,28rem)] sm:w-[min(100vw-2rem,28rem)]`}
                style={{
                    paddingTop: "max(env(safe-area-inset-top), 0px)",
                    paddingBottom: "max(env(safe-area-inset-bottom), 0px)",
                }}
            >
                <AnimatePresence initial={false}>
                    {queue.map((t) =>
                        <motion.div
                            key={t.id}
                            initial={{ opacity: 0, y: 12, scale: 0.98 }}
                            animate={{ opacity: 1, y: 0, scale: 1 }}
                            exit={{ opacity: 0, y: 8, scale: 0.98 }}
                            transition={{ type: "spring", stiffness: 380, damping: 28 }}
                            className="pointer-events-auto"
                        >
                            {renderItem ? renderItem(t) : <DefaultToastItem data={t} onClose={() => dismiss(t.id)} />}
                        </motion.div>
                    )}
                </AnimatePresence>
            </div>
        </ToastContext.Provider>
    );
};

export const useToast = () => {
    const ctx = useContext(ToastContext);
    if (!ctx) throw new Error("useToast must be used within <ToastProvider>");
    return ctx;
};
