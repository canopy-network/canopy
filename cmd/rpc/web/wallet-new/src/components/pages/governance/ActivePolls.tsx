import React from 'react';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';

interface ActivePollsProps {
  polls: Record<string, { proposalHash: string; proposalURL: string; accounts: any; validators: any }>;
}

const Badge = ({ children, tone = 'default' }: { children: React.ReactNode; tone?: 'default' | 'core' | 'treasury' | 'validator' }) => (
  <span
    className={`text-[10px] px-2 py-0.5 rounded-full border ${tone === 'core'
      ? 'bg-primary/20 text-primary border-primary/20'
      : tone === 'treasury'
        ? 'bg-orange-500/20 text-orange-400 border-orange-500/20'
        : tone === 'validator'
          ? 'bg-purple-500/20 text-purple-400 border-purple-500/20'
          : 'bg-gray-500/20 text-gray-400 border-gray-500/20'
      }`}
  >
    {children}
  </span>
);

export default function ActivePolls({ polls }: ActivePollsProps): JSX.Element {
  const { getText } = useManifest();
  const { votePoll } = useGovernanceActions('50003', '50002');
  const title = getText('ui.governance.sections.activePolls', 'Active Polls');

  const entries = Object.entries(polls || {});

  return (
    <div className="max-h-[40rem] overflow-y-auto">
      <div className="space-y-4">
        {entries.length === 0 && (
          <div className="text-text-muted text-sm">
            {getText('ui.governance.empty.noActivePolls', 'No active polls')}
          </div>
        )}
        {entries.map(([hash, data], index) => {
          const pollId = `POLL-${String(index + 2).padStart(3, '0')}`;

          // Use ONLY real data from API according to README
          const forPercent = data.accounts?.approvedPercent || 0;
          const againstPercent = data.accounts?.rejectPercent || 0;
          const totalVotes = data.accounts?.totalVotedTokens || 0;
          const cnpyVotes = data.accounts?.totalVotedTokens || 0;

          // No poll type determination - use default
          const pollType = 'validator';

          // No time data in API
          const timeRemaining = '0d 0h 0m';

          // Use real data from API
          const pollTitle = data.proposalURL || 'Poll Title';
          const pollDescription = 'No description available';

          return (
            <div key={hash} className="bg-bg-secondary flex flex-col rounded-lg border border-bg-accent p-4 min-h-[14.4rem] justify-between">
              {/* Header */}
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Badge tone="core">{pollId}</Badge>
                  <Badge tone="validator">
                    Validator
                  </Badge>
                </div>
                <div className="text-right">
                  <div className="text-white text-sm">{timeRemaining}</div>
                  <div className="text-[11px] text-text-muted">remaining</div>
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

                <div className="h-2 w-full bg-bg-primary rounded-full overflow-hidden mb-2 relative">
                  <div
                    className="absolute top-0 left-0 h-full bg-primary"
                    style={{ width: `${forPercent}%` }}
                  ></div>
                  <div
                    className="absolute top-0 left-0 h-full bg-red-500"
                    style={{ width: `${forPercent}%`, marginLeft: `${forPercent}%` }}
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
                                    toast.success('✅ Voto "Aprobar" registrado exitosamente!', {
                                      duration: 3000
                                    });
                                  },
                                  onError: (error: any) => {
                                    toast.error(`❌ Error al votar en poll:\n${error.message}`, {
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
                                    toast.success('✅ Voto "Rechazar" registrado exitosamente!', {
                                      duration: 3000
                                    });
                                  },
                                  onError: (error: any) => {
                                    toast.error(`❌ Error al votar en poll:\n${error.message}`, {
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