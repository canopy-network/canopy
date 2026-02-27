import React from "react";
import { motion } from "framer-motion";
import {
  AlertTriangle,
  ArrowLeftRight,
  CheckCircle2,
  CircleDashed,
  Droplets,
  Info,
  Loader2,
  Lock,
  Pencil,
  PlusCircle,
  RefreshCcw,
  ShoppingCart,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useConfig } from "@/app/providers/ConfigProvider";
import { isOrderLocked, RpcOrder, useOrdersData } from "@/hooks/useOrdersData";
import { cx } from "@/ui/cx";

const POLL_INTERVAL_MS = 6000;
const PER_PAGE = 20;

const ACTION_IDS = {
  createOrder: "orderCreate",
  repriceOrder: "orderReprice",
  voidOrder: "orderVoid",
  lockOrder: "orderLock",
  closeOrder: "orderClose",
  dexLimitOrder: "dexLimitOrder",
  dexLiquidityDeposit: "dexLiquidityDeposit",
  dexLiquidityWithdraw: "dexLiquidityWithdraw",
} as const;

const shortHex = (value: string, head = 6, tail = 4) => {
  const v = String(value ?? "");
  if (!v) return "-";
  if (v.length <= head + tail + 2) return v;
  return `${v.slice(0, head)}...${v.slice(-tail)}`;
};

const asNumber = (value: unknown): number => {
  const n = Number(value ?? 0);
  return Number.isFinite(n) ? n : 0;
};

const microToDisplay = (value: unknown, decimals: number): number =>
  asNumber(value) / Math.pow(10, decimals);

const formatAmount = (value: unknown, decimals: number, symbol?: string): string => {
  const normalized = microToDisplay(value, decimals);
  const amount = normalized.toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 6,
  });
  return symbol ? `${amount} ${symbol}` : amount;
};

const getErrorMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message;
  return "Could not load data from RPC.";
};

type SectionCardProps = {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
  children: React.ReactNode;
};

const SectionCard: React.FC<SectionCardProps> = ({ title, subtitle, actions, children }) => (
  <section className="canopy-card p-5 md:p-6">
    <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
      <div>
        <h2 className="text-lg font-semibold text-foreground">{title}</h2>
        {subtitle ? <p className="text-xs text-muted-foreground mt-1">{subtitle}</p> : null}
      </div>
      {actions}
    </div>
    {children}
  </section>
);

const TableShell: React.FC<{
  isLoading: boolean;
  error?: unknown;
  isEmpty: boolean;
  emptyText: string;
  children: React.ReactNode;
}> = ({ isLoading, error, isEmpty, emptyText, children }) => {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-10 text-muted-foreground text-sm">
        <Loader2 className="w-4 h-4 animate-spin mr-2" />
        Loading orders...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-500/30 bg-red-500/5 p-3 text-sm text-red-300">
        {getErrorMessage(error)}
      </div>
    );
  }

  if (isEmpty) {
    return <div className="text-sm text-muted-foreground py-10 text-center">{emptyText}</div>;
  }

  return <>{children}</>;
};

type InfoMessageProps = {
  title: string;
  description: string;
  tone?: "info" | "warning" | "success";
  meta?: React.ReactNode;
};

const InfoMessage: React.FC<InfoMessageProps> = ({
  title,
  description,
  tone = "info",
  meta,
}) => {
  const toneClass =
    tone === "warning"
      ? "border-amber-500/30 bg-amber-500/[0.06]"
      : tone === "success"
        ? "border-emerald-500/30 bg-emerald-500/[0.06]"
        : "border-blue-500/30 bg-blue-500/[0.06]";

  return (
    <div className={cx("rounded-lg border px-3 py-2.5 mb-4", toneClass)}>
      <div className="flex items-start gap-2">
        {tone === "warning" ? (
          <AlertTriangle className="w-4 h-4 mt-0.5 text-amber-400 shrink-0" />
        ) : (
          <Info className="w-4 h-4 mt-0.5 text-blue-400 shrink-0" />
        )}
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{title}</p>
          <p className="text-xs text-muted-foreground">{description}</p>
          {meta ? <div className="flex flex-wrap items-center gap-2 mt-2">{meta}</div> : null}
        </div>
      </div>
    </div>
  );
};

