import { useManifest } from '@/hooks/useManifest';
import React, { useMemo, useCallback, useRef, useEffect } from 'react';

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
    const ITEMS_PER_PAGE = 50;
    const MAX_LOGS = 1000;

    const limitedLogs = useMemo(() => {
        return logs.slice(-MAX_LOGS);
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

    const visibleLogs = useMemo(() => {
        const start = Math.max(0, limitedLogs.length - ITEMS_PER_PAGE);
        const end = limitedLogs.length;
        return limitedLogs.slice(start, end);
    }, [limitedLogs]);

    useEffect(() => {
        if (containerRef.current && !isPaused) {
            containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
    }, [visibleLogs, isPaused]);

    const LogLine = React.memo(({ log, index }: { log: string; index: number }) => (
        <div className="mb-1">
            {formatLogLine(log)}
        </div>
    ));
    return (
        <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6 min-h-[48rem]">
            <div className="flex justify-between items-center mb-4">
                <div>
                    <h2 className="text-text-primary text-lg font-bold">
                        Node Logs
                    </h2>
                    <p className="text-xs text-text-muted">
                        ({limitedLogs.length} lines, showing last {ITEMS_PER_PAGE})
                    </p>
                </div>
                <div className="flex gap-2">
                    <button
                        onClick={onPauseToggle}
                        className="p-2 hover:bg-bg-secondary rounded-md transition-colors"
                        title={isPaused ? "Resume" : "Pause"}
                    >
                        <i className={`fa-solid ${isPaused ? 'fa-play' : 'fa-pause'} text-text-muted`}></i>
                    </button>
                    <button
                        onClick={onClearLogs}
                        className="p-2 hover:bg-bg-secondary rounded-md transition-colors"
                        title="Clear"
                    >
                        <i className="fa-solid fa-trash text-text-muted"></i>
                    </button>
                    <button
                        onClick={onExportLogs}
                        className="p-2 hover:bg-bg-secondary rounded-md transition-colors"
                        title="Export Logs"
                    >
                        <i className="fa-solid fa-download text-text-muted"></i>
                    </button>
                </div>
            </div>
            <div
                ref={containerRef}
                className="bg-bg-primary rounded-lg text-gray-400 p-4 max-h-[41rem] overflow-y-auto font-mono text-xs"
            >
                {visibleLogs.length > 0 ? (
                    visibleLogs.map((log, index) => (
                        <LogLine key={`${index}-${log.slice(0, 20)}`} log={log} index={index} />
                    ))
                ) : (
                    <div className="text-text-muted">No logs available</div>
                )}
            </div>
        </div>
    );
}