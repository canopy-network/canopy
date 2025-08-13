# Wrapped Tokens Implementation for Canopy

This document outlines the design and implementation approach for adding wrapped token support to the Canopy blockchain.

## Current Architecture Analysis

Canopy already has:
- **Native CNPY token** as the base currency
- **Pseudo-contract system** for ERC20 compatibility (`fsm/ethereum.go`)
- **Cross-chain swap infrastructure** for token exchanges
- **ERC20 translation layer** with selectors and contract addresses

## Proposed Wrapped Token Architecture

### 1. Extend Pseudo-Contract System

Add new pseudo-contract addresses for wrapped tokens:

```go
// fsm/ethereum.go additions
const WrappedETHContractAddress = `0x0000000000000000000000000000000000000004`
const WrappedBTCContractAddress = `0x0000000000000000000000000000000000000005`
const WrappedUSDCContractAddress = `0x0000000000000000000000000000000000000006`

// Wrapped token selectors
const WrapTokenSelector = "ea598cb0"    // wrapToken(bytes)
const UnwrapTokenSelector = "de0e9a3e"  // unwrapToken(bytes)
```

### 2. New Message Types

Create new protobuf messages:

```protobuf
message MessageWrapToken {
  bytes original_chain_id = 1;           // Source chain ID (e.g., Ethereum)
  bytes original_token_address = 2;      // Original token contract address
  uint64 amount = 3;                     // Amount to wrap
  bytes recipient_address = 4;           // Canopy address to receive wrapped tokens
  bytes proof = 5;                       // Cross-chain proof of lock/burn
}

message MessageUnwrapToken {
  bytes original_chain_id = 1;           // Target chain ID  
  bytes original_token_address = 2;      // Original token contract address
  uint64 amount = 3;                     // Amount to unwrap
  bytes target_address = 4;              // Target chain address
  bytes wrapped_token_id = 5;            // Wrapped token identifier
}
```

### 3. Wrapped Token State Management

Extend the FSM with wrapped token tracking:

```go
// fsm/wrapped_tokens.go (new file)
type WrappedToken struct {
  Id                  []byte   // Unique wrapped token ID
  OriginalChainId     uint64   // Source blockchain
  OriginalTokenAddress []byte  // Original token contract
  TotalSupply         uint64   // Total wrapped supply
  Decimals            uint8    // Token decimals
  Symbol              string   // Token symbol
  Name                string   // Token name
}

type WrappedTokenBalance struct {
  Address     []byte
  TokenId     []byte
  Amount      uint64
}
```

### 4. Oracle Integration

Extend the existing oracle system to:
- **Monitor external chains** for token deposits/locks
- **Verify cross-chain proofs** before minting wrapped tokens
- **Handle unlock/release** when unwrapping tokens

```go
// cmd/rpc/oracle/wrapped_tokens.go (new file)
func (o *Oracle) WitnessTokenDeposit(chainId uint64, tokenAddress, userAddress []byte, amount uint64) (*types.DepositProof, error) {
  // Verify deposit on external chain
  // Generate cryptographic proof
  // Submit to Canopy for wrapped token minting
}
```

## Required Code Changes

### 5. Message Handlers

Add to `fsm/message.go`:

```go
case *MessageWrapToken:
    return s.HandleMessageWrapToken(x)
case *MessageUnwrapToken:
    return s.HandleMessageUnwrapToken(x)

func (s *StateMachine) HandleMessageWrapToken(msg *MessageWrapToken) lib.ErrorI {
    // Verify cross-chain proof via oracle
    // Mint wrapped tokens to recipient
    // Update wrapped token supply
    // Create wrapped token balance entry
}

func (s *StateMachine) HandleMessageUnwrapToken(msg *MessageUnwrapToken) lib.ErrorI {
    // Burn wrapped tokens from sender
    // Initiate cross-chain unlock/release
    // Update wrapped token supply
}
```

### 6. ERC20 Compatibility

Extend `fsm/ethereum.go` to handle wrapped token transfers:

