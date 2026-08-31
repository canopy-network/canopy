
import { createBrowserRouter, Navigate } from 'react-router-dom'
import MainLayout from '@/components/layouts/MainLayout'

import Dashboard from '@/app/pages/Dashboard'
import { KeyManagement } from '@/app/pages/KeyManagement'
import { Accounts } from '@/app/pages/Accounts'
import Staking from '@/app/pages/Staking'
import Monitoring from '@/app/pages/Monitoring'
import Governance from '@/app/pages/Governance'
import Orders from '@/app/pages/Orders'

const router = createBrowserRouter([
    {
        element: <MainLayout />,
        children: [
            { path: '/', element: <Dashboard /> },
            { path: '/accounts', element: <Accounts /> },
            { path: '/staking', element: <Staking /> },
            { path: '/governance', element: <Governance /> },
            { path: '/orders', element: <Orders /> },
            { path: '/monitoring', element: <Monitoring /> },
            { path: '/key-management', element: <KeyManagement /> },
            { path: '/all-addresses', element: <Navigate to="/accounts" replace /> },
        ],
    },
], {
    basename: import.meta.env.BASE_URL,
})

export default router
