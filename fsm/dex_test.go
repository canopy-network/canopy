package fsm

import (
	"bytes"
	"fmt"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
	"strings"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

var deadAddr, _ = crypto.NewAddressFromString(strings.Repeat("dead", 10))

var emptyDexBatch = &lib.DexBatch{
	Committee: 1,
	Orders:    []*lib.DexLimitOrder{},
	Deposits:  []*lib.DexLiquidityDeposit{},
	Withdraws: []*lib.DexLiquidityWithdraw{},
	PoolSize:  0,
	Receipts:  []bool{},
}

func TestHandleDexBatch(t *testing.T) {
	tests := []struct {
		name                string
		detail              string
		rootBuildHeight     uint64
		chainId             uint64
		buyBatch            *lib.DexBatch
		setupState          func(*StateMachine)
		errorContains       string
		expectedLockedBatch *lib.DexBatch // Expected locked batch after processing
	}{
		{
			name:                "nil batch",
			detail:              "test handling nil dex batch",
			chainId:             1,
			buyBatch:            nil,
			expectedLockedBatch: emptyDexBatch, // No batch should be locked
		},
		{
			name:            "no overwrite with chainId != rootChainId",
			detail:          "test no overwrite of buy batch for root chain",
			rootBuildHeight: 1,
			chainId:         2,
			buyBatch:        nil,
			setupState: func(sm *StateMachine) {
				// Setup liquidity pool
				p := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 1000,
				}
				require.NoError(t, sm.SetPool(p))
				mock := &MockRCManager{}
				mock.SetDexBatch(1, 1, 1, &lib.DexBatch{
					Committee: 1,
					PoolSize:  1000,
				})
				sm.RCManager = mock
			},
			expectedLockedBatch: emptyDexBatch,
		},
		{
			name:            "overwrite with chainId == rootChainId",
			detail:          "test handling overwrite of buy batch for root chain",
			rootBuildHeight: 1,
			chainId:         1,
			buyBatch:        nil,
			setupState: func(sm *StateMachine) {
				// Setup liquidity pool
				p := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 1000,
				}
				require.NoError(t, sm.SetPool(p))
				mock := &MockRCManager{}
				mock.SetDexBatch(1, 1, 1, &lib.DexBatch{
					Committee: 1,
					PoolSize:  1000,
				})
				sm.RCManager = mock
			},
			expectedLockedBatch: &lib.DexBatch{
				Committee: 1,
				PoolSize:  1000, // Should be updated to liquidity pool amount
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(&sm)
			}
			err := sm.HandleDexBatch(test.rootBuildHeight, test.chainId, test.buyBatch)
			if err != nil && test.errorContains != "" {
				require.ErrorContains(t, err, test.errorContains)
			}
			// the actual locked batch
			lockedBatch, getErr := sm.GetDexBatch(KeyForLockedBatch(test.chainId))
			require.NoError(t, getErr)
			require.EqualExportedValues(t, test.expectedLockedBatch, lockedBatch)
		})
	}
}

