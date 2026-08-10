#!/usr/bin/env node

/**
 * Canopy Network CLI
 */

import { defaultAppchainLauncher } from '../src/core/appchain-launcher.js';
import { defaultValidatorMesh } from '../src/core/mesh-validator.js';

const args = process.argv.slice(2);
const command = args[0] || 'help';

async function main() {
  switch (command.toLowerCase()) {
    case 'chains': {
      console.log('\n🌲 Active Canopy Nested Appchains:');
      defaultAppchainLauncher.getNestedChains().forEach(c => {
        console.log(`  • [${c.id}] ${c.name}`);
        console.log(`    Type:       ${c.chainType}`);
        console.log(`    Validators: ${c.validators}`);
        console.log(`    Staked:     ${c.stakedCnpy}\n`);
      });
      break;
    }

    case 'launch': {
      const name = args[1] || 'MyNestedAppchain';
      const category = args[2] || 'DeFi & Trading';
      console.log(`\n🚀 Launching Recursive Nested Appchain '${name}' on Canopy Seed Chain...`);
      const res = defaultAppchainLauncher.launchAppchain({ name, category, initialValidatorCount: 10, minStakeAmount: 500000 });
      console.log(`  Appchain ID:   ${res.appchain.id}`);
      console.log(`  Genesis Hash:  ${res.appchain.genesisHash}`);
      console.log(`  Register TX:   ${res.appchain.registerTx}`);
      console.log(`  Status:        ${res.appchain.status}\n`);
      break;
    }

    case 'mesh': {
      console.log('\n🛡️ Canopy Shared Security Mesh Metrics:');
      const mesh = defaultValidatorMesh.getSecurityMeshStatus();
      console.log(`  Active Validators: ${mesh.activeValidatorsCount}`);
      console.log(`  Total Staked:      ${mesh.totalStakedCNPY}`);
      console.log(`  BFT Block Time:    ${mesh.bftFinalityTimeMs} ms`);
      console.log(`  Security Sharing:  ${mesh.securitySharingRatio}\n`);
      break;
    }

    case 'studio': {
      console.log('\n🌐 Launching Canopy Studio on :3416...');
      await import('../src/server/app.js');
      break;
    }

    default: {
      console.log(`
╔══════════════════════════════════════════════════════════════════╗
║               🌲 CANOPY NETWORK APPCHAIN CLI                     ║
║       Recursive Nested Chain Launcher & Security Mesh Suite      ║
╚══════════════════════════════════════════════════════════════════╝

Commands:
  canopy-cli chains                    List active nested appchains
  canopy-cli launch [name] [category]  Deploy a new nested sovereign L1 appchain
  canopy-cli mesh                      View shared security mesh metrics
  canopy-cli studio                    Launch Interactive Web Studio on :3416
      `);
      break;
    }
  }
}

main().catch(err => {
  console.error('Error:', err.message);
  process.exit(1);
});
