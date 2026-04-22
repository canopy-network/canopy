import React from 'react'
import { useNavigate } from 'react-router-dom'
import { Blocks, Search } from 'lucide-react'
import { useLatestBlock } from '../../hooks/useApi'
import NetworkSelector from '../NetworkSelector'

const SUPPORT_URL = import.meta.env.VITE_SUPPORT_URL as string | undefined

const topBarButtonClass =
    'inline-flex h-9 items-center justify-center gap-2 whitespace-nowrap rounded-lg border border-white/15 px-3 text-sm font-medium text-zinc-400 transition-colors hover:bg-white/5 hover:text-white'

export const ExplorerTopBar = (): React.JSX.Element => {
    const navigate = useNavigate()
    const latestBlock = useLatestBlock()
    const [searchTerm, setSearchTerm] = React.useState('')

    const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault()
        const query = searchTerm.trim()
        if (!query) return
        navigate(`/search?q=${encodeURIComponent(query)}`)
        setSearchTerm('')
    }

    return (
        <header className="sticky top-0 z-[100] hidden h-16 flex-shrink-0 items-center justify-between gap-4 border-b border-border/40 bg-background px-6 lg:flex">
            <div className="flex min-w-0 items-center gap-3">
                <div className="flex h-9 items-center gap-2 rounded-lg border border-white/15 px-3 text-sm text-zinc-400">
                    <span className="relative flex h-1.5 w-1.5 flex-shrink-0">
                        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-white/70 opacity-70" />
                        <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-white" />
                    </span>
                    <Blocks className="h-4 w-4 flex-shrink-0" />
                    <span className="text-sm font-medium text-white">
                        #{latestBlock.data?.totalCount?.toLocaleString() || '0'}
                    </span>
                </div>
            </div>

            <form onSubmit={submitSearch} className="flex min-w-0 flex-1 justify-center px-4">
                <label className="flex h-11 w-full max-w-xl items-center gap-2 rounded-xl bg-card px-3 text-sm text-zinc-400 transition-colors focus-within:text-white">
                    <Search className="h-4 w-4 flex-shrink-0" />
                    <input
                        type="text"
                        value={searchTerm}
                        onChange={(event) => setSearchTerm(event.target.value)}
                        placeholder="Search blocks, transactions, addresses..."
                        className="h-full min-w-0 flex-1 bg-transparent text-sm text-white placeholder:text-zinc-500 outline-none"
                    />
                </label>
            </form>

            <div className="flex items-center gap-2">
                <button
                    type="button"
                    onClick={() => {
                        const url = SUPPORT_URL || 'https://discord.com/channels/1310733928436600912/1439049045145419806/1439945810446909560'
                        window.open(url, '_blank')
                    }}
                    className={topBarButtonClass}
                >
                    <svg
                        width="16"
                        height="16"
                        viewBox="0 0 16 16"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                        className="size-4 shrink-0"
                    >
                        <path
                            d="M8 9.33333V7M6.19615 12.3224L7.57285 13.4764C7.81949 13.6832 8.17861 13.6842 8.42643 13.4789L9.8253 12.32C9.94489 12.2209 10.0953 12.1667 10.2506 12.1667H11.5C12.6046 12.1667 13.5 11.2712 13.5 10.1667V4.5C13.5 3.39543 12.6046 2.5 11.5 2.5H4.5C3.39543 2.5 2.5 3.39543 2.5 4.5V10.1667C2.5 11.2712 3.39543 12.1667 4.5 12.1667H5.76788C5.9245 12.1667 6.07612 12.2218 6.19615 12.3224Z"
                            stroke="currentColor"
                            strokeLinecap="round"
                        />
                        <path d="M8 5.33337H8.00667" stroke="currentColor" strokeWidth="1.33333" strokeLinecap="round" />
                    </svg>
                    Create a Ticket
                </button>

                {import.meta.env.VITE_NODE_ENV === 'production' && <NetworkSelector />}
            </div>
        </header>
    )
}
