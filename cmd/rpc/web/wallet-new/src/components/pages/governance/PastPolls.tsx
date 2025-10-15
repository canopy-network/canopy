import React from 'react';
import { useManifest } from '@/hooks/useManifest';

interface PastPollsProps {
  polls: Record<string, { proposalHash: string; proposalURL: string; accounts: any; validators: any }>;
}

const Badge = ({ children, tone = 'default' }: { children: React.ReactNode; tone?: 'default' | 'core' | 'treasury' }) => (
  <span
    className={`text-[10px] px-2 py-0.5 rounded-full border ${
      tone === 'core'
        ? 'bg-primary/20 text-primary border-primary/20'
        : tone === 'treasury'
        ? 'bg-orange-500/20 text-orange-400 border-orange-500/20'
        : 'bg-gray-500/20 text-gray-400 border-gray-500/20'
    }`}
  >
    {children}
  </span>
);

export default function PastPolls({ polls }: PastPollsProps): JSX.Element {
  const { getText } = useManifest();
  const title = getText('ui.governance.sections.pastPolls', 'Past Polls');
  
  const entries = Object.entries(polls || {}).slice(0, 5);

  return (
    <div className="bg-bg-secondary rounded-xl border border-bg-accent p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-white text-lg font-bold">{title}</h2>
        <div className="flex items-center gap-3 text-xs">
          <input 
            className="bg-bg-primary border border-bg-accent rounded-md px-3 py-2 w-64 text-white placeholder-text-muted" 
            placeholder={getText('ui.governance.labels.searchPlaceholder', 'Search proposals...')} 
          />
          <div className="relative">
            <button className="bg-gray-500/20 text-gray-400 px-3 py-2 rounded-md border border-gray-500/20 flex items-center gap-2">
              <span>{getText('ui.governance.labels.allCategories', 'All Categories')}</span>
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          </div>
        </div>
      </div>
      
      <table className="w-full text-sm">
        <thead className="text-xs text-text-muted">
          <tr>
            <th className="py-2 text-left font-medium">Proposal</th>
            <th className="py-2 text-left font-medium">Category</th>
            <th className="py-2 text-left font-medium">Result</th>
            <th className="py-2 text-left font-medium">Turnout</th>
            <th className="py-2 text-left font-medium">Ended</th>
            <th className="py-2 text-right font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {entries.map(([hash, data], index) => {
            const pollId = `POLL-${String(index).padStart(3, '0')}`;
            const pollTitle = index === 0 ? 'Enable Cross-Chain Bridges' : 'Increase Staking Rewards';
            const pollDescription = index === 0 
              ? 'Enable bridging to Ethereum and Polygon networks'
              : 'Proposal to increase annual staking rewards from 12% to 15%';
            const category = index === 0 ? 'Core' : 'Treasury';
            const result = Math.random() > 0.5 ? 'Passed' : 'Failed';
            const turnout = `${(Math.random() * 30 + 40).toFixed(1)}%`;
            const ended = index === 0 ? '3 days ago' : '1 week ago';
            
            return (
              <tr key={hash} className="border-t border-bg-accent">
                <td className="py-3">
                  <div className="flex items-center gap-2">
                    <span className="text-white">{pollId}:</span>
                    <span className="text-white font-medium">{pollTitle}</span>
                  </div>
                  <div className="text-text-muted text-xs mt-1">{pollDescription}</div>
                </td>
                <td>
                  <Badge tone={category === 'Core' ? 'core' : 'treasury'}>
                    {category}
                  </Badge>
                </td>
                <td>
                  <span className={`inline-block px-2 py-0.5 rounded-full text-xs ${
                    result === 'Passed' 
                      ? 'bg-primary/20 text-primary' 
                      : 'bg-red-500/20 text-red-400'
                  }`}>
                    {result}
                  </span>
                </td>
                <td className="text-text-muted">{turnout}</td>
                <td className="text-text-muted">{ended}</td>
                <td className="text-right">
                  <button className="text-primary text-xs hover:underline">
                    {getText('ui.governance.labels.viewDetails', 'View Details')}
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}