import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import TableCard from '../Home/TableCard'
import blockDetailTexts from '../../data/blockDetail.json'
import transactionsTexts from '../../data/transactions.json'
import AnimatedNumber from '../AnimatedNumber'
import { useParams as useParamsHook } from '../../hooks/useApi'

interface Transaction {
    hash: string
    type?: string
    from: string
    to: string
    value: number
    fee: number
    messageType?: string
    height?: number
    sender?: string
    txHash?: string
    status?: 'success' | 'failed' | 'pending'
    age?: string
}

interface BlockTransactionsProps {
    transactions: Transaction[]
    totalTransactions: number
}

const BlockTransactions: React.FC<BlockTransactionsProps> = ({
    transactions,
    totalTransactions
}) => {
    const navigate = useNavigate()

    // Get params to access fee information
    const { data: paramsData } = useParamsHook(0)
    const feeParams = paramsData?.fee || {}

    // Map transaction type to fee param key (directly from endpoint)
    const getFeeParamKey = (type: string): string => {
        const typeMap: Record<string, string> = {
            'send': 'sendFee',
            'stake': 'stakeFee',
            'edit-stake': 'editStakeFee',
            'editStake': 'editStakeFee',
            'unstake': 'unstakeFee',
            'pause': 'pauseFee',
            'unpause': 'unpauseFee',
            'changeParameter': 'changeParameterFee',
            'daoTransfer': 'daoTransferFee',
            'certificateResults': 'certificateResultsFee',
            'subsidy': 'subsidyFee',
            'createOrder': 'createOrderFee',
            'editOrder': 'editOrderFee',
            'deleteOrder': 'deleteOrderFee',
        }
        return typeMap[type.toLowerCase()] || 'sendFee'
    }

    // Get minimum fee for a transaction type
    const getMinimumFeeForTxType = (type: string): number => {
        const feeKey = getFeeParamKey(type)
        return feeParams[feeKey] || feeParams.sendFee || 0
    }

    const truncate = (s: string, n: number = 6) => s.length <= n ? s : `${s.slice(0, n)}…${s.slice(-4)}`

    const formatAmount = (amount: number) => {
        if (!amount || amount === 0) return 'N/A'
        return `${amount.toLocaleString()} ${transactionsTexts.table.units.cnpy}`
    }

    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    const formatFee = (fee: number) => {
        if (!fee || fee === 0) return '0 CNPY'
        // Fee comes in micro denomination from endpoint, convert to CNPY
        const cnpy = toCNPY(fee)
        return `${cnpy.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })} CNPY`
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'success':
                return 'bg-green-500/20 text-green-400'
            case 'failed':
                return 'bg-red-500/20 text-red-400'
            case 'pending':
                return 'bg-yellow-500/20 text-yellow-400'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    const getTypeIcon = (type: string) => {
        switch (type.toLowerCase()) {
            case 'send':
                return 'bi bi-send'
            case 'transfer':
                return 'bi bi-send'
            case 'stake':
                return 'bi bi-file-lock2'
            case 'edit-stake':
                return 'bi bi-file-lock2'
            case 'unstake':
                return 'fa-solid fa-unlock'
            case 'swap':
                return 'bi bi-arrow-left-right'
            case 'governance':
                return 'fa-solid fa-vote-yea'
            case 'delegate':
                return 'bi bi-file-lock2'
            case 'undelegate':
                return 'fa-solid fa-user-times'
            case 'certificateresults':
            case 'certificate':
                return 'bi bi-c-circle-fill'
            default:
                return 'fa-solid fa-circle'
        }
    }

    const getTypeColor = (type: string) => {
        switch (type.toLowerCase()) {
            case 'transfer':
                return 'bg-blue-500/20 text-blue-400'
            case 'stake':
                return 'bg-green-500/20 text-green-400'
            case 'unstake':
                return 'bg-orange-500/20 text-orange-400'
            case 'swap':
                return 'bg-purple-500/20 text-purple-400'
            case 'governance':
                return 'bg-indigo-500/20 text-indigo-400'
            case 'delegate':
                return 'bg-cyan-500/20 text-cyan-400'
            case 'undelegate':
                return 'bg-pink-500/20 text-pink-400'
            case 'certificateresults':
                return 'bg-green-500/20 text-primary'
            default:
                return 'bg-gray-500/20 text-gray-400'
        }
    }

    // Get transaction type from messageType or type
    const getTransactionType = (tx: Transaction): string => {
        return tx.type || tx.messageType || 'send'
    }

    // Prepare columns for TableCard (same as TransactionsTable) with widths
    const columns = [
        { label: transactionsTexts.table.headers.hash, width: 'w-[13%]' },
        { label: transactionsTexts.table.headers.type, width: 'w-[16%]' },
        { label: transactionsTexts.table.headers.from, width: 'w-[13%]' },
        { label: transactionsTexts.table.headers.to, width: 'w-[13%]' },
        { label: transactionsTexts.table.headers.amount, width: 'w-[8%]' },
        { label: transactionsTexts.table.headers.fee, width: 'w-[8%]' },
        { label: transactionsTexts.table.headers.status, width: 'w-[11%]' },
        { label: transactionsTexts.table.headers.age, width: 'w-[10%]' }
    ]

    // Prepare rows for TableCard with same logic as TransactionsTable
    const rows = transactions.map((tx) => {
        const txType = getTransactionType(tx)
        // Fee comes in micro denomination from endpoint (as per TransactionsTable logic)
        const feeMicro = tx.fee || 0
        const amount = tx.value || 0

        return [
            // Hash
            <span
                key="hash"
                className="font-mono text-white text-sm cursor-pointer hover:text-green-400 hover:underline"
                onClick={() => navigate(`/transaction/${tx.hash}`)}
            >
                {truncate(tx.hash, 8)}
            </span>,

            // Type
            <div
                key="type"
                className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium ${getTypeColor(txType)}`}
            >
                <i className={`${getTypeIcon(txType)} text-xs`} style={{ fontSize: '0.875rem' }}></i>
                <span>{txType}</span>
            </div>,

            // From
            <Link
                key="from"
                to={`/account/${tx.from}`}
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline"
            >
                {truncate(tx.from, 12)}
            </Link>,

            // To
            <Link
                key="to"
                to={`/account/${tx.to}`}
                className="text-gray-400 font-mono text-sm hover:text-green-400 hover:underline"
            >
                {tx.to === 'N/A' ? (
                    <span className="text-gray-500">N/A</span>
                ) : (
                    truncate(tx.to, 12)
                )}
            </Link>,

            // Amount
            <span key="amount" className="text-white text-sm font-medium">
                {typeof amount === 'number' && amount > 0 ? (
                    <>
                        <AnimatedNumber
                            value={amount}
                            format={{ maximumFractionDigits: 4 }}
                            className="text-white"
                        />&nbsp; {transactionsTexts.table.units.cnpy}
                    </>
                ) : (
                    `0 ${transactionsTexts.table.units.cnpy}`
                )}
            </span>,

            // Fee (in micro denomination from endpoint) with minimum fee info
            <div key="fee" className="flex flex-col gap-1">
                <span className="text-gray-300 text-sm">
                    {typeof feeMicro === 'number' ? (
                        formatFee(feeMicro)
                    ) : (
                        formatFee(feeMicro || 0)
                    )}
                </span>
            </div>,

            // Status
            <div
                key="status"
                className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(tx.status || 'success')}`}
            >
                {tx.status === 'success' && <i className="fa-solid fa-check text-xs mr-1"></i>}
                {tx.status === 'failed' && <i className="fa-solid fa-times text-xs mr-1"></i>}
                {tx.status === 'pending' && <i className="fa-solid fa-clock text-xs mr-1"></i>}
                <span>{transactionsTexts.status[tx.status as keyof typeof transactionsTexts.status] || transactionsTexts.status.success}</span>
            </div>,

            // Age
            <span key="age" className="text-gray-400 text-sm">
                {tx.age || 'N/A'}
            </span>
        ]
    })

    return (
        <TableCard
            title={`${blockDetailTexts.transactions.title} (${totalTransactions})`}
            live={false}
            columns={columns}
            rows={rows}
            spacing={4}
            paginate={true}
            pageSize={10}
            totalCount={totalTransactions}
        />
    )
}

export default BlockTransactions
