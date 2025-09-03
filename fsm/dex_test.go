package fsm

import (
	"bytes"
	"fmt"
	"github.com/canopy-network/canopy/lib/crypto"
	"math"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

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
			expectedLockedBatch: &lib.DexBatch{
				Committee: 2,
				Orders:    []*lib.DexLimitOrder{},
				Deposits:  []*lib.DexLiquidityDeposit{},
				Withdraws: []*lib.DexLiquidityWithdraw{},
				PoolSize:  0,
				Receipts:  []bool{},
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
			expectedLockedBatch: &lib.DexBatch{
				Committee:    1,
				ReceiptHash:  bytes.Repeat([]byte{0x46}, 32), // Hash gets populated with 'F' characters during processing
				Orders:       []*lib.DexLimitOrder{},
				Deposits:     []*lib.DexLiquidityDeposit{},
				Withdraws:    []*lib.DexLiquidityWithdraw{},
				PoolSize:     1000, // Should be updated to liquidity pool amount
				Receipts:     []bool{},
				LockedHeight: 2,
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
				Amount: 176, // 300 - 62*2
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
		{
			name:          "locked batch: no receipts",
			detail:        "test handling a batch without receipts when locked",
			errorContains: "the dex batch receipt doesn't correspond to the last batch",
			chainId:       1,
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
			name:          "locked batch: mismatch receipts",
			detail:        "test handling a batch with mismatched when locked",
			chainId:       1,
			errorContains: "the dex batch receipt doesn't correspond to the last batch",
			buyBatch: &lib.DexBatch{
				Committee: 1,
				Orders: []*lib.DexLimitOrder{
					{
						AmountForSale:   25,
						RequestedAmount: 13,
						Address:         newTestAddressBytes(t, 2),
					},
				},
				Receipts: []bool{true, false},
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
				Receipts:  []bool{true},
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
				Receipts:  []bool{false},
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
				Amount: 176, // 300 - 62*2
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
					Withdraws: []*lib.DexLiquidityWithdraw{{
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
		PoolSize:     initPoolSize,
		LockedHeight: 2,
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
		Committee:    chain2Id,
		ReceiptHash:  lockedBatch.Hash(),
		PoolSize:     100,
		LockedHeight: 2,
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
		Withdraws: []*lib.DexLiquidityWithdraw{{
			Address: account1.Bytes(),
			Percent: 100,
		}},
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
		Withdraws: []*lib.DexLiquidityWithdraw{{
			Address: account1.Bytes(),
			Percent: 100,
		}},
		ReceiptHash:  emptyBatch.Hash(),
		PoolSize:     initPoolSize + depositAmount,
		LockedHeight: 2,
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
		Committee:    chain2Id,
		ReceiptHash:  lockedBatch.Hash(),
		PoolSize:     initPoolSize - expectedY,
		LockedHeight: 2,
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
		ReceiptHash:  emptyBatch.Hash(),
		PoolSize:     initPoolSize,
		LockedHeight: 2,
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
		Committee:    chain2Id,
		ReceiptHash:  lockedBatch.Hash(),
		Receipts:     []bool{true},
		PoolSize:     initPoolSize - expectedY,
		LockedHeight: 2,
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
		receipts     []bool
		chainId      uint64
		setupState   func(*StateMachine)
		expectError  bool
		errorMessage string
	}{
		{
			name:     "locked batch exists",
			detail:   "test when locked batch still exists (should exit early)",
			buyBatch: &lib.DexBatch{Committee: 1},
			receipts: []bool{true, false},
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
			receipts: []bool{true, false},
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
			receipts: []bool{},
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
			receipts: []bool{true, false, true},
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

			err := sm.RotateDexSellBatch(test.buyBatch, test.chainId, test.receipts)

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
