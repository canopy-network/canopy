
import { createBrowserRouter } from 'react-router-dom'
import MainLayout from '@/components/layouts/MainLayout'

import Dashboard from '@/app/pages/Dashboard'
import { KeyManagement } from '@/app/pages/KeyManagement'
import { Accounts } from '@/app/pages/Accounts'
import Staking from '@/app/pages/Staking'
import Monitoring from '@/app/pages/Monitoring'

// Placeholder components for the new routes
const Portfolio = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Portfolio - Próximamente</div></div>
const Governance = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Governance - Próximamente</div></div>

const router = createBrowserRouter([
    {
        element: <MainLayout />,
        children: [
            { path: '/', element: <Dashboard /> },
            { path: '/accounts', element: <Accounts /> },
            { path: '/portfolio', element: <Portfolio /> },
            { path: '/staking', element: <Staking /> },
            { path: '/governance', element: <Governance /> },
            { path: '/monitoring', element: <Monitoring /> },
            { path: '/key-management', element: <KeyManagement /> },
        ],
    },
], {
    basename: import.meta.env.BASE_URL,
})

export default router

