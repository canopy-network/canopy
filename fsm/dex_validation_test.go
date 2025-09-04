package fsm

import (
	"math/rand"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
)

func TestDexValidation(t *testing.T) {
	for i := 0; i < 1000; i++ {
		const (
			chain1Id = uint64(1)
			chain2Id = uint64(2)

			initialPoolAmount = uint64(10000)
			initialUserFunds  = uint64(1000)
		)

		// initialize chains
		chain1, chain2 := newTestStateMachine(t), newTestStateMachine(t)

		// setup accounts
		accounts := []crypto.AddressI{
			newTestAddress(t, 0),
			newTestAddress(t, 1),
			newTestAddress(t, 2),
		}

		for _, account := range accounts {
			require.NoError(t, chain1.AccountAdd(account, initialUserFunds))
			require.NoError(t, chain2.AccountAdd(account, initialUserFunds))
		}

		// initialize pools
		require.NoError(t, chain1.SetPool(&Pool{
			Id:     chain2Id + LiquidityPoolAddend,
			Amount: initialPoolAmount,
		}))
		require.NoError(t, chain2.SetPool(&Pool{
			Id:     chain1Id + LiquidityPoolAddend,
			Amount: initialPoolAmount,
		}))

		t.Logf("=== INITIAL STATE ===")
		logFullState(t, &chain1, &chain2, chain1Id, chain2Id, accounts)
		// Test 1: Validate a complete cross-chain swap (AMM mechanics)
		t.Logf("\n=== TEST 1: CROSS-CHAIN SWAP (AMM VALIDATION) ===")
		validateCrossChainSwap(t, &chain1, &chain2, chain1Id, chain2Id, accounts[0])
		clearLocks(t, chain1, chain2, chain1Id, chain2Id)
		// Test 2: Validate a complete liquidity deposit
		t.Logf("\n=== TEST 2: LIQUIDITY DEPOSIT VALIDATION ===")
		validateLiquidityDeposit(t, &chain1, &chain2, chain1Id, chain2Id, accounts[1])
		clearLocks(t, chain1, chain2, chain1Id, chain2Id)

		// Test 3: Validate a complete liquidity withdraw
		t.Logf("\n=== TEST 3: LIQUIDITY WITHDRAW VALIDATION ===")
		validateLiquidityWithdraw(t, &chain1, &chain2, chain1Id, chain2Id, accounts[1])
		clearLocks(t, chain1, chain2, chain1Id, chain2Id)
		
		// Final comprehensive validation
		t.Logf("\n=== FINAL VALIDATION ===")
		validateFinalSystemState(t, &chain1, &chain2, chain1Id, chain2Id, accounts)
	}
}

// calculateAMMOutput calculates expected output using Uniswap V2 formula
func calculateAMMOutput(dX, x, y uint64) uint64 {
	amountInWithFee := dX * 997
	return (amountInWithFee * y) / (x*1000 + amountInWithFee)
}

