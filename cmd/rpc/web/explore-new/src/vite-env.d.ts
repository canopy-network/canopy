/// <reference types="vite/client" />

interface ImportMetaEnv {
    readonly VITE_RPC_URL: string
    readonly VITE_ADMIN_RPC_URL: string
    readonly VITE_CHAIN_ID: string
    readonly VITE_NODE_ENV: string
    readonly VITE_PUBLIC_RPC_URL: string
    readonly VITE_PUBLIC_ADMIN_RPC_URL: string
}

interface ImportMeta {
    readonly env: ImportMetaEnv
}
