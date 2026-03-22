# Canopy Explorer - API Endpoints Implementation Guide

## Overview

This document outlines all the API endpoints that need to be implemented to complete the Canopy Explorer functionality. The explorer has three main views: **Analytics View**, **Transaction View**, and **Validator View**.

## Current Status

### ✅ **Already Implemented Endpoints**

The following endpoints are already available in the RPC server (`cmd/rpc/routes.go`):

#### Core Query Endpoints
- `GET /v1/` - Version information
- `POST /v1/query/height` - Get current block height
- `POST /v1/query/account` - Get account details
- `POST /v1/query/accounts` - Get accounts list
- `POST /v1/query/validator` - Get validator details
- `POST /v1/query/validators` - Get validators list
- `POST /v1/query/block-by-height` - Get block by height
- `POST /v1/query/block-by-hash` - Get block by hash
- `POST /v1/query/blocks` - Get blocks list
- `POST /v1/query/tx-by-hash` - Get transaction by hash
- `POST /v1/query/txs-by-height` - Get transactions by block height
- `POST /v1/query/txs-by-sender` - Get transactions by sender
- `POST /v1/query/txs-by-rec` - Get transactions by recipient
- `POST /v1/query/pending` - Get pending transactions
- `POST /v1/query/params` - Get network parameters
- `POST /v1/query/supply` - Get supply information
- `POST /v1/query/pool` - Get pool information
- `POST /v1/query/committee` - Get committee information
- `POST /v1/query/orders` - Get orders information

#### Admin Endpoints
- `GET /v1/admin/config` - Get server configuration
- `GET /v1/admin/peer-info` - Get peer information
- `GET /v1/admin/consensus-info` - Get consensus information

---

## 🚀 **Required Endpoints for Complete Implementation**

### 1. **Analytics View Endpoints**

#### 1.1 Network Health & Performance
```http
POST /v1/query/network-uptime
```
**Purpose**: Get network uptime percentage and health metrics
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "7d" // 1d, 7d, 30d, 90d
}
```
**Response**:
```json
{
  "uptime": 99.98,
  "downtime": 0.02,
  "lastOutage": "2024-01-15T10:30:00Z",
  "averageBlockTime": 6.2,
  "networkVersion": "v1.2.4"
}
```

#### 1.2 Historical Fee Data
```http
POST /v1/query/fee-trends
```
**Purpose**: Get historical transaction fee trends
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "7d",
  "granularity": "hour" // hour, day, week
}
```
**Response**:
```json
{
  "trends": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "averageFee": 0.0023,
      "medianFee": 0.0021,
      "minFee": 0.0015,
      "maxFee": 0.0050,
      "transactionCount": 1250
    }
  ],
  "summary": {
    "average7d": 0.0023,
    "change24h": 0.05,
    "trend": "increasing"
  }
}
```

#### 1.3 Staking Rewards History
```http
POST /v1/query/staking-rewards
```
**Purpose**: Get historical staking rewards and trends
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "30d",
  "validatorAddress": "optional"
}
```
**Response**:
```json
{
  "rewards": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "totalRewards": 1250.5,
      "averageAPY": 8.5,
      "activeValidators": 128,
      "totalStaked": 45513085780613
    }
  ],
  "summary": {
    "averageAPY": 8.5,
    "totalRewards30d": 37500.0,
    "trend": "stable"
  }
}
```

#### 1.4 Network Activity Metrics
```http
POST /v1/query/network-activity
```
**Purpose**: Get detailed network activity metrics
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "7d",
  "granularity": "hour"
}
```
**Response**:
```json
{
  "activity": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "transactions": 1250,
      "blocks": 144,
      "uniqueAddresses": 450,
      "volume": 125000.5
    }
  ],
  "summary": {
    "totalTransactions": 21000,
    "averageTPS": 0.35,
    "peakTPS": 2.1,
    "uniqueAddresses": 1250
  }
}
```

