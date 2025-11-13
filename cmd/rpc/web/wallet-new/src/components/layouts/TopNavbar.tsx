import React from 'react';
import { motion } from 'framer-motion';
import { useLocation } from 'react-router-dom';

const routeNames: Record<string, string> = {
    '/': 'Dashboard',
    '/accounts': 'Accounts',
    '/staking': 'Staking',
    '/governance': 'Governance',
    '/monitoring': 'Monitoring',
    '/key-management': 'Key Management'
};

export const TopNavbar = (): JSX.Element => {
    const location = useLocation();
    const currentRoute = routeNames[location.pathname] || 'Dashboard';

    return (
        <motion.header
            className="bg-bg-secondary border-b border-bg-accent px-4 sm:px-6 py-3 sm:py-4 sticky top-0 z-30 hidden lg:block"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
        >
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-xl font-semibold text-text-primary">
                        {currentRoute}
                    </h1>
                </div>
                <div className="flex items-center gap-4">
                    {/* Aquí puedes agregar notificaciones, perfil, etc */}
                </div>
            </div>
        </motion.header>
    );
};
