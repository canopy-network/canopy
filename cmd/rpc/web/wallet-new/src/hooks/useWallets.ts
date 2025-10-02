// src/hooks/useWallets.ts
import {  useQuery, useQueryClient } from '@tanstack/react-query';
import { QK } from '@/core/queryKeys';
// import { makeRpc } from '@/core/rpc';

export type Wallet = { id: string; name: string; address: string; isActive?: boolean };

async function fetchWallets(): Promise<Wallet[]> {
    // A: desde contexto
    const { wallets } = (window as any).__configCtx ?? {};
    return (wallets ?? []) as Wallet[];

    // B: desde Admin RPC
    // const rpc = makeRpc('admin');
    // const res = await rpc.get<{ wallets: Wallet[] }>('/admin/wallets');
    // return res.wallets;
}

export function useWallets() {
    const qc = useQueryClient();

    const query = useQuery({
        queryKey: QK.WALLETS,
        queryFn: fetchWallets,
        staleTime: 60_000,
        refetchOnWindowFocus: false,
    });

    const activeWallet = query.data?.find(w => w.isActive);

    return {
        data: query.data,
        isLoading: query.isLoading,
        error: query.error as Error | null,
        activeWallet,

    };
}
