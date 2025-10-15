import { useQuery } from '@tanstack/react-query';

interface ProposalMap {
  [hash: string]: {
    proposal: any;
    approve: boolean;
  };
}

interface PollMap {
  [hash: string]: {
    proposalHash: string;
    proposalURL: string;
    accounts: any;
    validators: any;
  };
}

export const useGovernanceData = (queryPort: string = '50002') => {
  const base = `http://localhost:${queryPort}`;

  const proposals = useQuery<ProposalMap>({
    queryKey: ['gov', 'proposals', queryPort],
    queryFn: async () => {
      const res = await fetch(`${base}/v1/gov/proposals`, { method: 'GET' });
      if (!res.ok) throw new Error('failed gov/proposals');
      return res.json();
    },
    refetchInterval: 30000,
    staleTime: 15000,
  });

  const poll = useQuery<PollMap>({
    queryKey: ['gov', 'poll', queryPort],
    queryFn: async () => {
      const res = await fetch(`${base}/v1/gov/poll`, { method: 'GET' });
      if (!res.ok) throw new Error('failed gov/poll');
      return res.json();
    },
    refetchInterval: 30000,
    staleTime: 15000,
  });

  return { proposals, poll };
};


