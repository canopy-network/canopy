# Documentation for `bft.go`

# Description

## `bft` Type

The `bft` type is a comprehensive structure that encapsulates the state and core logic of the NestBFT consensus algorithm. It is responsible for driving the consensus process forward through the core consensus phases, ensuring trusted and timely operation of the Canopy blockchain.

- **Consensus and View Management**:
  - Manages the current consensus phase for each replica. Tracks the current view from the perspective of the replica.
  - Through p2p communication it coordinates parallel phase progression between other participating replicas.

- **Leader Election**:
  - Employs Verifiable Random Functions (VRF) and a Sortition proces to ensure fair and unbiased leader selection.
  - The leader is elected based on a combination of randomness and voting power.


- **Block and Result Management**:
  - Manages the current blockchain block and its related data.
  - Guarantees proper proposal crafting and verification per consensus rules whilst documenting consensus results and slashing conditions.

- **Vote and Proposal Management**:
  - Records the votes received from the validators and the proposals originating from the leader.

- **Security Assurance**:
  - Employs security mitigations for griding and long chain attacks.
  - Enforces slashing and rewarding mechanisms.
  - Validates all proposals and votes.

## Consensus Phase Overview

The consensus process is broken down into 8 core phases and 2 recovery phases.
Each phase represents the smallest unit of the concensus process. Each round
consists of multiple phases, and each height may consist of multiple rounds.
These phases are executed sequentially and upon successful completion achieve
consensus on the next block.

At the beginning of each new block height the round is reset to 0 and restarts
consensus at the Election phase. If the a round of consensus does not succeed,
recovery phases are initiated in order to continue consensus.

### Phase Summaries

- **Election:**
  - Each replica runs a Verifiable Random Function (VRF); if selected as a candidate, the replica sends its VRF output to the other replicas.

- **ElectionVote:**
  - Each replica sends ELECTION votes (signature) for the leader based on the lowest VRF value. If no candidates exist, the process falls back to a stake-weighted-pseudorandom selection.

- **Propose:**
  - The leader collects ELECTION.VOTES from +2/3 of the replicas, each including the lock, evidence, and signature from the sender. If a valid lock exists for the current height, the leader uses that block as the proposal block. If no valid lock is found, the leader creates a new block to extend the blockchain.

- **ProposeVote:**
  - Each replica validates the PROPOSE message by verifying the aggregate signature, applying the proposal block against their state machine, and checking the header and results against what they produced. If valid, the replica sends a vote (signature) to the leader.

- **Precommit:**
  - The leader collects PROPOSE VOTES from +2/3 of the replicas, each including a signature from the sender. The leader sends a PRECOMMIT message attaching +2/3 signatures from the PROPOSE VOTE messages.

- **PrecommitVote:**
  - Each replica validates the PRECOMMIT message by verifying the aggregate signature. If valid, the replica sends a vote to the leader.

- **Commit:**
  - The leader collects PRECOMMIT VOTES from +2/3 of the replicas, each including a signature from the sender. The leader sends a COMMIT message attaching +2/3 signatures from the PRECOMMIT VOTE messages.

- **CommitProcess:**
  - Each replica validates the COMMIT message by verifying the aggregate signature. If valid, the replica commits the block to finality and resets the BFT for the next height.

The two recovery phases address situations where errors cause a premature exit from a round:

- **Round Interrupt**:
  - In this phase concensus is halted and reset. Each replica shares its current View with all other replicas. This allows replicas to synchronize in the Pacemaker phase.

- **Pacemaker**:
  - This phase synchronizes each replica to the highest round that a super-majority has observed, allowing the consensus process to restart with the Election phase.

# Key Concepts


#### View

The `View` field within the `BFT` struct is a component for tracking the current
period of the consensus process, defined by `Height`, `Round`, and `Phase`.

The `View` aids in synchronizing validators by providing a consistent reference
point for the current state of the blockchain. It ensures that all validators
are aligned regarding the block height, round, and phase they are operating in.
This alignment allows validators to correctly interpret proposals, cast votes,
and validate the results of the consensus process.

