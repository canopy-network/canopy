import React from 'react'
import { Link } from 'react-router-dom'
import validatorsTexts from '../../data/validators.json'
import AnimatedNumber from '../AnimatedNumber'
import TableCard from '../Home/TableCard'
import { formatPaginationRange, toCNPY } from '../../lib/utils'
import PageSizeSelect from '../shared/PageSizeSelect'

interface Validator {
    rank: number
    address: string
    name: string
    publicKey: string
    committees: number[]
    netAddress: string
    stakedAmount: number
    maxPausedHeight: number
    unstakingHeight: number
    output: string
    delegate: boolean
    compound: boolean
    chainsRestaked: number
    stakeWeight: number
    isActive: boolean
    isPaused: boolean
    isUnstaking: boolean
    activityScore: string
    estimatedRewardRate: number
    stakingPower: number
}

interface ValidatorsTableProps {
    validators: Validator[]
    loading?: boolean
    totalCount?: number
    currentPage?: number
    pageSize?: number
    onPageChange?: (page: number) => void
    onPageSizeChange?: (value: number) => void
    pageTitle?: string
    variant?: 'default' | 'accounts'
    headerActions?: React.ReactNode
    showRank?: boolean
    useCnpyBadge?: boolean
    stakingPowerAsText?: boolean
    showLiveIndicator?: boolean
    showTitle?: boolean
}

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4'
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4'

const CNPY_GRADIENTS = [
    'linear-gradient(135deg, #45ca46 0%, #2f8f36 100%)',
    'linear-gradient(135deg, #36cfc9 0%, #1677ff 100%)',
    'linear-gradient(135deg, #faad14 0%, #d46b08 100%)',
    'linear-gradient(135deg, #9254de 0%, #531dab 100%)',
    'linear-gradient(135deg, #f759ab 0%, #cf1322 100%)',
]

const getCnpyGradient = (seed: string) => {
    const total = Array.from(seed).reduce((sum, char) => sum + char.charCodeAt(0), 0)
    return CNPY_GRADIENTS[total % CNPY_GRADIENTS.length]
}

const CnpyBadge: React.FC<{ seed: string }> = ({ seed }) => (
    <div
        className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full p-[3px]"
        style={{ background: getCnpyGradient(seed) }}
    >
        <img src="/canopy-symbol-white.png" alt="" className="h-full w-full object-contain" />
    </div>
)

