
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import MainLayout from '../components/layouts/MainLayout'

import Dashboard from '../components/pages/Dashboard'
import { KeyManagement } from '@/components/pages/KeyManagement'
import { Accounts } from '@/components/pages/Accounts'
import Staking from '@/components/pages/Staking'
import Monitoring from '@/components/pages/Monitoring'
import Governance from '@/components/pages/Governance'

const router = createBrowserRouter([
    {
        element: <MainLayout />,           // tu layout con <Outlet/>
        children: [
            { path: '/', element: <Dashboard /> },
            { path: '/accounts', element: <Accounts /> },
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

