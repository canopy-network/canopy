package lib

import (
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestAddTransactionFeeOrdering(t *testing.T) {
	// pre-define a mempool with the default config
	mempool := NewMempool(DefaultMempoolConfig())
	// pre-define a test message
	sig := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// pre-define an any for testing
	a, e := NewAny(sig)
	require.NoError(t, e)
	// add a transaction
	err := mempool.AddTransactions(func() []byte {
		bz, err := Marshal(&Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
			Time: uint64(time.Now().UnixMicro()), Fee: 1000, NetworkId: 1, ChainId: 2})
		require.NoError(t, err)
		return bz
	}())
	require.NoError(t, err)
	// add another transaction with the same fee
	err = mempool.AddTransactions(func() []byte {
		bz, err := Marshal(&Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
			Time: uint64(time.Now().UnixMicro()), Fee: 1000, NetworkId: 1, ChainId: 3})
		require.NoError(t, err)
		return bz
	}())
	require.NoError(t, err)
	// add another transaction with a higher fee
	err = mempool.AddTransactions(func() []byte {
		bz, err := Marshal(&Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
			Time: uint64(time.Now().UnixMicro()), Fee: 1001, NetworkId: 1, ChainId: 1})
		require.NoError(t, err)
		return bz
	}())
	require.NoError(t, err)
	// add another transaction with the lowest fee
	err = mempool.AddTransactions(func() []byte {
		bz, err := Marshal(&Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
			Time: uint64(time.Now().UnixMicro()), Fee: 1, NetworkId: 1, ChainId: 5})
		require.NoError(t, err)
		return bz
	}())
	require.NoError(t, err)
	// add another transaction with the same fee
	err = mempool.AddTransactions(func() []byte {
		bz, err := Marshal(&Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
			Time: uint64(time.Now().UnixMicro()), Fee: 1000, NetworkId: 1, ChainId: 4})
		require.NoError(t, err)
		return bz
	}())
	require.NoError(t, err)
	it := mempool.Iterator()
	defer it.Close()
	// iterate through each
	for expected := 1; it.Valid(); it.Next() {
		tx := new(Transaction)
		require.NoError(t, Unmarshal(it.Key(), tx))
		// compare got vs expected
		require.Equal(t, expected, int(tx.ChainId))
		expected++
	}
}

func TestAddTransaction(t *testing.T) {
	// pre-define a test message
	sig := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// pre-define an any for testing
	a, e := NewAny(sig)
	require.NoError(t, e)
	// pre-define a transaction to add
	tx := &Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
		Time: uint64(time.Now().UnixMicro()), Fee: 1000, NetworkId: 1, ChainId: 2}
	// marshal to bytes
	transaction, err := Marshal(tx)
	require.NoError(t, err)
	tests := []struct {
		name    string
		detail  string
		mempool FeeMempool
		toAdd   []byte
		// expected
		transactions [][]byte
		count        int
		error        string
	}{
		{
			name:    "max tx size",
			detail:  "the tx size exceeds max (config)",
			mempool: FeeMempool{},
			toAdd:   transaction,
			error:   "max tx size",
		},
		{
			name:   "already exists",
			detail: "transaction not added because it already exists",
			mempool: FeeMempool{
				pool: MempoolTxs{
					m: map[string]struct{}{crypto.HashString(transaction): {}},
				},
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: 0,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      10,
				},
			},
			toAdd: transaction,
		},
		{
			name:   "recheck max tx count",
			detail: "max tx count causes a recheck",
			mempool: FeeMempool{
				pool: MempoolTxs{
					m: make(map[string]struct{}),
					s: make([]MempoolTx, 0),
				},
				txsBytes: 0,
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: 1,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      10,
				},
			},
			toAdd: transaction,
		},
		{
			name:   "recheck max total bytes",
			detail: "max total bytes",
			mempool: FeeMempool{
				pool: MempoolTxs{
					m: make(map[string]struct{}),
					s: make([]MempoolTx, 0),
				},
				txsBytes: 0,
				config: MempoolConfig{
					MaxTotalBytes:       0,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      10,
				},
			},
			toAdd: transaction,
		},
		{
			name:   "no recheck",
			detail: "there's no recheck as the transaction is added without exceeding limits",
			mempool: FeeMempool{
				pool: MempoolTxs{
					m: make(map[string]struct{}),
					s: make([]MempoolTx, 0),
				},
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      10,
				},
			},
			count: 1,
			toAdd: transaction,
			transactions: [][]byte{
				transaction,
			},
		},
		{
			name:   "multi-transaction",
			detail: "test transaction ordering with multi-transaction",
			mempool: FeeMempool{
				pool: MempoolTxs{
					m: make(map[string]struct{}),
					s: make([]MempoolTx, 0),
				},
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      10,
				},
			},
			count: 1,
			toAdd: transaction,
			transactions: [][]byte{
				transaction,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute function call
			err = test.mempool.AddTransactions(test.toAdd)
			// validate if an error is expected
			require.Equal(t, err != nil, test.error != "", err)
			// validate actual error if any
			if err != nil {
				require.ErrorContains(t, err, test.error, err)
				return
			}
			require.Equal(t, test.count, test.mempool.TxCount())
			// call get transaction
			gotTxs := test.mempool.GetTransactions(math.MaxUint64)
			require.Equal(t, test.transactions, gotTxs)
			// test mempool.Contains
			for _, txn := range test.transactions {
				require.True(t, test.mempool.Contains(crypto.HashString(txn)))
			}
		})
	}
}

