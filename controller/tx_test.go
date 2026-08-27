package controller

import (
	"math/big"
	"sync"
	"testing"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestGetProposalBlockFromMempool(t *testing.T) {
	c := &Controller{Mempool: &Mempool{}}

	p, ok := c.GetProposalBlockFromMempool()
	require.False(t, ok)
	require.Nil(t, p)

	expected := &CachedProposal{dirtyVersion: 1}
	c.Mempool.cachedProposal.Store(expected)

	p, ok = c.GetProposalBlockFromMempool()
	require.True(t, ok)
	require.Equal(t, expected, p)
}

func TestHandleTransactionsOnlyMarksDirtyOnSuccessfulNewTx(t *testing.T) {
	key, err := crypto.NewBLS12381PrivateKey()
	require.NoError(t, err)

	tx, errI := fsm.NewSendTransaction(key, key.PublicKey().Address(), 1, 1, 1, 1, 1, "")
	require.NoError(t, errI)

	txBytes, errI := lib.Marshal(tx)
	require.NoError(t, errI)

	m := &Mempool{
		Mempool: lib.NewMempool(lib.DefaultMempoolConfig()),
		L:       &sync.Mutex{},
	}

	require.NoError(t, m.HandleTransactions(txBytes))
	require.EqualValues(t, 1, m.dirtyVersion.Load())

	require.NoError(t, m.HandleTransactions(txBytes))
	require.EqualValues(t, 1, m.dirtyVersion.Load())

	require.Error(t, m.HandleTransactions([]byte("bad-tx")))
	require.EqualValues(t, 1, m.dirtyVersion.Load())

	config := lib.DefaultMempoolConfig()
	config.MaxTransactionCount = 1
	m = &Mempool{Mempool: lib.NewMempool(config), L: &sync.Mutex{}}
	require.NoError(t, m.HandleTransactions(txBytes))
	require.False(t, m.Contains(crypto.HashString(txBytes)))
	require.ErrorContains(t, m.HandleTransactionAndVerifyRetained(txBytes, nil), "evicted from mempool")
}

func TestGetPendingTxByHashUsesCachedResults(t *testing.T) {
	ctrl := &Controller{
		Mutex: &sync.Mutex{},
		Mempool: &Mempool{
			L: &sync.Mutex{},
			cachedResults: lib.TxResults{
				&lib.TxResult{TxHash: "abc123"},
			},
		},
	}
	tx, found := ctrl.GetPendingTxByHash("0xABC123")
	require.True(t, found)
	require.NotNil(t, tx)
	require.Equal(t, "abc123", tx.TxHash)

	tx, found = ctrl.GetPendingTxByHash("missing")
	require.False(t, found)
	require.Nil(t, tx)
}

func TestGetPendingTxByHashAcceptsEthereumHash(t *testing.T) {
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	to := common.Address{1}
	chainID, ok := fsm.CanopyIdsToEVMChainIdV2(1, 1)
	require.True(t, ok)
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(new(big.Int).SetUint64(chainID)), &types.DynamicFeeTx{
		ChainID: new(big.Int).SetUint64(chainID), GasFeeCap: big.NewInt(fsm.EthereumBaseFeePerGas), Gas: 21_000, To: &to,
		Value: fsm.UpscaleTo18Decimals(1),
	})
	raw, err := ethTx.MarshalBinary()
	require.NoError(t, err)
	tx, errI := fsm.RLPToCanopyTransactionV2(raw)
	require.NoError(t, errI)
	ctrl := &Controller{Mempool: &Mempool{L: &sync.Mutex{}, cachedResults: lib.TxResults{{Transaction: tx}}}}

	_, found := ctrl.GetPendingTxByHash(ethTx.Hash().Hex())
	require.True(t, found)
}

func TestPendingReadersReturnImmediatelyWhenMempoolIsLocked(t *testing.T) {
	mempool := &Mempool{
		L:             &sync.Mutex{},
		cachedResults: lib.TxResults{{TxHash: "abc123"}},
	}
	ctrl := &Controller{Mutex: &sync.Mutex{}, Mempool: mempool}
	mempool.L.Lock()
	defer mempool.L.Unlock()

	page, err := ctrl.GetPendingPage(lib.PageParams{PageNumber: 1, PerPage: 10})
	require.NoError(t, err)
	require.Zero(t, page.Count)
	require.Zero(t, page.TotalCount)

	tx, found := ctrl.GetPendingTxByHash("abc123")
	require.False(t, found)
	require.Nil(t, tx)

	address := crypto.NewAddress(make([]byte, crypto.AddressSize))
	require.EqualValues(t, 7, ctrl.GetPendingNonce(address, 7))
}

func TestGetPendingNonceUsesValidatedResults(t *testing.T) {
	address := crypto.NewAddress(make([]byte, crypto.AddressSize))
	ctrl := &Controller{Mempool: &Mempool{
		L: &sync.Mutex{},
		cachedResults: lib.TxResults{
			{Sender: address.Bytes(), Transaction: &lib.Transaction{Memo: fsm.RLPV2Indicator, Nonce: 2}},
			{Sender: address.Bytes(), Transaction: &lib.Transaction{Memo: fsm.RLPV2Indicator, Nonce: 5}},
		},
	}}

	require.EqualValues(t, 6, ctrl.GetPendingNonce(address, 1))
}
