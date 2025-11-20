// toast/DefaultToastItem.tsx
import React from "react";
import { ToastRenderData } from "./types";
import { X, CheckCircle2, XCircle, AlertTriangle, Info, Bell } from "lucide-react";
import { motion } from "framer-motion";

const VARIANT_STYLES: Record<NonNullable<ToastRenderData["variant"]>, {
    container: string;
    icon: React.ReactNode;
    iconBg: string;
}> = {
    success: {
        container: "bg-gradient-to-r from-bg-secondary to-bg-tertiary border-l-4 border-l-primary shadow-lg shadow-primary/20",
        icon: <CheckCircle2 className="h-5 w-5 text-primary" />,
        iconBg: "bg-primary/20"
    },
    error: {
        container: "bg-gradient-to-r from-bg-secondary to-bg-tertiary border-l-4 border-l-red-500 shadow-lg shadow-red-500/20",
        icon: <XCircle className="h-5 w-5 text-red-500" />,
        iconBg: "bg-red-500/20"
    },
    warning: {
        container: "bg-gradient-to-r from-bg-secondary to-bg-tertiary border-l-4 border-l-orange-500 shadow-lg shadow-orange-500/20",
        icon: <AlertTriangle className="h-5 w-5 text-orange-500" />,
        iconBg: "bg-orange-500/20"
    },
    info: {
        container: "bg-gradient-to-r from-bg-secondary to-bg-tertiary border-l-4 border-l-blue-500 shadow-lg shadow-blue-500/20",
        icon: <Info className="h-5 w-5 text-blue-500" />,
        iconBg: "bg-blue-500/20"
    },
    neutral: {
        container: "bg-gradient-to-r from-bg-secondary to-bg-tertiary border-l-4 border-l-gray-500 shadow-lg shadow-gray-500/10",
        icon: <Bell className="h-5 w-5 text-gray-400" />,
        iconBg: "bg-gray-500/20"
    },
};

export const DefaultToastItem: React.FC<{
    data: Required<ToastRenderData>;
    onClose: () => void;
}> = ({ data, onClose }) => {
    const styles = VARIANT_STYLES[data.variant ?? "neutral"];

    return (
        <motion.div
            className={`w-[420px] max-w-[92vw] rounded-xl border border-bg-accent backdrop-blur-md p-4 ${styles.container}`}
            initial={{ opacity: 0, x: 50 }}
            animate={{ opacity: 1, x: 0 }}
            exit={{ opacity: 0, x: 50 }}
            transition={{ type: "spring", stiffness: 300, damping: 30 }}
        >
            <div className="flex items-start gap-3">
                {/* Icon */}
                <motion.div
                    className={`flex-shrink-0 w-10 h-10 rounded-full ${styles.iconBg} flex items-center justify-center`}
                    initial={{ scale: 0, rotate: -180 }}
                    animate={{ scale: 1, rotate: 0 }}
                    transition={{ delay: 0.1, type: "spring", stiffness: 200 }}
                >
                    {data.icon || styles.icon}
                </motion.div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                    {data.title && (
                        <div className="font-semibold text-base leading-5 text-text-primary mb-1">
                            {data.title}
                        </div>
                    )}
                    {data.description && (
                        <div className="text-sm text-text-muted leading-relaxed">
                            {data.description}
                        </div>
                    )}

                    {/* Actions */}
                    {!!data.actions?.length && (
                        <div className="mt-3 flex flex-wrap gap-2">
                            {data.actions.map((a, i) =>
                                a.type === "link" ? (
                                    <a
                                        key={i}
                                        href={a.href}
                                        target={a.newTab ? "_blank" : undefined}
                                        rel={a.newTab ? "noreferrer" : undefined}
                                        className="text-sm font-medium text-primary hover:text-primary/80 underline underline-offset-2 transition-colors"
                                    >
                                        {a.label}
                                    </a>
                                ) : (
                                    <button
                                        key={i}
                                        onClick={a.onClick}
                                        className="text-sm font-medium rounded-lg px-3 py-1.5 bg-primary/10 text-primary hover:bg-primary/20 border border-primary/20 transition-all active:scale-95"
                                    >
                                        {a.label}
                                    </button>
                                )
                            )}
                        </div>
                    )}
                </div>

                {/* Close Button */}
                <button
                    onClick={onClose}
                    aria-label="Close"
                    className="flex-shrink-0 rounded-lg p-1.5 hover:bg-bg-accent/50 transition-colors active:scale-95"
                >
                    <X className="h-4 w-4 text-text-muted hover:text-text-primary" />
                </button>
            </div>
        </motion.div>
    );
};
