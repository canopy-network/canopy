import React from "react";
import { useAccounts } from "@/app/providers/AccountsProvider";
import { useDS } from "@/core/useDs";

const DEFAULT_PER_PAGE = 20;
const DEFAULT_POLL_INTERVAL_MS = 6000;

export type RpcOrder = {
  id: string;
  committee: number;
  data?: string;
  amountForSale: number;
  requestedAmount: number;
  sellerReceiveAddress: string;
  buyerSendAddress?: string;
  buyerChainDeadline?: number;
  sellersSendAddress: string;
};

type OrdersResponse = {
  pageNumber?: number;
  perPage?: number;
  results?: RpcOrder[];
  type?: string;
  count?: number;
  totalPages?: number;
  totalCount?: number;
};

type AdminConfigResponse = {
  chainId?: number | string;
};

const toSafeInt = (value: unknown): number | undefined => {
  const n = Number(value);
  if (!Number.isFinite(n)) return undefined;
  return Math.trunc(n);
};

const asList = (payload: OrdersResponse | undefined): RpcOrder[] =>
  Array.isArray(payload?.results) ? payload!.results! : [];

export const isOrderLocked = (order: RpcOrder): boolean =>
  !!String(order?.buyerSendAddress ?? "").trim();

export function useOrdersData(options?: {
  perPage?: number;
  pollIntervalMs?: number;
}) {
  const perPage = options?.perPage ?? DEFAULT_PER_PAGE;
  const pollIntervalMs = options?.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;

  const { selectedAddress, isReady: accountsReady } = useAccounts();

  const configQ = useDS<AdminConfigResponse>(
    "admin.config",
    {},
    {
      enabled: accountsReady,
      staleTimeMs: 5000,
      refetchIntervalMs: 10000,
      refetchOnWindowFocus: false,
    },
  );

  const committeeId = React.useMemo(
    () => toSafeInt(configQ.data?.chainId),
    [configQ.data],
  );

  const hasCommittee = typeof committeeId === "number" && committeeId > 0;
  const hasSelectedAddress = !!selectedAddress;

  const myOrdersQ = useDS<OrdersResponse>(
    "orders.bySeller",
    {
      account: { address: selectedAddress },
      page: 1,
      perPage,
    },
    {
      enabled: hasSelectedAddress && accountsReady,
      staleTimeMs: pollIntervalMs,
      refetchIntervalMs: pollIntervalMs,
      refetchOnWindowFocus: false,
    },
  );

  const availableOrdersQ = useDS<OrdersResponse>(
    "orders.byCommittee",
    {
      committee: committeeId,
      page: 1,
      perPage,
    },
    {
      enabled: hasCommittee,
      staleTimeMs: pollIntervalMs,
      refetchIntervalMs: pollIntervalMs,
      refetchOnWindowFocus: false,
    },
  );

  const fulfillOrdersQ = useDS<OrdersResponse>(
    "orders.byBuyer",
    {
      account: { address: selectedAddress },
      committee: committeeId,
      page: 1,
      perPage,
    },
    {
      enabled: hasSelectedAddress && hasCommittee,
      staleTimeMs: pollIntervalMs,
      refetchIntervalMs: pollIntervalMs,
      refetchOnWindowFocus: false,
    },
  );

  const myOrders = React.useMemo(() => asList(myOrdersQ.data), [myOrdersQ.data]);

  const availableOrders = React.useMemo(() => {
    const raw = asList(availableOrdersQ.data);
    if (!selectedAddress) return raw.filter((order) => !isOrderLocked(order));

    return raw.filter((order) => {
      const unlocked = !isOrderLocked(order);
      const isOwnOrder =
        String(order?.sellersSendAddress ?? "").toLowerCase() ===
        selectedAddress.toLowerCase();
      return unlocked && !isOwnOrder;
    });
  }, [availableOrdersQ.data, selectedAddress]);

  const fulfillOrders = React.useMemo(
    () => asList(fulfillOrdersQ.data),
    [fulfillOrdersQ.data],
  );

  const isLoadingAny =
    configQ.isLoading ||
    (hasSelectedAddress && myOrdersQ.isLoading) ||
    (hasCommittee && availableOrdersQ.isLoading) ||
    (hasSelectedAddress && hasCommittee && fulfillOrdersQ.isLoading);

  const hasAnyError =
    !!configQ.error || !!myOrdersQ.error || !!availableOrdersQ.error || !!fulfillOrdersQ.error;

  const refetchAll = React.useCallback(async () => {
    await Promise.all([
      configQ.refetch(),
      myOrdersQ.refetch(),
      availableOrdersQ.refetch(),
      fulfillOrdersQ.refetch(),
    ]);
  }, [configQ, myOrdersQ, availableOrdersQ, fulfillOrdersQ]);

  return {
    selectedAddress,
    committeeId,
    hasCommittee,
    hasSelectedAddress,
    myOrders,
    availableOrders,
    fulfillOrders,
    queries: {
      config: configQ,
      myOrders: myOrdersQ,
      availableOrders: availableOrdersQ,
      fulfillOrders: fulfillOrdersQ,
    },
    isLoadingAny,
    hasAnyError,
    refetchAll,
  };
}

