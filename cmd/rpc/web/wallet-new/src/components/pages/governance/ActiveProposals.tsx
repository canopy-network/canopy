import React from 'react';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';

interface ActiveProposalsProps {
  proposals: Record<string, { proposal: any; approve: boolean }>;
}

const Badge = ({ children, tone = 'default' }: { children: React.ReactNode; tone?: 'default' | 'core' | 'treasury' | 'blue' | 'terminated' | 'fee' | 'val' | 'cons' | 'gov' | 'paramKey' }) => (
  <span
    className={`text-[10px] px-2 py-0.5 rounded-full capitalize border ${tone === 'core'
      ? 'bg-primary/20 text-primary border-primary/20'
      : tone === 'treasury'
        ? 'bg-orange-500/20 text-orange-400 border-orange-500/20'
        : tone === 'blue'
          ? 'bg-blue-500/20 text-blue-400 border-blue-500/20'
          : tone === 'terminated'
            ? 'bg-red-500/20 text-red-400 border-red-500/20'
            : tone === 'fee'
              ? 'bg-blue-500/20 text-blue-400 border-blue-500/20'
              : tone === 'val'
                ? 'bg-purple-500/20 text-purple-400 border-purple-500/20'
                : tone === 'cons'
                  ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/20'
                  : tone === 'gov'
                    ? 'bg-pink-500/20 text-pink-400 border-pink-500/20'
                    : tone === 'paramKey'
                      ? 'bg-cyan-500/20 text-cyan-400 border-cyan-500/20'
                      : 'bg-gray-500/20 text-gray-400 border-gray-500/20'
      }`}
  >
    {children}
  </span>
);