func TestHandleDexBuyBatch(t *testing.T) {
	tests := []struct {
		name                string
		detail              string
		rootBuildHeight     uint64
		chainId             uint64
		buyBatch            *lib.DexBatch
		expectedHoldingPool *Pool
		expectedLiqPool     *Pool
		expectedAccounts    []*Account
		setupState          func(*StateMachine)
		errorContains       string
		expectedLockedBatch *lib.DexBatch // Expected locked batch after processing
	}{
		{
			name:                "nil batch",
			detail:              "test handling nil dex batch",
			chainId:             1,
			buyBatch:            nil,
			expectedLockedBatch: emptyDexBatch, // No batch should be locked
		},
		{
			name:    "no locked batch: liquidity deposit",
			detail:  "test handling a batch with liquidity deposit from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Deposits: []*lib.DexLiquidityDeposit{{
					Address: newTestAddressBytes(t, 1),
					Amount:  100,
				}},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				require.NoError(t, sm.AccountAdd(newTestAddress(t, 1), 1))
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 100,
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}},
				TotalPoolPoints: 141,
			},
		},
		{
			name:    "no locked batch: multi-liquidity deposit",
			detail:  "test handling a batch with liquidity deposit from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 1), Amount: 100},
					{Address: newTestAddressBytes(t, 2), Amount: 100},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 100,
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}},
				TotalPoolPoints: 100 + uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)-1), // rounding
			},
		},
		{
			name:    "no locked batch: full withdraw",
			detail:  "test handling a batch with a full liquidity withdraw from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Withdraws: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					},
				},
				PoolSize: 100, // CNPY
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 200 // JOEY
				liqPool.TotalPoolPoints = 141
				liqPool.Points = []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 142, // 200 - 58
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}},
				TotalPoolPoints: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  58, // 41/141 * 200
			}},
		},
		{
			name:    "no locked batch: partial withdraw",
			detail:  "test handling a batch with a partial liquidity withdraw from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Withdraws: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 1),
						Percent: 25,
					},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 200
				liqPool.TotalPoolPoints = 141
				liqPool.Points = []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 186, // 200 - 14
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  31, // 41-FLOOR(41*.25)
				}},
				TotalPoolPoints: 131,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  14, // FLOOR(41*.25)/141 * 200
			}},
		},
		{
			name:    "no locked batch: multi withdraw",
			detail:  "test handling a batch with a multi liquidity withdraw from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Withdraws: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					},
					{
						Address: newTestAddressBytes(t, 2),
						Percent: 100,
					},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 300
				liqPool.TotalPoolPoints = 172
				liqPool.Points = []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  36,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  36,
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 176, // 300 - 62*2
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}},
				TotalPoolPoints: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  62, // 36/172 * 300
			}, {
				Address: newTestAddressBytes(t, 2),
				Amount:  62, // 36/172 * 300
			}},
		},
		{
			name:    "no locked batch: withdraw and deposit",
			detail:  "test handling a batch with a liquidity withdraw then deposit from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Deposits: []*lib.DexLiquidityDeposit{
					{
						Address: newTestAddressBytes(t, 2),
						Amount:  100, // depositing 100 counter asset
					},
				},
				Withdraws: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					},
				},
				PoolSize: 100, // initial virtual size before deposit
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 200
				liqPool.TotalPoolPoints = 141 // Total LP points
				liqPool.Points = []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100, // burned LP points
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41, // LP points before withdraw
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 142, // pool after withdraw 41/141 ≈ 58 (200-58)
				Points: []*PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100, // burned points remain
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  55, // L=100 (current supply), x=71 (counter asset after withdraw), y=142 (local asset after withdraw), deposit=100
				}},
				TotalPoolPoints: 155, // 100 + 55 = new total after deposit
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  58, // 41/141 * 200 ≈ 58
			}},
		},
		{
			name:    "no locked batch: dex limit order (success)",
			detail:  "test handling a batch with a dex limit order from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 19,
						Address:         newTestAddressBytes(t, 1),
					},
				},
				PoolSize: 100, // initial virtual size before deposit
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// k = 10,000 = 100 * 100
			// dx = 124.925 = 100+(25*.003)
			// dy = 80.1 = 10,000 / 124.925
			// distribute = 19.99 = 100−80.01
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 81,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  19,
			}},
		},
		{
			name:    "no locked batch: dex limit order (fail)",
			detail:  "test handling a batch with a failed dex limit order from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 20,
						Address:         newTestAddressBytes(t, 1),
					},
				},
				PoolSize: 100, // initial virtual size before deposit
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// k = 10,000 = 100 * 100
			// dx = 124.925 = 100+(25*.003)
			// dy = 80.1 = 10,000 / 124.925
			// distribute = 19.99 = 100−80.01
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  0,
			}},
		},
		{
			name:    "no locked batch: multi-dex limit order",
			detail:  "test handling a batch with multiple dex limit orders from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 2),
					},
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 1),
					},
				},
				PoolSize: 100, // initial virtual size before deposit
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 68,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  13,
			}, {
				Address: newTestAddressBytes(t, 2),
				Amount:  19,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(&sm)
			}
			// execute the function call
			err := sm.HandleDexBuyBatch(test.chainId, test.buyBatch)
			if err != nil && test.errorContains != "" {
				require.ErrorContains(t, err, test.errorContains)
			}
			// check expected locked batch
			if test.expectedLockedBatch != nil {
				lockedBatch, getErr := sm.GetDexBatch(KeyForLockedBatch(test.chainId))
				require.NoError(t, getErr)
				require.EqualExportedValues(t, test.expectedLockedBatch, lockedBatch)
			}
			// check expected holding pool
			if test.expectedHoldingPool != nil {
				holdingPool, e := sm.GetPool(test.chainId + HoldingPoolAddend)
				require.NoError(t, e)
				require.EqualExportedValues(t, test.expectedHoldingPool, holdingPool)
			}
			// check expected liquidity pool
			if test.expectedLiqPool != nil {
				liquidityPool, e := sm.GetPool(test.chainId + LiquidityPoolAddend)
				require.NoError(t, e)
				require.EqualExportedValues(t, test.expectedLiqPool, liquidityPool)
			}
			// check expected accounts
			for _, expected := range test.expectedAccounts {
				got, e := sm.GetAccount(crypto.NewAddress(expected.Address))
				require.NoError(t, e)
				require.EqualExportedValues(t, expected, got)
			}
		})
	}
}

