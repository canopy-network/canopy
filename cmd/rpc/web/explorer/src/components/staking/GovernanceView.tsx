import React from 'react'
import { motion } from 'framer-motion'
import { useParams } from '../../hooks/useApi'
import stakingConfig from '../../data/staking.json'
import ExplorerOverviewCards from '../ExplorerOverviewCards'
import { formatPaginationRange, toCNPY } from '../../lib/utils'
import AnimatedNumber from '../AnimatedNumber'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

interface GovernanceParam {
    paramName: string
    paramValue: string | number
    paramSpace: string
    visible: boolean
}

const desktopHeaderClass =
    'px-2 py-1.5 text-left text-[11px] font-medium capitalize tracking-wider text-white/60 whitespace-nowrap sm:px-3 lg:px-4'
const desktopRowCellClass =
    'bg-[#1a1a1a] px-2 py-2 align-middle transition-colors group-hover:bg-[#272729] sm:px-3 lg:px-4'

const GovernanceView: React.FC = () => {
    const [currentPage, setCurrentPage] = React.useState(1)
    // Get governance parameters from the /v1/query/params endpoint
    const { data: paramsData, isLoading, error } = useParams(0)

    // Function to get governance parameters from API data
    const getGovernanceParams = (): GovernanceParam[] => {
        if (!paramsData) {
            // Fallback to config if no API data
            return stakingConfig.governance.parameters.filter(param => param.visible)
        }

        const params: GovernanceParam[] = []

        // Consensus parameters
        if (paramsData.consensus) {
            if (paramsData.consensus.blockSize !== undefined) {
                params.push({
                    paramName: 'blockSize',
                    paramValue: paramsData.consensus.blockSize,
                    paramSpace: 'consensus',
                    visible: true
                })
            }
            if (paramsData.consensus.protocolVersion !== undefined) {
                params.push({
                    paramName: 'protocolVersion',
                    paramValue: paramsData.consensus.protocolVersion,
                    paramSpace: 'consensus',
                    visible: true
                })
            }
            if (paramsData.consensus.rootChainID !== undefined) {
                params.push({
                    paramName: 'rootChainID',
                    paramValue: paramsData.consensus.rootChainID,
                    paramSpace: 'consensus',
                    visible: true
                })
            }
        }

        // Validator parameters
        if (paramsData.validator) {
            const validatorParams = [
                'unstakingBlocks',
                'maxPauseBlocks',
                'doubleSignSlashPercentage',
                'nonSignSlashPercentage',
                'maxNonSign',
                'nonSignWindow',
                'maxCommittees',
                'maxCommitteeSize',
                'earlyWithdrawalPenalty',
                'delegateUnstakingBlocks',
                'minimumOrderSize',
                'stakePercentForSubsidizedCommittee',
                'maxSlashPerCommittee',
                'delegateRewardPercentage',
                'buyDeadlineBlocks',
                'lockOrderFeeMultiplier'
            ]

            validatorParams.forEach(paramName => {
                if (paramsData.validator[paramName] !== undefined) {
                    params.push({
                        paramName,
                        paramValue: paramsData.validator[paramName],
                        paramSpace: 'validator',
                        visible: true
                    })
                }
            })
        }

        // Fee parameters
        if (paramsData.fee) {
            const feeParams = [
                'sendFee',
                'stakeFee',
                'editStakeFee',
                'pauseFee',
                'unpauseFee',
                'changeParameterFee',
                'daoTransferFee',
                'certificateResultsFee',
                'subsidyFee',
                'createOrderFee',
                'editOrderFee',
                'deleteOrderFee'
            ]

            feeParams.forEach(paramName => {
                if (paramsData.fee[paramName] !== undefined) {
                    params.push({
                        paramName,
                        paramValue: paramsData.fee[paramName],
                        paramSpace: 'fee',
                        visible: true
                    })
                }
            })
        }

        // Governance parameters
        if (paramsData.governance) {
            if (paramsData.governance.daoRewardPercentage !== undefined) {
                params.push({
                    paramName: 'daoRewardPercentage',
                    paramValue: paramsData.governance.daoRewardPercentage,
                    paramSpace: 'governance',
                    visible: true
                })
            }
        }

        return params
    }

    const governanceParams = getGovernanceParams()
    const itemsPerPage = stakingConfig.governance.table.pagination.itemsPerPage || 10

    const formatParamValue = (value: string | number, paramName: string) => {
        if (typeof value === 'number') {
            // Convert fees from micro denomination to CNPY
            if (paramName.includes('Fee') || paramName === 'minimumOrderSize') {
                const cnpyValue = toCNPY(value)
                return cnpyValue.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })
            }
            // Format percentages
            if (paramName.includes('Percentage') || paramName.includes('Percent') || paramName.includes('Cut')) {
                return `${value}%`
            }
            return value.toLocaleString()
        }
        return value.toString()
    }

    // Generate rows dynamically based on JSON configuration
    const totalPages = Math.max(1, Math.ceil(governanceParams.length / itemsPerPage))
    const startIdx = governanceParams.length === 0 ? 0 : (currentPage - 1) * itemsPerPage + 1
    const endIdx = Math.min(currentPage * itemsPerPage, governanceParams.length)

    const visiblePages = React.useMemo(() => {
        if (totalPages <= 6) return Array.from({ length: totalPages }, (_, i) => i + 1)
        const pageSet = new Set([1, totalPages, currentPage - 1, currentPage, currentPage + 1])
        return Array.from(pageSet).filter((page) => page >= 1 && page <= totalPages).sort((a, b) => a - b)
    }, [currentPage, totalPages])

    const goToPage = (page: number) => {
        setCurrentPage(Math.min(Math.max(1, page), totalPages))
    }

    const paginatedParams = React.useMemo(() => {
        const start = (currentPage - 1) * itemsPerPage
        return governanceParams.slice(start, start + itemsPerPage)
    }, [currentPage, governanceParams, itemsPerPage])

    const rows = paginatedParams.map((param, index) => {
        const row = []

        // Generate cells dynamically based on headers configuration
        Object.keys(stakingConfig.governance.table.headers).forEach((headerKey) => {
            let cellContent
            let cellClassName
            const cellAnimation = {
                initial: { opacity: 0, scale: 0.8 },
                animate: { opacity: 1, scale: 1 },
                transition: {
                    duration: stakingConfig.governance.table.animations.duration,
                    delay: index * stakingConfig.governance.table.animations.stagger
                }
            }

            switch (headerKey) {
                case 'paramName':
                    cellContent = param.paramName
                    cellClassName = stakingConfig.governance.table.styling.paramName
                    cellAnimation.initial = { opacity: 0, scale: 0.8 }
                    cellAnimation.animate = { opacity: 1, scale: 1 }
                    break
                case 'paramValue':
                    cellContent = formatParamValue(param.paramValue, param.paramName)
                    cellClassName = stakingConfig.governance.table.styling.paramValue
                    break
                case 'paramSpace':
                    cellContent = (
                        <>
                            <motion.i
                                className={`fa-solid ${stakingConfig.ui.icons[param.paramSpace] || stakingConfig.ui.icons.default} text-xs mr-1`}
                            ></motion.i>
                            <span className="capitalize">{param.paramSpace}</span>
                        </>
                    )
                    cellClassName = `${GREEN_BADGE_CLASS} capitalize`
                    break
                case 'paramType':
                    cellContent = (param as any).paramType || 'Unknown'
                    cellClassName = stakingConfig.governance.table.styling.paramType
                    break
                default:
                    // For any new headers added to JSON, use the param value directly
                    cellContent = param[headerKey] || ''
                    cellClassName = stakingConfig.governance.table.styling.paramValue
            }

            row.push(
                <motion.div
                    key={headerKey}
                    initial={cellAnimation.initial}
                    animate={cellAnimation.animate}
                    transition={cellAnimation.transition}
                >
                    <span className={cellClassName}>
                        {cellContent}
                    </span>
                </motion.div>
            )
        })

        return row
    })

    // Generate columns dynamically from JSON configuration
    const columns = Object.entries(stakingConfig.governance.table.headers).map(([key, label]) => ({
        key,
        label
    }))
    const governanceOverviewCards = stakingConfig.governance.stats.cards
        .filter((card) => card.visible)
        .map((card) => {
            let count = governanceParams.length
            if (card.title === 'Consensus Parameters') {
                count = governanceParams.filter(p => p.paramSpace === 'consensus').length
            } else if (card.title === 'Validator Parameters') {
                count = governanceParams.filter(p => p.paramSpace === 'validator').length
            } else if (card.title === 'Fee Parameters') {
                count = governanceParams.filter(p => p.paramSpace === 'fee').length
            } else if (card.title === 'Governance Parameters') {
                count = governanceParams.filter(p => p.paramSpace === 'governance').length
            }

            return {
                title: card.title,
                value: count.toLocaleString(),
                subValue: card.description,
                icon: `fa-solid ${card.icon}`,
            }
        })

    // Show loading state
    if (isLoading && stakingConfig.governance.table.loading.visible) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                <div className="mb-4">
                    <h2 className="explorer-page-title">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="explorer-page-subtitle">
                        {stakingConfig.governance.description}
                    </p>
                </div>
                <div className="bg-card rounded-lg p-8 text-center">
                    <i className={`fa-solid ${stakingConfig.governance.table.loading.spinner} fa-spin text-primary text-4xl mb-4`}></i>
                    <h3 className="text-white text-xl font-semibold mb-2">{stakingConfig.governance.table.loading.text}</h3>
                    <p className="text-gray-400">Fetching proposals from the network</p>
                </div>
            </motion.div>
        )
    }

    // Show error state
    if (error && stakingConfig.governance.table.error.visible) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                <div className="mb-6">
                    <h2 className="explorer-page-title">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="explorer-page-subtitle">
                        {stakingConfig.governance.description}
                    </p>
                </div>
                <div className="bg-card rounded-lg p-8 text-center border border-red-500/20">
                    <i className={`fa-solid ${stakingConfig.governance.table.error.icon} text-red-400 text-4xl mb-4`}></i>
                    <h3 className="text-white text-xl font-semibold mb-2">{stakingConfig.governance.table.error.text}</h3>
                    <p className="text-gray-400">Unable to fetch proposals from the network</p>
                    <p className="text-gray-500 text-sm mt-2">Using fallback data</p>
                </div>
            </motion.div>
        )
    }

    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.3 }}
        >
            {/* Header */}
            {stakingConfig.governance.visible && (
                <div className="mb-6">
                    <h2 className="explorer-page-title">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="explorer-page-subtitle">
                        {stakingConfig.governance.description}
                    </p>
                </div>
            )}

            {/* Governance Stats */}
            {stakingConfig.governance.stats.visible && (
                <ExplorerOverviewCards cards={governanceOverviewCards} className="mb-8" />
            )}

            {/* Governance Parameters Table */}
            {stakingConfig.governance.table.visible && (
                <div className="rounded-xl border border-white/10 bg-card p-5">
                    <div className="overflow-x-auto">
                        <table
                            className="w-full min-w-[880px]"
                            style={{ tableLayout: 'auto', borderCollapse: 'separate', borderSpacing: '0 4px' }}
                        >
                            <thead>
                                <tr>
                                    {columns.map((column) => (
                                        <th key={column.key} className={desktopHeaderClass}>
                                            {column.label}
                                        </th>
                                    ))}
                                </tr>
                            </thead>
                            <tbody>
                                {isLoading ? (
                                    Array.from({ length: itemsPerPage }).map((_, index) => (
                                        <tr key={`skeleton-${index}`} className="group animate-pulse">
                                            {columns.map((column, columnIndex) => (
                                                <td
                                                    key={`${column.key}-${index}`}
                                                    className={desktopRowCellClass}
                                                    style={{
                                                        borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderTopRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                        borderBottomRightRadius: columnIndex === columns.length - 1 ? '10px' : undefined,
                                                    }}
                                                >
                                                    <div className={`h-4 rounded bg-white/6 ${columnIndex === 0 ? 'w-40' : columnIndex === 1 ? 'w-28' : 'w-24'}`} />
                                                </td>
                                            ))}
                                        </tr>
                                    ))
                                ) : rows.length === 0 ? (
                                    <tr>
                                        <td colSpan={columns.length} className="px-5 py-10 text-center text-sm text-white/60">
                                            No governance parameters found
                                        </td>
                                    </tr>
                                ) : (
                                    rows.map((row, rowIndex) => (
                                        <tr key={`param-row-${rowIndex}`} className="group">
                                            {row.map((cell, columnIndex) => (
                                                <td
                                                    key={`param-cell-${rowIndex}-${columnIndex}`}
                                                    className={desktopRowCellClass}
                                                    style={{
                                                        borderTopLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderBottomLeftRadius: columnIndex === 0 ? '10px' : undefined,
                                                        borderTopRightRadius: columnIndex === row.length - 1 ? '10px' : undefined,
                                                        borderBottomRightRadius: columnIndex === row.length - 1 ? '10px' : undefined,
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

                    {!isLoading && governanceParams.length > 0 && (
                        <div className="mt-4 flex flex-col gap-3 text-sm text-white/60 md:flex-row md:items-center md:justify-between">
                            <div>
                                <span className="inline-flex items-baseline gap-1">
                                    <span>{formatPaginationRange(startIdx, endIdx)} of</span>
                                    <AnimatedNumber value={governanceParams.length} />
                                </span>
                            </div>

                            <div className="flex items-center gap-2">
                                <button
                                    type="button"
                                    onClick={() => goToPage(currentPage - 1)}
                                    disabled={currentPage === 1}
                                    className="rounded-lg border border-[#272729] bg-[#0f0f0f] px-3 py-1.5 text-white transition-colors hover:border-white/15 hover:bg-[#272729] disabled:cursor-not-allowed disabled:text-white/30 disabled:hover:border-[#272729] disabled:hover:bg-[#0f0f0f]"
                                >
                                    <i className="fa-solid fa-angle-left mr-1" />
                                    Previous
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
                                                className={`min-w-[36px] rounded-lg border px-3 py-1.5 transition-colors ${
                                                    currentPage === page
                                                        ? 'border-white/15 bg-[#272729] text-white'
                                                        : 'border-[#272729] bg-[#0f0f0f] text-white/70 hover:border-white/15 hover:bg-[#272729] hover:text-white'
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
                                    className="rounded-lg border border-[#272729] bg-[#0f0f0f] px-3 py-1.5 text-white transition-colors hover:border-white/15 hover:bg-[#272729] disabled:cursor-not-allowed disabled:text-white/30 disabled:hover:border-[#272729] disabled:hover:bg-[#0f0f0f]"
                                >
                                    Next
                                    <i className="fa-solid fa-angle-right ml-1" />
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            )}
        </motion.div>
    )
}

export default GovernanceView
