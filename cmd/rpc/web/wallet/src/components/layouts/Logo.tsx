import React from 'react'

type LogoProps = {
    size?: number
    className?: string
    showText?: boolean
}

const Logo: React.FC<LogoProps> = ({ className = '', showText = true }) => {
    return (
        <div className={`flex items-center ${className}`}>
            <img
                src={showText ? '/canopy-logo.png' : '/canopy-symbol.png'}
                alt="Canopy"
                className="h-8 w-auto object-contain"
            />
        </div>
    )
}

export default Logo