```go
func (s *StateMachine) translateWrappedTokenTx(tx *ethTypes.Transaction) (*lib.Transaction, lib.ErrorI) {
    // Handle wrapped token ERC20-style transactions
    // Route to appropriate wrapped token balances
    // Maintain full ERC20 compatibility
}
```

### 7. RPC Endpoints

Add new API endpoints in `cmd/rpc/`:

```go
// Query wrapped token balances
func (s *Server) GetWrappedTokenBalance(w http.ResponseWriter, r *http.Request)

// List all wrapped tokens
func (s *Server) GetWrappedTokens(w http.ResponseWriter, r *http.Request)

// Get wrapped token info
func (s *Server) GetWrappedTokenInfo(w http.ResponseWriter, r *http.Request)
```

### 8. CLI Commands

Add CLI support in `cmd/cli/`:

```bash
canopy wrap-token --chain ethereum --token 0x123... --amount 1000
canopy unwrap-token --token-id wrapped_eth_001 --amount 500 --target 0x456...
canopy query wrapped-balance --address canopy1abc... --token wrapped_eth
```

## Storage Implementation Options

### Option 1: Separate Wrapped Token Balance System (Recommended)

Create a new key-value prefix system specifically for wrapped tokens:

```go
// fsm/key.go additions
var (
    wrappedTokenPrefix        = []byte{15} // store key prefix for wrapped token metadata
    wrappedTokenBalancePrefix = []byte{16} // store key prefix for wrapped token balances
)

func WrappedTokenPrefix() []byte { return lib.JoinLenPrefix(wrappedTokenPrefix) }
func WrappedTokenBalancePrefix() []byte { return lib.JoinLenPrefix(wrappedTokenBalancePrefix) }

// Key: wrappedTokenBalancePrefix + address + tokenId
func KeyForWrappedTokenBalance(address crypto.AddressI, tokenId []byte) []byte {
    return lib.JoinLenPrefix(wrappedTokenBalancePrefix, address.Bytes(), tokenId)
}

// Key: wrappedTokenPrefix + tokenId  
func KeyForWrappedToken(tokenId []byte) []byte {
    return lib.JoinLenPrefix(wrappedTokenPrefix, tokenId)
}
```

**Storage Structure:**
```go
// fsm/wrapped_token.go (new file)
type WrappedToken struct {
    Id                   []byte  // Unique token identifier
    OriginalChainId      uint64  // Source blockchain
    OriginalTokenAddress []byte  // Original contract address
    TotalSupply          uint64  // Total wrapped supply
    Symbol               string  // Token symbol (e.g., "wETH")
    Name                 string  // Token name (e.g., "Wrapped Ethereum")
    Decimals             uint8   // Token decimals
}

type WrappedTokenBalance struct {
    Address crypto.AddressI  // Account address
    TokenId []byte          // Wrapped token ID
    Amount  uint64          // Balance amount
}

// Get wrapped token balance
func (s *StateMachine) GetWrappedTokenBalance(address crypto.AddressI, tokenId []byte) (uint64, lib.ErrorI) {
    bz, err := s.Get(KeyForWrappedTokenBalance(address, tokenId))
    if err != nil {
        return 0, nil // Return 0 if no balance exists
    }
    balance := &WrappedTokenBalance{}
    if err = lib.Unmarshal(bz, balance); err != nil {
        return 0, err
    }
    return balance.Amount, nil
}

// Set wrapped token balance
func (s *StateMachine) SetWrappedTokenBalance(address crypto.AddressI, tokenId []byte, amount uint64) lib.ErrorI {
    if amount == 0 {
        // Delete zero balances to save space
        return s.Delete(KeyForWrappedTokenBalance(address, tokenId))
    }
    balance := &WrappedTokenBalance{
        Address: address,
        TokenId: tokenId,
        Amount:  amount,
    }
    bz, err := lib.Marshal(balance)
    if err != nil {
        return err
    }
    return s.Set(KeyForWrappedTokenBalance(address, tokenId), bz)
}
```

