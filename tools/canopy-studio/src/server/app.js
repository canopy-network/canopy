/**
 * Canopy Appchain Studio Web Server
 */

import express from 'express';
import cors from 'cors';
import path from 'path';
import { fileURLToPath } from 'url';
import { CANOPY_CONFIG } from '../config.js';
import { defaultAppchainLauncher } from '../core/appchain-launcher.js';
import { defaultValidatorMesh } from '../core/mesh-validator.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const WEB_ROOT = path.join(__dirname, '../../web');

const app = express();
const PORT = process.env.PORT || 3416;

app.use(cors());
app.use(express.json());
app.use(express.static(WEB_ROOT));

// 1. Get Network Config & Active Nested Chains
app.get('/api/config', (req, res) => {
  res.json({
    network: CANOPY_CONFIG.network,
    nestedChains: defaultAppchainLauncher.getNestedChains(),
    securityMesh: defaultValidatorMesh.getSecurityMeshStatus(),
  });
});

// 2. Launch New Nested Appchain
app.post('/api/appchain/launch', (req, res) => {
  try {
    const result = defaultAppchainLauncher.launchAppchain(req.body);
    res.json(result);
  } catch (err) {
    res.status(400).json({ error: err.message });
  }
});

// 3. Security Mesh Metrics
app.get('/api/mesh', (req, res) => {
  res.json(defaultValidatorMesh.getSecurityMeshStatus());
});

// 4. Launch History
app.get('/api/appchain/history', (req, res) => {
  res.json(defaultAppchainLauncher.getLaunchHistory());
});

if (process.env.NODE_ENV !== 'test') {
  app.listen(PORT, () => {
    console.log(`\n======================================================`);
    console.log(`🌲 Canopy Network Appchain & Security Mesh Studio Running!`);
    console.log(`🌐 Web Dashboard: http://localhost:${PORT}`);
    console.log(`⛓️  Architecture: Recursive Nested Sovereign L1 Appchains`);
    console.log(`======================================================\n`);
  });
}

export default app;
