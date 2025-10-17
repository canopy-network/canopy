import {useDSInfinite} from "@/core/useDSInfinite";
import React, {useMemo} from "react";
import {Transaction} from "@/components/dashboard/RecentTransactionsCard";
import {useAccounts} from "@/app/providers/AccountsProvider";
import {useManifest} from "@/hooks/useManifest";
import {Action as ManifestAction} from "@/manifest/types";

export const useDashboard = () => {
    const [isActionModalOpen, setIsActionModalOpen] = React.useState(false);
    const [selectedActions, setSelectedActions] = React.useState<ManifestAction[]>([]);
    const { manifest ,loading: manifestLoading } = useManifest();


    const { selectedAddress, isReady: isAccountReady } = useAccounts()


    const txSentQuery = useDSInfinite<any[]>(
        'txs.sent',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 15_000,
            perPage: 20,
            getNextPageParam: (lastPage, pages) => {
                if (lastPage.length < 20) return undefined;
                return pages.length + 1;
            },
            selectItems: (d: any) => {
                return Array.isArray(d?.results) ? d.results : (Array.isArray(d) ? d : []);
            }

        }
    )

    const txReceivedQuery = useDSInfinite<any[]>(
        'txs.received',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 15_000,
            perPage: 20,
            getNextPageParam: (lastPage, pages) => {
                if (lastPage.length < 20) return undefined;
                return pages.length + 1;
            },
            selectItems: (d: any) => {
                return Array.isArray(d?.results) ? d.results : (Array.isArray(d) ? d : []);
            }
        }
    )

    const txFailedQuery = useDSInfinite<any[]>(
        'txs.failed',
        {account: {address: selectedAddress}},
        {
            enabled: !!selectedAddress && isAccountReady,
            refetchIntervalMs: 15_000,
            perPage: 20,
            getNextPageParam: (lastPage, pages) => {
                if (lastPage.length < 20) return undefined;
                return pages.length + 1;
            },
            selectItems: (d: any) => {
                return Array.isArray(d?.results) ? d.results : (Array.isArray(d) ? d : []);
            }
        }
    )


    const isTxLoading = txSentQuery.isLoading || txReceivedQuery.isLoading || txFailedQuery.isLoading;

    const allTxs = useMemo(() => {
        const sent =
            txSentQuery.data?.pages.flatMap(p =>
                p.items.map(i => ({
                    ...i,
                    transaction: {
                        ...i.transaction,
                        type: 'send',
                    },
                }))
            ) ?? [];

        const received =
            txReceivedQuery.data?.pages.flatMap(p =>
                p.items.map(i => ({
                    ...i,
                    transaction: {
                        ...i.transaction,
                        type: 'receive',
                    },
                }))
            ) ?? [];

        const failed =
            txFailedQuery.data?.pages.flatMap(p =>
                p.items.map(i => ({
                    ...i,
                    transaction: {
                        ...i.transaction,
                        type: i.transaction?.type ?? 'send',
                    },
                }))
            ) ?? [];

        const mergedTxs = [...sent, ...received, ...failed]

        return mergedTxs.map(tx => {
            return {
                hash: String(tx.txHash ?? ''),
                type: tx.transaction.type,
                amount: tx.transaction.msg.amount ?? 0,
                fee: tx.transaction.fee,
                //TODO: CHECK HOW TO GET THIS VALUE
                status: 'Confirmed',
                time: tx?.transaction?.time,
                address: tx.address,
            } as Transaction;
        });

    }, [txSentQuery.data, txReceivedQuery.data, txFailedQuery.data])

    const onRunAction = (action: ManifestAction) => {
        const actions = [action] ;
        if (action.relatedActions) {
            const relatedActions = manifest?.actions.filter(a => action?.relatedActions?.includes(a.id))

            if (relatedActions)
                actions.push(...relatedActions)
        }
        setSelectedActions(actions);
        setIsActionModalOpen(true);
    }

    return {
       isActionModalOpen,
       setIsActionModalOpen,
       selectedActions,
       setSelectedActions,
       manifest,
       manifestLoading,
       isTxLoading,
       allTxs,
       onRunAction,
    }
}