## Iterating Over Wrapped Token Balances

Here's how to iterate over all wrapped token balances for a specific address:

```go
// fsm/wrapped_token.go

// GetAllWrappedTokenBalances() returns all wrapped token balances for a specific address
func (s *StateMachine) GetAllWrappedTokenBalances(address crypto.AddressI) ([]*WrappedTokenBalance, lib.ErrorI) {
	var result []*WrappedTokenBalance
	
	// Create iterator over the wrapped token balance prefix + address
	addressPrefix := lib.JoinLenPrefix(wrappedTokenBalancePrefix, address.Bytes())
	it, err := s.Iterator(addressPrefix)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	
	// Iterate through all balances for this address
	for ; it.Valid(); it.Next() {
		// Extract tokenId from the key
		tokenId, e := TokenIdFromBalanceKey(it.Key())
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		
		// Unmarshal the balance
		balance := &WrappedTokenBalance{}
		if e = lib.Unmarshal(it.Value(), balance); e != nil {
			s.log.Error(e.Error())
			continue
		}
		
		// Set the address and tokenId (since they're not stored in the value)
		balance.Address = address
		balance.TokenId = tokenId
		
		result = append(result, balance)
	}
	
	return result, nil
}

// GetAllWrappedTokenBalancesPaginated() returns a paginated list of wrapped token balances for an address
func (s *StateMachine) GetAllWrappedTokenBalancesPaginated(address crypto.AddressI, p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// Create a new page for wrapped token balances
	page, res := lib.NewPage(p, WrappedTokenBalancesPageName), make(WrappedTokenBalancePage, 0)
	
	// Create prefix for this specific address
	addressPrefix := lib.JoinLenPrefix(wrappedTokenBalancePrefix, address.Bytes())
	
	// Load the page using the address-specific prefix
	err = page.Load(addressPrefix, false, &res, s.store, func(key, value []byte) (err lib.ErrorI) {
		// Extract tokenId from key
		tokenId, err := TokenIdFromBalanceKey(key)
		if err != nil {
			return err
		}
		
		// Unmarshal balance
		balance := &WrappedTokenBalance{}
		if err = lib.Unmarshal(value, balance); err != nil {
			return err
		}
		
		// Set the address and tokenId
		balance.Address = address
		balance.TokenId = tokenId
		
		// Add to results
		res = append(res, balance)
		return nil
	})
	
	return page, err
}

// Helper function to extract tokenId from a balance key
func TokenIdFromBalanceKey(key []byte) ([]byte, lib.ErrorI) {
	// Key structure: wrappedTokenBalancePrefix + addressLen + address + tokenIdLen + tokenId
	// We need to skip the prefix and address to get to the tokenId
	
	segments, err := lib.SplitLenPrefix(key)
	if err != nil {
		return nil, err
	}
	
	// segments[0] = wrappedTokenBalancePrefix
	// segments[1] = address  
	// segments[2] = tokenId
	if len(segments) < 3 {
		return nil, lib.ErrInvalidKey()
	}
	
	return segments[2], nil
}

// GetWrappedTokenBalancesForAllAddresses() returns all wrapped token balances in the system
func (s *StateMachine) GetWrappedTokenBalancesForAllAddresses() (map[string][]*WrappedTokenBalance, lib.ErrorI) {
	result := make(map[string][]*WrappedTokenBalance)
	
	// Iterate over ALL wrapped token balances
	it, err := s.Iterator(WrappedTokenBalancePrefix())
	if err != nil {
		return nil, err
	}
	defer it.Close()
	
	for ; it.Valid(); it.Next() {
		// Extract address and tokenId from key
		address, tokenId, e := AddressAndTokenIdFromBalanceKey(it.Key())
		if e != nil {
			s.log.Error(e.Error())
			continue
		}
		
		// Unmarshal balance
		balance := &WrappedTokenBalance{}
		if e = lib.Unmarshal(it.Value(), balance); e != nil {
			s.log.Error(e.Error())
			continue
		}
		
		// Set fields
		balance.Address = crypto.NewAddressFromBytes(address)
		balance.TokenId = tokenId
		
		// Group by address string
		addressStr := lib.BytesToString(address)
		result[addressStr] = append(result[addressStr], balance)
	}
	
	return result, nil
}

// Helper function to extract both address and tokenId from a balance key
func AddressAndTokenIdFromBalanceKey(key []byte) (address []byte, tokenId []byte, err lib.ErrorI) {
	segments, err := lib.SplitLenPrefix(key)
	if err != nil {
		return nil, nil, err
	}
	
	// segments[0] = wrappedTokenBalancePrefix
	// segments[1] = address
	// segments[2] = tokenId  
	if len(segments) < 3 {
		return nil, nil, lib.ErrInvalidKey()
	}
	
	return segments[1], segments[2], nil
}

// Support for pagination
const WrappedTokenBalancesPageName = "wrapped_token_balances"

type WrappedTokenBalancePage []*WrappedTokenBalance

func (p *WrappedTokenBalancePage) New() lib.Pageable { return &WrappedTokenBalancePage{} }

// Register the page type
func init() {
	lib.RegisteredPageables[WrappedTokenBalancesPageName] = new(WrappedTokenBalancePage)
}
```

