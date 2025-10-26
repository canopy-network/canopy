import React from 'react'
import { motion } from 'framer-motion'

interface FeeTrendsProps {
    fromBlock: string
    toBlock: string
    loading: boolean
    paramsData: any
    transactionsData: any
    blockGroups: Array<{
        start: number
        end: number
        label: string
        blockCount: number
    }>
}

const FeeTrends: React.FC<FeeTrendsProps> = ({ fromBlock, toBlock, loading, paramsData, transactionsData, blockGroups }) => {
    // Calculate real fee data from params and transactions
    const getFeeData = () => {
        if (!paramsData?.fee || !transactionsData?.results) {
            return {
                feeRange: '0 - 0 CNPY',
                totalFees: '0 CNPY',
                avgFee: 0,
                minFee: 0,
                maxFee: 0
            }
        }

        const feeParams = paramsData.fee
        const transactions = transactionsData.results

        // Get fee parameters
        const sendFee = feeParams.sendFee || 0
        const stakeFee = feeParams.stakeFee || 0
        const editStakeFee = feeParams.editStakeFee || 0
        const pauseFee = feeParams.pauseFee || 0
        const unpauseFee = feeParams.unpauseFee || 0
        const changeParameterFee = feeParams.changeParameterFee || 0
        const daoTransferFee = feeParams.daoTransferFee || 0
        const certificateResultsFee = feeParams.certificateResultsFee || 0
        const subsidyFee = feeParams.subsidyFee || 0
        const createOrderFee = feeParams.createOrderFee || 0
        const editOrderFee = feeParams.editOrderFee || 0
        const deleteOrderFee = feeParams.deleteOrderFee || 0

        // Calculate total fees from actual transactions
        const totalFees = transactions.reduce((sum: number, tx: any) => {
            return sum + (tx.fee || 0)
        }, 0)

        // Get min and max fees from params
        const allFees = [sendFee, stakeFee, editStakeFee, pauseFee, unpauseFee, changeParameterFee, daoTransferFee, certificateResultsFee, subsidyFee, createOrderFee, editOrderFee, deleteOrderFee].filter(fee => fee > 0)
        const minFee = allFees.length > 0 ? Math.min(...allFees) : 0
        const maxFee = allFees.length > 0 ? Math.max(...allFees) : 0
        const avgFee = transactions.length > 0 ? totalFees / transactions.length : 0

        // Convert from micro denomination to CNPY
        const minFeeCNPY = minFee / 1000000
        const maxFeeCNPY = maxFee / 1000000
        const totalFeesCNPY = totalFees / 1000000

        return {
            feeRange: `${minFeeCNPY.toFixed(1)} - ${maxFeeCNPY.toFixed(1)} CNPY`,
            totalFees: `${totalFeesCNPY.toFixed(1)} CNPY`,
            avgFee: avgFee / 1000000,
            minFee: minFeeCNPY,
            maxFee: maxFeeCNPY
        }
    }

    const feeData = getFeeData()

    const getDates = () => {
        const blockRange = parseInt(toBlock) - parseInt(fromBlock) + 1
        const periods = Math.min(blockRange, 30)
        const dates: string[] = []

        for (let i = 0; i < periods; i++) {
            const blockNumber = parseInt(fromBlock) + i
            dates.push(`#${blockNumber}`)
        }
        return dates
    }

    const dateLabels = getDates()

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

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.7 }}
            className="bg-card rounded-xl p-6 border border-gray-800/30 hover:border-gray-800/50 transition-colors duration-200"
        >
            <div className="mb-4">
                <h3 className="text-lg font-semibold text-white">
                    Fee Trends
                </h3>
                <p className="text-sm text-gray-400 mt-1">
                    Average Fee Over Time
                </p>
            </div>

            {/* Real fee data display */}
            <div className="h-32 flex flex-col justify-center items-center text-center">
                <div className="text-gray-400 space-y-2">
                    <div className="text-sm">Fee Range: {feeData.feeRange}</div>
                    <div className="text-sm">Total Fees: {feeData.totalFees}</div>
                    <div className="text-sm">Avg Fee: {feeData.avgFee.toFixed(3)} CNPY</div>
                </div>
            </div>

            <div className="mt-4 text-xs text-gray-400 text-center">
                <span>{dateLabels[0]} - {dateLabels[dateLabels.length - 1]}</span>
            </div>
        </motion.div>
    )
}

export default FeeTrends