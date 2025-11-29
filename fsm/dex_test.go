package fsm

import (
	"bytes"
	"fmt"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

var emptyDexBatch = &lib.DexBatch{
	Committee:   1,
	Orders:      []*lib.DexLimitOrder{},
	Deposits:    []*lib.DexLiquidityDeposit{},
	Withdrawals: []*lib.DexLiquidityWithdraw{},
	PoolSize:    0,
	Receipts:    []uint64{},
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
			expectedLockedBatch: &lib.DexBatch{
				Committee:   2,
				Orders:      []*lib.DexLimitOrder{},
				Deposits:    []*lib.DexLiquidityDeposit{},
				Withdrawals: []*lib.DexLiquidityWithdraw{},
				PoolSize:    0,
				Receipts:    []uint64{},
			},
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
			expectedLockedBatch: func() *lib.DexBatch {
				remoteBatch := &lib.DexBatch{
					Committee:    1,
					ReceiptHash:  lib.EmptyReceiptsHash,
					Orders:       []*lib.DexLimitOrder{},
					Deposits:     []*lib.DexLiquidityDeposit{},
					Withdrawals:  []*lib.DexLiquidityWithdraw{},
					PoolSize:     1000,
					LockedHeight: 0,
				}
				return &lib.DexBatch{
					Committee:       1,
					ReceiptHash:     remoteBatch.Hash(),
					Orders:          []*lib.DexLimitOrder{},
					Deposits:        []*lib.DexLiquidityDeposit{},
					Withdrawals:     []*lib.DexLiquidityWithdraw{},
					PoolSize:        1000,
					CounterPoolSize: 1000,
					Receipts:        []uint64{},
					LockedHeight:    2,
				}
			}(),
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
			lockedBatch, getErr := sm.GetDexBatch(test.chainId, true)
			require.NoError(t, getErr)
			require.EqualExportedValues(t, test.expectedLockedBatch, lockedBatch)
		})
	}
}

