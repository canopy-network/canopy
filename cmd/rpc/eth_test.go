package rpc

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/canopy-network/canopy/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestEthMaxPriorityFeePerGas(t *testing.T) {
	server := &Server{}
	result, err := server.EthMaxPriorityFeePerGas(nil)
	require.NoError(t, err)
	value := result.(hexutil.Big)
	require.Equal(t, "0x0", (*hexutil.Big)(&value).String())
}

func TestRLPV2EffectiveGasPriceDistinguishesPendingAndMined(t *testing.T) {
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	to := common.HexToAddress("0x0000000000000000000000000000000000000004")
	evmChainID, ok := fsm.CanopyIdsToEVMChainIdV2(1, 1)
	require.True(t, ok)
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(new(big.Int).SetUint64(evmChainID)), &types.DynamicFeeTx{
		ChainID: new(big.Int).SetUint64(evmChainID), GasFeeCap: big.NewInt(20_000_000_000), GasTipCap: big.NewInt(0),
		Gas: 21_000, To: &to,
	})
	tx := &lib.TxResult{Transaction: &lib.Transaction{Memo: fsm.RLPV2Indicator}}

	pendingPrice, err := effectiveGasPriceFromEthTx(tx, ethTx, true)
	require.NoError(t, err)
	require.Equal(t, "20000000000", pendingPrice.String())
	minedPrice, err := effectiveGasPriceFromEthTx(tx, ethTx, false)
	require.NoError(t, err)
	require.Equal(t, "10000000000", minedPrice.String())
}

func TestEthChainIDUsesV2SigningDomainImmediately(t *testing.T) {
	config := lib.DefaultConfig()
	server := &Server{config: config}

	result, err := server.EthChainId(nil)
	require.NoError(t, err)
	require.Equal(t, "0x140000001", hexutil.Uint64(result.(hexutil.Uint64)).String())
}

func TestEthGetBalanceIncludesVestedFunds(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	sm := newTestRPCStateMachine(t, db, log)
	address := crypto.NewAddress(bytes.Repeat([]byte{1}, crypto.AddressSize))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: address.Bytes(), Amount: 100,
		VestingAmount: 100, VestingStartHeight: 1, VestingCliffHeight: 5, VestingEndHeight: 10}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 1)

	result, rpcErr := (&Server{controller: &controller.Controller{FSM: sm}}).EthGetBalance([]any{"0x" + address.String(), latestBlockTag})
	require.NoError(t, rpcErr)
	balance := result.(hexutil.Big)
	require.Equal(t, fsm.UpscaleTo18Decimals(100), (*big.Int)(&balance))
}

func TestEthFeeHistoryTruncatesAtGenesis(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, storeErr := store.NewStoreInMemory(log)
	require.NoError(t, storeErr)

	sm := newTestRPCStateMachine(t, db, log)
	setFSMHeight(t, sm, 1)

	server := &Server{controller: &controller.Controller{FSM: sm}}
	result, rpcErr := server.EthFeeHistory([]any{"0xa", latestBlockTag, []any{5.0, 30.0, 50.0, 75.0}})
	require.NoError(t, rpcErr)

	feeHistory := result.(ethRPCFeeHistory)
	require.EqualValues(t, 0, feeHistory.OldestBlock)
	require.Len(t, feeHistory.BaseFeePerGas, 2)
	require.Len(t, feeHistory.GasUsedRatio, 1)
	require.Len(t, feeHistory.Reward, 1)
	require.Len(t, feeHistory.Reward[0], 4)
	baseFee := feeHistory.BaseFeePerGas[0]
	reward := feeHistory.Reward[0][0]
	require.Equal(t, "0x2540be400", (*hexutil.Big)(&baseFee).String())
	require.Equal(t, "0x0", (*hexutil.Big)(&reward).String())
}

func TestEthFeeHistoryZeroBlockCountReturnsNoData(t *testing.T) {
	result, err := new(Server).EthFeeHistory([]any{"0x0", latestBlockTag, nil})
	require.NoError(t, err)

	feeHistory := result.(ethRPCFeeHistory)
	require.Nil(t, feeHistory.BaseFeePerGas)
	require.Nil(t, feeHistory.GasUsedRatio)
	require.Nil(t, feeHistory.Reward)
	bz, jsonErr := json.Marshal(feeHistory)
	require.NoError(t, jsonErr)
	require.JSONEq(t, `{"oldestBlock":"0x0","gasUsedRatio":null}`, string(bz))
}

