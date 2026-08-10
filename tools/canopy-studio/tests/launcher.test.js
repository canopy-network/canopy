/**
 * Canopy Appchain Launcher Unit Tests
 */

import { defaultAppchainLauncher } from '../src/core/appchain-launcher.js';
import { defaultValidatorMesh } from '../src/core/mesh-validator.js';

async function runLauncherTests() {
  console.log('Testing Canopy Recursive Appchain Launcher & Validator Mesh...');

  // 1. Deploy Appchain
  const launch = defaultAppchainLauncher.launchAppchain({
    name: 'Quantum Trading Appchain',
    category: 'DeFi Hub',
    initialValidatorCount: 15,
    minStakeAmount: 500000,
  });

  if (!launch.success || !launch.appchain.genesisHash) {
    throw new Error('Recursive appchain deployment failed');
  }

  // 2. Security Mesh
  const mesh = defaultValidatorMesh.getSecurityMeshStatus();
  if (mesh.activeValidatorsCount < 10) {
    throw new Error('Security mesh status query failed');
  }

  console.log(`✅ Canopy Appchain Deployed & Registered on Seed Chain (${launch.appchain.id})!`);
}

runLauncherTests().catch(e => {
  console.error('❌ Launcher Test Failed:', e);
  process.exit(1);
});
