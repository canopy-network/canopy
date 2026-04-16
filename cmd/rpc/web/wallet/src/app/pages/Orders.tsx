import React from "react";
import { motion } from "framer-motion";
import {
  ArrowLeftRight,
  CheckCircle2,
  CircleDashed,
  Droplets,
  Lock,
  Pencil,
  PlusCircle,
  Trash2,
  Wallet,
} from "lucide-react";
import { Button } from "@/components/ui/Button";
import { StatusBadge } from "@/components/ui/StatusBadge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/Select";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useConfig } from "@/app/providers/ConfigProvider";
import { useAccountsList, useSelectedAccount } from "@/app/providers/AccountsProvider";
import { useDS } from "@/core/useDs";

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
  const { accounts } = useAccountsList();
  const { selectedAccount, switchAccount, selectedAddress } = useSelectedAccount();

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
      address: selectedAddress || "",
      committees: String(committeeId ?? ""),
    }),
    [selectedAddress, committeeId],
  );

  const runAction = React.useCallback(
    (actionId: string, prefilledData?: Record<string, unknown>) => {
      openAction(actionId, { prefilledData });
    },
    [openAction],
  );

  return (
    <motion.div
      className="min-h-screen bg-background"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="px-6 py-8 space-y-6">
        <section className="canopy-card p-5 md:p-6">
          <div className="flex items-center gap-2 mb-2">
            <h1 className="text-2xl font-semibold text-foreground">Orders</h1>
          </div>
          <p className="text-sm text-muted-foreground">
            Create, reprice, void, lock, and close orders or execute DEX operations.
            Fill in the order details manually in each form.
          </p>

          <div className="mt-4">
            <Select
              value={selectedAccount?.id ?? ""}
              onValueChange={switchAccount}
            >
              <SelectTrigger className="w-full gap-2 h-11 text-sm">
                <Wallet className="w-4 h-4 text-muted-foreground shrink-0" />
                <SelectValue placeholder="Select address" />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((account) => (
                  <SelectItem key={account.id} value={account.id}>
                    {account.nickname
                      ? `${account.nickname} (${shortHex(account.address)})`
                      : shortHex(account.address)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-wrap items-center gap-2 mt-3">
            <StatusBadge label={`Committee ${committeeId ?? "-"}`} status="info" size="sm" />
          </div>
        </section>

        <div className="grid grid-cols-1 xl:grid-cols-12 gap-6">
          <div className="xl:col-span-8 space-y-6">
            <section className="canopy-card p-5 md:p-6">
              <h2 className="text-lg font-semibold text-foreground mb-1">Committee Orderbook</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Seller and buyer lifecycle on committee orders: create, lock, reprice, void, and close.
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                <ActionCard
                  title="Create Order"
                  description="Create a new sell order on the committee."
                  icon={<PlusCircle className="w-4 h-4" />}
                  variant="default"
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.createOrder, {
                    ...prefill,
                    receiveAddress: selectedAddress || "",
                  })}
                />
                <ActionCard
                  title="Reprice Order"
                  description="Change the price of an existing open order."
                  icon={<Pencil className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.repriceOrder, prefill)}
                />
                <ActionCard
                  title="Void Order"
                  description="Cancel an open order you created."
                  icon={<Trash2 className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.voidOrder, prefill)}
                />
                <ActionCard
                  title="Lock Order"
                  description="Lock an available order as buyer."
                  icon={<Lock className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.lockOrder, {
                    ...prefill,
                    receiveAddress: selectedAddress || "",
                  })}
                />
                <ActionCard
                  title="Close Order"
                  description="Close a locked order to finalize the swap."
                  icon={<CheckCircle2 className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.closeOrder, prefill)}
                />
              </div>
            </section>
          </div>

          <aside className="xl:col-span-4 space-y-6 xl:sticky xl:top-[calc(var(--topbar-height,52px)+1rem)] self-start">
            <section className="canopy-card p-5 md:p-6">
              <h2 className="text-lg font-semibold text-foreground mb-1">DEX Operations</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Pool liquidity and limit-price swaps against DEX endpoints.
              </p>
              <div className="grid grid-cols-1 gap-3">
                <ActionCard
                  title="Limit Order"
                  description="Swap with a price constraint."
                  icon={<ArrowLeftRight className="w-4 h-4" />}
                  variant="default"
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.dexLimitOrder, { ...prefill, memo: "" })}
                />
                <ActionCard
                  title="Deposit Liquidity"
                  description="Add liquidity to the DEX pool."
                  icon={<Droplets className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.dexLiquidityDeposit, { ...prefill, memo: "" })}
                />
                <ActionCard
                  title="Withdraw Liquidity"
                  description="Remove liquidity from the DEX pool."
                  icon={<CircleDashed className="w-4 h-4" />}
                  disabled={!selectedAddress}
                  onClick={() => runAction(ACTION_IDS.dexLiquidityWithdraw, { ...prefill, memo: "" })}
                />
              </div>
            </section>
          </aside>
        </div>
      </div>
    </motion.div>
  );
}
