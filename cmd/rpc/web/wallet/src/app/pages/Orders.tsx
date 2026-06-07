import React, { useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeftRight,
  ChevronDown,
  CheckCircle2,
  CircleDashed,
  Droplets,
  Info,
  Lock,
  Pencil,
  PlusCircle,
  Trash2,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useConfig } from "@/app/providers/ConfigProvider";
import { useDS } from "@/core/useDs";
import { PageHeader } from "@/components/layouts/PageHeader";

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

type ActionCardProps = {
  title: string;
  description: string;
  icon: React.ReactNode;
  variant?: React.ComponentProps<typeof Button>["variant"];
  disabled?: boolean;
  onClick: () => void;
};

const ActionCard: React.FC<ActionCardProps> = ({
  title,
  description,
  icon,
  variant = "outline",
  disabled,
  onClick,
}) => (
  <div className="rounded-lg border border-border/60 bg-background/70 p-4 flex flex-col gap-3 h-full">
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground shrink-0">{icon}</span>
      <div className="min-w-0">
        <p className="text-sm font-medium text-foreground">{title}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
    <Button variant={variant} size="sm" disabled={disabled} onClick={onClick} className="w-full mt-auto">
      {icon}
      {title}
    </Button>
  </div>
);

type AdminConfigResponse = {
  chainId?: number | string;
};

const toSafeInt = (value: unknown): number | undefined => {
  const n = Number(value);
  if (!Number.isFinite(n)) return undefined;
  return Math.trunc(n);
};

export default function Orders(): JSX.Element {
  const { chain } = useConfig();
  const { openAction } = useActionModal();
  const [isHowItWorksOpen, setIsHowItWorksOpen] = useState(false);

  const configQ = useDS<AdminConfigResponse>("admin.config", {}, {
    staleTimeMs: 5000,
    refetchIntervalMs: 10000,
    refetchOnWindowFocus: false,
  });

  const committeeId = React.useMemo(
    () => toSafeInt(configQ.data?.chainId),
    [configQ.data],
  );

  const prefill = React.useMemo(
    () => ({
      committees: String(committeeId ?? ""),
    }),
    [committeeId],
  );

  const runAction = React.useCallback(
    (actionId: string, prefilledData?: Record<string, unknown>) => {
      openAction(actionId, { prefilledData });
    },
    [openAction],
  );

  return (
    <motion.div
      className="space-y-6"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <PageHeader
        title="Orders"
        subtitle="Create, reprice, void, lock, and close orders or execute DEX operations. Fill in the order details manually in each form."
      />

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <ActionCard
          title="AMM Limit Order"
          description="Swap with a price constraint."
          icon={<ArrowLeftRight className="w-4 h-4" />}
          variant="default"
          onClick={() => runAction(ACTION_IDS.dexLimitOrder, { ...prefill, memo: "" })}
        />
        <ActionCard
          title="AMM Deposit Liquidity"
          description="Add liquidity to the DEX pool."
          icon={<Droplets className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.dexLiquidityDeposit, { ...prefill, memo: "" })}
        />
        <ActionCard
          title="AMM Withdraw Liquidity"
          description="Remove liquidity from the DEX pool."
          icon={<CircleDashed className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.dexLiquidityWithdraw, { ...prefill, memo: "" })}
        />
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <ActionCard
          title="Orderbook Create"
          description="Create a new sell order on the committee."
          icon={<PlusCircle className="w-4 h-4" />}
          variant="default"
          onClick={() => runAction(ACTION_IDS.createOrder, prefill)}
        />
        <ActionCard
          title="Orderbook Reprice"
          description="Change the price of an existing open order."
          icon={<Pencil className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.repriceOrder, prefill)}
        />
        <ActionCard
          title="Orderbook Void"
          description="Cancel an open order you created."
          icon={<Trash2 className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.voidOrder, prefill)}
        />
        <ActionCard
          title="Orderbook Lock"
          description="Lock an available order as buyer."
          icon={<Lock className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.lockOrder, prefill)}
        />
        <ActionCard
          title="Orderbook Close"
          description="Close a locked order to finalize the swap."
          icon={<CheckCircle2 className="w-4 h-4" />}
          onClick={() => runAction(ACTION_IDS.closeOrder, prefill)}
        />
      </div>

      <section
        id="how-it-works"
        className="overflow-hidden rounded-2xl border border-[#272729] bg-[#171717] shadow-sm"
      >
        <button
          type="button"
          className="flex h-auto w-full items-start justify-between gap-3 px-5 py-5 text-left transition-colors hover:bg-[#0f0f0f]"
          onClick={() => setIsHowItWorksOpen((open) => !open)}
          aria-expanded={isHowItWorksOpen}
          aria-controls="orders-how-it-works-content"
        >
          <div className="min-w-0 flex-1 space-y-1">
            <div className="flex items-center gap-2">
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#216cd0]/12 text-[#216cd0]">
                <Info className="h-4 w-4" />
              </span>
              <h2 className="text-xl font-semibold tracking-tight text-foreground">How it works</h2>
            </div>
            <p className="text-sm text-muted-foreground">
                    A quick guide to committee orderbook actions, AMM operations, and when to use each flow.
            </p>
          </div>
          <ChevronDown
            className={`mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform ${
              isHowItWorksOpen ? "rotate-0" : "-rotate-90"
            }`}
          />
        </button>
        {isHowItWorksOpen && (
          <div
            id="orders-how-it-works-content"
            className="border-t border-[#272729] px-5 pb-5 pt-4"
          >
            <div className="space-y-4">
              <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                  <ArrowLeftRight className="h-4 w-4 text-white/70" />
                  Orderbook vs AMM
                </div>
                <div className="space-y-3 text-sm leading-relaxed text-muted-foreground">
                  <p>
                    The committee orderbook is a staged peer-to-peer flow. One side creates an order, another side locks it, and the trade is finalized only when the close step completes.
                  </p>
                  <p>
                    AMM operations are direct pool interactions. They are better for immediate swaps or liquidity management and do not use the separate create, lock, and close lifecycle.
                  </p>
                </div>
              </div>
              <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                  <PlusCircle className="h-4 w-4 text-white/70" />
                  Committee Orderbook Actions
                </div>
                <div className="space-y-3 text-sm leading-relaxed text-muted-foreground">
                  <p>
                    <span className="font-medium text-foreground">Create</span> opens a new committee order. <span className="font-medium text-foreground">Reprice</span> updates an open order before it is taken, and <span className="font-medium text-foreground">Void</span> cancels it.
                  </p>
                  <p>
                    <span className="font-medium text-foreground">Lock</span> reserves an available order for the counterparty, and <span className="font-medium text-foreground">Close</span> completes the swap once both sides are ready.
                  </p>
                </div>
              </div>
              <div className="rounded-xl border border-[#272729] bg-[#171717] p-5">
                <div className="mb-3 flex items-center gap-2 text-sm font-semibold text-foreground">
                  <Droplets className="h-4 w-4 text-white/70" />
                  AMM Operations
                </div>
                <div className="space-y-3 text-sm leading-relaxed text-muted-foreground">
                  <p>
                    <span className="font-medium text-foreground">Limit Order</span> places a trade with a price constraint. Use it when execution price matters more than speed.
                  </p>
                  <p>
                    <span className="font-medium text-foreground">Deposit Liquidity</span> adds funds to the pool, while <span className="font-medium text-foreground">Withdraw Liquidity</span> removes your position back out of it.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
      </section>
    </motion.div>
  );
}
