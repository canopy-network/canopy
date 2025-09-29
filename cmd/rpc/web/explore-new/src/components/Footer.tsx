import React from 'react'
import { Link } from 'react-router-dom'
import Logo from './Logo'

const Footer: React.FC = () => {
    return (
        <footer className="bg-navbar border-t border-gray-800/60">
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-6">
                <div className="flex items-center justify-between">
                    {/* Left side - Logo and Copyright */}
                    <div className="flex items-center gap-3">
                        <Logo size={140} showText={false} />
                        <span className="text-gray-400 text-sm">
                            © {new Date().getFullYear()} Canopy Block Explorer. All rights reserved.
                        </span>
                    </div>

                    {/* Right side - Links */}
                    <div className="flex items-center gap-6">
                        <Link
                            to="/api"
                            className="text-gray-400 hover:text-primary text-sm transition-colors duration-200"
                        >
                            API
                        </Link>
                        <Link
                            to="/docs"
                            className="text-gray-400 hover:text-primary text-sm transition-colors duration-200"
                        >
                            Docs
                        </Link>
                        <Link
                            to="/privacy"
                            className="text-gray-400 hover:text-primary text-sm transition-colors duration-200"
                        >
                            Privacy
                        </Link>
                        <Link
                            to="/terms"
                            className="text-gray-400 hover:text-primary text-sm transition-colors duration-200"
                        >
                            Terms
                        </Link>
                    </div>
                </div>
            </div>
        </footer>
    )
}

export default Footer