// validateCrossChainSwap tests a complete cross-chain swap following the working pattern
func validateCrossChainSwap(t *testing.T, chain1, chain2 *StateMachine, chain1Id, chain2Id uint64, account crypto.AddressI) {
	// Use randomized swap amount between 50-200 tokens
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	swapAmount := uint64(50 + rng.Intn(151)) // 50-200 tokens

	// Get current pool states to calculate expected output
	initialPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	initialPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Calculate expected output using AMM formula
	expectedOutput := calculateAMMOutput(swapAmount, initialPool2, initialPool1)

	// Set requested amount slightly below expected (conservative request)
	requestedAmount := expectedOutput * 95 / 100

	// Record initial state
	initialBalance2 := getAccountBalance(t, chain2, account)
	initialBalance1 := getAccountBalance(t, chain1, account)

	t.Logf("Random swap: %d tokens (expected output: %d, requesting: %d)",
		swapAmount, expectedOutput, requestedAmount)

	// Step 1: Submit swap on chain2 targeting chain1
	err := chain2.HandleMessageDexLimitOrder(&MessageDexLimitOrder{
		ChainId:         chain1Id,
		AmountForSale:   swapAmount,
		RequestedAmount: requestedAmount,
		Address:         account.Bytes(),
	})
	require.NoError(t, err)

	// Verify funds moved to holding pool
	holdingBalance := getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend)
	require.Equal(t, swapAmount, holdingBalance, "Funds should move to holding pool")

	// Step 2: Process complete cross-chain cycle (following TestDexSwap pattern)
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initialPool2,
	}
	emptyBatch.EnsureNonNil()

	err = chain2.HandleRemoteDexBatch(emptyBatch, chain1Id)
	require.NoError(t, err)

	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)

	err = chain1.HandleRemoteDexBatch(lockedBatch, chain2Id)
	require.NoError(t, err)

	replyBatch, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)

	if !replyBatch.IsEmpty() {
		err = chain2.HandleRemoteDexBatch(replyBatch, chain1Id)
		require.NoError(t, err)
	}

	// Step 3: Validate AMM mechanics worked correctly
	finalBalance1 := getAccountBalance(t, chain1, account)
	finalBalance2 := getAccountBalance(t, chain2, account)
	finalPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	finalPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Validate holding pools are cleared
	require.Equal(t, uint64(0), getPoolBalance(t, chain1, chain2Id+HoldingPoolAddend))
	require.Equal(t, uint64(0), getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend))

	tokensReceived := finalBalance1 - initialBalance1
	tokensGivenOut := initialPool1 - finalPool1
	tokensReceivedByPool := finalPool2 - initialPool2

	require.Equal(t, tokensReceived, tokensGivenOut, "Tokens out should equal tokens received")
	require.Equal(t, swapAmount, tokensReceivedByPool, "Pool should receive the swap amount")
	require.Equal(t, initialBalance2-swapAmount, finalBalance2, "Should spend tokens on chain2")

	// Validate proper AMM math using Uniswap V2 formula
	validateAMMFormula(t, swapAmount, tokensReceived, initialPool2, initialPool1)

	// Validate the swap met the minimum requested amount
	require.GreaterOrEqual(t, tokensReceived, requestedAmount,
		"Swap output (%d) should meet minimum requested amount (%d)", tokensReceived, requestedAmount)

	t.Logf("Swap: %d tokens → %d tokens (AMM slippage: %.2f%%)",
		swapAmount, tokensReceived, float64(swapAmount-tokensReceived)*100/float64(swapAmount))
}

