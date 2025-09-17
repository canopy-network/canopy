import React from 'react'
import TableCard from './TableCard'
import { useTransactions, useValidators } from '../../hooks/useApi'
import Logo from '../Logo'

const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

const normalizeList = (payload: any) => {
    if (!payload) return [] as any[]
    if (Array.isArray(payload)) return payload
    const found = payload.results || payload.list || payload.data || payload.validators || payload.transactions
    return Array.isArray(found) ? found : []
}

const ExtraTables: React.FC = () => {
    const { data: validatorsPage } = useValidators(1)
    const { data: txsPage } = useTransactions(1, 0)

    const validators = normalizeList(validatorsPage)
    const txs = normalizeList(txsPage)

    const totalStake = React.useMemo(() => validators.reduce((sum: number, v: any) => sum + Number(v.stakedAmount || 0), 0), [validators])
    const validatorRows: Array<React.ReactNode[]> = React.useMemo(() => {
        return validators.map((v: any, idx: number) => {
            const address = v.address || 'N/A'
            const stake = Number(v.stakedAmount ?? 0)
            const chainsStaked = Array.isArray(v.committees) ? v.committees.length : (Number(v.committees) || 0)
            const powerPct = totalStake > 0 ? (stake / totalStake) * 100 : 0
            const clampedPct = Math.max(0, Math.min(100, powerPct))
            return [
                <span className="text-gray-400">{idx + 1}</span>,
                <div className="flex items-center gap-2">
                    <div className="h-6 w-6 rounded-full bg-green-300/10 flex items-center justify-center text-xs text-primary">
                        {(String(address)[0] || 'V').toUpperCase()}
                    </div>
                    <span>{truncate(String(address), 16)}</span>
                </div>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-200">{chainsStaked || 'N/A'}</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-300">N/A</span>,
                <span className="text-gray-200">{stake ? stake.toLocaleString() : 'N/A'}</span>,
                <div className="flex items-center gap-2">
                    <div className="w-24 sm:w-32 h-3 bg-gray-700/60 rounded-full overflow-hidden">
                        <div className="h-3 bg-primary transition-[width] duration-500 ease-out" style={{ width: `${clampedPct}%` }}></div>
                    </div>
                    <i className="fa-solid fa-bolt text-primary/80 text-xs"></i>
                </div>,
            ]
        })
    }, [validators, totalStake])

    return (
        <div className="grid grid-cols-1 gap-6">
            <TableCard
                title="Validator Ranking"
                live={false}
                viewAllPath="/validators"
                paginate
                pageSize={10}
                columns={[
                    { label: 'Rank' },
                    { label: 'Name/Address' },
                    { label: 'Rewards %' },
                    { label: 'Chains Staked' },
                    { label: '24h change' },
                    { label: 'Blocks Produced' },
                    { label: 'Total Weight' },
                    { label: 'Weight Δ' },
                    { label: 'Total Stake' },
                    { label: 'Staking Power' },
                ]}
                rows={validatorRows}
            />

            <TableCard
                title="Recent Transactions"
                live
                columns={[
                    { label: 'Time' },
                    { label: 'Action' },
                    { label: 'Chain' },
                    { label: 'From' },
                    { label: 'To' },
                    { label: 'Amount' },
                    { label: 'Hash' },
                ]}
                paginate
                pageSize={10}
                rows={txs.map((t: any) => {
                    const ts = t.time || t.timestamp || t.blockTime
                    const mins = ts ? Math.floor((Date.now() - (Number(ts) / 1000)) / 60000) : null
                    const ago = mins != null && isFinite(mins) ? `${mins} min ago` : 'N/A'
                    const action = t.messageType || t.type || 'Transfer'
                    const chain = t.chain || 'Canopy'
                    const from = t.sender || t.from || 'N/A'
                    const to = t.recipient || t.to || 'N/A'
                    const amountRaw = t.amount ?? t.value ?? t.fee
                    const amount = (amountRaw != null && amountRaw !== '') ? amountRaw : 'N/A'
                    const hash = t.txHash || t.hash || 'N/A'
                    return [
                        <span className="text-gray-400">{ago}</span>,
                        <span className="bg-green-300/10 text-primary rounded-full px-2 py-1 text-xs">{action || 'N/A'}</span>,
                        <div className="flex items-center gap-2"><Logo size={16} className="rounded-full bg-primary" /><span>{String(chain)}</span></div>,
                        <span>{truncate(String(from))}</span>,
                        <span>{truncate(String(to))}</span>,
                        <span className="text-primary">{amount}</span>,
                        <span className="text-gray-400">{truncate(String(hash))}</span>,
                    ]
                })}
            />
        </div>
    )
}

export default ExtraTables


