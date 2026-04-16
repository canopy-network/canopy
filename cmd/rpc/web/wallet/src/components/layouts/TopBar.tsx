import React from 'react';
import { motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { Key, Blocks } from 'lucide-react';
import { useTotalStage } from '@/hooks/useTotalStage';
import { useDS } from '@/core/useDs';
import AnimatedNumber from '@/components/ui/AnimatedNumber';

export const TopBar = (): JSX.Element => {
    const { data: totalStage, isLoading: stageLoading } = useTotalStage();
    const { data: blockHeight } = useDS<{ height: number }>('height', {}, {
        staleTimeMs: 10_000,
        refetchIntervalMs: 10_000,
    });

    return (
        <motion.header
            className="relative z-20 hidden h-[52px] flex-shrink-0 items-center justify-between gap-3 border-b border-border/40 bg-background px-5 lg:flex"
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.2 }}
        >
            <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 rounded-md border border-white/15 bg-white/[0.06] px-2.5 py-1.5">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white/70 opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-white" />
                    </span>
                    <Blocks className="h-3 w-3 flex-shrink-0 text-white/60" />
                    <span className="num text-xs font-semibold text-white">
                        {blockHeight != null ? `#${blockHeight.height.toLocaleString()}` : '-'}
                    </span>
                </div>
            </div>

            <div className="flex items-center gap-2">
                <div className="hidden items-center gap-1.5 rounded-md border border-border/70 bg-secondary/75 px-2.5 py-1.5 sm:flex">
                    <span className="text-xs text-muted-foreground">Total</span>
                    {stageLoading ? (
                        <span className="num text-xs font-semibold text-white">...</span>
                    ) : (
                        <AnimatedNumber
                            value={totalStage ? totalStage / 1_000_000 : 0}
                            format={{ notation: 'compact', maximumFractionDigits: 1 }}
                            className="num text-xs font-semibold text-white"
                        />
                    )}
                    <span className="num text-xs text-muted-foreground/60">CNPY</span>
                </div>

                <div className="hidden h-4 w-px bg-border/70 sm:block" />

                <button
                    onClick={() => window.open("https://discord.com/channels/1310733928436600912/1439049045145419806/1439945810446909560", "_blank")}
                    className="flex h-8 items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.08] px-3 text-xs font-semibold text-white transition-all duration-150 hover:bg-white/[0.14]"
                >
                    <svg
                        width="16"
                        height="16"
                        viewBox="0 0 16 16"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                        className="h-3 w-3 flex-shrink-0"
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
                    className="flex h-8 items-center gap-1.5 rounded-md border border-white/15 bg-white/[0.08] px-3 text-xs font-semibold text-white transition-all duration-150 hover:bg-white/[0.14]"
                >
                    <Key className="h-3 w-3 flex-shrink-0" />
                    <span className="hidden sm:inline">Keys</span>
                </Link>
            </div>
        </motion.header>
    );
};