func TestHandleDexBatchReceipt(t *testing.T) {
	tests := []struct {
		name         string
		detail       string
		chainId      uint64
		batch        *lib.DexBatch
		setupState   func(StateMachine)
		expectLocked bool
		expectError  bool
		errorString  string
	}{
		{
			name:    "no locked batch",
			detail:  "test when there's no locked batch",
			chainId: 1,
			batch: &lib.DexBatch{
				Committee: 1,
				Receipts:  []bool{true},
			},
			expectLocked: false,
			expectError:  false,
		},
		{
			name:    "receipt mismatch",
			detail:  "test mismatch between locked orders and receipts",
			chainId: 1,
			batch: &lib.DexBatch{
				Committee: 1,
				Receipts:  []bool{true, false}, // 2 receipts
			},
			setupState: func(sm StateMachine) {
				lockedBatch := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:         newTestAddressBytes(t),
							AmountForSale:   100,
							RequestedAmount: 50,
						}, // Only 1 order
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), lockedBatch))
			},
			expectLocked: true,
			expectError:  true,
			errorString:  "mismatch",
		},
		{
			name:    "successful processing",
			detail:  "test successful receipt processing",
			chainId: 1,
			batch: &lib.DexBatch{
				Committee: 1,
				Receipts:  []bool{true, false},
				PoolSize:  1000,
			},
			setupState: func(sm StateMachine) {
				// Setup holding pool
				holdingPool := &Pool{
					Id:     sm.Config.ChainId + HoldingPoolAddend,
					Amount: 300,
				}
				require.NoError(t, sm.SetPool(holdingPool))

				// Setup liquidity pool
				liquidityPool := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 1000,
				}
				require.NoError(t, sm.SetPool(liquidityPool))

				// Setup locked batch
				lockedBatch := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:         newTestAddressBytes(t),
							AmountForSale:   100,
							RequestedAmount: 50,
						},
						{
							Address:         newTestAddressBytes(t, 1),
							AmountForSale:   200,
							RequestedAmount: 60,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), lockedBatch))
			},
			expectLocked: false,
			expectError:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			locked, err := sm.HandleDexBatchReceipt(test.chainId, test.batch)

			require.Equal(t, test.expectLocked, locked)
			require.Equal(t, test.expectError, err != nil, err)
			if err != nil && test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
			}
		})
	}
}