The `View` is included with every message sent between nodes and is required
during the recovery phases, where it is used to synchronize all replicas to
highest round seen by the super-majority of nodes.

#### Super-Majority

A super-majority refers to a threshold of agreement among the participating
replicas that is greater than a simple majority. Specifically, it requires more
than two-thirds (+2/3) of the voting power or votes from the replicas to agree
on a proposal or vote to be considered in consensus.

The super-majority threshold is applied in various phases of the consensus
process, such as during the ELECTION, PROPOSE, PRECOMMIT, and COMMIT phases,
where the leader collects votes from +2/3 of the replicas to justify consensus
on an election or proposal. This mechanism ensures that the system can function
effectively despite potential faulty or Byzantine nodes.

#### Proposal Locks

Once a proposal is supported by a quorum certificate, replicas will `lock` onto
the proposal. If consensus is not achieved in a specific round, the locked
proposal is preserved for future rounds. This ensures that even if the current
round does not reach consensus, the proposal is not discarded. Instead, it
remains a valid proposal for subsequent rounds. The leader in the next round can
propose this same proposal again, as it already has quorum support.

#### Quorum Certificates

Replicas use the Quorum Certificate (QC) to share important information with
other replicas. This information may include the current view of a replica, a
vote, or a super-majority consensus that validates an action.

Quorum Certificates (QCs) demonstrate that a specified majority of replicas (at
least two-thirds) have verified and agreed on a particular aspect of the
consensus process. These certificates confirm that consensus has been reached
without requiring constant direct communication among all replicas.

# Election Phase

The election phase utilizes a sortition process in conjunction with a Verifiable
Random Function (VRF) to ensure the selection of leaders is fair, uniform, and
unpredictable. This process relies on unique and non-manipulatable inputs of
seed data, which are crucial in resisting manipulation and providing a robust
defense against potential biases. The use of VRF ensures that the selection
process is both random and publically verifiable.

Validators create a digital signature on the sortition seed data using their
private key. The integer value derived from this signature, combined with their
voting power, determines their candidacy. The stake of a validator influences
this process, as a higher stake increases the probability of becoming a
candidate. This stake-weighted method ensures that validators with larger
contributions to the network have a greater likelihood of selection, aligning
the incentives of network participants with the security and integrity of the
blockchain.

### Sortition Seed Data

The integrity of the sortition seed data is paramount, as any manipulation could
lead to predictable and biased leader selection. By ensuring that the seed data
remains secure and non-manipulatable, NestBFT fosters an environment where
leadership is assigned fairly, maintaining unpredictability and fairness in the
network.

- **Round Field Inclusion**: The incorporation of the round field into the
  sortition data reduces the likelihood of the same leader being chosen in
  consecutive rounds. This mechanism mitigates the risk posed by a malicious or
  faulty leader.

- **Last Proposer Addresses Field**: NestBFT distinguishes itself from other
  protocols by utilizing the LastProposerAddresses field within its sortition
  seed data. This approach avoids reliance on manipulable inputs, such as the
  last block hash, which are susceptible to bias and grinding attacks. By
  eliminating these vulnerabilities, NestBFT ensures a fairer and less
  predictable leader selection process.

# Election Vote Phase

In the ELECTION VOTE phase, each replica reviews messages from candidates, which
include a Verifiable Random Function (VRF) output. This cryptographic function
generates a publicly verifiable random output. The replicas select the candidate
with the lowest VRF output as the leader, promoting a fair selection process.

If no candidate messages are received, the process defaults to a
stake-weighted-pseudorandom selection to ensure a leader is chosen.

Once a leader is determined, each replica sends a signed ELECTION vote to the
selected leader. In the following phase, the leader aggregates these votes. If
they receive votes from more than two-thirds of the replicas, they can justify
proposing a new block. This vote aggregation indicates consensus among the
replicas.

# Propose Phase

