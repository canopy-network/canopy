import React from 'react'

type LogoProps = {
    size?: number
    className?: string
    showText?: boolean
}

const Logo: React.FC<LogoProps> = ({ size, className = '', showText = true }) => {
    return (
        <div className={`flex items-center ${className}`}>
            <img
                src={showText ? '/canopy-wallet-logo.svg' : '/canopy-symbol.png'}
                alt="Canopy"
                className="w-auto object-contain"
                style={{ height: size ?? (showText ? 22 : 32) }}
            />
        </div>
    )
}

export default Logo