func TestPendingCacheDeduplicatesAndProtectsReplacement(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()

	hash := "0xabc123"
	first := &lib.Transaction{Nonce: 1}
	replacement := &lib.Transaction{Nonce: 2}
	cachePendingEthTx(hash, first)
	cachePendingEthTx(hash, replacement)
	got, ok := pseudoPendingTxsMap.Load(hash)
	require.True(t, ok)
	require.Same(t, first, got)
	require.EqualValues(t, 1, pendingEthTxCacheSize.Load())

	pseudoPendingTxsMap.Store(hash, replacement)

	require.False(t, pseudoPendingTxsMap.CompareAndDelete(hash, first))
	got, ok = pseudoPendingTxsMap.Load(hash)
	require.True(t, ok)
	require.Same(t, replacement, got)
}

func TestPendingCacheRejectsNewEntriesAtCapacity(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()

	pendingEthTxCacheSize.Store(ethPendingTxMaxEntries)
	cachePendingEthTx("0xfull", &lib.Transaction{Nonce: 1})

	_, ok := pseudoPendingTxsMap.Load("0xfull")
	require.False(t, ok)
	require.Equal(t, ethPendingTxMaxEntries, pendingEthTxCacheSize.Load())
}

func TestEthSendRawTransactionInsertsSynchronously(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	raw := mustNewSignedRawEthTxForRPCWithKey(t, key, 0)
	tx, txErr := fsm.RLPToCanopyTransactionV2(raw)
	require.NoError(t, txErr)
	mempool := &controller.Mempool{Mempool: lib.NewMempool(lib.DefaultMempoolConfig()), L: &sync.Mutex{}}
	ctrl := &controller.Controller{Mempool: mempool}
	setUnexportedField(t, ctrl, "isSyncing", &atomic.Bool{})
	server := &Server{controller: ctrl}

	result, err := server.EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.NoError(t, err)
	require.Equal(t, ethHashStringFromTransaction(tx), result)
	bz, marshalErr := lib.Marshal(tx)
	require.NoError(t, marshalErr)
	require.True(t, mempool.Contains(crypto.HashString(bz)))
	_, replacementErr := server.EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.ErrorContains(t, replacementErr, "replacement transaction underpriced")
}

func TestEthSendRawTransactionReplacementPolicy(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	mempool := &controller.Mempool{Mempool: lib.NewMempool(lib.DefaultMempoolConfig()), L: &sync.Mutex{}}
	ctrl := &controller.Controller{Mempool: mempool}
	setUnexportedField(t, ctrl, "isSyncing", &atomic.Bool{})
	server := &Server{controller: ctrl}
	originalRaw := mustNewSignedRawEthTxForRPCWithFees(t, key, 0, 10_000_000_000, 10)
	original, errI := fsm.RLPToCanopyTransactionV2(originalRaw)
	require.NoError(t, errI)
	_, err = server.EthSendRawTransaction([]any{hexutil.Encode(originalRaw)})
	require.NoError(t, err)

	for _, raw := range [][]byte{
		mustNewSignedRawEthTxForRPCWithFees(t, key, 0, 10_999_999_999, 11),
		mustNewSignedRawEthTxForRPCWithFees(t, key, 0, 11_000_000_000, 10),
	} {
		_, err = server.EthSendRawTransaction([]any{hexutil.Encode(raw)})
		require.ErrorContains(t, err, "replacement transaction underpriced")
	}
	replacementRaw := mustNewSignedRawEthTxForRPCWithFees(t, key, 0, 11_000_000_000, 11)
	replacement, errI := fsm.RLPToCanopyTransactionV2(replacementRaw)
	require.NoError(t, errI)
	_, err = server.EthSendRawTransaction([]any{hexutil.Encode(replacementRaw)})
	require.NoError(t, err)
	originalBz, errI := lib.Marshal(original)
	require.NoError(t, errI)
	replacementBz, errI := lib.Marshal(replacement)
	require.NoError(t, errI)
	require.False(t, mempool.Contains(crypto.HashString(originalBz)))
	require.True(t, mempool.Contains(crypto.HashString(replacementBz)))
	_, cached := pseudoPendingTxsMap.Load(ethHashStringFromTransaction(original))
	require.False(t, cached)
}

func TestEthSendRawTransactionIgnoresStaleReplacementCache(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	raw := mustNewSignedRawEthTxForRPCWithKey(t, key, 0)
	tx, errI := fsm.RLPToCanopyTransactionV2(raw)
	require.NoError(t, errI)
	bz, errI := lib.Marshal(tx)
	require.NoError(t, errI)
	mempool := &controller.Mempool{Mempool: lib.NewMempool(lib.DefaultMempoolConfig()), L: &sync.Mutex{}}
	ctrl := &controller.Controller{Mempool: mempool}
	setUnexportedField(t, ctrl, "isSyncing", &atomic.Bool{})
	server := &Server{controller: ctrl}
	_, err = server.EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.NoError(t, err)
	mempool.L.Lock()
	mempool.DeleteTransaction(bz)
	mempool.L.Unlock()
	_, err = server.EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.NoError(t, err)
	require.True(t, mempool.Contains(crypto.HashString(bz)))
}