During the PROPOSE phase of the NestBFT consensus algorithm, the leader is responsible for producing a new block proposal. Here are the steps involved in this phase:

1. **Collecting Election Votes**: The leader gathers election votes from more than two-thirds (+2/3) of the replicas. This serves as proof of the leader's qualification to propose a block.

2. **Proposal Block Selection**:
   - If a valid lock exists for the current height, the leader uses the locked block as the proposal block. This ensures that the proposal is based on a previously agreed-upon state, enhancing the stability and reliability of the consensus process.
   - If no valid lock is found, the leader creates a new block to extend the blockchain. This new block is constructed with the maximum number of transactions available in the mempool, ensuring efficient use of network resources.

3. **Creating the Proposal**: The proposal consists of the new proposed block and the associated results, which include the reward and slash recipients. A block contains the transactions to be processed, and the proposal is backed by the signatures from the election votes, confirming the leader's legitimacy.

4. **Distribution of Proposal**: The leader sends the newly created proposal (block, results, evidence) to all validators. The proposal is justified by attaching the signatures from more than two-thirds (+2/3) of the election votes, confirming the leader's legitimacy.

# Propose Vote Phase

In the NestBFT consensus algorithm, the PROPOSE VOTE phase involves several
critical steps where the replicas (nodes) validate the proposal put forward by
the leader. Each replica follows a systematic approach to ensure the proposal's
validity:

1. **Proposal Validation**: Each replica receives the PROPOSE message from the
   leader and must validate it by checking the aggregate signature. This step
   confirms that the leader's role is justified by having received votes from at
   least 2/3 of the replicas.

2. **State Application**: Replicas apply the proposal block against their
   individual state machines, ensuring consistency and accuracy with their own
   data and expected outcomes.

3. **Header and Results Verification**: Each replica checks the header and the
   results of the proposal against what they themselves have produced. This
   ensures there's no discrepancy in the data and operations.

4. **Signature Voting**: If the proposal is deemed valid, the replica sends a
   signature to the leader, effectively casting their vote. This vote guarantees
   the validity of the proposal and confirms the approval of the replica in its
   execution.

Additionally, during this phase, it is critical to check the HighQC (High
Quality Certificate) and determine if the safe node predicate is passed,
allowing for unlocking if the criteria are met. This is integral in ensuring
that the consensus process maintains robustness and integrity through reliable
and validated proposals.

# Precommit Phase

In the PRECOMMIT phase of the NestBFT consensus algorithm, the leader gathers
PROPOSE VOTES from over two-thirds of the replicas, each of which includes a
signature from the sender. The purpose of this phase is to ensure that the
proposed block has the endorsement of a super-majority, thereby enhancing the
security and reliability of the consensus. Once the leader has collected the
necessary votes, they send a PRECOMMIT message that attaches these signatures.
This justifies that a substantial portion of the quorum acknowledges the
validity of the proposed block.

During this phase, the leader works to compile evidence that the proposed block
is widely accepted. This involves ensuring that over two-thirds of the quorum
believe in the validity of the proposal. The PRECOMMIT phase plays a crucial
role in the consensus process as it transitions into the next stages where
further validation processes occur to solidify the agreement on the next block
in the blockchain.

# Precommit Vote Phase

In the PRECOMMIT-VOTE phase of the NestBFT consensus algorithm, each replica is
tasked with verifying the PRECOMMIT message. This involves ensuring that the
proposal comes from the expected proposer and that it is the anticipated
proposal. The replicas achieve this by validating the aggregate signature
contained within the PRECOMMIT message. If everything checks out, the replica
effectively locks into this proposal, meaning it is prepared to move forward
with this block in the consensus process.

Once the proposal is verified, each participating replica sends a vote back to
the proposer. This vote signals that the replica has verified and agrees with
the proposal, expressing its support for the super-majority needed to advance to
the next phase of the consensus process. The votes collected during this phase
play a crucial role in helping the leader justify that a significant portion of
the quorum—a +2/3 majority—believes the proposal is valid and ready to be
committed to the blockchain.

