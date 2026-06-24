import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  canoliqSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Core Concepts',
      link: {type: 'doc', id: 'concepts/how-it-works'},
      items: [
        'concepts/how-it-works',
        'concepts/two-tokens',
        'concepts/glossary',
      ],
    },
    {
      type: 'category',
      label: 'Getting Started',
      link: {type: 'doc', id: 'getting-started/overview'},
      items: [
        'getting-started/overview',
        'getting-started/building',
        'getting-started/committee-setup',
      ],
    },
    {
      type: 'category',
      label: 'Tokenomics',
      link: {type: 'doc', id: 'tokenomics/overview'},
      items: [
        'tokenomics/overview',
        'tokenomics/fee-structure',
        'tokenomics/vote-escrow',
        'tokenomics/vesting',
      ],
    },
    {
      type: 'category',
      label: 'Transactions',
      link: {type: 'doc', id: 'transactions/overview'},
      items: [
        'transactions/deposit-redeem',
        'transactions/cplq-operations',
        'transactions/reference',
      ],
    },
    {
      type: 'category',
      label: 'Governance',
      link: {type: 'doc', id: 'governance/overview'},
      items: [
        'governance/overview',
        'governance/governance-tiers',
        'governance/proposals',
        'governance/tally-execution',
      ],
    },
    {
      type: 'category',
      label: 'Advanced',
      link: {type: 'generated-index'},
      items: [
        'advanced/buyback',
        'advanced/treasury',
        'advanced/insurance',
        'advanced/tvl-cap',
        'advanced/autonomy-graduation',
        'advanced/alerts',
        'advanced/state-keys',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      link: {type: 'doc', id: 'api/overview'},
      items: [
        'api/overview',
        'api/endpoints',
      ],
    },
    {
      type: 'category',
      label: 'Protobuf Reference',
      link: {type: 'generated-index'},
      items: [
        'proto/messages',
        'proto/types',
      ],
    },
    'implementation-plan',
  ],
};

export default sidebars;
