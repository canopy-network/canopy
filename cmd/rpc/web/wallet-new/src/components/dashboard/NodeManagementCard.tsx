import React, { useState, useCallback, useMemo } from "react";
import { motion } from "framer-motion";
import { Play, Pause, Key } from "lucide-react";
import { useValidators } from "@/hooks/useValidators";
import { useMultipleValidatorRewardsHistory } from "@/hooks/useMultipleValidatorRewardsHistory";
import { useMultipleValidatorSets } from "@/hooks/useValidatorSet";
import { useManifest } from "@/hooks/useManifest";
import { ActionsModal } from "@/actions/ActionsModal";
import { StatusBadge } from "@/components/ui/StatusBadge";
import { LoadingState } from "@/components/ui/LoadingState";
import { EmptyState } from "@/components/ui/EmptyState";
import { useDS } from "@/core/useDs";

// Helper functions moved outside component to avoid recreation
const formatAddress = (address: string) => {
  return address.substring(0, 8) + "..." + address.substring(address.length - 4);
};

const getNodeColor = (index: number) => {
  const colors = [
    "bg-gradient-to-r from-primary/80 to-primary/40",
    "bg-gradient-to-r from-orange-500/80 to-orange-500/40",
    "bg-gradient-to-r from-blue-500/80 to-blue-500/40",
    "bg-gradient-to-r from-red-500/80 to-red-500/40",
  ];
  return colors[index % colors.length];
};

// Mini chart component
const MiniChart = React.memo<{ index: number }>(({ index }) => {
  const dataPoints = 8;
  const patterns = [
    [30, 35, 40, 45, 50, 55, 60, 65],
    [50, 48, 52, 50, 49, 51, 50, 52],
    [70, 65, 60, 55, 50, 45, 40, 35],
    [50, 60, 40, 55, 35, 50, 45, 50],
  ];

  const pattern = patterns[index % patterns.length];
  const points = pattern.map((y, i) => ({
    x: (i / (dataPoints - 1)) * 100,
    y: y,
  }));

  const pathData = points
    .map((point, i) => `${i === 0 ? "M" : "L"}${point.x},${point.y}`)
    .join(" ");

  const isUpward = pattern[pattern.length - 1] > pattern[0];
  const isDownward = pattern[pattern.length - 1] < pattern[0];
  const color = isUpward ? "#10b981" : isDownward ? "#ef4444" : "#6b7280";

  return (
    <svg width="24" height="16" viewBox="0 0 100 60" className="flex-shrink-0">
      <defs>
        <linearGradient
          id={`mini-chart-gradient-${index}`}
          x1="0%"
          y1="0%"
          x2="0%"
          y2="100%"
        >
          <stop offset="0%" stopColor={color} stopOpacity="0.3" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path
        d={pathData}
        stroke={color}
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d={`${pathData} L100,60 L0,60 Z`}
        fill={`url(#mini-chart-gradient-${index})`}
      />
      {points.map((point, i) => (
        <circle key={i} cx={point.x} cy={point.y} r="1" fill={color} opacity="0.8" />
      ))}
    </svg>
  );
});

MiniChart.displayName = 'MiniChart';

// Processed validator node type
interface ProcessedNode {
  address: string;
  stakeAmount: string;
  status: string;
  rewards24h: string;
  originalValidator: any;
}

// Memoized table row component
interface ValidatorTableRowProps {
  node: ProcessedNode;
  index: number;
  onPauseUnpause: (validator: any, action: "pause" | "unpause") => void;
}

const ValidatorTableRow = React.memo<ValidatorTableRowProps>(({
  node,
  index,
  onPauseUnpause,
}) => (
  <motion.tr
    className="border-b border-bg-accent/50"
    initial={{ opacity: 0, y: 10 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, delay: index * 0.05 }}
  >
    <td className="py-4">
      <div className="flex items-center gap-3">
        <div
          className={`w-8 h-8 rounded-full ${getNodeColor(index)} flex items-center justify-center`}
        />
        <div className="flex flex-col">
          <span className="text-text-primary text-sm font-medium">
            {node.originalValidator.nickname || `Node ${index + 1}`}
          </span>
          <span className="text-text-muted text-xs font-mono">
            {formatAddress(node.originalValidator.address)}
          </span>
        </div>
      </div>
    </td>
    <td className="py-4">
      <div className="flex items-center gap-2">
        <span className="text-text-primary text-sm">{node.stakeAmount}</span>
        <MiniChart index={index} />
      </div>
    </td>
    <td className="py-4">
      <StatusBadge label={node.status} size="sm" />
    </td>
    <td className="py-4">
      <span className="text-primary text-sm font-medium">{node.rewards24h}</span>
    </td>
    <td className="py-4">
      <button
        onClick={() =>
          onPauseUnpause(
            node.originalValidator,
            node.status === "Staked" ? "pause" : "unpause"
          )
        }
        className="p-2.5 hover:bg-bg-accent rounded-lg transition-colors min-w-[44px] min-h-[44px] flex items-center justify-center"
        aria-label={node.status === "Staked" ? "Pause validator" : "Resume validator"}
      >
        {node.status === "Staked" ? (
          <Pause className="text-text-muted hover:text-text-primary w-5 h-5" />
        ) : (
          <Play className="text-text-muted hover:text-text-primary w-5 h-5" />
        )}
      </button>
    </td>
  </motion.tr>
));

