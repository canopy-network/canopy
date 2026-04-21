import { Outlet } from 'react-router-dom'
import AppConfetti from './AppConfetti'
import { AppSidebar } from './AppSidebar'
import { TopBar } from './TopBar'

export default function MainLayout() {
    return (
        <div className="relative flex h-dvh min-h-dvh overflow-hidden bg-background">
            <AppConfetti />
            <AppSidebar />

            <div className="relative z-10 flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
                <TopBar />

                <main className="relative z-10 min-h-0 flex-1 overflow-y-auto pt-16 lg:pt-0">
                    <div className="mx-auto max-w-[1600px] px-4 py-4 sm:px-5 sm:py-5 lg:py-4">
                        <Outlet />
                    </div>
                </main>
            </div>
        </div>
    )
}
