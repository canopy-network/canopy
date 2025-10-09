
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import MainLayout from '../components/layouts/MainLayout'

import Dashboard from '@/app/pages/Dashboard'
import { KeyManagement } from '@/app/pages/KeyManagement'
import {Staking} from "@/app/pages/Staking";
import {Send} from "@/app/pages/actions/Send";


// Placeholder components for the new routes
const Portfolio = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Portfolio - Próximamente</div></div>
const Governance = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Governance - Próximamente</div></div>
const Monitoring = () => <div className="min-h-screen bg-bg-primary flex items-center justify-center"><div className="text-white text-xl">Monitoring - Próximamente</div></div>

const router = createBrowserRouter([
    {
        element: <MainLayout />,           // tu layout con <Outlet/>
        children: [
            { path: '/', element: <Dashboard /> },
            { path: '/portfolio', element: <Portfolio /> },
            { path: '/staking', element: <Staking/>},
            { path: '/governance', element: <Governance /> },
            { path: '/monitoring', element: <Monitoring /> },
            { path: '/key-management', element: <KeyManagement /> },
            {path: '/actions', children: [
                {path: 'send', element: <Send />}
            ]}

        ],
    },
], {
    basename: import.meta.env.BASE_URL,
})

export default router

