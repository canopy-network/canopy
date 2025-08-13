# Summary: whitepaper.txt and phases.txt

## Summary: whitepaper.txt

### Abstract and Introduction

• **Blockchain ecosystem paradox identified after 15 years**
  - Multi-chain future deemed essential for scalability
  - Most new projects retrofit established ecosystems like Ethereum
  - Absence of framework supporting full lifecycle from dApp to autonomous L1

• **Canopy Network's core solution**
  - Framework offering shared security service to blockchain applications
  - Seamless track to L1 independence through unique consensus algorithm
  - Plugin-based architecture enabling one-way integration with external chains
  - Progressive autonomy solution replacing financial barriers with community popularity

### Historical Context

• **Bitcoin's foundational paradigm (2009)**
  - Introduced peer-to-peer digital currency with Proof of Work
  - Solved Byzantine General's Problem through cryptographic principles
  - Limited by lack of programmability, energy efficiency, and scalability

• **Ethereum's programmable expansion (2013)**
  - Evolved from UTXO ledgers to multi-functional digital accounts
  - Enabled smart contracts and decentralized applications (dApps)
  - Suffered from congestion, high fees, and denial of service under load

• **Tendermint's modular approach**
  - Byzantine fault tolerant state machine replicator
  - Emphasized multichain model for scalability and interoperability
  - Provided SDK for L1 development but lacked ease of deployment and secure bootstrapping

• **Cosmos Network's interoperability vision (2016)**
  - "Internet of Blockchains" with Inter-Blockchain Communication (IBC)
  - Enabled cross-chain functionality and multi-chain communication
  - Still required building own state-machine and governance mechanisms

• **Polkadot's shared security model (2020)**
  - Multi-chain shared security through PoS base-chain
  - Sub-chains operate separately while using validator set security
  - Limited by paid service model with slot auctions and high economic barriers

• **Rollups and scaling solutions**
  - Offload computation while retaining base-chain security guarantees
  - More tightly coupled to base-chain than Polkadot sub-chains
  - Often rely on centralized sequencers, compromising decentralization

• **Avalanche's validator marketplace**
  - Platform for independent sub-chains with credible validator marketplace
  - Sub-chains attract base-chain validators through native token rewards
  - Questions about security improvement and validator allegiance switching

• **EigenLayer's restaking economics (2023)**
  - Validators reuse bonded tokens to secure additional services
  - Capital-efficient architecture exclusive to Ethereum ecosystem
  - Does not address interoperability challenges

### Layer 1 vs Dependent Applications Analysis

• **L1 independence costs**
  - High economic value and diverse validator set prerequisites
  - Enormous fundraising events required for security guarantees
  - Complex development requirements: state machine, consensus, P2P, Merkle tree

• **Dependent Apps limitations**
  - Introduces single point of failure with external protocol dependency
  - Users rely on third party for permissionless interaction
  - Limited representation in governance and upgrade barriers
  - Permanent dependence on host for historical data management

• **Real-world censorship examples**
  - 2022 Ethereum OFAC compliance pressure
  - Over 50% of blocks censored Tornado Cash transactions
  - Demonstrated centralized control over non-autonomous entities

• **Scalability bottlenecks in unichains**
  - All Dependent Apps compete for same computational power and storage
  - CryptoKitties (2017) caused network congestion and high gas fees
  - Decentraland faced similar scalability challenges leading to L2 migration

### Migration Challenges

• **CryptoKitties escape path (2017)**
  - Complete rewrite of technology to develop L1 blockchain 'Flow'
  - Three-year development and migration process
  - Potential missed opportunity for greater success

• **Decentraland's ongoing migration (2021-present)**
  - Pivot to Polygon L2 solution requiring significant technical rewrites
  - Compromise on decentralization with centralized sequencer dependency
  - Migration still ongoing three years later with unclear timeline

### Technical Architecture Overview

• **Canopy's core innovation**
  - Shared security service powered by novel restaking economic model
  - Fully programmable, modular framework for blockchain development
  - Advanced mechanisms: chain-halt rescue and long-range attack mitigation

• **Plugin-based communication model**
  - Validators interact with sub-chain nodes through bilateral APIs
  - Offloads state machine operation and block storage to sub-chain software
  - Manages consensus and peer-to-peer layers at base-chain level

• **Progressive autonomy features**
  - Predefined track for sub-chain graduation to independence
  - Permissionless integrations with existing chains without modifications
  - Committee organization of validators using same plugin

### Consensus and Security

• **NestBFT consensus algorithm**
  - Designed to withstand grinding, DDoS, and long-range attacks
  - Engineered for unreliable peer-to-peer environments
  - BLS multisignature aggregation for O(1) space complexity