// validateLiquidityDeposit tests a complete liquidity deposit following the working pattern
func validateLiquidityDeposit(t *testing.T, chain1, chain2 *StateMachine, chain1Id, chain2Id uint64, account crypto.AddressI) {
	// Use randomized deposit amount between 100-500 tokens
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	depositAmount := uint64(100 + rng.Intn(401)) // 100-500 tokens

	// Get current pool states (after previous swap test)
	initialPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	initialPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Get current liquidity point state
	pool1, _ := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	pool2, _ := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	initialLPPoints1 := pool1.TotalPoolPoints
	initialLPPoints2 := pool2.TotalPoolPoints

	// Record initial user balance on chain2 (where deposit originates)
	initialBalance2 := getAccountBalance(t, chain2, account)

	t.Logf("Depositing %d tokens to chain1 pool (targeting chain2)", depositAmount)

	// Step 1: Submit liquidity deposit on chain2 targeting chain1 (following TestDexDeposit)
	err := chain2.HandleMessageDexLiquidityDeposit(&MessageDexLiquidityDeposit{
		ChainId: chain1Id,
		Address: account.Bytes(),
		Amount:  depositAmount,
	})
	require.NoError(t, err)

	// Verify funds moved to holding pool on chain2
	holdingBalance := getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend)
	require.Equal(t, depositAmount, holdingBalance, "Funds should move to holding pool")

	// Step 2: Process complete cross-chain cycle (following TestDexDeposit pattern)
	// Chain2: trigger 'handle batch' with an empty batch from Chain1 and ensure 'next batch' became 'locked'
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initialPool1,
	}
	emptyBatch.EnsureNonNil()

	err = chain2.HandleRemoteDexBatch(emptyBatch, chain1Id)
	require.NoError(t, err)

	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)

	// Chain1: trigger 'handle batch' with the 'locked batch' from Chain2
	err = chain1.HandleRemoteDexBatch(lockedBatch, chain2Id)
	require.NoError(t, err)

	// Chain1: get the reply batch
	replyBatch, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)

	// Chain2: complete the cycle by processing the reply batch
	err = chain2.HandleRemoteDexBatch(replyBatch, chain1Id)
	require.NoError(t, err)

	// Step 3: Validate liquidity deposit mechanics worked correctly
	finalBalance2 := getAccountBalance(t, chain2, account)
	finalPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	finalPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Get final liquidity point state
	pool1Final, _ := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	pool2Final, _ := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	finalLPPoints1 := pool1Final.TotalPoolPoints
	finalLPPoints2 := pool2Final.TotalPoolPoints

	// Validate holding pools are cleared
	require.Equal(t, uint64(0), getPoolBalance(t, chain1, chain2Id+HoldingPoolAddend))
	require.Equal(t, uint64(0), getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend))

	// Validate user balance decreased by deposit amount
	require.Equal(t, initialBalance2-depositAmount, finalBalance2, "Should spend deposit amount")

	// Validate liquidity pool increased by deposit amount (on chain2 where deposit originated)
	require.Equal(t, initialPool2+depositAmount, finalPool2, "Pool should receive deposit amount")

	// Validate pool reserves remained constant on the other chain (no AMM activity)
	require.Equal(t, initialPool1, finalPool1, "Counter-pool should remain unchanged")

	// Validate liquidity points were assigned correctly (both chains get points symmetrically)
	require.Greater(t, finalLPPoints1, initialLPPoints1, "LP points should increase on target chain")
	require.Greater(t, finalLPPoints2, initialLPPoints2, "LP points should increase on deposit chain")

	// Validate user received liquidity points on both chains
	userPoints1, err := pool1Final.GetPointsFor(account.Bytes())
	require.NoError(t, err)
	userPoints2, err := pool2Final.GetPointsFor(account.Bytes())
	require.NoError(t, err)
	require.Greater(t, userPoints1, uint64(0), "User should receive liquidity points on target chain")
	require.Greater(t, userPoints2, uint64(0), "User should receive liquidity points on deposit chain")

	// Calculate expected points using the geometric mean formula for the deposit chain
	// ΔL = L * (√((x + deposit) * y) - √(x * y)) / √(x * y)
	if initialLPPoints2 > 0 {
		oldK := initialPool2 * initialPool1
		newK := (initialPool2 + depositAmount) * initialPool1
		expectedPoints := initialLPPoints2 * (lib.IntSqrt(newK) - lib.IntSqrt(oldK)) / lib.IntSqrt(oldK)

		require.Equal(t, expectedPoints, userPoints2,
			"User points (%d) should match expected (%d)", userPoints2, expectedPoints)
	}

	t.Logf("Deposit: %d tokens → %d LP points (chain1), %d LP points (chain2)", depositAmount, userPoints1, userPoints2)
	t.Logf("Pool1: %d (unchanged), Pool2: %d → %d (+%d)",
		initialPool1, initialPool2, finalPool2, depositAmount)
	t.Logf("LP Points1: %d → %d (+%d), LP Points2: %d → %d (+%d)",
		initialLPPoints1, finalLPPoints1, finalLPPoints1-initialLPPoints1,
		initialLPPoints2, finalLPPoints2, finalLPPoints2-initialLPPoints2)
}