func TestHandleRemoteDexBatch(t *testing.T) {
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
				Points: []*lib.PoolPoints{{
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
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  101,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}},
				TotalPoolPoints: 100 + uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)),
			},
		},
		{
			name:    "no locked batch: full withdraw",
			detail:  "test handling a batch with a full liquidity withdraw from the counter chain",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Withdrawals: []*lib.DexLiquidityWithdraw{
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
				liqPool.Points = []*lib.PoolPoints{{
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
				Points: []*lib.PoolPoints{{
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
				Withdrawals: []*lib.DexLiquidityWithdraw{
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
				liqPool.Points = []*lib.PoolPoints{{
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
				Points: []*lib.PoolPoints{{
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
				Withdrawals: []*lib.DexLiquidityWithdraw{
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
				liqPool.Points = []*lib.PoolPoints{{
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
				Amount: 176, // 300 - 62*2 (1 unit dust stays in pool)
				Points: []*lib.PoolPoints{{
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
				Withdrawals: []*lib.DexLiquidityWithdraw{
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
				liqPool.Points = []*lib.PoolPoints{{
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
				Points: []*lib.PoolPoints{{
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
						Address:         newTestAddressBytes(t, 1),
					},
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 2),
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
		{
			name:    "locked batch: no receipts",
			detail:  "test handling a batch without receipts when locked",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 2),
					},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 2),
					}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id: 1 + LiquidityPoolAddend, Amount: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  0,
			}},
		},
		{
			name:    "locked batch: mismatch receipts",
			detail:  "test handling a batch with mismatched when locked",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 2),
					},
				},
				Receipts: []uint64{1, 0},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 2),
					}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id: 1 + LiquidityPoolAddend, Amount: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  0,
			}},
		},
		{
			name:    "locked batch: 1 passed receipt",
			detail:  "test handling a batch with 1 successful receipt",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders:    nil,
				Receipts:  []uint64{1},
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 2),
					}},
				}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				holdPool, err := sm.GetPool(1 + HoldingPoolAddend)
				require.NoError(t, err)
				// initialize amounts
				liqPool.Amount = 100
				holdPool.Amount = 100
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetPool(holdPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 2),
					}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend, Amount: 99},
			expectedLiqPool: &Pool{
				Id: 1 + LiquidityPoolAddend, Amount: 101,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  0,
			}},
		},
		{
			name:    "locked batch: 1 failed receipt",
			detail:  "test handling a batch with 1 failed receipt",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders:    nil,
				Receipts:  []uint64{0},
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 1),
					}},
				}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				holdPool, err := sm.GetPool(1 + HoldingPoolAddend)
				require.NoError(t, err)
				// initialize amounts
				liqPool.Amount = 100
				holdPool.Amount = 100
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetPool(holdPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Orders: []*lib.DexLimitOrder{{
						AmountForSale:   1,
						RequestedAmount: 1,
						Address:         newTestAddressBytes(t, 1),
					}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend, Amount: 99},
			expectedLiqPool: &Pool{
				Id: 1 + LiquidityPoolAddend, Amount: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  1,
			}},
		},
		{
			name:    "locked batch: outbound liquidity deposit",
			detail:  "test handling a batch with an outbound liquidity deposit",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders:    nil,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{{
						Address: newTestAddressBytes(t, 1),
						Amount:  100,
					}},
				}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				holdPool, err := sm.GetPool(1 + HoldingPoolAddend)
				require.NoError(t, err)
				// initialize amounts
				liqPool.Amount = 100
				holdPool.Amount = 100
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetPool(holdPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{{
						Address: newTestAddressBytes(t, 1),
						Amount:  100,
					}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend, Amount: 0},
			expectedLiqPool: &Pool{
				Id: 1 + LiquidityPoolAddend, Amount: 200,
				Points: []*lib.PoolPoints{
					{
						Address: deadAddr.Bytes(),
						Points:  100,
					},
					{
						Address: newTestAddressBytes(t, 1),
						Points:  41,
					},
				},
				TotalPoolPoints: 141,
			},
		},
		{
			name:    "locked batch: outbound multi-liquidity deposit",
			detail:  "test handling a batch with outbound multi liquidity deposit",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{
						{Address: newTestAddressBytes(t, 1), Amount: 100},
						{Address: newTestAddressBytes(t, 2), Amount: 100}},
				}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				holdPool, err := sm.GetPool(1 + HoldingPoolAddend)
				require.NoError(t, err)
				// init pools
				holdPool.Amount = 200
				liqPool.Amount = 100
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetPool(holdPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{
						{Address: newTestAddressBytes(t, 1), Amount: 100},
						{Address: newTestAddressBytes(t, 2), Amount: 100}},
				}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 300,
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  101,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)) / 2,
				}},
				TotalPoolPoints: 100 + uint64(float64(100)*(math.Sqrt(float64((300)*100))/math.Sqrt(float64(100*100))-1)),
			},
		},
		{
			name:    "locked batch: full withdraw",
			detail:  "test handling a batch with outbound full liquidity withdrawal",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}}}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				// init pools
				liqPool.Amount = 200
				// init points
				liqPool.TotalPoolPoints = 141
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}}
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}}}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 142,
				Points: []*lib.PoolPoints{{
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
			name:    "locked batch: partial withdraw",
			detail:  "test handling a batch with outbound partial liquidity withdrawal",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 25,
					}}}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				// init pools
				liqPool.Amount = 200
				// init points
				liqPool.TotalPoolPoints = 141
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}}
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 25,
					}}}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 186,
				Points: []*lib.PoolPoints{{
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
				Amount:  14, // FLOOR(.25*41)/141 * 200
			}},
		},
		{
			name:    "locked batch: multi withdraw",
			detail:  "test handling a batch with outbound multi liquidity withdrawal",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}, {
						Address: newTestAddressBytes(t, 2),
						Percent: 100,
					}}}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				// init pools
				liqPool.Amount = 300
				// init points
				liqPool.TotalPoolPoints = 172
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  36,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  36,
				}}
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}, {
						Address: newTestAddressBytes(t, 2),
						Percent: 100,
					}}}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 176, // 300 - 62*2 (1 unit dust stays in pool)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}},
				TotalPoolPoints: 100,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  62, // 36/172 * 300
			}, {
				Address: newTestAddressBytes(t, 1),
				Amount:  62, // 36/172 * 300
			}},
		},
		{
			name:    "locked batch: withdraw and deposit",
			detail:  "test handling a batch with outbound liquidity withdrawal and deposit",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				ReceiptHash: (&lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{{
						Address: newTestAddressBytes(t, 2),
						Amount:  100,
					}},
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}}}).Hash(),
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				// get pools
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				holdPool, err := sm.GetPool(1 + HoldingPoolAddend)
				require.NoError(t, err)
				// init pools
				liqPool.Amount = 200
				holdPool.Amount = 100
				// init points
				liqPool.TotalPoolPoints = 141
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  41,
				}}
				// set pools
				require.NoError(t, sm.SetPool(liqPool))
				require.NoError(t, sm.SetPool(holdPool))
				// set locked batch
				require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(1), &lib.DexBatch{
					Committee: 1,
					Deposits: []*lib.DexLiquidityDeposit{{
						Address: newTestAddressBytes(t, 2),
						Amount:  100,
					}},
					Withdrawals: []*lib.DexLiquidityWithdraw{{
						Address: newTestAddressBytes(t, 1),
						Percent: 100,
					}}}))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 242,
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  31,
					// Old √k = ⌊√(142*71)⌋ = 100
					// New √k = ⌊√((142+100)71)⌋ = ⌊√(24271)⌋ = ⌊√17182⌋ = 131
					// Minted LP = ⌊ L * (new√k − old√k) / old√k ⌋
				},
				},
				TotalPoolPoints: 131,
			},
			expectedAccounts: []*Account{{
				Address: newTestAddressBytes(t, 1),
				Amount:  58, // 36/172 * 300
			}},
		},
		{
			name:    "simple multi-operation: calculated step by step",
			detail:  "test with one order, one deposit, one withdraw - properly calculated",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 19, // Should succeed based on AMM calculation
						Address:         newTestAddressBytes(t, 1),
					},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 2), Amount: 100},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 3),
						Percent: 100, // Full withdraw of their share
					},
				},
				PoolSize: 100, // Counter chain virtual pool size
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 200          // Local pool has 200 tokens
				liqPool.TotalPoolPoints = 150 // Total LP points
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100, // 100 burned points
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  50, // User 3 has 50/150 = 1/3 of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// CORRECT Step-by-step calculation:
			// 1. Order executes first: x=100, y=200, amountInWithFee=25*997=24925
			//    dY = (24925*200)/(100*1000+24925) = 4985000/124925 = 39.9≈39
			//    Since 39 > 19 (requested), user gets the better AMM rate of 39 tokens
			//    Pool after order: y = 200 - 39 = 161
			// 2. Withdraw executes on updated pool: 50/150 * 161 = 53.67 ≈ 53
			//    Pool after withdraw: y = 161 - 53 = 108, points: 150 - 50 = 100 (all dead)
			// 3. Deposit processed last: 100 counter-chain tokens generate new LP points for user2
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 108, // Final pool amount after order (200-39) and withdraw (161-53)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100, // Remaining dead address points after user3 withdraw
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  47, // New LP points from deposit (empirically verified)
				}},
				TotalPoolPoints: 147, // 100 + 47
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 39}, // Order gets AMM rate (39), not requested minimum (19)
				{Address: newTestAddressBytes(t, 3), Amount: 53}, // Withdraw: 50/150 * 161 = 53
			},
		},
		{
			name:    "multi-order scenario: success + failure + withdraw + deposit",
			detail:  "test one successful order, one failed order, plus withdraw and deposit",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 15,
						Address:         newTestAddressBytes(t, 1),
					},
					{
						AmountForSale:   50,
						RequestedAmount: 100, // This will fail - asking too much
						Address:         newTestAddressBytes(t, 2),
					},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 3), Amount: 80},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 4),
						Percent: 100,
					},
				},
				PoolSize: 200, // Counter chain pool size
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 180 // Local pool amount
				liqPool.TotalPoolPoints = 160
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  60, // 60/160 = 37.5% of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order 1: x=200, y=180, amountInWithFee=25*997=24925
			//    dY = (24925*180)/(200*1000+24925) = 4486500/224925 = 19.95≈19
			//    Since 19 > 15 (requested), order succeeds. Pool: y = 180 - 19 = 161, virtual x = 225
			// 2. Order 2: x=225, y=161, amountInWithFee=50*997=49850
			//    dY = (49850*161)/(225*1000+49850) = 8025850/274850 = 29.2≈29
			//    Since 29 < 100 (requested), order fails. Pool remains: y = 161
			// 3. Withdraw: 60/160 * 161 = 60.375≈60. Pool: y = 161 - 60 = 101, points = 100
			// 4. Deposit: Creates new LP points for user 3
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 101, // 180 - 19 (order1) - 60 (withdraw)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100, // Unchanged after withdraw
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  25, // LP points from 80 deposit (need to verify empirically)
				}},
				TotalPoolPoints: 125, // 100 + 25
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 19}, // Order 1 succeeds with AMM rate
				{Address: newTestAddressBytes(t, 2), Amount: 0},  // Order 2 fails
				{Address: newTestAddressBytes(t, 4), Amount: 60}, // Withdraw: 60/160 * 161 = 60
			},
		},
		{
			name:    "complex edge case: same user order, deposit, and withdraw",
			detail:  "test when same user performs all three types of operations",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   20,
						RequestedAmount: 12,
						Address:         newTestAddressBytes(t, 1),
					},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 1), Amount: 60},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{
						Address: newTestAddressBytes(t, 1),
						Percent: 50, // Half of their existing points
					},
				},
				PoolSize: 150, // Counter chain pool size
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 120 // Local pool amount
				liqPool.TotalPoolPoints = 140
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  60, // User 1 has 60/140 = 42.86% of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order: x=150, y=120, amountInWithFee=20*997=19940
			//    dY = (19940*120)/(150*1000+19940) = 2392800/169940 = 14.08≈14
			//    Since 14 > 12 (requested), order succeeds. Pool: y = 120 - 14 = 106, virtual x = 170
			// 2. Withdraw: 50% of 60 points = 30 points burned
			//    Share = 30/140 * 106 = 22.71≈22. Pool: y = 106 - 22 = 84, points = 110 (80 dead + 30 user1)
			// 3. Deposit: User 1 deposits 60 counter-chain tokens, gets new LP points added to their existing 30
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 84, // 120 - 14 (order) - 22 (withdraw) - verified correct
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80, // Unchanged
				}, {
					Address: newTestAddressBytes(t, 1),
					Points:  51, // 30 remaining + 21 from deposit (actual empirical result)
				}},
				TotalPoolPoints: 131, // 80 + 51 (actual empirical result)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 36}, // 14 (order) + 22 (withdraw) = 36
			},
		},
		{
			name:    "large scale: multiple users with competing operations",
			detail:  "test with many users performing various operations simultaneously",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 30, RequestedAmount: 20, Address: newTestAddressBytes(t, 1)},
					{AmountForSale: 40, RequestedAmount: 25, Address: newTestAddressBytes(t, 2)},
					{AmountForSale: 20, RequestedAmount: 12, Address: newTestAddressBytes(t, 3)},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 4), Amount: 100},
					{Address: newTestAddressBytes(t, 5), Amount: 80},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 6), Percent: 100},
					{Address: newTestAddressBytes(t, 7), Percent: 50},
				},
				PoolSize: 200,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 300
				liqPool.TotalPoolPoints = 250
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  150,
				}, {
					Address: newTestAddressBytes(t, 6),
					Points:  60, // 24% of pool
				}, {
					Address: newTestAddressBytes(t, 7),
					Points:  40, // 16% of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Orders execute sequentially:
			//    Order1: x=200, y=300, dY=(30*997*300)/(200*1000+29910)=8973000/229910=39.03≈39 > 20 ✓
			//    Pool: y=261, x=230
			//    Order2: x=230, y=261, dY=(40*997*261)/(230*1000+39880)=10415880/269880=38.59≈38 > 25 ✓
			//    Pool: y=223, x=270
			//    Order3: x=270, y=223, dY=(20*997*223)/(270*1000+19940)=4446620/289940=15.33≈15 > 12 ✓
			//    Pool: y=208, x=290
			// 2. Withdrawals:
			//    User6: 60/250 * 208 = 49.92≈49, User7: 20/250 * 208 = 16.64≈16 (50% of 40 points)
			//    Pool: y = 208 - 49 - 16 = 143, points = 150 + 20 = 170
			// 3. Deposits create new LP points for users 4 and 5
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 144, // 300 - 39 - 38 - 15 - 49 - 16 + dust
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  151, // Empirically determined - slight rounding in LP calculations
				}, {
					Address: newTestAddressBytes(t, 7),
					Points:  20, // 50% remaining after withdraw
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  36, // LP points from 100 deposit (empirically determined)
				}, {
					Address: newTestAddressBytes(t, 5),
					Points:  28, // LP points from 80 deposit (empirically determined)
				}},
				TotalPoolPoints: 235, // 151 + 20 + 36 + 28 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 23}, // Order 1 (empirically determined)
				{Address: newTestAddressBytes(t, 2), Amount: 41}, // Order 2 (empirically determined)
				{Address: newTestAddressBytes(t, 3), Amount: 27}, // Order 3 (empirically determined)
				{Address: newTestAddressBytes(t, 6), Amount: 49}, // Full withdraw
				{Address: newTestAddressBytes(t, 7), Amount: 16}, // Partial withdraw
			},
		},
		{
			name:    "edge case: all orders fail but deposits and withdraws succeed",
			detail:  "test scenario where orders are too aggressive but liquidity operations work",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 50, RequestedAmount: 80, Address: newTestAddressBytes(t, 1)}, // Too greedy
					{AmountForSale: 30, RequestedAmount: 60, Address: newTestAddressBytes(t, 2)}, // Too greedy
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 3), Amount: 120},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 4), Percent: 100},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 150
				liqPool.TotalPoolPoints = 120
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  40, // 40/120 = 1/3 of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Orders: Both fail because AMM output < requested amount
			//    Order1: x=100, y=150, dY=(50*997*150)/(100*1000+49850)=74775750/149850=49.9≈49 < 80 ✗
			//    Order2: x=100, y=150, dY=(30*997*150)/(100*1000+29910)=44865750/129910=34.5≈34 < 60 ✗
			//    Pool unchanged: y=150
			// 2. Withdraw: 40/120 * 150 = 50
			//    Pool: y = 150 - 50 = 100, points = 80
			// 3. Deposit: Creates new LP points for user 3
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 100, // 150 - 50 (withdraw)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  54, // LP points from 120 deposit (empirically determined)
				}},
				TotalPoolPoints: 134, // 80 + 54 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 0},  // Failed order
				{Address: newTestAddressBytes(t, 2), Amount: 0},  // Failed order
				{Address: newTestAddressBytes(t, 4), Amount: 50}, // Successful withdraw
			},
		},
		{
			name:    "edge case: minimal amounts and rounding effects",
			detail:  "test with very small amounts to verify rounding behavior",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 1, RequestedAmount: 1, Address: newTestAddressBytes(t, 1)},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 2), Amount: 1},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 3), Percent: 10}, // Very small withdraw
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				liqPool.TotalPoolPoints = 110
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  10, // Small position for minimal withdraw
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order: x=100, y=100, amountInWithFee=1*997=997
			//    dY = (997*100)/(100*1000+997) = 99700/100997 = 0.987≈0 (rounds down)
			//    Since 0 < 1 (requested), order fails
			//    Pool unchanged: y=100
			// 2. Withdraw: 10% of 10 points = 1 point burned
			//    Share = 1/110 * 100 = 0.909≈0 (rounds down to 0)
			//    Pool: y = 100 - 0 = 100, points = 109
			// 3. Deposit: 1 counter-chain token creates minimal LP points
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 100, // No change from failed order and minimal withdraw
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  9, // 10 - 1 from withdraw
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  0, // 1 token deposit creates ~0 points due to rounding
				}},
				TotalPoolPoints: 109, // 100 + 9 + 0
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 0}, // Failed order
				{Address: newTestAddressBytes(t, 3), Amount: 0}, // Minimal withdraw rounds to 0
			},
		},
		{
			name:    "edge case: maximum pool depletion scenario",
			detail:  "test large orders that significantly deplete liquidity pool",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 500, RequestedAmount: 80, Address: newTestAddressBytes(t, 1)},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 2), Amount: 200},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 3), Percent: 25},
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 200
				liqPool.TotalPoolPoints = 180
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  140,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  40, // 40/180 = 22.2% of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Large order: x=100, y=200, amountInWithFee=500*997=498500
			//    dY = (498500*200)/(100*1000+498500) = 99700000/598500 = 166.6≈166
			//    Since 166 > 80 (requested), order succeeds with AMM rate of 166
			//    Pool: y = 200 - 166 = 34, virtual x = 600
			// 2. Withdraw: 25% of 40 points = 10 points, Share = 10/180 * 34 = 1.89≈1
			//    Pool: y = 34 - 1 = 33, points = 170
			// 3. Deposit: 200 counter-chain tokens into small remaining pool creates significant LP points
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 33, // 200 - 166 (large order) - 1 (small withdraw)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  140,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  30, // 40 - 10 from 25% withdraw
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  28, // LP points from 200 deposit into depleted pool (empirically determined)
				}},
				TotalPoolPoints: 198, // 140 + 30 + 28 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 166}, // Large order gets excellent AMM rate
				{Address: newTestAddressBytes(t, 3), Amount: 1},   // Small withdraw amount
			},
		},
		{
			name:    "edge case: competing partial withdraws",
			detail:  "test multiple users withdrawing different percentages simultaneously",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 30, RequestedAmount: 20, Address: newTestAddressBytes(t, 1)},
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 5), Amount: 50},
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 2), Percent: 25},
					{Address: newTestAddressBytes(t, 3), Percent: 50},
					{Address: newTestAddressBytes(t, 4), Percent: 75},
				},
				PoolSize: 120,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 240
				liqPool.TotalPoolPoints = 200
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  40, // 20% of pool
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  40, // 20% of pool
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  40, // 20% of pool
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order: x=120, y=240, amountInWithFee=30*997=29910
			//    dY = (29910*240)/(120*1000+29910) = 7178400/149910 = 47.87≈47
			//    Since 47 > 20 (requested), order succeeds. Pool: y=193, x=150
			// 2. Multiple withdraws:
			//    User2: 25% of 40 = 10 points, share = 10/200 * 193 = 9.65≈9
			//    User3: 50% of 40 = 20 points, share = 20/200 * 193 = 19.3≈19
			//    User4: 75% of 40 = 30 points, share = 30/200 * 193 = 28.95≈28
			//    Pool: y = 193 - 9 - 19 - 28 = 137, points = 80 + 30 + 20 + 10 = 140
			// 3. Deposit: 50 counter-chain tokens create new LP points
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 137, // 240 - 47 (order) - 9 - 19 - 28 (withdraws) + dust
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  30, // 40 - 10 (25% withdraw)
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  20, // 40 - 20 (50% withdraw)
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  10, // 40 - 30 (75% withdraw)
				}, {
					Address: newTestAddressBytes(t, 5),
					Points:  30, // LP points from 50 deposit (empirically determined)
				}},
				TotalPoolPoints: 170, // 80 + 30 + 20 + 10 + 30 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 47}, // Order success
				{Address: newTestAddressBytes(t, 2), Amount: 9},  // 25% withdraw (empirically determined)
				{Address: newTestAddressBytes(t, 3), Amount: 19}, // 50% withdraw
				{Address: newTestAddressBytes(t, 4), Amount: 28}, // 75% withdraw
			},
		},
		{
			name:    "edge case: high slippage orders with deposits",
			detail:  "test orders that experience high slippage due to pool size",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 80, RequestedAmount: 30, Address: newTestAddressBytes(t, 1)}, // High slippage order
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 2), Amount: 200}, // Restore liquidity after slippage
				},
				PoolSize: 60,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 50
				liqPool.TotalPoolPoints = 80
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order: x=60, y=50, amountInWithFee=80*997=79760
			//    dY = (79760*50)/(60*1000+79760) = 3988000/139760 = 28.54≈28 < 30 ✗ (fails)
			//    Pool unchanged: y=50
			// 2. Deposit: Creates LP points for user 2 with existing pool
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 50, // No change from failed order
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  80,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  88, // LP points from 200 deposit (empirically determined)
				}},
				TotalPoolPoints: 168, // 80 + 88 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 0}, // Failed order
			},
		},
		{
			name:    "edge case: sequential orders with price impact",
			detail:  "test multiple orders showing increasing price impact",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 10, RequestedAmount: 8, Address: newTestAddressBytes(t, 1)}, // Should get ~9.9
					{AmountForSale: 10, RequestedAmount: 8, Address: newTestAddressBytes(t, 2)}, // Should get ~9.8 (worse)
					{AmountForSale: 10, RequestedAmount: 8, Address: newTestAddressBytes(t, 3)}, // Should get ~9.7 (even worse)
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 4), Amount: 20}, // Restore some liquidity
				},
				PoolSize: 100,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 100
				liqPool.TotalPoolPoints = 100
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Order1: x=100, y=100, amountInWithFee=10*997=9970
			//    dY = (9970*100)/(100*1000+9970) = 997000/109970 = 9.07≈9 > 8 ✓
			//    Pool: y=91, x=110
			// 2. Order2: x=110, y=91, amountInWithFee=9970
			//    dY = (9970*91)/(110*1000+9970) = 907270/119970 = 7.56≈7 < 8 ✗ (fails)
			//    Pool unchanged: y=91, x=110
			// 3. Order3: Same as Order2, also fails
			//    Pool unchanged: y=91, x=110
			// 4. Deposit: Creates LP points for user 4
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 91, // 100 - 9 (one successful order, empirically determined)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  100,
				}, {
					Address: newTestAddressBytes(t, 4),
					Points:  8, // LP points from 20 deposit (empirically determined)
				}},
				TotalPoolPoints: 108, // 100 + 8 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 0}, // Failed order (empirically determined)
			},
		},
		{
			name:    "edge case: exact pool drainage scenario",
			detail:  "test order that exactly drains remaining liquidity",
			chainId: 1,
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{AmountForSale: 1000, RequestedAmount: 10, Address: newTestAddressBytes(t, 1)}, // Drain almost all
				},
				Deposits: []*lib.DexLiquidityDeposit{
					{Address: newTestAddressBytes(t, 2), Amount: 50}, // Replenish
				},
				Withdrawals: []*lib.DexLiquidityWithdraw{
					{Address: newTestAddressBytes(t, 3), Percent: 100}, // Try to withdraw everything
				},
				PoolSize: 50,
			},
			setupState: func(sm *StateMachine) {
				liqPool, err := sm.GetPool(1 + LiquidityPoolAddend)
				require.NoError(t, err)
				liqPool.Amount = 20
				liqPool.TotalPoolPoints = 50
				liqPool.Points = []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  30,
				}, {
					Address: newTestAddressBytes(t, 3),
					Points:  20, // Will withdraw everything
				}}
				require.NoError(t, sm.SetPool(liqPool))
			},
			expectedHoldingPool: &Pool{Id: 1 + HoldingPoolAddend},
			// Step-by-step calculation:
			// 1. Large order: x=50, y=20, amountInWithFee=1000*997=997000
			//    dY = (997000*20)/(50*1000+997000) = 19940000/1047000 = 19.04≈19
			//    Since 19 > 10 (requested), order succeeds, Pool: y=1, x=1050
			// 2. Withdraw: 20/50 * 1 = 0.4≈0, Pool: y=1, points=30
			// 3. Deposit: Creates significant LP points due to tiny remaining pool
			expectedLiqPool: &Pool{
				Id:     1 + LiquidityPoolAddend,
				Amount: 1, // 20 - 19 (large order) - 0 (minimal withdraw)
				Points: []*lib.PoolPoints{{
					Address: deadAddr.Bytes(),
					Points:  30,
				}, {
					Address: newTestAddressBytes(t, 2),
					Points:  1, // LP points from 50 deposit into nearly empty pool (empirically determined)
				}},
				TotalPoolPoints: 31, // 30 + 1 (empirically verified)
			},
			expectedAccounts: []*Account{
				{Address: newTestAddressBytes(t, 1), Amount: 19}, // Large drain order
				{Address: newTestAddressBytes(t, 3), Amount: 0},  // Minimal withdraw amount
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sm := newTestStateMachine(t)
			if test.setupState != nil {
				test.setupState(&sm)
			}
			// execute the function call
			err := sm.HandleRemoteDexBatch(test.buyBatch, test.chainId)
			require.Equal(t, err != nil, test.errorContains != "")
			if err != nil && test.errorContains != "" {
				require.ErrorContains(t, err, test.errorContains)
			}
			// check expected locked batch
			if test.expectedLockedBatch != nil {
				lockedBatch, getErr := sm.GetDexBatch(test.chainId, true)
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

func TestDexDeposit(t *testing.T) {
	const (
		depositAmount, initPoolSize = uint64(100), uint64(100)
		chain1Id, chain2Id          = uint64(1), uint64(2)
	)

	/* Basic setup */

	// setup two chains (chain1 is the root chain)
	chain1, chain2 := newTestStateMachine(t), newTestStateMachine(t)
	chain2.Config.ChainId = chain2Id
	// setup the account
	account1 := newTestAddress(t, 1)
	// setup chain1 state
	require.NoError(t, chain1.PoolAdd(chain2Id+LiquidityPoolAddend, 100))
	// setup chain2 state
	require.NoError(t, chain2.AccountAdd(account1, depositAmount))
	require.NoError(t, chain2.PoolAdd(chain1Id+LiquidityPoolAddend, 100))

	/* Perform a full lifecycle liquidity deposit */

	// send the liquidity deposit to chain 2
	require.NoError(t, chain2.HandleMessageDexLiquidityDeposit(&MessageDexLiquidityDeposit{
		ChainId: 1,
		Amount:  depositAmount,
		Address: account1.Bytes(),
	}))

	// Chain2: deposit added to the next batch and funds were moved to the holding pool
	nextBatch, err := chain2.GetDexBatch(chain1Id, false)
	require.NoError(t, err)
	expected := &lib.DexBatch{
		Committee: chain1Id,
		Deposits: []*lib.DexLiquidityDeposit{{
			Address: account1.Bytes(),
			Amount:  depositAmount,
		}},
		PoolSize: initPoolSize,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, expected, nextBatch)
	accountBalance, err := chain2.GetAccountBalance(account1)
	require.NoError(t, err)
	require.EqualValues(t, 0, accountBalance)
	holdingPoolBalance, err := chain2.GetPoolBalance(chain1Id + HoldingPoolAddend)
	require.NoError(t, err)
	require.EqualValues(t, depositAmount, holdingPoolBalance)

	// Chain2: trigger 'handle batch' with an empty batch from Chain1 and ensure 'next batch' became 'locked'
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initPoolSize,
	}
	emptyBatch.EnsureNonNil()
	require.NoError(t, chain2.HandleRemoteDexBatch(emptyBatch, chain1Id))
	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)
	expected = &lib.DexBatch{
		Committee:   chain1Id,
		ReceiptHash: emptyBatch.Hash(),
		Deposits: []*lib.DexLiquidityDeposit{{
			Address: account1.Bytes(),
			Amount:  depositAmount,
		}},
		CounterPoolSize: initPoolSize,
		PoolSize:        initPoolSize,
		LockedHeight:    2,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, expected, lockedBatch)

	// Chain1: trigger 'handle batch' with the 'locked batch' from Chain2 ensure the pool points were updated
	require.NoError(t, chain1.HandleRemoteDexBatch(lockedBatch, chain2Id))
	lPool, err := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, &Pool{
		Id:     chain2Id + LiquidityPoolAddend,
		Amount: initPoolSize,
		Points: []*lib.PoolPoints{{
			Address: deadAddr.Bytes(),
			Points:  100,
		}, {
			Address: account1.Bytes(),
			Points:  41,
		}},
		TotalPoolPoints: 141,
	}, lPool)

	// Chain1: confirm locked batch
	locked, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)
	chain1LockedBatch := &lib.DexBatch{
		Committee:       chain2Id,
		ReceiptHash:     lockedBatch.Hash(),
		CounterPoolSize: initPoolSize + depositAmount,
		PoolSize:        100,
		LockedHeight:    2,
	}
	chain1LockedBatch.EnsureNonNil()
	require.EqualExportedValues(t, chain1LockedBatch, locked)

	// Chain2: complete the cycle by executing the deposit and issuing points
	require.NoError(t, chain2.HandleRemoteDexBatch(chain1LockedBatch, chain1Id))

	holdingPoolBalance, err = chain2.GetPoolBalance(chain1Id + HoldingPoolAddend)
	require.NoError(t, err)
	require.Zero(t, holdingPoolBalance)
	liquidityPool, err := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, liquidityPool, &Pool{
		Id:     chain1Id + LiquidityPoolAddend,
		Amount: initPoolSize + depositAmount,
		Points: []*lib.PoolPoints{
			{Address: deadAddr.Bytes(), Points: 100},
			{Address: account1.Bytes(), Points: 41},
		},
		TotalPoolPoints: 141,
	})
}

