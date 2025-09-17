import React, { useState } from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'

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
    rewards: Rewards
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
                return 'bg-blue-500'
            case 'bg-orange-500':
                return 'bg-orange-500'
            case 'bg-purple-500':
                return 'bg-purple-500'
            default:
                return 'bg-primary'
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
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-xl font-bold text-white">
                        {validatorDetailTexts.rewards.title}
                    </h2>
                    <div className="flex items-center gap-2">
                        <div className="text-lg font-bold text-white">
                            {formatNumber(validator.rewards.totalEarned)} {validatorDetailTexts.metrics.units.cnpy}
                        </div>
                        <div className="flex items-center gap-1">
                            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                            <span className="text-sm text-green-400">
                                {validatorDetailTexts.rewards.live}
                            </span>
                        </div>
                    </div>
                </div>

                {/* Navegación de pestañas */}
                <div className="flex gap-1 bg-gray-800/50 rounded-lg p-1">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${activeTab === tab.id
                                ? 'bg-primary text-black'
                                : 'text-gray-400 hover:text-white'
                                }`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </div>
            </div>

            {/* Contenido de las pestañas */}
            {activeTab === 'rewardsHistory' && (
                <div className="space-y-6">
                    {/* Resumen de ganancias */}
                    <div className="flex items-center gap-6 text-sm text-gray-400">
                        <span>
                            {formatReward(validator.rewards.last30Days)} {validatorDetailTexts.metrics.units.cnpy} {validatorDetailTexts.rewards.last30Days}
                        </span>
                    </div>

                    {/* Recompensas de producción de bloques */}
                    <div>
                        <h3 className="text-lg font-semibold text-white mb-4">
                            Canopy Main Chain ({validatorDetailTexts.rewards.subNav.blocksProduced.toLowerCase()})
                        </h3>
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-gray-700">
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.blockHeight}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.timestamp}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.reward}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.commission}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.netReward}
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {validator.rewards.blockRewards.map((reward, index) => (
                                        <tr key={index} className="border-b border-gray-800/50">
                                            <td className="py-3 px-4 text-sm text-white">
                                                {formatNumber(reward.blockHeight)}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-gray-400">
                                                {reward.timestamp}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-green-400">
                                                {formatReward(reward.reward)} {validatorDetailTexts.metrics.units.cnpy}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-gray-400">
                                                {formatCommission(reward.commission, 5)}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-green-400">
                                                {formatReward(reward.netReward)} {validatorDetailTexts.metrics.units.cnpy}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>

                    {/* Recompensas de cadenas anidadas */}
                    <div>
                        <h3 className="text-lg font-semibold text-white mb-4">
                            Nested Chain Rewards (Cross-chain validation rewards)
                        </h3>
                        <div className="mb-4 text-sm text-gray-400">
                            {formatReward(400.66)} Tokens {validatorDetailTexts.rewards.last30Days}
                        </div>
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-gray-700">
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.chain}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.committeeId}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.timestamp}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.reward}
                                        </th>
                                        <th className="text-left py-3 px-4 text-sm text-gray-400">
                                            {validatorDetailTexts.rewards.table.type}
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {validator.rewards.crossChainRewards.map((reward, index) => (
                                        <tr key={index} className="border-b border-gray-800/50">
                                            <td className="py-3 px-4">
                                                <div className="flex items-center gap-2">
                                                    <div className={`w-6 h-6 ${reward.color} rounded-full flex items-center justify-center`}>
                                                        <i className={`${reward.icon} text-white text-xs`}></i>
                                                    </div>
                                                    <span className="text-sm text-white">{reward.chain}</span>
                                                </div>
                                            </td>
                                            <td className="py-3 px-4 text-sm text-gray-400">
                                                {reward.committeeId}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-gray-400">
                                                {reward.timestamp}
                                            </td>
                                            <td className="py-3 px-4 text-sm text-green-400">
                                                {formatReward(reward.reward)} {reward.chain.split(' ')[0].toUpperCase()}
                                            </td>
                                            <td className="py-3 px-4">
                                                <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getProgressBarColor(reward.color)} text-white`}>
                                                    {validatorDetailTexts.rewards.types.tag}
                                                </span>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>

                    {/* Promedio diario */}
                    <div className="pt-4 border-t border-gray-700">
                        <div className="text-sm text-gray-400 text-center">
                            {validatorDetailTexts.rewards.averageDaily}: {formatNumber(validator.rewards.averageDaily)} {validatorDetailTexts.metrics.units.cnpy}/day
                        </div>
                    </div>
                </div>
            )}

            {/* Contenido para otras pestañas (placeholder) */}
            {activeTab !== 'rewardsHistory' && (
                <div className="text-center py-12">
                    <div className="text-gray-400">
                        {tabs.find(tab => tab.id === activeTab)?.label} content coming soon...
                    </div>
                </div>
            )}
        </div>
    )
}

export default ValidatorRewards