func TestFindUCP(t *testing.T) {
	tests := []struct {
		name            string
		detail          string
		batch           *lib.DexBatch
		setupState      func(StateMachine)
		expectedSuccess []bool
		expectError     bool
		errorString     string
	}{
		{
			name:   "invalid liquidity pool (k=0)",
			detail: "test with zero liquidity causing k=0",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Orders: []*lib.DexLimitOrder{
					{
						Address:         newTestAddressBytes(t),
						AmountForSale:   100,
						RequestedAmount: 50,
					},
				},
			},
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 0,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectError: true,
			errorString: "invalid liquidity pool",
		},
		{
			name:   "successful order matching",
			detail: "test successful uniform clearing price calculation",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Orders: []*lib.DexLimitOrder{
					{
						Address:         newTestAddressBytes(t),
						AmountForSale:   100,
						RequestedAmount: 1, // Low price, should be accepted
					},
					{
						Address:         newTestAddressBytes(t, 1),
						AmountForSale:   50,
						RequestedAmount: 1000, // High price, may not be accepted
					},
				},
				PoolSize: 1000,
			},
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 2000,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectedSuccess: []bool{true, false},
			expectError:     false,
		},
		{
			name:   "empty orders",
			detail: "test with no orders",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Orders:    []*lib.DexLimitOrder{},
				PoolSize:  1000,
			},
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:     sm.Config.ChainId + LiquidityPoolAddend,
					Amount: 2000,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectedSuccess: []bool{},
			expectError:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			success, err := sm.HandleDexBatchOrders(test.batch, nil)

			require.Equal(t, test.expectError, err != nil, err)
			if err != nil && test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
			}
			if !test.expectError {
				require.Equal(t, test.expectedSuccess, success)
			}
		})
	}
}

func TestHandleBatchDeposit(t *testing.T) {
	tests := []struct {
		name              string
		detail            string
		batch             *lib.DexBatch
		counterPoolAmount uint64
		outbound          bool
		setupState        func(StateMachine)
		expectError       bool
	}{
		{
			name:   "no deposits",
			detail: "test with empty deposits",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Deposits:  []*lib.DexLiquidityDeposit{},
				PoolSize:  1000,
			},
			counterPoolAmount: 2000,
			outbound:          true,
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:     lib.CanopyChainId + LiquidityPoolAddend,
					Amount: 1000,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectError: false,
		},
		{
			name:   "outbound deposits",
			detail: "test outbound deposit processing",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Deposits: []*lib.DexLiquidityDeposit{
					{
						Address: newTestAddressBytes(t),
						Amount:  100,
					},
					{
						Address: newTestAddressBytes(t, 1),
						Amount:  200,
					},
				},
				PoolSize: 1000,
			},
			counterPoolAmount: 2000,
			outbound:          true,
			setupState: func(sm StateMachine) {
				// Setup liquidity pool
				pool := &Pool{
					Id:              lib.CanopyChainId + LiquidityPoolAddend,
					Amount:          1000,
					TotalPoolPoints: 1000,
				}
				require.NoError(t, sm.SetPool(pool))

				// Setup holding pool
				holdingPool := &Pool{
					Id:     sm.Config.ChainId + HoldingPoolAddend,
					Amount: 500,
				}
				require.NoError(t, sm.SetPool(holdingPool))
			},
			expectError: false,
		},
		{
			name:   "inbound deposits",
			detail: "test inbound deposit processing",
			batch: &lib.DexBatch{
				Committee: 1,
				Deposits: []*lib.DexLiquidityDeposit{
					{
						Address: newTestAddressBytes(t),
						Amount:  100,
					},
				},
				PoolSize: 1000,
			},
			counterPoolAmount: 2000,
			outbound:          false,
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:              1 + LiquidityPoolAddend,
					Amount:          1000,
					TotalPoolPoints: 1000,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			err := sm.HandleBatchDeposit(test.batch, test.counterPoolAmount, test.outbound)

			require.Equal(t, test.expectError, err != nil, err)
		})
	}
}

