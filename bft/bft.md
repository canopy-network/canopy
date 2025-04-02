### QC

A quorum certificate (QC) is used in the context of Byzantine Fault Tolerance (BFT) as a form of proof within a consensus process. It is typically a collection of cryptographic signatures from validators in the network, indicating that a sufficiently large subset (usually more than two-thirds) of nodes have agreed on a particular value, such as a block in a blockchain.

In the context of the BFT process described, the quorum certificate is used to justify the validity of messages and proposals made by the leader or proposer. When a leader collects enough votes from the replicas (non-leader nodes), forming a +2/3 majority, it creates a quorum certificate. This certificate can then be used to propose, precommit, and finally commit a block. The QC ensures that the transition to the next phase has buy-in from a majority of the network, thus maintaining the integrity and safety of the consensus process.

### Key Features of the `BFT` Struct:

- **Consensus Cycle Management**: The `BFT` struct encapsulates various attributes and methods to conduct consensus over a blockchain network, involving phases such as Election, Proposal, Precommit, Commit, etc.

- **Voting Management**: It tracks and handles votes (`VotesForHeight`) and proposals (`ProposalsForHeight`) as part of the consensus process, ensuring that decisions are made based on a supermajority of +2/3.

- **Quorum Certificates**: Maintains Quorum Certificates (`HighQC`), which are evidence of agreement among replicant nodes, to move safely to the next steps of consensus.

- **Byzantine Fault Tolerance**: Collects evidence of faulty or malicious behavior among validators (`ByzantineEvidence`) to enhance the robustness against attacks.

- **Leadership Election**: Utilizes Verifiable Random Functions (VRF) and Cumulative Distribution Functions (CDF) to elect leaders for proposing new blocks, using `SortitionData`.

- **Synchronization**: Manages timers and synchronization with other nodes using the `PhaseTimer` and various control structures to handle timeouts and reset triggers (`ResetBFT`).

- **Security Enhancements**: Includes Verifiable Delay Functions (VDF) as a measure against long-range attacks by ensuring computational work before making decisions.

### Methods and Workflow:
- Each phase (Election, Propose, Precommit, Commit) is implemented with specific start methods like `StartElectionPhase()`, `StartProposePhase()`, etc., which conduct the respective operations.

- **Phase stepping**: Methods like `HandlePhase()` are responsible for transitioning between phases based on conditions such as phase timeouts or received messages.

- **Timeout Management**: Uses internally-managed timers to optimize synchronization speed and voter participation under varying network conditions.

- **Round and Height Management**: Features mechanisms to reset and increment consensus rounds and heights, adjusting the validator states and decisions accordingly.

Overall, this file provides a structured implementation of a BFT consensus mechanism, focusing on ensuring secure and efficient consensus even in the presence of faults or network delays. While it mentions HotStuff BFT, it appears to implement a tailored variant or part of the broader NestBFT under the Canopy network setup.

# Documentation for `bft.go`

## 1. Description

This file implements [briefly explain the core functionality, e.g., utility functions for string manipulation, the main application logic, data structures for X, etc.].

## 2. Purpose

The main goal of this module is to [explain the specific problem solved or responsibility handled, e.g., centralize all configuration loading, provide an interface for interacting with the database, define the core business logic for user authentication].

## 3. Key Components

### 3.1. Types

#### `type MyStructName`

Represents [describe what the struct represents].

* `FieldName1`: `type` - [Description of field purpose]
* `FieldName2`: `type` - [Description of field purpose]
    * #### `type MyInterfaceName`

Defines the contract for [describe what implementers of this interface should do].

* `MethodName1(params) returnType`: [Description of method purpose]
* `MethodName2(params) returnType`: [Description of method purpose]

### 3.2. Functions

#### `func FunctionName(param1 type, param2 type) (returnType, error)`

This function [describe what the function does].

