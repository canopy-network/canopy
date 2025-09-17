import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import BlocksFilters from './BlocksFilters'
import BlocksTable from './BlocksTable'
import { useBlocks } from '../../hooks/useApi'
import blocksTexts from '../../data/blocks.json'

interface Block {
    height: number
    timestamp: string
    age: string
    hash: string
    producer: string
    transactions: number
    gasPrice: number
    blockTime: number
}

const BlocksPage: React.FC = () => {
    const [activeFilter, setActiveFilter] = useState('all')
    const [blocks, setBlocks] = useState<Block[]>([])
    const [loading, setLoading] = useState(true)

    // Hook para obtener datos de bloques
    const { data: blocksData, isLoading } = useBlocks(1)

    // Normalizar datos de bloques
    const normalizeBlocks = (payload: any): Block[] => {
        if (!payload) return []

        // La estructura real es: { results: [...], totalCount: number }
        const blocksList = payload.results || payload.blocks || payload.list || payload.data || payload
        if (!Array.isArray(blocksList)) return []

        return blocksList.map((block: any) => {
            // Extraer datos del blockHeader
            const blockHeader = block.blockHeader || block
            const height = blockHeader.height || 0
            const timestamp = blockHeader.time || blockHeader.timestamp
            const hash = blockHeader.hash || 'N/A'
            const producer = blockHeader.proposerAddress || 'N/A'
            const transactions = blockHeader.numTxs || block.transactions?.length || 0
            const gasPrice = 0.025 // Valor por defecto ya que no está en los datos
            const blockTime = 6.2 // Valor por defecto

            // Calcular edad
            let age = 'N/A'
            if (timestamp) {
                const now = Date.now()
                // El timestamp viene en microsegundos, convertir a milisegundos
                const blockTimeMs = typeof timestamp === 'number' ?
                    (timestamp > 1e12 ? timestamp / 1000 : timestamp) :
                    new Date(timestamp).getTime()

                const diffMs = now - blockTimeMs
                const diffSecs = Math.floor(diffMs / 1000)
                const diffMins = Math.floor(diffSecs / 60)
                const diffHours = Math.floor(diffMins / 60)

                if (diffSecs < 60) {
                    age = `${diffSecs} ${blocksTexts.table.units.secsAgo}`
                } else if (diffMins < 60) {
                    age = `${diffMins} ${blocksTexts.table.units.minAgo}`
                } else {
                    age = `${diffHours} ${blocksTexts.table.units.hoursAgo}`
                }
            }

            return {
                height,
                timestamp: timestamp ? new Date(timestamp / 1000).toISOString() : 'N/A',
                age,
                hash,
                producer,
                transactions,
                gasPrice,
                blockTime
            }
        })
    }

    // Efecto para actualizar bloques cuando cambian los datos
    useEffect(() => {
        if (blocksData) {
            const normalizedBlocks = normalizeBlocks(blocksData)
            setBlocks(normalizedBlocks)
            setLoading(false)
        }
    }, [blocksData])

    // Efecto para simular actualización en tiempo real
    useEffect(() => {
        const interval = setInterval(() => {
            setBlocks(prevBlocks =>
                prevBlocks.map(block => {
                    const now = Date.now()
                    const blockTime = new Date(block.timestamp).getTime()
                    const diffMs = now - blockTime
                    const diffSecs = Math.floor(diffMs / 1000)
                    const diffMins = Math.floor(diffSecs / 60)
                    const diffHours = Math.floor(diffMins / 60)

                    let newAge = 'N/A'
                    if (diffSecs < 60) {
                        newAge = `${diffSecs} ${blocksTexts.table.units.secsAgo}`
                    } else if (diffMins < 60) {
                        newAge = `${diffMins} ${blocksTexts.table.units.minAgo}`
                    } else {
                        newAge = `${diffHours} ${blocksTexts.table.units.hoursAgo}`
                    }

                    return { ...block, age: newAge }
                })
            )
        }, 1000)

        return () => clearInterval(interval)
    }, [])

    const totalBlocks = blocksData?.totalCount || 0

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <BlocksFilters
                activeFilter={activeFilter}
                onFilterChange={setActiveFilter}
                totalBlocks={totalBlocks}
            />

            <BlocksTable
                blocks={blocks}
                loading={loading || isLoading}
            />
        </motion.div>
    )
}

export default BlocksPage