# Commit Phase

During the COMMIT phase of the NestBFT consensus algorithm, the leader collects
PRECOMMIT votes from more than two-thirds (i.e., +2/3) of the replicas. Each of
these votes includes a signature from the sender, which the leader then uses to
construct a COMMIT message. This message attaches the +2/3 signatures from the
PRECOMMIT VOTE messages, which serves as justification that a super-majority of
the quorum believes the proposal is valid.

Once the leader has prepared the COMMIT message, it is sent to all validators.
Upon receiving the COMMIT message, each replica validates it by verifying the
aggregate signature provided. If the validation is successful, each replica will
then commit the block to finality. Following this, the BFT setup is reset in
preparation for reaching consensus on the next block height. This phase
solidifies the proposed block, ensuring that it becomes a permanent part of the
blockchain.

# Commit Process Phase

The COMMIT PROCESS phase in the NestBFT consensus algorithm involves the
validation of the COMMIT message. Each replica verifies the aggregate signature
included in the COMMIT message to ensure that it is from the expected proposer
and that the proposal is valid. Once the aggregate signature is verified, the
proposal signifies that +2/3 of the quorum agrees that a super-majority believes
the proposal is valid.

Upon successful verification, the block is gossiped throughout the network and
committed to the local chain. This step finalizes the block within the chain at
the current height. Following the commitment, the replicas then reset the BFT
process for the next block height, enabling the continuation of the consensus
process for subsequent blocks.

# Recovery Phases

## ROUND INTERRUPT

The ROUND INTERRUPT phase occurs in response to a failure in the BFT cycle,
resulting in a premature exit from a round. This leads to the initiation of a
new round and an extension of the sleep time between phases. This extended sleep
time aims to alleviate any 'non-voter' issues that may have arisen. During this
phase, each replica sends its current View message to all other replicas. The
purpose of this action is to help alleviate round synchronization issues,
ensuring all replicas are aware of each other's state.

Furthermore, the `RoundInterrupt` function sets the ROUND-INTERRUPT phase and
sends a pacemaker message to all other replicas. This message includes a Quorum
Certificate that contains the current View's header. The process ensures that
the replicas remain in sync, and the timeout for ROUND INTERRUPT is configured
based on the remaining milliseconds in the round.

## PACEMAKER

Following the ROUND INTERRUPT phase, the PACEMAKER phase takes place. Each
replica calculates the highest round that a super-majority has observed and
jumps to it. This approach assists in addressing 'round out of sync' issues,
ensuring that all participants are aligned on the round they are currently
processing.

The PACEMAKER phase effectively resets the process and synchronizes the
replicas. This ensures that the replicas are working within the same round
observed by a super-majority, thus facilitating consensus. The process involves
setting the highest round that at least two-thirds of the replicas have seen,
allowing the network to adjust and realign, minimizing the risk of further
synchronization issues.

# Phase Times and Total Block Time

The mechanism used to control block times in NestBFT involves defining the
duration of each phase within the `config.json` file. The phases and their
initial configurations are as follows:

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

Each individual phase time can be adjusted according to the needs of the
application. The total block time is effectively the sum of all phase durations.
If you wish to extend the overall block time without altering the timing of
individual phases, you can increase the `commitProcessMS`. This approach allows
you to control the total block time while maintaining the structure of phase
timings.

# Locking & Safe Node Predicates

During the precommit vote phase, replicas will lock onto a proposal that has
been verified by the leader as having the majority vote behind it.

Should a round interrupt occur, the consensus process will reset to the election
phase, with replicas retaining the locked proposal. During the next propose
phase, this locked proposal will be used as the new proposal to be gossiped to
replicas.

In the propose vote phase, replicas will recognize that they still have a locked
proposal and will run the safe node predicate check to determine if they can
unlock. It is safe to unlock if:

- **SAFETY**: The block hash and result hash for the locked proposal and the
  received proposal are the same.