* **Parameters:**
    * `param1`: [Description of parameter purpose and constraints]
    * `param2`: [Description of parameter purpose and constraints]
* **Returns:**
    * `returnType`: [Description of the successful return value]
    * `error`: [Description of potential errors returned]
* **Usage Notes:** [Any specific context or important considerations when using this function]

### 3.3. Constants & Variables

* `const MyConstant = value`: [Description of the constant's purpose]
* `var myPackageVar type`: [Description of the variable's purpose and scope]

## 4. Usage

To use the functionality provided by this file:

1.  Import the package: `import "[your_module_path]/[package_name]"`
2.  [Explain how to instantiate types or call functions, e.g., Create an instance of `MyStructName`, Call `FunctionName` with appropriate parameters...]
# NestBFT

In the world of blockchain, achieving consensus efficiently and securely is crucial. This is where NestBFT comes in, leveraging voting power and majority votes to reach agreement on the next block in the chain. The approach involves multiple phases where each replica—or participant—interacts and contributes its part to the process.

Each phase in the NestBFT process builds upon the last and sets the stage for the next. For instance, data from the Election phase feeds into ElectionVote, establishing a leader whom the replicas can support or challenge. A super-majority serves as justification for decisions, ensuring consensus integrity at every stage.

## Consensus Phases & Rounds

The consensus process is broken down into 8 core phases and 2 recovery phases. Each phase represents the smallest unit of the consensus process. Each round consists of multiple phases, and each height may consist of multiple rounds. These phases are executed sequentially to achieve consensus on the next block.

Below is a list of each core phase and their primary purpose:

1. **Election**: Replicas gossip candidacy
2. **ElectionVote**: Replicas vote for a leader selected from the pool of gossiped candidates
3. **Propose**: The elected leader puts forth a proposed block for consideration
4. **ProposeVote**: Replicas validate proposed block and sends vote to leader
5. **Precommit**: Leader reviews block votes received from replicas
6. **PrecommitVote**: Replicas validate majority approval and send vote to leader
7. **Commit**: Leader verifies majority vote results
8. **CommitProcess**: Replicas validate majority signature and proceed to commit block

The two recovery phases are used when an error in the consensus process causes a premature exit to the round.

1. **RoundInterrupt**: During this phase each replica sends View to all other replicas to enable synchronization in the Pacemaker phase.

2. **Pacemaker**: This phase synchronizes each replica to the highest round a super-majority has seen and restarts the consensus process beginning with the Election phase.

# bft structure

- **Consensus Management**: NestBFT manages the consensus process for a blockchain system, specifically through a series of ordered phases derived from the Hotstuff protocol, aiming to reach agreement on new blocks efficiently.

- **Validator Coordination**: The structure coordinates between different validators in the network, organizing their votes and proposals through fields like *Votes* and *Proposals* to achieve consensus.

- **Leader Election**: Uses Verifiable Random Functions (VRF) and other cryptographic primitives to determine leadership roles and ensure a fair election process, as seen with fields such as *ProposerKey* and *SortitionData*.

- **Security Assurance**: Ensures the safety of the blockchain system by using cryptographic measures like quorum certificates (such as *HighQC*) and Verifiable Delay Functions (VDF) to protect against various attacks, thereby maintaining the integrity of the system.

- **Block Management**: Handles the current block being voted on and its associated data with fields like *Block*, *BlockHash*, ensuring that each new block is properly proposed and verified according to the consensus rules.

- **Result and Evidence Handling**: Manages the results of consensus rounds and possible slashing conditions through fields like *Results* to ensure accountability and trustworthiness among participants.

- **Decentralized Networking**: Interacts with decentralized network components to handle messages and support peer-to-peer communication within the consensus process.

# BFT Structure

The BFT structure is central to the consensus process, helping organize and facilitate the achievement of agreement among replicas in a decentralized network. It's like the backbone that ensures everyone is on the same page when creating new blocks. Here's a breakdown of its key roles:

- **Manage Current State**: Keeps track of the current period of the consensus process, such as the current height, round, and phase. This helps ensure the network is synchronized and progressing together.

- **Vote and Proposal Handling**: Organizes and records votes and proposals from replicas and the leader. By tracking these, it helps ensure that decisions are made with the required super-majority support, adding legitimacy to the consensus achieved.

- **Leader Election**: Utilizes Verifiable Random Function (VRF) to help select a leader fairly among replicas. It balances randomness and replicator voting power, ensuring that the elected leader can propose new blocks.

- **Verification and Security**: Features mechanisms for validating proposals, certificates, and voting power. This includes methods for handling locks and ensuring that the conditions meet safe node predicates, strengthening security against double-spending or tampering.

- **P2P Communication**: Supports sending messages between replicas for consensus messaging and block gossip, facilitating smooth communication among network participants.

- **External Interaction Management**: Connects with other parts of the network and application layers, interfacing with components like the FSM, P2P network, and storage facilities to streamline operations.
# The Block Proposer

An election is necessary to determine the next block proposer to ensure fair and decentralized decision-making. Without it, control could be manipulated by a single entity, compromising the blockchain's integrity and security.

NestBFT uses a unique election mechanism involving Practical VRF and linear stake-weighted thresholds. This approach enhances fairness by selecting potential leaders based on stake and randomness, protecting against attacks and ensuring decentralized block proposals.

# Election Phase

In the NestBFT consensus algorithm, the election phase leverages a sortition process combined with a Verifiable Random Function (VRF) to ensure a fair, uniform, and unpredictable selection of leaders. The process counts on the unique and non-manipulatable inputs of seed data to resist manipulation, providing a robust defense against potential biases. Validators generate a digital signature on this sortition seed data, where their stake is a critical factor, increasing the likelihood of becoming a candidate based on their voting power.

## Sortition Seed Data

The integrity of the sortition seed data is paramount, as any manipulation could lead to predictable and biased leader selection. By ensuring that the seed data remains secure and non-manipulatable, NestBFT fosters an environment where leadership is assigned fairly, maintaining unpredictability and fairness in the network.

- **Round Field Inclusion**: The incorporation of the round field into the sortition data reduces the likelihood of the same leader being chosen consecutively, promoting leader rotation. This mechanism benefits the network by mitigating the risk posed by a potentially malicious leader or one that contributed to a consensus failure, thus enhancing reliability and trust in the process.

- **Last Proposer Addresses Field**: NestBFT distinguishes itself from other protocols by utilizing the LastProposerAddresses field within its sortition seed data. This approach avoids reliance on manipulable inputs, such as the last block hash, which are susceptible to bias and grinding attacks. By eliminating these vulnerabilities, NestBFT ensures a fairer and less predictable leader selection process.

# Election Vote Phase

In the Election Vote phase, each replica examines candidate messages received from others, setting the stage for leader selection. By choosing the candidate with the lowest VRF output, replicas ensure that the process remains fair and unbiased. This phase seamlessly follows the election phase where potential leaders were identified and prepares the ground for the proposal phase, where the selected leader will propose the next block.

If no candidates are available, the process defaults to stake-weight-pseudorandom selection, ensuring progress is always made. Each replica then forwards its local VDF to the selected proposer, adding to their voting power. Finally, by sending their vote to the chosen proposer, replicas collectively build toward a majority vote, aligning consensus towards the next block proposal.

# Propose Phase

During the PROPOSE phase of the NestBFT consensus algorithm, the leader is responsible for producing a new block proposal. Here are the steps involved in this phase:

1. **Collecting ELECTION.VOTES**: The leader gathers ELECTION.VOTES from more than two-thirds (+2/3) of the replicas. Each vote includes the lock, evidence, and signature from the sender. This serves as proof of the leader's qualification to propose a block.

2. **Proposal Block Selection**:
   - If a valid lock exists for the current height that meets the criteria of Hotstuff's SAFE NODE PREDICATE, the leader uses the locked block as the proposal block.
   - If no valid lock is found, the leader creates a new block to extend the blockchain.

3. **Creating the Proposal**: The proposal consists of the new proposed block and the associated results, which include the reward and slash recipients. A block contains the transactions to be processed.

4. **Distribution of Proposal**: The leader sends the newly created proposal (block, results, evidence) to all validators. The proposal is justified by attaching the signatures from more than two-thirds (+2/3) of the ELECTION.VOTES, confirming the leader's legitimacy.

This phase is crucial as it lays the foundation for achieving consensus on the next block in the blockchain.

# Propose Vote Phase

In the NestBFT consensus algorithm, the PROPOSE VOTE phase involves several critical steps where the replicas (nodes) validate the proposal put forward by the leader. Each replica follows a systematic approach to ensure the proposal's validity:

1. **Proposal Validation**: Each replica receives the PROPOSE message from the leader and must validate it by checking the aggregate signature. This step confirms that the leader's role is justified by having received votes from at least 2/3 of the replicas.

2. **State Application**: Replicas apply the proposal block against their individual state machines, ensuring consistency and accuracy with their own data and expected outcomes.

3. **Header and Results Verification**: Each replica checks the header and the results of the proposal against what they themselves have produced. This ensures there's no discrepancy in the data and operations.

4. **Signature Voting**: If the proposal is deemed valid, the replica sends a signature to the leader, effectively casting their vote. This vote guarantees the validity of the proposal and confirms the approval of the replica in its execution.

Additionally, during this phase, it is critical to check the HighQC (High Quality Certificate) and determine if the safe node predicate is passed, allowing for unlocking if the criteria are met. This is integral in ensuring that the consensus process maintains robustness and integrity through reliable and validated proposals.

# Precommit Phase

In the PRECOMMIT phase of the NestBFT consensus algorithm, the leader gathers PROPOSE VOTES from over two-thirds of the replicas, each of which includes a signature from the sender. The purpose of this phase is to ensure that the proposed block has the endorsement of a super-majority, thereby enhancing the security and reliability of the consensus. Once the leader has collected the necessary votes, they send a PRECOMMIT message that attaches these signatures. This justifies that a substantial portion of the quorum acknowledges the validity of the proposed block.

During this phase, the leader works to compile evidence that the proposed block is widely accepted. This involves ensuring that over two-thirds of the quorum believe in the validity of the proposal. The PRECOMMIT phase plays a crucial role in the consensus process as it transitions into the next stages where further validation processes occur to solidify the agreement on the next block in the blockchain.

# Precommit Vote Phase

In the PRECOMMIT-VOTE phase of the NestBFT consensus algorithm, each replica is tasked with verifying the PRECOMMIT message. This involves ensuring that the proposal comes from the expected proposer and that it is the anticipated proposal. The replicas achieve this by validating the aggregate signature contained within the PRECOMMIT message. If everything checks out, the replica effectively locks into this proposal, meaning it is prepared to move forward with this block in the consensus process.

Once the proposal is verified, each participating replica sends a vote back to the proposer. This vote signals that the replica has verified and agrees with the proposal, expressing its support for the super-majority needed to advance to the next phase of the consensus process. The votes collected during this phase play a crucial role in helping the leader justify that a significant portion of the quorum—a +2/3 majority—believes the proposal is valid and ready to be committed to the blockchain.

# Commit Phase

During the COMMIT phase of the NestBFT consensus algorithm, the leader collects PRECOMMIT votes from more than two-thirds (i.e., +2/3) of the replicas. Each of these votes includes a signature from the sender, which the leader then uses to construct a COMMIT message. This message attaches the +2/3 signatures from the PRECOMMIT VOTE messages, which serves as justification that a super-majority of the quorum believes the proposal is valid.

Once the leader has prepared the COMMIT message, it is sent to all validators. Upon receiving the COMMIT message, each replica validates it by verifying the aggregate signature provided. If the validation is successful, each replica will then commit the block to finality. Following this, the BFT setup is reset in preparation for reaching consensus on the next block height. This phase solidifies the proposed block, ensuring that it becomes a permanent part of the blockchain.

# Commit Process Phase

The COMMIT PROCESS phase in the NestBFT consensus algorithm involves the validation of the COMMIT message. Each replica verifies the aggregate signature included in the COMMIT message to ensure that it is from the expected proposer and that the proposal is valid. Once the aggregate signature is verified, the proposal signifies that +2/3 of the quorum agrees that a super-majority believes the proposal is valid.

Upon successful verification, the block is gossiped throughout the network and committed to the local chain. This step finalizes the block within the chain at the current height. Following the commitment, the replicas then reset the BFT process for the next block height, enabling the continuation of the consensus process for subsequent blocks.

# Recovery Phases

## ROUND INTERRUPT

The ROUND INTERRUPT phase occurs in response to a failure in the BFT cycle, resulting in a premature exit from a round. This leads to the initiation of a new round and an extension of the sleep time between phases. This extended sleep time aims to alleviate any 'non-voter' issues that may have arisen. During this phase, each replica sends its current View message to all other replicas. The purpose of this action is to help alleviate round synchronization issues, ensuring all replicas are aware of each other's state.

Furthermore, the `RoundInterrupt` function sets the ROUND-INTERRUPT phase and sends a pacemaker message to all other replicas. This message includes a Quorum Certificate that contains the current View's header. The process ensures that the replicas remain in sync, and the timeout for ROUND INTERRUPT is configured based on the remaining milliseconds in the round.

## PACEMAKER

Following the ROUND INTERRUPT phase, the PACEMAKER phase takes place. Each replica calculates the highest round that a super-majority has observed and jumps to it. This approach assists in addressing 'round out of sync' issues, ensuring that all participants are aligned on the round they are currently processing. 

The PACEMAKER phase effectively resets the process and synchronizes the replicas. This ensures that the replicas are working within the same round observed by a super-majority, thus facilitating consensus. The process involves setting the highest round that at least two-thirds of the replicas have seen, allowing the network to adjust and realign, minimizing the risk of further synchronization issues.

# Phase Times and Total Block Time

The mechanism used to control block times in NestBFT involves defining the duration of each phase within the `config.json` file. The phases and their initial configurations are as follows:

```json
{
  "electionTimeoutMS": 2000,
  "electionVoteTimeoutMS": 2000,
  "proposeTimeoutMS": 3000,
  "proposeVoteTimeoutMS": 3000,
  "precommitTimeoutMS": 2000,
  "precommitVoteTimeoutMS": 2000,
  "commitTimeoutMS": 2000,
  "commitProcessMS": 3000
}
```

Each individual phase time can be adjusted according to the needs of the application. The total block time is effectively the sum of all phase durations. If you wish to extend the overall block time without altering the timing of individual phases, you can increase the `commitProcessMS`. This approach allows you to control the total block time while maintaining the structure of phase timings.

# Locking & Safe Node Predicates

During the precommit vote phase, replicas will lock onto a proposal that has been verified by the leader as having the majority vote behind it. 

Should a round interrupt occur, the consensus process will reset to the election phase, with replicas retaining the locked proposal. During the next propose phase, this locked proposal will be used as the new proposal to be gossiped to replicas.

In the propose vote phase, replicas will recognize that they still have a locked proposal and will run the safe node predicate check to determine if they can unlock. It is safe to unlock if:

- **SAFETY**: The block hash and result hash for the locked proposal and the received proposal are the same.
- **LIVENESS**: The round number in the received proposal is higher than that in the locked proposal.

These conditions ensure the integrity and progress of the consensus process.

