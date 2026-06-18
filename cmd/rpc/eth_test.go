package rpc

import (
	"encoding/json"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/canopy-network/canopy/controller"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestEthGetTransactionByHashReturnsPendingShapeWhenBlockMissing(t *testing.T) {
	server, txResult, ethHash := newTestEthServerWithPartiallyIndexedTx(t)

	got, err := server.EthGetTransactionByHash([]any{ethHash.Hex()})
	require.NoError(t, err)

	txMap, ok := got.(map[string]any)
	require.True(t, ok)
	require.Equal(t, ethHash.Hex(), txMap["hash"])
	require.Equal(t, common.BytesToAddress(txResult.Sender).Hex(), txMap["from"])
	require.Nil(t, txMap["blockHash"])
	require.Nil(t, txMap["blockNumber"])
	require.Nil(t, txMap["transactionIndex"])
}

func TestEthGetTransactionReceiptReturnsNullWhenBlockMissing(t *testing.T) {
	server, _, ethHash := newTestEthServerWithPartiallyIndexedTx(t)

	got, err := server.EthGetTransactionReceipt([]any{ethHash.Hex()})
	require.NoError(t, err)
	require.Equal(t, "null", string(got.(json.RawMessage)))
}

func TestEthGetTransactionByHashReturnsNullWhenHashIndexMissingDespiteWarmBlockCache(t *testing.T) {
	server, _, ethHash := newTestEthServerWithDeletedButCachedTx(t)

	got, err := server.EthGetTransactionByHash([]any{ethHash.Hex()})
	require.NoError(t, err)
	require.Equal(t, "null", string(got.(json.RawMessage)))
}

func TestEthGetTransactionReceiptReturnsNullWhenHashIndexMissingDespiteWarmBlockCache(t *testing.T) {
	server, _, ethHash := newTestEthServerWithDeletedButCachedTx(t)

	got, err := server.EthGetTransactionReceipt([]any{ethHash.Hex()})
	require.NoError(t, err)
	require.Equal(t, "null", string(got.(json.RawMessage)))
}

func TestEthGetTransactionCountUsesReplayHeight(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_000)
	address := "0xCb8EC4ee2540ecD077Ce57e4b151CD7848dF9beF"

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotLatest)

	gotSafe, err := server.EthGetTransactionCount([]any{address, safeBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotSafe)

	gotFinalized, err := server.EthGetTransactionCount([]any{address, finalizedBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotFinalized)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotPending)
}

func TestEthGetTransactionCountUsesPreviousReplayHeightForExplicitCurrentBlock(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_001)
	address := "0xCb8EC4ee2540ecD077Ce57e4b151CD7848dF9beF"

	got, err := server.EthGetTransactionCount([]any{address, hexutil.EncodeUint64(5_000)})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), got)
}

func TestEthGetTransactionCountAdvancesPastPendingNonceForPendingOnly(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_000)
	tx := newTestPendingRLPTransaction(t, 4_999)
	address := "0x" + senderFromTransaction(tx)
	hash := ethHashStringFromTransaction(tx)
	registerPendingEthTx(hash, tx)
	t.Cleanup(func() { clearPendingEthTx(hash) })

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotLatest)

	gotSafe, err := server.EthGetTransactionCount([]any{address, safeBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotSafe)

	gotFinalized, err := server.EthGetTransactionCount([]any{address, finalizedBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotFinalized)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_000), gotPending)
}

func TestEthGetTransactionCountLatestDoesNotAdvancePastLocalPendingNonce(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_002)
	tx := newTestPendingRLPTransaction(t, 5_000)
	address := "0x" + senderFromTransaction(tx)
	hash := ethHashStringFromTransaction(tx)
	registerPendingEthTx(hash, tx)
	t.Cleanup(func() { clearPendingEthTx(hash) })

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_000), gotLatest)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_001), gotPending)
}

func TestEthGetTransactionCountUsesConfirmedReplayFloorForMinedTxs(t *testing.T) {
	server, db := newTestEthServerAndStoreAtHeight(t, 5_000)
	tx := newTestPendingRLPTransaction(t, 5_005)
	address := "0x" + senderFromTransaction(tx)

	require.NoError(t, db.IndexTx(newTestIndexedTxResultFromPendingTx(t, tx, 1)))
	_, commitErr := db.Commit()
	require.NoError(t, commitErr)

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_006), gotLatest)

	gotSafe, err := server.EthGetTransactionCount([]any{address, safeBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_006), gotSafe)

	gotFinalized, err := server.EthGetTransactionCount([]any{address, finalizedBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_006), gotFinalized)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_006), gotPending)
}

