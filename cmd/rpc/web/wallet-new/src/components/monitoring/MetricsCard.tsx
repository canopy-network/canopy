import React from 'react';
import { motion } from 'framer-motion';

interface MetricItem {
    id: string;
    label: string;
    value: string | number;
    type?: 'status' | 'progress' | 'text' | 'address';
    color?: string;
    progress?: number;
    icon?: string;
}

interface MetricsCardProps {
    title?: string;
    metrics: MetricItem[];
    columns?: number;
    className?: string;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const MetricsCard: React.FC<MetricsCardProps> = ({
                                                            title,
                                                            metrics,
                                                            columns = 3,
                                                            className = "bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-4 mb-6"
                                                        }) => {
    const gridCols = {
        1: 'grid-cols-1',
        2: 'grid-cols-2',
        3: 'grid-cols-3',
        4: 'grid-cols-4'
    };

    const renderMetric = (metric: MetricItem) => {
        switch (metric.type) {
            case 'status':
                return (
                    <div key={metric.id} className="flex items-center gap-2">
                        <div className={`w-2 h-2 ${metric.color || 'bg-green-500'} rounded-full`}></div>
                        <div>
                            <div className="text-xs text-gray-400">{metric.label}</div>
                            <div className="text-white font-medium">{metric.value}</div>
                        </div>
                    </div>
                );

            case 'progress':
                return (
                    <div key={metric.id}>
                        <div className="text-xs text-gray-400">{metric.label}</div>
                        <div className="flex items-center gap-2">
                            <div className="flex-1 bg-[#2A2C35] h-2 rounded-full overflow-hidden">
                                <div
                                    className="bg-[#6fe3b4] h-full rounded-full"
                                    style={{ width: `${metric.progress || 0}%` }}
                                ></div>
                            </div>
                            <span className="text-white text-xs">{metric.progress}% complete</span>
                        </div>
                    </div>
                );

            case 'address':
                return (
                    <div key={metric.id} className="col-span-3">
                        <div className="text-xs text-gray-400">{metric.label}</div>
                        <div className="text-white font-medium font-mono">{metric.value}</div>
                    </div>
                );

            default:
                return (
                    <div key={metric.id}>
                        <div className="text-xs text-gray-400">{metric.label}</div>
                        <div className="text-white font-medium">{metric.value}</div>
                    </div>
                );
        }
    };

    return (
        <motion.div
            variants={itemVariants}
            className={className}
        >
            {title && (
                <h2 className="text-white text-lg font-bold mb-4">{title}</h2>
            )}
            <div className={`grid ${gridCols[columns as keyof typeof gridCols]} gap-4`}>
                {metrics.map(renderMetric)}
            </div>
        </motion.div>
    );
};
