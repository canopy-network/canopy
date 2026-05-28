import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'
import toast from 'react-hot-toast'
import { Copy } from 'lucide-react'
import CnpyColorIcon from '../ui/CnpyColorIcon'
import { GREEN_BADGE_CLASS } from '../ui/badgeStyles'

interface ValidatorDetail {
    address: string
    publicKey: string
    status: 'active' | 'paused' | 'unstaking' | 'delegate'
    stakedAmount: number
    committees: number[]
    delegate: boolean
    compound: boolean
    netAddress: string
    rank: number
    maxPausedHeight: number
    unstakingHeight: number
}

interface ValidatorDetailHeaderProps {
    validator: ValidatorDetail
}

const CopySymbol = () => <Copy aria-hidden="true" className="h-3.5 w-3.5" strokeWidth={2} />

const truncateMiddle = (value: string, start: number = 18, end: number = 18) => {
    if (!value || value.length <= start + end + 1) return value
    return `${value.slice(0, start)}…${value.slice(-end)}`
}

const ValidatorDetailHeader: React.FC<ValidatorDetailHeaderProps> = ({ validator }) => {
    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-[#35cd48]'
            case 'paused':
                return 'bg-[#ddb228]'
            case 'unstaking':
                return 'bg-[#ff1845]'
            case 'delegate':
                return 'bg-[#216cd0]'
            default:
                return 'bg-[#272729]'
        }
    }

    const getStatusText = (status: string) => {
        switch (status) {
            case 'active':
                return validatorDetailTexts.header.status.active
            case 'paused':
                return 'Paused'
            case 'unstaking':
                return 'Unstaking'
            case 'delegate':
                return 'Delegator'
            default:
                return 'Unknown'
        }
    }

    const copyToClipboard = (text: string, label: string = 'Address') => {
        navigator.clipboard.writeText(text)
        toast.success(`${label} copied to clipboard`, {
            duration: 2000,
            position: 'top-right',
            style: {
                background: '#171717',
                color: '#35cd48',
            },
        })
    }

    // Determine button label and icon based on validator type
    const getValidatorTypeInfo = () => {
        // Priority: Unstaking > Paused > Delegate > Validator
        if (validator.unstakingHeight > 0) {
            return {
                label: 'Unstaking',
                icon: 'fa-solid fa-arrow-down',
                color: GREEN_BADGE_CLASS
            }
        }
        if (validator.maxPausedHeight > 0) {
            return {
                label: 'Paused',
                icon: 'fa-solid fa-pause-circle',
                color: GREEN_BADGE_CLASS
            }
        }
        if (validator.delegate) {
            return {
                label: 'Delegator',
                icon: 'fa-solid fa-users',
                color: GREEN_BADGE_CLASS
            }
        }
        return {
            label: 'Validator',
            icon: 'fa-solid fa-shield-halved',
            color: GREEN_BADGE_CLASS
        }
    }

    const typeInfo = getValidatorTypeInfo()

    return (
        <div className="mb-6 rounded-lg border border-[#272729] bg-[#171717] p-4 sm:p-6">
            <div className="flex flex-col lg:flex-row items-start justify-between gap-4">
                {/* Validator information */}
                <div className="flex items-start gap-3 sm:gap-4 w-full lg:w-auto">
                    <CnpyColorIcon seed={validator.address} size={64} color="#E5E7EB" />

                    {/* Validator details */}
                    <div className="flex-1 min-w-0">
                        <div className="mb-3">
                            <div className="flex items-center gap-2 mb-1 flex-wrap">
                                <h1 className="text-base sm:text-xl md:text-2xl font-bold text-white break-all">
                                    {validator.address}
                                </h1>
                                <button
                                    type="button"
                                    onClick={() => copyToClipboard(validator.address)}
                                    aria-label="Copy validator address"
                                    className="flex-shrink-0 text-[#35cd48] transition-colors hover:text-white"
                                    title="Copy address"
                                >
                                    <CopySymbol />
                                </button>
                            </div>
                            {validator.netAddress && (
                                <div className="flex min-w-0 items-start gap-2 text-xs sm:text-sm">
                                    <span className="shrink-0 font-bold text-white">Net Address:</span>
                                    <span className="min-w-0 break-all text-white">{validator.netAddress}</span>
                                </div>
                            )}
                            {validator.publicKey && (
                                <div className="mt-2 flex min-w-0 items-start gap-2 text-xs sm:text-sm">
                                    <span className="shrink-0 font-bold text-white">Public Key:</span>
                                    <span className="min-w-0 text-white" title={validator.publicKey}>
                                        {truncateMiddle(validator.publicKey)}
                                    </span>
                                    <button
                                        type="button"
                                        onClick={() => copyToClipboard(validator.publicKey, 'Public key')}
                                        aria-label="Copy validator public key"
                                        className="mt-0.5 flex-shrink-0 text-[#35cd48] transition-colors hover:text-white"
                                        title="Copy public key"
                                    >
                                        <CopySymbol />
                                    </button>
                                </div>
                            )}
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
                            {/* Status */}
                            <div className="flex items-center gap-2">
                                <div className={`w-3 h-3 rounded-full flex-shrink-0 ${getStatusColor(validator.status)}`}></div>
                                <span className="text-xs sm:text-sm font-medium text-white">
                                    {getStatusText(validator.status)}
                                </span>
                            </div>

                            {/* Committees */}
                            <div className="text-start flex items-center gap-2">
                                <div className="text-xs sm:text-sm text-white/60 whitespace-nowrap">Committee IDs:</div>
                                <div className="text-xs sm:text-sm font-normal text-white break-words">
                                    {validator.committees.length > 0 ? validator.committees.join(', ') : 'None'}
                                </div>
                            </div>

                            {/* Rank */}
                            {validator.rank > 0 && (
                                <div className="text-start flex items-center gap-2">
                                    <div className="text-xs sm:text-sm text-white/60">Rank:</div>
                                    <div className="text-xs sm:text-sm font-normal text-white">
                                        #{validator.rank}
                                    </div>
                                </div>
                            )}

                            {/* Auto-Compound */}
                            <div className="text-start flex items-center gap-2">
                                <div className="text-xs sm:text-sm text-white/60 whitespace-nowrap">Auto-Compound:</div>
                                <div className={`text-xs sm:text-sm font-normal flex items-center gap-1 ${validator.compound ? 'text-[#35cd48]' : 'text-white/45'}`}>
                                    <i className={`fa-solid ${validator.compound ? 'fa-check-circle' : 'fa-times-circle'}`}></i>
                                    <span>{validator.compound ? 'Enabled' : 'Disabled'}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Type badge */}
                <div className="flex items-start justify-start gap-4 h-full w-full lg:w-auto">
                    <div className="flex items-start gap-3">
                        <span className={`${typeInfo.color} gap-2 px-3`}>
                            <i className={`${typeInfo.icon} text-sm`}></i>
                            <span className="text-sm font-medium">
                                {typeInfo.label}
                            </span>
                        </span>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ValidatorDetailHeader
