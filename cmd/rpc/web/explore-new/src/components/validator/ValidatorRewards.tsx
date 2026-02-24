import React, { useState } from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'
import TableCard from '../Home/TableCard'

interface BlockReward {
    blockHeight: number
    timestamp: string
    reward: number
    commission: number
    netReward: number
}

interface CrossChainReward {
    chain: string
    committeeId: string
    timestamp: string
    reward: number
    type: string
    icon: string
    color: string
}

interface Rewards {
    totalEarned: number
    last30Days: number
    averageDaily: number
    blockRewards: BlockReward[]
    crossChainRewards: CrossChainReward[]
}

interface ValidatorDetail {
    rewards?: Rewards
}

interface ValidatorRewardsProps {
    validator: ValidatorDetail
}

const ValidatorRewards: React.FC<ValidatorRewardsProps> = ({ validator }) => {
    const [activeTab, setActiveTab] = useState('rewardsHistory')

    const formatNumber = (num: number) => {
        return num.toLocaleString()
    }

    const formatReward = (reward: number) => {
        return `+${reward.toFixed(2)}`
    }

    const formatCommission = (commission: number, percentage: number) => {
        return `${commission.toFixed(2)} CNPY (${percentage}%)`
    }

    const getProgressBarColor = (color: string) => {
        switch (color) {
            case 'bg-blue-500':
                return 'bg-blue-500/30 text-blue-500'
            case 'bg-orange-500':
                return 'bg-orange-500/30 text-orange-500'
            case 'bg-purple-500':
                return 'bg-purple-500/30 text-purple-500'
            default:
                return 'bg-primary text-primary'
        }
    }

    const tabs = [
        { id: 'blocksProduced', label: validatorDetailTexts.rewards.subNav.blocksProduced },
        { id: 'stakeByCommittee', label: validatorDetailTexts.rewards.subNav.stakeByCommittee },
        { id: 'delegators', label: validatorDetailTexts.rewards.subNav.delegators },
        { id: 'rewardsHistory', label: validatorDetailTexts.rewards.subNav.rewardsHistory }
    ]

    return (
        <div className="bg-card rounded-lg p-6">
            {/* Header con navegación de pestañas */}
            <div className="mb-6">

                {/* Navegación de pestañas */}
                <div className="flex gap-1 border-b border-gray-700">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`px-4 py-2 text-sm font-medium transition-colors ${activeTab === tab.id
                                ? 'bg-primary text-black'
                                : 'text-gray-400 hover:text-white'
                                }`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </div>
                <div className="flex items-center justify-between mt-4">
                    <h2 className="text-xl font-bold text-white">
                        {validatorDetailTexts.rewards.title}
                    </h2>
                    <div className="flex items-center gap-4">
                        {validator.rewards && (
                            <div className="text-lg font-bold text-white">
                                <span className="text-sm text-gray-400 font-normal mr-2">
                                    Total Earned:
                                </span>
                                <span className="text-sm font-normal text-primary">
                                    {formatNumber(validator.rewards.totalEarned)} {validatorDetailTexts.metrics.units.cnpy}
                                </span>
                            </div>
                        )}
                        <div className="flex items-center gap-1 bg-green-500/10 rounded-full p-2 py-0.5">
                            <div className="w-1.5 h-1.5 bg-primary rounded-full animate-pulse"></div>
                            <span className="text-sm text-primary">
                                {validatorDetailTexts.rewards.live}
                            </span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Contenido de las pestañas */}
            {
                activeTab === 'rewardsHistory' && (
                    <div className="space-y-8">
                        {/* Resumen de ganancias */}
                        {validator.rewards && (
                            <div className="flex items-center gap-6 text-sm text-gray-400">
                                <span>
                                    {formatReward(validator.rewards.last30Days)} {validatorDetailTexts.metrics.units.cnpy} {validatorDetailTexts.rewards.last30Days}
                                </span>
                            </div>
                        )}

                        {/* Recompensas de producción de bloques */}
                        {validator.rewards && validator.rewards.blockRewards && validator.rewards.blockRewards.length > 0 ? (
                            <div>
                                <TableCard
                                    title={<div className="flex items-center flex-col justify-center"><div className="flex items-center justify-center gap-2"><div className="text-sm translate-y-3 bg-green-500/10 rounded-lg p-2 py-2 font-normal mr-2">
                                        <i className="fa-solid fa-leaf text-primary text-sm"></i>
                                    </div>Canopy Main Chain</div><p className="text-xs text-gray-400 font-normal translate-x-5.5 -translate-y-1">Block Production Rewards</p></div>}
                                    className="rounded-none border-none shadow-none p-5"
                                    live={false}
                                    columns={[
                                        { label: validatorDetailTexts.rewards.table.blockHeight },
                                        { label: validatorDetailTexts.rewards.table.timestamp },
                                        { label: validatorDetailTexts.rewards.table.reward },
                                        { label: validatorDetailTexts.rewards.table.commission },
                                        { label: validatorDetailTexts.rewards.table.netReward }
                                    ]}
                                    rows={validator.rewards.blockRewards.map((reward) => [
                                        <span className="text-primary">{formatNumber(reward.blockHeight)}</span>,
                                        <span className="text-gray-400">{reward.timestamp}</span>,
                                        <span className="text-primary">{formatReward(reward.reward)} {validatorDetailTexts.metrics.units.cnpy}</span>,
                                        <span className="text-gray-400">{formatCommission(reward.commission, 5)}</span>,
                                        <span className="text-primary">{formatReward(reward.netReward)} {validatorDetailTexts.metrics.units.cnpy}</span>
                                    ])}
                                    paginate={true}
                                    pageSize={10}
                                />
                            </div>
                        ) : (
                            <div className="text-center py-12 text-gray-400">
                                <p>Reward history data is not available from the endpoint.</p>
                                <p className="text-xs mt-2">According to the API documentation, reward history is not included in the validator endpoint response.</p>
                            </div>
                        )}

                        {/* Recompensas de cadenas anidadas */}
                        {validator.rewards && validator.rewards.crossChainRewards && validator.rewards.crossChainRewards.length > 0 && (
                            <div>
                                <div className="mb-4 text-sm text-gray-400">
                                    {formatReward(400.66)} Tokens {validatorDetailTexts.rewards.last30Days}
                                </div>
                                <TableCard
                                    title={<div className="flex items-center flex-col justify-center"><div className="flex items-center justify-center gap-2"><div className="text-sm translate-y-3 bg-blue-500/10 rounded-lg p-2 py-2 font-normal mr-2">
                                        <i className="fa-solid fa-bars text-blue-500 text-sm"></i>
                                    </div>Nested Chain Rewards</div><p className="text-xs text-gray-400 font-normal translate-x-5.5 -translate-y-1">Cross-chain validation rewards</p></div>}
                                    live={false}
                                    className="rounded-none border-none shadow-none p-5"
                                    columns={[
                                        { label: validatorDetailTexts.rewards.table.chain },
                                        { label: validatorDetailTexts.rewards.table.committeeId },
                                        { label: validatorDetailTexts.rewards.table.timestamp },
                                        { label: validatorDetailTexts.rewards.table.reward },
                                        { label: validatorDetailTexts.rewards.table.type }
                                    ]}
                                    rows={validator.rewards.crossChainRewards.map((reward) => [
                                        <div className="flex items-center gap-2">
                                            <div className={`w-6 h-6 ${reward.color} rounded-sm flex items-center justify-center`}>
                                                <i className={`${reward.icon} text-xs`}></i>
                                            </div>
                                            <span className="text-sm text-white">{reward.chain}</span>
                                        </div>,
                                        <span className="text-gray-400">{reward.committeeId}</span>,
                                        <span className="text-gray-400">{reward.timestamp}</span>,
                                        <span className="text-green-400">{formatReward(reward.reward)} {reward.chain.split(' ')[0].toUpperCase()}</span>,
                                        <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-normal ${getProgressBarColor(reward.color)}`}>
                                            {validatorDetailTexts.rewards.types.tag}
                                        </span>
                                    ])}
                                    paginate={true}
                                    pageSize={10}
                                />
                            </div>
                        )}

                        {/* Promedio diario */}
                        {validator.rewards && validator.rewards.averageDaily && (
                            <div className="pt-6 border-t border-gray-700">
                                <div className="text-sm text-gray-400 text-center">
                                    {validatorDetailTexts.rewards.averageDaily}: {formatNumber(validator.rewards.averageDaily)} {validatorDetailTexts.metrics.units.cnpy}/day
                                </div>
                            </div>
                        )}
                    </div>
                )
            }

            {/* Contenido para otras pestañas (placeholder) */}
            {
                activeTab !== 'rewardsHistory' && (
                    <div className="text-center py-12">
                        <div className="text-gray-400">
                            {tabs.find(tab => tab.id === activeTab)?.label} content coming soon...
                        </div>
                    </div>
                )
            }
        </div >
    )
}

export default ValidatorRewards