- **LIVENESS**: The round number in the received proposal is higher than that in
  the locked proposal.

These conditions ensure the integrity and progress of the consensus process.

# Documentation for `bft.go`

# Description

The bft.go file implements the core structures and logic for managing the
consensus process in blockchain systems using the NestBFT algorithm. It handles
the flow of information and actions across different phases of consensus to
ensure all replicas reach agreement on new blocks in a decentralized network.

# Key Concepts

### Consensus Phases & Rounds

The consensus process is broken down into 8 core phases and 2 recovery phases.
Each phase represents the smallest unit of the concensus process. Each round
consists of multiple phases, and each height may consist of multiple rounds.
These phases are executed sequentially and upon successful completion achieve
consensus on the next block.

Below is a list of each core phase and their primary purpose:

1. **Election**: Eligible replicas gossip their candidacy for the election
2. **ElectionVote**: Replicas vote for a leader from the pool of gossiped
   candidates
3. **Propose**: The elected leader produces a block proposal, relaying it to
   replicas
4. **ProposeVote**: Replicas validate proposed block and send validation vote to
   leader
5. **Precommit**: Leader reviews validation votes for super-majority consensus
6. **PrecommitVote**: Replicas validate consensus and send validation vote to
   leader
7. **Commit**: Leader aggregates votes, verifies majority of replicas approved
8. **CommitProcess**: Replicas verify proposer and proposal and commit block

The two recovery phases are used when an error in the consensus process causes a
premature exit to the round.

1. **RoundInterrupt**: During this phase each replica sends their current View
   to all other replicas to enable synchronization in the Pacemaker phase.

2. **Pacemaker**: This phase synchronizes each replica to the highest round a
   super-majority has seen and restarts the consensus process beginning with the
   Election phase.

### View

A view tracks the current state of the consensus from the perspective of a
replica, maintaining the current height, round, and phase.

In the case of consensus error, each replica sends its View to all other
replicas during the pacemaker phase. By sharing this information, replicas can
synchronize and jump to the highest round observed by the majority, ensuring all
replicas are on the same round and can proceed with another attempt at
consensus.

### Super-Majority Votes

Defined as two-thirds of replica votes, super-majorities are used to ensure that
all actions are justified with the required number of replicas in agreement.

### Proposal Locking

Once a super-majority of replicas validate a proposal, each replica "locks" the
proposal.

If consensus cannot be reached in a particular round, the locked proposal will
be retained for subsequent rounds. The leader in a new round can propose this
locked block because it has already received a quorum certificate, indicating
that it was previously agreed upon by the network.

### Quorum Certificates

Replicas utilize the Quorum Certificate (QC) to convey critical information to
other replicas. This information can include the current view of a replica, a
vote, or a super-majority consensus that serves to validate an action.

Quality Certificates (QCs) are essential in demonstrating that a specified
majority of replicas (at least two-thirds) have verified and reached agreement
on a particular aspect of the consensus process. By doing so, QCs enable
replicas to interact and validate actions with assurance. These certificates
play a critical role by confirming that consensus has been reached without
necessitating constant direct communication among all replicas.

# Election Phase

The election phase is determines the next block proposer, or leader, for the
next round of consensus, This phase employs a unique mechanism involving a
Practical Verifiable Random Function (VRF) and linear stake-weighted thresholds.
This approach enhances fairness by selecting potential leaders based on both
stake and randomness.

The election process utilizes a sortition method combined with VRF to ensure a
fair, uniform, and unpredictable selection of candidates. Validators generate a
digital signature on the sortition seed data, with their stake (voting power)
influencing their likelihood of becoming a candidate.

During this phase any replicas that have determined they are candidates will
send their candidacy to all other peers.

### Sortition Seed Data

Ensuring the integrity of the sortition seed data is crucial, as any tampering
could result in predictable or biased leader election. It is essential to
maintain the security and immutability of the seed data to uphold a fair and
unbiased selection process within the consensus mechanism.

