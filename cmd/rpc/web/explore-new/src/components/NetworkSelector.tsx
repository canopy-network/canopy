import React, { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface Network {
    id: string
    name: string
    rpcUrl: string
    adminRpcUrl: string
    chainId: number
    isTestnet?: boolean
}

const networks: Network[] = [
    {
        id: 'mainnet',
        name: 'Canopy Mainnet',
        rpcUrl: 'https://node1.canopy.us.nodefleet.net/rpc/',
        adminRpcUrl: 'https://node1.canopy.us.nodefleet.net/admin/',
        chainId: 1,
        isTestnet: false
    },
    {
        id: 'canary',
        name: 'Canary Mainnet',
        rpcUrl: 'https://node2.canopy.us.nodefleet.net/rpc/',
        adminRpcUrl: 'https://node2.canopy.us.nodefleet.net/admin/',
        chainId: 1,
        isTestnet: true
    }
]

const NetworkSelector: React.FC = () => {
    const [isOpen, setIsOpen] = useState(false)
    const [selectedNetwork, setSelectedNetwork] = useState<Network>(networks[0])
    const dropdownRef = useRef<HTMLDivElement>(null)

    // Load saved network from localStorage
    useEffect(() => {
        const savedNetworkId = localStorage.getItem('selectedNetworkId')
        if (savedNetworkId) {
            const network = networks.find(n => n.id === savedNetworkId)
            if (network) {
                setSelectedNetwork(network)
                updateApiConfig(network)
            }
        }
    }, [])

    // Close dropdown when clicking outside
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                setIsOpen(false)
            }
        }

        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    const updateApiConfig = (network: Network) => {
        // Update window.__CONFIG__ for immediate effect
        if (typeof window !== 'undefined') {
            window.__CONFIG__ = {
                rpcURL: network.rpcUrl,
                adminRPCURL: network.adminRpcUrl,
                chainId: network.chainId
            }
        }

        // Save to localStorage
        localStorage.setItem('selectedNetworkId', network.id)

        // Dispatch custom event to notify other components
        window.dispatchEvent(new CustomEvent('networkChanged', { detail: network }))
    }

    const handleNetworkSelect = (network: Network) => {
        setSelectedNetwork(network)
        updateApiConfig(network)
        setIsOpen(false)

        // Reload the page to apply new network settings
        window.location.reload()
    }

    return (
        <div className="relative" ref={dropdownRef}>
            <button
                onClick={() => setIsOpen(!isOpen)}
                className="flex items-center space-x-2 px-3 py-2 rounded-md text-sm font-medium text-gray-300 hover:text-primary hover:bg-gray-700 transition-colors duration-200"
            >
                <div className="flex items-center space-x-2">
                    <div className={`w-2 h-2 rounded-full ${selectedNetwork.isTestnet ? 'bg-yellow-400' : 'bg-green-400'}`} />
                    <span className="hidden sm:block">{selectedNetwork.name}</span>
                    <span className="sm:hidden">{selectedNetwork.name.split(' ')[0]}</span>
                </div>
                <motion.svg
                    className="h-4 w-4"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                    animate={{ rotate: isOpen ? 180 : 0 }}
                    transition={{ type: 'spring', stiffness: 300, damping: 20 }}
                >
                    <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 10.94l3.71-3.71a.75.75 0 111.06 1.06l-4.24 4.24a.75.75 0 01-1.06 0L5.21 8.29a.75.75 0 01.02-1.08z" clipRule="evenodd" />
                </motion.svg>
            </button>

            <AnimatePresence>
                {isOpen && (
                    <motion.div
                        initial={{ opacity: 0, y: -8, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -6, scale: 0.98 }}
                        transition={{ duration: 0.18, ease: 'easeOut' }}
                        className="absolute right-0 mt-2 min-w-[200px] overflow-hidden rounded-lg border border-gray-700/70 bg-card shadow-2xl z-50"
                    >
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="pointer-events-none absolute inset-0 bg-gradient-to-b from-white/2 to-transparent"
                        />
                        <div className="py-1 relative">
                            {networks.map((network, index) => (
                                <motion.button
                                    key={network.id}
                                    initial={{ opacity: 0, y: -6 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    transition={{ delay: 0.03 * index, duration: 0.14 }}
                                    onClick={() => handleNetworkSelect(network)}
                                    className={`w-full text-left px-3 py-2 text-sm font-normal transition-colors duration-200 flex items-center space-x-3 ${selectedNetwork.id === network.id
                                        ? 'text-primary bg-primary/10'
                                        : 'text-gray-300 hover:text-primary hover:bg-gray-700/70'
                                        }`}
                                >
                                    <div className={`w-2 h-2 rounded-full ${network.isTestnet ? 'bg-yellow-400' : 'bg-green-400'}`} />
                                    <div className="flex-1">
                                        <div className="font-medium">{network.name}</div>
                                        <div className="text-xs text-gray-500 truncate">{network.rpcUrl}</div>
                                    </div>
                                    {selectedNetwork.id === network.id && (
                                        <motion.svg
                                            initial={{ scale: 0 }}
                                            animate={{ scale: 1 }}
                                            className="h-4 w-4 text-primary"
                                            viewBox="0 0 20 20"
                                            fill="currentColor"
                                        >
                                            <path fillRule="evenodd" d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z" clipRule="evenodd" />
                                        </motion.svg>
                                    )}
                                </motion.button>
                            ))}
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    )
}

export default NetworkSelector
