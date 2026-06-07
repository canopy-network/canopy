import React from 'react'

type LogoProps = {
    size?: number
    className?: string
    showText?: boolean
}

const Logo: React.FC<LogoProps> = ({ size, className = '', showText = true }) => {
    const src = showText ? '/canopy-explorer-logo.svg' : '/canopy-symbol.png'
    const height = size ?? (showText ? 24 : 32)

    return (
        <div className={`flex items-center ${className}`}>
            <img
                src={src}
                alt={showText ? 'Canopy Explorer' : 'Canopy'}
                style={{ height }}
                className="flex-shrink-0 object-contain"
            />
        </div>
    )
}

export default Logo
