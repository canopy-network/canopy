import React from 'react'

type LogoProps = {
    size?: number
    className?: string
    showText?: boolean
}

const SYMBOL_THRESHOLD = 40;

const Logo: React.FC<LogoProps> = ({ size = 32, className = '' }) => {
    const useSymbol = size <= SYMBOL_THRESHOLD;
    const height = useSymbol ? size : Math.round(size * 0.18);

    return (
        <div className={`flex items-center ${className}`}>
            <img
                src={useSymbol ? '/canopy-symbol.png' : '/canopy-logo.png'}
                alt="Canopy"
                style={{ height }}
                className="flex-shrink-0 object-contain"
            />
        </div>
    )
}

export default Logo