func TestGetAndContainsTransaction(t *testing.T) {
	// pre-define a test message
	sig := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: newTestPublicKeyBytes(t),
	}
	// pre-define an any for testing
	a, e := NewAny(sig)
	require.NoError(t, e)
	// pre-define a transaction to add
	tx := &Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
		Time: uint64(time.Now().UnixMicro()), Fee: 1000, NetworkId: 1, ChainId: 1}
	// marshal to bytes
	transactionA, err := Marshal(tx)
	require.NoError(t, err)
	// pre-define a transaction to add
	tx = &Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
		Time: uint64(time.Now().UnixMicro()), Fee: 1001, NetworkId: 1, ChainId: 1}
	// marshal to bytes
	transactionB, err := Marshal(tx)
	// pre-define a transaction to add
	tx = &Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
		Time: uint64(time.Now().UnixMicro()), Fee: 999, NetworkId: 1, ChainId: 1}
	// marshal to bytes
	transactionC, err := Marshal(tx)
	// pre-define a transaction to add
	tx = &Transaction{MessageType: testMessageName, Msg: a, Signature: sig, CreatedHeight: 1,
		Time: uint64(time.Now().UnixMicro()), Fee: 1, NetworkId: 1, ChainId: 1}
	// marshal to bytes
	transactionD, err := Marshal(tx)
	require.NoError(t, err)
	// define test cases
	tests := []struct {
		name          string
		detail        string
		txs           [][]byte
		mempool       Mempool
		expectedCount uint64
		expectedTxs   [][]byte
		maxBytes      uint64
	}{
		{
			name:   "reap top 3 transactions",
			detail: "get the top 3 transactions only based on the max bytes ",
			txs: [][]byte{
				transactionA,
				transactionB,
				transactionC,
				transactionD,
			},
			mempool: &FeeMempool{
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      30,
				},
			},
			expectedCount: 3,
			expectedTxs: [][]byte{
				transactionB,
				transactionA,
				transactionC,
			},
			maxBytes: uint64(len(transactionA) + len(transactionB) + len(transactionC)),
		},
		{
			name:   "reap top 2 transactions",
			detail: "get the top 2 transactions only based on the max bytes",
			txs: [][]byte{
				transactionA,
				transactionB,
				transactionC,
				transactionD,
			},
			mempool: &FeeMempool{
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      30,
				},
			},
			expectedCount: 2,
			expectedTxs: [][]byte{
				transactionB,
				transactionA,
			},
			maxBytes: uint64(len(transactionA) + len(transactionB)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// pre-add the transactions
			for _, txn := range test.txs {
				err := test.mempool.AddTransactions(txn)
				require.NoError(t, err)
			}
			// get the transactions
			got := test.mempool.GetTransactions(test.maxBytes)
			// ensure the count is correct
			require.EqualValues(t, test.expectedCount, len(got))
			require.Equal(t, len(test.expectedTxs), len(got))
			// compare got vs expected
			for i := 0; i < len(got); i++ {
				require.Equal(t, test.expectedTxs[i], got[i])
				require.True(t, test.mempool.Contains(crypto.HashString(test.txs[i])))
			}
		})
	}
}

