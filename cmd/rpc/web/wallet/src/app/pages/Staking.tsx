import React, {
  useEffect,
  useRef,
  useMemo,
  useState,
  useCallback,
} from "react";
import { motion } from "framer-motion";
import { useStakingData } from "@/hooks/useStakingData";
import { useValidators } from "@/hooks/useValidators";
import { useAccountData } from "@/hooks/useAccountData";
import { useMultipleValidatorRewardsHistory } from "@/hooks/useMultipleValidatorRewardsHistory";
import { useManifest } from "@/hooks/useManifest";
import { useDSFetcher } from "@/core/dsFetch";
import { StatsCards } from "@/components/staking/StatsCards";
import { Toolbar } from "@/components/staking/Toolbar";
import { ValidatorList } from "@/components/staking/ValidatorList";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { useSelectedAccount } from "@/app/providers/AccountsProvider";
import { PageHeader } from "@/components/layouts/PageHeader";

type ValidatorRow = {
  address: string;
  nickname?: string;
  stakedAmount: number;
  status: "Staked" | "Paused" | "Unstaking" | "Delegate";
  rewards24h: number;
  isSynced: boolean;
  committees?: number[];
  compound?: boolean;
  delegate?: boolean;
  maxPausedHeight?: number;
  netAddress?: string;
  output?: string;
  publicKey?: string;
  unstakingHeight?: number;
};

const containerVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.6, staggerChildren: 0.1 } },
};

export default function Staking(): JSX.Element {
  const {
    data: staking = { totalStaked: 0, totalRewards: 0, chartData: [] } as any,
  } = useStakingData();
  const { totalStaked } = useAccountData();
  const { data: validators = [] } = useValidators();
  const { openAction } = useActionModal();
  const { selectedAccount } = useSelectedAccount();
  const dsFetch = useDSFetcher();

  const csvRef = useRef<HTMLAnchorElement>(null);

  const [chainCount, setChainCount] = useState<number>(0);

  const validatorAddresses = useMemo(
    () => validators.map((v: any) => v.address),
    [validators],
  );

  const { data: rewardsHistory = {} } =
    useMultipleValidatorRewardsHistory(validatorAddresses);

  useEffect(() => {
    let isCancelled = false;

    const run = async () => {
      try {
        const all = await dsFetch("validators");
        const ourAddresses = new Set(validators.map((v: any) => v.address));
        const committees = new Set<number>();
        (all || []).forEach((v: any) => {
          if (ourAddresses.has(v.address) && Array.isArray(v.committees)) {
            v.committees.forEach((c: number) => committees.add(c));
          }
        });
        if (!isCancelled) {
          setChainCount((prev) =>
            prev !== committees.size ? committees.size : prev,
          );
        }
      } catch {
        if (!isCancelled) setChainCount(0);
      }
    };

    if (validators.length > 0) run();
    return () => {
      isCancelled = true;
    };
  }, [validators]);

  // 🧮 Construir filas memoizadas
  const rows: ValidatorRow[] = useMemo(() => {
    return validators.map((v) => {
      const extra = v as unknown as Record<string, unknown>;
      return {
        address: v.address,
        nickname: v.nickname,
        stakedAmount: v.stakedAmount || 0,
        status: (v.unstaking ? "Unstaking" : v.paused ? "Paused" : v.delegate ? "Delegate" : "Staked") as ValidatorRow["status"],
        rewards24h: rewardsHistory[v.address]?.rewards24h || 0,
        isSynced: !v.paused,
        committees: extra.committees as number[] | undefined,
        compound: extra.compound as boolean | undefined,
        delegate: v.delegate,
        maxPausedHeight: extra.maxPausedHeight as number | undefined,
        netAddress: extra.netAddress as string | undefined,
        output: extra.output as string | undefined,
        publicKey: v.publicKey,
        unstakingHeight: v.unstakingHeight,
      };
    });
  }, [validators, rewardsHistory]);

  const prepareCSVData = useCallback(() => {
    const header = [
      "address",
      "nickname",
      "publicKey",
      "stakedAmount",
      "rewards24h",
      "status",
      "netAddress",
      "output",
      "compound",
      "committees",
      "unstakingHeight",
    ];
    const lines = [header.join(",")].concat(
      rows.map((r) =>
        [
          r.address,
          r.nickname || "",
          r.publicKey || "",
          r.stakedAmount,
          r.rewards24h,
          r.status,
          r.netAddress || "",
          r.output || "",
          String(r.compound ?? ""),
          r.committees?.join(";") ?? "",
          r.unstakingHeight ?? "",
        ].join(","),
      ),
    );
    return lines.join("\n");
  }, [rows]);

  const exportCSV = useCallback(() => {
    const csvContent = prepareCSVData();
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);

    if (csvRef.current) {
      csvRef.current.href = url;
      csvRef.current.download = "validators.csv";
      csvRef.current.click();
    }

    setTimeout(() => URL.revokeObjectURL(url), 100);
  }, [prepareCSVData]);

  const activeValidatorsCount = useMemo(
    () => validators.filter((v: any) => !v.paused).length,
    [validators],
  );

  // Handler to add stake - opens the "stake" action from manifest
  const handleAddStake = useCallback(() => {
    if (!selectedAccount?.address) {
      openAction("stake");
      return;
    }
    openAction("stake", {
      prefilledData: {
        operator: selectedAccount.address,
      },
    });
  }, [openAction, selectedAccount?.address]);

  return (
    <motion.div
      className="space-y-6"
      initial="hidden"
      animate="visible"
      variants={containerVariants}
    >
      {/* Hidden link for CSV export */}
      <a ref={csvRef} hidden aria-hidden="true" />

      <PageHeader
        title="Staking"
        subtitle="Track staked positions, rewards, and validator activity."
      />

      {/* Top stats */}
      <StatsCards
        totalStaked={totalStaked}
        totalRewards={staking.totalRewards24h || 0}
        validatorsCount={validators.length}
        chainCount={chainCount}
        activeValidatorsCount={activeValidatorsCount}
      />

      <div className="flex flex-col bg-card rounded-xl border border-border p-5">
        {/* Toolbar */}
        <Toolbar
          onAddStake={handleAddStake}
          onExportCSV={exportCSV}
          activeValidatorsCount={activeValidatorsCount}
        />

        {/* Validator List */}
        <ValidatorList validators={rows} />
      </div>
    </motion.div>
  );
}
