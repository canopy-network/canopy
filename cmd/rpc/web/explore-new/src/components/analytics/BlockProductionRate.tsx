import React from 'react'
import { motion } from 'framer-motion'

interface BlockProductionRateProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    blocksData: any
}

const BlockProductionRate: React.FC<BlockProductionRateProps> = ({ fromBlock, toBlock, loading, blocksData }) => {
    // Use real block data to calculate production rate by time intervals (10 minutes or 1 minute)
    const getBlockData = () => {
        if (!blocksData?.results || !Array.isArray(blocksData.results) || blocksData.results.length === 0) {
            // Silently return empty array without logging errors
            return []
        }

        const realBlocks = blocksData.results
        console.log(`Total blocks available: ${realBlocks.length}`)

        const fromBlockNum = parseInt(fromBlock) || 0
        const toBlockNum = parseInt(toBlock) || 0
        console.log(`Block range: ${fromBlockNum} to ${toBlockNum}`)

        // Filter blocks by the specified range
        const filteredBlocks = realBlocks.filter((block: any) => {
            const blockHeight = block.blockHeader?.height || block.height || 0
            return blockHeight >= fromBlockNum && blockHeight <= toBlockNum
        })
        console.log(`Filtered blocks count: ${filteredBlocks.length}`)

        // If no blocks in range, return empty array
        if (filteredBlocks.length === 0) {
            console.log("No blocks in the specified range")
            return []
        }

        // Sort blocks by timestamp (oldest first)
        filteredBlocks.sort((a: any, b: any) => {
            const timeA = a.blockHeader?.time || a.time || 0
            const timeB = b.blockHeader?.time || b.time || 0
            return timeA - timeB
        })

        // Extraer los tiempos reales de los bloques y agruparlos por hora
        const blocksByHour: { [hour: string]: number } = {}

        filteredBlocks.forEach((block: any) => {
            const blockTime = block.blockHeader?.time || block.time || 0
            const blockTimeMs = blockTime > 1e12 ? blockTime / 1000 : blockTime
            const blockDate = new Date(blockTimeMs)

            // Agrupar por hora:minuto (redondeando a intervalos de 10 minutos si hay muchos bloques)
            const minute = filteredBlocks.length < 20 ?
                blockDate.getMinutes() :
                Math.floor(blockDate.getMinutes() / 10) * 10

            const hourKey = `${blockDate.getHours().toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`

            if (!blocksByHour[hourKey]) {
                blocksByHour[hourKey] = 0
            }
            blocksByHour[hourKey]++
        })

        // Convertir el objeto a un array ordenado por hora
        const timeKeys = Object.keys(blocksByHour).sort()
        const timeGroups = timeKeys.map(key => blocksByHour[key])

        console.log('Real time keys:', timeKeys)
        console.log('Blocks per time interval:', timeGroups)

        // Guardar las claves de tiempo para usarlas en las etiquetas
        // @ts-ignore - Añadir propiedad temporal para compartir con getTimeIntervalLabels
        getBlockData.timeKeys = timeKeys

        return timeGroups
    }

    const blockData = getBlockData()
    const maxValue = Math.max(...blockData, 0)
    const minValue = Math.min(...blockData, 0)

    // Get time interval labels for the x-axis
    const getTimeIntervalLabels = () => {
        // @ts-ignore - Acceder a la propiedad temporal que guardamos en getBlockData
        const timeKeys = getBlockData.timeKeys || []

        if (!timeKeys.length) {
            return []
        }

        // Para cada clave de tiempo (HH:MM), crear una etiqueta
        return timeKeys.map(key => {
            // Si hay pocos bloques (< 20), mostrar solo la hora:minuto
            if (blocksData?.results?.length < 20) {
                return key
            }

            // Si hay muchos bloques, mostrar el rango de 10 minutos
            const [hour, minute] = key.split(':').map(Number)
            const endMinute = (minute + 10) % 60
            const endHour = endMinute < minute ? (hour + 1) % 24 : hour

            return `${key}-${endHour.toString().padStart(2, '0')}:${endMinute.toString().padStart(2, '0')}`
        })
    }

    const timeIntervalLabels = getTimeIntervalLabels()

    if (loading) {
        return (
            <div className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200">
                <div className="animate-pulse">
                    <div className="h-4 bg-gray-700 rounded w-1/2 mb-4"></div>
                    <div className="h-32 bg-gray-700 rounded"></div>
                </div>
            </div>
        )
    }

    // If no real data, show empty state
    if (blockData.length === 0 || maxValue === 0) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: 0.2 }}
                className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
            >
                <div className="mb-4">
                    <h3 className="text-lg font-semibold text-white">
                        Block Production Rate
                    </h3>
                    <p className="text-sm text-gray-400 mt-1">
                        Blocks per time interval
                    </p>
                </div>
                <div className="h-32 flex items-center justify-center">
                    <p className="text-gray-500 text-sm">No block data available</p>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.2 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Block Production Rate
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Blocks per {blocksData?.results?.length < 20 ? '1-minute' : '10-minute'} interval
                </p>
            </div>

            <div className="h-32 relative">
                <svg className="w-full h-full" viewBox="0 0 300 120">
                    {/* Grid lines */}
                    <defs>
                        <pattern id="grid-blocks" width="30" height="20" patternUnits="userSpaceOnUse">
                            <path d="M 30 0 L 0 0 0 20" fill="none" stroke="#374151" strokeWidth="0.5" />
                        </pattern>
                    </defs>
                    <rect width="100%" height="100%" fill="url(#grid-blocks)" />

                    {/* Area chart */}
                    <defs>
                        <linearGradient id="blockGradient" x1="0%" y1="0%" x2="0%" y2="100%">
                            <stop offset="0%" stopColor="#4ADE80" stopOpacity="0.3" />
                            <stop offset="100%" stopColor="#4ADE80" stopOpacity="0.1" />
                        </linearGradient>
                    </defs>

                    {blockData.length > 1 && (
                        <>
                            <path
                                fill="url(#blockGradient)"
                                d={`M 10,110 ${blockData.map((value, index) => {
                                    const x = (index / (blockData.length - 1)) * 280 + 10
                                    const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                                    return `${x},${y}`
                                }).join(' ')} L 290,110 Z`}
                            />

                            {/* Line */}
                            <polyline
                                fill="none"
                                stroke="#4ADE80"
                                strokeWidth="2"
                                points={blockData.map((value, index) => {
                                    const x = (index / (blockData.length - 1)) * 280 + 10
                                    const y = 110 - ((value - minValue) / (maxValue - minValue)) * 100
                                    return `${x},${y}`
                                }).join(' ')}
                            />
                        </>
                    )}

                    {/* Single point if only one data point */}
                    {blockData.length === 1 && (
                        <circle
                            cx="150"
                            cy="55"
                            r="4"
                            fill="#4ADE80"
                        />
                    )}
                </svg>

                {/* Y-axis labels */}
                <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400">
                    <span>{maxValue.toFixed(1)}</span>
                    <span>{((maxValue + minValue) / 2).toFixed(1)}</span>
                    <span>{minValue.toFixed(1)}</span>
                </div>
            </div>

            <div className="mt-4 flex justify-between text-xs text-gray-400">
                {timeIntervalLabels.map((label: string, index: number) => (
                    <span key={index} className="text-center flex-1 px-1 truncate">{label}</span>
                ))}
            </div>
        </motion.div>
    )
}

export default BlockProductionRate