func TestEthSendRawTransactionContinuesWhenPendingCacheIsFull(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()
	pendingEthTxCacheSize.Store(ethPendingTxMaxEntries)

	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	raw := mustNewSignedRawEthTxForRPCWithKey(t, key, 0)
	tx, txErr := fsm.RLPToCanopyTransactionV2(raw)
	require.NoError(t, txErr)
	mempool := &controller.Mempool{Mempool: lib.NewMempool(lib.DefaultMempoolConfig()), L: &sync.Mutex{}}
	ctrl := &controller.Controller{Mempool: mempool}
	setUnexportedField(t, ctrl, "isSyncing", &atomic.Bool{})

	result, rpcErr := (&Server{controller: ctrl}).EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.NoError(t, rpcErr)
	require.Equal(t, ethHashStringFromTransaction(tx), result)
	bz, marshalErr := lib.Marshal(tx)
	require.NoError(t, marshalErr)
	require.True(t, mempool.Contains(crypto.HashString(bz)))
	_, cached := pseudoPendingTxsMap.Load(result)
	require.False(t, cached)
}

func TestEthSendRawTransactionAllowsNonceBelowValidatedPendingFloor(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()

	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	raw := mustNewSignedRawEthTxForRPCWithKey(t, key, 2)
	mempool := &controller.Mempool{Mempool: lib.NewMempool(lib.DefaultMempoolConfig()), L: &sync.Mutex{}}
	setUnexportedField(t, mempool, "cachedResults", lib.TxResults{{
		Sender:      ethCrypto.PubkeyToAddress(key.PublicKey).Bytes(),
		Transaction: &lib.Transaction{Memo: fsm.RLPV2Indicator, Nonce: 5},
	}})
	ctrl := &controller.Controller{Mempool: mempool}
	setUnexportedField(t, ctrl, "isSyncing", &atomic.Bool{})

	_, rpcErr := (&Server{controller: ctrl}).EthSendRawTransaction([]any{hexutil.Encode(raw)})
	require.NoError(t, rpcErr)
}

func TestEthFeeHistoryAnchorsOldestBlockToRequestedNewestBlock(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	setFSMHeight(t, sm, 10)

	server := &Server{controller: &controller.Controller{FSM: sm}}
	result, rpcErr := server.EthFeeHistory([]any{"0x2", "0x5", nil})
	require.NoError(t, rpcErr)

	feeHistory := result.(ethRPCFeeHistory)
	require.EqualValues(t, 4, feeHistory.OldestBlock)
	require.Len(t, feeHistory.BaseFeePerGas, 3)
	require.Len(t, feeHistory.GasUsedRatio, 2)
	require.Nil(t, feeHistory.Reward)
	bz, jsonErr := json.Marshal(feeHistory)
	require.NoError(t, jsonErr)
	require.NotContains(t, string(bz), `"reward"`)
}

func TestSyntheticNativeTransactionUsesV2ChainID(t *testing.T) {
	server := new(Server)
	result, err := server.txToEthTransaction(nil, &lib.TxResult{
		Transaction: &lib.Transaction{ChainId: 1, NetworkId: 1},
	}, true)
	require.NoError(t, err)

	tx := result.(ethRPCTransaction)
	require.Equal(t, "0x140000001", tx.ChainID.String())
}

func TestSyntheticNativeTransactionPreservesLargeV2ChainID(t *testing.T) {
	networkID := uint64(1 << 31)
	expected, ok := fsm.CanopyIdsToEVMChainIdV2(1, networkID)
	require.True(t, ok)

	result, err := new(Server).txToEthTransaction(nil, &lib.TxResult{
		Transaction: &lib.Transaction{ChainId: 1, NetworkId: networkID},
	}, true)
	require.NoError(t, err)

	tx := result.(ethRPCTransaction)
	require.Equal(t, new(big.Int).SetUint64(expected), tx.ChainID.ToInt())
}

func TestEthFeeHistoryRejectsFutureNewestBlock(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	setFSMHeight(t, sm, 10)

	server := &Server{controller: &controller.Controller{FSM: sm}}
	_, rpcErr := server.EthFeeHistory([]any{"0x1", "0xa", nil})
	require.ErrorContains(t, rpcErr, "beyond the chain head")
}

func TestEthFeeHistoryRejectsOversizedBlockCount(t *testing.T) {
	server := &Server{}
	_, err := server.EthFeeHistory([]any{"0x401", latestBlockTag, nil})
	require.ErrorContains(t, err, "block count exceeds maximum")
}

