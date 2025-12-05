# event.go - Event Tracking in the Canopy Blockchain

This file handles event tracking for Canopy's state machine. Think of it as the blockchain's activity log - it records what happens when validators get rewarded, penalized, or when people trade tokens.

## What It Does

The event system keeps track of:
- Validator stuff (rewards, slashing, pausing)
- Token swaps on the DEX
- Order book trades
- Everything else that needs to be logged for monitoring

## Event Categories

We have three main types of events:

1. **Validator Events** - What happens to validators (good and bad)
2. **DEX Events** - Automated trading activity
3. **Order Book Events** - Traditional order-based trading

### How Events Are Structured

Every event has the same basic info:
- What type of event it is
- Who was involved (their address)
- When it happened (block height)
- Which chain it was on
- The specific details of what happened
- A reference ID to link related events

## The Different Events

### Validator Events

#### Getting Paid
`EventReward()` logs when validators earn rewards for doing their job. It tracks how much they got paid and which chain the reward came from.

#### Getting Punished
`EventSlash()` records when validators get penalized for misbehaving or breaking protocol rules. Nobody likes getting slashed, but it keeps the network honest.

#### Status Changes
We automatically track when validators change status:
- `EventAutoPause()` - When a validator gets benched for poor performance
- `EventAutoBeginUnstaking()` - When the system starts kicking out an underperforming validator
- `EventFinishUnstaking()` - When a validator finally leaves the active set

### DEX Events

#### Token Swaps
`EventDexSwap()` logs every token swap that happens through our automated market maker. We track:
- How much of what token was sold/bought
- Whether it was coming in or going out
- If the swap actually worked
- Who did it and on which chain

#### Liquidity Pool Activity
People can add or remove liquidity from our pools:
- `EventDexLiquidityDeposit()` - Someone added tokens to a pool
- `EventDexLiquidityWithdraw()` - Someone pulled their tokens out

### Order Book Trading

#### Traditional Trading
`EventOrderBookSwap()` handles the old-school way of trading with limit orders. We track:
- How much was traded
- Who the buyer and seller were
- The order details
- All the addresses involved

## How It Works Under the Hood

### Creating Events
The `addEvent()` function is where all events get created. Here's what happens:

1. We wrap the event data in a protocol buffer message
2. Create a standard event structure with all the metadata
3. Give it a reference ID so we can link related events later
4. Stamp it with the current block height
5. Add the chain ID if it's a multi-chain event
6. Store it in our event database

### Finding Events Later
Events are stored so you can easily find them by:
- Block height (when did this happen?)
- Address (what did this person do?)
- Event type (show me all the slashing events)
- Reference ID (find all related events)

### Multi-Chain Support
Since Canopy works across multiple chains, events can include which chain they came from. This lets us track cross-chain activities and keep everything organized.

### Working with the State Machine
Events are created at the same time as state changes, so they're always in sync. When something happens on the blockchain, the state changes AND we log an event - it's atomic, so you can't have one without the other.

## Why You Can Trust These Events

- Events happen at the exact same time as state changes - no inconsistencies
- Once an event is recorded, it can't be changed - permanent audit trail
- All data is structured consistently using protocol buffers
- Event block heights are always validated
- Reference IDs keep related events properly linked

## What People Use This For

- **Monitoring** - Keep an eye on validator performance and network health
- **Analytics** - Understand trading patterns and how liquidity flows
- **Auditing** - Have a permanent record of everything that happened
- **Integration** - Feed data to external dashboards and tools
- **Compliance** - Maintain records for regulatory purposes