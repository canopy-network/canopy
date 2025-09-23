import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import ValidatorsFilters from './ValidatorsFilters'
import ValidatorsTable from './ValidatorsTable'
import { useValidators, useBlocks } from '../../hooks/useApi'

interface Validator {
    rank: number
    address: string
    name: string // Nombre del validator (simulado)
    publicKey: string
    committees: number[]
    netAddress: string
    stakedAmount: number
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    // Campos calculados/derivados REALES
    chainsRestaked: number
    blocksProduced: number
    stakeWeight: number
    // Campos simulados (no disponibles en la API)
    reward24h: number
    rewardChange: number
    weightChange: number
    stakingPower: number
}

const ValidatorsPage: React.FC = () => {
    const [validators, setValidators] = useState<Validator[]>([])
    const [loading, setLoading] = useState(true)
    const [currentPage, setCurrentPage] = useState(1)

    // Hook to get validators data with pagination
    const { data: validatorsData, isLoading } = useValidators(currentPage)

    // Hook to get blocks data to calculate blocks produced
    const { data: blocksData } = useBlocks(1)

    // Function to get validator name from API
    const getValidatorName = (validator: any): string => {
        // Use netAddress as main name (more readable)
        if (validator.netAddress && validator.netAddress !== 'N/A') {
            return validator.netAddress
        }

        // Fallback to address if no netAddress
        if (validator.address && validator.address !== 'N/A') {
            return validator.address
        }

        return 'Unknown Validator'
    }

    // Function to count blocks produced by validator
    const countBlocksByValidator = (validatorAddress: string, blocks: any[]) => {
        if (!blocks || !Array.isArray(blocks)) return 0
        return blocks.filter((block: any) => {
            const blockHeader = block.blockHeader || block
            return blockHeader.proposerAddress === validatorAddress
        }).length
    }

    // Normalizar datos de validators
    const normalizeValidators = (payload: any, blocks: any[]): Validator[] => {
        if (!payload) return []

        // La estructura real es: { results: [...], totalCount: number }
        const validatorsList = payload.results || payload.validators || payload.list || payload.data || payload
        if (!Array.isArray(validatorsList)) return []

        // Calcular el total de stake para calcular porcentajes
        const totalStake = validatorsList.reduce((sum: number, validator: any) =>
            sum + (validator.stakedAmount || 0), 0)

        return validatorsList.map((validator: any, index: number) => {
            // Extraer datos del validator - REVISAR TODOS LOS CAMPOS POSIBLES
            const rank = index + 1
            const address = validator.address || 'N/A'

            // Obtener nombre del validator desde la API
            const name = getValidatorName(validator)

            const publicKey = validator.publicKey || 'N/A'
            const committees = validator.committees || []
            const netAddress = validator.netAddress || 'N/A'
            const stakedAmount = validator.stakedAmount || 0
            const maxPausedHeight = validator.maxPausedHeight || 0
            const unstakingHeight = validator.unstakingHeight || 0
            const output = validator.output || 'N/A'
            const delegate = validator.delegate || false
            const compound = validator.compound || false

            // Calcular campos derivados REALES
            const stakeWeight = totalStake > 0 ? (stakedAmount / totalStake) * 100 : 0
            const chainsRestaked = committees.length
            const blocksProduced = countBlocksByValidator(address, blocks) // REAL: contando bloques

            // Campos simulados (no disponibles en la API)
            const reward24h = Math.random() * 50 + 10 // Simulado 10-60%
            const rewardChange = (Math.random() - 0.5) * 20 // Simulado -10% a +10%
            const weightChange = (Math.random() - 0.5) * 10 // Simulado -5% a +5%
            const stakingPower = Math.min(stakeWeight * 2.5, 100) // Calculado basado en stake weight

            return {
                rank,
                address,
                name, // Nombre real de la API o generado
                publicKey,
                committees,
                netAddress,
                stakedAmount,
                maxPausedHeight,
                unstakingHeight,
                output,
                delegate,
                compound,
                reward24h: Math.round(reward24h * 10) / 10, // Simulado
                rewardChange: Math.round(rewardChange * 100) / 100, // Simulado
                chainsRestaked, // REAL
                blocksProduced, // REAL
                stakeWeight: Math.round(stakeWeight * 100) / 100, // REAL
                weightChange: Math.round(weightChange * 100) / 100, // Simulado
                stakingPower: Math.round(stakingPower * 100) / 100 // Calculado
            }
        })
    }

    // Efecto para actualizar validators cuando cambian los datos
    useEffect(() => {
        if (validatorsData && blocksData) {
            const blocksList = blocksData.results || blocksData.blocks || blocksData.list || blocksData.data || blocksData
            const normalizedValidators = normalizeValidators(validatorsData, Array.isArray(blocksList) ? blocksList : [])
            setValidators(normalizedValidators)
            setLoading(false)
        }
    }, [validatorsData, blocksData])

    // Effect to update dynamic data every second
    useEffect(() => {
        const interval = setInterval(() => {
            setValidators((prevValidators) =>
                prevValidators.map((validator) => {
                    // Simular cambios en reward y weight
                    const newRewardChange = (Math.random() - 0.5) * 20
                    const newWeightChange = (Math.random() - 0.5) * 10

                    return {
                        ...validator,
                        rewardChange: Math.round(newRewardChange * 100) / 100,
                        weightChange: Math.round(newWeightChange * 100) / 100
                    }
                })
            )
        }, 5000) // Actualizar cada 5 segundos

        return () => clearInterval(interval)
    }, [])

    const totalValidators = validatorsData?.totalCount || 0

    const handlePageChange = (page: number) => {
        setCurrentPage(page)
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: "easeInOut" }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10"
        >
            <ValidatorsFilters
                totalValidators={totalValidators}
            />

            <ValidatorsTable
                validators={validators}
                loading={loading || isLoading}
                totalCount={totalValidators}
                currentPage={currentPage}
                onPageChange={handlePageChange}
            />
        </motion.div>
    )
}

export default ValidatorsPage
