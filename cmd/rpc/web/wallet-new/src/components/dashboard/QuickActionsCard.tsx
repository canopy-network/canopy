import React from 'react';
import { motion } from 'framer-motion';
import { Send, Download, Lock, ArrowLeftRight } from 'lucide-react';
import { Manifest } from '@/hooks/useManifest';

interface QuickActionsCardProps {
    manifest: Manifest | null;
}

export const QuickActionsCard = ({ manifest }: QuickActionsCardProps): JSX.Element => {
    const cardVariants = {
        hidden: { opacity: 0, y: 20 },
        visible: {
            opacity: 1,
            y: 0,
            transition: { duration: 0.4, delay: 0.2 }
        }
    };

    const buttonVariants = {
        hover: {
            scale: 1.05,
            transition: { duration: 0.2 }
        }
    };

    const actions = [
        {
            id: 'Send',
            label: 'Send',
            icon: Send,
            color: 'bg-green-500 hover:bg-green-600',
            action: manifest?.actions.find(a => a.id === 'Send')
        },
        {
            id: 'Receive',
            label: 'Receive',
            icon: Download,
            color: 'bg-blue-500 hover:bg-blue-600',
            action: null // No hay acción específica para receive en el manifest
        },
        {
            id: 'Stake',
            label: 'Stake',
            icon: Lock,
            color: 'bg-purple-500 hover:bg-purple-600',
            action: manifest?.actions.find(a => a.id === 'Stake')
        },
        {
            id: 'Swap',
            label: 'Swap',
            icon: ArrowLeftRight,
            color: 'bg-orange-500 hover:bg-orange-600',
            action: null // No hay acción específica para swap en el manifest
        }
    ];

    const handleActionClick = (action: any) => {
        if (action) {
            // Aquí implementarías la lógica para ejecutar la acción del manifest
            console.log('Executing action:', action.id);
        } else {
            // Para acciones que no están en el manifest, implementar lógica específica
            console.log('Custom action not in manifest');
        }
    };

    return (
        <motion.div
            className="bg-bg-secondary rounded-lg p-6 border border-bg-accent"
            variants={cardVariants}
        >
            <h3 className="text-white text-lg font-medium mb-4">Quick Actions</h3>
            
            <div className="grid grid-cols-2 gap-3">
                {actions.map((action, index) => (
                    <motion.button
                        key={action.id}
                        className={`${action.color} text-white p-4 rounded-lg flex flex-col items-center gap-2 font-medium transition-colors`}
                        variants={buttonVariants}
                        whileHover="hover"
                        onClick={() => handleActionClick(action.action)}
                    >
                        <action.icon className="w-5 h-5" />
                        <span className="text-sm">{action.label}</span>
                    </motion.button>
                ))}
            </div>
        </motion.div>
    );
};
