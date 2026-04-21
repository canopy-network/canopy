import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'
import CnpyColorIcon from '../ui/CnpyColorIcon'

interface NestedChain {
    name: string
    committeeId: number
    stakedAmount: number
    percentage: number
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

    const subscriptDigits = ['₀', '₁', '₂', '₃', '₄', '₅', '₆', '₇', '₈', '₉']

    const formatPercentage = (num: number) => {
        if (num === 0) return '0.00%'
        if (num >= 0.01) return `${num.toFixed(2)}%`

        // For very small numbers, use subscript notation: 0.0₅34%
        const str = num.toFixed(10)
        const [, decimal] = str.split('.')
        if (!decimal) return `${num.toFixed(2)}%`

        let leadingZeros = 0
        for (const c of decimal) {
            if (c === '0') leadingZeros++
            else break
        }

        if (leadingZeros < 2) return `${num.toFixed(4)}%`

        const significantDigits = decimal.slice(leadingZeros, leadingZeros + 3)
        const subscript = subscriptDigits[leadingZeros]
        return `0.0${subscript}${significantDigits}%`
    }

    return (
        <div className="mb-6 rounded-lg border border-[#272729] bg-[#171717] p-4 sm:p-6">
            <div className="mb-4 sm:mb-6">
                <h2 className="mb-2 text-lg font-bold text-white sm:text-xl">
                    {validatorDetailTexts.stakeByChains.title}
                </h2>
                <div className="break-words text-xs text-white/60 sm:text-sm">
                    {validatorDetailTexts.stakeByChains.totalDelegated}: {formatNumber(toCNPY(validator.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                </div>
            </div>

            <div className="space-y-3 sm:space-y-4">
                {validator.nestedChains.map((chain, index) => (
                    <div key={index} className="flex flex-col items-start justify-between gap-3 rounded-lg border border-[#272729] bg-[#0f0f0f] p-3 sm:flex-row sm:items-center sm:gap-4 sm:p-4">
                        <div className="flex items-start gap-3 sm:gap-4 w-full sm:w-auto">
                            {/* Chain icon */}
                            <CnpyColorIcon seed={chain.committeeId} size={40} color={chain.color} className="sm:h-10 sm:w-10" />

                            {/* Chain information */}
                            <div className="flex-1 min-w-0">
                                <div className="text-white font-medium text-sm sm:text-base">
                                    {chain.name}
                                </div>
                                <div className="text-xs text-white/60 sm:text-sm">
                                    Committee ID: {chain.committeeId}
                                </div>
                            </div>
                        </div>

                        {/* Stake information */}
                        <div className="flex items-center gap-4 sm:gap-6 w-full sm:w-auto justify-between sm:justify-end">
                            <div className="text-left sm:text-right">
                                <div className="text-white font-medium text-sm sm:text-base">
                                    {formatNumber(toCNPY(chain.stakedAmount))} {validatorDetailTexts.metrics.units.cnpy}
                                </div>
                                <div className="text-sm font-semibold sm:text-base" style={{ color: chain.color }}>
                                    {formatPercentage(chain.percentage)}
                                </div>
                            </div>
                        </div>
                    </div>
                ))}
            </div>

            {/* Total Network Control */}
            <div className="mt-4 border-t border-[#272729] pt-4 sm:mt-6">
                <div className="flex flex-col items-center justify-center gap-1 text-center text-xs text-white/60 sm:flex-row sm:gap-2 sm:text-sm">
                    <p>{validatorDetailTexts.stakeByChains.totalNetworkControl}:</p>
                    <p className="text-[#35cd48]">
                        {validator.nestedChains.length > 0 ? formatPercentage(validator.nestedChains[0].percentage) : '0.00%'} of total network stake
                    </p>
                </div>
            </div>
        </div>
    )
}

export default ValidatorStakeChains