- **Round Field Inclusion**: The incorporation of the round field into the
  sortition data reduces the likelihood of the same leader being chosen in
  consecutive rounds, promoting leader rotation. This mechanism benefits the
  network by mitigating the risk posed by malicious or faulty leaders

- **Last Proposer Addresses Field**: NestBFT differentiates itself from other
  protocols by incorporating the LastProposerAddresses field within its
  sortition seed data. This strategy mitigates the risk associated with
  manipulable inputs like the last block hash, which are vulnerable to bias and
  grinding attacks. By addressing these potential weaknesses, NestBFT delivers a
  leader selection process that is more equitable and resistant to prediction
  and manipulation.

# Election Vote Phase

In the Election Vote phase, each node evaluates the candidate messages it
received during the prior phase and selects the candidate with the smallest VRF
(Verifiable Random Function) output as the leader.

If there are no eligible candidates, the system defaults to a stake-weighted
pseudorandom selection, ensuring that a leader is always appointed.

Following this selection, each replica sends their votes to the chosen leader.

# Propose Phase

During the propose phase, the designated leader is tasked with crafting a new
block proposal. The steps involved in this phase are as follows:

1. **Vote Collection**: Each replica accumulates votes it receives to ascertain
whether it has been designated as the leader for the current round.

2. **Proposal Block Selection**:
   - If a valid lock for the current block height exists that satisfies the
     conditions of the safe block predicate, the leader utilizes this locked
     block as the proposal block.
   - In the absence of a valid lock, the leader generates a new block proposal.

3. **Creating the Proposal**: The proposal consists of the new proposed block
   and the associated results, which include the reward and slash recipients.

4. **Distribution of Proposal**: The leader sends the newly created proposal
   (block, results, evidence) to all validators. The proposal is justified by
   attaching the signatures from more than two-thirds (+2/3) of the replicas as
   justification for this leader proposing a block.

# Propose Vote Phase

In the propose vote phase the replicas validate the proposal put forward by the
leader. Each replica follows a systematic approach to ensure the proposal's
validity:

1. **Proposal Validation**: Each replica receives the propose message from the
   leader and must validate it by checking the aggregate signature. This step
   confirms that the leader's role is justified by having received votes from at
   least 2/3 of the replicas.

2. **Unlocking Locked Proposal**: Before applying the new proposal, each replica
   checks if there is a previously locked proposal. If a lock exists, the
   replica verifies whether the new proposal supersedes the locked one based on
   the consensus rules. If the new proposal is valid and has a higher priority,
   the lock is released.

3. **State Application**: Replicas apply the proposal block against their
   individual state machines, ensuring consistency and accuracy with their own
   data and expected outcomes.


4. **Header and Results Verification**: Each replica checks the header and the
   results of the proposal against what they themselves have produced. This
   ensures there's no discrepancy in the data and operations.

5. **Signature Voting**: If the proposal is deemed valid, the replica sends a
   signed propose vote to the leader, casting their vote signifying they have
   verified this proposal.

# Precommit Phase

In the precommit phase the designated leader is responsible for aggregating
propose votes from more than two-thirds of the replica nodes. Each of these
votes encapsulates a cryptographic signature from the respective sender,
ensuring authenticity and integrity. The primary objective of this phase is to
attain super-majority approval for the proposed block.

Once the leader amasses the requisite votes, they send out a PRECOMMIT message,
which includes these signatures as proof of consensus. This step signifies that
a super-majority endorsesVthe validity of the proposed block.

# Precommit Vote Phase

In the precommit vote phase each replica is tasked with verifying the received
precommit message. This involves ensuring that the proposal comes from the
expected proposer and that it is the anticipated proposal. The replicas achieve
this by validating the aggregate signature contained within the precommit
message.

Once the proposal is verified, each participating replica sends a vote back to
the proposer. This vote signals that the replica has verified and agrees with
the proposal, expressing its support for the super-majority needed to advance to
the next phase of the consensus process. The votes collected during this phase
allow the leader to justify that this proposal has received super-majority
support and is ready to be committed to the blockchain.

