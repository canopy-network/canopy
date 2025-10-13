import React from 'react';
import { motion } from 'framer-motion';

interface PerformanceMetricsCardProps {
    processCPU: number;
    systemCPU: number;
    memoryUsage: number;
    diskIO: number;
}

const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4 } }
};

export const PerformanceMetricsCard: React.FC<PerformanceMetricsCardProps> = ({
    processCPU,
    systemCPU,
    memoryUsage,
    diskIO
}) => {
    const performanceMetrics = [
        {
            id: 'processCPU',
            label: 'Process CPU',
            value: processCPU,
            color: '#6fe3b4'
        },
        {
            id: 'systemCPU',
            label: 'System CPU',
            value: systemCPU,
            color: '#f59e0b'
        },
        {
            id: 'memoryUsage',
            label: 'Memory Usage',
            value: memoryUsage,
            color: '#ef4444'
        },
        {
            id: 'diskIO',
            label: 'Disk I/O',
            value: diskIO,
            color: '#8b5cf6'
        }
    ];

    const renderMetricBar = (metric: typeof performanceMetrics[0]) => (
        <div key={metric.id}>
            <div className="text-gray-400 text-sm mb-2">{metric.label}</div>
            <div className="h-24 bg-[#16171D] rounded-md flex items-end justify-center relative">
                <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-xl font-bold">{metric.value.toFixed(2)}%</span>
                </div>
                <div 
                    className="w-full rounded-md transition-all duration-500"
                    style={{ 
                        height: `${Math.min(metric.value, 100)}%`,
                        backgroundColor: metric.color
                    }}
                ></div>
            </div>
        </div>
    );

    return (
        <motion.div 
            variants={itemVariants}
            className="bg-[#1E1F26] rounded-xl border border-[#2A2C35] p-6"
        >
            <h2 className="text-white text-lg font-bold mb-4">Performance Metrics</h2>
            <div className="grid grid-cols-2 gap-6">
                {performanceMetrics.map(renderMetricBar)}
            </div>
        </motion.div>
    );
};