func TestHandleBatchWithdraw(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		batch       *lib.DexBatch
		setupState  func(StateMachine)
		expectError bool
	}{
		{
			name:   "no withdrawals",
			detail: "test with empty withdrawals",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Withdraws: []*lib.DexLiquidityWithdraw{},
				PoolSize:  1000,
			},
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:     lib.CanopyChainId + LiquidityPoolAddend,
					Amount: 1000,
				}
				require.NoError(t, sm.SetPool(pool))
			},
			expectError: false,
		},
		{
			name:   "valid withdrawals",
			detail: "test valid withdrawal processing",
			batch: &lib.DexBatch{
				Committee: lib.CanopyChainId,
				Withdraws: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t),
						Percent: 50, // 50% withdrawal
					},
				},
				PoolSize: 1000,
			},
			setupState: func(sm StateMachine) {
				pool := &Pool{
					Id:              lib.CanopyChainId + LiquidityPoolAddend,
					Amount:          1000,
					TotalPoolPoints: 1000,
				}
				// Add points for the address
				require.NoError(t, pool.AddPoints(newTestAddressBytes(t), 100))
				require.NoError(t, sm.SetPool(pool))
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			err := sm.HandleBatchWithdraw(test.batch)

			require.Equal(t, test.expectError, err != nil, err)
		})
	}
}

func TestRotateDexSellBatch(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		receipts    []bool
		chainId     uint64
		setupState  func(StateMachine)
		expectError bool
	}{
		{
			name:     "locked batch exists",
			detail:   "test when locked batch still exists (should exit)",
			receipts: []bool{true, false},
			chainId:  1,
			setupState: func(sm StateMachine) {
				// Create a locked batch that hasn't been processed
				lockedBatch := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t),
							AmountForSale: 100,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), lockedBatch))
			},
			expectError: false, // Function returns early, no error
		},
		{
			name:     "successful rotation",
			detail:   "test successful batch rotation",
			receipts: []bool{true, false},
			chainId:  1,
			setupState: func(sm StateMachine) {
				// Create next batch to rotate
				nextBatch := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t),
							AmountForSale: 100,
						},
						{
							Address:       newTestAddressBytes(t, 1),
							AmountForSale: 200,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForNextBatch(1), nextBatch))

				// Setup liquidity pool
				lPool := &Pool{
					Id:     1 + LiquidityPoolAddend,
					Amount: 1500,
				}
				require.NoError(t, sm.SetPool(lPool))
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			err := sm.RotateDexSellBatch(test.receipts, test.chainId)

			require.Equal(t, test.expectError, err != nil, err)
		})
	}
}

func TestSetGetDexBatch(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		key         []byte
		batch       *lib.DexBatch
		expectError bool
	}{
		{
			name:   "set and get batch",
			detail: "test setting and getting a dex batch",
			key:    KeyForNextBatch(1),
			batch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						Address:         newTestAddressBytes(t),
						AmountForSale:   100,
						RequestedAmount: 50,
					},
				},
				PoolSize: 1000,
			},
			expectError: false,
		},
		{
			name:   "empty batch",
			detail: "test with empty batch",
			key:    KeyForLockedBatch(2),
			batch: &lib.DexBatch{
				Committee: 2,
				Orders:    []*lib.DexLimitOrder{},
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)

			// Test SetDexBatch
			err := sm.SetDexBatch(test.key, test.batch)
			require.Equal(t, test.expectError, err != nil, err)

			if !test.expectError {
				// Test GetDexBatch
				got, err := sm.GetDexBatch(test.key)
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, test.batch.Committee, got.Committee)
				require.Equal(t, len(test.batch.Orders), len(got.Orders))

				// Compare orders
				for i, order := range test.batch.Orders {
					require.True(t, bytes.Equal(order.Address, got.Orders[i].Address))
					require.Equal(t, order.AmountForSale, got.Orders[i].AmountForSale)
					require.Equal(t, order.RequestedAmount, got.Orders[i].RequestedAmount)
				}
			}
		})
	}
}

