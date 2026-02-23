/**
 * SparklineChart — thin area line chart built on Chart.js v4 + react-chartjs-2.
 *
 * Registers only the modules it needs (tree-shakeable).
 * Designed for small embedded charts inside cards.
 */
import React, { useMemo, useRef } from 'react';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Filler,
    Tooltip,
    type ChartOptions,
    type ChartData,
} from 'chart.js';
import { Line } from 'react-chartjs-2';

// Register once at module level — safe to call multiple times
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, Tooltip);

// ── Design tokens ────────────────────────────────────────────────────────────
const PRIMARY        = '#35CD48';                  // hsl(128 60% 51%)
const PRIMARY_STROKE = 'rgba(53,205,72,0.50)';    // dimmed line — softer on dark bg
const PRIMARY_FILL   = 'rgba(53,205,72,0.06)';    // subtle area fill fallback
const CARD_BG        = 'hsl(0,0%,13%)';
const BORDER_CLR     = 'hsl(0,0%,20%)';
const MUTED          = '#6b7280';

// ── Types ────────────────────────────────────────────────────────────────────
export interface SparklinePoint {
    value: number;
    label: string;
}

export interface SparklineChartProps {
    /** Data points — {value, label} */
    data: SparklinePoint[];
    /** Function to format a raw value into the tooltip body string */
    formatValue?: (v: number) => string;
    /**
     * Accent colour used for hover dot and tooltip text.
     * Defaults to primary green.
     */
    color?: string;
    /**
     * Stroke (line) colour. Defaults to `color` at ~50% opacity for a softer look.
     * Pass an explicit rgba string to override.
     */
    strokeColor?: string;
    /** Fill colour. If omitted, derived from `color`. */
    fillColor?: string;
    /** Whether to show grid lines — default false */
    showGrid?: boolean;
    /** className applied to the wrapper div */
    className?: string;
    /** Explicit height (CSS). Defaults to 100% */
    height?: number | string;
}

// ── Colour helpers ────────────────────────────────────────────────────────────
/** Extract [r,g,b] from a hex (#rrggbb) or rgb/rgba(...) string. */
function toRgb(color: string): [number, number, number] {
    const hex = color.match(/^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})/i);
    if (hex) return [parseInt(hex[1], 16), parseInt(hex[2], 16), parseInt(hex[3], 16)];
    const rgb = color.match(/rgba?\(\s*(\d+),\s*(\d+),\s*(\d+)/);
    if (rgb) return [+rgb[1], +rgb[2], +rgb[3]];
    return [53, 205, 72]; // fallback: primary green
}

/** Build a top-to-bottom fade gradient for the area fill. */
function createGradient(
    ctx: CanvasRenderingContext2D,
    chartArea: { top: number; bottom: number },
    color: string,
): CanvasGradient {
    const [r, g, b] = toRgb(color);
    const grad = ctx.createLinearGradient(0, chartArea.top, 0, chartArea.bottom);
    grad.addColorStop(0, `rgba(${r},${g},${b},0.12)`);
    grad.addColorStop(1, `rgba(${r},${g},${b},0)`);
    return grad;
}

// ── Component ────────────────────────────────────────────────────────────────
export const SparklineChart = React.memo<SparklineChartProps>(({
    data,
    formatValue,
    color       = PRIMARY,
    strokeColor,
    fillColor   = PRIMARY_FILL,
    showGrid    = false,
    className   = '',
    height      = '100%',
}) => {
    const chartRef = useRef<ChartJS<'line'>>(null);

    // Compute a dimmed stroke if the caller didn't provide one explicitly
    const resolvedStroke = useMemo(() => {
        if (strokeColor) return strokeColor;
        if (color === PRIMARY) return PRIMARY_STROKE;
        const [r, g, b] = toRgb(color);
        return `rgba(${r},${g},${b},0.50)`;
    }, [color, strokeColor]);

    const labels = useMemo(() => data.map(d => d.label), [data]);
    const values = useMemo(() => data.map(d => d.value), [data]);

    const chartData = useMemo((): ChartData<'line'> => ({
        labels,
        datasets: [{
            data: values,
            fill: 'start',
            backgroundColor: fillColor,   // overridden by gradient plugin
            borderColor: resolvedStroke,
            borderWidth: 1.5,
            pointRadius: 0,
            pointHoverRadius: 3,
            pointHoverBackgroundColor: color,
            pointHoverBorderColor: CARD_BG,
            pointHoverBorderWidth: 1.5,
            tension: 0.5,                  // smooth cubic interpolation
        }],
    }), [labels, values, color, resolvedStroke, fillColor]);

    const options = useMemo((): ChartOptions<'line'> => ({
        responsive: true,
        maintainAspectRatio: false,
        // Disable animation entirely so background refetches don't cause
        // the line to re-draw from scratch and "flash".
        animation: false,

        plugins: {
            legend: { display: false },
            tooltip: {
                enabled: true,
                mode: 'index',
                intersect: false,
                backgroundColor: CARD_BG,
                borderColor: BORDER_CLR,
                borderWidth: 1,
                titleColor: MUTED,
                titleFont: { family: "'JetBrains Mono', monospace", size: 10 },
                bodyColor: color,
                bodyFont: { family: "'JetBrains Mono', monospace", size: 12, weight: 'bold' },
                padding: { x: 10, y: 8 },
                cornerRadius: 8,
                callbacks: {
                    title: (items) => items[0]?.label ?? '',
                    label: (ctx) => {
                        const y = ctx.parsed.y ?? 0;
                        return formatValue
                            ? formatValue(y)
                            : y.toLocaleString('en-US', { maximumFractionDigits: 2 });
                    },
                },
            },
        },

        scales: {
            x: {
                display: false,
                grid: { display: false },
                border: { display: false },
            },
            y: {
                display: false,
                grid: { display: showGrid, color: BORDER_CLR },
                border: { display: false },
                beginAtZero: false,
                // Always add headroom so the line never touches the edge —
                // even when all values are equal (flat line).
                afterDataLimits(scale) {
                    const range = scale.max - scale.min;
                    const pad   = range > 0 ? range * 0.12 : Math.abs(scale.max) * 0.08 || 1;
                    scale.min  -= pad;
                    scale.max  += pad;
                },
            },
        },

        interaction: {
            mode: 'index',
            intersect: false,
        },

        elements: {
            line: { borderCapStyle: 'round', borderJoinStyle: 'round' },
        },
    }), [color, showGrid, formatValue]);

    // ── Gradient plugin (runs before draw) ───────────────────────────────────
    const gradientPlugin = useMemo(() => ({
        id: 'gradientFill',
        beforeDraw(chart: ChartJS) {
            const { ctx, chartArea, data: cd } = chart;
            if (!chartArea || !cd.datasets[0]) return;
            const grad = createGradient(ctx, chartArea, color);
            (cd.datasets[0] as any).backgroundColor = grad;
        },
    }), [color]);

    if (data.length === 0) {
        return (
            <div
                className={`flex items-center justify-center text-xs text-muted-foreground font-body ${className}`}
                style={{ height }}
            >
                No data
            </div>
        );
    }

    return (
        <div className={className} style={{ height, width: '100%' }}>
            <Line
                ref={chartRef}
                data={chartData}
                options={options}
                plugins={[gradientPlugin]}
            />
        </div>
    );
});

SparklineChart.displayName = 'SparklineChart';