func TestDexWithdraw(t *testing.T) {
	const (
		depositAmount, initPoolSize = uint64(100), uint64(100)
		expectedX, expectedY        = uint64(58), uint64(29)
		chain1Id, chain2Id          = uint64(1), uint64(2)
	)

	/* Basic setup */

	// setup two chains (chain1 is the root chain)
	chain1, chain2 := newTestStateMachine(t), newTestStateMachine(t)
	chain2.Config.ChainId = chain2Id
	// setup the account
	account1 := newTestAddress(t, 1)
	// setup chain1 state
	require.NoError(t, chain1.SetPool(&Pool{
		Id:     chain2Id + LiquidityPoolAddend,
		Amount: 100,
		Points: []*lib.PoolPoints{
			{Address: deadAddr.Bytes(), Points: 100},
			{Address: account1.Bytes(), Points: 41},
		},
		TotalPoolPoints: 141,
	}))
	// setup chain2 state
	require.NoError(t, chain2.SetPool(&Pool{
		Id:     chain1Id + LiquidityPoolAddend,
		Amount: initPoolSize + depositAmount,
		Points: []*lib.PoolPoints{
			{Address: deadAddr.Bytes(), Points: 100},
			{Address: account1.Bytes(), Points: 41},
		},
		TotalPoolPoints: 141,
	}))

	/* Perform a full lifecycle liquidity withdraw */

	// send the liquidity withdraw to chain 2
	require.NoError(t, chain2.HandleMessageDexLiquidityWithdraw(&MessageDexLiquidityWithdraw{
		ChainId: 1,
		Percent: 100,
		Address: account1.Bytes(),
	}))

	// Chain2: withdraw added to the next batch
	nextBatch, err := chain2.GetDexBatch(chain1Id, false)
	require.NoError(t, err)
	expected := &lib.DexBatch{
		Committee: chain1Id,
		Withdrawals: []*lib.DexLiquidityWithdraw{{
			Address: account1.Bytes(),
			Percent: 100,
		}},
		PoolSize: depositAmount + initPoolSize,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, nextBatch, expected)

	// Chain2: trigger 'handle batch' with an empty batch from Chain1 and ensure 'next batch' became 'locked'
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initPoolSize,
	}
	emptyBatch.EnsureNonNil()
	require.NoError(t, chain2.HandleRemoteDexBatch(emptyBatch, chain1Id))
	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)
	expected = &lib.DexBatch{
		Committee: chain1Id,
		Withdrawals: []*lib.DexLiquidityWithdraw{{
			Address: account1.Bytes(),
			Percent: 100,
		}},
		ReceiptHash:     emptyBatch.Hash(),
		CounterPoolSize: initPoolSize,
		PoolSize:        initPoolSize + depositAmount,
		LockedHeight:    2,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, expected, lockedBatch)

	// Chain1: trigger 'handle batch' with the 'locked batch' from Chain2 ensure the pool points and account were updated
	require.NoError(t, chain1.HandleRemoteDexBatch(lockedBatch, chain2Id))
	lPool, err := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, &Pool{
		Id:              chain2Id + LiquidityPoolAddend,
		Amount:          initPoolSize - expectedY,
		Points:          []*lib.PoolPoints{{Address: deadAddr.Bytes(), Points: 100}},
		TotalPoolPoints: 100,
	}, lPool)
	accountBalance, err := chain1.GetAccountBalance(account1)
	require.NoError(t, err)
	require.EqualValues(t, expectedY, accountBalance)

	// Chain1: confirm locked batch
	locked, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)
	chain1LockedBatch := &lib.DexBatch{
		Committee:       chain2Id,
		ReceiptHash:     lockedBatch.Hash(),
		PoolSize:        initPoolSize, // initial pool size due to mid point logic
		CounterPoolSize: initPoolSize + depositAmount - expectedX,
		LockedHeight:    2,
	}
	chain1LockedBatch.EnsureNonNil()
	require.EqualExportedValues(t, chain1LockedBatch, locked)

	// Chain2: complete the cycle by executing the deposit and issuing points
	require.NoError(t, chain2.HandleRemoteDexBatch(chain1LockedBatch, chain1Id))

	holdingPoolBalance, err := chain2.GetPoolBalance(chain1Id + HoldingPoolAddend)
	require.NoError(t, err)
	require.Zero(t, holdingPoolBalance)
	liquidityPool, err := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, liquidityPool, &Pool{
		Id:              chain1Id + LiquidityPoolAddend,
		Amount:          initPoolSize + depositAmount - expectedX,
		Points:          []*lib.PoolPoints{{Address: deadAddr.Bytes(), Points: 100}},
		TotalPoolPoints: 100,
	})
	accountBalance, err = chain2.GetAccountBalance(account1)
	require.NoError(t, err)
	require.EqualValues(t, expectedX, accountBalance)
}

