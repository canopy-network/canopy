# Network Analytics - Real vs Simulated Data

## 📊 **REAL DATA** (obtained from API)

### ✅ **Available in API:**
- **Validator Count** - Real number of active validators
- **Total Value Locked (TVL)** - Real total stake (45.5T wei = ~45.5M CNPY)
- **Pending Transactions** - Real number of pending transactions
- **Block Time** - Calculated from real block timestamps
- **Block Size** - Real block size (1MB)
- **Validator Weights** - Real distribution based on validator stakes
- **Transaction Types** - Real categorization by `messageType` (certificateResults)
- **Network Activity** - Based on real transaction data
- **Block Production Rate** - Based on real block data

### 🔄 **API Hooks used:**
- `useValidators()` - To count validators and calculate distribution
- `useSupply()` - To get real TVL (staked: 45513085780613 wei)
- `usePending()` - To count pending transactions (totalCount: 0)
- `useTransactions()` - For network activity data (messageType: certificateResults)
- `useBlocks()` - For block production data (height: 634691)
- `useParams()` - For consensus parameters (blockSize: 1000000)

## 🎭 **SIMULATED DATA** (not available in API)

### ❌ **Not available in API:**
- **Network Uptime** - No uptime endpoint
- **Average Transaction Fee (7d)** - No historical averages endpoint
- **Network Version** - Not available in API
- **Staking Trends** - No historical rewards endpoint
- **Fee Trends** - No fee trends endpoint

### 📈 **Charts with hybrid data:**
- **Network Activity** - Uses real data as base, simulates temporal distribution
- **Block Production Rate** - Uses real data as base, simulates temporal distribution
- **Transaction Types** - Categorizes real data by `messageType` (certificateResults), simulates temporal distribution

## 🏷️ **Visual Indicators**

In the interface, labels are shown to distinguish:
- 🟢 **(REAL)** - Data obtained directly from API
- 🟠 **(SIM)** - Simulated data because not available in API

## 🔧 **Future Improvements**

To get more real data, new endpoints would be needed:
1. **Network Uptime** - Network health endpoint
2. **Historical Fees** - Historical fees endpoint
3. **Staking Rewards** - Historical rewards endpoint
4. **Network Version** - Version information endpoint
5. **Historical Data** - Endpoints for historical transaction and block data

## 📝 **Technical Notes**

- Real data updates automatically via React Query
- Simulated data updates every 5 seconds to simulate real-time
- Charts use real data as base and apply realistic variations
- Transaction categorization uses real `messageType` field (certificateResults)
- Block Time is calculated from real timestamps of consecutive blocks
- TVL is converted from wei to millions (45.5T wei = ~45.5M CNPY)
- Block Size is obtained from params.consensus.blockSize (1MB)