#### 1.5 Block Production Analytics
```http
POST /v1/query/block-production
```
**Purpose**: Get block production rate and validator performance
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "7d"
}
```
**Response**:
```json
{
  "production": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "blocksProduced": 144,
      "averageBlockTime": 6.2,
      "validatorPerformance": {
        "totalValidators": 128,
        "activeValidators": 125,
        "averageUptime": 99.2
      }
    }
  ],
  "summary": {
    "averageBlockTime": 6.2,
    "totalBlocks": 1008,
    "productionRate": 144.0
  }
}
```

### 2. **Transaction View Endpoints**

#### 2.1 Enhanced Transaction Search
```http
POST /v1/query/transactions-advanced
```
**Purpose**: Advanced transaction search with multiple filters
**Request Body**:
```json
{
  "chainId": 1,
  "pageNumber": 1,
  "perPage": 50,
  "filters": {
    "type": "send",
    "status": "success",
    "fromDate": "2024-01-01T00:00:00Z",
    "toDate": "2024-01-31T23:59:59Z",
    "minAmount": 100,
    "maxAmount": 10000,
    "address": "0x123...",
    "blockHeight": 1000
  },
  "sortBy": "timestamp",
  "sortOrder": "desc"
}
```
**Response**:
```json
{
  "results": [
    {
      "hash": "0xabc123...",
      "type": "send",
      "from": "0x123...",
      "to": "0x456...",
      "amount": 1000.5,
      "fee": 0.0023,
      "status": "success",
      "blockHeight": 1000,
      "blockHash": "0xdef456...",
      "timestamp": "2024-01-15T10:30:00Z",
      "gasUsed": 21000,
      "gasPrice": 0.0000001,
      "messageType": "send",
      "rawData": "..."
    }
  ],
  "totalCount": 50000,
  "pageNumber": 1,
  "perPage": 50,
  "totalPages": 1000,
  "hasMore": true
}
```

#### 2.2 Transaction Statistics
```http
POST /v1/query/transaction-stats
```
**Purpose**: Get transaction statistics and metrics
**Request Body**:
```json
{
  "chainId": 1,
  "timeRange": "7d"
}
```
**Response**:
```json
{
  "stats": {
    "totalTransactions": 50000,
    "successfulTransactions": 49500,
    "failedTransactions": 500,
    "pendingTransactions": 0,
    "averageTransactionTime": 6.2,
    "transactionTypes": {
      "send": 40000,
      "stake": 5000,
      "unstake": 2000,
      "governance": 1000,
      "other": 2000
    },
    "volume": {
      "total": 1250000.5,
      "average": 25000.0,
      "median": 15000.0
    }
  },
  "trends": {
    "dailyGrowth": 5.2,
    "weeklyGrowth": 12.5,
    "monthlyGrowth": 25.8
  }
}
```

#### 2.3 Failed Transactions Analysis
```http
POST /v1/query/failed-transactions
```
**Purpose**: Get detailed information about failed transactions
**Request Body**:
```json
{
  "chainId": 1,
  "pageNumber": 1,
  "perPage": 50,
  "timeRange": "7d"
}
```
**Response**:
```json
{
  "results": [
    {
      "hash": "0xabc123...",
      "from": "0x123...",
      "to": "0x456...",
      "amount": 1000.5,
      "fee": 0.0023,
      "errorCode": "INSUFFICIENT_FUNDS",
      "errorMessage": "Account balance too low",
      "blockHeight": 1000,
      "timestamp": "2024-01-15T10:30:00Z",
      "gasUsed": 21000,
      "gasLimit": 21000
    }
  ],
  "totalCount": 500,
  "errorSummary": {
    "INSUFFICIENT_FUNDS": 200,
    "GAS_LIMIT_EXCEEDED": 150,
    "INVALID_SIGNATURE": 100,
    "OTHER": 50
  }
}
```

### 3. **Validator View Endpoints**

#### 3.1 Validator Performance Metrics
```http
POST /v1/query/validator-performance
```
**Purpose**: Get detailed validator performance metrics
**Request Body**:
```json
{
  "chainId": 1,
  "validatorAddress": "0x123...",
  "timeRange": "30d"
}
```
**Response**:
```json
{
  "performance": {
    "address": "0x123...",
    "name": "CanopyGuard",
    "uptime": 99.8,
    "blocksProduced": 1250,
    "blocksMissed": 5,
    "averageBlockTime": 6.1,
    "commission": 5.0,
    "delegationCount": 450,
    "totalDelegated": 1000000.5,
    "selfStake": 50000.0,
    "rewards": {
      "totalEarned": 2500.5,
      "last30Days": 250.0,
      "averageDaily": 8.33,
      "apy": 8.5
    },
    "rank": 15,
    "status": "active",
    "jailed": false,
    "unstakingHeight": 0
  },
  "history": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "blocksProduced": 144,
      "uptime": 100.0,
      "rewards": 8.33
    }
  ]
}
```

#### 3.2 Validator Rewards History
```http
POST /v1/query/validator-rewards
```
**Purpose**: Get detailed validator rewards history
**Request Body**:
```json
{
  "chainId": 1,
  "validatorAddress": "0x123...",
  "timeRange": "30d",
  "pageNumber": 1,
  "perPage": 100
}
```
**Response**:
```json
{
  "rewards": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "blockHeight": 1000,
      "reward": 0.5,
      "commission": 0.025,
      "netReward": 0.475,
      "delegatorRewards": 0.45,
      "type": "block_reward"
    }
  ],
  "summary": {
    "totalRewards": 250.0,
    "totalCommission": 12.5,
    "netRewards": 237.5,
    "averageDaily": 8.33,
    "apy": 8.5
  },
  "totalCount": 720,
  "pageNumber": 1,
  "perPage": 100
}
```

#### 3.3 Validator Delegations
```http
POST /v1/query/validator-delegations
```
**Purpose**: Get validator delegation information
**Request Body**:
```json
{
  "chainId": 1,
  "validatorAddress": "0x123...",
  "pageNumber": 1,
  "perPage": 50
}
```
**Response**:
```json
{
  "delegations": [
    {
      "delegatorAddress": "0x456...",
      "amount": 10000.0,
      "shares": 10000.0,
      "timestamp": "2024-01-15T10:30:00Z",
      "blockHeight": 1000,
      "reward": 0.5,
      "commission": 0.025
    }
  ],
  "summary": {
    "totalDelegations": 450,
    "totalAmount": 1000000.5,
    "averageDelegation": 2222.22,
    "largestDelegation": 50000.0
  },
  "totalCount": 450,
  "pageNumber": 1,
  "perPage": 50
}
```

#### 3.4 Validator Chain Participation
```http
POST /v1/query/validator-chains
```
**Purpose**: Get validator participation in different chains/committees
**Request Body**:
```json
{
  "chainId": 1,
  "validatorAddress": "0x123..."
}
```
**Response**:
```json
{
  "chains": [
    {
      "chainId": 1,
      "chainName": "Canopy Mainnet",
      "committeeId": 1,
      "stakeAmount": 50000.0,
      "status": "active",
      "rewards": 250.0,
      "uptime": 99.8,
      "blocksProduced": 1250
    },
    {
      "chainId": 2,
      "chainName": "Canopy Testnet",
      "committeeId": 2,
      "stakeAmount": 25000.0,
      "status": "active",
      "rewards": 125.0,
      "uptime": 98.5,
      "blocksProduced": 625
    }
  ],
  "summary": {
    "totalChains": 2,
    "totalStake": 75000.0,
    "totalRewards": 375.0,
    "averageUptime": 99.15
  }
}
```

### 4. **Additional Utility Endpoints**

#### 4.1 Network Statistics
```http
POST /v1/query/network-stats
```
**Purpose**: Get comprehensive network statistics
**Request Body**:
```json
{
  "chainId": 1
}
```
**Response**:
```json
{
  "stats": {
    "totalBlocks": 1000000,
    "totalTransactions": 50000000,
    "totalAccounts": 125000,
    "totalValidators": 128,
    "activeValidators": 125,
    "totalStaked": 45513085780613,
    "averageBlockTime": 6.2,
    "networkUptime": 99.98,
    "currentHeight": 1000000,
    "genesisTime": "2023-01-01T00:00:00Z"
  }
}
```

---

## 🔧 **Implementation Priority**

### **Phase 1 - Critical (High Priority)**
1. `POST /v1/query/transactions-advanced` - Enhanced transaction search
2. `POST /v1/query/validator-performance` - Validator performance metrics
3. `POST /v1/query/network-stats` - Network statistics

### **Phase 2 - Important (Medium Priority)**
1. `POST /v1/query/fee-trends` - Fee trends for analytics
2. `POST /v1/query/validator-rewards` - Validator rewards history
3. `POST /v1/query/transaction-stats` - Transaction statistics
4. `POST /v1/query/network-activity` - Network activity metrics

### **Phase 3 - Enhancement (Low Priority)**
1. `POST /v1/query/network-uptime` - Network uptime
2. `POST /v1/query/staking-rewards` - Staking rewards history
3. `POST /v1/query/block-production` - Block production analytics
4. `POST /v1/query/validator-delegations` - Validator delegations
5. `POST /v1/query/validator-chains` - Validator chain participation
6. `POST /v1/query/failed-transactions` - Failed transactions analysis

---

## 📝 **Implementation Notes**

### **Request/Response Format**
- All endpoints use POST method with JSON request body
- Include `chainId` in all requests for multi-chain support
- Use consistent pagination with `pageNumber`, `perPage`, `totalCount`, `totalPages`
- Include proper error handling with HTTP status codes

---

## 🎯 **Expected Outcomes**

Once all endpoints are implemented, the Canopy Explorer will have:

1. **Complete Analytics View** with real-time network metrics, fee trends, and staking analytics
2. **Advanced Transaction View** with comprehensive search, filtering, and analysis capabilities
3. **Detailed Validator View** with performance metrics, rewards history, and delegation information
4. **Enhanced User Experience** with fast search, real-time updates, and comprehensive data visualization

