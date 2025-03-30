# The Block Proposer

An election is necessary to determine the next block proposer to ensure fair and decentralized decision-making. Without it, control could be manipulated by a single entity, compromising the blockchain's integrity and security.

NestBFT uses a unique election mechanism involving Practical VRF and linear stake-weighted thresholds. This approach enhances fairness by selecting potential leaders based on stake and randomness, protecting against attacks and ensuring decentralized block proposals.

# Election Phase

The election phase of the NestBFT consensus algorithm involves a sortition process combined with a Verifiable Random Function (VRF) signature. These mechanisms help determine the leader for a given round.

## Importance of Sortition Seed Data Parts

1. **Round**: The inclusion of "Round" in the sortition seed data ensures that the VRF signature is unique for every round. This uniqueness is crucial as it mandates the election of a new leader if the current round concludes without reaching consensus. By changing the leader candidate each round, the algorithm aims to prevent repeated failures caused by a faulty or malicious leader in consecutive rounds.

2. **LastProposerAddresses**: The use of "LastProposerAddresses" in generating the VRF signature helps ensure that previous leaders cannot manipulate the variables used in the current round's leader election process. This inclusion provides an additional layer of fairness and randomness in the selection process by incorporating historical leadership data, preventing bias or undue advantage to past leaders.

## Stake-Weighted Selection

In the NestBFT election phase, the higher the voting power (stake) of a validator, the greater their chance of being selected as a candidate for leadership. This stake-weighted selection ensures that validators who have invested more resources into the network have a proportionate influence, aligning with the principles of stake-based blockchain networks.

# Election Vote Phase

In the Election Vote Phase of the NestBFT consensus algorithm, replicas evaluate messages from potential leaders known as Candidates based on their VRF (Verifiable Random Function) outputs. The replica nodes review these outputs to determine the leader. If no Candidate messages are received, a fallback mechanism selects a leader randomly, weighed by the stake. 

After identifying the leader, replicas send a digitally signed election vote to the selected leader, attaching any Byzantine evidence or 'Locked' QC (quorum certificate) they have, alongside their VDF (Verifiable Delay Function) output. These votes are aggregable, meaning they can be collectively assessed to determine a consensus. This process helps the network agree on which proposal to proceed with while ensuring fair representation of voting power based on the stake of the participants.

I'm sorry, but I don't have specific information about the NestBFT consensus algorithm within the provided context. However, based on what I know, I can describe a typical propose phase in many BFT-style algorithms:

# Propose Phase

In the propose phase of a consensus algorithm like NestBFT, the leader's role is crucial:

1. **Block Proposal Creation**: The leader generates a new block proposal. If there's an existing proposal that has been previously locked, the leader may opt to use this rather than constructing a new one. This acts as a mechanism to ensure consistency and stability.

2. **Components of a Proposal**: The proposal typically includes:
   - The proposed block itself, which comprises transactions.
   - Results related to rewards and penalties, often referred to as reward and slash recipients. These determine who gets rewarded for their participation in consensus and who gets penalized for malicious or faulty behavior.

3. **Proposal Distribution**: Once the proposal is ready, the leader disseminates it to all participating validators in the network. This step is essential for moving the network towards agreement on the new block.

If you have any additional or specific questions about the NestBFT algorithm or its various phases, feel free to ask.

# Propose Vote Phase

The `PROPOSEVOTE` phase of the NestBFT consensus algorithm is not specifically detailed in the provided context. However, based on the typical flow of consensus algorithms like BFT, the `PROPOSEVOTE` phase likely involves the following steps:

1. **Proposal Review**: After the `PROPOSE` phase, validators (other than the leader) review the block proposal made by the leader. This proposal includes aggregated votes or signatures that justify the validity and acceptance of the proposed block.

2. **Validation**: Validators validate the proposal to ensure it is legitimate, checking the accompanying signatures for the required threshold (typically +2/3) of voting power.

