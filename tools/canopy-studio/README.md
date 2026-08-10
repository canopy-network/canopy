# 🌲 Canopy Appchain Studio & Security Mesh

An interactive recursive appchain launcher, **Nested Chain Simulator**, and **Shared Security Mesh Dashboard** for **Canopy Network**.

---

## 🌟 Key Features

- 🌲 **Recursive Appchain Deployment**: Launch custom nested sovereign L1 appchains registered on the Canopy Seed Chain.
- 🛡️ **BFT Shared Security Mesh**: Simulate PoS validator staking, block proposer selection, and 1.2s BFT finality.
- 🌐 **Interactive Web Studio**: Real-time appchain deployer, ecosystem directory, and validator inspector on `http://localhost:3416`.
- ⌨️ **Universal CLI (`canopy-cli`)**: Terminal utility for deploying nested appchains and querying mesh metrics.

---

## 🚀 Quickstart

```bash
# Launch Canopy Studio
npm start
# Open http://localhost:3416

# Or run via CLI
node bin/canopy-cli.js chains
node bin/canopy-cli.js launch "ApexChain" "DeFi Hub"
node bin/canopy-cli.js mesh
```