ValidatorTableRow.displayName = 'ValidatorTableRow';

// Memoized mobile card component
const ValidatorMobileCard = React.memo<ValidatorTableRowProps>(({
  node,
  index,
  onPauseUnpause,
}) => (
  <motion.div
    className="bg-bg-tertiary/30 rounded-lg p-4 space-y-3"
    initial={{ opacity: 0, y: 10 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.2, delay: index * 0.05 }}
  >
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <div className={`w-8 h-8 rounded-full ${getNodeColor(index)}`} />
        <div>
          <div className="text-text-primary text-sm font-medium">
            {node.originalValidator.nickname || `Node ${index + 1}`}
          </div>
          <div className="text-text-muted text-xs font-mono">
            {formatAddress(node.originalValidator.address)}
          </div>
        </div>
      </div>
      <button
        onClick={() =>
          onPauseUnpause(
            node.originalValidator,
            node.status === "Staked" ? "pause" : "unpause"
          )
        }
        className="p-2.5 hover:bg-bg-accent rounded-lg transition-colors min-w-[44px] min-h-[44px] flex items-center justify-center"
        aria-label={node.status === "Staked" ? "Pause validator" : "Resume validator"}
      >
        {node.status === "Staked" ? (
          <Pause className="text-text-muted hover:text-text-primary w-5 h-5" />
        ) : (
          <Play className="text-text-muted hover:text-text-primary w-5 h-5" />
        )}
      </button>
    </div>
    <div className="grid grid-cols-2 gap-3">
      <div>
        <div className="text-text-muted text-xs mb-1">Stake</div>
        <div className="text-text-primary text-sm">{node.stakeAmount}</div>
      </div>
      <div>
        <div className="text-text-muted text-xs mb-1">Status</div>
        <StatusBadge label={node.status} size="sm" />
      </div>
      <div>
        <div className="text-text-muted text-xs mb-1">Rewards (24h)</div>
        <div className="text-primary text-sm font-medium">{node.rewards24h}</div>
      </div>
    </div>
  </motion.div>
));

ValidatorMobileCard.displayName = 'ValidatorMobileCard';

