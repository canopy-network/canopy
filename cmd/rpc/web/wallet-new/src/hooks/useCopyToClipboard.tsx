import { useToast } from "@/toast/ToastContext";
import { Copy, Check } from "lucide-react";
import { useCallback } from "react";

export const useCopyToClipboard = () => {
    const toast = useToast();

    const copyToClipboard = useCallback(async (text: string, label?: string) => {
        try {
            await navigator.clipboard.writeText(text);

            toast.success({
                title: "Copied to clipboard",
                description: label || "Text copied successfully",
                icon: <Check className="h-5 w-5" />,
                durationMs: 2000,
            });

            return true;
        } catch (err) {
            toast.error({
                title: "Failed to copy",
                description: "Unable to copy to clipboard. Please try again.",
                icon: <Copy className="h-5 w-5" />,
                sticky: false,
                durationMs: 3000,
            });

            return false;
        }
    }, [toast]);

    return { copyToClipboard };
};
