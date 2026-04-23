import React, { useMemo, useCallback, useRef, useEffect, useState } from 'react';
import { Check, Copy, Download, Pause, Play, Trash2 } from 'lucide-react';

interface NodeLogsProps {
    logs: string[];
    isPaused: boolean;
    onPauseToggle: () => void;
    onClearLogs: () => void;
    onExportLogs: () => void;
}

export default function NodeLogs({
                                     logs,
                                     isPaused,
                                     onPauseToggle,
                                     onClearLogs,
                                     onExportLogs
                                 }: NodeLogsProps): JSX.Element {
    const containerRef = useRef<HTMLDivElement>(null);
    const copyResetRef = useRef<number | null>(null);
    const MAX_LOGS = 1000;
    const [copied, setCopied] = useState(false);

    const limitedLogs = useMemo(() => {
        return logs.slice(0, MAX_LOGS);
    }, [logs]);

    const formatLogLine = useCallback((line: string) => {
        const patterns = [
            [/\[90m/g, '<span style="color: #9ca3af">'],
            [/\[0m/g, '</span>'],
            [/\[34mDEBUG/g, '<span style="color: #3b82f6; font-weight: 500">DEBUG</span>'],
            [/\[32mINFO/g, '<span style="color: #22d3ee; font-weight: 500">INFO</span>'],
            [/\[33mWARN/g, '<span style="color: #fbbf24; font-weight: 500">WARN</span>'],
            [/\[31mERROR/g, '<span style="color: #ef4444; font-weight: 500">ERROR</span>'],
            [/(node-\d+)/g, '<span style="color: #ffffff; font-weight: 500">$1</span>'],
            [/(PROPOSE|PROPOSE_VOTE|PRECOMMIT_VOTE)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(🔒|Locked on proposal)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(👑|Proposer is)/g, '<span style="color: #f59e0b; font-weight: 500">$1</span>'],
            [/(Validating proposal from leader)/g, '<span style="color: #f59e0b; font-weight: 500">$1</span>'],
            [/(Applying block)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(✅|is valid)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(VDF disabled)/g, '<span style="color: #22d3ee; font-weight: 500">$1</span>'],
            [/([a-f0-9]{8,})/g, '<span style="color: #22d3ee; font-family: monospace; font-weight: 400">$1</span>'],
            [/(message from proposer:)/g, '<span style="color: #9ca3af; font-weight: 400">$1</span>'],
            [/(Process time|Wait time)/g, '<span style="color: #fbbf24; font-weight: 500">$1</span>'],
            [/(Self sending)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(Sending to \d+ replicas)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(Adding vote from replica)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(Received.*message from)/g, '<span style="color: #22d3ee; font-weight: 500">$1</span>'],
            [/(Committing to store)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(Indexing block)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(TryCommit block)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(Handling peer block)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(Handling block message)/g, '<span style="color: #3b82f6; font-weight: 500">$1</span>'],
            [/(Gossiping certificate)/g, '<span style="color: #22d3ee; font-weight: 500">$1</span>'],
            [/(Sent peer book request)/g, '<span style="color: #22d3ee; font-weight: 500">$1</span>'],
            [/(Reset BFT)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(NEW_HEIGHT|NEW_COMMITTEE)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(Updating must connects)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(Updating root chain info)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(Done checking mempool)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(Validating mempool)/g, '<span style="color: #fbbf24; font-weight: 500">$1</span>'],
            [/(🔒|Committed block)/g, '<span style="color: #10b981; font-weight: 500">$1</span>'],
            [/(✉️|Received new block)/g, '<span style="color: #22d3ee; font-weight: 500">$1</span>'],
            [/(🗳️|Self is a leader candidate)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(Voting.*as the proposer)/g, '<span style="color: #a855f7; font-weight: 500">$1</span>'],
            [/(No election candidates)/g, '<span style="color: #fbbf24; font-weight: 500">$1</span>'],
            [/(falling back to weighted pseudorandom)/g, '<span style="color: #fbbf24; font-weight: 500">$1</span>'],
            [/(Self is the proposer)/g, '<span style="color: #f59e0b; font-weight: 500">$1</span>'],
            [/(Producing proposal as leader)/g, '<span style="color: #f59e0b; font-weight: 500">$1</span>']
        ];

        let html = line;
        for (const [pattern, replacement] of patterns) {
            html = html.replace(pattern, replacement as string);
        }

        return <span dangerouslySetInnerHTML={{ __html: html }} />;
    }, []);

    useEffect(() => {
        if (containerRef.current && !isPaused) {
            containerRef.current.scrollTop = 0;
        }
    }, [limitedLogs, isPaused]);

    useEffect(() => {
        return () => {
            if (copyResetRef.current !== null) {
                window.clearTimeout(copyResetRef.current);
            }
        };
    }, []);

    const handleCopyLogs = useCallback(async () => {
        if (limitedLogs.length === 0 || !navigator.clipboard) return;

        await navigator.clipboard.writeText(limitedLogs.join('\n'));
        setCopied(true);

        if (copyResetRef.current !== null) {
            window.clearTimeout(copyResetRef.current);
        }

        copyResetRef.current = window.setTimeout(() => {
            setCopied(false);
            copyResetRef.current = null;
        }, 1500);
    }, [limitedLogs]);

    const LogLine = React.memo(({ log, index }: { log: string; index: number }) => (
        <div className="mb-1">
            {formatLogLine(log)}
        </div>
    ));
    return (
        <div className="bg-card rounded-xl border border-border p-6">
            <div className="flex justify-between items-center mb-4">
                <div>
                    <h2 className="text-foreground text-lg font-bold">
                        Node Logs
                    </h2>
                    <p className="text-xs text-muted-foreground">
                        ({limitedLogs.length} newest lines)
                    </p>
                </div>
                <div className="flex gap-2">
                    <button
                        onClick={onPauseToggle}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80"
                        title={isPaused ? "Resume" : "Pause"}
                        aria-label={isPaused ? "Resume logs" : "Pause logs"}
                    >
                        {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
                    </button>
                    <button
                        onClick={onClearLogs}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80"
                        title="Clear"
                    >
                        <Trash2 className="h-4 w-4" />
                    </button>
                    <button
                        onClick={handleCopyLogs}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80 disabled:pointer-events-none disabled:opacity-40"
                        title={copied ? 'Copied' : 'Copy Logs'}
                        aria-label="Copy logs"
                        disabled={limitedLogs.length === 0}
                    >
                        {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </button>
                    <button
                        onClick={onExportLogs}
                        className="rounded-md border border-[#272729] p-2 text-white/60 transition-colors hover:border-white/15 hover:bg-[#0f0f0f] hover:text-white/80"
                        title="Export Logs"
                    >
                        <Download className="h-4 w-4" />
                    </button>
                </div>
            </div>
            <div
                ref={containerRef}
                className="h-[26rem] overflow-y-auto rounded-lg bg-background p-4 text-xs text-muted-foreground lg:h-[34rem]"
            >
                {limitedLogs.length > 0 ? (
                    limitedLogs.map((log, index) => (
                        <LogLine key={`${index}-${log.slice(0, 20)}`} log={log} index={index} />
                    ))
                ) : (
                    <div className="text-muted-foreground">No logs available</div>
                )}
            </div>
        </div>
    );
}
