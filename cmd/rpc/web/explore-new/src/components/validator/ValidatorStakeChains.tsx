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
        <div className="bg-card rounded-lg p-4 sm:p-6 mb-6">
            <div className="mb-4 sm:mb-6">
                <h2 className="text-lg sm:text-xl font-bold text-white mb-2">
                    {validatorDetailTexts.stakeByChains.title}
                </h2>
                <div className="text-xs sm:text-sm text-gray-400 break-words">
                    {validatorDetailTexts.stakeByChains.totalDelegated}: {formatNumber(toCNPY(validator.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                </div>
            </div>

            <div className="space-y-3 sm:space-y-4">
                {validator.nestedChains.map((chain, index) => (
                    <div key={index} className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 sm:gap-4 p-3 sm:p-4 bg-gray-800/30 rounded-lg">
                        <div className="flex items-start gap-3 sm:gap-4 w-full sm:w-auto">
                            {/* Icono de la cadena */}
                            <div className={`w-8 h-8 sm:w-10 sm:h-10 ${chain.color} rounded-md flex items-center justify-center flex-shrink-0`}>
                                <i className={`${chain.icon} text-xs sm:text-sm`}></i>
                            </div>

                            {/* Información de la cadena */}
                            <div className="flex-1 min-w-0">
                                <div className="text-white font-medium text-sm sm:text-base">
                                    {chain.name}
                                </div>
                                <div className="text-xs sm:text-sm text-gray-400">
                                    Committee ID: {chain.committeeId}
                                </div>
                                {/* Barra de progreso */}
                                <div className="w-full mt-2 sm:hidden">
                                    <div className="w-full bg-gray-700 rounded-full h-2">
                                        <div
                                            className={`h-2 rounded-full transition-all duration-300 ${getProgressBarColor(chain.color)}`}
                                            style={{ width: `${chain.percentage}%` }}
                                        ></div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        {/* Barra de progreso - Desktop */}
                        <div className="hidden sm:block w-full sm:w-auto sm:flex-1 max-w-xs">
                            <div className="w-full bg-gray-700 rounded-full h-2">
                                <div
                                    className={`h-2 rounded-full transition-all duration-300 ${getProgressBarColor(chain.color)}`}
                                    style={{ width: `${chain.percentage}%` }}
                                ></div>
                            </div>
                        </div>

                        {/* Información del stake */}
                        <div className="flex items-center gap-4 sm:gap-6 w-full sm:w-auto justify-between sm:justify-end">
                            <div className="text-left sm:text-right">
                                <div className="text-white font-medium text-sm sm:text-base">
                                    {formatNumber(toCNPY(chain.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                                </div>
                                <div className="text-xs sm:text-sm text-gray-400">
                                    {formatPercentage(chain.percentage)}
                                </div>
                            </div>
                        </div>
                    </div>
                ))}
            </div>

            {/* Total Network Control */}
            <div className="mt-4 sm:mt-6 pt-4 border-t border-gray-700">
                <div className="text-xs sm:text-sm text-gray-400 text-center flex flex-col sm:flex-row items-center justify-center gap-1 sm:gap-2">
                    <p>{validatorDetailTexts.stakeByChains.totalNetworkControl}:</p>
                    <p className="text-primary">
                        {validator.nestedChains.length > 0 ? formatPercentage(validator.nestedChains[0].percentage) : '0.00%'} of total network stake
                    </p>
                </div>
            </div>
        </div>
    )
}

export default ValidatorStakeChains
