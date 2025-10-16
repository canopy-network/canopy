import React from 'react';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';

interface ActivePollsProps {
  polls: Record<string, { proposalHash: string; proposalURL: string; accounts: any; validators: any }>;
}

const Badge = ({ children, tone = 'default' }: { children: React.ReactNode; tone?: 'default' | 'core' | 'treasury' | 'validator' | 'accounts' | 'validators' | 'terminated' }) => (
  <span
    className={`text-[10px] px-2 py-0.5 rounded-full capitalize border ${tone === 'core'
      ? 'bg-primary/20 text-primary border-primary/20'
      : tone === 'treasury'
        ? 'bg-orange-500/20 text-orange-400 border-orange-500/20'
        : tone === 'validator'
          ? 'bg-purple-500/20 text-purple-400 border-purple-500/20'
          : tone === 'accounts'
            ? 'bg-green-500/20 text-green-400 border-green-500/20'
            : tone === 'validators'
              ? 'bg-blue-500/20 text-blue-400 border-blue-500/20'
              : tone === 'terminated'
                ? 'bg-red-500/20 text-red-400 border-red-500/20'
                : 'bg-gray-500/20 text-gray-400 border-gray-500/20'
      }`}
  >
    {children}
  </span>
);

export default function ActivePolls({ polls }: ActivePollsProps): JSX.Element {
  const { votePoll, currentHeight } = useGovernanceActions('50003', '50002');
  const { getText } = useManifest();
  // Calculate blocks remaining for a poll (assuming polls have endBlock)
  const calculateBlocksRemaining = (endBlock: number) => {
    const height = currentHeight.data || 0;
    if (height >= endBlock) {
      return 'Terminated';
    } else {
      const blocksRemaining = endBlock - height;
      return `${blocksRemaining} blocks`;
    }
  };

  // Get poll status for sorting and display
  const getPollStatus = (endBlock: number) => {
    const height = currentHeight.data || 0;
    if (height >= endBlock) return 'terminated';
    return 'active';
  };

  // Sort polls by status: active first, then terminated
  const entries = Object.entries(polls || {}).sort(([, dataA], [, dataB]) => {
    const statusA = 'active'; // getPollStatus(dataA.endBlock || 0);
    const statusB = 'active'; // getPollStatus(dataB.endBlock || 0);
    
    const statusOrder = { active: 0, terminated: 1 };
    return statusOrder[statusA as keyof typeof statusOrder] - statusOrder[statusB as keyof typeof statusOrder];
  });

  return (
    <div className="max-h-[40rem] overflow-y-auto">
      <div className="space-y-4">
        {entries.length === 0 && (
          <div className="text-text-muted text-sm">
            {getText('ui.governance.empty.noActivePolls', 'No active polls')}
          </div>
        )}
        {entries.map(([hash, data], index) => {
          const pollId = `POLL-${String(index + 1).padStart(3, '0')}`;

          // Use ONLY real data from API according to README
          const rawFor = data.accounts?.approvedPercent || 0;
          const rawAgainst = data.accounts?.rejectPercent || 0;

          // Normalizar para que For + Against = 100 % (evita fondo gris)
          const totalPercent = rawFor + rawAgainst;
          const forPercent = totalPercent > 0 ? (rawFor / totalPercent) * 100 : 0;
          const againstPercent = totalPercent > 0 ? (rawAgainst / totalPercent) * 100 : 0;
          const totalVotes = data.accounts?.totalVotedTokens || 0;
          const cnpyVotes = data.accounts?.totalVotedTokens || 0;

          // Calculate blocks remaining (assuming polls have endBlock - for now using a placeholder)
          const endBlock = 2500; // This should come from the poll data structure
          const blocksRemaining = calculateBlocksRemaining(endBlock);
          const pollStatus = getPollStatus(endBlock);

          // Use real data from API
          const pollTitle = data.proposalURL || 'Poll Title';
          const pollDescription = 'No description available';

          return (
            <div key={hash} className="bg-bg-secondary flex flex-col rounded-lg border border-bg-accent p-4 min-h-[14.4rem] justify-between">
              {/* Header */}
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Badge tone="core">{pollId}</Badge>
                  <Badge tone="accounts">Accounts</Badge>
                  <Badge tone="validators">Validators</Badge>
                  {pollStatus === 'terminated' && (
                    <Badge tone="terminated">Terminated</Badge>
                  )}
                </div>
                <div className="text-right">
                  <div className="text-white text-sm">{blocksRemaining}</div>
                  {pollStatus !== 'terminated' && (
                    <div className="text-[11px] text-text-muted">remaining</div>
                  )}
                </div>
              </div>

              {/* Body */}
              <div className="mb-3">
                <div className="text-white font-semibold mb-1">{pollTitle}</div>
                <div className="text-text-muted text-xs mb-3">{pollDescription}</div>

                <div className="mb-1 flex justify-between items-center">
                  <span className="text-xs text-primary">For: {forPercent}%</span>
                  <span className="text-xs text-red-400">Against: {againstPercent}%</span>
                </div>

                <div className="h-2 w-full bg-bg-primary rounded-full overflow-hidden mb-2 flex">
                  {/* Green segment */}
                  <div
                    className="h-full bg-primary"
                    style={{ width: `${forPercent}%` }}
                  ></div>
                  {/* Red segment */}
                  <div
                    className="h-full bg-red-500"
                    style={{ width: `${againstPercent}%` }}
                  ></div>
                </div>

                <div className="text-text-muted text-xs mb-3">{totalVotes} votes • {cnpyVotes} CNPY</div>
              </div>

              {/* Footer */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 text-xs text-text-muted">
                  <button className="hover:text-white text-primary flex items-center gap-1">
                    <i className="fa-solid fa-up-right-and-down-left-from-center"></i>
                    {getText('ui.governance.labels.details', 'Details')}
                  </button>
                  <button className="hover:text-white flex items-center gap-1">
                    <i className="fa-brands fa-discord"></i>
                    {getText('ui.governance.labels.discuss', 'Discuss')}
                  </button>
                </div>

                        <div className="flex items-center gap-2">
                          <button 
                            onClick={() => {
                              votePoll.mutate(
                                { 
                                  pollJSON: JSON.stringify({ proposalHash: hash, proposalURL: data.proposalURL }), 
                                  pollApprove: true, 
                                  password: 'test' 
                                },
                                {
                                  onSuccess: () => {
                                    toast.success('✅ "Approve" vote registered successfully!', {
                                      duration: 3000
                                    });
                                  },
                                  onError: (error: any) => {
                                    toast.error(`❌ Error voting in poll:\n${error.message}`, {
                                      duration: 4000,
                                      style: { whiteSpace: 'pre-line' }
                                    });
                                  }
                                }
                              );
                            }}
                            className="bg-primary text-muted px-3 py-1 rounded-md text-sm hover:bg-primary/80"
                          >
                            Approve
                          </button>
                          <button 
                            onClick={() => {
                              votePoll.mutate(
                                { 
                                  pollJSON: JSON.stringify({ proposalHash: hash, proposalURL: data.proposalURL }), 
                                  pollApprove: false, 
                                  password: 'test' 
                                },
                                {
                                  onSuccess: () => {
                                    toast.success('✅ "Reject" vote registered successfully!', {
                                      duration: 3000
                                    });
                                  },
                                  onError: (error: any) => {
                                    toast.error(`❌ Error voting in poll:\n${error.message}`, {
                                      duration: 4000,
                                      style: { whiteSpace: 'pre-line' }
                                    });
                                  }
                                }
                              );
                            }}
                            className="bg-red-500 text-white px-3 py-1 rounded-md text-sm hover:bg-red-500/80"
                          >
                            Reject
                          </button>
                        </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}