## Usage Examples

```go
// Get all wrapped token balances for a specific address
address := crypto.NewAddressFromString("canopy1abc...")
balances, err := fsm.GetAllWrappedTokenBalances(address)
if err != nil {
    return err
}

for _, balance := range balances {
    fmt.Printf("Address: %s, Token: %s, Amount: %d\n", 
        balance.Address.String(), 
        lib.BytesToString(balance.TokenId), 
        balance.Amount)
}

// Get paginated results (for large numbers of wrapped tokens)
pageParams := lib.PageParams{
    Limit:  10,
    Offset: 0,
}
page, err := fsm.GetAllWrappedTokenBalancesPaginated(address, pageParams)
if err != nil {
    return err
}

wrappedBalances := page.Data.(*WrappedTokenBalancePage)
for _, balance := range *wrappedBalances {
    // Process each balance
}

// Add wrapped token balance
tokenId := []byte("wrapped_eth_001")
address := crypto.NewAddressFromString("canopy1abc...")
err := fsm.SetWrappedTokenBalance(address, tokenId, 1000000) // 1 wETH

// Get wrapped token balance  
balance, err := fsm.GetWrappedTokenBalance(address, tokenId)

// Transfer wrapped tokens
err = fsm.WrappedTokenTransfer(fromAddr, toAddr, tokenId, amount)
```

## Implementation Benefits

1. **Full ERC20 Compatibility** - Wrapped tokens work with existing Ethereum tooling
2. **Cross-Chain Bridge** - Secure token movement between chains
3. **Decentralized Validation** - Committee consensus for wrap/unwrap operations
4. **Native Integration** - Wrapped tokens participate in Canopy's swap system
5. **Oracle Security** - Cryptographic proofs prevent double-spending
6. **Efficient Storage** - Dedicated key space with address-specific iteration
7. **Pagination Support** - Handles large numbers of wrapped tokens gracefully

## Security Considerations

- **Oracle Consensus** - Require committee majority for wrap/unwrap operations
- **Proof Verification** - Validate cross-chain transaction proofs
- **Rate Limiting** - Prevent spam attacks on wrap/unwrap functions
- **Emergency Pause** - Admin functions to halt wrapping during incidents

## Key Benefits of the Storage Approach

1. **Efficient Address-Specific Queries** - Uses key prefix to only iterate over balances for a specific address
2. **Pagination Support** - Handles large numbers of wrapped tokens gracefully  
3. **System-Wide Queries** - Can iterate over ALL wrapped token balances when needed
4. **Memory Efficient** - Doesn't load unnecessary data into memory
5. **Follows Canopy Patterns** - Uses the same iteration patterns as accounts, validators, etc.

The key insight is that by structuring the keys as `prefix + address + tokenId`, you can efficiently query:
- All balances for a specific address (iterate with address prefix)
- A specific balance (direct key lookup)
- All balances in the system (iterate with just the base prefix)

