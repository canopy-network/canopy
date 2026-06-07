import React from 'react'
import { motion } from 'framer-motion'
import DexBatchesTable from './DexBatchesTable'
import type { DexBatchRow } from './DexBatchesTable'
import { useDexBatch, useNextDexBatch } from '../../hooks/useApi'
import dexTexts from '../../data/dex.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'

// Backend treats chainId 0 as a wildcard that returns batches for all committees
const ALL_COMMITTEES = 0

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
    const { data: lockedData, isLoading: lockedLoading } = useDexBatch(ALL_COMMITTEES)
    const { data: nextData, isLoading: nextLoading } = useNextDexBatch(ALL_COMMITTEES)

    const isLoading = lockedLoading || nextLoading

    const apiRows: DexBatchRow[] = isLoading
        ? []
        : [...toRows(lockedData, 'Locked'), ...toRows(nextData, 'Next')]
    const rows: DexBatchRow[] = apiRows
    const lockedRows = React.useMemo(() => rows.filter((row) => row.batchType === 'Locked'), [rows])
    const nextRows = React.useMemo(() => rows.filter((row) => row.batchType === 'Next'), [rows])
    const overviewCards = React.useMemo(() => {
        const committees = new Set(rows.map((row) => row.committee))
        const totalOrders = rows.reduce((sum, row) => sum + row.orders, 0)

        return [
            {
                title: 'Locked Batches',
                value: lockedRows.length.toLocaleString(),
                subValue: 'Processing now',
                icon: 'fa-solid fa-lock',
            },
            {
                title: 'Next Batches',
                value: nextRows.length.toLocaleString(),
                subValue: 'Queued next',
                icon: 'fa-solid fa-forward-step',
            },
            {
                title: 'Committees',
                value: committees.size.toLocaleString(),
                subValue: 'Active committees',
                icon: 'fa-solid fa-diagram-project',
            },
            {
                title: 'Orders',
                value: totalOrders.toLocaleString(),
                subValue: 'Visible orders',
                icon: 'fa-solid fa-book',
            },
        ]
    }, [rows])

    if (isLoading) {
        return (
            <div className="w-full">
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
            className="w-full"
        >
            <div className="mb-4">
                <h1 className="explorer-page-title">
                    {dexTexts.page.title}
                </h1>
                <p className="explorer-page-subtitle">
                    {dexTexts.page.description}
                </p>
            </div>

            <ExplorerOverviewCards cards={overviewCards} className="mb-8" />

            <div className="space-y-8">
                <DexBatchesTable title="Locked Batches" rows={lockedRows} loading={isLoading} />
                <DexBatchesTable title="Next Batches" rows={nextRows} loading={isLoading} />
            </div>
        </motion.div>
    )
}

export default DexBatchesPage