export default function ActiveProposals({ proposals }: ActiveProposalsProps): JSX.Element {
  const { addVote, deleteVote, currentHeight } = useGovernanceActions('50003', '50002');

  // Get badge color based on parameterSpace
  const getParameterSpaceColor = (parameterSpace: string) => {
    switch (parameterSpace) {
      case 'fee': return 'fee';
      case 'val': return 'val';
      case 'cons': return 'cons';
      case 'gov': return 'gov';
      default: return 'blue';
    }
  };

  // Calculate blocks remaining for a proposal
  const calculateBlocksRemaining = (startHeight: number, endHeight: number) => {
    const height = currentHeight.data || 0;
    if (height < startHeight) {
      const blocksUntilStart = startHeight - height;
      return `${blocksUntilStart} blocks until start`;
    } else if (height >= endHeight) {
      return 'Terminated';
    } else {
      const blocksRemaining = endHeight - height;
      return `${blocksRemaining} blocks`;
    }
  };

  // Get proposal status for sorting and display
  const getProposalStatus = (startHeight: number, endHeight: number) => {
    const height = currentHeight.data || 0;
    if (height < startHeight) return 'pending';
    if (height >= endHeight) return 'terminated';
    return 'active';
  };

  // Sort proposals by status: active first, then pending, then terminated
  const entries = Object.entries(proposals || {}).sort(([, dataA], [, dataB]) => {
    const statusA = getProposalStatus(dataA.proposal?.msg?.startHeight || 0, dataA.proposal?.msg?.endHeight || 0);
    const statusB = getProposalStatus(dataB.proposal?.msg?.startHeight || 0, dataB.proposal?.msg?.endHeight || 0);

    const statusOrder = { active: 0, pending: 1, terminated: 2 };
    return statusOrder[statusA as keyof typeof statusOrder] - statusOrder[statusB as keyof typeof statusOrder];
  });

  return (
    <div className="max-h-[40rem] overflow-y-auto">
      <div className="space-y-4">
        {entries.length === 0 && (
          <div className="text-text-muted text-sm">
            No active proposals
          </div>
        )}
        {entries.map(([hash, data], index) => {
          const key = data.proposal?.msg?.parameterKey || data.proposal?.type || 'protocolVersion';
          const proposalId = `PROP-${String(index + 1).padStart(3, '0')}`;

          // Calculate blocks remaining based on block heights
          const startHeight = data.proposal?.msg?.startHeight || 0;
          const endHeight = data.proposal?.msg?.endHeight || 0;
          const blocksRemaining = calculateBlocksRemaining(startHeight, endHeight);
          const proposalStatus = getProposalStatus(startHeight, endHeight);

          // Check if proposal is active (current height is between start and end)
          const height = currentHeight.data || 0;
          const isActive = height >= startHeight && height < endHeight;

          const hasVoted = data.approve !== undefined;
          const voteType = hasVoted ? (data.approve ? 'For' : 'Against') : null;

          return (
            <div key={hash} className="bg-bg-secondary flex flex-col rounded-lg border border-bg-accent p-5 min-h-[10rem] justify-between">
              {/* Header */}
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Badge tone="core">{proposalId}</Badge>
                  <Badge tone={getParameterSpaceColor(data.proposal?.msg?.parameterSpace || '')}>
                    {data.proposal?.msg?.parameterSpace || 'Core'}
                  </Badge>
                  <Badge tone="paramKey">
                    {data.proposal?.msg?.parameterKey || 'Unknown'}
                  </Badge>
                  {proposalStatus === 'terminated' && (
                    <Badge tone="terminated">Terminated</Badge>
                  )}
                </div>
                <div className="text-right">
                  <div className="text-white text-sm">{blocksRemaining}</div>
                  <div className="text-[11px] text-text-muted">remaining</div>
                </div>
              </div>

              {/* Body */}
              <div className="mb-3">
                <div className="text-white font-semibold mb-1">{key}</div>
                <div className="text-text-muted text-xs max-w-[60ch] line-clamp-3">{data.proposal?.memo || 'No description available'}</div>
              </div>

              {/* Footer */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 text-xs text-text-muted">
                  <button className="hover:text-white text-primary flex items-center gap-1">
                    <i className="fa-solid fa-up-right-and-down-left-from-center"></i>
                    Details
                  </button>
                  <button className="hover:text-white flex items-center gap-1">
                    <i className="fa-brands fa-discord"></i>
                    Discuss
                  </button>
                </div>

                <div className="flex items-center gap-2">
                  {hasVoted ? (
                    <>
                      <div className="flex items-center gap-2">
                        <p className="text-sm  text-text-muted">Voted:</p>
                        <div className={`${voteType === 'For' ? 'bg-primary text-muted' : 'bg-status-error text-muted'} px-3 py-1 rounded-md text-sm`}>
                          {voteType}
                        </div>
                      </div>
                      <button
                        onClick={() => {
                          addVote.mutate(
                            { proposal: data.proposal, approve: !data.approve },
                            {
                              onSuccess: () => {
                                toast.success(`✅ Vote changed to "${!data.approve ? 'For' : 'Against'}" successfully!`, {
                                  duration: 3000
                                });
                              },
                              onError: (error: any) => {
                                toast.error(`❌ Error changing vote:\n${error.message}`, {
                                  duration: 4000,
                                  style: { whiteSpace: 'pre-line' }
                                });
                              }
                            }
                          );
                        }}
                        disabled={!isActive}
                        className={`px-3 py-1.5 rounded-md text-sm font-medium ${isActive
                          ? 'bg-gray-500/20 text-gray-400 hover:bg-gray-500/30'
                          : 'bg-gray-500/20 text-gray-400 cursor-not-allowed'
                          }`}
                      >
                        Change to {data.approve ? 'Against' : 'For'}
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        onClick={() => {
                          addVote.mutate(
                            { proposal: data.proposal, approve: true },
                            {
                              onSuccess: () => {
                                toast.success('✅ "For" vote registered successfully!', {
                                  duration: 3000
                                });
                              },
                              onError: (error: any) => {
                                toast.error(`❌ Error voting:\n${error.message}`, {
                                  duration: 4000,
                                  style: { whiteSpace: 'pre-line' }
                                });
                              }
                            }
                          );
                        }}
                        disabled={!isActive}
                        className={`px-3 py-1 rounded-md text-sm ${isActive
                          ? 'bg-primary text-muted hover:bg-primary/80'
                          : 'bg-gray-500/20 text-gray-400 cursor-not-allowed'
                          }`}
                      >
                        For
                      </button>
                      <button
                        onClick={() => {
                          addVote.mutate(
                            { proposal: data.proposal, approve: false },
                            {
                              onSuccess: () => {
                                toast.success('✅ "Against" vote registered successfully!', {
                                  duration: 3000
                                });
                              },
                              onError: (error: any) => {
                                toast.error(`❌ Error voting:\n${error.message}`, {
                                  duration: 4000,
                                  style: { whiteSpace: 'pre-line' }
                                });
                              }
                            }
                          );
                        }}
                        disabled={!isActive}
                        className={`px-3 py-1 rounded-md text-sm font-medium ${isActive
                          ? 'bg-red-500 text-muted hover:bg-status-error/80'
                          : 'bg-gray-500/20 text-gray-400 cursor-not-allowed'
                          }`}
                      >
                        Against
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}