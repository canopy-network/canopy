package store

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const testHeight = 1

func TestGetTxByHash(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	txRes, _, hash, _, _ := newTestTxResult(t)
	require.NoError(t, store.IndexTx(txRes))
	txResult, err := store.GetTxByHash(hash)
	require.NoError(t, err)
	gotBytes, err := lib.Marshal(txResult)
	require.NoError(t, err)
	wantedBytes, err := lib.Marshal(txRes)
	require.NoError(t, err)
	require.True(t, bytes.Equal(gotBytes, wantedBytes))
}

func TestGetTxByHeight(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()
	txRes, _, _, _, _ := newTestTxResult(t)
	require.NoError(t, store.IndexTx(txRes))
	_, err := store.Commit()
	require.NoError(t, err)
	txResults, err := store.GetTxsByHeightNonPaginated(testHeight, true)
	require.NoError(t, err)
	require.Len(t, txResults, 1)
	gotBytes, err := lib.Marshal(txResults[0])
	require.NoError(t, err)
	wantedBytes, err := lib.Marshal(txRes)
	require.NoError(t, err)
	require.True(t, bytes.Equal(gotBytes, wantedBytes))
}

func TestGetTxByEthereumHash(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	txResult, ethHash := newRLPBackedTxResult(t)
	require.NoError(t, store.IndexTx(txResult))
	got, err := store.GetTxByHash(ethHash.Bytes())
	require.NoError(t, err)
	gotBytes, err := lib.Marshal(got)
	require.NoError(t, err)
	wantBytes, err := lib.Marshal(txResult)
	require.NoError(t, err)
	require.True(t, bytes.Equal(gotBytes, wantBytes))
}

func TestDeleteTxsForHeightRemovesEthereumHashAlias(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	txResult, ethHash := newRLPBackedTxResult(t)
	require.NoError(t, store.IndexTx(txResult))
	_, err := store.Commit()
	require.NoError(t, err)
	require.NoError(t, store.DeleteTxsForHeight(testHeight))

	raw, err := store.Indexer.db.Get(store.txHashKey(ethHash.Bytes()))
	require.NoError(t, err)
	got, err := store.GetTxByHash(ethHash.Bytes())
	require.NoError(t, err)
	require.Len(t, raw, 0)
	require.True(t, got == nil || got.TxHash == "")
}

func TestGetLatestMinedEthereumNonce(t *testing.T) {
	store, _, cleanup := testStore(t)
	defer cleanup()

	txResult, _ := newRLPBackedTxResult(t)
	require.NoError(t, store.IndexTx(txResult))

	nonce, ok, err := store.GetLatestMinedEthereumNonce(crypto.NewAddress(txResult.Sender))
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, txResult.Transaction.CreatedHeight, nonce)
}

const ethGasPriceTestValue = 10_000_000_000

func ptrAddress(address common.Address) *common.Address { return &address }

func newRLPBackedTxResult(t *testing.T) (*lib.TxResult, common.Hash) {
	key, err := ethCrypto.GenerateKey()
	require.NoError(t, err)
	ethTx := types.MustSignNewTx(key, types.LatestSignerForChainID(big.NewInt(1)), &types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     7,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(ethGasPriceTestValue),
		Gas:       21_000,
		To:        ptrAddress(common.HexToAddress("0x0000000000000000000000000000000000000001")),
		Value:     big.NewInt(42),
	})
	rawEthTx, err := ethTx.MarshalBinary()
	require.NoError(t, err)

	protoHash := bytes.Repeat([]byte{0x11}, 32)
	return &lib.TxResult{
		Sender:    common.HexToAddress("0x0000000000000000000000000000000000000002").Bytes(),
		Recipient: common.HexToAddress("0x0000000000000000000000000000000000000001").Bytes(),
		Height:    testHeight,
		Index:     0,
		Transaction: &lib.Transaction{
			Memo:      "RLP",
			Signature: &lib.Signature{Signature: rawEthTx},
		},
		TxHash: hex.EncodeToString(protoHash),
	}, ethTx.Hash()
}

func newTestTxResult(t *testing.T) (r *lib.TxResult, tx *lib.Transaction, hash []byte, msg *lib.CommitID, address crypto.AddressI) {
	pk, err := crypto.NewEd25519PrivateKey()
	require.NoError(t, err)
	msg = &lib.CommitID{
		Height: 1,
		Root:   []byte("root"),
	}
	address = pk.PublicKey().Address()
	a, err := lib.NewAny(msg)
	require.NoError(t, err)
	tx = &lib.Transaction{
		MessageType: "commit_id",
		Msg:         a,
		Time:        uint64(time.Now().UnixMicro()),
		Fee:         1,
	}
	require.NoError(t, tx.Sign(pk))
	hash, err = tx.GetHash()
	require.NoError(t, err)
	r = &lib.TxResult{
		Sender:      address.Bytes(),
		Recipient:   address.Bytes(),
		MessageType: "commit_id",
		Height:      testHeight,
		Index:       0,
		Transaction: tx,
		TxHash:      lib.BytesToString(hash),
	}
	return
}