# Commit Phase

During the commit phase of the NestBFT consensus algorithm, the leader collects
precommit votes from a super-majority of replicas. Each of these votes includes
a signature from the sender, which the leader then uses to construct a COMMIT
message. This message attaches the +2/3 signatures from the PRECOMMIT VOTE
messages, which serves as justification that a super-majority of the quorum
believes the proposal is valid.

Once the leader has prepared the COMMIT message, containing the aggregate
signatures, it is sent to all validators in prepation for the Commit Process
phase.

# Commit Process Phase

The COMMIT PROCESS phase in the NestBFT consensus algorithm involves the
validation of the COMMIT message. Each replica verifies the aggregate signature
included in the COMMIT message to ensure that it is from the expected proposer
and that the proposal is valid. Once the aggregate signature is verified, the
proposal signifies that +2/3 of the quorum agrees that a super-majority believes
the proposal is valid.

Upon successful verification, the block is gossiped throughout the network and
committed to the local store. This step finalizes the block within the chain at
the current height. Following the commitment, the replicas then reset the BFT
process for the next block height.

# Recovery Phases

## ROUND INTERRUPT

The ROUND INTERRUPT phase occurs in response to a failure in the consensus
cycle, resulting in a premature exit from a round. This leads to the initiation
of a new round and an extension of the sleep time between phases. This extended
sleep time aims to alleviate any 'non-voter' issues that may have arisen.

During this phase, each replica sends its current View message to all other
replicas. The purpose of this action is to allow the next phase, Pacemaker, to
sync all replicas on the highest round a super-majority has observed.

## PACEMAKER

The PACEMAKER phase effectively resets the process and synchronizes the
replicas. This ensures that the replicas are working within the same round
observed by a super-majority, thus facilitating consensus.

The process involves setting the highest round that at least two-thirds of the
replicas have seen, allowing the network to adjust and realign, minimizing the
risk of further synchronization issues.

# Phase Times and Total Block Time

The mechanism used to control block times in NestBFT involves defining the
duration of each phase within the `config.json` file. The phases and their
initial configurations are as follows:

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

Each individual phase time can be adjusted according to the needs of the
application. The total block time is effectively the sum of all phase durations.
If you wish to extend the overall block time without altering the timing of
individual phases, you can increase the `commitProcessMS`. This approach allows
you to control the total block time while maintaining the structure of phase
timings.

# Locking & Safe Node Predicates

During the precommit vote phase, replicas lock onto a proposal that the leader
has verified as having the majority vote. Locking is crucial as it prevents
replicas from committing to conflicting proposals.

If a round interrupt occurs the consensus process resets to the election phase.
However, replicas retain their locked proposal. In the subsequent propose phase,
this locked proposal is used as the proposal for the round.

In the propose vote phase, replicas recognize their locked proposal and perform
a safe node predicate check to determine if they can unlock. Unlocking is safe
under the following conditions:

- **SAFETY**: The block hash and result hash of the locked proposal match those
  of the received proposal. This ensures that the proposal is consistent with
  what was previously agreed upon.
- **LIVENESS**: The round number in the received proposal is higher than that in
  the locked proposal. This condition ensures that the system can progress and
  is not stuck on an outdated proposal.


# Chart
```mermaid
block-beta
    columns 4

    E["Election"]
    space
    space
    EV["ElectionVote"]
    space:4
    P["Propose"]
    space
    space
    PV["ProposeVote"]
    space:4
    PC["Precommit"]
    space
    space
    PCV["PrecommitVote"]
    space:4
    C["Commit"]
    space
    space
    CP["CommitProcess"]

    E--"Replicas Send Candidacy"-->EV
    EV--"Replicas Choose Leader"-->P
    P--"Leader Proposes Block"-->PV
    PV--"Replices Verify Proposal"-->PC
    PC--"Verified Majority"-->PCV
    PCV--"Replicas Validate Proposal"-->C
    C--"Majority Confirmed"-->CP
```
