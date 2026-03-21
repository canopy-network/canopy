import React from 'react'
import { motion } from 'framer-motion'
import DexBatchesTable from './DexBatchesTable'
import type { DexBatchRow } from './DexBatchesTable'
import { useDexBatch, useNextDexBatch } from '../../hooks/useApi'
import dexTexts from '../../data/dex.json'

const normalizeBatch = (data: any, batchType: 'Locked' | 'Next'): DexBatchRow => {
    const receiptHash = data?.receiptHash
    return {
        batchType,
        committee: data?.committee ?? 0,
        receiptHash: receiptHash ? String(receiptHash) : 'N/A',
        orders: data?.orders?.length ?? 0,
        deposits: data?.deposits?.length ?? 0,
        withdraws: data?.withdrawals?.length ?? 0,
        poolSize: data?.poolSize ?? 0,
        totalPoolPoints: data?.totalPoolPoints ?? 0,
        lockedHeight: data?.lockedHeight ?? 0,
        receipts: data?.receipts?.length ?? 0,
    }
}

const toRows = (data: any, batchType: 'Locked' | 'Next'): DexBatchRow[] => {
    if (!data) return []
    const list = Array.isArray(data) ? data : [data]
    return list.map((item: any) => normalizeBatch(item, batchType))
}

const DexBatchesPage: React.FC = () => {
    const { data: lockedData, isLoading: lockedLoading } = useDexBatch(0)
    const { data: nextData, isLoading: nextLoading } = useNextDexBatch(0)

    const isLoading = lockedLoading || nextLoading

    const rows: DexBatchRow[] = isLoading
        ? []
        : [...toRows(lockedData, 'Locked'), ...toRows(nextData, 'Next')]

    if (isLoading) {
        return (
            <div className="mx-auto px-4 sm:px-6 lg:px-8 py-10">
                <div className="animate-pulse">
                    <div className="h-8 bg-gray-700/50 rounded w-1/4 mb-6"></div>
                    <div className="h-40 bg-gray-700/50 rounded"></div>
                </div>
            </div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3, ease: 'easeInOut' }}
            className="mx-auto px-4 sm:px-6 lg:px-8 py-10 max-w-[100rem]"
        >
            <div className="mb-6">
                <h1 className="text-2xl font-bold text-white mb-2">
                    {dexTexts.page.title}
                </h1>
                <p className="text-gray-400">
                    {dexTexts.page.description}
                </p>
            </div>

            <DexBatchesTable rows={rows} loading={isLoading} />
        </motion.div>
    )
}

export default DexBatchesPage
