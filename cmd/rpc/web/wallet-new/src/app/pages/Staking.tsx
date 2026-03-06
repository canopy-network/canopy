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

type ValidatorRow = {
  address: string;
  nickname?: string;
  stakedAmount: number;
  status: "Staked" | "Paused" | "Unstaking";
  rewards24h: number;
  chains?: string[];
  isSynced: boolean;
  // Additional validator information
  committees?: number[];
  compound?: boolean;
  delegate?: boolean;
  maxPausedHeight?: number;
  netAddress?: string;
  output?: string;
  publicKey?: string;
  unstakingHeight?: number;
};

const chainLabels = ["DEX", "CAN"] as const;

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
  const dsFetch = useDSFetcher();

  const csvRef = useRef<HTMLAnchorElement>(null);

  const [searchTerm, setSearchTerm] = useState("");
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
    return validators.map((v: any) => ({
      address: v.address,
      nickname: v.nickname,
      stakedAmount: v.stakedAmount || 0,
      status: v.unstaking ? "Unstaking" : v.paused ? "Paused" : "Staked",
      rewards24h: rewardsHistory[v.address]?.rewards24h || 0,
      chains:
        v.committees?.map(
          (id: number) => chainLabels[id % chainLabels.length],
        ) || [],
      isSynced: !v.paused,
      // Additional info
      committees: v.committees,
      compound: v.compound,
      delegate: v.delegate,
      maxPausedHeight: v.maxPausedHeight,
      netAddress: v.netAddress,
      output: v.output,
      publicKey: v.publicKey,
      unstakingHeight: v.unstakingHeight,
    }));
  }, [validators, rewardsHistory]);

  const filtered: ValidatorRow[] = useMemo(() => {
    const q = searchTerm.toLowerCase();
    if (!q) return rows;
    return rows.filter(
      (r) =>
        (r.nickname || "").toLowerCase().includes(q) ||
        r.address.toLowerCase().includes(q),
    );
  }, [rows, searchTerm]);

  const prepareCSVData = useCallback(() => {
    const header = [
      "address",
      "nickname",
      "stakedAmount",
      "rewards24h",
      "status",
    ];
    const lines = [header.join(",")].concat(
      filtered.map((r) =>
        [
          r.address,
          r.nickname || "",
          r.stakedAmount,
          r.rewards24h,
          r.status,
        ].join(","),
      ),
    );
    return lines.join("\n");
  }, [filtered]);

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
    openAction("stake");
  }, [openAction]);

  return (
    <motion.div
      className="min-h-screen bg-background"
      initial="hidden"
      animate="visible"
      variants={containerVariants}
    >
      {/* Hidden link for CSV export */}
      <a ref={csvRef} style={{ display: "none" }} />

      <div className="px-6 py-8">
        {/* Top stats */}
        <StatsCards
          totalStaked={totalStaked}
          totalRewards={staking.totalRewards24h || 0}
          validatorsCount={validators.length}
          chainCount={chainCount}
          activeValidatorsCount={activeValidatorsCount}
        />

        <div className="flex flex-col bg-card rounded-xl p-6 border border-border">
          {/* Toolbar */}
          <Toolbar
            searchTerm={searchTerm}
            onSearchChange={setSearchTerm}
            onAddStake={handleAddStake}
            onExportCSV={exportCSV}
            activeValidatorsCount={activeValidatorsCount}
          />

          {/* Validator List */}
          <ValidatorList validators={filtered} />
        </div>
      </div>
    </motion.div>
  );
}
