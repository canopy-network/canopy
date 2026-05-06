import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  canoliqSidebar: [
    'intro',
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
        'tokenomics/vesting',
      ],
    },
    {
      type: 'category',
      label: 'Transactions',
      link: {type: 'doc', id: 'transactions/overview'},
      items: [
        'transactions/deposit-redeem',
        'transactions/cliq-operations',
        'transactions/reference',
      ],
    },
    {
      type: 'category',
      label: 'Governance',
      link: {type: 'doc', id: 'governance/overview'},
      items: [
        'governance/overview',
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
