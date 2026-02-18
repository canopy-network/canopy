import { Outlet } from 'react-router-dom'
import { AppSidebar } from './AppSidebar'
import { TopBar } from './TopBar'

export default function MainLayout() {
    return (
        <div className="flex h-screen overflow-hidden bg-background relative">
            {/* Permanent left sidebar — desktop */}
            <AppSidebar />

            {/* Right column: topbar + scrollable content */}
            <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
                <TopBar />

                <main className="flex-1 overflow-y-auto relative z-10">
                    {/* pt-[52px] on mobile to clear the fixed mobile header */}
                    <div className="px-4 py-4 pt-[68px] lg:pt-4 sm:px-5 sm:py-5 max-w-[1600px] mx-auto">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    )
}
