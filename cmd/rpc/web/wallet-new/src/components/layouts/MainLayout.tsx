import { Outlet, NavLink } from 'react-router-dom'
import { Navbar } from "./Navbar";
import { Footer } from "./Footer";

export default function MainLayout() {
    return (
        <div className="min-h-screen flex flex-col">
            <Navbar />
            <main className="flex-1 py-4 px-6 bg-primary-foreground">
                <Outlet />
            </main>
            <Footer />
        </div>
    )
}