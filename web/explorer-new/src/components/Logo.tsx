import React from 'react'

type LogoProps = {
    size?: number
    className?: string
}

// Logo Canopy (hoja dentro de un recuadro redondeado)
const Logo: React.FC<LogoProps> = ({ size = 28, className }) => {
    const rounded = 6
    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 48 48"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className={className}
            aria-label="Canopy"
            role="img"
        >
            <rect x="2" y="2" width="44" height="44" rx={rounded} fill="#4ADE80" />
            <path
                d="M30.5 14.5c-4.2 0-7.9 2.7-9.3 6.7-.4 1.1-1.4 1.8-2.6 1.8H14c0 5.7 4.6 10.3 10.3 10.3 5.7 0 10.3-4.6 10.3-10.3 0-4.5-2.9-8.4-7-9.8Z"
                fill="#0F172A"
            />
            <path
                d="M18.8 25.2c1.5 3.3 4.8 5.6 8.6 5.6 1.4 0 2.8-.3 4-.9-2.8-1.2-5.1-3.5-6.3-6.3-1.4.9-3.6 1.6-6.3 1.6Z"
                fill="#111827"
                opacity=".5"
            />
        </svg>
    )
}

export default Logo