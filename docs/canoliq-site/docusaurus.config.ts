import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'canoLiq',
  tagline: 'Liquid Staking Protocol on Canopy Network',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://canopy-network.github.io',
  baseUrl: '/canopy/',

  organizationName: 'canopy-network',
  projectName: 'canopy',

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/canopy-network/canopy/tree/main/docs/canoliq-site/',
          routeBasePath: 'docs',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/canoliq-social-card.png',
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'canoLiq',
      logo: {
        alt: 'canoLiq Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'canoliqSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://github.com/canopy-network/canopy',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Introduction', to: '/docs/intro'},
            {label: 'Getting Started', to: '/docs/getting-started/overview'},
          ],
        },
        {
          title: 'Community',
          items: [
            {label: 'Discord', href: 'https://discord.gg/pNcSJj7Wdh'},
            {label: 'X', href: 'https://x.com/CNPYNetwork'},
          ],
        },
        {
          title: 'More',
          items: [
            {label: 'Canopy Network', href: 'https://canopynetwork.org'},
            {label: 'GitHub', href: 'https://github.com/canopy-network/canopy'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Canopy Network. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'protobuf'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
