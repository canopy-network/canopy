import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';
import { useManifest } from './useManifest';

interface CreateProposalData {
  paramSpace: string;
  paramKey: string;
  paramValue: string;
  startHeight: number;
  endHeight: number;
  memo: string;
  password: string;
}

interface CreatePollData {
  proposal: string;
  endBlock: number;
  url: string;
  password: string;
}

interface VotePollData {
  pollJSON: string;
  pollApprove: boolean;
  password: string;
}

export const useGovernanceActions = (adminPort: string = '50003', queryPort: string = '50002') => {
  const { activeAccount } = useAccounts();
  const { manifest } = useManifest();
  const queryClient = useQueryClient();
  const base = `http://localhost:${adminPort}`;

  // Get governance actions from manifest
  const governanceAction = manifest?.actions?.find((action: any) => action.id === 'Governance');
  const createProposalAction = governanceAction?.actions?.find((action: any) => action.id === 'CreateProposal');
  const createPollAction = governanceAction?.actions?.find((action: any) => action.id === 'CreatePoll');
  const addVoteAction = governanceAction?.actions?.find((action: any) => action.id === 'AddVote');
  const deleteVoteAction = governanceAction?.actions?.find((action: any) => action.id === 'DeleteVote');
  const votePollAction = governanceAction?.actions?.find((action: any) => action.id === 'VotePoll');

  const createProposal = useMutation({
    mutationFn: async (data: CreateProposalData) => {
      if (!activeAccount) throw new Error('No active account');
      if (!createProposalAction?.rpc) throw new Error('CreateProposal action not found in manifest');
      
      // Validate address format (should be 40 hex characters)
      if (!/^[0-9a-fA-F]{40}$/.test(activeAccount.address)) {
        throw new Error(`Invalid address format: ${activeAccount.address}. Expected 40 hex characters.`);
      }

      const endpoint = `${base}${createProposalAction.rpc.path}`;
      const payload = {
        address: activeAccount.address,
        paramSpace: data.paramSpace,
        paramKey: data.paramKey,
        paramValue: String(data.paramValue), // Convert to string
        startHeight: data.startHeight,
        endHeight: data.endHeight,
        memo: data.memo,
        fee: 0,
        submit: true,
        password: data.password
      };

      const response = await fetch(endpoint, {
        method: createProposalAction.rpc.method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Proposal creation failed:', response.status, errorText);
        throw new Error(`Failed to create proposal: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      console.log('Proposal created successfully:', result);
      return result;
    },
    onSuccess: () => {
      // Invalidate and refetch governance data with correct query keys
      console.log('Invalidating governance queries after proposal creation...');
      queryClient.invalidateQueries({ queryKey: ['gov'] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'proposals', queryPort] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'poll', queryPort] });
    },
  });

  const createPoll = useMutation({
    mutationFn: async (data: CreatePollData) => {
      if (!activeAccount) throw new Error('No active account');
      if (!createPollAction?.rpc) throw new Error('CreatePoll action not found in manifest');

      const pollJSON = {
        proposal: data.proposal,
        endBlock: data.endBlock,
        URL: data.url
      };

      const endpoint = `${base}${createPollAction.rpc.path}`;
      const payload = {
        address: activeAccount.address,
        pollJSON: pollJSON, // Send as object, not string
        fee: 0,
        submit: true,
        password: data.password
      };

      console.log('Creating poll with payload:', payload);
      console.log('PollJSON object:', pollJSON);
      console.log('PollJSON stringified:', JSON.stringify(pollJSON));
      console.log('Endpoint:', endpoint);

      const response = await fetch(endpoint, {
        method: createPollAction.rpc.method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Poll creation failed:', response.status, errorText);
        throw new Error(`Failed to create poll: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      console.log('Poll created successfully:', result);
      console.log('Transaction hash:', result);
      console.log('Note: Poll will appear in the list once the transaction is included in a block');
      return result;
    },
    onSuccess: () => {
      // Invalidate and refetch governance data with correct query keys
      console.log('Invalidating governance queries after poll creation...');
      queryClient.invalidateQueries({ queryKey: ['gov'] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'proposals', queryPort] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'poll', queryPort] });
    },
  });

  const votePoll = useMutation({
    mutationFn: async (data: VotePollData) => {
      if (!activeAccount) throw new Error('No active account');
      if (!votePollAction?.rpc) throw new Error('VotePoll action not found in manifest');

      const endpoint = `${base}${votePollAction.rpc.path}`;
      const payload = {
        address: activeAccount.address,
        pollJSON: data.pollJSON,
        pollApprove: data.pollApprove,
        fee: 0,
        submit: true,
        password: data.password
      };

      console.log('Voting on poll with payload:', payload);
      console.log('Endpoint:', endpoint);

      const response = await fetch(endpoint, {
        method: votePollAction.rpc.method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Poll vote failed:', response.status, errorText);
        throw new Error(`Failed to vote on poll: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      console.log('Poll vote successful:', result);
      return result;
    },
    onSuccess: () => {
      // Invalidate and refetch governance data with correct query keys
      console.log('Invalidating governance queries after poll vote...');
      queryClient.invalidateQueries({ queryKey: ['gov'] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'proposals', queryPort] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'poll', queryPort] });
    },
  });

  // Add vote to proposal (using manifest)
  const addVote = useMutation({
    mutationFn: async (data: { proposal: any; approve: boolean }) => {
      if (!addVoteAction?.rpc) throw new Error('AddVote action not found in manifest');

      const endpoint = `${base}${addVoteAction.rpc.path}`;
      const payload = {
        approve: data.approve,
        proposal: data.proposal
      };

      console.log('Adding vote with payload:', payload);
      console.log('Endpoint:', endpoint);

      const response = await fetch(endpoint, {
        method: addVoteAction.rpc.method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Add vote failed:', response.status, errorText);
        throw new Error(`Failed to add vote: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      console.log('Vote added successfully:', result);
      return result;
    },
    onSuccess: () => {
      console.log('Invalidating governance queries after adding vote...');
      queryClient.invalidateQueries({ queryKey: ['gov'] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'proposals', queryPort] });
    },
  });

  // Delete vote from proposal (using manifest)
  const deleteVote = useMutation({
    mutationFn: async (data: { proposal: any }) => {
      if (!deleteVoteAction?.rpc) throw new Error('DeleteVote action not found in manifest');

      const endpoint = `${base}${deleteVoteAction.rpc.path}`;
      const payload = {
        proposal: data.proposal
      };

      console.log('Deleting vote with payload:', payload);
      console.log('Endpoint:', endpoint);

      const response = await fetch(endpoint, {
        method: deleteVoteAction.rpc.method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorText = await response.text();
        console.error('Delete vote failed:', response.status, errorText);
        throw new Error(`Failed to delete vote: ${response.status} ${errorText}`);
      }

      const result = await response.json();
      console.log('Vote deleted successfully:', result);
      return result;
    },
    onSuccess: () => {
      console.log('Invalidating governance queries after deleting vote...');
      queryClient.invalidateQueries({ queryKey: ['gov'] });
      queryClient.invalidateQueries({ queryKey: ['gov', 'proposals', queryPort] });
    },
  });

  return {
    createProposal,
    createPoll,
    votePoll,
    addVote,
    deleteVote,
  };
};