// validateLiquidityWithdraw tests a complete liquidity withdraw following the working pattern
func validateLiquidityWithdraw(t *testing.T, chain1, chain2 *StateMachine, chain1Id, chain2Id uint64, account crypto.AddressI) {
	// Use randomized withdraw percentage for robust testing
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	withdrawPercent := uint64(25 + rng.Intn(76)) // 25-100% withdrawal

	// Get current pool states (after previous tests)
	initialPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	initialPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Get current liquidity point state
	pool1, _ := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	pool2, _ := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	initialLPPoints1 := pool1.TotalPoolPoints
	initialLPPoints2 := pool2.TotalPoolPoints

	// Get user's current liquidity points to calculate withdrawal amounts
	userPoints1, err := pool1.GetPointsFor(account.Bytes())
	require.NoError(t, err)
	userPoints2, err := pool2.GetPointsFor(account.Bytes())
	require.NoError(t, err)

	// Skip if user has no liquidity points (from previous deposit test)
	if userPoints1 == 0 && userPoints2 == 0 {
		t.Logf("Skipping withdraw test - user has no liquidity points")
		return
	}

	// Validate user has symmetric points (should be equal on both chains)
	require.Equal(t, userPoints1, userPoints2, "User should have equal LP points on both chains")

	// Record initial user balances
	initialBalance1 := getAccountBalance(t, chain1, account)
	initialBalance2 := getAccountBalance(t, chain2, account)

	t.Logf("Withdrawing %d%% of liquidity (%d points on chain1, %d points on chain2)",
		withdrawPercent, userPoints1, userPoints2)

	// Calculate expected withdrawal amounts using pro-rata calculation with percentage
	// Points to be removed = floor(userPoints * withdrawPercent / 100)
	pointsToRemove1 := userPoints1 * withdrawPercent / 100
	pointsToRemove2 := userPoints2 * withdrawPercent / 100

	// Expected withdrawal amounts based on points being removed
	expectedWithdraw1 := initialPool1 * pointsToRemove1 / initialLPPoints1
	expectedWithdraw2 := initialPool2 * pointsToRemove2 / initialLPPoints2

	// Validate withdrawal amounts don't exceed pool balances (sanity check)
	require.LessOrEqual(t, expectedWithdraw1, initialPool1, "Withdrawal1 should not exceed pool balance")
	require.LessOrEqual(t, expectedWithdraw2, initialPool2, "Withdrawal2 should not exceed pool balance")

	// Step 1: Submit liquidity withdraw on chain2 targeting chain1 (following TestDexWithdraw)
	err = chain2.HandleMessageDexLiquidityWithdraw(&MessageDexLiquidityWithdraw{
		ChainId: chain1Id,
		Percent: withdrawPercent,
		Address: account.Bytes(),
	})
	require.NoError(t, err)

	// Step 2: Process complete cross-chain cycle (following TestDexWithdraw pattern)
	// Chain2: trigger 'handle batch' with an empty batch from Chain1 and ensure 'next batch' became 'locked'
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initialPool1,
	}
	emptyBatch.EnsureNonNil()

	err = chain2.HandleRemoteDexBatch(emptyBatch, chain1Id)
	require.NoError(t, err)

	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)

	// Chain1: trigger 'handle batch' with the 'locked batch' from Chain2
	err = chain1.HandleRemoteDexBatch(lockedBatch, chain2Id)
	require.NoError(t, err)

	// Chain1: get the reply batch
	replyBatch, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)

	// Chain2: complete the cycle by processing the reply batch
	err = chain2.HandleRemoteDexBatch(replyBatch, chain1Id)
	require.NoError(t, err)

	// Step 3: Validate liquidity withdraw mechanics worked correctly
	finalBalance1 := getAccountBalance(t, chain1, account)
	finalBalance2 := getAccountBalance(t, chain2, account)
	finalPool1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend)
	finalPool2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend)

	// Get final liquidity point state
	pool1Final, _ := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	pool2Final, _ := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	finalLPPoints1 := pool1Final.TotalPoolPoints
	finalLPPoints2 := pool2Final.TotalPoolPoints

	// Validate holding pools are cleared
	require.Equal(t, uint64(0), getPoolBalance(t, chain1, chain2Id+HoldingPoolAddend))
	require.Equal(t, uint64(0), getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend))

	// Validate user received withdrawal amounts on both chains
	tokensReceived1 := finalBalance1 - initialBalance1
	tokensReceived2 := finalBalance2 - initialBalance2
	require.Greater(t, tokensReceived1, uint64(0), "Should receive tokens on target chain")
	require.Greater(t, tokensReceived2, uint64(0), "Should receive tokens on withdraw chain")

	// Validate pools decreased by withdrawal amounts
	require.Equal(t, initialPool1-expectedWithdraw1, finalPool1, "Pool1 should decrease by withdrawal amount")
	require.Equal(t, initialPool2-expectedWithdraw2, finalPool2, "Pool2 should decrease by withdrawal amount")

	// Validate liquidity points were removed correctly
	require.Equal(t, initialLPPoints1-pointsToRemove1, finalLPPoints1, "LP points should decrease on target chain")
	require.Equal(t, initialLPPoints2-pointsToRemove2, finalLPPoints2, "LP points should decrease on withdraw chain")

	// Validate remaining user points
	userPointsRemaining1, err := pool1Final.GetPointsFor(account.Bytes())
	expectedRemaining1 := userPoints1 - pointsToRemove1
	if expectedRemaining1 > 0 {
		require.NoError(t, err)
		require.Equal(t, expectedRemaining1, userPointsRemaining1,
			"User should have %d points remaining on target chain", expectedRemaining1)
	} else {
		// User should have no points left after full withdrawal
		if err == nil {
			require.Equal(t, uint64(0), userPointsRemaining1, "User should have no points left on target chain")
		}
	}

	userPointsRemaining2, err := pool2Final.GetPointsFor(account.Bytes())
	expectedRemaining2 := userPoints2 - pointsToRemove2
	if expectedRemaining2 > 0 {
		require.NoError(t, err)
		require.Equal(t, expectedRemaining2, userPointsRemaining2,
			"User should have %d points remaining on withdraw chain", expectedRemaining2)
	} else {
		// User should have no points left after full withdrawal
		if err == nil {
			require.Equal(t, uint64(0), userPointsRemaining2, "User should have no points left on withdraw chain")
		}
	}

	// Validate withdrawal amounts match expected pro-rata calculation
	require.Equal(t, expectedWithdraw1, tokensReceived1,
		"Chain1 withdrawal (%d) should match expected (%d)", tokensReceived1, expectedWithdraw1)
	require.Equal(t, expectedWithdraw2, tokensReceived2,
		"Chain2 withdrawal (%d) should match expected (%d)", tokensReceived2, expectedWithdraw2)

	t.Logf("Withdraw: %d%% → %d tokens (chain1), %d tokens (chain2)",
		withdrawPercent, tokensReceived1, tokensReceived2)
	t.Logf("Pool1: %d → %d (-%d), Pool2: %d → %d (-%d)",
		initialPool1, finalPool1, expectedWithdraw1, initialPool2, finalPool2, expectedWithdraw2)
	t.Logf("LP Points1: %d → %d (-%d), LP Points2: %d → %d (-%d)",
		initialLPPoints1, finalLPPoints1, pointsToRemove1, initialLPPoints2, finalLPPoints2, pointsToRemove2)
}

