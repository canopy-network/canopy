import { Outlet } from 'react-router-dom'
import { AppSidebar } from './AppSidebar'
import { TopBar } from './TopBar'

export default function MainLayout() {
    return (
        <div className="relative flex h-screen overflow-hidden bg-background">
            <div className="pointer-events-none absolute inset-0 z-0 bg-[radial-gradient(80%_60%_at_82%_-10%,rgba(53,205,72,0.12),transparent_55%)]" />

            <AppSidebar />

            <div className="relative z-10 flex min-w-0 flex-1 flex-col overflow-hidden">
                <TopBar />

                <main className="relative z-10 flex-1 overflow-y-auto">
                    <div className="mx-auto max-w-[1600px] px-4 py-4 pt-[68px] sm:px-5 sm:py-5 lg:pt-4">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    )
}
