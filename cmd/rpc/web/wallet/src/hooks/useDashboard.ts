import { useQuery } from "@tanstack/react-query";
import { useDSFetcher } from "@/core/dsFetch";
import React, { useCallback, useMemo } from "react";
import { Transaction } from "@/components/dashboard/RecentTransactionsCard";
import { useAccounts, useAccountsList } from "@/app/providers/AccountsProvider";
import { useManifest } from "@/hooks/useManifest";
import { Action as ManifestAction } from "@/manifest/types";


const TX_POLL_INTERVAL_MS = 6000;
const TX_PER_PAGE = 20;

const parseMemoJson = (memo: unknown): Record<string, unknown> | null => {
  if (typeof memo !== "string" || memo.trim() === "") return null;
  try {
    const parsed = JSON.parse(memo);
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch {
    return null;
  }
};

const inferTxType = (row: Record<string, unknown>): string => {
  const txn = row?.transaction as Record<string, unknown> | undefined;
  const type = String(txn?.type ?? (row as Record<string, unknown>)?.messageType ?? "");
  if (type && type !== "send") return type;

  const memo = parseMemoJson(txn?.memo);
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

const extractAmountMicro = (row: Record<string, unknown>): number => {
  for (const key of ["amount", "value", "amountForSale"]) {
    if (row[key] !== undefined) return toNumber(row[key]);
  }

  const txn = row.transaction as Record<string, unknown> | undefined;
  for (const key of ["amount", "value", "amountForSale"]) {
    if (txn?.[key] !== undefined) return toNumber(txn[key]);
  }

  const msg = txn?.msg as Record<string, unknown> | undefined;
  if (!msg) return 0;

  // Flat message shapes returned by tx endpoints for DEX/order operations.
  if (msg.amountForSale !== undefined) return toNumber(msg.amountForSale);
  if (msg.amount !== undefined) return toNumber(msg.amount);

  // Wrapped message shapes used by some query responses.
  for (const key of [
    "messageSend",
    "messageStake",
    "messageEditStake",
    "messageDAOTransfer",
    "messageSubsidy",
    "messageDexLiquidityDeposit",
  ]) {
    const inner = msg[key] as Record<string, unknown> | undefined;
    if (inner?.amount !== undefined) return toNumber(inner.amount);
  }

  for (const key of [
    "messageCreateOrder",
    "messageEditOrder",
    "messageDexLimitOrder",
  ]) {
    const inner = msg[key] as Record<string, unknown> | undefined;
    if (inner?.amountForSale !== undefined) return toNumber(inner.amountForSale);
  }

  return 0;
};

const makeTx = (
  i: Record<string, unknown>,
  overrides?: { type?: string; status?: string },
): Transaction => {
  const txn = i.transaction as Record<string, unknown> | undefined;
  return {
    hash: String(i.txHash ?? i.hash ?? ""),
    type: overrides?.type ?? inferTxType(i),
    amount: extractAmountMicro(i),
    fee: txn?.fee as number | undefined,
    status: normalizeStatus(overrides?.status ?? txn?.status, "Confirmed"),
    time: toNumber(txn?.time ?? i.time),
    address: (i.address ?? i.sender) as string | undefined,
    error: i.error as Transaction["error"],
  };
};

interface TxPage {
  results?: Record<string, unknown>[];
  txs?: Record<string, unknown>[];
  transactions?: Record<string, unknown>[];
  data?: Record<string, unknown>[];
  totalCount?: number;
  totalPages?: number;
  paging?: { totalPages?: number };
  [key: string]: unknown;
}

function extractItems(raw: TxPage | Record<string, unknown>[] | null): Record<string, unknown>[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw;
  if (Array.isArray(raw.results)) return raw.results;
  if (Array.isArray(raw.txs)) return raw.txs;
  if (Array.isArray(raw.transactions)) return raw.transactions;
  if (Array.isArray(raw.data)) return raw.data;
  return [];
}

export const useDashboard = () => {
  const [isActionModalOpen, setIsActionModalOpen] = React.useState(false);
  const [selectedActions, setSelectedActions] = React.useState<ManifestAction[]>([]);
  const [prefilledData, setPrefilledData] = React.useState<Record<string, unknown> | undefined>(undefined);
  const { manifest, loading: manifestLoading } = useManifest();
  const { isReady: isAccountReady } = useAccounts();
  const { accounts, loading: accountsLoading } = useAccountsList();
  const dsFetch = useDSFetcher();

  const allAddresses = useMemo(
    () => accounts.map((a) => a.address).filter(Boolean),
    [accounts],
  );

  const addressesKey = useMemo(
    () => allAddresses.sort().join(","),
    [allAddresses],
  );

  const txQuery = useQuery({
    queryKey: ["dashboard.allTxs", addressesKey],
    enabled: !accountsLoading && allAddresses.length > 0 && isAccountReady,
    staleTime: TX_POLL_INTERVAL_MS,
    refetchInterval: TX_POLL_INTERVAL_MS,
    queryFn: async (): Promise<Transaction[]> => {
      const byHash = new Map<string, Transaction>();

      const upsertTx = (tx: Transaction, account: string) => {
        if (!tx.hash) return;
        const existing = byHash.get(tx.hash);
        if (existing) {
          const accts = existing.relatedAccounts ?? [];
          if (!accts.includes(account)) accts.push(account);
          existing.relatedAccounts = accts;
        } else {
          tx.relatedAccounts = [account];
          byHash.set(tx.hash, tx);
        }
      };

      const fetchForAddress = async (address: string) => {
        const ctx = { account: { address }, page: 1, perPage: TX_PER_PAGE };

        const [sent, received, failed] = await Promise.all([
          dsFetch<TxPage>("txs.sent", ctx).catch(() => null),
          dsFetch<TxPage>("txs.received", ctx).catch(() => null),
          dsFetch<TxPage>("txs.failed", ctx).catch(() => null),
        ]);

        for (const item of extractItems(received as TxPage | null)) {
          upsertTx(makeTx(item, { type: "receive" }), address);
        }
        for (const item of extractItems(sent as TxPage | null)) {
          upsertTx(makeTx(item), address);
        }
        for (const item of extractItems(failed as TxPage | null)) {
          upsertTx(makeTx(item, { status: "Failed" }), address);
        }
      };

      await Promise.all(allAddresses.map(fetchForAddress));

      return Array.from(byHash.values()).sort((a, b) => b.time - a.time);
    },
  });

  const onRunAction = (action: ManifestAction, actionPrefilledData?: Record<string, unknown>) => {
    const actions = [action];
    if (action.relatedActions) {
      const relatedActions = manifest?.actions.filter((a: ManifestAction) =>
        action?.relatedActions?.includes(a.id),
      );
      if (relatedActions) actions.push(...relatedActions);
    }
    setSelectedActions(actions);
    setPrefilledData(actionPrefilledData);
    setIsActionModalOpen(true);
  };

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
    isTxLoading: txQuery.isLoading,
    allTxs: txQuery.data ?? [],
    onRunAction,
    prefilledData,
  };
};