// validateFinalSystemState performs comprehensive final validation
func validateFinalSystemState(t *testing.T, chain1, chain2 *StateMachine, chain1Id, chain2Id uint64, accounts []crypto.AddressI) {
	// Calculate total funds
	expectedPerChain := uint64(len(accounts)*1000 + 10000) // 3*1000 + 10000 = 13000

	total1 := getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend) + getPoolBalance(t, chain1, chain2Id+HoldingPoolAddend)
	total2 := getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend) + getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend)

	for _, account := range accounts {
		total1 += getAccountBalance(t, chain1, account)
		total2 += getAccountBalance(t, chain2, account)
	}

	require.Equal(t, expectedPerChain, total1, "Chain1 fund conservation failed")
	require.Equal(t, expectedPerChain, total2, "Chain2 fund conservation failed")

	// Validate LP points consistency
	pool1, _ := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	pool2, _ := chain2.GetPool(chain1Id + LiquidityPoolAddend)

	if len(pool1.Points) > 0 {
		calculatedTotal := uint64(0)
		for _, point := range pool1.Points {
			calculatedTotal += point.Points
		}
		require.Equal(t, pool1.TotalPoolPoints, calculatedTotal, "Chain1 LP points mismatch")
	}

	if len(pool2.Points) > 0 {
		calculatedTotal := uint64(0)
		for _, point := range pool2.Points {
			calculatedTotal += point.Points
		}
		require.Equal(t, pool2.TotalPoolPoints, calculatedTotal, "Chain2 LP points mismatch")
	}

	t.Logf("Fund conservation: %d total per chain", expectedPerChain)
	t.Logf("LP Points - Chain1: %d, Chain2: %d", pool1.TotalPoolPoints, pool2.TotalPoolPoints)
}