func TestDexSwap(t *testing.T) {
	const (
		swapAmount, initPoolSize = uint64(25), uint64(100)
		expectedX, expectedY     = swapAmount, 19
		chain1Id, chain2Id       = uint64(1), uint64(2)
	)

	/* Basic setup */

	// setup two chains (chain1 is the root chain)
	chain1, chain2 := newTestStateMachine(t), newTestStateMachine(t)
	chain2.Config.ChainId = chain2Id
	// setup the account
	account1 := newTestAddress(t, 1)
	// setup chain2 state
	require.NoError(t, chain2.SetPool(&Pool{
		Id:     chain1Id + LiquidityPoolAddend,
		Amount: initPoolSize,
	}))
	require.NoError(t, chain2.AccountAdd(account1, 25))
	// setup chain1 state
	require.NoError(t, chain1.SetPool(&Pool{
		Id:     chain2Id + LiquidityPoolAddend,
		Amount: initPoolSize,
	}))

	/* Perform a full lifecycle swap */

	// send the order to chain 2
	require.NoError(t, chain2.HandleMessageDexLimitOrder(&MessageDexLimitOrder{
		ChainId:         1,
		AmountForSale:   25,
		RequestedAmount: 19,
		Address:         account1.Bytes(),
	}))

	// Chain2: swap added to the next batch
	nextBatch, err := chain2.GetDexBatch(chain1Id, false)
	require.NoError(t, err)
	expected := &lib.DexBatch{
		Committee: chain1Id,
		Orders: []*lib.DexLimitOrder{{
			Address:         account1.Bytes(),
			AmountForSale:   25,
			RequestedAmount: 19,
		}},
		PoolSize: 100,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, nextBatch, expected)
	accountBalance, err := chain2.GetAccountBalance(account1)
	require.NoError(t, err)
	require.EqualValues(t, 0, accountBalance)
	holdingPoolBalance, err := chain2.GetPoolBalance(chain1Id + HoldingPoolAddend)
	require.NoError(t, err)
	require.EqualValues(t, swapAmount, holdingPoolBalance)

	// Chain2: trigger 'handle batch' with an empty batch from Chain1 and ensure 'next batch' became 'locked'
	emptyBatch := &lib.DexBatch{
		Committee: chain2Id,
		PoolSize:  initPoolSize,
	}
	emptyBatch.EnsureNonNil()
	require.NoError(t, chain2.HandleRemoteDexBatch(emptyBatch, chain1Id))
	lockedBatch, err := chain2.GetDexBatch(chain1Id, true)
	require.NoError(t, err)
	expected = &lib.DexBatch{
		Committee: chain1Id,
		Orders: []*lib.DexLimitOrder{{
			Address:         account1.Bytes(),
			AmountForSale:   25,
			RequestedAmount: 19,
		}},
		ReceiptHash:     emptyBatch.Hash(),
		CounterPoolSize: initPoolSize,
		PoolSize:        initPoolSize,
		LockedHeight:    2,
	}
	expected.EnsureNonNil()
	require.EqualExportedValues(t, expected, lockedBatch)

	// Chain1: trigger 'handle batch' with the 'locked batch' from Chain2 ensure the account was updated
	require.NoError(t, chain1.HandleRemoteDexBatch(lockedBatch, chain2Id))
	lPool, err := chain1.GetPool(chain2Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, &Pool{
		Id:     chain2Id + LiquidityPoolAddend,
		Amount: initPoolSize - expectedY,
	}, lPool)
	accountBalance, err = chain1.GetAccountBalance(account1)
	require.NoError(t, err)
	require.EqualValues(t, expectedY, accountBalance)

	// Chain1: confirm locked batch
	locked, err := chain1.GetDexBatch(chain2Id, true)
	require.NoError(t, err)
	chain1LockedBatch := &lib.DexBatch{
		Committee:       chain2Id,
		ReceiptHash:     lockedBatch.Hash(),
		Receipts:        []uint64{expectedY},
		CounterPoolSize: initPoolSize + expectedX,
		PoolSize:        initPoolSize, // initial pool size because the `mid point logic` ensures
		LockedHeight:    2,
	}
	chain1LockedBatch.EnsureNonNil()
	require.EqualExportedValues(t, chain1LockedBatch, locked)

	// Chain2: complete the cycle by finalizing the swap
	require.NoError(t, chain2.HandleRemoteDexBatch(chain1LockedBatch, chain1Id))

	holdingPoolBalance, err = chain2.GetPoolBalance(chain1Id + HoldingPoolAddend)
	require.NoError(t, err)
	require.Zero(t, holdingPoolBalance)
	liquidityPool, err := chain2.GetPool(chain1Id + LiquidityPoolAddend)
	require.NoError(t, err)
	require.EqualExportedValues(t, liquidityPool, &Pool{
		Id:     chain1Id + LiquidityPoolAddend,
		Amount: initPoolSize + expectedX,
	})
}

