# Documentation for `bft.go`

# Description

The `bft.go` file implements the core structures and logic for managing the consensus process in blockchain systems using a specific BFT algorithm. It handles the flow of information and actions across different phases of consensus to ensure all replicas reach agreement on new blocks in a decentralized network.

# Key Components

## The BFT type

The `bft` type encapsulates all state required for replicas to synchronously transition through each phase in unison.

- **View Management**: Tracks the current state of the consensus process, maintaining the current height, round, and phase. This, combined with the recovery phases, allows all replicas to remain synchronized during the consensus process and recover in the case of consensus failure.

- **Leader Election**: Uses Verifiable Random Functions (VRF) and sortition seed data to elect a leader and ensure a fair election process.

- **Validator Coordination**: The structure coordinates between different validators in the network, organizing their votes and proposals through fields like *Votes* and *Proposals* to achieve consensus.

- **Super-Majority Votes**: Defined as two-thirds of replica votes, super-majorities are used to ensure that all actions are justified with the required number of replicas in agreement.

- **Proposal Locking**: Allows the replica to "lock" onto a proposal to aid in recovery should the consensus process fail before the round is complete.

- **Quorum Certificates**: Quorum certificates are used to validate and communicate actions taken by nodes, ensuring that decisions are backed by a majority of voting power.

- **Block Management**: Handles new block proposals by allowing the elected leader to put forth proposals for the next block. Replicas then evaluate these proposals, and upon receiving sufficient votes of confidence, the block is committed to the chain.

- **Byzantine Evidence**: Maintains evidence of Byzantine behavior to track and prevent issues like double signing.

## Consensus Phases & Rounds

The consensus process is broken down into 8 core phases and 2 recovery phases. Each phase represents the smallest unit of the consensus process. Each round consists of multiple phases, and each height may consist of multiple rounds. These phases are executed sequentially to achieve consensus on the next block.

Below is a list of each core phase and their primary purpose:

1. **Election**: Replicas gossip candidacy.
2. **ElectionVote**: Replicas vote for a leader selected from the pool of gossiped candidates.
3. **Propose**: The elected leader puts forth a proposed block for consideration.
4. **ProposeVote**: Replicas validate the proposed block and send a vote to the leader.
5. **Precommit**: The leader reviews block votes received from replicas.
6. **PrecommitVote**: Replicas validate majority approval and send votes to the leader.
7. **Commit**: The leader verifies majority vote results.
8. **CommitProcess**: Replicas validate the majority signature and proceed to commit the block.

The two recovery phases are used when an error in the consensus process causes a premature exit to the round.

1. **RoundInterrupt**: During this phase, each replica sends a View to all other replicas to enable synchronization in the Pacemaker phase.

2. **Pacemaker**: This phase synchronizes each replica to the highest round a super-majority has seen and restarts the consensus process beginning with the Election phase.
The context appears to be readable and of good quality, so here's an introductory paragraph for the core logic of the BFT consensus.

# NestBFT

Welcome to the world of Byzantine Fault Tolerant (BFT) consensus, specifically tailored for blockchain developers. In this algorithm, the focus is on achieving consensus through a series of structured phases. Here’s a brief overview of the process, using terms like "voting power" and "majority vote" that you'll encounter frequently:

- **ELECTION:** This is where it begins. Each replica runs a Verifiable Random Function (VRF) to determine if it's selected as a candidate, sending its VRF output to other replicas. The successors depend on the ELECTION votes from this phase.

- **ELECTION-VOTE:** Replicas cast ELECTION votes for the leader, determined by the lowest VRF value. If no candidates emerge, a fallback to stake-weighted pseudorandom selection occurs. The majority of these votes set the stage for the next phase, the proposal formulation.

- **PROPOSE:** The leader, chosen with +2/3 voting power, compiles these votes, including their locks and evidence, to craft a new block proposal. The next phase hinges on the acceptance of this proposal. 

Each phase interacts symbiotically with the preceding and succeeding phases to fortify the consensus process. This algorithm leverages the power dynamics of voting to achieve a robust and decentralized consensus. 
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