func TestEthGetTransactionCountExplicitCurrentBlockRespectsConfirmedReplayFloor(t *testing.T) {
	server, db := newTestEthServerAndStoreAtHeight(t, 5_008)
	tx := newTestPendingRLPTransaction(t, 5_006)
	address := "0x" + senderFromTransaction(tx)

	require.NoError(t, db.IndexTx(newTestIndexedTxResultFromPendingTx(t, tx, 1)))
	_, commitErr := db.Commit()
	require.NoError(t, commitErr)

	got, err := server.EthGetTransactionCount([]any{address, hexutil.EncodeUint64(5_007)})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_007), got)
}

func TestEthGetTransactionCountExplicitCurrentBlockDoesNotAdvancePastLocalPendingNonce(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_003)
	tx := newTestPendingRLPTransaction(t, 5_001)
	address := "0x" + senderFromTransaction(tx)
	hash := ethHashStringFromTransaction(tx)
	registerPendingEthTx(hash, tx)
	t.Cleanup(func() { clearPendingEthTx(hash) })

	got, err := server.EthGetTransactionCount([]any{address, hexutil.EncodeUint64(5_002)})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_001), got)
}

func TestEthGetTransactionCountLatestClampsToLowestLocalPendingNonce(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_004)
	txA := newTestPendingRLPTransaction(t, 5_001)
	txB := newTestPendingRLPTransaction(t, 5_002)
	address := "0x" + senderFromTransaction(txA)
	hashA := ethHashStringFromTransaction(txA)
	hashB := ethHashStringFromTransaction(txB)
	registerPendingEthTx(hashA, txA)
	registerPendingEthTx(hashB, txB)
	t.Cleanup(func() {
		clearPendingEthTx(hashA)
		clearPendingEthTx(hashB)
	})

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_001), gotLatest)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_003), gotPending)
}

func TestEthGetTransactionCountExplicitCurrentBlockClampsToLowestLocalPendingNonce(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_004)
	txA := newTestPendingRLPTransaction(t, 5_001)
	txB := newTestPendingRLPTransaction(t, 5_002)
	address := "0x" + senderFromTransaction(txA)
	hashA := ethHashStringFromTransaction(txA)
	hashB := ethHashStringFromTransaction(txB)
	registerPendingEthTx(hashA, txA)
	registerPendingEthTx(hashB, txB)
	t.Cleanup(func() {
		clearPendingEthTx(hashA)
		clearPendingEthTx(hashB)
	})

	got, err := server.EthGetTransactionCount([]any{address, hexutil.EncodeUint64(5_003)})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_001), got)
}

func TestEthGetTransactionCountKeepsPendingNonceReservedWhenBlockMissing(t *testing.T) {
	server, db := newTestEthServerAndStoreAtHeight(t, 5_000)
	tx := newTestPendingRLPTransaction(t, 4_999)
	address := "0x" + senderFromTransaction(tx)
	hash := ethHashStringFromTransaction(tx)
	registerPendingEthTx(hash, tx)
	t.Cleanup(func() { clearPendingEthTx(hash) })

	require.NoError(t, db.IndexTx(newTestIndexedTxResultFromPendingTx(t, tx, 1)))
	_, commitErr := db.Commit()
	require.NoError(t, commitErr)

	gotByHash, err := server.EthGetTransactionByHash([]any{hash})
	require.NoError(t, err)
	txMap, ok := gotByHash.(map[string]any)
	require.True(t, ok)
	require.Nil(t, txMap["blockHash"])
	require.Nil(t, txMap["blockNumber"])

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotLatest)

	gotPending, err := server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(5_000), gotPending)
}

