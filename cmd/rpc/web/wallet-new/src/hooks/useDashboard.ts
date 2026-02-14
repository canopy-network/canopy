import {useDSInfinite} from "@/core/useDSInfinite";
import React, {useMemo} from "react";
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
            refetchIntervalMs: 60_000,
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
            refetchIntervalMs: 60_000,
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
                        // @ts-ignore
                        ...i.transaction,
                    },
                }))
            ) ?? [];

        const received =
            txReceivedQuery.data?.pages.flatMap(p =>
                p.items.map(i => ({
                    ...i,
                    transaction: {
                        // @ts-ignore
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
                        // @ts-ignore
                        ...i.transaction,
                        type: 'stake',
                        status: 'Failed',
                    },

                }))
            ) ?? [];


        const mergedTxs = [...sent, ...received, ...failed]

        return mergedTxs.map(tx => {
            return {
                // @ts-ignore
                hash: String(tx.txHash ?? ''),
                type: tx.transaction.type,
                amount: tx.transaction.msg.amount ?? 0,
                fee: tx.transaction.fee,
                //TODO: CHECK HOW TO GET THIS VALUE
                status:  tx.transaction.status ?? 'Confirmed',
                time: tx?.transaction?.time,
                // @ts-ignore
                address: tx.address,
            } as Transaction;
        }).sort((a, b) => b.time - a.time);

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
    }
}