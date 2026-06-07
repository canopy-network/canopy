import { useToast } from "@/toast/ToastContext";
import { Copy, Check } from "lucide-react";
import { useCallback } from "react";

const fallbackCopy = (text: string) => {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.setAttribute("readonly", "");
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand("copy");
    document.body.removeChild(textarea);
};

export const useCopyToClipboard = () => {
    const toast = useToast();

    const copyToClipboard = useCallback(async (text: string, label?: string) => {
        try {
            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text);
            } else {
                fallbackCopy(text);
            }

            toast.success({
                title: "Copied to clipboard",
                description: label || "Text copied successfully",
                icon: <Check className="h-5 w-5" />,
                durationMs: 4000,
            });

            return true;
        } catch (err) {
            toast.error({
                title: "Failed to copy",
                description: "Unable to copy to clipboard. Please try again.",
                icon: <Copy className="h-5 w-5" />,
                sticky: false,
                durationMs: 5000,
            });

            return false;
        }
    }, [toast]);

    return { copyToClipboard };
};
