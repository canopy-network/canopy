import React from 'react';
import toast from 'react-hot-toast';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceActions } from '@/hooks/useGovernanceActions';

interface ActiveProposalsProps {
  proposals: Record<string, { proposal: any; approve: boolean }>;
}

const Badge = ({ children, tone = 'default' }: { children: React.ReactNode; tone?: 'default' | 'core' | 'treasury' | 'blue' }) => (
  <span
    className={`text-[10px] px-2 py-0.5 rounded-full border ${tone === 'core'
      ? 'bg-primary/20 text-primary border-primary/20'
      : tone === 'treasury'
        ? 'bg-orange-500/20 text-orange-400 border-orange-500/20'
        : tone === 'blue'
          ? 'bg-blue-500/20 text-blue-400 border-blue-500/20'
          : 'bg-gray-500/20 text-gray-400 border-gray-500/20'
      }`}
  >
    {children}
  </span>
);

export default function ActiveProposals({ proposals }: ActiveProposalsProps): JSX.Element {
  const { getText } = useManifest();
  const { addVote, deleteVote } = useGovernanceActions('50003', '50002');


  const entries = Object.entries(proposals || {});

  return (
    <div className="max-h-[40rem] overflow-y-auto">
      <div className="space-y-4">
        {entries.length === 0 && (
          <div className="text-text-muted text-sm">
            {getText('ui.governance.empty.noActiveProposals', 'No active proposals')}
          </div>
        )}
        {entries.map(([hash, data], index) => {
          const key = data.proposal?.msg?.parameterKey || data.proposal?.type || 'protocolVersion';
          const proposalId = `PROP-${String(index + 1).padStart(3, '0')}`;

          // Use real data from API - no calculations
          const timeRemaining = '0d 0h 0m'; // No time data in API response

          const hasVoted = data.approve !== undefined;
          const voteType = hasVoted ? (data.approve ? 'For' : 'Against') : null;

          return (
            <div key={hash} className="bg-bg-secondary flex flex-col rounded-lg border border-bg-accent p-6 min-h-[15rem] justify-between">
              {/* Header */}
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <Badge tone="core">{proposalId}</Badge>
                  <Badge tone="blue">Core</Badge>
                </div>
                <div className="text-right">
                  <div className="text-white text-sm">{timeRemaining}</div>
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
                    {getText('ui.governance.labels.details', 'Details')}
                  </button>
                  <button className="hover:text-white flex items-center gap-1">
                    <i className="fa-brands fa-discord"></i>
                    {getText('ui.governance.labels.discuss', 'Discuss')}
                  </button>
                </div>

                        <div className="flex items-center gap-2">
                          {hasVoted ? (
                            <>
                              <button className={`${voteType === 'For' ? 'bg-primary/20 text-primary' : 'bg-red-500/20 text-red-400'} px-3 py-1 rounded-md text-sm`}>
                                {voteType}
                              </button>
                              <button 
                                onClick={() => {
                                  deleteVote.mutate(
                                    { proposal: data.proposal },
                                    {
                                      onSuccess: () => {
                                        toast.success('✅ Voto eliminado exitosamente!', {
                                          duration: 3000
                                        });
                                      },
                                      onError: (error: any) => {
                                        toast.error(`❌ Error al eliminar voto:\n${error.message}`, {
                                          duration: 4000,
                                          style: { whiteSpace: 'pre-line' }
                                        });
                                      }
                                    }
                                  );
                                }}
                                className="bg-gray-500/20 text-gray-400 px-3 py-1 rounded-md text-sm hover:bg-gray-500/30"
                              >
                                Delete
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
                                        toast.success('✅ Voto "A favor" registrado exitosamente!', {
                                          duration: 3000
                                        });
                                      },
                                      onError: (error: any) => {
                                        toast.error(`❌ Error al votar:\n${error.message}`, {
                                          duration: 4000,
                                          style: { whiteSpace: 'pre-line' }
                                        });
                                      }
                                    }
                                  );
                                }}
                                className="bg-primary/20 text-primary px-3 py-1 rounded-md text-sm hover:bg-primary/30"
                              >
                                {getText('ui.governance.labels.vote', 'Vote')}
                              </button>
                              <button 
                                onClick={() => {
                                  addVote.mutate(
                                    { proposal: data.proposal, approve: false },
                                    {
                                      onSuccess: () => {
                                        toast.success('✅ Voto "En contra" registrado exitosamente!', {
                                          duration: 3000
                                        });
                                      },
                                      onError: (error: any) => {
                                        toast.error(`❌ Error al votar:\n${error.message}`, {
                                          duration: 4000,
                                          style: { whiteSpace: 'pre-line' }
                                        });
                                      }
                                    }
                                  );
                                }}
                                className="bg-red-500/20 text-red-400 px-3 py-1 rounded-md text-sm hover:bg-red-500/30"
                              >
                                {getText('ui.governance.labels.against', 'Against')}
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