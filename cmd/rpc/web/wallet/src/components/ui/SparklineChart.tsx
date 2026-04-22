import React, { useId, useMemo, useRef, useState } from 'react';

export interface SparklinePoint {
    value: number;
    label: string;
}

export interface SparklineChartProps {
    data: SparklinePoint[];
    formatValue?: (v: number) => string;
    color?: string;
    strokeColor?: string;
    fillColor?: string;
    showGrid?: boolean;
    className?: string;
    height?: number | string;
}

const UP_COLOR = '#35cd48';
const DOWN_COLOR = '#ff1744';

type TrendRun = {
    key: string;
    color: string;
    startIndex: number;
    endIndex: number;
    linePoints: string;
    polygonPoints: string;
};

export const SparklineChart = React.memo<SparklineChartProps>(({
    data,
    formatValue,
    color,
    strokeColor,
    fillColor,
    className = '',
    height = '100%',
}) => {
    const gradientId = useId();
    const containerRef = useRef<HTMLDivElement>(null);
    const [activeIndex, setActiveIndex] = useState<number | null>(null);

    const chart = useMemo(() => {
        if (data.length < 2) return null;

        const width = 100;
        const viewHeight = 32;
        const horizontalPadding = 0;
        const verticalPadding = 5;
        const values = data.map((point) => point.value);
        const min = Math.min(...values);
        const max = Math.max(...values);
        const range = max - min || 1;

        const coordinates = values.map((value, index) => {
            const x = horizontalPadding + (index / (values.length - 1)) * (width - horizontalPadding * 2);
            const y = verticalPadding + (1 - ((value - min) / range)) * (viewHeight - verticalPadding * 2);
            return {
                x,
                y,
                value,
                label: data[index]?.label ?? '',
                formattedValue: formatValue
                    ? formatValue(value)
                    : value.toLocaleString('en-US', { maximumFractionDigits: 2 }),
            };
        });

        const points = coordinates.map(({ x, y }) => `${x},${y}`).join(' ');
        const segmentColors = coordinates.slice(0, -1).map((point, index) => {
            const next = coordinates[index + 1];
            const isSegmentUp = next.value >= point.value;
            return strokeColor ?? color ?? (isSegmentUp ? UP_COLOR : DOWN_COLOR);
        });
        const trendRuns: TrendRun[] = [];

        if (segmentColors.length > 0) {
            let runStart = 0;

            for (let i = 1; i <= segmentColors.length; i += 1) {
                const isBoundary = i === segmentColors.length || segmentColors[i] !== segmentColors[runStart];

                if (!isBoundary) continue;

                const runEndSegment = i - 1;
                const runPoints = coordinates.slice(runStart, runEndSegment + 2);
                const linePoints = runPoints.map(({ x, y }) => `${x},${y}`).join(' ');
                const first = runPoints[0];
                const last = runPoints[runPoints.length - 1];
                const polygonPoints = `${linePoints} ${last.x},${viewHeight} ${first.x},${viewHeight}`;

                trendRuns.push({
                    key: `${runStart}-${runEndSegment}-${segmentColors[runStart]}`,
                    color: segmentColors[runStart],
                    startIndex: runStart,
                    endIndex: runEndSegment + 1,
                    linePoints,
                    polygonPoints,
                });

                runStart = i;
            }
        }

        const lastValue = values[values.length - 1];
        const firstValue = values[0];
        const isUp = lastValue >= firstValue;
        const finalStroke = strokeColor ?? color ?? (isUp ? UP_COLOR : DOWN_COLOR);
        const title = `${data[0]?.label ?? 'Start'}: ${formatValue ? formatValue(firstValue) : firstValue} | ${data[data.length - 1]?.label ?? 'Now'}: ${formatValue ? formatValue(lastValue) : lastValue}`;
        const currentY = coordinates[coordinates.length - 1]?.y ?? viewHeight / 2;
        const currentX = coordinates[coordinates.length - 1]?.x ?? width;
        const gridRatios = [0.2, 0.4, 0.6, 0.8];
        const gridXPositions = gridRatios.map((ratio) => {
            const index = Math.round(ratio * (coordinates.length - 1));
            return coordinates[index]?.x ?? width * ratio;
        });
        const gridYPositions = gridRatios.map((ratio) =>
            verticalPadding + (viewHeight - verticalPadding * 2) * ratio,
        );

        return {
            width,
            viewHeight,
            coordinates,
            currentX,
            currentY,
            gridXPositions,
            gridYPositions,
            trendRuns,
            stroke: finalStroke,
            title,
        };
    }, [color, data, fillColor, formatValue, strokeColor]);

    const activePoint =
        chart && activeIndex != null ? chart.coordinates[activeIndex] : null;
    const activeColor =
        chart && activeIndex != null
            ? chart.trendRuns.find((run) => activeIndex >= run.startIndex && activeIndex <= run.endIndex)?.color ?? chart.stroke
            : chart?.stroke;

    if (!chart) {
        return (
            <div
                className={`flex items-center justify-center text-xs text-muted-foreground ${className}`}
                style={{ height }}
            >
                No data
            </div>
        );
    }

    const handlePointerMove = (clientX: number) => {
        const rect = containerRef.current?.getBoundingClientRect();
        if (!rect || rect.width <= 0) return;
        const x = Math.max(0, Math.min(rect.width, clientX - rect.left));
        const ratio = x / rect.width;
        const index = Math.round(ratio * (chart.coordinates.length - 1));
        setActiveIndex(index);
    };

    return (
        <div
            ref={containerRef}
            className={`relative ${className}`}
            style={{ height, width: '100%' }}
            onMouseMove={(event) => handlePointerMove(event.clientX)}
            onMouseLeave={() => setActiveIndex(null)}
            onTouchStart={(event) => handlePointerMove(event.touches[0].clientX)}
            onTouchMove={(event) => handlePointerMove(event.touches[0].clientX)}
            onTouchEnd={() => setActiveIndex(null)}
        >
            {activePoint ? (
                <div
                    className="pointer-events-none absolute top-1 z-10 -translate-x-1/2 rounded-md border border-white/10 bg-card/95 px-2 py-1 text-[10px] font-medium text-foreground shadow-lg backdrop-blur-sm"
                    style={{ left: `${(activePoint.x / chart.width) * 100}%` }}
                >
                    <div>{activePoint.formattedValue}</div>
                    {activePoint.label ? (
                        <div className="text-[9px] text-muted-foreground">{activePoint.label}</div>
                    ) : null}
                </div>
            ) : null}
            <svg
                viewBox={`0 0 ${chart.width} ${chart.viewHeight}`}
                preserveAspectRatio="none"
                style={{ display: 'block', width: '100%', height: '100%' }}
                aria-label={chart.title}
            >
                {chart.title ? <title>{chart.title}</title> : null}
                {chart.gridYPositions.map((y, index) => (
                    <line
                        key={`h-${index}`}
                        x1={0}
                        y1={y}
                        x2={chart.width}
                        y2={y}
                        stroke="rgba(255,255,255,0.08)"
                        strokeWidth="0.16"
                    />
                ))}
                {chart.gridXPositions.map((x, index) => (
                    <line
                        key={`v-${index}`}
                        x1={x}
                        y1={0}
                        x2={x}
                        y2={chart.viewHeight}
                        stroke="rgba(255,255,255,0.08)"
                        strokeWidth="0.16"
                    />
                ))}
                <line
                    x1={0}
                    y1={chart.currentY}
                    x2={chart.width}
                    y2={chart.currentY}
                    stroke={UP_COLOR}
                    strokeOpacity="0.4"
                    strokeWidth="0.22"
                    strokeDasharray="0.8 1.2"
                />
                <defs>
                    {chart.trendRuns.map((run, index) => (
                        <linearGradient key={run.key} id={`${gradientId}-${index}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="0%" stopColor={run.color} stopOpacity="0.5" />
                            <stop offset="100%" stopColor={run.color} stopOpacity="0" />
                        </linearGradient>
                    ))}
                </defs>
                {chart.trendRuns.map((run, index) => (
                    <polygon
                        key={run.key}
                        points={run.polygonPoints}
                        fill={`url(#${gradientId}-${index})`}
                    />
                ))}
                {chart.trendRuns.map((run) => (
                    <polyline
                        key={run.key}
                        points={run.linePoints}
                        fill="none"
                        stroke={run.color}
                        strokeWidth="0.5"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                    />
                ))}
                {activePoint ? (
                    <line
                        x1={activePoint.x}
                        y1={0}
                        x2={activePoint.x}
                        y2={chart.viewHeight}
                        stroke={activeColor}
                        strokeOpacity="0.2"
                        strokeWidth="0.45"
                        strokeDasharray="1.5 1.5"
                    />
                ) : null}
            </svg>
            {activePoint ? (
                <div
                    className="pointer-events-none absolute -translate-x-1/2 -translate-y-1/2 rounded-full border border-[#18181b] shadow-[0_0_0_1px_rgba(24,24,27,0.7)]"
                    style={{
                        left: `${(activePoint.x / chart.width) * 100}%`,
                        top: `${(activePoint.y / chart.viewHeight) * 100}%`,
                        width: '8px',
                        height: '8px',
                        backgroundColor: activeColor,
                    }}
                />
            ) : null}
            <div
                className="pointer-events-none absolute -translate-x-1/2 -translate-y-1/2"
                style={{
                    left: `${(chart.currentX / chart.width) * 100}%`,
                    top: `${(chart.currentY / chart.viewHeight) * 100}%`,
                }}
            >
                <span className="absolute left-1/2 top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 animate-ping rounded-full bg-[#35cd48]/60" />
                <span className="relative block h-1.5 w-1.5 rounded-full bg-[#35cd48] shadow-[0_0_8px_rgba(53,205,72,0.85)]" />
            </div>
        </div>
    );
});

SparklineChart.displayName = 'SparklineChart';
