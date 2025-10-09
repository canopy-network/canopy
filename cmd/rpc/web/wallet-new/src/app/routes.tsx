
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import MainLayout from '../components/layouts/MainLayout'

import Dashboard from '../components/pages/Dashboard'
import { KeyManagement } from '@/components/pages/KeyManagement'
import { Accounts } from '@/components/pages/Accounts'

// Placeholder components for the new routes
const Portfolio = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Portfolio - Próximamente</div></div>
const Staking = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Staking - Próximamente</div></div>
const Governance = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Governance - Próximamente</div></div>
const Monitoring = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Monitoring - Próximamente</div></div>

const router = createBrowserRouter([
    {
        element: <MainLayout />,           // tu layout con <Outlet/>
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

