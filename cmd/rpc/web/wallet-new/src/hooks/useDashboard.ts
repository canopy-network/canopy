import {useDSInfinite} from "@/core/useDSInfinite";
import React, {useCallback, useMemo} from "react";
import {Transaction} from "@/components/dashboard/RecentTransactionsCard";
import {useAccounts} from "@/app/providers/AccountsProvider";
import {useManifest} from "@/hooks/useManifest";
import {Action as ManifestAction} from "@/manifest/types";

export const useDashboard = () => {
    const [isActionModalOpen, setIsActionModalOpen] = React.useState(false);
    const [selectedActions, setSelectedActions] = React.useState<ManifestAction[]>([]);
    const [prefilledData, setPrefilledData] = React.useState<Record<string, any> | undefined>(undefined);
    const { manifest ,loading: manifestLoading } = useManifest();


    const { selectedAddress, isReady: isAccountReady } = useAccounts()


    const txSentQuery = useDSInfinite<any[]>(
        'txs.sent',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 60_000,
            perPage: 20,
        }
    )

    const txReceivedQuery = useDSInfinite<any[]>(
        'txs.received',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 60_000,
            perPage: 20,
        }
    )

    const txFailedQuery = useDSInfinite<any[]>(
        'txs.failed',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 60_000,
            perPage: 20,
        }
    )


    const isTxLoading = txSentQuery.isLoading || txReceivedQuery.isLoading || txFailedQuery.isLoading;

    const hasMoreTxs =
        (txSentQuery.hasNextPage ?? false) ||
        (txReceivedQuery.hasNextPage ?? false) ||
        (txFailedQuery.hasNextPage ?? false);

    const isFetchingMoreTxs =
        txSentQuery.isFetchingNextPage ||
        txReceivedQuery.isFetchingNextPage ||
        txFailedQuery.isFetchingNextPage;

    const fetchMoreTxs = useCallback(async () => {
        const promises: Promise<any>[] = [];
        if (txSentQuery.hasNextPage) promises.push(txSentQuery.fetchNextPage());
        if (txReceivedQuery.hasNextPage) promises.push(txReceivedQuery.fetchNextPage());
        if (txFailedQuery.hasNextPage) promises.push(txFailedQuery.fetchNextPage());
        if (promises.length > 0) await Promise.all(promises);
    }, [txSentQuery, txReceivedQuery, txFailedQuery]);

    // Total count hint from the first page of txs.sent (most reliable source for totalCount)
    const serverTotalCount = useMemo(() => {
        const raw = txSentQuery.data?.pages?.[0]?.raw;
        return typeof raw?.totalCount === 'number' ? raw.totalCount : undefined;
    }, [txSentQuery.data]);

    const allTxs = useMemo(() => {
        const toTx = (i: any, typeOverride?: string, statusOverride?: string): Transaction => ({
            hash: String(i.txHash ?? ''),
            type: typeOverride ?? i.transaction?.type ?? '',
            amount: i.transaction?.msg?.amount ?? 0,
            fee: i.transaction?.fee,
            status: statusOverride ?? i.transaction?.status ?? 'Confirmed',
            time: i.transaction?.time,
            address: i.address,
            error: i.error ?? undefined,
        });

        const received = (txReceivedQuery.data?.pages.flatMap(p => p.items) ?? [])
            .map(i => toTx(i, 'receive'));

        const sent = (txSentQuery.data?.pages.flatMap(p => p.items) ?? [])
            .map(i => toTx(i));

        const failed = (txFailedQuery.data?.pages.flatMap(p => p.items) ?? [])
            .map(i => toTx(i, undefined, 'Failed'));

        // Deduplicate by txHash — priority: failed > sent > received (last write wins)
        const byHash = new Map<string, Transaction>();
        for (const tx of [...received, ...sent, ...failed]) {
            if (tx.hash) byHash.set(tx.hash, tx);
        }

        return Array.from(byHash.values()).sort((a, b) => b.time - a.time);

    }, [txSentQuery.data, txReceivedQuery.data, txFailedQuery.data])

    const onRunAction = (action: ManifestAction, actionPrefilledData?: Record<string, any>) => {
        const actions = [action] ;
        if (action.relatedActions) {
            const relatedActions = manifest?.actions.filter(a => action?.relatedActions?.includes(a.id))

            if (relatedActions)
                actions.push(...relatedActions)
        }
        setSelectedActions(actions);
        setPrefilledData(actionPrefilledData);
        setIsActionModalOpen(true);
    }

    // Clear prefilledData when modal closes
    const handleCloseModal = React.useCallback(() => {
        setIsActionModalOpen(false);
        setPrefilledData(undefined);
    }, []);

    return {
       isActionModalOpen,
       setIsActionModalOpen: handleCloseModal,
       selectedActions,
       setSelectedActions,
       manifest,
       manifestLoading,
       isTxLoading,
       allTxs,
       onRunAction,
       prefilledData,
       hasMoreTxs,
       isFetchingMoreTxs,
       fetchMoreTxs,
       serverTotalCount,
    }
}