• **Leader election mechanism**
  - DDoS resistant using simplified Verifiable Random Function
  - Seed data: last_proposer_addresses, consensus height, and round
  - Fallback to stake_weighted_random algorithm when no candidates

• **Long-range attack mitigation**
  - Verifiable Delay Functions ensure temporal consistency
  - Each block requires sequential proof of elapsed time
  - Checkpointing-as-a-service for sub-chains with exportable historical data

### Tokenomics and Incentives

• **CNPY token fundamentals**
  - No pre-mint or pre-mine following fair launch principles
  - Fixed total supply with regular halvings similar to Bitcoin
  - Block reward: 71,429 CNPY per block, halved every 1,050,000 blocks

• **Fair distribution model**
  - New CNPY distributed evenly among subsidized sub-chain committees
  - Committee subsidization based on predefined stake threshold
  - Supports SUBSIDIZE_TRANSACTION for additional community funding

• **Restaking economics**
  - Validators reuse staked collateral across multiple sub-chains
  - Eliminates economic barriers to entry for new sub-chains
  - Immediate economic security and access to experienced operators

• **Long-term sustainability mechanisms**
  - Sub-chain block rewards align incentives beyond base-chain rewards
  - Community-driven subsidies create two-sided marketplace
  - Flexible reward distribution enables continued interaction post-graduation

### Governance Structure

• **Validator-based governance**
  - Validators act as elected representatives for sub-chains
  - Authorize PARAMETER_CHANGE and DAO_TRANSFER transactions
  - Built-in on-chain polling for transparent community sentiment

• **Multi-chain DAO potential**
  - Interlinking validator committees across sub-chains
  - "DAO of DAOs" - United Nations model for L1s
  - Unprecedented governance stability and multi-protocol cooperation

### Addressing Common Concerns

• **Economic security with graduation**
  - Sub-chains provide native token incentives throughout lifecycle
  - Flexible reward distribution maintains relationship post-graduation
  - API compatibility enables ongoing plugin operations

• **Validator centralization risks**
  - More resilient than single validator set systems like Ethereum/Polkadot
  - Safety eject feature prevents cascading failures
  - Removes slashed validators above threshold to protect network integrity

• **FPGA/VDF vulnerabilities**
  - Takes advantage of fastest hardware on base-chain committee
  - Social consensus fallback with bi-annual checkpointing
  - Vast improvement over legacy checkpoint-dependent systems

---

## Summary: phases.txt

### NestBFT Consensus Process Overview

• **Core structure organization**
  - 8 core phases and 2 recovery phases in consensus process
  - Each phase represents smallest unit in consensus process
  - Sequential execution of phases to achieve block consensus

### Core Consensus Phases

• **ELECTION phase**
  - Each replica runs Verifiable Random Function (VRF)
  - Selected candidates send VRF output to other replicas
  - Determines potential leaders through cryptographic lottery

• **ELECTION-VOTE phase**
  - Replicas send ELECTION votes for leader with lowest VRF value
  - Fallback to stake-weighted-pseudorandom selection if no candidates
  - Establishes leader through signature-based voting

• **PROPOSE phase**
  - Leader collects +2/3 ELECTION VOTES including lock, evidence, signature
  - Uses valid lock for current height meeting SAFE NODE PREDICATE
  - Creates new block if no valid lock found, sends proposal with justification

• **PROPOSE-VOTE phase**
  - Replicas validate PROPOSE message through aggregate signature verification
  - Apply proposal block against state machine and verify header/results
  - Send vote to leader vouching for proposal validity

• **PRECOMMIT phase**
  - Leader collects +2/3 PROPOSE VOTES with signatures
  - Sends PRECOMMIT message with aggregated signatures
  - Justifies that supermajority believes proposal is valid

• **PRECOMMIT-VOTE phase**
  - Replicas validate PRECOMMIT message through signature verification
  - Send vote confirming evidence of supermajority proposal belief
  - Establishes quorum agreement on proposal validity

• **COMMIT phase**
  - Leader collects +2/3 PRECOMMIT VOTES with signatures
  - Sends COMMIT message with aggregated PRECOMMIT VOTE signatures
  - Justifies supermajority agreement on proposal validity

• **COMMIT PROCESS phase**
  - Replicas validate COMMIT message through signature verification
  - Commit block to finality upon validation
  - Reset BFT mechanism for next height

### Recovery Mechanisms

• **ROUND INTERRUPT recovery phase**
  - Triggered by BFT cycle failure causing premature round exit
  - Results in new round with extended sleep time between phases
  - Replicas send View to all others for round synchronization

• **PACEMAKER recovery phase**
  - Follows ROUND_INTERRUPT for coordination recovery
  - Calculates highest round seen by supermajority
  - Jumps to calculated round to resolve synchronization issues