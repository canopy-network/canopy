import React from 'react'
import { motion } from 'framer-motion'
import TableCard from '../Home/TableCard'
import { useDAO } from '../../hooks/useApi'
import stakingConfig from '../../data/staking.json'

interface GovernanceParam {
    paramName: string
    paramValue: string | number
    paramSpace: string
    visible: boolean
}

const GovernanceView: React.FC = () => {
    // Use endpoint configuration from JSON
    const daoEndpoint = stakingConfig.endpoints.dao
    const { data: daoData, isLoading, error } = useDAO(daoEndpoint.params.id)


    // Function to get governance parameters from config
    const getGovernanceParams = (): GovernanceParam[] => {
        // Filter visible parameters from config
        const visibleParams = stakingConfig.governance.parameters.filter(param => param.visible)

        // If there's real DAO data, try to use some real values
        if (daoData && daoData.id && daoData.amount) {
            return visibleParams.map(param => {
                const updatedParam = { ...param }

                if (param.paramName === 'rootChainID' && daoData.id) {
                    updatedParam.paramValue = daoData.id.toString()
                }
                if (param.paramName === 'minimumOrderSize' && daoData.amount) {
                    const minOrder = Math.floor(daoData.amount / 1000000) // Convert to CNPY
                    updatedParam.paramValue = minOrder.toLocaleString()
                }

                return updatedParam
            })
        }

        return visibleParams
    }

    const governanceParams = getGovernanceParams()

    const getParamSpaceColor = (space: string) => {
        return stakingConfig.ui.colors[space] || stakingConfig.ui.colors.default
    }

    const formatParamValue = (value: string | number) => {
        if (typeof value === 'number') {
            return value.toLocaleString()
        }
        return value.toString()
    }

    // Generate rows dynamically based on JSON configuration
    const rows = governanceParams.map((param, index) => {
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
                    cellContent = formatParamValue(param.paramValue)
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
                    cellClassName = `${stakingConfig.governance.table.styling.paramSpace} ${getParamSpaceColor(param.paramSpace)}`
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
                <motion.span
                    key={headerKey}
                    className={cellClassName}
                    initial={cellAnimation.initial}
                    animate={cellAnimation.animate}
                    transition={cellAnimation.transition}
                >
                    {cellContent}
                </motion.span>
            )
        })

        return row
    })

    // Generate columns dynamically from JSON configuration
    const columns = Object.entries(stakingConfig.governance.table.headers).map(([key, label]) => ({
        key,
        label
    }))

    // Show loading state
    if (isLoading && stakingConfig.governance.table.loading.visible) {
        return (
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.3 }}
            >
                <div className="mb-6">
                    <h2 className="text-2xl font-bold text-white mb-2">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="text-gray-400">
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
                    <h2 className="text-2xl font-bold text-white mb-2">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="text-gray-400">
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
                    <h2 className="text-2xl font-bold text-white mb-2">
                        {stakingConfig.governance.title}
                    </h2>
                    <p className="text-gray-400">
                        {stakingConfig.governance.description}
                    </p>
                    {daoData ? (
                        <p className="text-primary text-sm mt-2">
                            <i className="fa-solid fa-database mr-1"></i>
                            {stakingConfig.governance.daoDataText}
                        </p>
                    ) : (
                        <p className="text-yellow-400 text-sm mt-2">
                            <i className="fa-solid fa-exclamation-triangle mr-1"></i>
                            {stakingConfig.governance.daoDataTextFallback}
                        </p>
                    )}
                </div>
            )}

            {/* Governance Parameters Table */}
            {stakingConfig.governance.table.visible && (
                <TableCard
                    title={stakingConfig.governance.table.title}
                    columns={columns}
                    rows={rows}
                    totalCount={governanceParams.length}
                    currentPage={1}
                    onPageChange={() => { }}
                    loading={isLoading}
                    spacing={stakingConfig.governance.table.spacing}
                />
            )}

            {/* Governance Stats */}
            {stakingConfig.governance.stats.visible && (
                <motion.div
                    className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-6"
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.5, delay: 0.4 }}
                >
                    {stakingConfig.governance.stats.cards.map((card, index) => {
                        if (!card.visible) return null

                        const getColorClass = (color: string) => {
                            switch (color) {
                                case 'blue': return 'text-blue-400'
                                case 'primary': return 'text-primary'
                                case 'purple': return 'text-purple-400'
                                default: return 'text-gray-400'
                            }
                        }

                        const getCount = () => {
                            if (card.title === 'Consensus Parameters') {
                                return governanceParams.filter(p => p.paramSpace === 'consensus').length
                            } else if (card.title === 'Validator Parameters') {
                                return governanceParams.filter(p => p.paramSpace === 'validator').length
                            } else {
                                return governanceParams.length
                            }
                        }

                        return (
                            <motion.div
                                key={card.title}
                                className="bg-card rounded-lg p-6 border border-gray-800/50 relative"
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                transition={{ duration: 0.3, delay: 0.1 + (index * 0.1) }}
                            >
                                {/* Icon in top-right */}
                                <div className="absolute top-4 right-4">
                                    <i className={`fa-solid ${card.icon} ${getColorClass(card.color)} text-xl`}></i>
                                </div>

                                {/* Title */}
                                <div className="mb-4">
                                    <h3 className="text-white font-medium text-sm">{card.title}</h3>
                                </div>

                                {/* Main Value */}
                                <div className="mb-2">
                                    <div className={`text-3xl font-bold ${getColorClass(card.color)}`}>
                                        {getCount()}
                                    </div>
                                </div>

                                {/* Description */}
                                <div className="flex items-center gap-2">
                                    <span className="text-gray-400 text-sm">
                                        {card.description}
                                    </span>
                                </div>
                            </motion.div>
                        )
                    })}
                </motion.div>
            )}
        </motion.div>
    )
}

export default GovernanceView
