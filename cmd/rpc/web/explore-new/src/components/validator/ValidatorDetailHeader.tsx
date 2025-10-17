import React from 'react'
import validatorDetailTexts from '../../data/validatorDetail.json'
import toast from 'react-hot-toast'

interface ValidatorDetail {
    address: string
    name: string
    status: 'active' | 'inactive' | 'jailed'
    rank: number
    stakeWeight: number
    validatorName: string
}

interface ValidatorDetailHeaderProps {
    validator: ValidatorDetail
}

const ValidatorDetailHeader: React.FC<ValidatorDetailHeaderProps> = ({ validator }) => {
    const getStatusColor = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-500'
            case 'inactive':
                return 'bg-gray-500'
            case 'jailed':
                return 'bg-red-500'
            default:
                return 'bg-gray-500'
        }
    }

    const getStatusText = (status: string) => {
        switch (status) {
            case 'active':
                return validatorDetailTexts.header.status.active
            case 'inactive':
                return validatorDetailTexts.header.status.inactive
            case 'jailed':
                return validatorDetailTexts.header.status.jailed
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

    return (
        <div className="bg-card rounded-lg p-6 mb-6">
            <div className="flex items-start justify-between">
                {/* Información del Validador */}
                <div className="flex items-center gap-4">
                    {/* Avatar del Validador */}
                    <div className="w-16 h-16 bg-gradient-to-br from-green-300/20 to-green-300/10 rounded-full flex items-center justify-center">
                        <span className="text-2xl font-bold text-primary">
                            {validator.validatorName.charAt(0)}
                        </span>
                    </div>

                    {/* Detalles del Validador */}
                    <div className="flex items-center gap-4 flex-col">
                        <div>
                            <h1 className="text-2xl font-bold text-white mb-1">
                                {validator.validatorName}
                            </h1>
                            <div className="flex items-center gap-4 text-sm text-gray-400">
                                <div className="flex items-center gap-2">
                                    Address:
                                    <span className="font-mono text-primary">
                                        {validator.address}
                                    </span>
                                    <i className="fa-solid fa-copy cursor-pointer hover:text-white transition-colors"
                                        onClick={() => copyToClipboard(validator.address)}
                                        title="Copy address"></i>
                                </div>
                            </div>
                        </div>
                        <div className="flex items-start justify-start gap-4 w-full">
                            {/* Estado */}
                            <div className="flex items-center justify-start gap-2 w-full">
                                <div className={`w-3 h-3 rounded-full ${getStatusColor(validator.status)}`}></div>
                                <span className="text-sm font-medium text-primary">
                                    {getStatusText(validator.status)}
                                </span>
                            </div>

                            {/* Rank */}
                            <div className="text-start flex items-center justify-start gap-2 w-full">
                                <div className="text-sm text-gray-400">Rank:</div>
                                <div className="text-sm font-normal text-white">
                                    #{validator.rank}
                                </div>
                            </div>

                            {/* Stake Weight */}
                            <div className="text-start flex items-center justify-start gap-2 w-full">
                                <div className="text-sm text-gray-400 text-nowrap">Stake Weight:</div>
                                <div className="text-sm font-normal text-white">
                                    {validator.stakeWeight}%
                                </div>
                            </div>
                        </div>
                    </div>

                </div>

                {/* Estado y Acciones */}
                <div className="flex items-start justify-start gap-4 h-full">

                    {/* Botones de Acción */}
                    <div className="flex items-start gap-3">
                        <button className="flex items-center gap-2 bg-primary text-black px-4 py-2 rounded-lg hover:bg-primary/90 transition-colors">
                            <i className="fa-solid fa-coins text-sm"></i>
                            <span className="text-sm font-medium">
                                {validatorDetailTexts.header.actions.delegate}
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
