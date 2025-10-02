
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import MainLayout from '../components/layouts/MainLayout'

import Dashboard from '../app/pages/Dashboard'
import { KeyManagement } from '@/components/pages/KeyManagement'

const router = createBrowserRouter([
    {
        element: <MainLayout />,           // tu layout con <Outlet/>
        children: [
            { path: '/', element: <Dashboard /> },
            { path: '/key-management', element: <KeyManagement /> },
        ],
    },
], {
    basename: import.meta.env.BASE_URL,
})

export default router