func TestDeleteTransaction(t *testing.T) {
	// define test cases
	tests := []struct {
		name          string
		detail        string
		mempool       Mempool
		delete        [][]byte
		expectedTxs   []MempoolTx
		expectedCount uint64
	}{
		{
			name:   "delete the first transaction",
			detail: "delete the transaction with the highest fee",
			mempool: &FeeMempool{
				pool: MempoolTxs{
					m: map[string]struct{}{
						crypto.HashString([]byte("b")): {},
						crypto.HashString([]byte("a")): {},
						crypto.HashString([]byte("c")): {},
					},
					s: []MempoolTx{
						{
							Tx:  []byte("b"),
							Fee: 1001,
						},
						{
							Tx:  []byte("a"),
							Fee: 1000,
						},
						{
							Tx:  []byte("c"),
							Fee: 999,
						},
					},
				},
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      30,
				},
			},
			delete: [][]byte{
				[]byte("b"),
			},
			expectedCount: 2,
			expectedTxs: []MempoolTx{
				{
					Tx:  []byte("a"),
					Fee: 1000,
				},
				{
					Tx:  []byte("c"),
					Fee: 999,
				},
			},
		},
		{
			name:   "delete the second two transactions",
			detail: "delete the 2 transactions with the lowest fees",
			mempool: &FeeMempool{
				pool: MempoolTxs{
					m: map[string]struct{}{
						crypto.HashString([]byte("b")): {},
						crypto.HashString([]byte("a")): {},
						crypto.HashString([]byte("c")): {},
					},
					s: []MempoolTx{
						{
							Tx:  []byte("b"),
							Fee: 1001,
						},
						{
							Tx:  []byte("a"),
							Fee: 1000,
						},
						{
							Tx:  []byte("c"),
							Fee: 999,
						},
					},
				},
				config: MempoolConfig{
					MaxTotalBytes:       math.MaxUint64,
					MaxTransactionCount: math.MaxUint32,
					IndividualMaxTxSize: math.MaxUint32,
					DropPercentage:      30,
				},
			},
			delete: [][]byte{
				[]byte("a"),
				[]byte("c"),
			},
			expectedCount: 1,
			expectedTxs: []MempoolTx{
				{
					Tx:  []byte("b"),
					Fee: 1001,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// delete the transactions
			for _, toDelete := range test.delete {
				test.mempool.DeleteTransaction(toDelete)
			}
			// get the transactions left
			got := test.mempool.GetTransactions(math.MaxUint64)
			// ensure the count is correct
			require.EqualValues(t, test.expectedCount, len(got))
			require.Equal(t, len(test.expectedTxs), len(got))
			// compare got vs expected
			for i := 0; i < len(got); i++ {
				require.Equal(t, test.expectedTxs[i].Tx, got[i])
				require.True(t, test.mempool.Contains(crypto.HashString(test.expectedTxs[i].Tx)))
			}
		})
	}
}

func TestFailedTxCache(t *testing.T) {
	rawPubKey := newTestPublicKeyBytes(t)
	pubKey, err := crypto.NewPublicKeyFromBytes(rawPubKey)
	require.NoError(t, err)

	// pre-define a test message
	sig := &Signature{
		PublicKey: newTestPublicKeyBytes(t),
		Signature: rawPubKey,
	}
	// pre-define an any for testing
	a, e := NewAny(sig)
	require.NoError(t, e)
	// pre-define a transaction
	tx := &Transaction{
		MessageType: testMessageName,
		Msg:         a,
		Signature:   sig,
		Time:        uint64(time.Now().UnixMicro()),
		Fee:         1,
		Memo:        "memo",
	}
	// marshal transaction to bytes
	txBytes, err := Marshal(tx)
	require.NoError(t, err)

	// define test cases
	tests := []struct {
		name                    string
		dissallowedMessageTypes []string
		txBytes                 []byte
		hash                    string
		err                     error
		expectedResult          bool
		address                 string
	}{
		{
			name:                    "valid transaction",
			dissallowedMessageTypes: []string{},
			txBytes:                 txBytes,
			hash:                    crypto.HashString(txBytes),
			err:                     nil,
			expectedResult:          true,
			address:                 pubKey.Address().String(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a new failed tx cache
			cache := NewFailedTxCache(test.dissallowedMessageTypes...)
			// create a new failed tx
			failedTx := NewFailedTx(test.txBytes, test.err)
			// add transaction to cache
			result := cache.Add(failedTx)
			// validate result
			require.Equal(t, test.expectedResult, result)
			if test.expectedResult {
				// validate cache
				failedTx, ok := cache.Get(test.hash)
				require.True(t, ok)
				require.Equal(t, test.err, failedTx.Error)
				require.EqualExportedValues(t, tx, failedTx.Transaction)

				// validate get all
				failedTxs := cache.GetFailedForAddress(test.address)
				require.Len(t, failedTxs, 1)
				require.Equal(t, failedTx, failedTxs[0])

				// validate removal
				cache.Remove(test.hash)
				_, ok = cache.Get(test.hash)
				require.False(t, ok)
			} else {
				// validate cache
				tx, ok := cache.Get(test.hash)
				require.False(t, ok)
				require.Nil(t, tx)
			}
		})
	}
}