This wrapped token implementation would leverage Canopy's existing infrastructure while adding robust cross-chain token bridging capabilities. The design maintains compatibility with Ethereum tooling while providing the security and decentralization benefits of Canopy's committee-based consensus system.

## Issues with Nested-Chain-Only Implementation

### Root Chain Integration Problems

If wrapped tokens only exist on nested chains, several significant issues arise:

#### 1. **Root Chain Order Book Problem**
From the swap analysis (`controller/result.go:202-204`):
- The **root chain maintains the master order book** for all cross-chain swaps
- All chains query the root chain's order book to see available swaps
- If wrapped tokens only exist on nested chains, **they can't participate in the global swap system**

#### 2. **Cross-Chain Liquidity Fragmentation**
```
Root Chain: Has CNPY, can see all orders
Nested Chain A: Has wETH, wBTC (isolated)  
Nested Chain B: Has wUSDC, wDAI (isolated)
```
Users can't easily swap between wrapped tokens on different chains.

#### 3. **Committee Validation Issues**
Looking at `fsm/committee.go:81-117` - committees get subsidized based on stake percentage:
- Root chain committee validates all cross-chain activities
- If wrapped tokens are only on nested chains, **root chain can't validate wrapped token operations**
- Creates trust/security gaps in the bridge system

#### 4. **Oracle Integration Problems**
From `controller/result.go:213` - nested chains report to root chain via certificate results:
- Wrapped token operations would need **root chain awareness** for proper validation
- Without root chain integration, oracle consensus becomes fragmented

## Alternative Architectures (When Root Chain Cannot Be Modified)

### Option 1: Nested Chain with Oracle Bridge (Recommended)

Create a **standalone wrapped token system** on your nested chain that uses external oracles for cross-chain validation:

```go
// Your nested chain: wrapped_token.go
type ExternalOracle struct {
    ChainId     uint64   // External chain (Ethereum, Bitcoin, etc.)
    Validators  [][]byte // Oracle validator addresses  
    Threshold   uint64   // Required signatures for consensus
}

func (s *StateMachine) HandleMessageWrapToken(msg *MessageWrapToken) lib.ErrorI {
    // Verify external oracle consensus (not root chain committee)
    if !s.VerifyExternalOracleProof(msg.Proof, msg.OriginalChainId) {
        return ErrInvalidOracleProof()
    }
    
    // Mint wrapped tokens on THIS nested chain only
    return s.MintWrappedToken(msg.RecipientAddress, msg.Amount, msg.OriginalTokenAddress)
}
```

### Option 2: Multi-Signature Bridge Contracts

Deploy bridge contracts on external chains that your nested chain can validate:

```solidity
// Ethereum side: Bridge contract
contract CanopyBridge {
    mapping(address => bool) public validators;
    uint256 public threshold;
    
    function lockTokens(address token, uint256 amount, bytes32 canopyAddress) external {
        // Lock tokens in escrow
        // Emit event for nested chain oracle to witness
    }
    
    function unlockTokens(address token, uint256 amount, bytes[] memory signatures) external {
        // Verify nested chain validator signatures
        // Release tokens from escrow
    }
}
```

```go
// Your nested chain: Monitor Ethereum events
func (o *Oracle) WatchEthereumDeposits() {
    // Watch for lockTokens events
    // Generate proofs for nested chain
    // Submit MessageWrapToken transactions
}
```

### Option 3: Native Token Swaps (No Root Chain Dependency) 

Create a **direct peer-to-peer swap system** on your nested chain:

```go
// Native cross-chain swaps without root chain involvement
type DirectSwapOrder struct {
    SellTokenId     []byte  // Your wrapped token
    BuyChainId      uint64  // External chain
    BuyTokenAddress []byte  // External token contract
    SellerAddress   []byte  // Your nested chain address
    BuyerProofAddr  []byte  // External chain address for proof
}

func (s *StateMachine) CreateDirectSwap(order *DirectSwapOrder) lib.ErrorI {
    // Escrow wrapped tokens on nested chain
    // Wait for buyer to prove external payment
    // Release escrowed tokens to buyer
}
```

