import React from 'react'

const Footer: React.FC = () => {
    return (
        <footer className="mt-6 border-t border-white/10 bg-transparent">
            <div className="mx-auto px-1 py-6 sm:px-2">
                {/* Desktop Layout */}
                <div className="hidden md:flex items-center justify-between">
                    {/* Right side - Links */}
                    <div className="flex items-center gap-6">
                        <a
                            href="https://canopy-network.gitbook.io/docs/secure-canopy/node-runner"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200"
                        >
                            API
                        </a>
                        <a
                            href="https://canopy-network.gitbook.io/docs"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200"
                        >
                            Docs
                        </a>
                        <a
                            href="https://www.canopynetwork.org/privacy-policy"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200"
                        >
                            Privacy
                        </a>
                        <a
                            href="https://www.canopynetwork.org/terms-of-service"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200"
                        >
                            Terms
                        </a>
                    </div>
                </div>

                {/* Mobile Layout */}
                <div className="md:hidden">
                    {/* Links Grid */}
                    <div className="grid grid-cols-2 gap-3 mb-4">
                        <a
                            href="https://canopy-network.gitbook.io/docs/secure-canopy/node-runner"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200 text-center py-2"
                        >
                            API
                        </a>
                        <a
                            href="https://canopy-network.gitbook.io/docs"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200 text-center py-2"
                        >
                            Docs
                        </a>
                        <a
                            href="https://www.canopynetwork.org/privacy-policy"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200 text-center py-2"
                        >
                            Privacy
                        </a>
                        <a
                            href="https://www.canopynetwork.org/terms-of-service"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-white text-sm transition-colors duration-200 text-center py-2"
                        >
                            Terms
                        </a>
                    </div>
                </div>
            </div>
        </footer>
    )
}

export default Footer