const LiveIndicator = () => (
    <div className="relative inline-flex items-center gap-1.5 rounded-full bg-[#35cd48]/5 px-4 py-1">
        <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-[#35cd48] shadow-[0_0_4px_rgba(53,205,72,0.8)]" />
        <span className="text-sm font-medium text-[#35cd48]">Live</span>
    </div>
)

const ValidatorsTable: React.FC<ValidatorsTableProps> = ({
    validators,
    loading = false,
    totalCount = 0,
    currentPage = 1,
    pageSize = 10,
    onPageChange,
    onPageSizeChange,
    pageTitle,
    variant = 'default',
    headerActions,
    showRank = true,
    useCnpyBadge = false,
    stakingPowerAsText = false,
    showLiveIndicator = true,
    showTitle = true,
}) => {
    const truncateMiddle = (value: string, leading = 14, trailing = 10) => {
        if (!value || value.length <= leading + trailing + 1) return value || 'N/A'
        return `${value.slice(0, leading)}…${value.slice(-trailing)}`
    }

    const formatActivityScore = (score: string) => {
        const colors = {
            Active: 'bg-primary/20 text-primary',
            Standby: 'bg-yellow-500/20 text-yellow-400',
            Paused: 'bg-orange-500/20 text-orange-400',
            Unstaking: 'bg-red-500/20 text-red-400',
            Delegate: 'bg-blue-500/20 text-blue-400',
            Inactive: 'bg-gray-500/20 text-gray-400',
        }

        const colorClass = colors[score as keyof typeof colors] || colors.Inactive

        return (
            <span className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${colorClass}`}>
                {score}
            </span>
        )
    }

    const getValidatorIcon = (address: string) => {
        let hash = 0
        for (let i = 0; i < address.length; i++) {
            const char = address.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash &= hash
        }

        const icons = [
            'fa-solid fa-leaf',
            'fa-solid fa-tree',
            'fa-solid fa-seedling',
            'fa-solid fa-mountain',
            'fa-solid fa-sun',
            'fa-solid fa-moon',
            'fa-solid fa-star',
            'fa-solid fa-heart',
            'fa-solid fa-gem',
            'fa-solid fa-crown',
            'fa-solid fa-shield',
            'fa-solid fa-key',
            'fa-solid fa-lock',
            'fa-solid fa-unlock',
            'fa-solid fa-bolt',
            'fa-solid fa-fire',
            'fa-solid fa-water',
            'fa-solid fa-wind',
            'fa-solid fa-snowflake',
            'fa-solid fa-cloud',
        ]

        return icons[Math.abs(hash) % icons.length]
    }

    const maxStakeAmount = validators.length > 0
        ? Math.max(...validators.map((validator) => Number(validator.stakedAmount || 0)), 1)
        : 1

    const baseColumns = [
        { key: 'rank', label: 'Rank', widthWithRank: 'w-[6%]', widthWithoutRank: '' },
        { key: 'address', label: 'Validator Name/Address', widthWithRank: 'w-[22%]', widthWithoutRank: 'w-[28%]' },
        { key: 'reward', label: 'Reward % (24h)', widthWithRank: 'w-[11%]', widthWithoutRank: 'w-[12%]' },
        { key: 'status', label: 'Status', widthWithRank: 'w-[14%]', widthWithoutRank: 'w-[14%]' },
        { key: 'chains', label: 'Chains Restaked', widthWithRank: 'w-[10%]', widthWithoutRank: 'w-[10%]' },
        { key: 'weight', label: 'Stake Weight', widthWithRank: 'w-[12%]', widthWithoutRank: 'w-[12%]' },
        { key: 'stake', label: 'Total Stake (CNPY)', widthWithRank: 'w-[15%]', widthWithoutRank: 'w-[16%]' },
        { key: 'power', label: 'Staking Power', widthWithRank: 'w-[10%]', widthWithoutRank: 'w-[8%]' },
    ]

    const columns = baseColumns
        .filter((column) => showRank || column.key !== 'rank')
        .map((column) => ({
            label: column.label,
            width: showRank ? column.widthWithRank : column.widthWithoutRank,
        }))

    const rows = validators.map((validator) => {
        const identifier = validator.netAddress && validator.netAddress !== 'tcp://delegating' && validator.netAddress !== 'N/A'
            ? validator.netAddress
            : validator.address

        const cells: React.ReactNode[] = []

        if (showRank) {
            cells.push(
                <span className="text-sm font-medium text-white tabular-nums">
                    <AnimatedNumber value={validator.rank} className="text-white" />
                </span>
            )
        }

        cells.push(
            <Link
                to={`/validator/${validator.address}?rank=${validator.rank}`}
                className="flex max-w-[18rem] items-center gap-2 overflow-hidden text-sm font-medium text-white transition-colors hover:text-primary"
                title={identifier}
            >
                {useCnpyBadge ? (
                    <CnpyBadge seed={validator.address} />
                ) : (
                    <span className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-green-300/10 text-primary">
                        <i className={`${getValidatorIcon(validator.address)} text-xs`} />
                    </span>
                )}
                <span className="overflow-hidden text-ellipsis whitespace-nowrap">
                    {truncateMiddle(identifier)}
                </span>
            </Link>
        )

        cells.push(
            <span className="text-sm font-medium text-primary">
                <AnimatedNumber
                    value={validator.estimatedRewardRate}
                    format={{ maximumFractionDigits: 2 }}
                    suffix="%"
                    className="text-primary"
                />
            </span>
        )

        cells.push(formatActivityScore(validator.activityScore))

        cells.push(
            <span className="text-sm text-white tabular-nums">
                <AnimatedNumber value={validator.chainsRestaked} className="text-white" />
            </span>
        )

        cells.push(
            <span className="text-sm text-white tabular-nums">
                <AnimatedNumber
                    value={validator.stakeWeight}
                    format={{ maximumFractionDigits: 2 }}
                    suffix="%"
                    className="text-white"
                />
            </span>
        )

        cells.push(
            <span className="text-sm text-white tabular-nums">
                <AnimatedNumber
                    value={toCNPY(validator.stakedAmount)}
                    format={{ minimumFractionDigits: 2, maximumFractionDigits: 2 }}
                    className="text-white"
                />
                <span className="ml-1 text-white/50">CNPY</span>
            </span>
        )

        cells.push(
            stakingPowerAsText ? (
                <span className="text-sm text-white tabular-nums">
                    <AnimatedNumber
                        value={validator.stakingPower}
                        format={{ maximumFractionDigits: 2 }}
                        suffix="%"
                        className="text-white"
                    />
                </span>
            ) : (
                <div className="w-full rounded-full bg-white/10">
                    <div
                        className="h-2.5 rounded-full bg-primary transition-all duration-300"
                        style={{
                            width: `${Math.max(0, Math.min(100, (Number(validator.stakedAmount || 0) / maxStakeAmount) * 100))}%`,
                        }}
                    />
                </div>
            )
        )

        return cells
    })

    const totalPages = Math.max(1, Math.ceil(totalCount / pageSize))
    const startIdx = totalCount === 0 ? 0 : (currentPage - 1) * pageSize + 1
    const endIdx = Math.min(currentPage * pageSize, totalCount)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(pageSet)
            .filter((page) => page >= 1 && page <= totalPages)
            .sort((a, b) => a - b)
    }, [currentPage, totalPages])

    const goToPage = (page: number) => {
        if (!onPageChange) return
        onPageChange(Math.min(Math.max(1, page), totalPages))
    }

    if (variant === 'accounts') {
        const hasHeader = showTitle || Boolean(headerActions) || showLiveIndicator

        return (
            <div className="rounded-xl border border-white/10 bg-card p-5">
                {hasHeader && (
                    <div className="mb-5 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                        <div className="flex items-center gap-3">
                            {showTitle && (
                                <h2 className="wallet-card-title tracking-tight">
                                    {pageTitle || validatorsTexts.page.title}
                                    {loading && <i className="fa-solid fa-circle-notch fa-spin pl-2 text-sm text-white/40" aria-hidden="true"></i>}
                                </h2>
                            )}
                        </div>

                        <div className="flex flex-wrap items-center gap-3">
                            {headerActions}
                            {showLiveIndicator && <LiveIndicator />}
                        </div>
                    </div>
                )}

                <div className="overflow-x-auto">
                    <table
                        className="w-full min-w-[1180px]"
                        style={{ tableLayout: 'fixed', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                    >
                        <thead>
                            <tr>
                                {columns.map((column) => (
                                    <th key={String(column.label)} className={`${desktopHeaderClass} ${column.width || ''}`}>
                                        {column.label}
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {loading ? (
                                Array.from({ length: pageSize }).map((_, index) => (
                                    <tr key={`skeleton-${index}`} className="group animate-pulse">
                                        {columns.map((column, columnIndex) => (
                                            <td
                                                key={`${index}-${columnIndex}`}
                                                className={`${desktopRowCellClass} ${column.width || ''}`}
                                                style={{
                                                    borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                    borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                    borderTopRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                    borderBottomRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                }}
                                            >
                                                <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-28' : columnIndex === columns.length - 1 ? 'w-16' : 'w-20'}`} />
                                            </td>
                                        ))}
                                    </tr>
                                ))
                            ) : rows.length === 0 ? (
                                <tr>
                                    <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                        No validators found
                                    </td>
                                </tr>
                            ) : (
                                rows.map((cells, rowIndex) => (
                                    <tr key={validators[rowIndex]?.address || rowIndex} className="group">
                                        {cells.map((cell, cellIndex) => (
                                            <td
                                                key={`${validators[rowIndex]?.address || rowIndex}-${cellIndex}`}
                                                className={`${desktopRowCellClass} ${columns[cellIndex]?.width || ''}`}
                                                style={{
                                                    borderTopLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                                    borderBottomLeftRadius: cellIndex === 0 ? '10px' : undefined,
                                                    borderTopRightRadius: cellIndex === cells.length - 1 ? '10px' : undefined,
                                                    borderBottomRightRadius: cellIndex === cells.length - 1 ? '10px' : undefined,
                                                }}
                                            >
                                                {cell}
                                            </td>
                                        ))}
                                    </tr>
                                ))
                            )}
                        </tbody>
                    </table>
                </div>

                {!loading && totalCount > 0 && (
                    <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
                        <div className="flex items-center gap-3">
                            {formatPaginationRange(startIdx, endIdx)} of <AnimatedNumber value={totalCount} />
                            {onPageSizeChange && (
                                <PageSizeSelect value={pageSize} onChange={onPageSizeChange} />
                            )}
                        </div>

                        <div className="flex items-center gap-2">
                            <button
                                type="button"
                                onClick={() => goToPage(currentPage - 1)}
                                disabled={currentPage === 1}
                                className="explorer-pagination-button px-3 py-1.5"
                                aria-label="Previous page"
                            >
                                <i className="fa-solid fa-angle-left" />
                            </button>

                            {visiblePages.map((page, index, arr) => {
                                const prevPage = arr[index - 1]
                                const showDots = index > 0 && page - (prevPage || 0) > 1

                                return (
                                    <React.Fragment key={page}>
                                        {showDots && <span className="px-1 text-white/40">…</span>}
                                        <button
                                            type="button"
                                            onClick={() => goToPage(page)}
                                            className={`explorer-pagination-button explorer-pagination-page px-3 py-1.5 ${
                                                currentPage === page ? 'explorer-pagination-page-active' : ''
                                            }`}
                                        >
                                            {page}
                                        </button>
                                    </React.Fragment>
                                )
                            })}

                            <button
                                type="button"
                                onClick={() => goToPage(currentPage + 1)}
                                disabled={currentPage === totalPages}
                                className="explorer-pagination-button px-3 py-1.5"
                                aria-label="Next page"
                            >
                                <i className="fa-solid fa-angle-right" />
                            </button>
                        </div>
                    </div>
                )}
            </div>
        )
    }

    return (
        <TableCard
            title={pageTitle || validatorsTexts.page.title}
            live={showLiveIndicator}
            columns={columns}
            rows={rows}
            loading={loading}
            paginate={true}
            totalCount={totalCount}
            currentPage={currentPage}
            onPageChange={onPageChange}
            showEntriesSelector={!!onPageSizeChange}
            currentEntriesPerPage={pageSize}
            onEntriesPerPageChange={onPageSizeChange}
            spacing={4}
        />
    )
}

export default ValidatorsTable
