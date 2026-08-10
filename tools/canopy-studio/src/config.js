/**
 * Canopy Network Architecture & Nested Chains Configuration
 */

export const CANOPY_CONFIG = {
  network: {
    name: 'Canopy Main Seed Chain',
    chainId: 'canopy-seed-1',
    consensus: 'Canopy BFT (Byzantine Fault Tolerant PoS)',
    blockTimeMs: 1200,
    nativeCurrency: 'CNPY',
  },
  nestedChains: [
    {
      id: 'chain_defi_hub',
      name: 'Canopy Financial (DeFi Hub Appchain)',
      chainType: 'Nested Sovereign L1',
      validators: 24,
      stakedCnpy: '12,500,000 CNPY',
      status: 'ACTIVE_NESTED_CHAIN',
    },
    {
      id: 'chain_gaming_mesh',
      name: 'Canopy Gaming & Virtual Worlds',
      chainType: 'Nested Sovereign L1',
      validators: 16,
      stakedCnpy: '8,200,000 CNPY',
      status: 'ACTIVE_NESTED_CHAIN',
    },
    {
      id: 'chain_rwa_privacy',
      name: 'Confidential Real-World Assets Appchain',
      chainType: 'Confidential Nested L1',
      validators: 12,
      stakedCnpy: '15,000,000 CNPY',
      status: 'ACTIVE_NESTED_CHAIN',
    },
  ],
};