func TestEthFeeHistoryRejectsOversizedRewardPercentileList(t *testing.T) {
	server := &Server{}
	percentiles := make([]any, ethFeeHistoryMaxRewardPercentiles+1)
	for i := range percentiles {
		percentiles[i] = float64(i)
	}
	_, err := server.EthFeeHistory([]any{"0x1", latestBlockTag, percentiles})
	require.ErrorContains(t, err, "reward percentile count exceeds maximum")
}

func TestEthFeeHistoryRejectsOutOfRangeRewardPercentile(t *testing.T) {
	server := &Server{}
	_, err := server.EthFeeHistory([]any{"0x1", latestBlockTag, []any{101.0}})
	require.ErrorContains(t, err, "invalid reward percentile")
}

func TestEthFeeHistoryRejectsNonIncreasingRewardPercentiles(t *testing.T) {
	server := &Server{}
	_, err := server.EthFeeHistory([]any{"0x1", latestBlockTag, []any{25.0, 25.0}})
	require.ErrorContains(t, err, "reward percentiles must be strictly increasing")
}

func TestEthGetTransactionCountEarliestMatchesHexZero(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	addr := crypto.NewAddress(bytes.Repeat([]byte{0x11}, crypto.AddressSize))
	now := uint64(time.Now().UnixMicro())

	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addr.Bytes(), Amount: 100, Nonce: 7}))
	require.NoError(t, db.IndexBlock(&lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 1,
			Hash:   crypto.Hash([]byte("block-1")),
			Time:   now,
		},
	}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 2)

	server := &Server{controller: &controller.Controller{FSM: sm}}
	earliest, rpcErr := server.EthGetTransactionCount([]any{"0x" + addr.String(), earliestBlockTag})
	require.NoError(t, rpcErr)
	hexZero, rpcErr := server.EthGetTransactionCount([]any{"0x" + addr.String(), "0x0"})
	require.NoError(t, rpcErr)

	require.Equal(t, hexZero, earliest)
	require.EqualValues(t, 7, earliest.(hexutil.Uint64))
}

func TestEthGetTransactionCountPendingIncludesLocalSubmissionWithoutValidatedProposal(t *testing.T) {
	clearAllPendingEthTxsForTest()
	defer clearAllPendingEthTxsForTest()

	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	sm := newTestRPCStateMachine(t, db, log)
	key, keyErr := ethCrypto.GenerateKey()
	require.NoError(t, keyErr)
	addr := crypto.NewAddress(ethCrypto.PubkeyToAddress(key.PublicKey).Bytes())
	require.NoError(t, sm.SetParams(fsm.DefaultParams()))
	require.NoError(t, sm.SetAccount(&fsm.Account{Address: addr.Bytes(), Amount: 100, Nonce: 0}))
	_, err = db.Commit()
	require.NoError(t, err)
	setFSMHeight(t, sm, 1)

	rawEthTx := mustNewSignedRawEthTxForRPCWithKey(t, key, 0)
	tx, errI := fsm.RLPToCanopyTransactionV2(rawEthTx)
	require.NoError(t, errI)
	hash := ethHashStringFromTransaction(tx)
	pseudoPendingTxsMap.Store(hash, tx)

	server := &Server{
		controller: &controller.Controller{
			FSM: sm,
		},
	}

	got, rpcErr := server.EthGetTransactionCount([]any{"0x" + addr.String(), pendingBlockTag})
	require.NoError(t, rpcErr)
	require.EqualValues(t, 1, got.(hexutil.Uint64))
	pending, rpcErr := server.EthGetTransactionByHash([]any{hash})
	require.NoError(t, rpcErr)
	require.Nil(t, pending.(map[string]any)["blockHash"])
}

func clearAllPendingEthTxsForTest() {
	pseudoPendingTxsMap.Range(func(key, _ any) bool {
		pseudoPendingTxsMap.Delete(key)
		return true
	})
	pendingEthTxCacheSize.Store(0)
}

func mustNewSignedRawEthTxForRPCWithKey(t *testing.T, key *ecdsa.PrivateKey, nonce uint64) []byte {
	return mustNewSignedRawEthTxForRPCWithFees(t, key, nonce, 10_000_000_000, 1)
}

func mustNewSignedRawEthTxForRPCWithFees(t *testing.T, key *ecdsa.PrivateKey, nonce uint64, feeCap, tipCap int64) []byte {
	t.Helper()
	recipient := common.HexToAddress("0x0000000000000000000000000000000000000004")
	evmChainID, ok := fsm.CanopyIdsToEVMChainIdV2(1, 1)
	require.True(t, ok)
	chainID := new(big.Int).SetUint64(evmChainID)
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(tipCap),
		GasFeeCap: big.NewInt(feeCap),
		Gas:       21_000,
		To:        &recipient,
		Value:     big.NewInt(1_000_000_000_000),
	})
	rawEthTx, err := ethTx.MarshalBinary()
	require.NoError(t, err)
	return rawEthTx
}
