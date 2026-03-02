import { useDSInfinite } from "@/core/useDSInfinite";
import React, { useCallback, useMemo } from "react";
import { Transaction } from "@/components/dashboard/RecentTransactionsCard";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useManifest } from "@/hooks/useManifest";
import { Action as ManifestAction } from "@/manifest/types";
import { useConfig } from "@/app/providers/ConfigProvider";

const TX_POLL_INTERVAL_MS = 6000;
const TX_PER_PAGE = 20;

const parseMemoJson = (memo: unknown): Record<string, any> | null => {
  if (typeof memo !== "string" || memo.trim() === "") return null;
  try {
    const parsed = JSON.parse(memo);
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch {
    return null;
  }
};

const inferTxType = (row: any): string => {
  const type = String(row?.transaction?.type ?? row?.messageType ?? "");
  if (type && type !== "send") return type;

  const memo = parseMemoJson(row?.transaction?.memo);
  if (memo) {
    if (memo.closeOrder === true && memo.orderId) return "closeOrder";

    const hasLockOrderShape =
      !!memo.orderId &&
      (memo.buyerSendAddress != null ||
        memo.buyerReceiveAddress != null ||
        memo.buyerChainDeadline != null);
    if (hasLockOrderShape) return "lockOrder";

    if (memo.votePoll != null) return "votePoll";
  }

  return type || "send";
};

const normalizeStatus = (status: unknown, fallback = "Confirmed"): string => {
  const raw = String(status ?? fallback).toLowerCase();
  if (raw === "failed" || raw === "error") return "Failed";
  if (raw === "included" || raw === "confirmed" || raw === "success") return "Confirmed";
  return fallback;
};

const toNumber = (value: unknown): number => {
  const n = Number(value ?? 0);
  return Number.isFinite(n) ? n : 0;
};

export const useDashboard = () => {
  const [isActionModalOpen, setIsActionModalOpen] = React.useState(false);
  const [selectedActions, setSelectedActions] = React.useState<ManifestAction[]>([]);
  const [prefilledData, setPrefilledData] = React.useState<Record<string, any> | undefined>(undefined);
  const { manifest, loading: manifestLoading } = useManifest();
  const { chain } = useConfig();
  const { selectedAddress, isReady: isAccountReady } = useAccounts();

  const hasDistinctRootRpc = useMemo(() => {
    const base = String(chain?.rpc?.base ?? "").trim();
    const root = String(chain?.rpc?.root ?? "").trim();
    return !!root && root !== base;
  }, [chain?.rpc?.base, chain?.rpc?.root]);

  const txCtx = { account: { address: selectedAddress } };

  const txSentQuery = useDSInfinite<any[]>("txs.sent", txCtx, {
    enabled: !!selectedAddress && isAccountReady,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const txReceivedQuery = useDSInfinite<any[]>("txs.received", txCtx, {
    enabled: !!selectedAddress && isAccountReady,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const txFailedQuery = useDSInfinite<any[]>("txs.failed", txCtx, {
    enabled: !!selectedAddress && isAccountReady,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const txRootSentQuery = useDSInfinite<any[]>("txs.root.sent", txCtx, {
    enabled: !!selectedAddress && isAccountReady && hasDistinctRootRpc,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const txRootReceivedQuery = useDSInfinite<any[]>("txs.root.received", txCtx, {
    enabled: !!selectedAddress && isAccountReady && hasDistinctRootRpc,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const txRootFailedQuery = useDSInfinite<any[]>("txs.root.failed", txCtx, {
    enabled: !!selectedAddress && isAccountReady && hasDistinctRootRpc,
    refetchIntervalMs: TX_POLL_INTERVAL_MS,
    perPage: TX_PER_PAGE,
  });

  const isTxLoading =
    txSentQuery.isLoading ||
    txReceivedQuery.isLoading ||
    txFailedQuery.isLoading ||
    (hasDistinctRootRpc &&
      (txRootSentQuery.isLoading ||
        txRootReceivedQuery.isLoading ||
        txRootFailedQuery.isLoading));

  const hasMoreTxs =
    (txSentQuery.hasNextPage ?? false) ||
    (txReceivedQuery.hasNextPage ?? false) ||
    (txFailedQuery.hasNextPage ?? false) ||
    (txRootSentQuery.hasNextPage ?? false) ||
    (txRootReceivedQuery.hasNextPage ?? false) ||
    (txRootFailedQuery.hasNextPage ?? false);

  const isFetchingMoreTxs =
    txSentQuery.isFetchingNextPage ||
    txReceivedQuery.isFetchingNextPage ||
    txFailedQuery.isFetchingNextPage ||
    txRootSentQuery.isFetchingNextPage ||
    txRootReceivedQuery.isFetchingNextPage ||
    txRootFailedQuery.isFetchingNextPage;

  const fetchMoreTxs = useCallback(async () => {
    const promises: Promise<any>[] = [];

    if (txSentQuery.hasNextPage) promises.push(txSentQuery.fetchNextPage());
    if (txReceivedQuery.hasNextPage) promises.push(txReceivedQuery.fetchNextPage());
    if (txFailedQuery.hasNextPage) promises.push(txFailedQuery.fetchNextPage());

    if (txRootSentQuery.hasNextPage) promises.push(txRootSentQuery.fetchNextPage());
    if (txRootReceivedQuery.hasNextPage) promises.push(txRootReceivedQuery.fetchNextPage());
    if (txRootFailedQuery.hasNextPage) promises.push(txRootFailedQuery.fetchNextPage());

    if (promises.length > 0) await Promise.all(promises);
  }, [
    txSentQuery,
    txReceivedQuery,
    txFailedQuery,
    txRootSentQuery,
    txRootReceivedQuery,
    txRootFailedQuery,
  ]);

  const serverTotalCount = useMemo(() => {
    const localRaw = txSentQuery.data?.pages?.[0]?.raw;
    const rootRaw = txRootSentQuery.data?.pages?.[0]?.raw;
    const localCount = typeof localRaw?.totalCount === "number" ? localRaw.totalCount : undefined;
    const rootCount = typeof rootRaw?.totalCount === "number" ? rootRaw.totalCount : undefined;

    if (typeof localCount === "number" && typeof rootCount === "number") {
      return Math.max(localCount, rootCount);
    }
    return localCount ?? rootCount;
  }, [txSentQuery.data, txRootSentQuery.data]);

  const allTxs = useMemo(() => {
    const makeTx = (
      i: any,
      overrides?: {
        type?: string;
        status?: string;
      },
    ): Transaction => ({
      hash: String(i.txHash ?? i.hash ?? ""),
      type: overrides?.type ?? inferTxType(i),
      amount: toNumber(i.transaction?.msg?.amount),
      fee: i.transaction?.fee,
      status: normalizeStatus(overrides?.status ?? i.transaction?.status, "Confirmed"),
      time: toNumber(i.transaction?.time ?? i.time),
      address: i.address ?? i.sender,
      error: i.error ?? undefined,
    });

    const localReceived = (txReceivedQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) =>
      makeTx(i, { type: "receive" }),
    );
    const localSent = (txSentQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) => makeTx(i));
    const localFailed = (txFailedQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) =>
      makeTx(i, { status: "Failed" }),
    );

    const rootReceived = (txRootReceivedQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) =>
      makeTx(i, { type: "receive" }),
    );
    const rootSent = (txRootSentQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) => makeTx(i));
    const rootFailed = (txRootFailedQuery.data?.pages.flatMap((p) => p.items) ?? []).map((i) =>
      makeTx(i, { status: "Failed" }),
    );

    // Deduplicate by hash. Priority (last write wins): failed > sent > received.
    const byHash = new Map<string, Transaction>();
    for (const tx of [
      ...localReceived,
      ...rootReceived,
      ...localSent,
      ...rootSent,
      ...localFailed,
      ...rootFailed,
    ]) {
      if (tx.hash) byHash.set(tx.hash, tx);
    }

    return Array.from(byHash.values()).sort((a, b) => b.time - a.time);
  }, [
    txSentQuery.data,
    txReceivedQuery.data,
    txFailedQuery.data,
    txRootSentQuery.data,
    txRootReceivedQuery.data,
    txRootFailedQuery.data,
  ]);

  const onRunAction = (action: ManifestAction, actionPrefilledData?: Record<string, any>) => {
    const actions = [action];
    if (action.relatedActions) {
      const relatedActions = manifest?.actions.filter((a) => action?.relatedActions?.includes(a.id));

      if (relatedActions) actions.push(...relatedActions);
    }
    setSelectedActions(actions);
    setPrefilledData(actionPrefilledData);
    setIsActionModalOpen(true);
  };

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
  };
};