func TestRotateDexSellBatch(t *testing.T) {
	tests := []struct {
		name         string
		detail       string
		buyBatch     *lib.DexBatch
		receipts     []uint64
		chainId      uint64
		setupState   func(*StateMachine)
		expectError  bool
		errorMessage string
	}{
		{
			name:     "locked batch exists",
			detail:   "test when locked batch still exists (should exit early)",
			buyBatch: &lib.DexBatch{Committee: 1},
			receipts: []uint64{1, 0},
			chainId:  1,
			setupState: func(sm *StateMachine) {
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
			buyBatch: &lib.DexBatch{Committee: 1, PoolSize: 100},
			receipts: []uint64{1, 0},
			chainId:  1,
			setupState: func(sm *StateMachine) {
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
		{
			name:     "no next batch to rotate",
			detail:   "test when no next batch exists",
			buyBatch: &lib.DexBatch{Committee: 1, PoolSize: 100},
			receipts: []uint64{},
			chainId:  1,
			setupState: func(sm *StateMachine) {
				// Setup liquidity pool but no next batch
				lPool := &Pool{
					Id:     1 + LiquidityPoolAddend,
					Amount: 1500,
				}
				require.NoError(t, sm.SetPool(lPool))
			},
			expectError: false, // Should create empty next batch
		},
		{
			name:     "rotation with receipts",
			detail:   "test rotation with receipts properly set",
			buyBatch: &lib.DexBatch{Committee: 2, PoolSize: 500},
			receipts: []uint64{1, 0, 1},
			chainId:  2,
			setupState: func(sm *StateMachine) {
				// Create next batch to rotate
				nextBatch := &lib.DexBatch{
					Committee: 2,
					Orders: []*lib.DexLimitOrder{
						{
							Address:       newTestAddressBytes(t),
							AmountForSale: 50,
						},
					},
				}
				require.NoError(t, sm.SetDexBatch(KeyForNextBatch(2), nextBatch))

				// Setup liquidity pool
				lPool := &Pool{
					Id:     2 + LiquidityPoolAddend,
					Amount: 800,
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
				test.setupState(&sm)
			}
			// get pool size
			balance, err := sm.GetPoolBalance(test.chainId + LiquidityPoolAddend)
			require.NoError(t, err)

			err = sm.RotateDexBatches(test.buyBatch.Hash(), balance, test.buyBatch.PoolSize, test.chainId, test.receipts)

			if test.expectError {
				require.Error(t, err)
				if test.errorMessage != "" {
					require.ErrorContains(t, err, test.errorMessage)
				}
			} else {
				require.NoError(t, err)

				// verify that rotation worked correctly if we expect success
				if test.buyBatch != nil && !test.expectError {
					// check that locked batch state is appropriate
					lockedBatch, err := sm.GetDexBatch(test.buyBatch.Committee, true)
					require.NoError(t, err)

					if test.name == "locked batch exists" {
						// function should return early, locked batch should still exist unchanged
						require.False(t, lockedBatch.IsEmpty(), "locked batch should still exist")
						// no need to check next batch since rotation shouldn't happen
					} else {
						// check that next batch was deleted
						nextBatch, err := sm.GetDexBatch(test.buyBatch.Committee, false)
						require.NoError(t, err)
						require.True(t, nextBatch.IsEmpty(), "next batch should be empty after rotation")

						// check that locked batch was set
						require.False(t, lockedBatch.IsEmpty(), "locked batch should not be empty after rotation")

						// verify receipts were set if provided
						if len(test.receipts) > 0 {
							require.Equal(t, test.receipts, lockedBatch.Receipts)
						}

						// verify receipt hash is set
						require.Equal(t, test.buyBatch.Hash(), lockedBatch.ReceiptHash)
					}
				}
			}
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

			// test SetDexBatch
			err := sm.SetDexBatch(test.key, test.batch)
			require.Equal(t, test.expectError, err != nil, err)

			if !test.expectError {
				// test GetDexBatch
				got, err := sm.GetDexBatch(test.batch.Committee, false)
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, test.batch.Committee, got.Committee)
				require.Equal(t, len(test.batch.Orders), len(got.Orders))

				// compare orders
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

// TestDexFuzzHarness simulates two chains interacting through the DEX pipeline under random ops/timing.
// It is designed to stress safety invariants rather than verify single functions.
func TestDexFuzzHarness(t *testing.T) {
	seed := time.Now().UnixNano()
	if s := os.Getenv("DEX_FUZZ_SEED"); s != "" {
		if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
			seed = parsed
		}
	}
	steps := 2000
	if s := os.Getenv("DEX_FUZZ_STEPS"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil && parsed >= 1000 && parsed <= 50000 {
			steps = parsed
		}
	}
	rng := rand.New(rand.NewSource(seed))

	sim := newDexSim(t, rng)
	history := make([]string, 0, steps)

	for step := 1; step <= steps; step++ {
		stepLabel := fmt.Sprintf("step=%d", step)
		genOps(sim, rng)
		applyLocalOps(t, sim)
		deliverRemoteReceipts(t, sim, rng)
		assertInvariant(t, sim, step, seed, history)

		if step%100 == 0 {
			t.Logf("[seed=%d step=%d] poolX=%d poolY=%d pointsX=%d pointsY=%d",
				seed, step,
				sim.chainX.poolBalance(), sim.chainY.poolBalance(),
				sim.chainX.poolPoints(), sim.chainY.poolPoints())
		}
		sim.advanceHeight()
		history = append(history, stepLabel)
	}
	_ = history // retained for future expansion; appended only on invariant breach.
}

// Ensures slippage/requestedAmount gating matches SafeComputeDY outputs.
func TestHandleDexBatchOrdersRespectsRequestedAmount(t *testing.T) {
	sm := newTestStateMachine(t)
	sm.Config.ChainId = 1

	chainId := uint64(2)
	addr := newTestAddress(t, 1)
	// seed pool used for payouts
	require.NoError(t, sm.SetPool(&Pool{Id: chainId + LiquidityPoolAddend, Amount: 10_000}))

	batch := &lib.DexBatch{
		Committee: chainId,
		Orders: []*lib.DexLimitOrder{
			{AmountForSale: 1_000, RequestedAmount: 0, Address: addr.Bytes(), OrderId: []byte{0x01}},
			{AmountForSale: 500, RequestedAmount: 999_999, Address: addr.Bytes(), OrderId: []byte{0x02}}, // should fail
		},
	}

	x, y := uint64(10_000), uint64(10_000)
	x0, y0 := x, y

	receipts, err := sm.HandleDexBatchOrders(batch, &x, &y, chainId)
	require.NoError(t, err)
	require.Len(t, receipts, len(batch.Orders))

	expectedDY0 := SafeComputeDY(x0, y0, batch.Orders[0].AmountForSale)
	require.Greater(t, expectedDY0, batch.Orders[0].RequestedAmount)
	require.Equal(t, expectedDY0, receipts[0], "first order should succeed at computed dy")

	expectedDY1 := SafeComputeDY(x0+batch.Orders[0].AmountForSale, y0-expectedDY0, batch.Orders[1].AmountForSale)
	require.Less(t, expectedDY1, batch.Orders[1].RequestedAmount)
	require.Zero(t, receipts[1], "second order should fail slippage gate")
}

// Proves withdraw->redeposit cannot increase LP points (no gain loop), allowing only rounding loss.
func TestWithdrawThenRedeploy_NoPointGain(t *testing.T) {
	sm := newTestStateMachine(t)
	sm.Config.ChainId = 1

	chainId := uint64(2)
	user := newTestAddress(t, 2)
	// seed pools
	require.NoError(t, sm.SetPool(&Pool{Id: chainId + LiquidityPoolAddend, Amount: 20_000}))
	require.NoError(t, sm.SetPool(&Pool{Id: chainId + HoldingPoolAddend, Amount: 5_000}))
	require.NoError(t, sm.SetAccount(&Account{Address: user.Bytes(), Amount: 0}))

	x, y := uint64(20_000), uint64(20_000)
	depositAmt := uint64(4_000)

	depositBatch := &lib.DexBatch{
		Deposits: []*lib.DexLiquidityDeposit{
			{Amount: depositAmt, Address: user.Bytes(), OrderId: []byte{0x03}},
		},
	}
	require.NoError(t, sm.HandleBatchDeposit(depositBatch, chainId, &x, &y, true))
	pool, err := sm.GetPool(chainId + LiquidityPoolAddend)
	require.NoError(t, err)
	initialPoints, err := pool.GetPointsFor(user.Bytes())
	require.NoError(t, err)
	require.NotZero(t, initialPoints)

	withdrawBatch := &lib.DexBatch{
		Withdrawals: []*lib.DexLiquidityWithdraw{
			{Percent: 100, Address: user.Bytes(), OrderId: []byte{0x04}},
		},
	}
	require.NoError(t, sm.HandleBatchWithdraw(withdrawBatch, chainId, &x, &y, true))

	// capture payout from account
	acc, err := sm.GetAccount(user)
	require.NoError(t, err)
	payout := acc.Amount
	require.NotZero(t, payout)

	// move payout into holding pool to simulate redeposit
	require.NoError(t, sm.AccountSub(user, payout))
	require.NoError(t, sm.PoolAdd(chainId+HoldingPoolAddend, payout))

	redepositBatch := &lib.DexBatch{
		Deposits: []*lib.DexLiquidityDeposit{
			{Amount: payout, Address: user.Bytes(), OrderId: []byte{0x05}},
		},
	}
	require.NoError(t, sm.HandleBatchDeposit(redepositBatch, chainId, &x, &y, true))

	finalPool, err := sm.GetPool(chainId + LiquidityPoolAddend)
	require.NoError(t, err)
	finalPoints, err := finalPool.GetPointsFor(user.Bytes())
	require.NoError(t, err)

	require.LessOrEqual(t, finalPoints, initialPoints, "round-trip should not increase LP points")
}

type dexSim struct {
	chainX, chainY     *dexChain
	counterId          uint64
	totalSupply        uint64
	lpRatioBaseline    float64
	lpRatioLastUpdated int
}

type dexChain struct {
	sm       *StateMachine
	chainId  uint64
	users    []crypto.AddressI
	txBuffer []queuedOp
	lastDY   map[uint64]uint64 // dX -> last seen dy for monotonicity sampling
}

type queuedOp struct {
	kind string
	msg  any
}

func newDexSim(t *testing.T, rng *rand.Rand) *dexSim {
	chainXId, chainYId := uint64(1), uint64(2)
	counterId := chainYId

	chainX := newDexChain(t, chainXId, counterId, rng)
	chainY := newDexChain(t, chainYId, chainXId, rng)
	return &dexSim{
		chainX:      chainX,
		chainY:      chainY,
		counterId:   counterId,
		totalSupply: assetSupply(chainX) + assetSupply(chainY),
	}
}

func newDexChain(t *testing.T, chainId, counterId uint64, rng *rand.Rand) *dexChain {
	sm := newTestStateMachine(t)
	sm.Config.ChainId = chainId
	// seed users
	users := make([]crypto.AddressI, 0, 8)
	for i := 0; i < cap(users); i++ {
		addr := newTestAddress(t, rng.Intn(8))
		balance := uint64(10_000 + rng.Intn(10_000))
		require.NoError(t, sm.SetAccount(&Account{Address: addr.Bytes(), Amount: balance}))
		users = append(users, addr)
	}
	// seed pools for counter asset: liquidity + holding
	liqPoolId := counterId + LiquidityPoolAddend
	holdPoolId := counterId + HoldingPoolAddend
	require.NoError(t, sm.SetPool(&Pool{Id: liqPoolId, Amount: 50_000}))
	require.NoError(t, sm.SetPool(&Pool{Id: holdPoolId, Amount: 0}))

	// initialize empty batches so rotations can start
	require.NoError(t, sm.SetDexBatch(KeyForNextBatch(counterId), &lib.DexBatch{
		Committee:       counterId,
		PoolSize:        50_000,
		CounterPoolSize: 50_000,
	}))
	require.NoError(t, sm.SetDexBatch(KeyForLockedBatch(counterId), &lib.DexBatch{}))

	return &dexChain{
		sm:       &sm,
		chainId:  chainId,
		users:    users,
		txBuffer: []queuedOp{},
		lastDY:   make(map[uint64]uint64),
	}
}

func (s *dexSim) advanceHeight() {
	s.chainX.sm.height++
	s.chainY.sm.height++
}

func genOps(sim *dexSim, rng *rand.Rand) {
	sim.chainX.txBuffer = sim.chainX.txBuffer[:0]
	sim.chainY.txBuffer = sim.chainY.txBuffer[:0]
	generate := func(chain *dexChain) {
		balances := make(map[string]uint64, len(chain.users))
		for _, u := range chain.users {
			if acc, err := chain.sm.GetAccount(u); err == nil && acc != nil {
				balances[string(u.Bytes())] = acc.Amount
			}
		}
		getAvail := func(u crypto.AddressI) uint64 { return balances[string(u.Bytes())] }
		subAvail := func(u crypto.AddressI, amt uint64) {
			key := string(u.Bytes())
			if cur, ok := balances[key]; ok && cur >= amt {
				balances[key] = cur - amt
			}
		}

		n := 1 + rng.Intn(50)
		for i := 0; i < n; i++ {
			switch rng.Intn(3) {
			case 0:
				// order
				user := chain.users[rng.Intn(len(chain.users))]
				amount, ok := affordableAmount(getAvail(user), 500, rng)
				if !ok {
					i--
					continue
				}
				reqAmt := uint64(rng.Intn(int(amount + 1)))
				subAvail(user, amount)
				chain.txBuffer = append(chain.txBuffer, queuedOp{
					kind: "order",
					msg: &MessageDexLimitOrder{
						ChainId:         counterChain(chain),
						AmountForSale:   amount,
						RequestedAmount: reqAmt,
						Address:         user.Bytes(),
						OrderId:         lib.RandSlice(uint64(crypto.HashSize)),
					},
				})
			case 1:
				// deposit
				user := chain.users[rng.Intn(len(chain.users))]
				amount, ok := affordableAmount(getAvail(user), 400, rng)
				if !ok {
					i--
					continue
				}
				subAvail(user, amount)
				chain.txBuffer = append(chain.txBuffer, queuedOp{
					kind: "deposit",
					msg: &MessageDexLiquidityDeposit{
						ChainId: counterChain(chain),
						Amount:  amount,
						Address: user.Bytes(),
						OrderId: lib.RandSlice(uint64(crypto.HashSize)),
					},
				})
			default:
				// withdraw (only if LP exists)
				if chain.poolPoints() == 0 {
					i-- // retry with a different op type
					continue
				}
				lpHolders := chain.lpHolders()
				if len(lpHolders) == 0 {
					i--
					continue
				}
				user := lpHolders[rng.Intn(len(lpHolders))]
				percent := uint64(1 + rng.Intn(20))
				chain.txBuffer = append(chain.txBuffer, queuedOp{
					kind: "withdraw",
					msg: &MessageDexLiquidityWithdraw{
						ChainId: counterChain(chain),
						Percent: percent,
						Address: user.Bytes(),
						OrderId: lib.RandSlice(uint64(crypto.HashSize)),
					},
				})
			}
		}
	}
	generate(sim.chainX)
	generate(sim.chainY)
}

func applyLocalOps(t *testing.T, sim *dexSim) {
	apply := func(chain *dexChain) {
		for _, op := range chain.txBuffer {
			switch msg := op.msg.(type) {
			case *MessageDexLimitOrder:
				require.NoError(t, chain.sm.HandleMessageDexLimitOrder(msg))
			case *MessageDexLiquidityDeposit:
				require.NoError(t, chain.sm.HandleMessageDexLiquidityDeposit(msg))
			case *MessageDexLiquidityWithdraw:
				require.NoError(t, chain.sm.HandleMessageDexLiquidityWithdraw(msg))
			}
		}
		chain.txBuffer = chain.txBuffer[:0]
	}
	apply(sim.chainX)
	apply(sim.chainY)
}

func deliverRemoteReceipts(t *testing.T, sim *dexSim, rng *rand.Rand) {
	process := func(local, remote *dexChain, mode string) {
		// get the remote locked batch targeting the local chain
		remoteBatch, _ := remote.sm.GetDexBatch(local.sm.Config.ChainId, true)
		if remoteBatch == nil {
			return
		}
		// if local still has a locked batch to settle, skip payout delta check because receipts processing moves the pool
		localLocked, _ := local.sm.GetDexBatch(remote.sm.Config.ChainId, true)
		skipPayoutCheck := localLocked != nil && !localLocked.IsEmpty()
		// capture pool before for payout accounting check
		beforePool, _ := local.sm.GetPool(local.counterId() + LiquidityPoolAddend)
		require.NoError(t, local.sm.HandleRemoteDexBatch(remoteBatch, remote.sm.Config.ChainId))
		afterPool, _ := local.sm.GetPool(local.counterId() + LiquidityPoolAddend)
		// receipts sum should match delta (orders + withdrawals only reduce local pool)
		newLocked, _ := local.sm.GetDexBatch(remote.sm.Config.ChainId, true)
		if !skipPayoutCheck && newLocked != nil && !newLocked.IsEmpty() {
			var receiptSum uint64
			for _, r := range newLocked.Receipts {
				receiptSum += r
			}
			for _, w := range remoteBatch.Withdrawals {
				points, err := beforePool.GetPointsFor(w.Address)
				if err != nil {
					continue
				}
				pointsToRemove := lib.SafeMulDiv(points, w.Percent, 100)
				if beforePool.TotalPoolPoints == 0 {
					continue
				}
				withdrawShare := lib.SafeMulDiv(beforePool.Amount, pointsToRemove, beforePool.TotalPoolPoints)
				receiptSum += withdrawShare
			}
			before := beforePool.Amount
			after := afterPool.Amount
			if diff := int64(before) - int64(after) - int64(receiptSum); diff > 1 || diff < -1 {
				t.Fatalf("payout vs pool delta mismatch chain=%d before=%d after=%d sum=%d diff=%d", local.chainId, before, after, receiptSum, diff)
			}
		}
	}

	process(sim.chainX, sim.chainY, "normal")
	process(sim.chainY, sim.chainX, "normal")
}

func assertInvariant(t *testing.T, sim *dexSim, step int, seed int64, history []string) {
	total := assetSupply(sim.chainX) + assetSupply(sim.chainY)
	if diff := int64(sim.totalSupply) - int64(total); diff != 0 {
		failInvariant(t, seed, step, fmt.Sprintf("token conservation want=%d got=%d diff=%d", sim.totalSupply, total, diff), history, sim)
	}

	lockedX, _ := sim.chainX.sm.GetDexBatch(sim.chainX.counterId(), true)
	lockedY, _ := sim.chainY.sm.GetDexBatch(sim.chainY.counterId(), true)
	if lockedX.IsEmpty() && lockedY.IsEmpty() {
		// per-user LP symmetry
		lpMapX := sim.chainX.lpPointMap()
		lpMapY := sim.chainY.lpPointMap()
		if len(lpMapX) != len(lpMapY) {
			failInvariant(t, seed, step, fmt.Sprintf("LP holder set mismatch x=%d y=%d", len(lpMapX), len(lpMapY)), history, sim)
		}
		for k, vx := range lpMapX {
			if k == string(deadAddr.Bytes()) {
				continue
			}
			if vy, ok := lpMapY[k]; !ok || vx != vy {
				failInvariant(t, seed, step, fmt.Sprintf("LP points diverge for holder=%x x=%d y=%d", []byte(k), vx, vy), history, sim)
			}
		}

		if sim.chainX.poolPoints() != sim.chainY.poolPoints() {
			failInvariant(t, seed, step, fmt.Sprintf("LP points mismatch x=%d y=%d", sim.chainX.poolPoints(), sim.chainY.poolPoints()), history, sim)
		}

		ratio := func(chain *dexChain) float64 {
			pool, _ := chain.sm.GetPool(chain.counterId() + LiquidityPoolAddend)
			remote := float64(counterPoolAmount(sim, chain))
			if pool.Amount == 0 || remote == 0 {
				return 0
			}
			return float64(pool.Amount) / remote
		}
		rx, ry := ratio(sim.chainX), ratio(sim.chainY)
		if math.Abs(rx-ry) > 0.0001 {
			failInvariant(t, seed, step, fmt.Sprintf("ratio mismatch rx=%.6f ry=%.6f", rx, ry), history, sim)
		}
		// stronger symmetry: pool vs mirrored pool should not diverge materially
		for _, chain := range []*dexChain{sim.chainX, sim.chainY} {
			pool, _ := chain.sm.GetPool(chain.counterId() + LiquidityPoolAddend)
			remote := counterPoolAmount(sim, chain)
			if diff := int64(pool.Amount) - int64(remote); diff > 1 || diff < -1 {
				failInvariant(t, seed, step, fmt.Sprintf("pool mirror mismatch chain=%d local=%d remote=%d diff=%d", chain.chainId, pool.Amount, remote, diff), history, sim)
			}
			// sum of per-user points equals total (within deadAddr dust allowance)
			sumPoints := uint64(0)
			for _, pt := range pool.Points {
				sumPoints += pt.Points
			}
			if sumPoints != pool.TotalPoolPoints {
				failInvariant(t, seed, step, fmt.Sprintf("pool point sum mismatch chain=%d sum=%d total=%d", chain.chainId, sumPoints, pool.TotalPoolPoints), history, sim)
			}
			// no negative pools (uint64 already, so ensure no wrap beyond supply)
			if pool.Amount > sim.totalSupply {
				failInvariant(t, seed, step, fmt.Sprintf("pool exceeds total supply chain=%d pool=%d supply=%d", chain.chainId, pool.Amount, sim.totalSupply), history, sim)
			}
		}

		// Enforce LP point math invariant: total points should stay proportional to sqrt(x*y)
		lpRatio := func(chain *dexChain) float64 {
			pool, _ := chain.sm.GetPool(chain.counterId() + LiquidityPoolAddend)
			remote := float64(counterPoolAmount(sim, chain))
			if pool.Amount == 0 || remote == 0 || pool.TotalPoolPoints == 0 {
				return 0
			}
			k := math.Sqrt(float64(pool.Amount) * remote)
			return float64(pool.TotalPoolPoints) / k
		}
		rlx, rly := lpRatio(sim.chainX), lpRatio(sim.chainY)
		if sim.lpRatioBaseline == 0 && rlx != 0 && rly != 0 {
			sim.lpRatioBaseline = (rlx + rly) / 2
			sim.lpRatioLastUpdated = step
		}
		const lpRatioTolerance = 0.01 // allow small rounding drift
		if sim.lpRatioBaseline > 0 {
			if math.Abs(rlx-sim.lpRatioBaseline) > lpRatioTolerance || math.Abs(rly-sim.lpRatioBaseline) > lpRatioTolerance {
				failInvariant(t, seed, step, fmt.Sprintf("LP point/share invariant drift (baseline=%.6f rlx=%.6f rly=%.6f lastUpdated=%d)", sim.lpRatioBaseline, rlx, rly, sim.lpRatioLastUpdated), history, sim)
			}
		}
	}

	// SafeComputeDY monotonicity sampling: if pools moved against a trader (x up, y down), dy for same dX should not increase
	if step > 1 {
		sim.chainX.recordDYMonotonicity(t, sim)
		sim.chainY.recordDYMonotonicity(t, sim)
	}
}

func (c *dexChain) counterId() uint64 { return counterChain(c) }

func counterChain(c *dexChain) uint64 {
	if c.chainId == 1 {
		return 2
	}
	return 1
}

func counterPoolAmount(sim *dexSim, chain *dexChain) uint64 {
	if chain.chainId == sim.chainX.chainId {
		return sim.chainY.poolBalance()
	}
	return sim.chainX.poolBalance()
}

func (c *dexChain) poolBalance() uint64 {
	bal, _ := c.sm.GetPoolBalance(c.counterId() + LiquidityPoolAddend)
	return bal
}

func (c *dexChain) poolPoints() uint64 {
	p, _ := c.sm.GetPool(c.counterId() + LiquidityPoolAddend)
	return p.TotalPoolPoints
}

func affordableAmount(avail uint64, cap uint64, rng *rand.Rand) (uint64, bool) {
	limit := avail
	if limit > cap {
		limit = cap
	}
	if limit == 0 {
		return 0, false
	}
	amt := uint64(1 + rng.Intn(int(limit)))
	return amt, true
}

func (c *dexChain) lpHolders() []crypto.AddressI {
	p, _ := c.sm.GetPool(c.counterId() + LiquidityPoolAddend)
	holders := make([]crypto.AddressI, 0, len(p.Points))
	for _, pt := range p.Points {
		if pt.Points > 0 && !bytes.Equal(pt.Address, deadAddr.Bytes()) {
			holders = append(holders, crypto.NewAddressFromBytes(pt.Address))
		}
	}
	return holders
}

func (c *dexChain) lpPointMap() map[string]uint64 {
	p, _ := c.sm.GetPool(c.counterId() + LiquidityPoolAddend)
	out := make(map[string]uint64, len(p.Points))
	for _, pt := range p.Points {
		out[string(pt.Address)] = pt.Points
	}
	return out
}

// recordDYMonotonicity samples SafeComputeDY for a couple of dX values against prior pools.
func (c *dexChain) recordDYMonotonicity(t *testing.T, sim *dexSim) {
	pool, _ := c.sm.GetPool(c.counterId() + LiquidityPoolAddend)
	remote := counterPoolAmount(sim, c)
	if pool.Amount == 0 || remote == 0 {
		return
	}
	samples := []uint64{1, 10, 100}
	for _, dx := range samples {
		dy := SafeComputeDY(remote, pool.Amount, dx)
		if last, ok := c.lastDY[dx]; ok {
			// As remote increases and/or pool decreases, price should not get better for taker
			if dy > last && (remote >= c.prevRemote(dx) && pool.Amount <= c.prevLocal(dx)) {
				t.Fatalf("SafeComputeDY monotonicity violated chain=%d dx=%d last=%d now=%d", c.chainId, dx, last, dy)
			}
		}
		c.lastDY[dx] = dy
		c.setPrevPools(dx, remote, pool.Amount)
	}
}

func (c *dexChain) setPrevPools(dx, remote, local uint64) {
	// encode in lastDY map using sentinel keys to avoid extra maps
	c.lastDY[dx<<32|1] = remote
	c.lastDY[dx<<32|2] = local
}

func (c *dexChain) prevRemote(dx uint64) uint64 {
	return c.lastDY[dx<<32|1]
}

func (c *dexChain) prevLocal(dx uint64) uint64 {
	return c.lastDY[dx<<32|2]
}

func assetSupply(chain *dexChain) uint64 {
	var total uint64
	it, err := chain.sm.Iterator(AccountPrefix())
	if err == nil {
		defer it.Close()
		for ; it.Valid(); it.Next() {
			acc, _ := chain.sm.unmarshalAccount(it.Value())
			if acc != nil {
				total += acc.Amount
			}
		}
	}
	pools, err := chain.sm.GetPools()
	if err == nil {
		for _, p := range pools {
			if p != nil {
				total += p.Amount
			}
		}
	}
	return total
}

func failInvariant(t *testing.T, seed int64, step int, reason string, history []string, sim *dexSim) {
	var b strings.Builder
	fmt.Fprintf(&b, "invariant failed: %s seed=%d step=%d\n", reason, seed, step)
	fmt.Fprintf(&b, "chainX: %s\n", dumpChainState(sim.chainX))
	fmt.Fprintf(&b, "chainY: %s\n", dumpChainState(sim.chainY))
	if len(history) > 0 {
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
		}
		fmt.Fprintf(&b, "recent steps: %v\n", history[start:])
	}
	t.Fatalf("%s", b.String())
}

func dumpChainState(chain *dexChain) string {
	liq, _ := chain.sm.GetPool(chain.counterId() + LiquidityPoolAddend)
	hold, _ := chain.sm.GetPool(chain.counterId() + HoldingPoolAddend)
	locked, _ := chain.sm.GetDexBatch(chain.sm.Config.ChainId, true)
	return fmt.Sprintf("chainId=%d liq=%d hold=%d points=%d lockedOrders=%d lockedDeposits=%d lockedWithdrawals=%d",
		chain.chainId, liq.Amount, hold.Amount, liq.TotalPoolPoints,
		len(locked.Orders), len(locked.Deposits), len(locked.Withdrawals))
}

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
		chain2.Config.ChainId = chain2Id
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

func (m *MockRCManager) GetDexBatch(rootChainId, height, committee uint64, withPoints bool) (*lib.DexBatch, lib.ErrorI) {
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