// validateAMMFormula validates the swap output matches Uniswap V2 formula
func validateAMMFormula(t *testing.T, dX, dY, x, y uint64) {
	// Uniswap V2 AMM formula with 0.3% fee:
	// amountInWithFee = dX * 997
	// dY_expected = (amountInWithFee * y) / (x * 1000 + amountInWithFee)

	amountInWithFee := dX * 997
	expectedDY := (amountInWithFee * y) / (x*1000 + amountInWithFee)

	require.Equal(t, expectedDY, dY,
		"AMM output doesn't match Uniswap V2 formula: expected %d, got %d (input: %d, x: %d, y: %d)",
		expectedDY, dY, dX, x, y)

	// Validate constant product formula: (x + dX) * (y - dY) ≥ x * y
	// Account for fee by checking: (x + dX) * (y - dY) ≥ x * y * 997 / 1000
	newProduct := (x + dX) * (y - dY)
	minRequiredProduct := (x * y * 997) / 1000

	require.GreaterOrEqual(t, newProduct, minRequiredProduct,
		"Constant product invariant violated after fees: (%d + %d) * (%d - %d) = %d < %d",
		x, dX, y, dY, newProduct, minRequiredProduct)

	// Validate slippage is reasonable (should be positive due to fees and slippage)
	priceImpact := float64(dX-dY) / float64(dX) * 100
	require.Greater(t, priceImpact, 0.0, "Price impact should be positive (fees + slippage)")
	require.Less(t, priceImpact, 50.0, "Price impact too high: %.2f%%", priceImpact)

	t.Logf("AMM Formula Validated: %d → %d (expected: %d, price impact: %.2f%%)",
		dX, dY, expectedDY, priceImpact)
}

// Helper functions
func getAccountBalance(t *testing.T, sm *StateMachine, account crypto.AddressI) uint64 {
	balance, err := sm.GetAccountBalance(account)
	require.NoError(t, err)
	return balance
}

func getPoolBalance(t *testing.T, sm *StateMachine, poolId uint64) uint64 {
	balance, err := sm.GetPoolBalance(poolId)
	require.NoError(t, err)
	return balance
}

func logFullState(t *testing.T, chain1, chain2 *StateMachine, chain1Id, chain2Id uint64, accounts []crypto.AddressI) {
	t.Logf("Chain1 liquidity pool: %d", getPoolBalance(t, chain1, chain2Id+LiquidityPoolAddend))
	t.Logf("Chain2 liquidity pool: %d", getPoolBalance(t, chain2, chain1Id+LiquidityPoolAddend))
	t.Logf("Chain1 holding pool: %d", getPoolBalance(t, chain1, chain2Id+HoldingPoolAddend))
	t.Logf("Chain2 holding pool: %d", getPoolBalance(t, chain2, chain1Id+HoldingPoolAddend))

	for i, account := range accounts {
		bal1 := getAccountBalance(t, chain1, account)
		bal2 := getAccountBalance(t, chain2, account)
		t.Logf("Account%d - Chain1: %d, Chain2: %d", i+1, bal1, bal2)
	}
}

func clearLocks(t *testing.T, chain1, chain2 StateMachine, chain1Id, chain2Id uint64) {
	require.NoError(t, chain1.Delete(KeyForLockedBatch(chain2Id)))
	require.NoError(t, chain1.Delete(KeyForNextBatch(chain2Id)))
	require.NoError(t, chain2.Delete(KeyForLockedBatch(chain1Id)))
	require.NoError(t, chain2.Delete(KeyForNextBatch(chain1Id)))
}