type OrderTypeMetricProps = {
  title: string;
  hint: string;
  count: number;
  status: React.ComponentProps<typeof StatusBadge>["status"];
  icon: React.ReactNode;
};

const OrderTypeMetric: React.FC<OrderTypeMetricProps> = ({
  title,
  hint,
  count,
  status,
  icon,
}) => (
  <div className="rounded-lg border border-border/60 bg-background/70 p-3">
    <div className="flex items-center justify-between gap-2">
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-muted-foreground shrink-0">{icon}</span>
        <p className="text-xs uppercase tracking-wider text-muted-foreground truncate">{title}</p>
      </div>
      <StatusBadge label={String(count)} status={status} size="sm" />
    </div>
    <p className="text-xs text-muted-foreground mt-2">{hint}</p>
  </div>
);

export default function Orders(): JSX.Element {
  const { chain } = useConfig();
  const { openAction } = useActionModal();

  const {
    selectedAddress,
    committeeId,
    myOrders,
    availableOrders,
    fulfillOrders,
    queries,
    refetchAll,
  } = useOrdersData({
    perPage: PER_PAGE,
    pollIntervalMs: POLL_INTERVAL_MS,
  });

  const symbol = String(chain?.denom?.symbol ?? "CNPY");
  const decimals = asNumber(chain?.denom?.decimals) || 6;

  const runAction = React.useCallback(
    (actionId: string, prefilledData?: Record<string, unknown>) => {
      openAction(actionId, {
        prefilledData,
        onFinish: () => {
          void refetchAll();
        },
      });
    },
    [openAction, refetchAll],
  );

  const baseOrderPrefill = React.useCallback(
    (order: RpcOrder) => ({
      address: selectedAddress || order.sellersSendAddress || "",
      receiveAddress: order.sellerReceiveAddress || "",
      committees: String(order.committee ?? committeeId ?? ""),
      orderId: order.id,
      amount: microToDisplay(order.amountForSale, decimals),
      receiveAmount: microToDisplay(order.requestedAmount, decimals),
      data: order.data || "",
      fee: 0,
      memo: "",
    }),
    [selectedAddress, committeeId, decimals],
  );

  const dexPrefill = React.useMemo(
    () => ({
      address: selectedAddress || "",
      committees: String(committeeId ?? ""),
      fee: 0,
      memo: "",
    }),
    [selectedAddress, committeeId],
  );

  const myLockedCount = React.useMemo(
    () => myOrders.filter((order) => isOrderLocked(order)).length,
    [myOrders],
  );

  const myOpenCount = myOrders.length - myLockedCount;

  const nextFulfillDeadline = React.useMemo(() => {
    let next: number | undefined;
    for (const order of fulfillOrders) {
      const deadline = asNumber(order.buyerChainDeadline || 0);
      if (!deadline) continue;
      if (next == null || deadline < next) next = deadline;
    }
    return next;
  }, [fulfillOrders]);

  return (
    <motion.div
      className="min-h-screen bg-background"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="px-6 py-8 space-y-6">
        <section className="canopy-card p-5 md:p-6">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2 mb-2">
                <ShoppingCart className="w-5 h-5 text-primary" />
                <h1 className="text-2xl font-semibold text-foreground">Orders</h1>
              </div>
              <p className="text-sm text-muted-foreground">
                Manage sell orders, lock available offers, close pending purchases, and execute DEX
                operations from one module.
              </p>
            </div>
            <Button variant="outline" size="sm" onClick={() => void refetchAll()}>
              <RefreshCcw className="w-4 h-4" />
              Refresh
            </Button>
          </div>

          <div className="flex flex-wrap items-center gap-2 mt-4">
            <StatusBadge label={`Committee ${committeeId ?? "-"}`} status="info" size="sm" />
            <StatusBadge label={`Polling ${Math.round(POLL_INTERVAL_MS / 1000)}s`} status="live" size="sm" pulse />
            <StatusBadge
              label={selectedAddress ? shortHex(selectedAddress, 8, 6) : "No selected account"}
              status={selectedAddress ? "active" : "warning"}
              size="sm"
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-2 mt-4">
            <OrderTypeMetric
              title="Seller Open"
              hint="Can be repriced or voided."
              count={myOpenCount}
              status="active"
              icon={<Pencil className="w-4 h-4" />}
            />
            <OrderTypeMetric
              title="Seller Locked"
              hint="Locked by buyer, waiting close."
              count={myLockedCount}
              status="warning"
              icon={<Lock className="w-4 h-4" />}
            />
            <OrderTypeMetric
              title="Buy Opportunities"
              hint="Orders available to lock."
              count={availableOrders.length}
              status="info"
              icon={<ShoppingCart className="w-4 h-4" />}
            />
            <OrderTypeMetric
              title="To Fulfill"
              hint="Locked by you, pending close."
              count={fulfillOrders.length}
              status="warning"
              icon={<CheckCircle2 className="w-4 h-4" />}
            />
          </div>
        </section>

        <div className="grid grid-cols-1 xl:grid-cols-12 gap-6">
          <div className="xl:col-span-8 space-y-6">
            <section className="canopy-card p-5 md:p-6">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-semibold text-foreground">Cross-Chain Orderbook</h2>
                  <p className="text-xs text-muted-foreground mt-1">
                    Seller and buyer lifecycle on committee orders: create, lock, reprice, void, and close.
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge label="Root-chain seller ops" status="info" size="sm" />
                  <StatusBadge label="Nested-chain buyer ops" status="warning" size="sm" />
                </div>
              </div>
            </section>

        <SectionCard
          title="My Orders"
          subtitle="Orders created by the selected address (seller)."
          actions={
            <Button
              size="sm"
              onClick={() =>
                runAction(ACTION_IDS.createOrder, {
                  address: selectedAddress || "",
                  receiveAddress: selectedAddress || "",
                  committees: String(committeeId ?? ""),
                  fee: 0,
                })
              }
              disabled={!selectedAddress}
            >
              <PlusCircle className="w-4 h-4" />
              Create Order
            </Button>
          }
        >
          <InfoMessage
            title="Seller Order Types"
            description="Open orders can be repriced or voided. Once locked by a buyer, seller actions are disabled until buyer closes."
            tone="info"
            meta={
              <>
                <StatusBadge label={`${myOpenCount} open`} status="active" size="sm" />
                <StatusBadge label={`${myLockedCount} locked`} status="warning" size="sm" />
              </>
            }
          />
          <TableShell
            isLoading={queries.myOrders.isLoading}
            error={queries.myOrders.error}
            isEmpty={!myOrders.length}
            emptyText={selectedAddress ? "No orders created yet." : "Select an account to load orders."}
          >
            <div className="overflow-x-auto">
              <table className="w-full min-w-[920px]">
                <thead className="text-left text-xs uppercase tracking-wider text-muted-foreground border-b border-border/60">
                  <tr>
                    <th className="py-2 pr-3">Order ID</th>
                    <th className="py-2 pr-3">Type</th>
                    <th className="py-2 pr-3">Committee</th>
                    <th className="py-2 pr-3">Amount For Sale</th>
                    <th className="py-2 pr-3">Requested</th>
                    <th className="py-2 pr-3">Status</th>
                    <th className="py-2 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {myOrders.map((order) => {
                    const locked = isOrderLocked(order);
                    return (
                      <tr
                        key={order.id}
                        className={cx(
                          "border-b border-border/40 last:border-b-0",
                          locked ? "bg-amber-500/[0.03]" : "bg-emerald-500/[0.03]",
                        )}
                      >
                        <td className="py-3 pr-3 font-mono text-sm text-foreground">
                          {shortHex(order.id, 8, 6)}
                        </td>
                        <td className="py-3 pr-3">
                          <StatusBadge
                            label={locked ? "Seller Locked" : "Seller Open"}
                            status={locked ? "warning" : "active"}
                            size="sm"
                          />
                        </td>
                        <td className="py-3 pr-3 text-sm text-foreground">{order.committee}</td>
                        <td className="py-3 pr-3 text-sm text-foreground">
                          {formatAmount(order.amountForSale, decimals, symbol)}
                        </td>
                        <td className="py-3 pr-3 text-sm text-foreground">
                          {formatAmount(order.requestedAmount, decimals)}
                        </td>
                        <td className="py-3 pr-3">
                          <StatusBadge
                            label={locked ? "Locked" : "Available"}
                            status={locked ? "warning" : "active"}
                            size="sm"
                          />
                        </td>
                        <td className="py-3 text-right">
                          {!locked ? (
                            <div className="inline-flex items-center gap-2">
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => runAction(ACTION_IDS.repriceOrder, baseOrderPrefill(order))}
                              >
                                <Pencil className="w-3.5 h-3.5" />
                                Reprice
                              </Button>
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() =>
                                  runAction(ACTION_IDS.voidOrder, {
                                    address: selectedAddress || order.sellersSendAddress || "",
                                    committees: String(order.committee ?? committeeId ?? ""),
                                    orderId: order.id,
                                    fee: 0,
                                  })
                                }
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                                Void
                              </Button>
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">Waiting buyer close</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </TableShell>
        </SectionCard>

        <SectionCard
          title="Available Orders"
          subtitle={`Unlocked orders on committee ${committeeId ?? "-"}.`}
        >
          <InfoMessage
            title="Buy Opportunity Orders"
            description="Only unlocked orders from other sellers are listed here. Lock reserves the order using your selected nested-chain buyer account."
            tone="info"
            meta={<StatusBadge label={`${availableOrders.length} lockable`} status="info" size="sm" />}
          />
          <TableShell
            isLoading={queries.availableOrders.isLoading}
            error={queries.availableOrders.error}
            isEmpty={!availableOrders.length}
            emptyText="No available orders on this committee."
          >
            <div className="overflow-x-auto">
              <table className="w-full min-w-[900px]">
                <thead className="text-left text-xs uppercase tracking-wider text-muted-foreground border-b border-border/60">
                  <tr>
                    <th className="py-2 pr-3">Order ID</th>
                    <th className="py-2 pr-3">Type</th>
                    <th className="py-2 pr-3">Seller</th>
                    <th className="py-2 pr-3">For Sale</th>
                    <th className="py-2 pr-3">Requested</th>
                    <th className="py-2 pr-3">Receive Address</th>
                    <th className="py-2 text-right">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {availableOrders.map((order) => (
                    <tr key={order.id} className="border-b border-border/40 last:border-b-0 bg-blue-500/[0.03]">
                      <td className="py-3 pr-3 font-mono text-sm text-foreground">
                        {shortHex(order.id, 8, 6)}
                      </td>
                      <td className="py-3 pr-3">
                        <StatusBadge label="Buy Opportunity" status="info" size="sm" />
                      </td>
                      <td className="py-3 pr-3 font-mono text-sm text-foreground">
                        {shortHex(order.sellersSendAddress, 8, 6)}
                      </td>
                      <td className="py-3 pr-3 text-sm text-foreground">
                        {formatAmount(order.amountForSale, decimals, symbol)}
                      </td>
                      <td className="py-3 pr-3 text-sm text-foreground">
                        {formatAmount(order.requestedAmount, decimals)}
                      </td>
                      <td className="py-3 pr-3 font-mono text-sm text-muted-foreground">
                        {shortHex(order.sellerReceiveAddress, 8, 6)}
                      </td>
                      <td className="py-3 text-right">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() =>
                            runAction(ACTION_IDS.lockOrder, {
                              address: selectedAddress || "",
                              receiveAddress: selectedAddress || "",
                              orderId: order.id,
                              fee: 0,
                            })
                          }
                          disabled={!selectedAddress}
                        >
                          <Lock className="w-3.5 h-3.5" />
                          Lock
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </TableShell>
        </SectionCard>

        <SectionCard
          title="Orders To Fulfill"
          subtitle="Orders locked by the selected address (buyer) pending close."
        >
          <InfoMessage
            title="Pending Buyer Closures"
            description="Close before deadline and ensure balance for requested amount. Close can fail if address is not the lock buyer or if deadline is too close."
            tone="warning"
            meta={
              <>
                <StatusBadge label={`${fulfillOrders.length} pending close`} status="warning" size="sm" />
                {nextFulfillDeadline ? (
                  <StatusBadge
                    label={`next deadline ${nextFulfillDeadline.toLocaleString()}`}
                    status="info"
                    size="sm"
                  />
                ) : null}
              </>
            }
          />
          <TableShell
            isLoading={queries.fulfillOrders.isLoading}
            error={queries.fulfillOrders.error}
            isEmpty={!fulfillOrders.length}
            emptyText="No pending orders to fulfill."
          >
            <div className="overflow-x-auto">
              <table className="w-full min-w-[980px]">
                <thead className="text-left text-xs uppercase tracking-wider text-muted-foreground border-b border-border/60">
                  <tr>
                    <th className="py-2 pr-3">Order ID</th>
                    <th className="py-2 pr-3">Type</th>
                    <th className="py-2 pr-3">Seller</th>
                    <th className="py-2 pr-3">Amount To Receive</th>
                    <th className="py-2 pr-3">Amount To Send</th>
                    <th className="py-2 pr-3">Deadline</th>
                    <th className="py-2 pr-3">Send To</th>
                    <th className="py-2 text-right">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {fulfillOrders.map((order) => (
                    <tr key={order.id} className="border-b border-border/40 last:border-b-0 bg-amber-500/[0.03]">
                      <td className="py-3 pr-3 font-mono text-sm text-foreground">
                        {shortHex(order.id, 8, 6)}
                      </td>
                      <td className="py-3 pr-3">
                        <StatusBadge label="Buyer Locked" status="warning" size="sm" />
                      </td>
                      <td className="py-3 pr-3 font-mono text-sm text-foreground">
                        {shortHex(order.sellersSendAddress, 8, 6)}
                      </td>
                      <td className="py-3 pr-3 text-sm text-foreground">
                        {formatAmount(order.amountForSale, decimals, symbol)}
                      </td>
                      <td className="py-3 pr-3 text-sm text-foreground">
                        {formatAmount(order.requestedAmount, decimals)}
                      </td>
                      <td className="py-3 pr-3 text-sm text-foreground">
                        {asNumber(order.buyerChainDeadline || 0).toLocaleString()}
                      </td>
                      <td className="py-3 pr-3 font-mono text-sm text-muted-foreground">
                        {shortHex(order.sellerReceiveAddress, 8, 6)}
                      </td>
                      <td className="py-3 text-right">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() =>
                            runAction(ACTION_IDS.closeOrder, {
                              address: selectedAddress || "",
                              orderId: order.id,
                              fee: 0,
                            })
                          }
                          disabled={!selectedAddress}
                        >
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          Close
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </TableShell>
        </SectionCard>

          </div>

          <aside className="xl:col-span-4 space-y-6 xl:sticky xl:top-[calc(var(--topbar-height,52px)+1rem)] self-start">
            <section className="canopy-card p-5 md:p-6">
              <div className="space-y-3">
                <h2 className="text-lg font-semibold text-foreground">DEX Order Lane</h2>
                <p className="text-xs text-muted-foreground">
                  Separate from cross-chain orderbook. Use this lane for pool liquidity and limit-price swaps.
                </p>
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge label="Pool operations" status="active" size="sm" />
                  <StatusBadge label="Same action modal UX" status="info" size="sm" />
                </div>
              </div>
            </section>

        <SectionCard
          title="DEX Orders"
          subtitle="Liquidity and limit-order operations against DEX pool endpoints."
        >
          <InfoMessage
            title="DEX Operation Types"
            description="Limit Order swaps with price constraint, Deposit adds pool liquidity, Withdraw removes liquidity by percentage."
            tone="success"
            meta={
              <>
                <StatusBadge label="order = swap intent" status="info" size="sm" />
                <StatusBadge label="deposit = add liquidity" status="active" size="sm" />
                <StatusBadge label="withdraw = remove liquidity" status="warning" size="sm" />
              </>
            }
          />
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Button onClick={() => runAction(ACTION_IDS.dexLimitOrder, dexPrefill)} disabled={!selectedAddress}>
              <ArrowLeftRight className="w-4 h-4" />
              Order
            </Button>
            <Button
              variant="secondary"
              onClick={() => runAction(ACTION_IDS.dexLiquidityDeposit, dexPrefill)}
              disabled={!selectedAddress}
            >
              <Droplets className="w-4 h-4" />
              Deposit
            </Button>
            <Button
              variant="outline"
              onClick={() => runAction(ACTION_IDS.dexLiquidityWithdraw, dexPrefill)}
              disabled={!selectedAddress}
            >
              <CircleDashed className="w-4 h-4" />
              Withdraw
            </Button>
          </div>
        </SectionCard>
          </aside>
        </div>
      </div>
    </motion.div>
  );
}