3. **Vote Generation**: Upon validation, if a validator agrees with the proposal, they generate a vote message as a digital signature to express their agreement with the proposal.

4. **Vote Transmission**: This vote is then sent back to the leader, or possibly gossiped to other validators, contributing to the consensus process.

These steps typically ensure that the proposal reaches consensus and can proceed to the next phases like `PRECOMMIT` and `COMMIT`.

If more specific details about the `PROPOSEVOTE` phase are required, they could be found in technical documentation or source code related to the NestBFT consensus protocol outside the provided context.

# Precommit Phase

In the PRECOMMIT phase of the NestBFT consensus algorithm, the protocol transitions to the point where the leader node aims to justify consensus on a block proposal. During this phase, the leader aggregates votes from the replica validators and sends a PRECOMMIT message. The main actions that occur in this phase are:

1. **Logging the Current View**: The leader logs the current view information to maintain a record of the ongoing consensus process.

2. **Checking Proposer Role**: The leader checks if it is the current proposer. If not, it exits the process for this phase.

3. **Gathering Majority Vote**: The leader calls `GetMajorityVote()` to gather votes from replica validators. This step involves aggregating signatures that account for at least a two-thirds majority of the voting power.

4. **Handling Errors**: If there is an error in obtaining the majority vote, the leader logs the error and invokes `RoundInterrupt()` to handle this situation, likely attempting to recover or retry the consensus process.

5. **Sending PRECOMMIT Message**: Once a valid majority vote and aggregated signature are obtained, the leader sends a PRECOMMIT message to all replicas. This message includes the quorum certificate (QC) with headers reflecting the vote view, block hash, results hash, the proposer’s key, and the aggregated signature.

The PRECOMMIT phase is crucial as it sets the stage for the next phase, PRECOMMIT_VOTE, where replicas lock on the proposal after validating the justification, but no locking occurs during the PRECOMMIT phase itself.

# Precommit Vote Phase

The PRECOMMITVOTE phase is a crucial step in the consensus process of the NestBFT algorithm. In this phase, replicas (or Validator nodes that are not acting as the Leader) send their votes to the Leader. Each vote is a digital signature based on the current state of the blockchain, specifically the block and its associated data being proposed. The purpose of the PRECOMMITVOTE phase is to gather consensus from the replicas by achieving a +2/3 majority of voting power.

During the PRECOMMITVOTE phase, replicas review the Leader’s proposal from the previous phase and decide whether to support it. If enough replicas agree, by submitting their votes to the Leader, a quorum certificate can be formed that justifies moving the proposed block to the next stage of the consensus process. The votes are organized by `Payload Hash` to ensure consistency in what the replicas are voting on.

The successful aggregation of these votes by the Leader demonstrates that the proposed block has substantial support, which is necessary for the network to achieve consensus and guarantees that the block can proceed to the next step within the consensus cycle.

# Commit Phase

During the Commit Phase of the NestBFT consensus algorithm, the following steps are taken:

1. **Leader's Role**: 
   - The leader collects and aggregates votes from the replica validators.
   - It identifies whether a +2/3 majority of voting power has been achieved. This majority is necessary for moving forward with consensus.

2. **Validation of Majority Vote**: 
   - The leader checks if the aggregated votes are valid and constitute a +2/3 majority.

3. **Sending the Commit Message**: 
   - If the vote is valid, the leader sends a Commit message to all the validators. This message asserts that consensus has been achieved for the current block.

In essence, the Commit Phase is where the leader finalizes the consensus decision by confirming majority support and communicating this decision to the network of validators.

# Commit Process Phase

During the COMMITPROCESS phase of the NestBFT consensus algorithm, the replicas commit the block to the chain. This phase is crucial for achieving consensus, as it finalizes the block that has been proposed and agreed upon by the network participants. The replicas ensure that the block is valid and then proceed to append it to their local blockchain, thereby updating the state of the network and preparing for subsequent phases in the consensus process.

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
