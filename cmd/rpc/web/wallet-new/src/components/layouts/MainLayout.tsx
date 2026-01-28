import { Outlet } from 'react-router-dom'
import { Sidebar } from "./Sidebar";
import { TopNavbar } from "./TopNavbar";
import { Footer } from "./Footer";

export default function MainLayout() {
    return (
        <div className="flex flex-col h-screen overflow-hidden">
            {/* Top Navbar - Desktop only (lg+) */}
            <TopNavbar />

            {/* Mobile/Tablet Header + Sidebar (< lg) */}
            <Sidebar />

            {/* Main Content Area */}
            <div className="flex-1 overflow-y-auto bg-primary-foreground">
                <div className="py-4 px-4 sm:px-6 lg:px-8 max-w-[1920px] mx-auto">
                    <Outlet />
                </div>
                <Footer />
            </div>
        </div>
    )
}