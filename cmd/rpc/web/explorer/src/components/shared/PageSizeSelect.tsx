import React from 'react'

interface PageSizeSelectProps {
    value: number
    options?: number[]
    onChange: (value: number) => void
    label?: string
}

const DEFAULT_OPTIONS = [10, 25, 50, 100]

const PageSizeSelect: React.FC<PageSizeSelectProps> = ({
    value,
    options = DEFAULT_OPTIONS,
    onChange,
    label = 'Rows',
}) => {
    return (
        <label className="inline-flex items-center gap-2 text-[11px] uppercase tracking-[0.18em] text-white/40">
            <span>{label}</span>
            <span className="relative inline-flex items-center">
                <select
                    value={value}
                    onChange={(e) => onChange(Number(e.target.value))}
                    className="appearance-none rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 pr-7 text-[11px] font-medium tracking-normal text-white/75 outline-none transition-colors hover:border-white/20 focus:border-[#35cd48]/40"
                    aria-label={`${label} per page`}
                >
                    {options.map((option) => (
                        <option key={option} value={option} className="bg-[#111111] text-white">
                            {option}
                        </option>
                    ))}
                </select>
                <i className="fa-solid fa-angle-down pointer-events-none absolute right-2 text-[10px] text-white/35" />
            </span>
        </label>
    )
}

export default PageSizeSelect
