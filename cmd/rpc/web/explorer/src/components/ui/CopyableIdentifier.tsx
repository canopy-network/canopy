import React from 'react'
import { Copy } from 'lucide-react'
import { Link } from 'react-router-dom'
import toast from 'react-hot-toast'

interface CopyableIdentifierProps {
    value: string
    label?: string
    children: React.ReactNode
    to?: string
    className?: string
    iconClassName?: string
}

const fallbackCopy = (value: string) => {
    const textarea = document.createElement('textarea')
    textarea.value = value
    textarea.setAttribute('readonly', '')
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
}

const copyValue = async (value: string, label: string) => {
    try {
        if (navigator.clipboard && window.isSecureContext) {
            await navigator.clipboard.writeText(value)
        } else {
            fallbackCopy(value)
        }
        toast.success(`${label} copied`)
    } catch {
        toast.error(`Unable to copy ${label.toLowerCase()}`)
    }
}

const CopyableIdentifier: React.FC<CopyableIdentifierProps> = ({
    value,
    label = 'Identifier',
    children,
    to,
    className = '',
    iconClassName = '',
}) => (
    <span className={`inline-flex min-w-0 items-center gap-1.5 ${className}`}>
        {to ? (
            <Link
                to={to}
                data-row-click-ignore="true"
                className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap transition-colors hover:text-primary"
                title={value}
            >
                {children}
            </Link>
        ) : (
            <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
                {children}
            </span>
        )}
        <button
            type="button"
            data-row-click-ignore="true"
            className="group/copy inline-flex shrink-0 items-center justify-center rounded p-0.5 text-white/35 transition-colors hover:bg-white/5 hover:text-primary"
            title={`Copy ${label.toLowerCase()}: ${value}`}
            aria-label={`Copy ${label.toLowerCase()}`}
            onClick={(event) => {
                event.preventDefault()
                event.stopPropagation()
                void copyValue(value, label)
            }}
        >
            <span className="sr-only">
                Copy {label.toLowerCase()}
            </span>
            <Copy
                className={`h-3.5 w-3.5 shrink-0 opacity-0 transition-opacity group-hover/copy:opacity-100 group-focus-visible/copy:opacity-100 ${iconClassName}`}
                strokeWidth={2}
                aria-hidden="true"
            />
        </button>
    </span>
)

export default CopyableIdentifier
