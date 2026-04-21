import React from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Key, Blocks } from 'lucide-react';
import { useTotalStage } from '@/hooks/useTotalStage';
import { useDS } from '@/core/useDs';
import { useDenom } from '@/hooks/useDenom';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

const SUPPORT_URL = import.meta.env.VITE_SUPPORT_URL as string | undefined;
const topBarButtonClass =
    "inline-flex items-center justify-center cursor-pointer gap-2 whitespace-nowrap rounded-lg text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] aria-invalid:ring-destructive/20 aria-invalid:border-destructive h-9 px-3 text-muted-foreground hover:text-white border border-white/15";

export const TopBar = (): JSX.Element => {
    const { data: totalStage, isLoading: stageLoading } = useTotalStage();
    const { symbol, factor } = useDenom();
    const { data: blockHeight } = useDS<{ height: number }>('height', {}, {
        staleTimeMs: 10_000,
        refetchIntervalMs: 10_000,
    });

    return (
        <motion.header
            className="sticky top-0 z-[100] hidden h-16 flex-shrink-0 items-center justify-between gap-3 border-b border-border/40 bg-background px-6 lg:flex"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            <div className="flex items-center gap-3">
                <div className="flex h-9 items-center gap-2 rounded-lg border border-white/15 px-3 text-sm text-muted-foreground">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white/70 opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-white" />
                    </span>
                    <Blocks className="h-4 w-4 flex-shrink-0" />
                    <span className="num text-sm font-medium text-foreground">
                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '-'}
                    </span>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <div className="hidden h-9 items-center gap-1.5 rounded-lg border border-white/15 px-3 text-sm text-muted-foreground sm:flex">
                    <span>Total</span>
                    {stageLoading ? (
                        <span className="num text-sm font-medium text-foreground">...</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / factor : 0}
                            format={{ notation: 'compact', minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                            className="num text-sm font-medium text-foreground"
                        />
                    )}
                    <span className="num text-sm text-muted-foreground/80">{symbol}</span>
                </div>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <button
                    onClick={() => {
                        const url = SUPPORT_URL || "https://discord.com/channels/1310733928436600912/1439049045145419806/1439945810446909560";
                        window.open(url, "_blank");
                    }}
                    className={topBarButtonClass}
                >
                    <svg
                        width="16"
                        height="16"
                        viewBox="0 0 16 16"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                        className="size-4 shrink-0"
                    >
                        <path
                            d="M8 9.33333V7M6.19615 12.3224L7.57285 13.4764C7.81949 13.6832 8.17861 13.6842 8.42643 13.4789L9.8253 12.32C9.94489 12.2209 10.0953 12.1667 10.2506 12.1667H11.5C12.6046 12.1667 13.5 11.2712 13.5 10.1667V4.5C13.5 3.39543 12.6046 2.5 11.5 2.5H4.5C3.39543 2.5 2.5 3.39543 2.5 4.5V10.1667C2.5 11.2712 3.39543 12.1667 4.5 12.1667H5.76788C5.9245 12.1667 6.07612 12.2218 6.19615 12.3224Z"
                            stroke="currentColor"
                            strokeLinecap="round"
                        />
                        <path d="M8 5.33337H8.00667" stroke="currentColor" strokeWidth="1.33333" strokeLinecap="round" />
                    </svg>
                    <span className="hidden sm:inline">Create a Ticket</span>
                </button>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <Link
                    to="/key-management"
                    className={topBarButtonClass}
                >
                    <Key className="size-4 shrink-0" />
                    Keys
                </Link>
            </div>
        </motion.header>
    );
};
