import React from "react";
import { motion } from "framer-motion";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { useValidatorRewardsHistory } from "@/hooks/useValidatorRewardsHistory";
import { useActionModal } from "@/app/providers/ActionModalProvider";
import { LockOpen, Pause, Pen, Play, Globe, Key, Copy } from "lucide-react";

interface ValidatorCardProps {
  validator: {
    address: string;
    nickname?: string;
    stakedAmount: number;
    status: "Staked" | "Paused" | "Unstaking" | "Delegate";
    rewards24h: number;
    committees?: number[];
    isSynced: boolean;
    delegate?: boolean;
    netAddress?: string;
    publicKey?: string;
    output?: string;
  };
  index: number;
}

const formatStakedAmount = (amount: number) => {
  if (!amount && amount !== 0) return "0.00";
  return (amount / 1000000).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
};

const formatRewards = (amount: number) => {
  if (!amount && amount !== 0) return "+0.00";
  return `+${(amount / 1000000).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
};

const truncateAddress = (address: string) =>
  `${address.substring(0, 4)}…${address.substring(address.length - 4)}`;

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.4 } },
};

export const ValidatorCard: React.FC<ValidatorCardProps> = ({
  validator,
  index,
}) => {
  const { copyToClipboard } = useCopyToClipboard();
  const { openAction } = useActionModal();

  const { data: rewardsHistory, isLoading: rewardsLoading } =
    useValidatorRewardsHistory(validator.address);

  const handlePauseUnpause = () => {
    const actionId =
      validator.status === "Staked" ? "pauseValidator" : "unpauseValidator";
    openAction(actionId, {
      prefilledData: {
        validatorAddress: validator.address,
        signerAddress: validator.address,
      },
    });
  };

  const handleEditStake = () => {
    openAction("stake", {
      prefilledData: {
        operator: validator.address,
        selectCommittees: validator.committees || [],
      },
    });
  };

  const handleUnstake = () => {
    openAction("unstake", {
      prefilledData: {
        validatorAddress: validator.address,
      },
    });
  };

  return (
    <motion.div
      variants={itemVariants}
      className="bg-card rounded-xl border border-border/60 relative overflow-hidden"
    >
      <div className="p-4">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 lg:gap-6 items-center">
          {/* Validator identity */}
          <div className="lg:col-span-3">
            <div className="flex flex-col">
              <div className="text-primary capitalize font-medium mb-1 flex items-center">
                <span className="mr-2">
                  {validator.nickname || `Node ${index + 1}`}
                </span>
                <button className="text-bg-accent">
                  <i className="fa-solid fa-server text-muted-foreground text-xs"></i>
                </button>
              </div>
              <div className="text-muted-foreground text-sm font-mono">
                {truncateAddress(validator.address)}
              </div>
              <button
                className="text-primary text-xs mt-1 text-left w-fit"
                onClick={() =>
                  copyToClipboard(
                    validator.address,
                    `Validator ${validator.nickname || "address"}`,
                  )
                }
              >
                <i className="fa-solid fa-copy"></i> Copy
              </button>

              {/* Public Key */}
              {validator.publicKey && (
                <div className="flex items-center gap-1 mt-1.5" title="Public Key">
                  <Key className="w-3 h-3 text-muted-foreground/60 flex-shrink-0" />
                  <span className="text-muted-foreground text-xs font-mono">
                    {truncateAddress(validator.publicKey)}
                  </span>
                  <button
                    className="p-0.5 rounded hover:bg-accent/60 text-muted-foreground/40 hover:text-primary transition-colors flex-shrink-0"
                    onClick={() => copyToClipboard(validator.publicKey!, "Public Key")}
                    title="Copy Public Key"
                  >
                    <Copy className="w-2.5 h-2.5" />
                  </button>
                </div>
              )}

              {/* Net Address */}
              {validator.netAddress && (
                <div className="flex items-center gap-1 mt-1" title="Network Address">
                  <Globe className="w-3 h-3 text-muted-foreground/60 flex-shrink-0" />
                  <span className="text-muted-foreground text-xs font-mono">
                    {validator.netAddress}
                  </span>
                  <button
                    className="p-0.5 rounded hover:bg-accent/60 text-muted-foreground/40 hover:text-primary transition-colors flex-shrink-0"
                    onClick={() => copyToClipboard(validator.netAddress!, "Network Address")}
                    title="Copy Network Address"
                  >
                    <Copy className="w-2.5 h-2.5" />
                  </button>
                </div>
              )}

              {/* Committees */}
              {(validator.committees?.length ?? 0) > 0 && (
                <div className="flex items-center mt-2 gap-1.5 flex-wrap">
                  <span className="text-muted-foreground text-xs">Committees:</span>
                  {validator.committees!.map((id) => (
                    <span
                      key={id}
                      className="px-2 py-0.5 text-xs bg-primary/15 text-primary border border-primary/20 rounded font-mono font-medium"
                    >
                      {id}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Stats section */}
          <div className="lg:col-span-6 grid grid-cols-2 sm:grid-cols-2 gap-4">
            <div className="flex flex-col">
              <div className="text-foreground font-medium">
                {formatStakedAmount(validator.stakedAmount)} CNPY
              </div>
              <div className="text-muted-foreground text-xs">Total Staked</div>
            </div>

            <div className="flex flex-col">
              <div className="text-primary font-medium">
                {rewardsLoading
                  ? "..."
                  : formatRewards(rewardsHistory?.change24h || 0)}
              </div>
              <div className="text-muted-foreground text-xs">24h Rewards</div>
            </div>
          </div>

          {/* Status and Actions */}
          <div className="lg:col-span-3 flex flex-col sm:flex-row lg:flex-col xl:flex-row items-start sm:items-center lg:items-end xl:items-center justify-between lg:justify-end gap-3">
            <div className="flex items-center gap-2">
              <span
                className={`${
                  validator.status === "Staked" || validator.status === "Delegate"
                    ? "bg-primary/20 text-primary"
                    : validator.status === "Paused"
                      ? "bg-yellow-500/20 text-yellow-400"
                      : "bg-red-500/20 text-red-400"
                } text-xs px-3 py-1 rounded-full whitespace-nowrap`}
              >
                {validator.status}
              </span>
              <span
                className={`w-2 h-2 ${validator.isSynced ? "bg-primary" : "bg-red-500"} rounded-full flex-shrink-0`}
              ></span>
            </div>

            {validator.status !== "Unstaking" && (
              <div className="flex items-center gap-2">
                {!validator.delegate && (
                  <button
                    className="p-2 border border-border/60 rounded-lg transition-colors flex items-center gap-1.5 hover:bg-accent group hover:border-primary/40"
                    onClick={handlePauseUnpause}
                    title={validator.status === "Staked" ? "Pause Validator" : "Unpause Validator"}
                  >
                    {validator.status === 'Paused' ?
                      (<Play className={"w-4 h-4 text-foreground text-sm group-hover:text-primary"}/>) :
                      (<Pause className={"w-4 h-4 text-foreground text-sm group-hover:text-primary"}/>)
                    }
                    <span className="text-xs text-muted-foreground group-hover:text-primary hidden sm:inline">
                      {validator.status === 'Paused' ? 'Resume' : 'Pause'}
                    </span>
                  </button>
                )}
                <button
                  className="p-2 hover:bg-accent group hover:border-primary/40 border border-border/60 rounded-lg transition-colors flex items-center gap-1.5"
                  onClick={handleEditStake}
                  title="Edit Stake"
                >
                  <Pen className="w-4 h-4 text-foreground text-sm group-hover:text-primary" />
                  <span className="text-xs text-muted-foreground group-hover:text-primary hidden sm:inline">
                    Edit
                  </span>
                </button>

                <button
                  className="p-2 hover:bg-accent group hover:border-primary/40 border border-border/60 rounded-lg transition-colors flex items-center gap-1.5"
                  onClick={handleUnstake}
                  title="Unstake Validator"
                >
                  <LockOpen className="w-4 h-4 text-foreground text-sm group-hover:text-primary" />
                  <span className="text-xs text-muted-foreground group-hover:text-primary hidden sm:inline">
                    Unstake
                  </span>
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </motion.div>
  );
};
