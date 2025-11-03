import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'
import toast from 'react-hot-toast'

interface ValidatorDetail {
    address: string
    status: 'active' | 'paused' | 'unstaking' | 'inactive'
    stakedAmount: number
    committees: number[]
    delegate: boolean
    netAddress: string
    rank: number
    maxPausedHeight: number
    unstakingHeight: number
}

interface ValidatorDetailHeaderProps {
    validator: ValidatorDetail
}

const ValidatorDetailHeader: React.FC<ValidatorDetailHeaderProps> = ({ validator }) => {
    // Helper function to convert micro denomination to CNPY
    const toCNPY = (micro: number): number => {
        return micro / 1000000
    }

    // Generate deterministic icon based on address
    const getValidatorIcon = (address: string) => {
        // Create a simple hash from address to get a consistent index
        let hash = 0
        for (let i = 0; i < address.length; i++) {
            const char = address.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash // Convert to 32-bit integer
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
            'fa-solid fa-fire',
            'fa-solid fa-water',
            'fa-solid fa-wind',
            'fa-solid fa-snowflake',
            'fa-solid fa-gem',
            'fa-solid fa-circle',
            'fa-solid fa-square',
            'fa-solid fa-diamond'
        ]

        return icons[Math.abs(hash) % icons.length]
    }

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-500'
            case 'paused':
                return 'bg-yellow-500'
            case 'unstaking':
                return 'bg-orange-500'
            case 'inactive':
                return 'bg-gray-500'
            default:
                return 'bg-gray-500'
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
            case 'inactive':
                return validatorDetailTexts.header.status.inactive
            default:
                return 'Unknown'
        }
    }

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        // Here you could add a success notification
        toast.success('Address copied to clipboard', {
            duration: 2000,
            position: 'top-right',
            style: {
                background: '#1A1B23',
                color: '#4ADE80',
            },
        })
    }

    const shareToSocialMedia = (url: string) => {
        navigator.share({
            title: 'Share this validator',
            text: 'Share this validator',
            url: url
        })
    }

    // Determine button label and icon based on validator type
    const getValidatorTypeInfo = () => {
        // Priority: Unstaking > Paused > Delegate > Validator
        if (validator.unstakingHeight > 0) {
            return {
                label: 'Unstaking',
                icon: 'fa-solid fa-arrow-down',
                color: 'bg-orange-500 text-white'
            }
        }
        if (validator.maxPausedHeight > 0) {
            return {
                label: 'Paused',
                icon: 'fa-solid fa-pause-circle',
                color: 'bg-yellow-500 text-white'
            }
        }
        if (validator.delegate) {
            return {
                label: 'Delegator',
                icon: 'fa-solid fa-users',
                color: 'bg-blue-500 text-white'
            }
        }
        return {
            label: 'Validator',
            icon: 'fa-solid fa-shield-halved',
            color: 'bg-primary text-black'
        }
    }

    const typeInfo = getValidatorTypeInfo()

    return (
        <div className="bg-card rounded-lg p-6 mb-6">
            <div className="flex items-start justify-between">
                {/* Información del Validador */}
                <div className="flex items-center gap-4">
                    {/* Icono determinístico del Validador */}
                    <div className="w-16 h-16 bg-gradient-to-br from-green-300/20 to-green-300/10 rounded-full flex items-center justify-center">
                        <i className={`${getValidatorIcon(validator.address)} text-primary text-2xl`}></i>
                    </div>

                    {/* Detalles del Validador */}
                    <div className="flex items-center gap-4 flex-col">
                        <div>
                            <div className="flex items-center gap-2 mb-1">
                                <h1 className="text-2xl font-bold text-white">
                                    {validator.address}
                                </h1>
                                <i className="fa-solid fa-copy cursor-pointer hover:text-primary transition-colors text-gray-400"
                                    onClick={() => copyToClipboard(validator.address)}
                                    title="Copy address"></i>
                            </div>
                            {validator.netAddress && (
                                <div className="text-sm text-gray-400">
                                    {validator.netAddress}
                                </div>
                            )}
                        </div>
                        <div className="flex items-start justify-start gap-4 w-full">
                            {/* Estado */}
                            <div className="flex items-center justify-start gap-2 w-full">
                                <div className={`w-3 h-3 rounded-full ${getStatusColor(validator.status)}`}></div>
                                <span className="text-sm font-medium text-primary">
                                    {getStatusText(validator.status)}
                                </span>
                            </div>

                            {/* Committees */}
                            <div className="text-start flex items-center justify-start gap-2 w-full">
                                <div className="text-sm text-gray-400 text-nowrap">Committees:</div>
                                <div className="text-sm font-normal text-white">
                                    {validator.committees.length > 0 ? validator.committees.join(', ') : 'None'}
                                </div>
                            </div>

                            {/* Rank */}
                            {validator.rank > 0 && (
                                <div className="text-start flex items-center justify-start gap-2 w-full">
                                    <div className="text-sm text-gray-400">Rank:</div>
                                    <div className="text-sm font-normal text-white">
                                        #{validator.rank}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>

                </div>

                {/* Estado y Acciones */}
                <div className="flex items-start justify-start gap-4 h-full">

                    {/* Botones de Acción */}
                    <div className="flex items-start gap-3">
                        <button className={`flex items-center gap-2 ${typeInfo.color} px-4 py-2 rounded-lg hover:opacity-90 transition-colors`}>
                            <i className={`${typeInfo.icon} text-sm`}></i>
                            <span className="text-sm font-medium">
                                {typeInfo.label}
                            </span>
                        </button>
                        <button type="button" onClick={() => {
                            shareToSocialMedia(window.location.href)
                        }} className="flex items-start gap-2 bg-input border border-gray-800/60 text-gray-300 px-4 py-2 rounded-lg hover:bg-gray-800/50 transition-colors">
                            <i className="fa-solid fa-share text-sm translate-y-1"></i>
                            <span className="text-sm font-medium">
                                {validatorDetailTexts.header.actions.share}
                            </span>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ValidatorDetailHeader
