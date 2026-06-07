import { Outlet } from 'react-router-dom'
import Footer from '../Footer'
import { ExplorerSidebar } from './ExplorerSidebar'
import { ExplorerTopBar } from './ExplorerTopBar'

export default function ExplorerLayout() {
    return (
        <div className="relative flex h-dvh min-h-dvh overflow-hidden bg-background">
            <ExplorerSidebar />

            <div className="relative z-10 flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
                <ExplorerTopBar />

                <main className="relative z-10 min-h-0 flex-1 overflow-y-auto pt-16 lg:pt-0">
                    <div className="mx-auto flex min-h-full w-full max-w-[1600px] flex-col px-4 py-4 sm:px-5 sm:py-5 lg:py-4">
                        <div className="min-h-0 flex-1">
                            <Outlet />
                        </div>
                        <Footer />
                    </div>
                </main>
            </div>
        </div>
    )
}
