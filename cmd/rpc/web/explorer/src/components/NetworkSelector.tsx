import React, { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Check, ChevronDown } from 'lucide-react'

interface Network {
    id: string
    name: string
    rpcUrl: string
    adminRpcUrl: string
    chainId: number
    isTestnet?: boolean
}

const isNetlifyHost = typeof window !== 'undefined' && window.location.hostname === 'canopy.nodefleet.net'

const networks: Network[] = [
    {
        id: 'mainnet',
        name: 'Canopy Mainnet',
        rpcUrl: isNetlifyHost ? '/rpc-node1' : 'https://node1.canopy.us.nodefleet.net/rpc',
        adminRpcUrl: isNetlifyHost ? '/admin-node1' : 'https://node1.canopy.us.nodefleet.net/admin',
        chainId: 1,
        isTestnet: false
    },
    {
        id: 'canary',
        name: 'Canary Mainnet',
        rpcUrl: isNetlifyHost ? '/rpc-node2' : 'https://node2.canopy.us.nodefleet.net/rpc',
        adminRpcUrl: isNetlifyHost ? '/admin-node2' : 'https://node2.canopy.us.nodefleet.net/admin',
        chainId: 1,
        isTestnet: true
    }
]

interface NetworkSelectorProps {
    fullWidth?: boolean
}

const buttonClass =
    'inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-white/15 px-3 text-sm font-medium text-zinc-400 transition-colors hover:bg-white/5 hover:text-white'

const NetworkSelector: React.FC<NetworkSelectorProps> = ({ fullWidth = false }) => {
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
        <div className="relative max-w-full" ref={dropdownRef}>
            <button
                onClick={() => setIsOpen(!isOpen)}
                className={`${buttonClass} ${fullWidth ? 'w-full' : 'max-w-full'}`}
            >
                <div className="flex min-w-0 flex-1 items-center space-x-2 overflow-hidden">
                    <div className={`w-2 h-2 rounded-full flex-shrink-0 ${selectedNetwork.isTestnet ? 'bg-yellow-400' : 'bg-green-400'}`} />
                    <span className="truncate whitespace-nowrap overflow-hidden text-ellipsis text-sm">{selectedNetwork.name}</span>
                </div>
                <motion.div
                    animate={{ rotate: isOpen ? 180 : 0 }}
                    transition={{ type: 'spring', stiffness: 300, damping: 20 }}
                >
                    <ChevronDown className="h-4 w-4 flex-shrink-0" />
                </motion.div>
            </button>

            <AnimatePresence>
                {isOpen && (
                    <motion.div
                        initial={{ opacity: 0, y: -8, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -6, scale: 0.98 }}
                        transition={{ duration: 0.18, ease: 'easeOut' }}
                        className={`absolute z-50 mt-2 overflow-hidden rounded-lg border border-white/10 bg-card shadow-2xl ${fullWidth ? 'left-0 right-0 min-w-full' : 'right-0 min-w-[220px]'}`}
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
                                    className={`flex w-full items-center space-x-3 px-3 py-2 text-left text-sm font-normal transition-colors duration-200 ${selectedNetwork.id === network.id
                                        ? 'bg-primary/10 text-primary'
                                        : 'text-gray-300 hover:bg-white/10 hover:text-white'
                                        }`}
                                >
                                    <div className={`w-2 h-2 rounded-full ${network.isTestnet ? 'bg-yellow-400' : 'bg-green-400'}`} />
                                    <div className="flex-1">
                                        <div className="font-medium">{network.name}</div>
                                        <div className="text-xs text-gray-500 truncate">{network.rpcUrl}</div>
                                    </div>
                                    {selectedNetwork.id === network.id && (
                                        <motion.div
                                            initial={{ scale: 0 }}
                                            animate={{ scale: 1 }}
                                        >
                                            <Check className="h-4 w-4 text-primary" />
                                        </motion.div>
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
