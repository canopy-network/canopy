/**
 * Canopy Recursive Appchain Launcher Engine
 */

import crypto from 'crypto';
import { CANOPY_CONFIG } from '../config.js';

export class CanopyAppchainLauncher {
  constructor() {
    this.nestedChains = new Map();
    CANOPY_CONFIG.nestedChains.forEach(c => this.nestedChains.set(c.id, { ...c }));
    this.deployedLogs = [];
  }

  /**
   * Deploy a new Recursive Nested Appchain on Canopy
   */
  launchAppchain({ name, category, initialValidatorCount, minStakeAmount }) {
    if (!name) throw new Error('Appchain name is required');

    const chainId = `chain_${name.toLowerCase().replace(/[^a-z0-9]/g, '_')}_${Date.now()}`;
    const genesisHash = '0x' + crypto.randomBytes(32).toString('hex');
    const registerTx = '0x' + crypto.randomBytes(32).toString('hex');

    const appchain = {
      id: chainId,
      name,
      category: category || 'General Appchain',
      chainType: 'Nested Sovereign L1',
      validators: parseInt(initialValidatorCount) || 5,
      stakedCnpy: `${parseFloat(minStakeAmount || 100000).toLocaleString()} CNPY`,
      genesisHash,
      registerTx,
      status: 'ACTIVE_NESTED_CHAIN',
      createdAt: new Date().toISOString(),
    };

    this.nestedChains.set(chainId, appchain);

    const log = {
      id: `launch_${Date.now()}`,
      appchain,
      timestamp: new Date().toISOString(),
      status: 'REGISTERED_ON_CANOPY_SEED',
    };

    this.deployedLogs.unshift(log);
    return { success: true, appchain, log };
  }

  getNestedChains() {
    return Array.from(this.nestedChains.values());
  }

  getLaunchHistory() {
    return this.deployedLogs;
  }
}

export const defaultAppchainLauncher = new CanopyAppchainLauncher();