### Option 4: Federated Oracle Network

Create your own oracle network for the nested chain:

```go
// oracle/federated_oracle.go
type FederatedOracle struct {
    Nodes     map[string]*OracleNode  // Independent oracle operators
    Consensus uint64                  // Required agreement percentage
}

func (f *FederatedOracle) ValidateExternalDeposit(chainId uint64, txHash []byte) (*DepositProof, error) {
    // Query multiple oracle nodes
    // Require consensus threshold
    // Generate cryptographic proof
    // Submit to nested chain for wrapped token minting
}
```

### Option 5: Plasma-Style Commitments

Use the existing **certificate results system** to commit wrapped token state to root chain without modifying it:

```go
// Include wrapped token state in existing certificate results
func (s *StateMachine) ExtendCertificateResults(results *lib.CertificateResult) {
    // Pack wrapped token state changes into existing certificate data
    // Root chain stores this opaquely without understanding it
    // Other nested chains can verify wrapped token states
    wrappedTokenData := s.SerializeWrappedTokenChanges()
    
    // Pack into existing certificate results structure
    if results.Orders == nil {
        results.Orders = &lib.Orders{}
    }
    // Use existing data field to store wrapped token commitments
    results.Orders.LockOrders = append(results.Orders.LockOrders, &lib.LockOrder{
        Data: wrappedTokenData,  // Piggyback on existing structure
    })
}
```

### Option 6: Inter-Chain Communication Protocol

Create a **standardized message format** that nested chains can use to communicate wrapped token operations:

```go
// inter_chain_messaging.go
type InterChainMessage struct {
    SourceChainId uint64
    TargetChainId uint64
    MessageType   string  // "wrap_token", "unwrap_token", "transfer"
    Payload       []byte  // Serialized wrapped token operation
    Signatures    [][]byte // Multi-sig from source chain validators
}

func (s *StateMachine) SendInterChainMessage(targetChain uint64, msg *InterChainMessage) lib.ErrorI {
    // Send via existing P2P network
    // Target chain validates signatures
    // Execute wrapped token operation
}
```

## Recommended Architecture (No Root Chain Modifications)

**Hybrid Federated Oracle + Direct Swaps:**

1. **External Oracle Network** - Independent operators validate cross-chain deposits
2. **Nested Chain Wrapped Tokens** - Full ERC20 compatibility on your chain
3. **Direct P2P Swaps** - Users can trade wrapped tokens directly
4. **Bridge Contracts** - Secure lock/unlock on external chains
5. **Certificate Commitments** - Use existing root chain infrastructure for state commitments

```go
// Your implementation would focus on:
type WrappedTokenSystem struct {
    Oracle     *FederatedOracle      // External validation
    Bridge     *BridgeContractManager // Cross-chain contracts  
    SwapEngine *DirectSwapEngine     // P2P trading
    Committer  *StateCommitter       // Root chain commitments
}
```

### Benefits of This Approach:

1. **No Root Chain Modifications Required** - Works with existing Canopy infrastructure
2. **Full Wrapped Token Functionality** - Complete ERC20 compatibility
3. **Cross-Chain Security** - Multi-signature validation from independent oracles
4. **Direct Trading** - P2P swaps without root chain dependency
5. **State Commitments** - Leverage existing certificate results for transparency

### Implementation Strategy:

1. **Phase 1**: Implement wrapped token storage and ERC20 compatibility on nested chain
2. **Phase 2**: Deploy bridge contracts on external chains (Ethereum, etc.)
3. **Phase 3**: Build federated oracle network for cross-chain validation
4. **Phase 4**: Create direct swap system for wrapped token trading
5. **Phase 5**: Add state commitments via existing certificate results

This approach gives you **full wrapped token functionality** without requiring root chain modifications, while still leveraging Canopy's existing infrastructure where possible.