func TestEthGetTransactionCountPendingErrorsWhenPendingNonceHitsAcceptanceCeiling(t *testing.T) {
	server := newTestEthServerAtHeight(t, 5_000)
	nonce := server.maximumAcceptedEthereumNonce()
	tx := newTestPendingRLPTransaction(t, nonce)
	address := "0x" + senderFromTransaction(tx)
	hash := ethHashStringFromTransaction(tx)
	registerPendingEthTx(hash, tx)
	t.Cleanup(func() { clearPendingEthTx(hash) })

	gotLatest, err := server.EthGetTransactionCount([]any{address, latestBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotLatest)

	gotSafe, err := server.EthGetTransactionCount([]any{address, safeBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotSafe)

	gotFinalized, err := server.EthGetTransactionCount([]any{address, finalizedBlockTag})
	require.NoError(t, err)
	require.Equal(t, hexutil.Uint64(4_999), gotFinalized)

	_, err = server.EthGetTransactionCount([]any{address, pendingBlockTag})
	require.ErrorContains(t, err, "no replay-safe nonce available within accepted window")
}

func newTestEthServerWithPartiallyIndexedTx(t *testing.T) (*Server, *lib.TxResult, common.Hash) {
	t.Helper()

	log := lib.NewDefaultLogger()
	storeI, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	db := storeI.(*store.Store)

	sm := newTestRPCStateMachine(t, db, log)
	txResult, ethHash := newTestRLPBackedTxResult(t)
	require.NoError(t, db.IndexTx(txResult))
	_, err = db.Commit()
	require.NoError(t, err)

	setFSMHeight(t, sm, db.Version())

	return &Server{
		controller:       newTestRPCController(sm),
		config:           lib.DefaultConfig(),
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}, txResult, ethHash
}

func newTestEthServerWithDeletedButCachedTx(t *testing.T) (*Server, *lib.TxResult, common.Hash) {
	t.Helper()

	log := lib.NewDefaultLogger()
	storeI, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	db := storeI.(*store.Store)

	sm := newTestRPCStateMachine(t, db, log)
	txResult, ethHash := newTestRLPBackedTxResult(t)
	block := &lib.BlockResult{
		BlockHeader: &lib.BlockHeader{
			Height: 1,
			Hash:   ethCrypto.Keccak256([]byte("block-cached")),
			Time:   uint64(time.Now().UnixMicro()),
		},
		Transactions: []*lib.TxResult{txResult},
	}

	require.NoError(t, db.IndexBlock(block))
	_, err = db.Commit()
	require.NoError(t, err)
	_, err = db.GetBlockByHeight(block.BlockHeader.Height)
	require.NoError(t, err)
	require.NoError(t, db.DeleteTxsForHeight(block.BlockHeader.Height))
	_, err = db.Commit()
	require.NoError(t, err)

	setFSMHeight(t, sm, db.Version())

	return &Server{
		controller:       newTestRPCController(sm),
		config:           lib.DefaultConfig(),
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}, txResult, ethHash
}

func newTestEthServerAtHeight(t *testing.T, height uint64) *Server {
	t.Helper()

	server, _ := newTestEthServerAndStoreAtHeight(t, height)
	return server
}

func newTestEthServerAndStoreAtHeight(t *testing.T, height uint64) (*Server, *store.Store) {
	t.Helper()

	log := lib.NewDefaultLogger()
	storeI, err := store.NewStoreInMemory(log)
	require.NoError(t, err)
	db := storeI.(*store.Store)

	sm := newTestRPCStateMachine(t, db, log)
	setFSMHeight(t, sm, height)

	return &Server{
		controller:       newTestRPCController(sm),
		config:           lib.DefaultConfig(),
		indexerBlobCache: newIndexerBlobCache(8),
		logger:           log,
	}, db
}

func newTestRPCController(sm *fsm.StateMachine) *controller.Controller {
	return &controller.Controller{
		FSM: sm,
		Mempool: &controller.Mempool{
			Mempool: lib.NewMempool(lib.DefaultMempoolConfig()),
			L:       &sync.Mutex{},
		},
	}
}

func newTestRLPBackedTxResult(t *testing.T) (*lib.TxResult, common.Hash) {
	t.Helper()

	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)

	recipient := common.HexToAddress("0x0000000000000000000000000000000000000011")
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(big.NewInt(1)), &types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     7,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(10_000_000_000),
		Gas:       21_000,
		To:        &recipient,
		Value:     new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	})
	rawEthTx, err := ethTx.MarshalBinary()
	require.NoError(t, err)

	return &lib.TxResult{
		Sender:    sender.Bytes(),
		Recipient: recipient.Bytes(),
		Height:    1,
		Index:     0,
		Transaction: &lib.Transaction{
			Memo:      "RLP",
			Signature: &lib.Signature{Signature: rawEthTx},
		},
		TxHash: "1111111111111111111111111111111111111111111111111111111111111111",
	}, ethTx.Hash()
}

func newTestPendingRLPTransaction(t *testing.T, nonce uint64) *lib.Transaction {
	t.Helper()

	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)

	recipient := common.HexToAddress("0x0000000000000000000000000000000000000011")
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(big.NewInt(4_294_967_297)), &types.DynamicFeeTx{
		ChainID:   big.NewInt(4_294_967_297),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(10_000_000_000),
		Gas:       21_000,
		To:        &recipient,
		Value:     new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	})
	rawEthTx, err := ethTx.MarshalBinary()
	require.NoError(t, err)

	tx, err := fsm.RLPToCanopyTransaction(rawEthTx)
	require.NoError(t, err)
	return tx
}

func newTestIndexedTxResultFromPendingTx(t *testing.T, tx *lib.Transaction, height uint64) *lib.TxResult {
	t.Helper()

	ethTx, ok := ethTransactionFromCanopyTx(tx)
	require.True(t, ok)

	sender := common.HexToAddress("0x" + senderFromTransaction(tx))
	var recipient []byte
	if to := ethTx.To(); to != nil {
		recipient = to.Bytes()
	}

	return &lib.TxResult{
		Sender:    sender.Bytes(),
		Recipient: recipient,
		Height:    height,
		Index:     0,
		Transaction: &lib.Transaction{
			Memo:          tx.Memo,
			CreatedHeight: tx.CreatedHeight,
			Fee:           tx.Fee,
			Msg:           tx.Msg,
			Signature:     tx.Signature,
		},
		TxHash: strings.Repeat("2", 64),
	}
}
