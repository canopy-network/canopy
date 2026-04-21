declare global {
    interface Window {
        __CONFIG__?: {
            rpcURL: string;
            adminRPCURL: string;
            chainId: number;
        };
    }
}

export { };
