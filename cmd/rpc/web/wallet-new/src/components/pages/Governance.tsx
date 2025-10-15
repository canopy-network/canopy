import React, { useState } from 'react';
import { useManifest } from '@/hooks/useManifest';
import { useGovernanceData } from '@/hooks/useGovernance';
import ActiveProposals from './governance/ActiveProposals';
import ActivePolls from './governance/ActivePolls';
import PastProposals from './governance/PastProposals';
import PastPolls from './governance/PastPolls';
import CreateProposalModal from '@/components/modals/CreateProposalModal';
import CreatePollModal from '@/components/modals/CreatePollModal';

export default function Governance(): JSX.Element {
  const { getText } = useManifest();
  const { proposals, poll } = useGovernanceData('50002');
  const [isCreateProposalOpen, setIsCreateProposalOpen] = useState(false);
  const [isCreatePollOpen, setIsCreatePollOpen] = useState(false);

  const loading = proposals.isLoading || poll.isLoading;

  const handleCreateSuccess = () => {
    // Refetch data after successful creation
    proposals.refetch();
    poll.refetch();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">{getText('ui.governance.title', 'Active Proposals')}</h1>
          <p className="text-text-muted text-sm">{getText('ui.governance.subtitle', 'Vote on proposals that shape the future of the Canopy ecosystem')}</p>
        </div>
        <div className='flex lg:flex-row flex-col justify-between gap-4 lg:w-[49%] w-full'>
          <h1 className="text-2xl font-bold text-text-primary">{getText('ui.governance.sections.activePolls', 'Active Polls')}</h1>
          <div className='flex gap-4'>
            <button 
              onClick={() => setIsCreatePollOpen(true)}
              className="bg-primary text-muted px-3 font-medium py-1 rounded-md text-sm hover:bg-primary/80"
            >
              <i className="fa-solid fa-download"></i>
              {getText('ui.governance.labels.createPoll', 'Create a Poll')}
            </button>
            <button 
              onClick={() => setIsCreateProposalOpen(true)}
              className="bg-primary text-muted px-3 py-1 font-medium rounded-md text-sm hover:bg-primary/80"
            >
              <i className="fa-solid fa-download"></i>
              {getText('ui.governance.labels.createProposal', 'Create a Proposal')}
            </button>
          </div>
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="bg-bg-secondary rounded-xl border border-bg-accent h-64 animate-pulse" />
          <div className="bg-bg-secondary rounded-xl border border-bg-accent h-64 animate-pulse" />
          <div className="bg-bg-secondary rounded-xl border border-bg-accent h-64 animate-pulse" />
          <div className="bg-bg-secondary rounded-xl border border-bg-accent h-64 animate-pulse" />
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <ActiveProposals proposals={proposals.data || {}} />
            <ActivePolls polls={poll.data || {}} />
          </div>
          <div className="flex flex-col gap-6">
            <PastProposals proposals={proposals.data || {}} />
            <PastPolls polls={poll.data || {}} />
          </div>
        </>
      )}

      {/* Modals */}
      <CreateProposalModal
        isOpen={isCreateProposalOpen}
        onClose={() => setIsCreateProposalOpen(false)}
        onSuccess={handleCreateSuccess}
      />
      <CreatePollModal
        isOpen={isCreatePollOpen}
        onClose={() => setIsCreatePollOpen(false)}
        onSuccess={handleCreateSuccess}
      />
    </div>
  );
}


