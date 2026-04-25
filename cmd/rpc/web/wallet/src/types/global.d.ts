declare global {
    interface Window {
        __CONFIG__?: {
            rpcURL: string;
            adminRPCURL: string;
            explorerBaseURL?: string;
            chainId: number;
        };
    }
}

export { };
