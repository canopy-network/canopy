// Global type declarations

// Extend Window interface to include __CONFIG__
declare global {
    interface Window {
        __CONFIG__?: {
            rpcURL: string;
            adminRPCURL: string;
            chainId: number;
        };
    }
}

// Export to make it a module
export { };