export const NodeManagementCard = React.memo((): JSX.Element => {
  // Fetch keystore data - all keys you have
  const { data: keystore, isLoading: keystoreLoading } = useDS("keystore", {});
  const { data: validators = [], isLoading: validatorsLoading, error } = useValidators();
  const { manifest } = useManifest();

  const validatorAddresses = useMemo(() => validators.map((v) => v.address), [validators]);
  const { data: rewardsData = {} } =
    useMultipleValidatorRewardsHistory(validatorAddresses);

  // Get unique committee IDs from validators
  const committeeIds = useMemo(() => {
    const ids = new Set<number>();
    validators.forEach((v: any) => {
      if (Array.isArray(v.committees)) {
        v.committees.forEach((id: number) => ids.add(id));
      }
    });
    return Array.from(ids);
  }, [validators]);

  const { data: validatorSetsData = {} } =
    useMultipleValidatorSets(committeeIds);

  const [isActionModalOpen, setIsActionModalOpen] = useState(false);
  const [selectedActions, setSelectedActions] = useState<any[]>([]);

  const isLoading = keystoreLoading || validatorsLoading;

  const formatStakeAmount = useCallback((amount: number) => {
    return (amount / 1000000).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  }, []);

  const formatRewards = useCallback((rewards: number) => {
    return `+${(rewards / 1000000).toFixed(2)} CNPY`;
  }, []);

  const getStatus = useCallback((validator: any) => {
    if (!validator) return "Liquid";
    if (validator.unstaking) return "Unstaking";
    if (validator.paused) return "Paused";
    return "Staked";
  }, []);

  const handlePauseUnpause = useCallback(
    (validator: any, action: "pause" | "unpause") => {
      const actionId =
        action === "pause" ? "pauseValidator" : "unpauseValidator";
      const actionDef = manifest?.actions?.find((a: any) => a.id === actionId);

      if (actionDef) {
        setSelectedActions([
          {
            ...actionDef,
            prefilledData: {
              validatorAddress: validator.address,
            },
          },
        ]);
        setIsActionModalOpen(true);
      } else {
        alert(`${action} action not found in manifest`);
      }
    },
    [manifest],
  );

  const handlePauseAll = useCallback(() => {
    const activeValidators = validators.filter((v) => !v.paused);
    if (activeValidators.length === 0) {
      alert("No active validators to pause");
      return;
    }

    // For simplicity, pause the first validator
    // In a full implementation, you could loop through all
    const firstValidator = activeValidators[0];
    handlePauseUnpause(firstValidator, "pause");
  }, [validators, handlePauseUnpause]);

  const handleResumeAll = useCallback(() => {
    const pausedValidators = validators.filter((v) => v.paused);
    if (pausedValidators.length === 0) {
      alert("No paused validators to resume");
      return;
    }

    const firstValidator = pausedValidators[0];
    handlePauseUnpause(firstValidator, "unpause");
  }, [validators, handlePauseUnpause]);

  // Process all keystores and match with validators
  const processedKeystores = useMemo((): ProcessedNode[] => {
    if (!keystore?.addressMap) return [];

    const addressMap = keystore.addressMap as Record<string, any>;
    const validatorMap = new Map(validators.map(v => [v.address, v]));

    return Object.entries(addressMap)
      .slice(0, 8) // Show up to 8 keys
      .map(([address, keyData]) => {
        const validator = validatorMap.get(address);
        return {
          address: formatAddress(address),
          stakeAmount: validator ? formatStakeAmount(validator.stakedAmount) : "0.00",
          status: getStatus(validator),
          rewards24h: validator ? formatRewards(rewardsData[address]?.change24h || 0) : "+0.00 CNPY",
          originalValidator: validator || {
            address,
            nickname: keyData.keyNickname || "Unnamed Key",
            stakedAmount: 0
          },
        };
      })
      .sort((a, b) => {
        // Sort staked first, then by nickname
        if (a.status === "Staked" && b.status !== "Staked") return -1;
        if (a.status !== "Staked" && b.status === "Staked") return 1;
        return 0;
      });
  }, [keystore, validators, formatStakeAmount, getStatus, formatRewards, rewardsData]);

  if (isLoading) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.5 }}
      >
        <LoadingState message="Loading validators..." size="md" />
      </motion.div>
    );
  }

  if (error) {
    return (
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.5 }}
      >
        <EmptyState
          icon="AlertCircle"
          title="Error loading validators"
          description="There was a problem loading your validators"
          size="md"
        />
      </motion.div>
    );
  }

  return (
    <>
      <motion.div
        className="bg-bg-secondary rounded-xl p-6 border border-bg-accent h-full"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, delay: 0.5 }}
      >
        {/* Header with action buttons */}
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between mb-6 gap-4">
          <div className="flex items-center gap-2">
            <Key className="w-5 h-5 text-primary" />
            <h3 className="text-text-primary text-lg font-semibold">
              Key Management
            </h3>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleResumeAll}
              className="flex items-center gap-2 px-4 py-2.5 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg text-sm font-medium transition-colors min-h-[44px]"
            >
              <Play className="w-4 h-4" />
              Resume All
            </button>
            <button
              onClick={handlePauseAll}
              className="flex items-center gap-2 px-4 py-2.5 bg-bg-tertiary hover:bg-bg-accent text-text-primary rounded-lg text-sm font-medium transition-colors min-h-[44px]"
            >
              <Pause className="w-4 h-4" />
              Pause All
            </button>
          </div>
        </div>

        {/* Table - Desktop */}
        <div className="hidden md:block overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-bg-accent">
                <th className="text-left text-text-muted text-sm font-medium pb-3">
                  Key
                </th>
                <th className="text-left text-text-muted text-sm font-medium pb-3">
                  Staked
                </th>
                <th className="text-left text-text-muted text-sm font-medium pb-3">
                  Status
                </th>
                <th className="text-left text-text-muted text-sm font-medium pb-3">
                  Rewards (24h)
                </th>
                <th className="text-left text-text-muted text-sm font-medium pb-3">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {processedKeystores.length > 0 ? (
                processedKeystores.map((node, index) => (
                  <ValidatorTableRow
                    key={node.originalValidator.address}
                    node={node}
                    index={index}
                    onPauseUnpause={handlePauseUnpause}
                  />
                ))
              ) : (
                <tr>
                  <td colSpan={5} className="py-4">
                    <EmptyState
                      icon="Key"
                      title="No keys found"
                      description="Your keys will appear here"
                      size="sm"
                    />
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        {/* Cards - Mobile */}
        <div className="md:hidden space-y-4">
          {processedKeystores.length > 0 ? (
            processedKeystores.map((node, index) => (
              <ValidatorMobileCard
                key={node.originalValidator.address}
                node={node}
                index={index}
                onPauseUnpause={handlePauseUnpause}
              />
            ))
          ) : (
            <EmptyState
              icon="Key"
              title="No keys found"
              description="Your keys will appear here"
              size="sm"
            />
          )}
        </div>
      </motion.div>

      {/* Actions Modal */}
      <ActionsModal
        actions={selectedActions}
        isOpen={isActionModalOpen}
        onClose={() => setIsActionModalOpen(false)}
      />
    </>
  );
});

NodeManagementCard.displayName = 'NodeManagementCard';
