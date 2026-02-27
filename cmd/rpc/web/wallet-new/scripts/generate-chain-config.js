#!/usr/bin/env node

/**
 * Generate chain.json with RPC URLs from environment variables
 * This script runs before the build to inject the correct RPC endpoints
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Read environment variables
const rpcTarget = process.env.VITE_WALLET_RPC_PROXY_TARGET || 'http://localhost:50002';
const adminRpcTarget = process.env.VITE_WALLET_ADMIN_RPC_PROXY_TARGET || 'http://localhost:50003';
const rootRpcTarget = process.env.VITE_ROOT_WALLET_RPC_PROXY_TARGET || 'http://localhost:50002';

// Path to chain.json template and output
const templatePath = path.join(__dirname, '../public/plugin/canopy/chain.json.template');
const outputPath = path.join(__dirname, '../public/plugin/canopy/chain.json');

// Check if template exists, if not use the current chain.json as template
let chainConfig;
if (fs.existsSync(templatePath)) {
  chainConfig = JSON.parse(fs.readFileSync(templatePath, 'utf-8'));
} else if (fs.existsSync(outputPath)) {
  // Use existing chain.json as template
  chainConfig = JSON.parse(fs.readFileSync(outputPath, 'utf-8'));
} else {
  console.error('Error: chain.json not found');
  process.exit(1);
}

// Update RPC URLs
chainConfig.rpc = {
  base: rpcTarget,
  admin: adminRpcTarget,
  root: rootRpcTarget
};

// Write the updated config
fs.writeFileSync(outputPath, JSON.stringify(chainConfig, null, 2));

console.log(`✅ Generated chain.json with RPC targets:`);
console.log(`   - base: ${rpcTarget}`);
console.log(`   - admin: ${adminRpcTarget}`);
console.log(`   - root: ${rootRpcTarget}`);
