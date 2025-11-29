import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'

interface NestedChain {
    name: string
    committeeId: number
    stakedAmount: number
    percentage: number
    icon: string
    color: string
}

interface ValidatorDetail {
    stakedAmount: number
    nestedChains: NestedChain[]
}

interface ValidatorStakeChainsProps {
    validator: ValidatorDetail
}

const ValidatorStakeChains: React.FC<ValidatorStakeChainsProps> = ({ validator }) => {
    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    const formatNumber = (num: number) => {
        return num.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 6 })
    }

    const formatPercentage = (num: number) => {
        return `${num.toFixed(2)}%`
    }

    const getProgressBarColor = (color: string) => {
        switch (color) {
            case 'bg-green-500':
                return 'bg-green-500'
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

    return (
        <div className="bg-card rounded-lg p-6 mb-6">
            <div className="mb-6">
                <h2 className="text-xl font-bold text-white mb-2">
                    {validatorDetailTexts.stakeByChains.title}
                </h2>
                <div className="text-sm text-gray-400">
                    {validatorDetailTexts.stakeByChains.totalDelegated}: {formatNumber(toCNPY(validator.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                </div>
            </div>

            <div className="space-y-4">
                {validator.nestedChains.map((chain, index) => (
                    <div key={index} className="flex items-center justify-between p-4 bg-gray-800/30 rounded-lg">
                        <div className="flex items-start gap-4 flex-col w-full">
                            <div className="flex items-center gap-4">
                                {/* Icono de la cadena */}
                                <div className={`w-10 h-10 ${chain.color} rounded-md flex items-center justify-center`}>
                                    <i className={`${chain.icon} text-sm`}></i>
                                </div>

                                {/* Información de la cadena */}
                                <div>
                                    <div className="text-white font-medium">
                                        {chain.name}
                                    </div>
                                    <div className="text-sm text-gray-400">
                                        Committee ID: {chain.committeeId}
                                    </div>
                                </div>
                            </div>
                            {/* Barra de progreso */}
                            <div className="w-full">
                                <div className="w-full bg-gray-700 rounded-full h-2">
                                    <div
                                        className={`h-2 rounded-full transition-all duration-300 ${getProgressBarColor(chain.color)}`}
                                        style={{ width: `${chain.percentage}%` }}
                                    ></div>
                                </div>
                            </div>
                        </div>

                        {/* Información del stake */}
                        <div className="flex items-center gap-6">
                            <div className="text-right">
                                <div className="text-white font-medium">
                                    {formatNumber(toCNPY(chain.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                                </div>
                                <div className="text-sm text-gray-400">
                                    {formatPercentage(chain.percentage)}
                                </div>
                            </div>

                        </div>
                    </div>
                ))}
            </div>

            {/* Total Network Control */}
            <div className="mt-6 pt-4 border-t border-gray-700">
                <div className="text-sm text-gray-400 text-center flex  items-center gap-2 w-full">
                    <p>{validatorDetailTexts.stakeByChains.totalNetworkControl}: </p>
                    <p className="text-primary">
                        {validator.nestedChains.length > 0 ? formatPercentage(validator.nestedChains[0].percentage) : '0.00%'} of total network stake
                    </p>
                </div>
            </div>
        </div>
    )
}

export default ValidatorStakeChains