func TestGetDexBatches(t *testing.T) {
	tests := []struct {
		name        string
		detail      string
		lockedBatch bool
		setupState  func(StateMachine)
		expectedLen int
	}{
		{
			name:        "no batches",
			detail:      "test when no batches exist",
			lockedBatch: true,
			expectedLen: 0,
		},
		{
			name:        "locked batches",
			detail:      "test getting locked batches",
			lockedBatch: true,
			setupState: func(sm StateMachine) {
				batch1 := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t),
							AmountForSale: 100,
						},
					},
				}
				batch2 := &lib.DexBatch{
					Committee: 2,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t, 1),
							AmountForSale: 200,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), batch1))
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(2), batch2))
			},
			expectedLen: 2,
		},
		{
			name:        "next batches",
			detail:      "test getting next batches",
			lockedBatch: false,
			setupState: func(sm StateMachine) {
				batch := &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t),
							AmountForSale: 100,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForNextBatch(1), batch))
			},
			expectedLen: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(sm)
			}

			batches, err := sm.GetDexBatches(test.lockedBatch)
			require.NoError(t, err)
			require.Len(t, batches, test.expectedLen)
		})
	}
}

var _ lib.RCManagerI = new(MockRCManager)

// MockRCManager is a minimal mock implementation of lib.RCManagerI for testing
type MockRCManager struct {
	dexBatches map[string]*lib.DexBatch // key: "rootChainId:height:committee"
}

func (m *MockRCManager) SetDexBatch(rootChainId, height, committee uint64, batch *lib.DexBatch) {
	if m.dexBatches == nil {
		m.dexBatches = make(map[string]*lib.DexBatch)
	}
	key := fmt.Sprintf("%d:%d:%d", rootChainId, height, committee)
	m.dexBatches[key] = batch
}

// RCManagerI interface implementation
func (m *MockRCManager) Publish(chainId uint64, info *lib.RootChainInfo) {}

func (m *MockRCManager) ChainIds() []uint64 { return []uint64{1, 2} }

func (m *MockRCManager) GetHeight(rootChainId uint64) uint64 { return 100 }

func (m *MockRCManager) GetRootChainInfo(rootChainId, chainId uint64) (*lib.RootChainInfo, lib.ErrorI) {
	return &lib.RootChainInfo{}, nil
}

func (m *MockRCManager) GetValidatorSet(rootChainId, height, id uint64) (lib.ValidatorSet, lib.ErrorI) {
	return lib.ValidatorSet{}, nil
}

func (m *MockRCManager) GetLotteryWinner(rootChainId, height, id uint64) (*lib.LotteryWinner, lib.ErrorI) {
	return &lib.LotteryWinner{}, nil
}

func (m *MockRCManager) GetOrders(rootChainId, rootHeight, id uint64) (*lib.OrderBook, lib.ErrorI) {
	return &lib.OrderBook{}, nil
}

func (m *MockRCManager) GetOrder(rootChainId, height uint64, orderId string, chainId uint64) (*lib.SellOrder, lib.ErrorI) {
	return &lib.SellOrder{}, nil
}

func (m *MockRCManager) GetDexBatch(rootChainId, height, committee uint64) (*lib.DexBatch, lib.ErrorI) {
	key := fmt.Sprintf("%d:%d:%d", rootChainId, height, committee)
	if batch, exists := m.dexBatches[key]; exists {
		return batch, nil
	}

	// Return empty batch if not found
	return &lib.DexBatch{
		Committee: committee,
		Orders:    []*lib.DexLimitOrder{},
	}, nil
}

func (m *MockRCManager) IsValidDoubleSigner(rootChainId, height uint64, address string) (*bool, lib.ErrorI) {
	result := false
	return &result, nil
}

func (m *MockRCManager) GetMinimumEvidenceHeight(rootChainId, rootHeight uint64) (h *uint64, e lib.ErrorI) {
	return
}

func (m *MockRCManager) GetCheckpoint(rootChainId, height, id uint64) (blockHash lib.HexBytes, i lib.ErrorI) {
	return
}

func (m *MockRCManager) Transaction(rootChainId uint64, tx lib.TransactionI) (hash *string, err lib.ErrorI) {
	return
}
